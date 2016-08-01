package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/rosenhouse/reflex/client"
	"github.com/rosenhouse/reflex/handler"
	"github.com/rosenhouse/reflex/metric"
	"github.com/rosenhouse/reflex/peer"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
	"github.com/tedsuo/rata"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/localip"
)

func parseLogLevel(level string) lager.LogLevel {
	switch level {
	case "debug", "DEBUG", "d", strconv.Itoa(int(lager.DEBUG)):
		return lager.DEBUG
	case "info", "INFO", "i", strconv.Itoa(int(lager.INFO)):
		return lager.INFO
	case "error", "ERROR", "e", strconv.Itoa(int(lager.ERROR)):
		return lager.ERROR
	case "fatal", "FATAL", "f", strconv.Itoa(int(lager.FATAL)):
		return lager.FATAL
	}
	return lager.DEBUG
}

func main() {
	logger := lager.NewLogger("reflex")
	sink := lager.NewReconfigurableSink(lager.NewWriterSink(os.Stdout, lager.DEBUG), lager.DEBUG)
	logger.RegisterSink(sink)

	config, err := GetConfig(logger, os.Environ())
	if err != nil {
		logger.Fatal("parse-config", err)
	}
	sink.SetMinLevel(config.LogLevel)

	myIP, err := localip.LocalIP()
	if err != nil {
		logger.Fatal("local-ip", err)
	}
	logger.Info("local-ip", lager.Data{"ip": myIP})

	metricStore := metric.NewStore(config.MetricMaxCapacity)

	client := &client.Client{
		HTTPClient: http.DefaultClient,
		Port:       config.Port,

		ReportRoundTripLatency: func(d time.Duration) {
			metricStore.Report("round_trip", d.Seconds())
		},
	}

	peers := peer.NewList(config.TTL, myIP)

	heartbeat := peer.Heartbeat{
		Leader:        config.Leader,
		CheckInterval: config.TTL,
		Peers:         peers,
		Logger:        logger,
		Client:        client,
	}

	peerListHandler := &handler.PeerList{
		Logger: logger,
		Peers:  peers,
	}

	peerPostHandler := &handler.PeerPost{
		Logger:      logger,
		Peers:       peers,
		AllowedCIDR: config.AllowedPeers,
	}

	metricsDataHandler := &handler.MetricsData{
		Logger:         logger,
		SnapshotGetter: func() interface{} { return metricStore.Snapshot() },
	}
	metricsDisplayHandler := &handler.MetricsDisplay{
		Logger: logger,
	}

	routes := rata.Routes{
		{Name: "peers_list", Method: "GET", Path: "/peers"},
		{Name: "peers_upsert", Method: "POST", Path: "/peers"},
		{Name: "metrics_data", Method: "GET", Path: "/metrics/data"},
		{Name: "metrics_display", Method: "GET", Path: "/metrics"},
		{Name: "metrics_display", Method: "GET", Path: "/"},
	}

	handlers := rata.Handlers{
		"peers_list":      peerListHandler,
		"peers_upsert":    peerPostHandler,
		"metrics_data":    metricsDataHandler,
		"metrics_display": metricsDisplayHandler,
	}
	router, err := rata.NewRouter(routes, handlers)
	if err != nil {
		logger.Fatal("new-router", err)
	}

	httpServer := http_server.New(fmt.Sprintf("%s:%d", "0.0.0.0", config.Port), router)
	members := grouper.Members{
		{"http_server", httpServer},
		{"list_culler", ifrit.RunFunc(peers.RunCullerLoop)},
		{"heart_beater", ifrit.RunFunc(heartbeat.RunHeartbeat)},
	}

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))
	logger.Info("started")

	err = <-monitor.Wait()
	if err != nil {
		logger.Fatal("monitor", err)
	}
}
