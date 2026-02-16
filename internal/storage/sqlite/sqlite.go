package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"raycoon/internal/storage"
	"raycoon/internal/storage/models"
)

// dbHandle is the common interface between *sql.DB and *sql.Tx.
type dbHandle interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// DB implements the Storage interface using SQLite
type DB struct {
	db *sql.DB
}

// New creates a new SQLite storage instance
func New(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	storage := &DB{db: db}

	// Run migrations
	if err := runMigrations(storage); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return storage, nil
}

// Close closes the database connection
func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) handle() dbHandle { return d.db }

// BeginTx starts a new transaction
func (d *DB) BeginTx(ctx context.Context) (storage.Transaction, error) {
	tx, err := d.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &Tx{tx: tx}, nil
}

// Tx implements the Transaction interface
type Tx struct {
	tx *sql.Tx
}

func (t *Tx) Commit() error   { return t.tx.Commit() }
func (t *Tx) Rollback() error { return t.tx.Rollback() }
func (t *Tx) handle() dbHandle { return t.tx }

func (t *Tx) BeginTx(ctx context.Context) (storage.Transaction, error) {
	return nil, fmt.Errorf("nested transactions not supported")
}

func (t *Tx) Close() error { return nil }

// ─── Group operations ───────────────────────────────────────────────────────

func (d *DB) CreateGroup(ctx context.Context, group *models.Group) error {
	return createGroup(ctx, d.handle(), group)
}
func (t *Tx) CreateGroup(ctx context.Context, group *models.Group) error {
	return createGroup(ctx, t.handle(), group)
}

