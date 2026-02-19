package tun

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"

	"github.com/xjasonlyu/tun2socks/v2/engine"
)

// ErrNotRoot is returned when TUN mode is used without root/admin privileges.
var ErrNotRoot = errors.New(
	"TUN mode requires elevated privileges.\n" +
		"  macOS/Linux: sudo raycoon connect <config> -m tunnel\n" +
		"  Windows:     run as Administrator",
)

// tunDNS is the DNS server address used by the TUN device.
// tun2socks uses 198.18.0.0/15 for its virtual network; 198.18.0.2 is a common
// DNS intercept address routed through the TUN.
const tunDNS = "198.18.0.2"

// Enable creates a TUN device, starts tun2socks, and configures system routes
// so all traffic flows through the TUN to the given SOCKS5 proxy.
//
// remoteAddrs are the proxy server IPs that must bypass the TUN to avoid loops.
func Enable(socksPort int, remoteAddrs []string) error {
	if err := checkPrivileges(); err != nil {
		return err
	}

	// Detect current gateway before we change anything.
	gateway, iface, err := detectGateway()
	if err != nil {
		return fmt.Errorf("failed to detect gateway: %w", err)
	}

	// Determine TUN device name.
	deviceName := "tun0"
	if runtime.GOOS == "darwin" {
		deviceName = "utun69"
	}

	// Save state for crash recovery (before making changes).
	if err := saveState(gateway, iface, remoteAddrs, deviceName); err != nil {
		return fmt.Errorf("failed to save TUN state: %w", err)
	}

	// Configure and start tun2socks engine.
	key := &engine.Key{
		Proxy:  fmt.Sprintf("socks5://127.0.0.1:%d", socksPort),
		Device: fmt.Sprintf("tun://%s", deviceName),
		MTU:    DefaultMTU,
	}
	engine.Insert(key)
	engine.Start()

	// Add bypass routes for remote server IPs via original gateway.
	if err := addBypassRoutes(remoteAddrs, gateway); err != nil {
		engine.Stop()
		removeState()
		return fmt.Errorf("failed to add bypass routes: %w", err)
	}

	// Replace default route to go through TUN device.
	if err := setDefaultRouteTUN(deviceName); err != nil {
		removeBypassRoutes(remoteAddrs)
		engine.Stop()
		removeState()
		return fmt.Errorf("failed to set TUN default route: %w", err)
	}

	// Configure DNS to use the TUN DNS address.
	if err := configureDNS(tunDNS); err != nil {
		restoreDefaultRoute(gateway)
		removeBypassRoutes(remoteAddrs)
		engine.Stop()
		removeState()
		return fmt.Errorf("failed to configure DNS: %w", err)
	}

	return nil
}

// Disable tears down the TUN device and restores original system routes.
func Disable() error {
	// Read saved state to know what to restore.
	path, err := stateFilePath()
	if err != nil {
		return fmt.Errorf("failed to get state file path: %w", err)
	}

	state, err := readStateFile(path)
	if err != nil {
		// No state file — try to stop engine anyway.
		engine.Stop()
		return nil
	}

	// Restore DNS first (before route changes).
	restoreDNS()

	// Restore original default route.
	if state.Gateway != "" {
		restoreDefaultRoute(state.Gateway)
	}

	// Remove bypass routes.
	if len(state.RemoteAddrs) > 0 {
		removeBypassRoutes(state.RemoteAddrs)
	}

	// Stop tun2socks engine (destroys TUN device).
	engine.Stop()

	// Clean up state file.
	removeState()

	return nil
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
