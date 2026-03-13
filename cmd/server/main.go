package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"md/internal/api"
	"md/internal/cache"
	"md/internal/config"
	"md/internal/storage"
)

var (
	Version   = "dev"
	GitSHA    = "unknown"
	BuildDate = "unknown"
)

func main() {
	// Structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("starting MD",
		"version", Version,
		"git_sha", GitSHA,
		"build_date", BuildDate,
	)

	// Configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Ensure storage directory exists
	if err := os.MkdirAll(cfg.StoragePath, 0755); err != nil {
		slog.Error("failed to create storage directory", "path", cfg.StoragePath, "error", err)
		os.Exit(1)
	}

	// Redis client (optional)
	var redisClient *cache.Client
	if cfg.RedisURL != "" {
		redisClient, err = cache.New(cfg.RedisURL)
		if err != nil {
			slog.Warn("redis connection failed, cache disabled", "error", err)
		} else {
			slog.Info("redis connected", "host", cfg.RedisHost())
		}
	}

	// File storage
	fileStore := storage.New(cfg.StoragePath)

	// HTTP router
	server := api.NewRouter(cfg, fileStore, redisClient, Version)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           server.Handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      120 * time.Second, // longer for PDF export
		IdleTimeout:       120 * time.Second,
	}

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("http server listening", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http server error", "error", err)
			os.Exit(1)
		}
	}()

	<-stop
	slog.Info("shutting down gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	server.Shutdown()

	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			slog.Warn("redis close error", "error", err)
		}
	}

	fmt.Println("md stopped")
}