func createGroup(ctx context.Context, h dbHandle, group *models.Group) error {
	query := `
		INSERT INTO groups (name, description, is_global, subscription_url, auto_update, update_interval, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	result, err := h.ExecContext(ctx, query,
		group.Name, group.Description, group.IsGlobal, group.SubscriptionURL,
		group.AutoUpdate, group.UpdateInterval, group.UserAgent,
	)
	if err != nil {
		return fmt.Errorf("failed to create group: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	group.ID = id
	return nil
}

func (d *DB) GetGroup(ctx context.Context, id int64) (*models.Group, error) {
	return getGroup(ctx, d.handle(), id)
}
func (t *Tx) GetGroup(ctx context.Context, id int64) (*models.Group, error) {
	return getGroup(ctx, t.handle(), id)
}

func getGroup(ctx context.Context, h dbHandle, id int64) (*models.Group, error) {
	query := `
		SELECT id, name, description, is_global, subscription_url, auto_update, update_interval,
		       last_updated, next_update, user_agent, created_at, updated_at
		FROM groups WHERE id = ?
	`
	group := &models.Group{}
	err := h.QueryRowContext(ctx, query, id).Scan(
		&group.ID, &group.Name, &group.Description, &group.IsGlobal, &group.SubscriptionURL,
		&group.AutoUpdate, &group.UpdateInterval, &group.LastUpdated, &group.NextUpdate,
		&group.UserAgent, &group.CreatedAt, &group.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("group not found")
	}
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (d *DB) GetGroupByName(ctx context.Context, name string) (*models.Group, error) {
	return getGroupByName(ctx, d.handle(), name)
}
func (t *Tx) GetGroupByName(ctx context.Context, name string) (*models.Group, error) {
	return getGroupByName(ctx, t.handle(), name)
}

func getGroupByName(ctx context.Context, h dbHandle, name string) (*models.Group, error) {
	query := `
		SELECT id, name, description, is_global, subscription_url, auto_update, update_interval,
		       last_updated, next_update, user_agent, created_at, updated_at
		FROM groups WHERE name = ?
	`
	group := &models.Group{}
	err := h.QueryRowContext(ctx, query, name).Scan(
		&group.ID, &group.Name, &group.Description, &group.IsGlobal, &group.SubscriptionURL,
		&group.AutoUpdate, &group.UpdateInterval, &group.LastUpdated, &group.NextUpdate,
		&group.UserAgent, &group.CreatedAt, &group.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("group not found")
	}
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (d *DB) GetAllGroups(ctx context.Context) ([]*models.Group, error) {
	return getAllGroups(ctx, d.handle())
}
func (t *Tx) GetAllGroups(ctx context.Context) ([]*models.Group, error) {
	return getAllGroups(ctx, t.handle())
}

func getAllGroups(ctx context.Context, h dbHandle) ([]*models.Group, error) {
	query := `
		SELECT id, name, description, is_global, subscription_url, auto_update, update_interval,
		       last_updated, next_update, user_agent, created_at, updated_at
		FROM groups ORDER BY is_global DESC, name ASC
	`
	rows, err := h.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*models.Group
	for rows.Next() {
		group := &models.Group{}
		err := rows.Scan(
			&group.ID, &group.Name, &group.Description, &group.IsGlobal, &group.SubscriptionURL,
			&group.AutoUpdate, &group.UpdateInterval, &group.LastUpdated, &group.NextUpdate,
			&group.UserAgent, &group.CreatedAt, &group.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

func (d *DB) UpdateGroup(ctx context.Context, group *models.Group) error {
	return updateGroup(ctx, d.handle(), group)
}
func (t *Tx) UpdateGroup(ctx context.Context, group *models.Group) error {
	return updateGroup(ctx, t.handle(), group)
}

func updateGroup(ctx context.Context, h dbHandle, group *models.Group) error {
	query := `
		UPDATE groups
		SET name = ?, description = ?, subscription_url = ?, auto_update = ?, update_interval = ?,
		    last_updated = ?, next_update = ?, user_agent = ?
		WHERE id = ?
	`
	_, err := h.ExecContext(ctx, query,
		group.Name, group.Description, group.SubscriptionURL, group.AutoUpdate, group.UpdateInterval,
		group.LastUpdated, group.NextUpdate, group.UserAgent, group.ID,
	)
	return err
}

func (d *DB) DeleteGroup(ctx context.Context, id int64) error {
	return deleteGroup(ctx, d.handle(), id)
}
func (t *Tx) DeleteGroup(ctx context.Context, id int64) error {
	return deleteGroup(ctx, t.handle(), id)
}

func deleteGroup(ctx context.Context, h dbHandle, id int64) error {
	group, err := getGroup(ctx, h, id)
	if err != nil {
		return err
	}
	if group.IsGlobal {
		return fmt.Errorf("cannot delete global group")
	}
	_, err = h.ExecContext(ctx, "DELETE FROM groups WHERE id = ?", id)
	return err
}

func (d *DB) GetGlobalGroup(ctx context.Context) (*models.Group, error) {
	return getGlobalGroup(ctx, d.handle())
}
func (t *Tx) GetGlobalGroup(ctx context.Context) (*models.Group, error) {
	return getGlobalGroup(ctx, t.handle())
}

func getGlobalGroup(ctx context.Context, h dbHandle) (*models.Group, error) {
	query := `
		SELECT id, name, description, is_global, subscription_url, auto_update, update_interval,
		       last_updated, next_update, user_agent, created_at, updated_at
		FROM groups WHERE is_global = 1 LIMIT 1
	`
	group := &models.Group{}
	err := h.QueryRowContext(ctx, query).Scan(
		&group.ID, &group.Name, &group.Description, &group.IsGlobal, &group.SubscriptionURL,
		&group.AutoUpdate, &group.UpdateInterval, &group.LastUpdated, &group.NextUpdate,
		&group.UserAgent, &group.CreatedAt, &group.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return group, nil
}

func (d *DB) GetDueGroups(ctx context.Context) ([]*models.Group, error) {
	return getDueGroups(ctx, d.handle())
}
func (t *Tx) GetDueGroups(ctx context.Context) ([]*models.Group, error) {
	return getDueGroups(ctx, t.handle())
}

func getDueGroups(ctx context.Context, h dbHandle) ([]*models.Group, error) {
	query := `
		SELECT id, name, description, is_global, subscription_url, auto_update, update_interval,
		       last_updated, next_update, user_agent, created_at, updated_at
		FROM groups
		WHERE subscription_url IS NOT NULL
		  AND auto_update = 1
		  AND (next_update IS NULL OR next_update <= datetime('now'))
	`
	rows, err := h.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*models.Group
	for rows.Next() {
		group := &models.Group{}
		err := rows.Scan(
			&group.ID, &group.Name, &group.Description, &group.IsGlobal, &group.SubscriptionURL,
			&group.AutoUpdate, &group.UpdateInterval, &group.LastUpdated, &group.NextUpdate,
			&group.UserAgent, &group.CreatedAt, &group.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		groups = append(groups, group)
	}
	return groups, rows.Err()
}

// ─── Config operations ──────────────────────────────────────────────────────

func (d *DB) CreateConfig(ctx context.Context, config *models.Config) error {
	return createConfig(ctx, d.handle(), config)
}
func (t *Tx) CreateConfig(ctx context.Context, config *models.Config) error {
	return createConfig(ctx, t.handle(), config)
}

func createConfig(ctx context.Context, h dbHandle, config *models.Config) error {
	authConfig, err := json.Marshal(config.AuthConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal auth config: %w", err)
	}
	transportConfig, err := json.Marshal(config.TransportConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal transport config: %w", err)
	}
	tlsConfig, err := json.Marshal(config.TLSConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal TLS config: %w", err)
	}
	tags, err := json.Marshal(config.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		INSERT INTO configs (name, protocol, group_id, address, port, auth_config, network, transport_config,
		                     tls_enabled, tls_config, uri, from_subscription, enabled, tags, notes, use_count)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := h.ExecContext(ctx, query,
		config.Name, config.Protocol, config.GroupID, config.Address, config.Port,
		authConfig, config.Network, transportConfig, config.TLSEnabled, tlsConfig,
		config.URI, config.FromSubscription, config.Enabled, tags, config.Notes, config.UseCount,
	)
	if err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	config.ID = id
	return nil
}

func (d *DB) GetConfig(ctx context.Context, id int64) (*models.Config, error) {
	return getConfig(ctx, d.handle(), id)
}
func (t *Tx) GetConfig(ctx context.Context, id int64) (*models.Config, error) {
	return getConfig(ctx, t.handle(), id)
}

func getConfig(ctx context.Context, h dbHandle, id int64) (*models.Config, error) {
	query := `
		SELECT id, name, protocol, group_id, address, port, auth_config, network, transport_config,
		       tls_enabled, tls_config, uri, from_subscription, enabled, tags, notes, last_used, use_count,
		       created_at, updated_at
		FROM configs WHERE id = ?
	`
	config := &models.Config{}
	var authConfig, transportConfig, tlsConfig, tags []byte
	err := h.QueryRowContext(ctx, query, id).Scan(
		&config.ID, &config.Name, &config.Protocol, &config.GroupID, &config.Address, &config.Port,
		&authConfig, &config.Network, &transportConfig, &config.TLSEnabled, &tlsConfig,
		&config.URI, &config.FromSubscription, &config.Enabled, &tags, &config.Notes,
		&config.LastUsed, &config.UseCount, &config.CreatedAt, &config.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("config not found")
	}
	if err != nil {
		return nil, err
	}
	config.AuthConfig = authConfig
	config.TransportConfig = transportConfig
	config.TLSConfig = tlsConfig
	if err := json.Unmarshal(tags, &config.Tags); err != nil {
		config.Tags = []string{}
	}
	return config, nil
}

func (d *DB) GetConfigByName(ctx context.Context, name string) (*models.Config, error) {
	return getConfigByName(ctx, d.handle(), name)
}
func (t *Tx) GetConfigByName(ctx context.Context, name string) (*models.Config, error) {
	return getConfigByName(ctx, t.handle(), name)
}

func getConfigByName(ctx context.Context, h dbHandle, name string) (*models.Config, error) {
	query := `
		SELECT id, name, protocol, group_id, address, port, auth_config, network, transport_config,
		       tls_enabled, tls_config, uri, from_subscription, enabled, tags, notes, last_used, use_count,
		       created_at, updated_at
		FROM configs WHERE name = ?
	`
	config := &models.Config{}
	var authConfig, transportConfig, tlsConfig, tags []byte
	err := h.QueryRowContext(ctx, query, name).Scan(
		&config.ID, &config.Name, &config.Protocol, &config.GroupID, &config.Address, &config.Port,
		&authConfig, &config.Network, &transportConfig, &config.TLSEnabled, &tlsConfig,
		&config.URI, &config.FromSubscription, &config.Enabled, &tags, &config.Notes,
		&config.LastUsed, &config.UseCount, &config.CreatedAt, &config.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("config not found")
	}
	if err != nil {
		return nil, err
	}
	config.AuthConfig = authConfig
	config.TransportConfig = transportConfig
	config.TLSConfig = tlsConfig
	if err := json.Unmarshal(tags, &config.Tags); err != nil {
		config.Tags = []string{}
	}
	return config, nil
}

func (d *DB) GetAllConfigs(ctx context.Context, filter storage.ConfigFilter) ([]*models.Config, error) {
	return getAllConfigs(ctx, d.handle(), filter)
}
func (t *Tx) GetAllConfigs(ctx context.Context, filter storage.ConfigFilter) ([]*models.Config, error) {
	return getAllConfigs(ctx, t.handle(), filter)
}

func getAllConfigs(ctx context.Context, h dbHandle, filter storage.ConfigFilter) ([]*models.Config, error) {
	query := `
		SELECT id, name, protocol, group_id, address, port, auth_config, network, transport_config,
		       tls_enabled, tls_config, uri, from_subscription, enabled, tags, notes, last_used, use_count,
		       created_at, updated_at
		FROM configs WHERE 1=1
	`
	args := []interface{}{}

	if filter.GroupID != nil {
		query += " AND group_id = ?"
		args = append(args, *filter.GroupID)
	}
	if filter.Protocol != nil {
		query += " AND protocol = ?"
		args = append(args, *filter.Protocol)
	}
	if filter.Enabled != nil {
		query += " AND enabled = ?"
		args = append(args, *filter.Enabled)
	}
	if filter.FromSubscription != nil {
		query += " AND from_subscription = ?"
		args = append(args, *filter.FromSubscription)
	}
	if filter.SearchTerm != "" {
		query += " AND (name LIKE ? OR address LIKE ? OR notes LIKE ?)"
		searchPattern := "%" + filter.SearchTerm + "%"
		args = append(args, searchPattern, searchPattern, searchPattern)
	}
	query += " ORDER BY name ASC"

	rows, err := h.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*models.Config
	for rows.Next() {
		config := &models.Config{}
		var authConfig, transportConfig, tlsConfig, tags []byte
		err := rows.Scan(
			&config.ID, &config.Name, &config.Protocol, &config.GroupID, &config.Address, &config.Port,
			&authConfig, &config.Network, &transportConfig, &config.TLSEnabled, &tlsConfig,
			&config.URI, &config.FromSubscription, &config.Enabled, &tags, &config.Notes,
			&config.LastUsed, &config.UseCount, &config.CreatedAt, &config.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		config.AuthConfig = authConfig
		config.TransportConfig = transportConfig
		config.TLSConfig = tlsConfig
		if err := json.Unmarshal(tags, &config.Tags); err != nil {
			config.Tags = []string{}
		}

		// Filter by tags if specified
		if len(filter.Tags) > 0 {
			hasAllTags := true
			for _, filterTag := range filter.Tags {
				found := false
				for _, configTag := range config.Tags {
					if strings.EqualFold(configTag, filterTag) {
						found = true
						break
					}
				}
				if !found {
					hasAllTags = false
					break
				}
			}
			if !hasAllTags {
				continue
			}
		}

		configs = append(configs, config)
	}
	return configs, rows.Err()
}

func (d *DB) UpdateConfig(ctx context.Context, config *models.Config) error {
	return updateConfig(ctx, d.handle(), config)
}
func (t *Tx) UpdateConfig(ctx context.Context, config *models.Config) error {
	return updateConfig(ctx, t.handle(), config)
}

func updateConfig(ctx context.Context, h dbHandle, config *models.Config) error {
	authConfig, err := json.Marshal(config.AuthConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal auth config: %w", err)
	}
	transportConfig, err := json.Marshal(config.TransportConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal transport config: %w", err)
	}
	tlsConfig, err := json.Marshal(config.TLSConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal TLS config: %w", err)
	}
	tags, err := json.Marshal(config.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		UPDATE configs
		SET name = ?, protocol = ?, group_id = ?, address = ?, port = ?, auth_config = ?,
		    network = ?, transport_config = ?, tls_enabled = ?, tls_config = ?, uri = ?,
		    from_subscription = ?, enabled = ?, tags = ?, notes = ?, last_used = ?, use_count = ?
		WHERE id = ?
	`
	_, err = h.ExecContext(ctx, query,
		config.Name, config.Protocol, config.GroupID, config.Address, config.Port, authConfig,
		config.Network, transportConfig, config.TLSEnabled, tlsConfig, config.URI,
		config.FromSubscription, config.Enabled, tags, config.Notes, config.LastUsed, config.UseCount,
		config.ID,
	)
	return err
}

