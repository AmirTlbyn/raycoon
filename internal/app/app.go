package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"raycoon/internal/config/parser"
	"raycoon/internal/core/tun"
	"raycoon/internal/storage"
	"raycoon/internal/storage/sqlite"
)

// App represents the application context
type App struct {
	Storage storage.Storage
	Parser  *parser.Registry
	Config  *Config
}

// Config represents application configuration
type Config struct {
	DBPath string
}

// New creates a new application instance
func New() (*App, error) {
	// Get default config directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "raycoon")
	dataDir := filepath.Join(homeDir, ".local", "share", "raycoon")

	// Create directories if they don't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Initialize storage
	dbPath := filepath.Join(dataDir, "raycoon.db")
	store, err := sqlite.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Clean up stale TUN state from any previous crash.
	tun.CleanupIfNeeded()

	// Initialize parser registry
	parserRegistry := parser.NewRegistry()

	app := &App{
		Storage: store,
		Parser:  parserRegistry,
		Config: &Config{
			DBPath: dbPath,
		},
	}

	// Ensure global group exists
	if err := app.ensureGlobalGroup(); err != nil {
		return nil, fmt.Errorf("failed to ensure global group: %w", err)
	}

	return app, nil
}

// Close closes the application and releases resources
func (a *App) Close() error {
	if a.Storage != nil {
		return a.Storage.Close()
	}
	return nil
}

func (a *App) ensureGlobalGroup() error {
	ctx := context.Background()
	_, err := a.Storage.GetGlobalGroup(ctx)
	if err != nil {
		// Global group should be created by migrations, but check anyway
		return fmt.Errorf("global group not found: %w", err)
	}
	return nil
}
