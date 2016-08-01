package science

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/rosenhouse/reflex/peer"

	"code.cloudfoundry.org/lager"
)

type BandwidthExperimentResult struct {
	NumBytes        int64   `json:"num_bytes"`
	DurationSeconds float64 `json:"duration_seconds"`
	AvgBandwidth    float64 `json:"avg_bandwidth"`
	SHA256          string  `json:"sha256"`
}

type scienceClient interface {
	TestBandwidth(logger lager.Logger, host string, payloadSize int64) (*BandwidthExperimentResult, error)
}

type BandwidthExperiment struct {
	Peers         peer.List
	Logger        lager.Logger
	CheckInterval time.Duration
	Client        scienceClient
	PayloadSize   int64

	ReportAvgBandwidth func(float64)
}

func (b *BandwidthExperiment) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	rand.Seed(time.Now().UnixNano())
	nextInterval, _ := time.ParseDuration(fmt.Sprintf("%ds", rand.Intn(5)))
	close(ready)

	for {
		select {
		case <-signals:
			return nil
		case <-time.After(nextInterval):
			b.run()
		}

		jitter := (rand.Float64() + 0.5) * b.CheckInterval.Seconds() / 2
		nextInterval = time.Duration(jitter) * time.Second
		b.Logger.Debug("next-interval", lager.Data{"seconds": nextInterval.Seconds()})
	}
}

func (b *BandwidthExperiment) run() {
	logger := b.Logger.Session("bandwidth-experiment")

	candidates := b.Peers.Snapshot(logger)
	if len(candidates) < 1 {
		return
	}

	target := candidates[rand.Intn(len(candidates))].Host
	logger = logger.WithData(lager.Data{"target": target})

	result, err := b.Client.TestBandwidth(logger, target, b.PayloadSize)
	if err != nil {
		logger.Error("test-bandwidth", err)
		return
	}

	b.ReportAvgBandwidth(result.AvgBandwidth)
	logger.Debug("done", lager.Data{"avg-bandwidth": result.AvgBandwidth})
}