func (d *DB) DeleteConfig(ctx context.Context, id int64) error {
	return deleteConfig(ctx, d.handle(), id)
}
func (t *Tx) DeleteConfig(ctx context.Context, id int64) error {
	return deleteConfig(ctx, t.handle(), id)
}

func deleteConfig(ctx context.Context, h dbHandle, id int64) error {
	_, err := h.ExecContext(ctx, "DELETE FROM configs WHERE id = ?", id)
	return err
}

func (d *DB) DeleteConfigsByGroup(ctx context.Context, groupID int64, fromSubscriptionOnly bool) error {
	return deleteConfigsByGroup(ctx, d.handle(), groupID, fromSubscriptionOnly)
}
func (t *Tx) DeleteConfigsByGroup(ctx context.Context, groupID int64, fromSubscriptionOnly bool) error {
	return deleteConfigsByGroup(ctx, t.handle(), groupID, fromSubscriptionOnly)
}

func deleteConfigsByGroup(ctx context.Context, h dbHandle, groupID int64, fromSubscriptionOnly bool) error {
	query := "DELETE FROM configs WHERE group_id = ?"
	if fromSubscriptionOnly {
		query += " AND from_subscription = 1"
	}
	_, err := h.ExecContext(ctx, query, groupID)
	return err
}

