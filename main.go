package main

import (
	"fmt"
	"os"

	"github.com/rosenhouse/reflex/handler"
	"github.com/rosenhouse/reflex/peer"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
	"github.com/tedsuo/rata"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/localip"
)

func main() {
	logger := lager.NewLogger("reflex")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, lager.DEBUG))

	config, err := GetConfig(logger, os.Environ())
	if err != nil {
		logger.Fatal("parse-config", err)
	}

	myIP, err := localip.LocalIP()
	if err != nil {
		logger.Fatal("local-ip", err)
	}
	logger.Info("local-ip", lager.Data{"ip": myIP})
	peers := peer.NewList(config.TTL, myIP)

	peerListHandler := &handler.PeerList{
		Logger: logger,
		Peers:  peers,
	}

	peerPostHandler := &handler.PeerPost{
		Logger:      logger,
		Peers:       peers,
		AllowedCIDR: config.AllowedPeers,
	}

	routes := rata.Routes{
		{Name: "peers_list", Method: "GET", Path: "/"},
		{Name: "peers_list", Method: "GET", Path: "/peers"},
		{Name: "peers_upsert", Method: "POST", Path: "/peers"},
	}

	handlers := rata.Handlers{
		"peers_list":   peerListHandler,
		"peers_upsert": peerPostHandler,
	}
	router, err := rata.NewRouter(routes, handlers)
	if err != nil {
		logger.Fatal("new-router", err)
	}

	httpServer := http_server.New(fmt.Sprintf("%s:%d", "0.0.0.0", config.Port), router)
	members := grouper.Members{
		{"http_server", httpServer},
		{"list_culler", ifrit.RunFunc(peers.RunCullerLoop)},
	}

	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))
	logger.Info("started")

	err = <-monitor.Wait()
	if err != nil {
		logger.Fatal("monitor", err)
	}
}
