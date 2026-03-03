// Package logger provides a structured, zero-allocation logger for the application.
// It wraps zerolog with sensible defaults for both local development and production.
package logger

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Logger is the application logger type.
type Logger = zerolog.Logger

// New creates a logger configured for the given environment and log level.
//
// In non-production environments, output is human-readable (console format).
// In production, output is structured JSON for log aggregation.
func New(level, environment string) Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	zerolog.TimeFieldFormat = time.RFC3339

	var writer io.Writer
	if environment != "production" {
		writer = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
		}
	} else {
		writer = os.Stdout
	}

	return zerolog.New(writer).
		Level(lvl).
		With().
		Timestamp().
		Caller().
		Logger()
}

// WithService returns a logger with the service name added as a field.
func WithService(l Logger, service string) Logger {
	return l.With().Str("service", service).Logger()
}

// WithRequestID returns a logger with the request ID added as a field.
func WithRequestID(l Logger, requestID string) Logger {
	return l.With().Str("request_id", requestID).Logger()
}
