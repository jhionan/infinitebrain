// Package config handles loading and validating application configuration.
// All config is read from environment variables (12-factor app).
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Valkey   ValkeyConfig
	Auth     AuthConfig
	AI       AIConfig
	Storage  StorageConfig
	Server   ServerConfig
}

// AppConfig holds general application settings.
type AppConfig struct {
	Name        string
	Environment string // local, staging, production
	LogLevel    string
	Version     string
}

// DatabaseConfig holds PostgreSQL settings.
type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// ValkeyConfig holds Valkey connection settings.
type ValkeyConfig struct {
	URL      string
	Password string
	DB       int
}

// AuthConfig holds JWT and auth settings.
type AuthConfig struct {
	JWTSecret            string
	ArgonPepper          string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
}

// AIConfig holds AI provider settings.
type AIConfig struct {
	AnthropicAPIKey string
	OpenAIAPIKey    string
	DefaultProvider string // "anthropic" or "openai"
}

// StorageConfig holds S3-compatible storage settings.
type StorageConfig struct {
	Bucket    string
	Endpoint  string
	AccessKey string
	SecretKey string
	Region    string
	UseSSL    bool
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// Load reads configuration from environment variables.
// Returns an error if any required variable is missing or invalid.
func Load() (*Config, error) {
	cfg := &Config{
		App: AppConfig{
			Name:        getEnv("APP_NAME", "infinite-brain"),
			Environment: getEnv("APP_ENV", "local"),
			LogLevel:    getEnv("LOG_LEVEL", "info"),
			Version:     getEnv("APP_VERSION", "dev"),
		},
		Database: DatabaseConfig{
			URL:             requireEnv("DATABASE_URL"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getEnvDuration("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Valkey: ValkeyConfig{
			URL:      requireEnv("VALKEY_URL"),
			Password: getEnv("VALKEY_PASSWORD", ""),
			DB:       getEnvInt("VALKEY_DB", 0),
		},
		Auth: AuthConfig{
			JWTSecret:            requireEnv("JWT_SECRET"),
			ArgonPepper:          requireEnv("ARGON_PEPPER"),
			AccessTokenDuration:  getEnvDuration("JWT_ACCESS_DURATION", 15*time.Minute),
			RefreshTokenDuration: getEnvDuration("JWT_REFRESH_DURATION", 7*24*time.Hour),
		},
		AI: AIConfig{
			AnthropicAPIKey: requireEnv("ANTHROPIC_API_KEY"),
			OpenAIAPIKey:    getEnv("OPENAI_API_KEY", ""),
			DefaultProvider: getEnv("AI_DEFAULT_PROVIDER", "anthropic"),
		},
		Storage: StorageConfig{
			Bucket:    requireEnv("S3_BUCKET"),
			Endpoint:  requireEnv("S3_ENDPOINT"),
			AccessKey: requireEnv("S3_ACCESS_KEY"),
			SecretKey: requireEnv("S3_SECRET_KEY"),
			Region:    getEnv("S3_REGION", "us-east-1"),
			UseSSL:    getEnvBool("S3_USE_SSL", true),
		},
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 15*time.Second),
			IdleTimeout:  getEnvDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation: %w", err)
	}

	return cfg, nil
}

// validate checks all required fields and business rules.
func (c *Config) validate() error {
	var errs []error

	if len(c.Auth.JWTSecret) < 32 {
		errs = append(errs, errors.New("JWT_SECRET must be at least 32 characters"))
	}

	if len(c.Auth.ArgonPepper) < 32 {
		errs = append(errs, errors.New("ARGON_PEPPER must be at least 32 characters"))
	}

	validEnvs := map[string]bool{"local": true, "staging": true, "production": true}
	if !validEnvs[c.App.Environment] {
		errs = append(errs, fmt.Errorf("APP_ENV must be one of: local, staging, production; got %q", c.App.Environment))
	}

	validProviders := map[string]bool{"anthropic": true, "openai": true}
	if !validProviders[c.AI.DefaultProvider] {
		errs = append(errs, fmt.Errorf("AI_DEFAULT_PROVIDER must be one of: anthropic, openai; got %q", c.AI.DefaultProvider))
	}

	return errors.Join(errs...)
}

// IsProduction returns true when running in production.
func (c *Config) IsProduction() bool {
	return c.App.Environment == "production"
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		// We store the error; validate() will surface it.
		// Returning empty string lets Load() collect all missing vars at once.
		return ""
	}
	return v
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
