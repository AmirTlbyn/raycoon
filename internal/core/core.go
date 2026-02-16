package core

import (
	"context"
	"io"

	"raycoon/internal/core/types"
)

// ProxyCore defines the interface for proxy core implementations
type ProxyCore interface {
	// Lifecycle
	Start(ctx context.Context, config *types.CoreConfig) error
	Stop(ctx context.Context) error
	Restart(ctx context.Context, config *types.CoreConfig) error

	// Status
	IsRunning() bool
	GetStatus() (*types.Status, error)
	GetVersion() (string, error)

	// Stats
	GetStats() (*types.Stats, error)

	// Logs
	GetLogs() (io.Reader, error)
}
