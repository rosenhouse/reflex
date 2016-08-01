package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"code.cloudfoundry.org/lager"

	"github.com/rosenhouse/reflex/peer"
)

type Client struct {
	HTTPClient *http.Client
	Port       int

	ReportRoundTripLatency func(time.Duration)
}

func (c *Client) doAndGetResults(logger lager.Logger, method, url string) ([]peer.Glimpse, error) {
	startTime := time.Now()

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	results := []peer.Glimpse{}
	err = json.NewDecoder(resp.Body).Decode(&results)

	roundTripLatency := time.Since(startTime)
	c.ReportRoundTripLatency(roundTripLatency)
	if roundTripLatency > time.Second {
		logger.Info("slow-round-trip", lager.Data{"seconds": roundTripLatency.Seconds()})
	}
	return results, err
}

func (c *Client) ReadLeader(logger lager.Logger, leader string) ([]peer.Glimpse, error) {
	url := fmt.Sprintf("http://%s/peers", leader)
	return c.doAndGetResults(logger, "GET", url)
}

func (c *Client) PostAndReadSnapshot(logger lager.Logger, host string) ([]peer.Glimpse, error) {
	url := fmt.Sprintf("http://%s:%d/peers", host, c.Port)
	return c.doAndGetResults(logger, "POST", url)
}
