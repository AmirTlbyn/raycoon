//go:build !windows

package tun

import "os"

// checkPrivileges returns an error if not running as root.
func checkPrivileges() error {
	if os.Geteuid() != 0 {
		return ErrNotRoot
	}
	return nil
}
