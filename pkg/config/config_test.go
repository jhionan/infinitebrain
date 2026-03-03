package config_test

import (
	"testing"
	"time"

	"github.com/rian/infinite_brain/pkg/config"
)

func setTestEnv(t *testing.T, env map[string]string) {
	t.Helper()
	for k, v := range env {
		t.Setenv(k, v)
	}
}

func validEnv() map[string]string {
	return map[string]string{
		"DATABASE_URL":      "postgres://user:pass@localhost:5432/testdb",
		"REDIS_URL":         "redis://localhost:6379",
		"JWT_SECRET":        "super-secret-key-that-is-at-least-32-chars!!",
		"ANTHROPIC_API_KEY": "sk-ant-test",
		"S3_BUCKET":         "test-bucket",
		"S3_ENDPOINT":       "http://localhost:9000",
		"S3_ACCESS_KEY":     "minioadmin",
		"S3_SECRET_KEY":     "minioadmin",
	}
}

func TestLoad_Success_WithValidEnv(t *testing.T) {
	setTestEnv(t, validEnv())

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected config, got nil")
	}
}

func TestLoad_Defaults_AppliedWhenEnvNotSet(t *testing.T) {
	setTestEnv(t, validEnv())

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.App.Environment != "local" {
		t.Errorf("expected default env 'local', got %q", cfg.App.Environment)
	}
	if cfg.Server.Port != "8080" {
		t.Errorf("expected default port '8080', got %q", cfg.Server.Port)
	}
	if cfg.Auth.AccessTokenDuration != 15*time.Minute {
		t.Errorf("expected 15m access token, got %v", cfg.Auth.AccessTokenDuration)
	}
}

func TestLoad_Failure_JWTSecretTooShort(t *testing.T) {
	env := validEnv()
	env["JWT_SECRET"] = "short"
	setTestEnv(t, env)

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for short JWT secret, got nil")
	}
}

func TestLoad_Failure_InvalidEnvironment(t *testing.T) {
	env := validEnv()
	env["APP_ENV"] = "development" // not valid
	setTestEnv(t, env)

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid APP_ENV, got nil")
	}
}

func TestLoad_Failure_InvalidAIProvider(t *testing.T) {
	env := validEnv()
	env["AI_DEFAULT_PROVIDER"] = "gemini"
	setTestEnv(t, env)

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid AI provider, got nil")
	}
}

func TestIsProduction_ReturnsTrueForProductionEnv(t *testing.T) {
	env := validEnv()
	env["APP_ENV"] = "production"
	setTestEnv(t, env)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.IsProduction() {
		t.Error("expected IsProduction() to return true")
	}
}

func TestIsProduction_ReturnsFalseForLocalEnv(t *testing.T) {
	setTestEnv(t, validEnv())

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.IsProduction() {
		t.Error("expected IsProduction() to return false for local env")
	}
}
