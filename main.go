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

	"main/internal/audit"
	"main/internal/auth"
	"main/internal/authorization"
	"main/internal/config"
	"main/internal/database"
	"main/internal/httpapi"
	"main/internal/identity"
	"main/internal/logging"
	"main/internal/setup"
	"main/internal/systemsettings"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	logger, logStream := logging.NewWithStream(logging.StreamOptions{
		Capacity:  logging.DefaultStreamCapacity,
		Retention: logging.DefaultRetention,
	})
	slog.SetDefault(logger)

	if err := config.Load(); err != nil {
		return fmt.Errorf("load environment: %w", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	frontendAssets, err := frontendFS()
	if err != nil {
		return fmt.Errorf("load embedded frontend assets: %w", err)
	}

	dataDir, err := filepath.Abs(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("resolve data directory %q: %w", cfg.DataDir, err)
	}

	dbPath := filepath.Join(dataDir, cfg.DBName)
	listenAddr := fmt.Sprintf(":%d", cfg.Port)

	logging.LogStartupBanner(logger, listenAddr, dataDir)

	dbContainer, err := database.Open(context.Background(), database.Options{
		Path: dbPath,
	})
	if err != nil {
		return fmt.Errorf("open database %q: %w", dbPath, err)
	}
	defer closeDB(dbContainer)

	slog.Info("database ready", "path", dbPath)

	if err := authorization.EnsureCatalog(context.Background(), dbContainer.DB()); err != nil {
		return fmt.Errorf("seed authorization catalog: %w", err)
	}

	authorizationService := authorization.NewService()
	identityService := identity.NewService(dbContainer.DB())
	systemSettingsService := systemsettings.NewService(dbContainer.DB())
	setupService := setup.NewService(dbContainer.DB(), identityService)
	auditService := audit.NewService(dbContainer.DB())
	authService := auth.NewService(auth.Options{
		Secret:             []byte(cfg.JWTSecret),
		TTL:                cfg.JWTTTL,
		RefreshIdleTTL:     cfg.RefreshIdleTTL,
		RefreshAbsoluteTTL: cfg.RefreshAbsoluteTTL,
		RefreshCookieName:  cfg.RefreshCookieName,
	}, dbContainer.DB(), identityService, authorizationService, systemSettingsService)

	server := &http.Server{
		Addr: listenAddr,
		Handler: httpapi.NewHandlerWithOptions(httpapi.HandlerOptions{
			Logger:         logger,
			LogStream:      logStream,
			Auth:           authService,
			Authorization:  authorizationService,
			Identity:       identityService,
			Setup:          setupService,
			SystemSettings: systemSettingsService,
			Audit:          auditService,
			DataDir:        dataDir,
			FrontendFS:     frontendAssets,
			LogAPIRequests: cfg.APIRequestLogEnabled,
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
			return fmt.Errorf("http server stopped unexpectedly: %w", err)
		}
		return nil
	case <-signalCtx.Done():
		slog.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	if err := <-serverErrCh; err != nil {
		return fmt.Errorf("http server stopped unexpectedly: %w", err)
	}

	slog.Info("application stopped")
	return nil
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
