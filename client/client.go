package client

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rosenhouse/reflex/peer"
)

type Client struct {
	HTTPClient *http.Client
	Port       int
}

func (c *Client) doAndGetResults(method, url string) ([]peer.Glimpse, error) {
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
	return results, err
}

func (c *Client) ReadLeader(leader string) ([]peer.Glimpse, error) {
	url := fmt.Sprintf("http://%s/peers", leader)
	return c.doAndGetResults("GET", url)
}

func (c *Client) PostAndReadSnapshot(host string) ([]peer.Glimpse, error) {
	url := fmt.Sprintf("http://%s:%d/peers", host, c.Port)
	return c.doAndGetResults("POST", url)
}
