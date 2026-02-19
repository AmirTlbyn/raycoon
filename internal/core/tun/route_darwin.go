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

// configureTUNAddress assigns a point-to-point IP address to the TUN device.
// This is required on macOS because tun2socks creates the device but doesn't
// configure an IP on it — without this, routes pointing to the TUN have no next hop.
func configureTUNAddress(deviceName, addr string) error {
	if err := run("ifconfig", deviceName, addr, addr, "up"); err != nil {
		return fmt.Errorf("failed to configure %s: %w", deviceName, err)
	}
	return nil
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

// setDefaultRouteTUN captures all traffic through the TUN device by adding two
// /1 routes that are more specific than the default (0/0) route. This avoids
// deleting the real default route, making cleanup and crash recovery safer.
func setDefaultRouteTUN(tunDevice string) error {
	if err := run("route", "add", "0/1", tunGateway); err != nil {
		return fmt.Errorf("failed to add 0/1 TUN route: %w", err)
	}
	if err := run("route", "add", "128/1", tunGateway); err != nil {
		// Roll back the first route so we don't leave partial state.
		run("route", "delete", "0/1")
		return fmt.Errorf("failed to add 128/1 TUN route: %w", err)
	}
	return nil
}

// restoreDefaultRoute removes the /1 overlay routes and ensures the original
// default route via gateway is present. Handles both new (/1 overlay) and old
// (single default replacement) code so crash-recovery works across upgrades.
func restoreDefaultRoute(gateway string) error {
	// Remove the two /1 overlay routes (new approach). Ignore errors — they
	// may not be present if an old version of the code was used.
	run("route", "delete", "0/1")
	run("route", "delete", "128/1")
	// Ensure the original default route is present. If we never deleted it
	// (new approach) this is a no-op; if an old crash left it missing this
	// restores it.
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
