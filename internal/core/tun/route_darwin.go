package tun

import (
	"fmt"
	"os/exec"
	"strings"
)

// detectGateway detects the current default gateway and interface on macOS.
func detectGateway() (gateway, iface string, err error) {
	out, err := exec.Command("route", "-n", "get", "default").Output()
	if err != nil {
		return "", "", fmt.Errorf("failed to detect default gateway: %w", err)
	}

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "gateway:") {
			gateway = strings.TrimSpace(strings.TrimPrefix(line, "gateway:"))
		}
		if strings.HasPrefix(line, "interface:") {
			iface = strings.TrimSpace(strings.TrimPrefix(line, "interface:"))
		}
	}

	if gateway == "" || iface == "" {
		return "", "", fmt.Errorf("could not parse default gateway (gateway=%q, interface=%q)", gateway, iface)
	}
	return gateway, iface, nil
}

// addBypassRoutes adds host routes for remote server IPs via the original gateway
// so traffic to the proxy server itself doesn't loop through the TUN device.
func addBypassRoutes(remoteAddrs []string, gateway string) error {
	for _, addr := range remoteAddrs {
		if err := run("route", "add", "-host", addr, gateway); err != nil {
			return fmt.Errorf("failed to add bypass route for %s: %w", addr, err)
		}
	}
	return nil
}

// removeBypassRoutes removes the host routes added by addBypassRoutes.
func removeBypassRoutes(remoteAddrs []string) error {
	var firstErr error
	for _, addr := range remoteAddrs {
		if err := run("route", "delete", "-host", addr); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to remove bypass route for %s: %w", addr, err)
		}
	}
	return firstErr
}

// setDefaultRouteTUN replaces the default route to go through the TUN device.
func setDefaultRouteTUN(tunDevice string) error {
	// Get the TUN device's point-to-point address to use as gateway.
	// tun2socks assigns 198.18.0.1 as the device address by default.
	tunGateway := "198.18.0.1"

	if err := run("route", "delete", "default"); err != nil {
		return fmt.Errorf("failed to delete current default route: %w", err)
	}
	if err := run("route", "add", "default", tunGateway); err != nil {
		return fmt.Errorf("failed to add TUN default route: %w", err)
	}
	return nil
}

// restoreDefaultRoute restores the original default route via the original gateway.
func restoreDefaultRoute(gateway string) error {
	// Delete TUN route (ignore error — may already be gone).
	run("route", "delete", "default")
	if err := run("route", "add", "default", gateway); err != nil {
		return fmt.Errorf("failed to restore default route via %s: %w", gateway, err)
	}
	return nil
}

// configureDNS sets DNS servers on all active network services.
func configureDNS(dnsServer string) error {
	services, err := activeNetworkServices()
	if err != nil {
		return err
	}
	for _, svc := range services {
		if err := run("networksetup", "-setdnsservers", svc, dnsServer); err != nil {
			return fmt.Errorf("failed to set DNS on %s: %w", svc, err)
		}
	}
	return nil
}

// restoreDNS resets DNS servers to automatic (empty) on all active network services.
func restoreDNS() error {
	services, err := activeNetworkServices()
	if err != nil {
		return err
	}
	var firstErr error
	for _, svc := range services {
		if err := run("networksetup", "-setdnsservers", svc, "empty"); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("failed to restore DNS on %s: %w", svc, err)
		}
	}
	return firstErr
}

// activeNetworkServices returns all non-disabled network services.
func activeNetworkServices() ([]string, error) {
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return nil, err
	}

	var services []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "An asterisk") || strings.HasPrefix(line, "*") {
			continue
		}
		services = append(services, line)
	}

	if len(services) == 0 {
		return nil, fmt.Errorf("no active network services found")
	}
	return services, nil
}

func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}
