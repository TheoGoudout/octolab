package main

import (
	"context"
	"log/slog"
	"net/http"
	"octolab/octoshim/middleware"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Config struct {
	GitLabURL       string
	GitLabPAT       string
	GitLabProjectID string
	LogLevel        string
	ListenAddr      string
}

func configFromEnv() Config {
	level := os.Getenv("BRIDGE_LOG_LEVEL")
	if level == "" {
		level = "info"
	}
	addr := os.Getenv("OCTOSHIM_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	return Config{
		GitLabURL:       os.Getenv("BRIDGE_GITLAB_URL"),
		GitLabPAT:       os.Getenv("BRIDGE_GITLAB_PAT"),
		GitLabProjectID: os.Getenv("BRIDGE_GITLAB_PROJECT_ID"),
		LogLevel:        level,
		ListenAddr:      addr,
	}
}

func main() {
	cfg := configFromEnv()

	var logLevel slog.Level
	switch cfg.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	default:
		logLevel = slog.LevelInfo
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})))

	if cfg.GitLabPAT == "" {
		slog.Error("BRIDGE_GITLAB_PAT is required")
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", dispatch)

	// Apply middleware: logging wraps auth
	var handler http.Handler = mux
	handler = middleware.AuthSwap(cfg.GitLabPAT)(handler)
	handler = middleware.RequestLogger(cfg.GitLabPAT)(handler)

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	slog.Info("octoshim starting", "addr", cfg.ListenAddr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("octoshim shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("graceful shutdown failed", "err", err)
	}
}
