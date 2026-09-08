package plugins

import (
	"sync"
	"time"
)

// Probe is one peer's cached connectivity state. An empty Conn is a cold
// cache, which the callers render optimistically rather than blanking a
// peer's surfaces before the first probe lands.
type Probe struct {
	Conn      string // "" | "ok" | "down"
	Version   string
	CheckedAt time.Time
}

// ProbeTTL bounds how often a peer's /health is re-checked, so a settings
// render never fans out a probe per row per navigation.
const ProbeTTL = 10 * time.Second

// Peers is what monbooru keeps about the plugins it has paired with: the
// child process of every managed one, and one cached probe result each.
// Button rendering gates on the cache, so a peer that went away stops
// offering surfaces that would only fail.
type Peers struct {
	*Supervisor

	mu     sync.Mutex
	probes map[string]Probe
}

// NewPeers returns a Peers with nothing running and a cold cache.
func NewPeers(callbackURL func() string, done <-chan struct{}) *Peers {
	return &Peers{Supervisor: New(callbackURL, done), probes: map[string]Probe{}}
}

// ProbeSeed is what the last probe said, or the zero Probe for a peer
// nothing has reached yet.
func (p *Peers) ProbeSeed(name string) Probe {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.probes[name]
}

// ClearProbe forgets what a peer's last probe said, leaving the cold
// cache's optimistic reading until the next one lands.
func (p *Peers) ClearProbe(name string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.probes, name)
}

func (p *Peers) SetProbe(name string, pr Probe) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.probes[name] = pr
}

// MarkDown records a failed call so the peer's surfaces go inert without
// waiting for the next scheduled probe.
func (p *Peers) MarkDown(name string) {
	p.SetProbe(name, Probe{Conn: "down", CheckedAt: time.Now()})
}

// Stale reports whether a peer's cached state has aged out of the TTL and
// is worth re-probing.
func (p *Peers) Stale(name string) bool {
	return time.Since(p.ProbeSeed(name).CheckedAt) >= ProbeTTL
}
