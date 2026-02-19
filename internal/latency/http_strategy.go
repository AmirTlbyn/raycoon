package latency

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"raycoon/internal/paths"

	"raycoon/internal/core/types"
	"raycoon/internal/core/xray"
	"raycoon/internal/storage/models"
)

const httpTestURL = "http://www.gstatic.com/generate_204"

// HTTPStrategy measures latency by starting a temporary xray instance and
// making an HTTP request through the SOCKS5 proxy. More accurate than TCP
// but heavier — validates the full proxy chain.
type HTTPStrategy struct{}

// NewHTTPStrategy creates a new HTTP strategy.
func NewHTTPStrategy() (*HTTPStrategy, error) {
	if _, err := findXrayBinary(); err != nil {
		return nil, fmt.Errorf("http strategy requires xray binary: %w", err)
	}
	return &HTTPStrategy{}, nil
}

func (s *HTTPStrategy) Name() string { return "http" }

func (s *HTTPStrategy) Test(ctx context.Context, config *models.Config) (int, error) {
	// Pick random free ports so parallel tests don't conflict.
	socksPort, err := freePort()
	if err != nil {
		return 0, fmt.Errorf("failed to find free SOCKS port: %w", err)
	}
	httpPort, err := freePort()
	if err != nil {
		return 0, fmt.Errorf("failed to find free HTTP port: %w", err)
	}

	// Build minimal xray config for testing.
	coreConfig := &types.CoreConfig{
		Config:     config,
		VPNMode:    types.VPNModeProxy,
		SOCKSPort:  socksPort,
		HTTPPort:   httpPort,
		LogLevel:   "none",
		DNSServers: []string{"8.8.8.8", "1.1.1.1"},
	}

	xrayConfig, err := xray.GenerateTestConfig(coreConfig)
	if err != nil {
		return 0, fmt.Errorf("failed to generate test config: %w", err)
	}

	configJSON, err := json.MarshalIndent(xrayConfig, "", "  ")
	if err != nil {
		return 0, fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write temp config with unique name per port.
	tmpDir, _ := paths.CacheDir()
	configPath := filepath.Join(tmpDir, fmt.Sprintf("latency_test_%d.json", socksPort))
	if err := os.WriteFile(configPath, configJSON, 0600); err != nil {
		return 0, fmt.Errorf("failed to write test config: %w", err)
	}
	defer os.Remove(configPath)

	xrayPath, err := findXrayBinary()
	if err != nil {
		return 0, err
	}

	// Start xray. Use a fresh context timeout to avoid inheriting parent's deadline.
	cmd := exec.Command(xrayPath, "run", "-c", configPath)
	xrayDir := filepath.Dir(xrayPath)
	cmd.Env = append(os.Environ(), "XRAY_LOCATION_ASSET="+xrayDir)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("failed to start test xray: %w", err)
	}

	// Always kill process on exit.
	defer func() {
		cmd.Process.Kill()
		cmd.Wait()
	}()

	// Wait for the SOCKS port to be ready (up to 3 seconds).
	addr := fmt.Sprintf("127.0.0.1:%d", socksPort)
	if !waitForPort(addr, 3*time.Second) {
		return 0, fmt.Errorf("xray failed to start listening on %s", addr)
	}

	// Make HTTP request through the SOCKS5 proxy.
	proxyURL, _ := url.Parse(fmt.Sprintf("socks5://127.0.0.1:%d", socksPort))
	transport := &http.Transport{
		Proxy:                 http.ProxyURL(proxyURL),
		DisableKeepAlives:     true,
		ResponseHeaderTimeout: 10 * time.Second,
	}
	client := &http.Client{
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects.
		},
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", httpTestURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("http request failed: %w", err)
	}
	resp.Body.Close()
	elapsed := time.Since(start)

	// Validate response (Google returns 204 No Content).
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return int(elapsed.Milliseconds()), nil
}

// freePort asks the OS for an available TCP port.
func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port, nil
}

// waitForPort polls a TCP address until it's accepting connections or timeout.
func waitForPort(addr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// findXrayBinary finds the xray binary in common locations.
func findXrayBinary() (string, error) {
	locations := []string{
		"xray",
		"/usr/local/bin/xray",
		"/usr/bin/xray",
		"/opt/xray/xray",
	}

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

	return "", fmt.Errorf("xray binary not found")
}
