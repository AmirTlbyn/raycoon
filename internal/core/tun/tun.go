package tun

import (
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sort"
	"strings"
	"syscall"
	"time"

	"raycoon/internal/paths"
)

// ErrNotRoot is returned when TUN mode is used without root/admin privileges.
var ErrNotRoot = errors.New(
	"TUN mode requires elevated privileges.\n" +
		"  macOS/Linux: sudo raycoon connect <config> -m tun\n" +
		"  Note: disconnect does not require sudo.",
)

// tunGateway is the point-to-point address assigned to the TUN device.
const tunGateway = "198.18.0.1"

// tunDNS is the DNS server set on the system while TUN is active.
// Routing this through the proxy defeats ISP DNS injection for blocked domains.
const tunDNS = "8.8.8.8"

// Enable spawns the TUN daemon subprocess and waits for it to signal readiness.
//
// The daemon is a separate long-lived process (a hidden "tund" subcommand of
// the raycoon binary) that keeps the TUN device and tun2socks engine alive
// after the CLI exits. Enable() returns once the daemon has written its state
// file (≈ 2 s), or errors after a 15-second timeout.
func Enable(socksPort int, remoteAddrs []string) error {
	if err := checkPrivileges(); err != nil {
		return err
	}

	// Clean up any stale state / stop file from a previous session.
	CleanupIfNeeded()
	if stopPath, err := stopFilePath(); err == nil {
		os.Remove(stopPath)
	}

	// Locate this binary — we'll re-exec ourselves as the daemon.
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to locate raycoon binary: %w", err)
	}

	// Build daemon command line.
	args := []string{"tund", fmt.Sprintf("--socks-port=%d", socksPort)}
	for _, addr := range remoteAddrs {
		args = append(args, "--bypass="+addr)
	}

	// Redirect daemon output to a log file for post-mortem debugging.
	logPath := ""
	if cacheDir, err2 := paths.CacheDir(); err2 == nil {
		logPath = cacheDir + "/tund.log"
	}

	var logFile *os.File
	if logPath != "" {
		logFile, _ = os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	}

	cmd := exec.Command(self, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if logFile != nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start TUN daemon: %w", err)
	}

	if logFile != nil {
		// The file descriptor is inherited by the child; we can close our copy.
		logFile.Close()
	}

	// Poll for the state file — the daemon writes it once the TUN is ready.
	statePath, err := stateFilePath()
	if err != nil {
		cmd.Process.Kill()
		return fmt.Errorf("failed to get state file path: %w", err)
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(statePath); err == nil {
			return nil // Daemon is ready.
		}
		// Check if child exited prematurely (error case).
		if cmd.ProcessState != nil {
			return fmt.Errorf("TUN daemon exited prematurely — check %s", logPath)
		}
		time.Sleep(300 * time.Millisecond)
	}

	cmd.Process.Kill()
	return fmt.Errorf("TUN daemon did not become ready within 15 s — check %s", logPath)
}

// Disable signals the TUN daemon to shut down by creating a stop file, then
// waits for it to finish cleaning up (routes, DNS, engine). Falls back to
// direct cleanup if the daemon does not respond within 10 s.
func Disable() error {
	stopPath, err := stopFilePath()
	if err != nil {
		return fmt.Errorf("failed to get stop file path: %w", err)
	}

	// Signal the daemon.
	if err := os.WriteFile(stopPath, []byte("stop"), 0644); err != nil {
		// If we can't write the stop file (e.g. permission issue after a crash),
		// fall through to direct cleanup.
		_ = err
	}

	// Wait for daemon to finish and remove the state file.
	statePath, _ := stateFilePath()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(statePath); os.IsNotExist(err) {
			return nil // Daemon cleaned up.
		}
		time.Sleep(300 * time.Millisecond)
	}

	// Daemon didn't respond (crash / missing) — clean up directly.
	os.Remove(stopPath)
	if state, err := readStateFile(statePath); err == nil {
		restoreDNS()
		if state.Gateway != "" {
			restoreDefaultRoute(state.Gateway)
		}
		if len(state.RemoteAddrs) > 0 {
			removeBypassRoutes(state.RemoteAddrs)
		}
		removeState()
	}

	return nil
}

// listTUNInterfaces returns sorted names of all tun/utun interfaces.
func listTUNInterfaces() []string {
	ifaces, _ := net.Interfaces()
	var names []string
	for _, i := range ifaces {
		if strings.HasPrefix(i.Name, "tun") || strings.HasPrefix(i.Name, "utun") {
			names = append(names, i.Name)
		}
	}
	sort.Strings(names)
	return names
}

// findNewTUNInterface returns the name of a TUN interface that appeared
// after engine.Start() by comparing with a snapshot taken before.
func findNewTUNInterface(before []string) string {
	beforeSet := make(map[string]bool, len(before))
	for _, n := range before {
		beforeSet[n] = true
	}
	after := listTUNInterfaces()
	for _, n := range after {
		if !beforeSet[n] {
			return n
		}
	}
	return ""
}
