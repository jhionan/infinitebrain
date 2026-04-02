package health_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rian/infinite_brain/internal/health"
)

// fakeProbe implements health.Probe for testing.
type fakeProbe struct {
	err error
}

func (f *fakeProbe) Ping(_ context.Context) error { return f.err }

var errFake = errors.New("fake error")

func TestChecker_Ready_AllHealthy(t *testing.T) {
	c := health.NewChecker(
		health.WithProbe("db", &fakeProbe{}),
		health.WithProbe("valkey", &fakeProbe{}),
	)

	result := c.Ready(context.Background())
	if !result.OK {
		t.Errorf("expected OK=true, got false; checks=%v", result.Checks)
	}
	for name, ok := range result.Checks {
		if !ok {
			t.Errorf("probe %q reported unhealthy", name)
		}
	}
}

func TestChecker_Ready_OneUnhealthy(t *testing.T) {
	c := health.NewChecker(
		health.WithProbe("db", &fakeProbe{}),
		health.WithProbe("valkey", &fakeProbe{err: errFake}),
	)

	result := c.Ready(context.Background())
	if result.OK {
		t.Error("expected OK=false when a probe fails")
	}
	if result.Checks["valkey"] {
		t.Error("expected valkey to be false")
	}
	if !result.Checks["db"] {
		t.Error("expected db to be true")
	}
}

func TestChecker_Live_AlwaysTrue(t *testing.T) {
	c := health.NewChecker()
	if !c.Live() {
		t.Error("Live() must always return true for a running process")
	}
}
