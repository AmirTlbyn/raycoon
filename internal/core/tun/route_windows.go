package tun

import (
	"fmt"
	"os/exec"
	"strings"
)

// detectGateway detects the current default gateway and interface on Windows.
func detectGateway() (gateway, iface string, err error) {
	// Use PowerShell to get the default route reliably.
	out, err := exec.Command("powershell", "-Command",
		"(Get-NetRoute -DestinationPrefix '0.0.0.0/0' | Sort-Object RouteMetric | Select-Object -First 1 | Format-List NextHop,InterfaceAlias)").Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to detect default gateway: %w", err)
	}

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "NextHop") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				gateway = strings.TrimSpace(parts[1])
			}
		}
		if strings.HasPrefix(line, "InterfaceAlias") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				iface = strings.TrimSpace(parts[1])
			}
		}
	}

	if gateway == "" || iface == "" {
		return "", "", fmt.Errorf("could not parse default gateway from: %s", string(out))
	}
	return gateway, iface, nil
}

// addBypassRoutes adds host routes for remote server IPs via the original gateway.
func addBypassRoutes(remoteAddrs []string, gateway string) error {
	for _, addr := range remoteAddrs {
		if err := run("route", "add", addr, "mask", "255.255.255.255", gateway); err != nil {
			return fmt.Errorf("failed to add bypass route for %s: %w", addr, err)
		}
	}
	return nil
}

// removeBypassRoutes removes the bypass host routes.
func removeBypassRoutes(remoteAddrs []string) error {
	var firstErr error
	for _, addr := range remoteAddrs {
		if err := run("route", "delete", addr); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to remove bypass route for %s: %w", addr, err)
		}
	}
	return firstErr
}

// setDefaultRouteTUN sets the default route through the TUN device.
func setDefaultRouteTUN(tunDevice string) error {
	tunGateway := "198.18.0.1"
	if err := run("route", "add", "0.0.0.0", "mask", "0.0.0.0", tunGateway, "metric", "1"); err != nil {
		return fmt.Errorf("failed to set TUN default route: %w", err)
	}
	return nil
}

// restoreDefaultRoute restores the original default route.
func restoreDefaultRoute(gateway string) error {
	// Remove the TUN route (ignore errors).
	run("route", "delete", "0.0.0.0", "mask", "0.0.0.0", "198.18.0.1")
	if err := run("route", "add", "0.0.0.0", "mask", "0.0.0.0", gateway); err != nil {
		return fmt.Errorf("failed to restore default route via %s: %w", gateway, err)
	}
	return nil
}

// configureDNS sets DNS via netsh on the detected interface.
func configureDNS(dnsServer string) error {
	// We need the interface name — use a known default. The caller sets it via state.
	if err := run("netsh", "interface", "ip", "set", "dns", "name=Local Area Connection", "static", dnsServer); err != nil {
		// Fallback: try setting via PowerShell.
		return run("powershell", "-Command",
			fmt.Sprintf("Set-DnsClientServerAddress -InterfaceAlias '*' -ServerAddresses %s", dnsServer))
	}
	return nil
}

// restoreDNS resets DNS to automatic.
func restoreDNS() error {
	return run("powershell", "-Command",
		"Set-DnsClientServerAddress -InterfaceAlias '*' -ResetServerAddresses")
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