// ─── Latency operations ─────────────────────────────────────────────────────

func (d *DB) RecordLatency(ctx context.Context, latency *models.LatencyTest) error {
	return recordLatency(ctx, d.handle(), latency)
}
func (t *Tx) RecordLatency(ctx context.Context, latency *models.LatencyTest) error {
	return recordLatency(ctx, t.handle(), latency)
}

func recordLatency(ctx context.Context, h dbHandle, latency *models.LatencyTest) error {
	query := `
		INSERT INTO latency_tests (config_id, latency_ms, success, error_message, test_strategy)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := h.ExecContext(ctx, query,
		latency.ConfigID, latency.LatencyMS, latency.Success, latency.ErrorMessage, latency.TestStrategy,
	)
	if err != nil {
		return fmt.Errorf("failed to record latency: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	latency.ID = id
	return nil
}

func (d *DB) GetLatestLatency(ctx context.Context, configID int64) (*models.LatencyTest, error) {
	return getLatestLatency(ctx, d.handle(), configID)
}
func (t *Tx) GetLatestLatency(ctx context.Context, configID int64) (*models.LatencyTest, error) {
	return getLatestLatency(ctx, t.handle(), configID)
}

func getLatestLatency(ctx context.Context, h dbHandle, configID int64) (*models.LatencyTest, error) {
	query := `
		SELECT id, config_id, latency_ms, success, error_message, test_strategy, tested_at
		FROM latency_tests
		WHERE config_id = ?
		ORDER BY tested_at DESC
		LIMIT 1
	`
	latency := &models.LatencyTest{}
	err := h.QueryRowContext(ctx, query, configID).Scan(
		&latency.ID, &latency.ConfigID, &latency.LatencyMS, &latency.Success,
		&latency.ErrorMessage, &latency.TestStrategy, &latency.TestedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return latency, nil
}

func (d *DB) GetLatencyHistory(ctx context.Context, configID int64, limit int) ([]*models.LatencyTest, error) {
	return getLatencyHistory(ctx, d.handle(), configID, limit)
}
func (t *Tx) GetLatencyHistory(ctx context.Context, configID int64, limit int) ([]*models.LatencyTest, error) {
	return getLatencyHistory(ctx, t.handle(), configID, limit)
}

func getLatencyHistory(ctx context.Context, h dbHandle, configID int64, limit int) ([]*models.LatencyTest, error) {
	query := `
		SELECT id, config_id, latency_ms, success, error_message, test_strategy, tested_at
		FROM latency_tests
		WHERE config_id = ?
		ORDER BY tested_at DESC
		LIMIT ?
	`
	rows, err := h.QueryContext(ctx, query, configID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var latencies []*models.LatencyTest
	for rows.Next() {
		latency := &models.LatencyTest{}
		err := rows.Scan(
			&latency.ID, &latency.ConfigID, &latency.LatencyMS, &latency.Success,
			&latency.ErrorMessage, &latency.TestStrategy, &latency.TestedAt,
		)
		if err != nil {
			return nil, err
		}
		latencies = append(latencies, latency)
	}
	return latencies, rows.Err()
}

// ─── Settings operations ────────────────────────────────────────────────────

func (d *DB) GetSetting(ctx context.Context, key string) (string, error) {
	return getSetting(ctx, d.handle(), key)
}
func (t *Tx) GetSetting(ctx context.Context, key string) (string, error) {
	return getSetting(ctx, t.handle(), key)
}

func getSetting(ctx context.Context, h dbHandle, key string) (string, error) {
	var value string
	err := h.QueryRowContext(ctx, "SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("setting not found: %s", key)
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func (d *DB) SetSetting(ctx context.Context, key, value string) error {
	return setSetting(ctx, d.handle(), key, value)
}
func (t *Tx) SetSetting(ctx context.Context, key, value string) error {
	return setSetting(ctx, t.handle(), key, value)
}

func setSetting(ctx context.Context, h dbHandle, key, value string) error {
	query := `
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value
	`
	_, err := h.ExecContext(ctx, query, key, value)
	return err
}

func (d *DB) GetAllSettings(ctx context.Context) (map[string]string, error) {
	return getAllSettings(ctx, d.handle())
}
func (t *Tx) GetAllSettings(ctx context.Context) (map[string]string, error) {
	return getAllSettings(ctx, t.handle())
}

func getAllSettings(ctx context.Context, h dbHandle) (map[string]string, error) {
	rows, err := h.QueryContext(ctx, "SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		settings[key] = value
	}
	return settings, rows.Err()
}

// ─── Active connection operations ───────────────────────────────────────────

func (d *DB) SetActiveConnection(ctx context.Context, conn *models.ActiveConnection) error {
	return setActiveConnection(ctx, d.handle(), conn)
}
func (t *Tx) SetActiveConnection(ctx context.Context, conn *models.ActiveConnection) error {
	return setActiveConnection(ctx, t.handle(), conn)
}

func setActiveConnection(ctx context.Context, h dbHandle, conn *models.ActiveConnection) error {
	query := `
		INSERT INTO active_connection (id, config_id, core_type, vpn_mode, started_at)
		VALUES (1, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			config_id = excluded.config_id,
			core_type = excluded.core_type,
			vpn_mode = excluded.vpn_mode,
			started_at = excluded.started_at
	`
	_, err := h.ExecContext(ctx, query, conn.ConfigID, conn.CoreType, conn.VPNMode)
	return err
}

func (d *DB) GetActiveConnection(ctx context.Context) (*models.ActiveConnection, error) {
	return getActiveConnection(ctx, d.handle())
}
func (t *Tx) GetActiveConnection(ctx context.Context) (*models.ActiveConnection, error) {
	return getActiveConnection(ctx, t.handle())
}

func getActiveConnection(ctx context.Context, h dbHandle) (*models.ActiveConnection, error) {
	query := `SELECT id, config_id, core_type, vpn_mode, started_at FROM active_connection WHERE id = 1`
	conn := &models.ActiveConnection{}
	err := h.QueryRowContext(ctx, query).Scan(
		&conn.ID, &conn.ConfigID, &conn.CoreType, &conn.VPNMode, &conn.StartedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (d *DB) ClearActiveConnection(ctx context.Context) error {
	return clearActiveConnection(ctx, d.handle())
}
func (t *Tx) ClearActiveConnection(ctx context.Context) error {
	return clearActiveConnection(ctx, t.handle())
}

func clearActiveConnection(ctx context.Context, h dbHandle) error {
	_, err := h.ExecContext(ctx, "DELETE FROM active_connection WHERE id = 1")
	return err
}
