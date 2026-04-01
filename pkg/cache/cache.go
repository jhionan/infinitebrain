// Package cache provides a Valkey (Redis-protocol) connection pool.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/valkey-io/valkey-go"
)

// Config holds Valkey connection settings.
type Config struct {
	Addr     string
	Password string
}

// Client wraps a valkey client with a minimal interface.
type Client struct {
	v valkey.Client
}

// New creates a Valkey client and verifies connectivity via PING.
// Returns an error if the server is unreachable.
func New(ctx context.Context, cfg Config) (*Client, error) {
	v, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{cfg.Addr},
		Password:    cfg.Password,
	})
	if err != nil {
		return nil, fmt.Errorf("create valkey client: %w", err)
	}

	c := &Client{v: v}
	if err := c.Ping(ctx); err != nil {
		v.Close()
		return nil, fmt.Errorf("valkey ping: %w", err)
	}

	return c, nil
}

// Ping verifies the connection is alive.
func (c *Client) Ping(ctx context.Context) error {
	if err := c.v.Do(ctx, c.v.B().Ping().Build()).Error(); err != nil {
		return fmt.Errorf("valkey PING: %w", err)
	}
	return nil
}

// Set stores a key-value pair with an optional TTL (0 = no expiry).
func (c *Client) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	cmd := c.v.B().Set().Key(key).Value(value)
	var err error
	if ttl > 0 {
		err = c.v.Do(ctx, cmd.Ex(ttl).Build()).Error()
	} else {
		err = c.v.Do(ctx, cmd.Build()).Error()
	}
	if err != nil {
		return fmt.Errorf("valkey SET %q: %w", key, err)
	}
	return nil
}

// Get retrieves a value by key. Returns an error if the key does not exist.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	val, err := c.v.Do(ctx, c.v.B().Get().Key(key).Build()).ToString()
	if err != nil {
		return "", fmt.Errorf("valkey GET %q: %w", key, err)
	}
	return val, nil
}

// Close releases all connections.
func (c *Client) Close() {
	c.v.Close()
}
