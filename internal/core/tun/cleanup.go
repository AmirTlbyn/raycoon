package tun

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"raycoon/internal/paths"
)

// tunState is persisted to disk so we can recover from crashes.
type tunState struct {
	Gateway     string   `json:"gateway"`
	Interface   string   `json:"interface"`
	RemoteAddrs []string `json:"remote_addrs"`
	DeviceName  string   `json:"device_name"`
}

// stateFilePath returns the path to the TUN state file.
// Uses paths.CacheDir() so the directory is chowned to the real user when
// running under sudo, allowing non-root disconnect to read the state file.
func stateFilePath() (string, error) {
	dir, err := paths.CacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get cache dir: %w", err)
	}
	return dir + "/tun.state", nil
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

// readStateFile reads and parses the TUN state file.
func readStateFile(path string) (*tunState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state tunState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

// CleanupIfNeeded checks for a stale TUN state file and restores the system
// if found. Called on application startup and before Enable().
//
// Strategy:
//  1. Signal the daemon via the stop file and wait up to 3 s for it to exit.
//  2. If the daemon does not respond (it may have crashed), restore routes and
//     DNS directly. This requires root; without it the commands will silently
//     fail but the next sudo invocation will succeed.
func CleanupIfNeeded() {
	statePath, err := stateFilePath()
	if err != nil {
		return
	}

	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return // No stale state — nothing to do.
	}

	// Try to signal a live daemon first.
	stopPath, stopErr := stopFilePath()
	if stopErr == nil {
		_ = os.WriteFile(stopPath, []byte("stop"), 0644)

		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := os.Stat(statePath); os.IsNotExist(err) {
				return // Daemon cleaned up successfully.
			}
			time.Sleep(300 * time.Millisecond)
		}
		os.Remove(stopPath) // Tidy up the stop file if daemon didn't see it.
	}

	// Daemon is gone (crash/restart) — clean up directly.
	data, err := os.ReadFile(statePath)
	if err != nil {
		os.Remove(statePath)
		return
	}

	var state tunState
	if err := json.Unmarshal(data, &state); err != nil {
		os.Remove(statePath)
		return
	}

	if state.Gateway != "" {
		restoreDefaultRoute(state.Gateway)
	}
	if len(state.RemoteAddrs) > 0 {
		removeBypassRoutes(state.RemoteAddrs)
	}
	restoreDNS()
	os.Remove(statePath)
}
