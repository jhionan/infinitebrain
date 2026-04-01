package cache_test

import (
	"context"
	"testing"

	"github.com/rian/infinite_brain/pkg/cache"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
)

// newTestValkey starts a Valkey container and returns its address (host:port).
func newTestValkey(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	ctr, err := redis.Run(ctx,
		"valkey/valkey:9-alpine",
		testcontainers.WithWaitStrategy(
			wait.ForLog("Ready to accept connections"),
		),
	)
	if err != nil {
		t.Fatalf("start valkey container: %v", err)
	}
	t.Cleanup(func() {
		if err := ctr.Terminate(ctx); err != nil {
			t.Logf("terminate container: %v", err)
		}
	})

	addr, err := ctr.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("get endpoint: %v", err)
	}
	return addr
}

func TestNew_ConnectsSuccessfully(t *testing.T) {
	addr := newTestValkey(t)

	client, err := cache.New(context.Background(), cache.Config{
		Addr: addr,
	})
	if err != nil {
		t.Fatalf("cache.New: %v", err)
	}
	defer client.Close()

	if err := client.Ping(context.Background()); err != nil {
		t.Errorf("Ping: %v", err)
	}
}

func TestNew_InvalidAddr_ReturnsError(t *testing.T) {
	_, err := cache.New(context.Background(), cache.Config{
		Addr: "localhost:1",
	})
	if err == nil {
		t.Error("expected error for unreachable addr, got nil")
	}
}

func TestSetGet_RoundTrips(t *testing.T) {
	addr := newTestValkey(t)
	client, err := cache.New(context.Background(), cache.Config{Addr: addr})
	if err != nil {
		t.Fatalf("cache.New: %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	if err := client.Set(ctx, "hello", "world", 0); err != nil {
		t.Fatalf("Set: %v", err)
	}

	val, err := client.Get(ctx, "hello")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != "world" {
		t.Errorf("expected 'world', got %q", val)
	}
}
