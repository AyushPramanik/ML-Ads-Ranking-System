// Command server runs the ads ranking HTTP API.
//
// It loads the CTR model and feature spec produced by the Python pipeline, wires
// up the ad store (PostgreSQL or in-memory) and feature cache (Redis or
// in-process), and serves the REST API with graceful shutdown on SIGINT/SIGTERM.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/plainvue/ml-ads-ranking/ranking/internal/api"
	"github.com/plainvue/ml-ads-ranking/ranking/internal/cache"
	"github.com/plainvue/ml-ads-ranking/ranking/internal/config"
	"github.com/plainvue/ml-ads-ranking/ranking/internal/features"
	"github.com/plainvue/ml-ads-ranking/ranking/internal/logging"
	"github.com/plainvue/ml-ads-ranking/ranking/internal/model"
	"github.com/plainvue/ml-ads-ranking/ranking/internal/ranking"
	"github.com/plainvue/ml-ads-ranking/ranking/internal/store"
)

func main() {
	healthCheck := flag.Bool("health-check", false, "probe /health and exit (for container healthchecks)")
	flag.Parse()

	if *healthCheck {
		if err := probeHealth(); err != nil {
			fmt.Fprintln(os.Stderr, "health check failed:", err)
			os.Exit(1)
		}
		return
	}

	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

// probeHealth performs an HTTP GET against the local /health endpoint. It is the
// container healthcheck for distroless images, which have no shell or curl.
func probeHealth() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	addr := cfg.HTTPAddr
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://" + addr + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := logging.New(cfg.LogLevel, cfg.LogFormat)
	slog.SetDefault(logger)

	mdl, err := model.Load(cfg.ModelPath)
	if err != nil {
		return err
	}
	logger.Info("model loaded",
		"path", cfg.ModelPath, "trees", len(mdl.Trees), "features", mdl.NumFeatures)

	spec, err := features.Load(cfg.FeatureSpecPath)
	if err != nil {
		return err
	}
	logger.Info("feature spec loaded", "version", spec.Version, "features", len(spec.FeatureColumns))

	// Bounded context for dependency startup.
	startupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	st, err := buildStore(startupCtx, cfg, logger)
	if err != nil {
		return err
	}
	defer st.Close()

	fc, err := buildCache(startupCtx, cfg, logger)
	if err != nil {
		return err
	}
	defer fc.Close()

	ranker := ranking.New(mdl, spec, st, fc)
	handlers := api.NewHandlers(ranker, st, spec, logger)
	router := api.NewRouter(handlers, logger)

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	return serve(srv, cfg.ShutdownTimeout, logger)
}

// buildStore selects the PostgreSQL store when a DSN is configured, otherwise an
// in-memory seeded catalog.
func buildStore(ctx context.Context, cfg config.Config, logger *slog.Logger) (store.Store, error) {
	if cfg.PostgresDSN == "" {
		logger.Warn("no postgres DSN configured; using in-memory ad catalog")
		return store.NewSeededMemoryStore(), nil
	}
	st, err := store.NewPostgresStore(ctx, cfg.PostgresDSN)
	if err != nil {
		return nil, err
	}
	logger.Info("connected to postgres")
	return st, nil
}

// buildCache selects the Redis cache when an address is configured, otherwise an
// in-process cache.
func buildCache(ctx context.Context, cfg config.Config, logger *slog.Logger) (cache.Cache, error) {
	if cfg.RedisAddr == "" {
		logger.Warn("no redis address configured; using in-process feature cache")
		return cache.NewMemoryCache(cfg.FeatureCacheTTL), nil
	}
	c, err := cache.NewRedisCache(ctx, cfg.RedisAddr, cfg.FeatureCacheTTL)
	if err != nil {
		return nil, err
	}
	logger.Info("connected to redis")
	return c, nil
}

// serve runs the HTTP server until an interrupt signal arrives, then drains
// in-flight requests within the shutdown timeout.
func serve(srv *http.Server, shutdownTimeout time.Duration, logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("shutdown signal received; draining connections")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		logger.Info("server stopped cleanly")
		return nil
	}
}
