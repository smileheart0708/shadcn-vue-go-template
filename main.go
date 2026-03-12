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
)

const shutdownTimeout = 10 * time.Second

func main() {
	logger := logging.New()
	slog.SetDefault(logger)

	dataDir, err := filepath.Abs(config.DataDir)
	if err != nil {
		slog.Error("failed to resolve data directory", "error", err, "data_dir", config.DataDir)
		return
	}

	dbPath := filepath.Join(dataDir, config.DBName)
	listenAddr := fmt.Sprintf(":%d", config.Port)
	logging.LogStartupBanner(logger, listenAddr, dataDir)
	slog.Info("starting application", "data_dir", dataDir, "db_path", dbPath)

	dbContainer, err := database.Open(context.Background(), database.Options{
		Path: dbPath,
	})
	if err != nil {
		slog.Error("failed to open database", "error", err)
		return
	}
	defer closeDB(dbContainer)

	slog.Info("database ready", "path", dbPath)

	server := &http.Server{
		Addr:              listenAddr,
		Handler:           httpapi.NewHandler(logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	serverErrCh := make(chan error, 1)
	go func() {
		slog.Info("http server listening", "addr", listenAddr)
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
