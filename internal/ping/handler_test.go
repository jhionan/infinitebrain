package ping_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	pingv1 "github.com/rian/infinite_brain/api/gen/ping/v1"
	"github.com/rian/infinite_brain/api/gen/ping/v1/pingv1connect"
	"github.com/rian/infinite_brain/internal/ping"
)

func TestPingHandler_Ping_EchosMessage(t *testing.T) {
	path, handler := pingv1connect.NewPingServiceHandler(ping.NewHandler())
	mux := http.NewServeMux()
	mux.Handle(path, handler)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := pingv1connect.NewPingServiceClient(srv.Client(), srv.URL)
	resp, err := client.Ping(context.Background(), connect.NewRequest(&pingv1.PingRequest{
		Message: "hello",
	}))
	if err != nil {
		t.Fatalf("Ping: %v", err)
	}
	if resp.Msg.Message != "hello" {
		t.Errorf("expected 'hello', got %q", resp.Msg.Message)
	}
}

func TestPingHandler_Ping_EmptyMessage(t *testing.T) {
	path, handler := pingv1connect.NewPingServiceHandler(ping.NewHandler())
	mux := http.NewServeMux()
	mux.Handle(path, handler)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := pingv1connect.NewPingServiceClient(srv.Client(), srv.URL)
	resp, err := client.Ping(context.Background(), connect.NewRequest(&pingv1.PingRequest{}))
	if err != nil {
		t.Fatalf("Ping with empty message: %v", err)
	}
	if resp.Msg.Message != "" {
		t.Errorf("expected empty message, got %q", resp.Msg.Message)
	}
}
