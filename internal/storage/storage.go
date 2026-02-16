package storage

import (
	"context"
	"raycoon/internal/storage/models"
)

// Storage defines the interface for data persistence
type Storage interface {
	// Group operations
	CreateGroup(ctx context.Context, group *models.Group) error
	GetGroup(ctx context.Context, id int64) (*models.Group, error)
	GetGroupByName(ctx context.Context, name string) (*models.Group, error)
	GetAllGroups(ctx context.Context) ([]*models.Group, error)
	UpdateGroup(ctx context.Context, group *models.Group) error
	DeleteGroup(ctx context.Context, id int64) error
	GetGlobalGroup(ctx context.Context) (*models.Group, error)
	GetDueGroups(ctx context.Context) ([]*models.Group, error) // Groups with subscriptions due for update

	// Config operations
	CreateConfig(ctx context.Context, config *models.Config) error
	GetConfig(ctx context.Context, id int64) (*models.Config, error)
	GetConfigByName(ctx context.Context, name string) (*models.Config, error)
	GetAllConfigs(ctx context.Context, filter ConfigFilter) ([]*models.Config, error)
	UpdateConfig(ctx context.Context, config *models.Config) error
	DeleteConfig(ctx context.Context, id int64) error
	DeleteConfigsByGroup(ctx context.Context, groupID int64, fromSubscriptionOnly bool) error

	// Latency operations
	RecordLatency(ctx context.Context, latency *models.LatencyTest) error
	GetLatestLatency(ctx context.Context, configID int64) (*models.LatencyTest, error)
	GetLatencyHistory(ctx context.Context, configID int64, limit int) ([]*models.LatencyTest, error)

	// Settings operations
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
	GetAllSettings(ctx context.Context) (map[string]string, error)

	// Active connection
	SetActiveConnection(ctx context.Context, conn *models.ActiveConnection) error
	GetActiveConnection(ctx context.Context) (*models.ActiveConnection, error)
	ClearActiveConnection(ctx context.Context) error

	// Transactions
	BeginTx(ctx context.Context) (Transaction, error)

	// Close closes the storage connection
	Close() error
}

// ConfigFilter represents filters for querying configs
type ConfigFilter struct {
	GroupID          *int64
	Protocol         *string
	Enabled          *bool
	FromSubscription *bool
	SearchTerm       string // Search in name, address, notes
	Tags             []string
}

// Transaction represents a database transaction
type Transaction interface {
	Commit() error
	Rollback() error
	Storage
}
