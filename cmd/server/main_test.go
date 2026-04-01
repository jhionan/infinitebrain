package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildMux_HealthLive_Returns200(t *testing.T) {
	srv := httptest.NewServer(buildMux())
	defer srv.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/health/live", nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /health/live: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Fatalf("reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestBuildMux_HealthReady_Returns200(t *testing.T) {
	srv := httptest.NewServer(buildMux())
	defer srv.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/health/ready", nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}

	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /health/ready: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Fatalf("reading response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
