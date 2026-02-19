package tun

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// tunState is persisted to disk so we can recover from crashes.
type tunState struct {
	Gateway     string   `json:"gateway"`
	Interface   string   `json:"interface"`
	RemoteAddrs []string `json:"remote_addrs"`
	DeviceName  string   `json:"device_name"`
}

// stateFilePath returns the path to the TUN state file.
func stateFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(homeDir, ".cache", "raycoon")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "tun.state"), nil
}

// saveState writes the current TUN state to disk for crash recovery.
func saveState(gateway, iface string, remoteAddrs []string, deviceName string) error {
	path, err := stateFilePath()
	if err != nil {
		return fmt.Errorf("failed to get state file path: %w", err)
	}

	state := tunState{
		Gateway:     gateway,
		Interface:   iface,
		RemoteAddrs: remoteAddrs,
		DeviceName:  deviceName,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}
	return nil
}

// removeState deletes the state file after a clean shutdown.
func removeState() {
	if path, err := stateFilePath(); err == nil {
		os.Remove(path)
	}
}

// CleanupIfNeeded checks for a stale TUN state file and restores routes if found.
// Should be called on application startup.
func CleanupIfNeeded() {
	path, err := stateFilePath()
	if err != nil {
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return // No state file — nothing to clean up.
	}

	var state tunState
	if err := json.Unmarshal(data, &state); err != nil {
		os.Remove(path)
		return
	}

	// Restore routes from the stale state.
	if state.Gateway != "" {
		restoreDefaultRoute(state.Gateway)
	}
	if len(state.RemoteAddrs) > 0 {
		removeBypassRoutes(state.RemoteAddrs)
	}
	restoreDNS()

	os.Remove(path)
}
