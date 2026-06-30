package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	// Ensure a clean environment for the variables under test.
	for _, k := range []string{
		"RANKING_HTTP_ADDR", "RANKING_READ_TIMEOUT", "RANKING_POSTGRES_DSN",
		"RANKING_REDIS_ADDR", "RANKING_FEATURE_CACHE_TTL",
	} {
		t.Setenv(k, "")
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.ReadTimeout != 5*time.Second {
		t.Errorf("ReadTimeout = %v, want 5s", cfg.ReadTimeout)
	}
	if cfg.FeatureCacheTTL != 5*time.Minute {
		t.Errorf("FeatureCacheTTL = %v, want 5m", cfg.FeatureCacheTTL)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("RANKING_HTTP_ADDR", ":9090")
	t.Setenv("RANKING_SHUTDOWN_TIMEOUT", "30s")
	t.Setenv("RANKING_POSTGRES_DSN", "postgres://localhost/db")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":9090" {
		t.Errorf("HTTPAddr = %q, want :9090", cfg.HTTPAddr)
	}
	if cfg.ShutdownTimeout != 30*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 30s", cfg.ShutdownTimeout)
	}
	if cfg.PostgresDSN != "postgres://localhost/db" {
		t.Errorf("PostgresDSN = %q", cfg.PostgresDSN)
	}
}

func TestLoadRejectsBadDuration(t *testing.T) {
	t.Setenv("RANKING_READ_TIMEOUT", "not-a-duration")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid duration")
	}
}
