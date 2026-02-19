package xray

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"raycoon/internal/core/types"
	"raycoon/internal/paths"
)

// Xray implements the ProxyCore interface for Xray-core
type Xray struct {
	cmd           *exec.Cmd
	configPath    string
	logPath       string
	pidPath       string
	startTime     time.Time
	running       int32 // atomic: 0=not running, 1=running
	mu            sync.Mutex
	logBuffer     *bytes.Buffer
	statsCollector *statsCollector
}

// New creates a new Xray core instance
func New() (*Xray, error) {
	// Check if xray binary exists
	xrayPath, err := findXrayBinary()
	if err != nil {
		return nil, fmt.Errorf("xray binary not found: %w (install from https://github.com/XTLS/Xray-core)", err)
	}

	// Create temp directories (use real user's home even under sudo).
	tmpDir, err := paths.CacheDir()
	if err != nil {
		return nil, err
	}

	x := &Xray{
		configPath: filepath.Join(tmpDir, "config.json"),
		logPath:    filepath.Join(tmpDir, "xray.log"),
		pidPath:    filepath.Join(tmpDir, "xray.pid"),
		logBuffer:  &bytes.Buffer{},
	}

	// Verify xray works
	cmd := exec.Command(xrayPath, "version")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("xray binary is not executable: %w", err)
	}

	return x, nil
}

// Start starts the Xray core
func (x *Xray) Start(ctx context.Context, config *types.CoreConfig) error {
	x.mu.Lock()
	defer x.mu.Unlock()

	if atomic.LoadInt32(&x.running) == 1 {
		return fmt.Errorf("xray is already running")
	}

	// Generate Xray config
	xrayConfig, err := generateXrayConfig(config)
	if err != nil {
		return fmt.Errorf("failed to generate xray config: %w", err)
	}

	// Write config to file
	configJSON, err := json.MarshalIndent(xrayConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(x.configPath, configJSON, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	// When invoked via sudo, make the config readable by the real user so
	// xray (which will run as the real user) can load it.
	paths.ChownToRealUser(x.configPath)

	// Find xray binary
	xrayPath, err := findXrayBinary()
	if err != nil {
		return err
	}

	// Create command - use exec.Command (not CommandContext) so xray survives CLI exit
	x.cmd = exec.Command(xrayPath, "run", "-c", x.configPath)

	// Set XRAY_LOCATION_ASSET so xray finds geoip.dat and geosite.dat
	xrayDir := filepath.Dir(xrayPath)
	x.cmd.Env = append(os.Environ(), "XRAY_LOCATION_ASSET="+xrayDir)

	// Detach process from parent so it survives when CLI exits.
	x.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// When invoked via sudo, drop xray back to the real user's UID/GID so
	// that a later non-root "raycoon disconnect" can signal it. TUN setup
	// is done by the parent process (which keeps root), so xray itself
	// does not need elevated privileges.
	if uid, gid, ok := paths.RealUser(); ok {
		x.cmd.SysProcAttr.Credential = &syscall.Credential{
			Uid: uint32(uid),
			Gid: uint32(gid),
		}
	}

	// Setup logging
	logFile, err := os.Create(x.logPath)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}
	paths.ChownToRealUser(x.logPath)

	// Create multi-writer to write to both file and buffer
	multiWriter := io.MultiWriter(logFile, x.logBuffer)
	x.cmd.Stdout = multiWriter
	x.cmd.Stderr = multiWriter

	// Start process
	if err := x.cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("failed to start xray: %w", err)
	}

	atomic.StoreInt32(&x.running, 1)
	x.startTime = time.Now()

	// Save PID to file for cross-process tracking
	pidStr := fmt.Sprintf("%d", x.cmd.Process.Pid)
	os.WriteFile(x.pidPath, []byte(pidStr), 0644)
	paths.ChownToRealUser(x.pidPath)

	// Initialize stats collector
	x.statsCollector = newStatsCollector(x.logPath)

	// Close the log file after a short delay - the process is detached
	// and writes directly to its stdout/stderr which go to the log file
	go func() {
		// Wait for process to exit (for cleanup)
		x.cmd.Wait()
		atomic.StoreInt32(&x.running, 0)
		logFile.Close()
		os.Remove(x.pidPath)
	}()

	// Wait briefly to check if xray exits immediately (config errors etc.)
	time.Sleep(1 * time.Second)

	if atomic.LoadInt32(&x.running) == 0 {
		logContent, _ := os.ReadFile(x.logPath)
		if len(logContent) > 0 {
			return fmt.Errorf("xray failed to start:\n%s", string(logContent))
		}
		return fmt.Errorf("xray failed to start, check logs at: %s", x.logPath)
	}

	return nil
}

