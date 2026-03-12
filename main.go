package main

import (
	"context"
	"log/slog"
	"path/filepath"

	"main/internal/config"
	"main/internal/database"
)

func main() {
	dbPath := filepath.Join(config.DataDir, config.DBName)
	slog.Info("starting application", "data_dir", config.DataDir, "db_path", dbPath)

	_, err := database.Open(context.Background(), database.Options{
		Path: dbPath,
	})
	if err != nil {
		slog.Error("failed to open database", "error", err)
		return
	}
	slog.Info("database ready", "path", dbPath)
}
