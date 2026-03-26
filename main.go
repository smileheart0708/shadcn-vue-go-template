package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"main/internal/config"
	"main/internal/database"
	"main/internal/httpapi"
	"main/internal/logging"
	"main/internal/users"
)

const shutdownTimeout = 10 * time.Second

func main() {
	logger, logStream := logging.NewWithStream(logging.StreamOptions{
		Capacity: logging.DefaultStreamCapacity,
	})
	slog.SetDefault(logger)

	if err := config.Load(); err != nil {
		slog.Error("failed to load environment", "error", err)
		return
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		return
	}

	frontendAssets, err := frontendFS()
	if err != nil {
		slog.Error("failed to load embedded frontend assets", "error", err)
		return
	}

	dataDir, err := filepath.Abs(cfg.DataDir)
	if err != nil {
		slog.Error("failed to resolve data directory", "error", err, "data_dir", cfg.DataDir)
		return
	}

	dbPath := filepath.Join(dataDir, cfg.DBName)
	listenAddr := fmt.Sprintf(":%d", cfg.Port)

	logging.LogStartupBanner(logger, listenAddr, dataDir)
	if cfg.UsesDefaultJWTSecret() {
		slog.Warn("using default JWT secret; set JWT_SECRET in production")
	}

	dbContainer, err := database.Open(context.Background(), database.Options{
		Path: dbPath,
	})
	if err != nil {
		slog.Error("failed to open database", "error", err)
		return
	}
	defer closeDB(dbContainer)

	slog.Info("database ready", "path", dbPath)

	userStore := users.NewStore(dbContainer.DB())
	if err := users.NewBootstrapManager(userStore, dataDir, logger).Ensure(context.Background()); err != nil {
		slog.Error("failed to bootstrap default super admin", "error", err)
		return
	}

	server := &http.Server{
		Addr: listenAddr,
		Handler: httpapi.NewHandlerWithOptions(httpapi.HandlerOptions{
			Logger:         logger,
			LogStream:      logStream,
			DB:             dbContainer.DB(),
			DataDir:        dataDir,
			FrontendFS:     frontendAssets,
			LogAPIRequests: cfg.APIRequestLogEnabled,
			Auth: httpapi.AuthOptions{
				Secret:             []byte(cfg.JWTSecret),
				TTL:                cfg.JWTTTL,
				RefreshIdleTTL:     cfg.RefreshIdleTTL,
				RefreshAbsoluteTTL: cfg.RefreshAbsoluteTTL,
			},
		}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		// SSE responses may stay open for a long time; rely on client cancellation and shutdown instead.
		WriteTimeout:   0,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	serverErrCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrCh <- err
			return
		}
		serverErrCh <- nil
	}()

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErrCh:
		if err != nil {
			slog.Error("http server stopped unexpectedly", "error", err)
		}
		return
	case <-signalCtx.Done():
		slog.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shut down http server", "error", err)
		return
	}

	if err := <-serverErrCh; err != nil {
		slog.Error("http server stopped unexpectedly", "error", err)
		return
	}

	slog.Info("application stopped")
}

func closeDB(dbContainer *database.DBContainer) {
	if dbContainer == nil {
		return
	}
	if err := dbContainer.Close(); err != nil {
		slog.Error("failed to close database", "error", err)
		return
	}
	slog.Info("database closed", "path", dbContainer.Path())
}
