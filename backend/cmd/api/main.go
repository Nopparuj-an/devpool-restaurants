// Command api runs the restaurants HTTP API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	_ "time/tzdata" // embed tz database so LoadLocation works in slim images (ADR-0007)

	"github.com/gin-gonic/gin"

	"restaurants/internal/platform/config"
	"restaurants/internal/platform/database"
	"restaurants/internal/platform/storage"
	"restaurants/internal/server"
)

func main() {
	// `api healthcheck` is for container healthchecks: the distroless image has no curl.
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func healthcheck() int {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	client := http.Client{Timeout: 3 * time.Second}
	res, err := client.Get("http://" + addr + "/api/healthz")
	if err != nil {
		return 1
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func run() error {
	// pgx returns timestamps in time.Local; the API speaks UTC only (R-TIME-1).
	time.Local = time.UTC
	if os.Getenv(gin.EnvGinMode) == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool); err != nil {
		return err
	}
	slog.Info("migrations applied")

	srv := &http.Server{
		Addr: cfg.HTTPAddr,
		Handler: server.New(pool, server.Options{
			CookieSecure: cfg.CookieSecure,
			Images:       storage.NewS3(cfg.S3),
			ImageBaseURL: cfg.ImageBaseURL,
			Google:       cfg.Google,
		}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.HTTPAddr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		slog.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
