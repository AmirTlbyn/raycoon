package paths

import (
	"os"
	"os/user"
	"path/filepath"
	"strconv"
)

// HomeDir returns the real user's home directory, even when running under sudo.
// Under sudo, os.UserHomeDir() returns /var/root (macOS) or /root (Linux),
// but we want the invoking user's home so that PID files, configs, and DB
// are in the same location regardless of privilege level.
func HomeDir() (string, error) {
	// Check SUDO_USER first — set by sudo to the original invoking user.
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		u, err := user.Lookup(sudoUser)
		if err == nil {
			return u.HomeDir, nil
		}
	}
	return os.UserHomeDir()
}

// RealUser returns the UID and GID of the real invoking user when running
// under sudo (via SUDO_UID / SUDO_GID). Returns ok=false when not under sudo.
func RealUser() (uid, gid int, ok bool) {
	sudoUID := os.Getenv("SUDO_UID")
	if sudoUID == "" {
		return 0, 0, false
	}
	u, err := strconv.ParseInt(sudoUID, 10, 64)
	if err != nil {
		return 0, 0, false
	}
	var g int64
	if sudoGID := os.Getenv("SUDO_GID"); sudoGID != "" {
		g, _ = strconv.ParseInt(sudoGID, 10, 64)
	}
	return int(u), int(g), true
}

// ChownToRealUser changes the owner of path to the real invoking user when
// running under sudo. It is a no-op when not under sudo.
func ChownToRealUser(path string) {
	if uid, gid, ok := RealUser(); ok {
		os.Chown(path, uid, gid)
	}
}

// CacheDir returns ~/.cache/raycoon, creating it if needed.
// When running under sudo the directory is chowned to the real user so that
// later non-root invocations can read/write PID files and logs.
func CacheDir() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".cache", "raycoon")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	ChownToRealUser(dir)
	return dir, nil
}

// DataDir returns ~/.local/share/raycoon, creating it if needed.
// When running under sudo the directory is chowned to the real user so that
// later non-root invocations can access the database.
func DataDir() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".local", "share", "raycoon")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	ChownToRealUser(dir)
	return dir, nil
}

// ConfigDir returns ~/.config/raycoon, creating it if needed.
// When running under sudo the directory is chowned to the real user so that
// later non-root invocations can read/write settings.
func ConfigDir() (string, error) {
	home, err := HomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".config", "raycoon")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	ChownToRealUser(dir)
	return dir, nil
}
