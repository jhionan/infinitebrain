package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/rian/infinite_brain/pkg/config"
)

func buildTestMux() http.Handler {
	cfg := &config.Config{}
	cfg.Auth.JWTSecret = "test-secret-that-is-32chars-long!!"
	cfg.Auth.AccessTokenDuration = 15 * time.Minute
	cfg.Auth.RefreshTokenDuration = 7 * 24 * time.Hour
	cfg.Auth.ArgonPepper = "test-pepper-for-unit-tests-must-be-long"
	return buildMux(cfg, nil, zerolog.Nop())
}

func TestBuildMux_HealthLive_Returns200(t *testing.T) {
	srv := httptest.NewServer(buildTestMux())
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
	srv := httptest.NewServer(buildTestMux())
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

func TestParseAllowedOrigins_EmptyString_ReturnsEmptySlice(t *testing.T) {
	got := parseAllowedOrigins("")
	if len(got) != 0 {
		t.Errorf("got %v, want empty slice", got)
	}
}

func TestParseAllowedOrigins_CommaSeparated_TrimsWhitespace(t *testing.T) {
	got := parseAllowedOrigins("https://a.com , https://b.com")
	want := []string{"https://a.com", "https://b.com"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseAllowedOrigins_SingleOrigin_ReturnsSingleElement(t *testing.T) {
	got := parseAllowedOrigins("https://example.com")
	if len(got) != 1 || got[0] != "https://example.com" {
		t.Errorf("got %v, want [https://example.com]", got)
	}
}

func TestBuildMux_Metrics_Returns200(t *testing.T) {
	srv := httptest.NewServer(buildTestMux())
	defer srv.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/metrics", nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Fatalf("reading response: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestBuildMux_Honeypot_Returns404(t *testing.T) {
	srv := httptest.NewServer(buildTestMux())
	defer srv.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL+"/.env", nil)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("GET /.env: %v", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Fatalf("reading response: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestBuildMux_SecurityHeaders_Present(t *testing.T) {
	srv := httptest.NewServer(buildTestMux())
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

	headers := map[string]string{
		"X-Frame-Options":        "DENY",
		"X-Content-Type-Options": "nosniff",
	}
	for header, want := range headers {
		if got := resp.Header.Get(header); got != want {
			t.Errorf("%s = %q, want %q", header, got, want)
		}
	}
}

func TestBuildMux_RequestID_PresentInResponse(t *testing.T) {
	srv := httptest.NewServer(buildTestMux())
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
	if got := resp.Header.Get("X-Request-ID"); got == "" {
		t.Error("X-Request-ID header missing from response")
	}
}