// Stop stops the Xray core
func (x *Xray) Stop(ctx context.Context) error {
	x.mu.Lock()
	defer x.mu.Unlock()

	// Try to find the process - either via cmd or PID file
	var proc *os.Process

	if x.cmd != nil && x.cmd.Process != nil {
		proc = x.cmd.Process
	} else {
		// Try PID file (cross-process stop)
		pidBytes, err := os.ReadFile(x.pidPath)
		if err == nil {
			pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
			if err == nil {
				proc, err = os.FindProcess(pid)
				if err != nil {
					proc = nil
				}
			}
		}
	}

	if proc == nil {
		atomic.StoreInt32(&x.running, 0)
		os.Remove(x.pidPath)
		return nil
	}

	// Send interrupt signal
	if err := proc.Signal(os.Interrupt); err != nil {
		// Force kill if interrupt fails
		if killErr := proc.Kill(); killErr != nil {
			// Process might already be dead
			atomic.StoreInt32(&x.running, 0)
			os.Remove(x.pidPath)
			return nil
		}
	}

	// Wait for process to exit with timeout
	done := make(chan struct{}, 1)
	go func() {
		proc.Wait()
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		// Timeout, force kill
		proc.Kill()
	}

	atomic.StoreInt32(&x.running, 0)
	os.Remove(x.pidPath)
	return nil
}

// Restart restarts the Xray core with new configuration
func (x *Xray) Restart(ctx context.Context, config *types.CoreConfig) error {
	if err := x.Stop(ctx); err != nil {
		return err
	}

	time.Sleep(500 * time.Millisecond)

	return x.Start(ctx, config)
}

// IsRunning returns whether Xray is currently running
func (x *Xray) IsRunning() bool {
	if atomic.LoadInt32(&x.running) == 1 {
		return true
	}

	// Check PID file (cross-process check)
	pidBytes, err := os.ReadFile(x.pidPath)
	if err != nil {
		return false
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if err != nil {
		return false
	}

	// Check if process is actually running
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, FindProcess always succeeds. Send signal 0 to check if alive.
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// GetStatus returns the current status
func (x *Xray) GetStatus() (*types.Status, error) {
	running := atomic.LoadInt32(&x.running) == 1

	status := &types.Status{
		Running:  running,
		CoreType: "xray",
	}

	if running && x.cmd != nil && x.cmd.Process != nil {
		status.PID = x.cmd.Process.Pid
		status.StartedAt = x.startTime
		status.Uptime = time.Since(x.startTime)
	}

	return status, nil
}

// GetStats returns real-time statistics
func (x *Xray) GetStats() (*types.Stats, error) {
	if atomic.LoadInt32(&x.running) == 0 || x.statsCollector == nil {
		return &types.Stats{}, nil
	}

	return x.statsCollector.GetStats()
}

// GetVersion returns the Xray version
func (x *Xray) GetVersion() (string, error) {
	xrayPath, err := findXrayBinary()
	if err != nil {
		return "", err
	}

	cmd := exec.Command(xrayPath, "version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get xray version: %w", err)
	}

	// Parse version from output
	scanner := bufio.NewScanner(bytes.NewReader(output))
	if scanner.Scan() {
		return scanner.Text(), nil
	}

	return string(output), nil
}

// GetLogs returns the log output
func (x *Xray) GetLogs() (io.Reader, error) {
	// Read from log file
	if _, err := os.Stat(x.logPath); err == nil {
		file, err := os.Open(x.logPath)
		if err != nil {
			return nil, err
		}
		return file, nil
	}

	// Fallback to buffer
	return bytes.NewReader(x.logBuffer.Bytes()), nil
}

// findXrayBinary finds the xray binary in common locations
func findXrayBinary() (string, error) {
	// Check common locations
	locations := []string{
		"xray",                // In PATH
		"/usr/local/bin/xray",
		"/usr/bin/xray",
		"/opt/xray/xray",
	}

	// Also check in real user's home directory (works under sudo).
	if homeDir, err := paths.HomeDir(); err == nil {
		locations = append(locations, filepath.Join(homeDir, ".local", "bin", "xray"))
		locations = append(locations, filepath.Join(homeDir, ".local", "share", "raycoon", "cores", "xray"))
	}

	for _, loc := range locations {
		path, err := exec.LookPath(loc)
		if err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("xray binary not found in any common location")
}
