# Feature: Configuration System

**Task ID**: T-002
**Status**: completed
**Epic**: Foundation

## Goal

Load, validate, and expose all application configuration from environment variables.
Fail fast at startup if required variables are missing or invalid.

## Acceptance Criteria

- [x] `pkg/config/config.go` — struct-based config with all sections
- [x] All required vars cause startup failure with a clear error message
- [x] Optional vars have sensible defaults
- [x] JWT secret validated for minimum length (32 chars)
- [x] APP_ENV validated against allowed values
- [x] AI provider validated against allowed values
- [x] `IsProduction()` helper method
- [x] Unit tests covering success, defaults, and all validation failures
- [x] `configs/example.env` documents every variable

## Key Design Decisions

- No external config library dependency in first iteration — pure `os.Getenv`
- All required fields return empty string and errors are collected then surfaced in `validate()`
- Duration values use Go's `time.ParseDuration` so they're human-readable in env files

## Variables Reference

See `configs/example.env` for the full list.
