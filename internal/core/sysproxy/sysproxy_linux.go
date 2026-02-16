package sysproxy

import (
	"fmt"
	"os/exec"
)

const gnomeProxySchema = "org.gnome.system.proxy"

// Enable sets the GNOME system proxy (Ubuntu/GNOME desktops) to route traffic
// through the local SOCKS/HTTP proxy.
func Enable(socksPort, httpPort int) error {
	commands := [][]string{
		// Set proxy mode to manual.
		{"gsettings", "set", gnomeProxySchema, "mode", "manual"},

		// SOCKS proxy.
		{"gsettings", "set", gnomeProxySchema + ".socks", "host", "127.0.0.1"},
		{"gsettings", "set", gnomeProxySchema + ".socks", "port", fmt.Sprint(socksPort)},

		// HTTP proxy.
		{"gsettings", "set", gnomeProxySchema + ".http", "host", "127.0.0.1"},
		{"gsettings", "set", gnomeProxySchema + ".http", "port", fmt.Sprint(httpPort)},

		// HTTPS proxy.
		{"gsettings", "set", gnomeProxySchema + ".https", "host", "127.0.0.1"},
		{"gsettings", "set", gnomeProxySchema + ".https", "port", fmt.Sprint(httpPort)},
	}

	for _, args := range commands {
		if err := exec.Command(args[0], args[1:]...).Run(); err != nil {
			return fmt.Errorf("failed to run %v: %w", args, err)
		}
	}

	return nil
}

// Disable removes the GNOME system proxy settings.
func Disable() error {
	return exec.Command("gsettings", "set", gnomeProxySchema, "mode", "none").Run()
}
