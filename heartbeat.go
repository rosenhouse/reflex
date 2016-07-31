package main

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/rosenhouse/reflex/client"
	"github.com/rosenhouse/reflex/peer"
)

type Heartbeat struct {
	Leader        string
	Peers         peer.List
	Logger        lager.Logger
	CheckInterval time.Duration
	Client        *client.Client
}

func (h *Heartbeat) RunHeartbeat(signals <-chan os.Signal, ready chan<- struct{}) error {
	rand.Seed(time.Now().UnixNano())
	nextInterval, _ := time.ParseDuration(fmt.Sprintf("%ds", rand.Intn(5)))
	close(ready)

	for {
		select {
		case <-signals:
			return nil
		case <-time.After(nextInterval):
			h.check()
		}

		jitter := (rand.Float64() + 0.5) * h.CheckInterval.Seconds() / 2
		nextInterval = time.Duration(jitter) * time.Second
		h.Logger.Debug("next-interval", lager.Data{"seconds": nextInterval.Seconds()})
	}
}

func (h *Heartbeat) check() {
	logger := h.Logger.Session("heartbeat")
	defer logger.Debug("done")

	if h.Leader != "" {
		leaderPeers, err := h.Client.ReadLeader(h.Leader)
		if err != nil {
			logger.Error("get-from-leader", err, lager.Data{"leader": h.Leader})
			return
		}
		logger.Info("get-from-leader", lager.Data{"candidate-peers": leaderPeers})
		h.Peers.UpsertUntrusted(logger, leaderPeers)
	}

	ttlThreshhold := int(h.CheckInterval.Seconds())
	candidates := h.Peers.Snapshot(logger)

	wg := sync.WaitGroup{}
	for _, peer := range candidates {
		if peer.TTL <= ttlThreshhold || rand.Float32() > 0.5 {
			wg.Add(1)
			go func(peerHost string) {
				defer wg.Done()
				morePeers, err := h.Client.PostAndReadSnapshot(peerHost)
				if err != nil {
					logger.Error("post-to-peer", err, lager.Data{"peer": peerHost})
					return
				}
				logger.Debug("post-to-peer", lager.Data{"peer": peerHost})
				h.Peers.Upsert(logger, peerHost)
				h.Peers.UpsertUntrusted(logger, morePeers)
			}(peer.Host)
		}
	}
	wg.Wait()
}
