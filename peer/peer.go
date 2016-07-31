package peer

import (
	"os"
	"sort"
	"sync"
	"time"

	"code.cloudfoundry.org/lager"
)

type Glimpse struct {
	Host string
	TTL  int
}

type byTTL []Glimpse

func (s byTTL) Len() int {
	return len(s)
}
func (s byTTL) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s byTTL) Less(i, j int) bool {
	return s[i].TTL < s[j].TTL
}

type List interface {
	Upsert(lager.Logger, string)
	Snapshot(lager.Logger) []Glimpse
	RunCullerLoop(signals <-chan os.Signal, ready chan<- struct{}) error
}

func NewList(defaultTTL time.Duration, thisHost string) List {
	return &peerList{
		Lock:       &sync.Mutex{},
		Peers:      make(map[string]time.Time),
		DefaultTTL: defaultTTL,
		ThisHost:   thisHost,
	}
}

type peerList struct {
	Lock       *sync.Mutex
	Peers      map[string]time.Time
	DefaultTTL time.Duration
	ThisHost   string
}

func (p *peerList) self() Glimpse {
	return Glimpse{
		Host: p.ThisHost,
		TTL:  int(p.DefaultTTL.Seconds()),
	}
}

func (p *peerList) Upsert(logger lager.Logger, host string) {
	expireTime := time.Now().Add(p.DefaultTTL)
	p.Lock.Lock()
	defer p.Lock.Unlock()

	p.Peers[host] = expireTime
	logger.Info("upserted", lager.Data{"host": host, "expires": expireTime})
}

func (p *peerList) Snapshot(logger lager.Logger) []Glimpse {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	now := time.Now()

	results := []Glimpse{}
	for host, expiry := range p.Peers {
		ttl := expiry.Sub(now)
		results = append(results, Glimpse{Host: host, TTL: int(ttl.Seconds())})
	}

	sort.Sort(byTTL(results))
	results = append(results, p.self())
	logger.Debug("snapshot", lager.Data{"peers": results})
	return results
}

func (p *peerList) cull() {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	cutOff := time.Now()
	culled := make(map[string]time.Time)

	for host, expiry := range p.Peers {
		if expiry.After(cutOff) {
			culled[host] = expiry
		}
	}

	p.Peers = culled
}

func (p *peerList) RunCullerLoop(signals <-chan os.Signal, ready chan<- struct{}) error {
	pollInterval := 5 * time.Second
	close(ready)

	for {
		select {
		case <-signals:
			return nil
		case <-time.After(pollInterval):
			p.cull()
		}
	}
}
