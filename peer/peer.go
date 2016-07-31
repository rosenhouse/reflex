package peer

import (
	"os"
	"sort"
	"strings"
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
	UpsertUntrusted(logger lager.Logger, candidates []Glimpse)
	Snapshot(lager.Logger) []Glimpse
	RunCullerLoop(signals <-chan os.Signal, ready chan<- struct{}) error
}

func NewList(defaultTTL time.Duration, thisHost string) List {
	return &peerList{
		Lock:         &sync.Mutex{},
		Peers:        make(map[string]time.Time),
		DefaultTTL:   defaultTTL,
		ThisHost:     thisHost,
		CullInterval: defaultTTL / 2,
	}
}

type peerList struct {
	Lock         *sync.Mutex
	Peers        map[string]time.Time
	DefaultTTL   time.Duration
	ThisHost     string
	CullInterval time.Duration
}

func (p *peerList) Upsert(logger lager.Logger, host string) {
	p.upsertWithTTL(logger, host, p.DefaultTTL)
}

func (p *peerList) upsertWithTTL(logger lager.Logger, host string, ttl time.Duration) {
	expireTime := time.Now().Add(ttl)
	p.Lock.Lock()
	defer p.Lock.Unlock()

	host = strings.TrimSpace(host)
	ttlSec := int(ttl.Seconds())
	if p.Peers[host].Before(expireTime) {
		p.Peers[host] = expireTime
		logger.Info("upserted", lager.Data{"host": host, "ttl": ttlSec})
	} else {
		logger.Debug("no-op-upsert", lager.Data{"host": host, "ignored-ttl": ttlSec})
	}
}

func (p *peerList) UpsertUntrusted(logger lager.Logger, candidates []Glimpse) {
	const distrustFactor = 2

	for _, candidate := range candidates {
		newTTL := time.Duration(candidate.TTL) * time.Second
		if newTTL > p.DefaultTTL {
			newTTL = p.DefaultTTL
		}
		newTTL /= distrustFactor
		p.upsertWithTTL(logger, candidate.Host, newTTL)
	}
}

func (p *peerList) Snapshot(logger lager.Logger) []Glimpse {
	p.Lock.Lock()
	defer p.Lock.Unlock()

	now := time.Now()

	results := []Glimpse{}
	for host, expiry := range p.Peers {
		if ttl := int(expiry.Sub(now).Seconds()); ttl > 0 {
			results = append(results, Glimpse{Host: host, TTL: ttl})
		}
	}

	sort.Sort(byTTL(results))
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
	culled[p.ThisHost] = time.Now().Add(p.DefaultTTL)

	p.Peers = culled
}

func (p *peerList) RunCullerLoop(signals <-chan os.Signal, ready chan<- struct{}) error {
	close(ready)

	for {
		select {
		case <-signals:
			return nil
		case <-time.After(p.CullInterval):
			p.cull()
		}
	}
}
