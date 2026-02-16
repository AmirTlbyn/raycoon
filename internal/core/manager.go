package core

import (
	"context"
	"fmt"
	"sync"

	"raycoon/internal/core/types"
	"raycoon/internal/core/xray"
	"raycoon/internal/storage/models"
)

// Manager manages proxy cores and active connections
type Manager struct {
	activeCore ProxyCore
	coreType   types.CoreType
	mu         sync.RWMutex
}

// NewManager creates a new core manager
func NewManager(coreType types.CoreType) (*Manager, error) {
	var core ProxyCore
	var err error

	switch coreType {
	case types.CoreTypeXray:
		core, err = xray.New()
		if err != nil {
			return nil, fmt.Errorf("failed to create xray core: %w", err)
		}
	case types.CoreTypeSingbox:
		return nil, fmt.Errorf("singbox core not yet implemented")
	default:
		return nil, fmt.Errorf("unsupported core type: %s", coreType)
	}

	return &Manager{
		activeCore: core,
		coreType:   coreType,
	}, nil
}

// Start starts the proxy core with the given configuration
func (m *Manager) Start(ctx context.Context, config *types.CoreConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeCore.IsRunning() {
		return fmt.Errorf("core is already running")
	}

	return m.activeCore.Start(ctx, config)
}

// Stop stops the running proxy core
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.activeCore.IsRunning() {
		return fmt.Errorf("core is not running")
	}

	return m.activeCore.Stop(ctx)
}

// Restart restarts the proxy core with new configuration
func (m *Manager) Restart(ctx context.Context, config *types.CoreConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeCore.IsRunning() {
		if err := m.activeCore.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop core: %w", err)
		}
	}

	return m.activeCore.Start(ctx, config)
}

// IsRunning returns whether the core is currently running
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.activeCore.IsRunning()
}

// GetStatus returns the current status of the core
func (m *Manager) GetStatus() (*types.Status, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status, err := m.activeCore.GetStatus()
	if err != nil {
		return nil, err
	}

	status.CoreType = string(m.coreType)
	return status, nil
}

// GetStats returns real-time statistics
func (m *Manager) GetStats() (*types.Stats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.activeCore.GetStats()
}

// GetVersion returns the core version
func (m *Manager) GetVersion() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.activeCore.GetVersion()
}

// SwitchCore switches to a different core type
func (m *Manager) SwitchCore(ctx context.Context, coreType types.CoreType, config *types.CoreConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Stop current core if running
	if m.activeCore.IsRunning() {
		if err := m.activeCore.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop current core: %w", err)
		}
	}

	// Create new core
	var newCore ProxyCore
	var err error

	switch coreType {
	case types.CoreTypeXray:
		newCore, err = xray.New()
	case types.CoreTypeSingbox:
		return fmt.Errorf("singbox core not yet implemented")
	default:
		return fmt.Errorf("unsupported core type: %s", coreType)
	}

	if err != nil {
		return fmt.Errorf("failed to create new core: %w", err)
	}

	m.activeCore = newCore
	m.coreType = coreType

	// Start new core if config provided
	if config != nil {
		if err := newCore.Start(ctx, config); err != nil {
			return fmt.Errorf("failed to start new core: %w", err)
		}
	}

	return nil
}

// GetCoreType returns the current core type
func (m *Manager) GetCoreType() types.CoreType {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.coreType
}

// BuildCoreConfig builds a CoreConfig from settings and proxy config
func BuildCoreConfig(proxyConfig *models.Config, vpnMode types.VPNMode, socksPort, httpPort int) *types.CoreConfig {
	return &types.CoreConfig{
		Config:    proxyConfig,
		VPNMode:   vpnMode,
		SOCKSPort: socksPort,
		HTTPPort:  httpPort,
		LogLevel:  "none",
		DNSServers: []string{
			"8.8.8.8",
			"1.1.1.1",
		},
	}
}
