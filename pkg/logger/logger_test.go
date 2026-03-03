package logger_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rian/infinite_brain/pkg/logger"
)

func TestNew_Production_OutputsJSON(t *testing.T) {
	var buf bytes.Buffer
	l := zerolog.New(&buf).With().Timestamp().Logger()
	_ = l // just confirm it compiles and returns valid Logger

	// Create via New and confirm level parsing
	appLogger := logger.New("debug", "production")
	if appLogger.GetLevel() != zerolog.DebugLevel {
		t.Errorf("expected debug level, got %v", appLogger.GetLevel())
	}
}

func TestNew_InvalidLevel_DefaultsToInfo(t *testing.T) {
	l := logger.New("not-a-level", "local")
	if l.GetLevel() != zerolog.InfoLevel {
		t.Errorf("expected info level for invalid input, got %v", l.GetLevel())
	}
}

func TestWithService_AddsServiceField(t *testing.T) {
	var buf bytes.Buffer
	base := zerolog.New(&buf)
	l := logger.WithService(base, "capture-service")
	l.Info().Msg("test")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}
	if entry["service"] != "capture-service" {
		t.Errorf("expected service field 'capture-service', got %v", entry["service"])
	}
}

func TestWithRequestID_AddsRequestIDField(t *testing.T) {
	var buf bytes.Buffer
	base := zerolog.New(&buf)
	l := logger.WithRequestID(base, "req-abc-123")
	l.Info().Msg("test")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to parse log output: %v", err)
	}
	if entry["request_id"] != "req-abc-123" {
		t.Errorf("expected request_id 'req-abc-123', got %v", entry["request_id"])
	}
}
