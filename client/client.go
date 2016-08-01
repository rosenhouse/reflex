package client

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/rosenhouse/reflex/peer"
	"github.com/rosenhouse/reflex/science"
)

type Client struct {
	HTTPClient *http.Client
	Port       int

	ReportRoundTripLatency func(time.Duration)
}

func (c *Client) doAndUnmarshal(method, url string, requestBody io.Reader, result interface{}) error {
	req, err := http.NewRequest(method, url, requestBody)
	if err != nil {
		return err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return json.NewDecoder(resp.Body).Decode(result)
}

func (c *Client) doPeerSync(logger lager.Logger, method, url string) ([]peer.Glimpse, error) {
	startTime := time.Now()

	results := []peer.Glimpse{}
	err := c.doAndUnmarshal(method, url, nil, &results)
	if err != nil {
		return nil, err
	}

	roundTripLatency := time.Since(startTime)
	c.ReportRoundTripLatency(roundTripLatency)
	if roundTripLatency > time.Second {
		logger.Info("slow-round-trip", lager.Data{"seconds": roundTripLatency.Seconds()})
	}
	return results, err
}

func (c *Client) ReadLeader(logger lager.Logger, leader string) ([]peer.Glimpse, error) {
	url := fmt.Sprintf("http://%s/peers", leader)
	return c.doPeerSync(logger, "GET", url)
}

func (c *Client) PostAndReadSnapshot(logger lager.Logger, host string) ([]peer.Glimpse, error) {
	url := fmt.Sprintf("http://%s:%d/peers", host, c.Port)
	return c.doPeerSync(logger, "POST", url)
}

func (c *Client) TestBandwidth(logger lager.Logger, host string, payloadSize int64) (*science.BandwidthExperimentResult, error) {
	url := fmt.Sprintf("http://%s:%d/bandwidth", host, c.Port)

	localHasher := sha256.New()
	payload := io.TeeReader(io.LimitReader(rand.Reader, payloadSize), localHasher)
	results := &science.BandwidthExperimentResult{}

	logger.Debug("starting", lager.Data{"payload": payloadSize})
	err := c.doAndUnmarshal("POST", url, payload, results)
	if err != nil {
		return nil, err
	}

	localSHA256Sum := hex.EncodeToString(localHasher.Sum(nil))
	if localSHA256Sum != results.SHA256 {
		err := fmt.Errorf("sha mismatch")
		logger.Error("invalid-result", err, lager.Data{"local-sha256": localSHA256Sum, "remote-result": results})
		return nil, err
	}

	logger.Debug("complete")
	return results, nil
}
