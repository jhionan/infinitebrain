// Package health implements liveness and readiness probes.
package health

import "context"

// Probe is anything that can report its health via a Ping call.
type Probe interface {
	Ping(ctx context.Context) error
}

// ReadyResult holds the outcome of a readiness check.
type ReadyResult struct {
	OK     bool
	Checks map[string]bool
}

// Checker runs named probes and aggregates results.
type Checker struct {
	probes map[string]Probe
}

// Option configures a Checker.
type Option func(*Checker)

// WithProbe adds a named probe to the Checker.
func WithProbe(name string, p Probe) Option {
	return func(c *Checker) {
		c.probes[name] = p
	}
}

// NewChecker creates a Checker with the provided probes.
func NewChecker(opts ...Option) *Checker {
	c := &Checker{probes: make(map[string]Probe)}
	for _, o := range opts {
		o(c)
	}
	return c
}

// Live reports whether the process itself is alive.
// Always returns true — if the handler responds, the process is alive.
func (c *Checker) Live() bool { return true }

// Ready runs all probes and returns an aggregated result.
// The overall OK is true only if every probe succeeds.
func (c *Checker) Ready(ctx context.Context) ReadyResult {
	checks := make(map[string]bool, len(c.probes))
	allOK := true
	for name, p := range c.probes {
		ok := p.Ping(ctx) == nil
		checks[name] = ok
		if !ok {
			allOK = false
		}
	}
	return ReadyResult{OK: allOK, Checks: checks}
}
