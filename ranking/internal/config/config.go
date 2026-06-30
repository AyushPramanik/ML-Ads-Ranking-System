// Package config loads ranking-service settings from the environment.
//
// All variables use the RANKING_ prefix and have production-safe defaults so the
// binary runs with zero configuration in development.
package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds all runtime settings for the ranking service.
type Config struct {
	HTTPAddr        string
	LogLevel        string
	LogFormat       string
	ModelPath       string
	FeatureSpecPath string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration

	// PostgresDSN is optional. When empty, the service uses an in-memory ad
	// catalog seeded with sample data, so it runs without a database.
	PostgresDSN string

	// RedisAddr is optional. When empty, the service uses an in-process feature
	// cache instead of Redis.
	RedisAddr       string
	FeatureCacheTTL time.Duration
}

// Load resolves configuration from the environment.
func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        getEnv("RANKING_HTTP_ADDR", ":8080"),
		LogLevel:        getEnv("RANKING_LOG_LEVEL", "info"),
		LogFormat:       getEnv("RANKING_LOG_FORMAT", "json"),
		ModelPath:       getEnv("RANKING_MODEL_PATH", "../artifacts/model.json"),
		FeatureSpecPath: getEnv("RANKING_FEATURE_SPEC_PATH", "../artifacts/feature_spec.json"),
		PostgresDSN:     getEnv("RANKING_POSTGRES_DSN", ""),
		RedisAddr:       getEnv("RANKING_REDIS_ADDR", ""),
	}

	var err error
	if cfg.ReadTimeout, err = getDuration("RANKING_READ_TIMEOUT", 5*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.WriteTimeout, err = getDuration("RANKING_WRITE_TIMEOUT", 10*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.ShutdownTimeout, err = getDuration("RANKING_SHUTDOWN_TIMEOUT", 15*time.Second); err != nil {
		return Config{}, err
	}
	if cfg.FeatureCacheTTL, err = getDuration("RANKING_FEATURE_CACHE_TTL", 5*time.Minute); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) (time.Duration, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid duration for %s: %w", key, err)
	}
	return d, nil
}
