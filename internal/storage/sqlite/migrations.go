package sqlite

const schema = `
-- Groups table (combines subscription groups and config organization)
CREATE TABLE IF NOT EXISTS groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    is_global BOOLEAN DEFAULT 0,
    subscription_url TEXT,
    auto_update BOOLEAN DEFAULT 1,
    update_interval INTEGER DEFAULT 86400,
    last_updated TIMESTAMP,
    next_update TIMESTAMP,
    user_agent TEXT DEFAULT 'Raycoon/1.0',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Proxy configs table
CREATE TABLE IF NOT EXISTS configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    protocol TEXT NOT NULL,
    group_id INTEGER NOT NULL,

    -- Connection details
    address TEXT NOT NULL,
    port INTEGER NOT NULL,

    -- Auth details (JSON for flexibility)
    auth_config TEXT NOT NULL,

    -- Transport details
    network TEXT DEFAULT 'tcp',
    transport_config TEXT,

    -- TLS details
    tls_enabled BOOLEAN DEFAULT 0,
    tls_config TEXT,

    -- Original URI
    uri TEXT,

    -- Source tracking
    from_subscription BOOLEAN DEFAULT 0,

    -- Metadata
    enabled BOOLEAN DEFAULT 1,
    tags TEXT,
    notes TEXT,

    -- Stats
    last_used TIMESTAMP,
    use_count INTEGER DEFAULT 0,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (group_id) REFERENCES groups(id) ON DELETE CASCADE
);

-- Latency test results
CREATE TABLE IF NOT EXISTS latency_tests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    config_id INTEGER NOT NULL,
    latency_ms INTEGER,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    test_strategy TEXT DEFAULT 'tcp',
    tested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (config_id) REFERENCES configs(id) ON DELETE CASCADE
);

-- Application settings
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Active connection tracking
CREATE TABLE IF NOT EXISTS active_connection (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    config_id INTEGER NOT NULL,
    core_type TEXT NOT NULL,
    vpn_mode TEXT NOT NULL,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (config_id) REFERENCES configs(id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_configs_group_id ON configs(group_id);
CREATE INDEX IF NOT EXISTS idx_configs_protocol ON configs(protocol);
CREATE INDEX IF NOT EXISTS idx_configs_enabled ON configs(enabled);
CREATE INDEX IF NOT EXISTS idx_configs_from_subscription ON configs(from_subscription);
CREATE INDEX IF NOT EXISTS idx_groups_next_update ON groups(next_update);
CREATE INDEX IF NOT EXISTS idx_latency_tests_config_id ON latency_tests(config_id);
CREATE INDEX IF NOT EXISTS idx_latency_tests_tested_at ON latency_tests(tested_at);

-- Triggers for updated_at
CREATE TRIGGER IF NOT EXISTS update_groups_timestamp AFTER UPDATE ON groups
BEGIN
    UPDATE groups SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_configs_timestamp AFTER UPDATE ON configs
BEGIN
    UPDATE configs SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_settings_timestamp AFTER UPDATE ON settings
BEGIN
    UPDATE settings SET updated_at = CURRENT_TIMESTAMP WHERE key = NEW.key;
END;
`

const defaultData = `
-- Create default global group
INSERT OR IGNORE INTO groups (id, name, description, is_global)
VALUES (1, 'global', 'Default global group', 1);

-- Insert default settings
INSERT OR IGNORE INTO settings (key, value) VALUES
    ('active_core', 'xray'),
    ('vpn_mode', 'proxy'),
    ('proxy_port', '1080'),
    ('http_proxy_port', '1081'),
    ('latency_test_timeout', '5000'),
    ('latency_test_workers', '10'),
    ('subscription_user_agent', 'Raycoon/1.0'),
    ('log_level', 'info');
`

// RunMigrations executes the database schema and default data
func runMigrations(db *DB) error {
	// Execute schema
	if _, err := db.db.Exec(schema); err != nil {
		return err
	}

	// Insert default data
	if _, err := db.db.Exec(defaultData); err != nil {
		return err
	}

	return nil
}
