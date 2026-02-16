package errors

import (
	"errors"
	"fmt"
)

// Common error types
var (
	// Core errors
	ErrCoreNotRunning     = errors.New("core is not running")
	ErrCoreAlreadyRunning = errors.New("core is already running")
	ErrCoreNotFound       = errors.New("core binary not found")
	ErrCoreStartFailed    = errors.New("failed to start core")
	ErrCoreStopFailed     = errors.New("failed to stop core")

	// Config errors
	ErrConfigNotFound     = errors.New("config not found")
	ErrConfigInvalid      = errors.New("invalid config")
	ErrConfigDisabled     = errors.New("config is disabled")
	ErrProtocolUnsupported = errors.New("protocol not supported")
	ErrURIInvalid         = errors.New("invalid URI")

	// Group errors
	ErrGroupNotFound   = errors.New("group not found")
	ErrGroupIsGlobal   = errors.New("cannot modify global group")
	ErrGroupHasConfigs = errors.New("group has configs")

	// Subscription errors
	ErrSubscriptionNotFound  = errors.New("subscription not found")
	ErrSubscriptionFetchFailed = errors.New("failed to fetch subscription")
	ErrSubscriptionDecodeFailed = errors.New("failed to decode subscription")
	ErrSubscriptionEmpty      = errors.New("subscription is empty")

	// Connection errors
	ErrNoActiveConnection    = errors.New("no active connection")
	ErrAlreadyConnected      = errors.New("already connected")
	ErrConnectionFailed      = errors.New("connection failed")
	ErrConnectionTimeout     = errors.New("connection timeout")

	// Latency errors
	ErrLatencyTestFailed   = errors.New("latency test failed")
	ErrLatencyTestTimeout  = errors.New("latency test timeout")
	ErrNoLatencyData       = errors.New("no latency data available")
)

// ConfigError represents a config-related error
type ConfigError struct {
	ConfigID int64
	Name     string
	Err      error
}

func (e *ConfigError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("config '%s' (ID: %d): %v", e.Name, e.ConfigID, e.Err)
	}
	return fmt.Sprintf("config (ID: %d): %v", e.ConfigID, e.Err)
}

func (e *ConfigError) Unwrap() error {
	return e.Err
}

// SubscriptionError represents a subscription-related error
type SubscriptionError struct {
	URL  string
	Name string
	Err  error
}

func (e *SubscriptionError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("subscription '%s': %v", e.Name, e.Err)
	}
	return fmt.Sprintf("subscription '%s': %v", e.URL, e.Err)
}

func (e *SubscriptionError) Unwrap() error {
	return e.Err
}

// CoreError represents a core-related error
type CoreError struct {
	CoreType string
	Err      error
}

func (e *CoreError) Error() string {
	return fmt.Sprintf("%s core: %v", e.CoreType, e.Err)
}

func (e *CoreError) Unwrap() error {
	return e.Err
}

// NetworkError represents a network-related error
type NetworkError struct {
	Address string
	Port    int
	Err     error
}

func (e *NetworkError) Error() string {
	return fmt.Sprintf("network error (%s:%d): %v", e.Address, e.Port, e.Err)
}

func (e *NetworkError) Unwrap() error {
	return e.Err
}
