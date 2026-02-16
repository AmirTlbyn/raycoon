package xray

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"raycoon/internal/core/types"
)

// statsCollector queries xray's gRPC stats API for traffic data.
type statsCollector struct {
	xrayPath  string
	apiAddr   string
	startTime time.Time

	// Cached totals from last successful query.
	lastUpload   uint64
	lastDownload uint64
	lastQueryAt  time.Time

	// Speeds calculated from deltas.
	upSpeed   uint64
	downSpeed uint64
}

// newStatsCollector creates a stats collector that queries xray API.
func newStatsCollector(logPath string) *statsCollector {
	xrayPath, _ := findXrayBinary()

	return &statsCollector{
		xrayPath:    xrayPath,
		apiAddr:     fmt.Sprintf("127.0.0.1:%d", statsAPIPort),
		startTime:   time.Now(),
		lastQueryAt: time.Now(),
	}
}

// GetStats queries xray API for real-time traffic statistics.
func (sc *statsCollector) GetStats() (*types.Stats, error) {
	stats := &types.Stats{
		TotalUpload:   sc.lastUpload,
		TotalDownload: sc.lastDownload,
		UploadSpeed:   sc.upSpeed,
		DownloadSpeed: sc.downSpeed,
	}

	if sc.xrayPath == "" {
		return stats, nil
	}

	// Query all stats from xray API.
	cmd := exec.Command(sc.xrayPath, "api", "stats", "-s", sc.apiAddr, "-pattern", "")
	output, err := cmd.Output()
	if err != nil {
		return stats, nil // Return cached stats on error.
	}

	up, down := parseStatsOutput(string(output))

	now := time.Now()
	elapsed := now.Sub(sc.lastQueryAt).Seconds()
	if elapsed > 0 && sc.lastQueryAt != sc.startTime {
		if up >= sc.lastUpload {
			sc.upSpeed = uint64(float64(up-sc.lastUpload) / elapsed)
		}
		if down >= sc.lastDownload {
			sc.downSpeed = uint64(float64(down-sc.lastDownload) / elapsed)
		}
	}

	sc.lastUpload = up
	sc.lastDownload = down
	sc.lastQueryAt = now

	stats.TotalUpload = up
	stats.TotalDownload = down
	stats.UploadSpeed = sc.upSpeed
	stats.DownloadSpeed = sc.downSpeed

	return stats, nil
}

// parseStatsOutput parses the JSON output from `xray api stats`.
// Output format: {"stat":[{"name":"inbound>>>socks-in>>>traffic>>>uplink","value":"12345"}, ...]}
func parseStatsOutput(output string) (upload, download uint64) {
	var result struct {
		Stat []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"stat"`
	}

	if err := json.Unmarshal([]byte(output), &result); err != nil {
		// Try line-by-line parsing for older xray versions.
		return parseStatsLines(output)
	}

	for _, s := range result.Stat {
		val, _ := strconv.ParseUint(s.Value, 10, 64)
		name := strings.ToLower(s.Name)
		if strings.Contains(name, "uplink") && !strings.Contains(name, "api") {
			upload += val
		} else if strings.Contains(name, "downlink") && !strings.Contains(name, "api") {
			download += val
		}
	}
	return
}

// parseStatsLines handles the line-by-line output format.
func parseStatsLines(output string) (upload, download uint64) {
	var currentName string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			currentName = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "name:")))
		} else if strings.HasPrefix(line, "value:") {
			valStr := strings.TrimSpace(strings.TrimPrefix(line, "value:"))
			val, _ := strconv.ParseUint(valStr, 10, 64)
			if strings.Contains(currentName, "uplink") && !strings.Contains(currentName, "api") {
				upload += val
			} else if strings.Contains(currentName, "downlink") && !strings.Contains(currentName, "api") {
				download += val
			}
		}
	}
	return
}

// formatBytes formats bytes to human-readable string.
func formatBytesCompact(b uint64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(gb))
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(mb))
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(kb))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// GetStatsAPIAddr returns the stats API address.
func GetStatsAPIAddr() string {
	return fmt.Sprintf("127.0.0.1:%d", statsAPIPort)
}

// GetStatsAPIPort returns the stats API port.
func GetStatsAPIPort() int {
	return statsAPIPort
}

// FindXrayBinaryPath exposes findXrayBinary for the stats collector.
func FindXrayBinaryPath() (string, error) {
	return findXrayBinary()
}

// QueryStats queries xray stats API and returns formatted upload/download strings.
func QueryStats(xrayPath, apiAddr string) (upload, download uint64, err error) {
	cmd := exec.Command(xrayPath, "api", "stats", "-s", apiAddr, "-pattern", "")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query xray stats: %w", err)
	}
	up, down := parseStatsOutput(string(output))
	return up, down, nil
}

// GetStatsPort returns the configured stats port, exported for use by the TUI status tab.
func GetStatsPort() int {
	return statsAPIPort
}

// Ensure the exported log path is still available for debugging.
func (sc *statsCollector) LogPath() string {
	return filepath.Dir(sc.xrayPath) // Approximation; the real log path is set by Xray.
}
