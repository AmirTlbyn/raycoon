package tun

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/xjasonlyu/tun2socks/v2/engine"
	"raycoon/internal/paths"
)

// stopFilePath returns the path of the stop-signal file that Disable() creates
// to tell the daemon to exit.
func stopFilePath() (string, error) {
	dir, err := paths.CacheDir()
	if err != nil {
		return "", err
	}
	return dir + "/tun.stop", nil
}

// RunDaemon is the entry point for the long-running TUN daemon subprocess.
// It creates the TUN device, configures routes and DNS, then blocks until
// the stop file appears or SIGTERM/SIGINT is received, then cleans up.
//
// This function is called from the hidden "tund" cobra command which is
// spawned as a detached child process by Enable().
func RunDaemon(socksPort int, remoteAddrs []string) error {
	// Open log file first so every error below is visible for debugging.
	var lf *os.File
	if cacheDir, err := paths.CacheDir(); err == nil {
		if f, err := os.OpenFile(cacheDir+"/tund.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err == nil {
			lf = f
			defer lf.Close()
			paths.ChownToRealUser(lf.Name())
		}
	}
	logf := func(format string, args ...interface{}) {
		if lf != nil {
			fmt.Fprintf(lf, "[%s] "+format+"\n", append([]interface{}{time.Now().Format("15:04:05.000")}, args...)...)
		}
	}
	logf("tund starting — socksPort=%d bypasses=%v", socksPort, remoteAddrs)

	if err := checkPrivileges(); err != nil {
		logf("checkPrivileges: %v", err)
		return err
	}

	// Detect current gateway before we change anything.
	gateway, iface, err := detectGateway()
	if err != nil {
		logf("detectGateway: %v", err)
		return fmt.Errorf("failed to detect gateway: %w", err)
	}
	logf("gateway=%s iface=%s", gateway, iface)

	// Snapshot TUN interfaces before creating a new one.
	beforeIfaces := listTUNInterfaces()

	// Start tun2socks engine — this creates the TUN device.
	deviceHint := "tun0"
	if runtime.GOOS == "darwin" {
		deviceHint = "utun99"
	}
	key := &engine.Key{
		Proxy:    fmt.Sprintf("socks5://127.0.0.1:%d", socksPort),
		Device:   fmt.Sprintf("tun://%s", deviceHint),
		MTU:      DefaultMTU,
		LogLevel: "silent",
		// Do NOT set Interface — see note in tun.go Enable().
	}
	engine.Insert(key)
	engine.Start()
	logf("engine started")

	// Discover the actual TUN interface name — retry for up to 3 s in case
	// the kernel hasn't made the interface visible yet.
	var deviceName string
	ifaceDeadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(ifaceDeadline) {
		deviceName = findNewTUNInterface(beforeIfaces)
		if deviceName != "" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if deviceName == "" {
		logf("failed to detect TUN device after engine.Start()")
		engine.Stop()
		return fmt.Errorf("failed to detect TUN device after creation")
	}
	logf("TUN device: %s", deviceName)

	// Assign IP to the TUN so routes can use it as a gateway.
	if err := configureTUNAddress(deviceName, tunGateway); err != nil {
		logf("configureTUNAddress: %v", err)
		engine.Stop()
		return fmt.Errorf("failed to configure TUN address: %w", err)
	}
	logf("TUN address configured (%s)", tunGateway)

	// Build full bypass list: proxy server addresses + DNS servers.
	// DNS servers are bypassed (go direct) to avoid UDP-over-SOCKS5 issues.
	// Xray TLS/HTTP sniffing overrides poisoned IPs to real domain names,
	// so blocked domains still get proxied correctly despite direct DNS.
	allBypasses := make([]string, 0, len(remoteAddrs)+2)
	allBypasses = append(allBypasses, remoteAddrs...)
	allBypasses = append(allBypasses, tunDNS, "1.1.1.1")

	// Add bypass routes so bypassed traffic doesn't loop through TUN.
	if err := addBypassRoutes(allBypasses, gateway); err != nil {
		logf("addBypassRoutes: %v", err)
		engine.Stop()
		return fmt.Errorf("failed to add bypass routes: %w", err)
	}
	logf("bypass routes added: %v", allBypasses)

	// Route all internet traffic through TUN via two /1 overlay routes.
	if err := setDefaultRouteTUN(deviceName); err != nil {
		logf("setDefaultRouteTUN: %v", err)
		removeBypassRoutes(allBypasses)
		engine.Stop()
		return fmt.Errorf("failed to set TUN default route: %w", err)
	}
	logf("default route set via TUN")

	// Point system DNS to 8.8.8.8 (which bypasses TUN via the bypass route
	// added above — avoids UDP-over-SOCKS5 reliability issues).
	if err := configureDNS(tunDNS); err != nil {
		logf("configureDNS: %v", err)
		restoreDefaultRoute(gateway)
		removeBypassRoutes(allBypasses)
		engine.Stop()
		return fmt.Errorf("failed to configure DNS: %w", err)
	}
	logf("DNS configured (%s → direct)", tunDNS)

	// Write state file LAST — Enable() polls for this file to know the TUN
	// is fully configured and ready. Writing it before routes/DNS are set
	// would cause the connect command to return before the TUN is usable.
	// Save allBypasses (not just remoteAddrs) so cleanup removes DNS routes.
	if err := saveState(gateway, iface, allBypasses, deviceName); err != nil {
		logf("saveState: %v", err)
		restoreDNS()
		restoreDefaultRoute(gateway)
		removeBypassRoutes(allBypasses)
		engine.Stop()
		return fmt.Errorf("failed to save TUN state: %w", err)
	}
	logf("state saved — TUN ready")

	// Block until Disable() signals us (via stop file) or we get a signal.
	stopPath, _ := stopFilePath()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			goto cleanup
		case <-ticker.C:
			if _, err := os.Stat(stopPath); err == nil {
				os.Remove(stopPath)
				goto cleanup
			}
		}
	}

cleanup:
	logf("shutting down — restoring system state")
	// Restore everything in reverse order.
	restoreDNS()
	restoreDefaultRoute(gateway)
	removeBypassRoutes(allBypasses)
	engine.Stop()
	removeState()
	logf("tund exited cleanly")
	return nil
}
