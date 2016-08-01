package peer

import (
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"code.cloudfoundry.org/lager"
)

type peerClient interface {
	ReadLeader(logger lager.Logger, leader string) ([]Glimpse, error)
	PostAndReadSnapshot(logger lager.Logger, host string) ([]Glimpse, error)
}

type Heartbeat struct {
	Leader        string
	Peers         List
	Logger        lager.Logger
	CheckInterval time.Duration
	Client        peerClient
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
		leaderLogger := logger.Session("read-leader").WithData(lager.Data{"leader": h.Leader})
		leaderPeers, err := h.Client.ReadLeader(leaderLogger, h.Leader)
		if err != nil {
			leaderLogger.Error("get-from-leader", err)
			return
		}
		leaderLogger.Info("get-from-leader", lager.Data{"candidate-peers": leaderPeers})
		h.Peers.UpsertUntrusted(leaderLogger, leaderPeers)
	}

	ttlThreshhold := int(h.CheckInterval.Seconds())
	candidates := h.Peers.Snapshot(logger)

	wg := sync.WaitGroup{}
	for _, peer := range candidates {
		if peer.TTL <= ttlThreshhold || rand.Float32() > 0.5 {
			wg.Add(1)
			go func(peerHost string) {
				defer wg.Done()
				peerLogger := logger.Session("post-peer").WithData(lager.Data{"peer": peerHost})
				morePeers, err := h.Client.PostAndReadSnapshot(peerLogger, peerHost)
				if err != nil {
					peerLogger.Error("post-to-peer", err)
					return
				}
				peerLogger.Debug("post-to-peer")
				h.Peers.Upsert(peerLogger, peerHost)
				h.Peers.UpsertUntrusted(peerLogger, morePeers)
			}(peer.Host)
		}
	}
	wg.Wait()
}
