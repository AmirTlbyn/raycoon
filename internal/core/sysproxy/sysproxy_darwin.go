package sysproxy

import (
	"fmt"
	"os/exec"
	"strings"
)

// Enable sets macOS system proxy to route all traffic through the local SOCKS/HTTP proxy.
func Enable(socksPort, httpPort int) error {
	ifaces, err := activeNetworkServices()
	if err != nil {
		return fmt.Errorf("failed to detect network services: %w", err)
	}

	for _, iface := range ifaces {
		// Enable SOCKS proxy.
		if err := run("networksetup", "-setsocksfirewallproxy", iface, "127.0.0.1", fmt.Sprint(socksPort)); err != nil {
			return fmt.Errorf("failed to set SOCKS proxy on %s: %w", iface, err)
		}
		if err := run("networksetup", "-setsocksfirewallproxystate", iface, "on"); err != nil {
			return fmt.Errorf("failed to enable SOCKS proxy on %s: %w", iface, err)
		}

		// Enable HTTP proxy.
		if err := run("networksetup", "-setwebproxy", iface, "127.0.0.1", fmt.Sprint(httpPort)); err != nil {
			return fmt.Errorf("failed to set HTTP proxy on %s: %w", iface, err)
		}
		if err := run("networksetup", "-setwebproxystate", iface, "on"); err != nil {
			return fmt.Errorf("failed to enable HTTP proxy on %s: %w", iface, err)
		}

		// Enable HTTPS proxy.
		if err := run("networksetup", "-setsecurewebproxy", iface, "127.0.0.1", fmt.Sprint(httpPort)); err != nil {
			return fmt.Errorf("failed to set HTTPS proxy on %s: %w", iface, err)
		}
		if err := run("networksetup", "-setsecurewebproxystate", iface, "on"); err != nil {
			return fmt.Errorf("failed to enable HTTPS proxy on %s: %w", iface, err)
		}
	}

	return nil
}

// Disable removes the system proxy settings on all active interfaces.
func Disable() error {
	ifaces, err := activeNetworkServices()
	if err != nil {
		return fmt.Errorf("failed to detect network services: %w", err)
	}

	var firstErr error
	for _, iface := range ifaces {
		if err := run("networksetup", "-setsocksfirewallproxystate", iface, "off"); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := run("networksetup", "-setwebproxystate", iface, "off"); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := run("networksetup", "-setsecurewebproxystate", iface, "off"); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

// activeNetworkServices returns ALL non-disabled network services.
// Previous implementation only matched known keywords (Wi-Fi, Ethernet, etc.)
// which missed services on some macOS configurations.
func activeNetworkServices() ([]string, error) {
	out, err := exec.Command("networksetup", "-listallnetworkservices").Output()
	if err != nil {
		return nil, err
	}

	var services []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		// Skip header line, empty lines, and disabled services (marked with *).
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
