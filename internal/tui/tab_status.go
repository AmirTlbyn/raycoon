package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"raycoon/internal/core/types"
	"raycoon/internal/storage/models"
)

type statusModel struct {
	width  int
	height int

	running bool
	status  *types.Status
	stats   *types.Stats
}

func newStatusModel() statusModel {
	return statusModel{}
}

func (sm *statusModel) setSize(w, h int) {
	sm.width = w
	sm.height = h
}

func (sm *statusModel) updateStatus(msg statusResultMsg) {
	sm.running = msg.running
	sm.status = msg.status
	sm.stats = msg.stats
}

func (sm *statusModel) Update(msg tea.Msg, root *Model) tea.Cmd {
	return nil
}

func (sm *statusModel) View(connected bool, config *models.Config, conn *models.ActiveConnection) string {
	var content string
	if !connected || config == nil || conn == nil {
		content = sm.viewDisconnected()
	} else {
		content = sm.viewConnected(config, conn)
	}
	return forceHeight(content, sm.width, sm.height)
}

func (sm *statusModel) viewDisconnected() string {
	content := lipgloss.JoinVertical(lipgloss.Left,
		cardTitleStyle.Render("Connection Status"),
		"",
		lipgloss.NewStyle().Foreground(colorDimFg).Render("Not connected"),
		"",
		dimStyle.Render("Go to Configs tab and press 'c' to connect"),
	)

	w := sm.width - 6
	if w < 30 {
		w = 30
	}
	return cardStyle.Width(w).Render(content)
}

func (sm *statusModel) viewConnected(config *models.Config, conn *models.ActiveConnection) string {
	var sections []string

	// Connection card.
	connRows := []string{
		sm.row("Status", successStyle.Render("Connected")),
		sm.row("Config", config.Name),
		sm.row("Protocol", config.Protocol),
		sm.row("Address", fmt.Sprintf("%s:%d", config.Address, config.Port)),
		sm.row("Core", conn.CoreType),
		sm.row("Mode", conn.VPNMode),
		sm.row("Started", conn.StartedAt.Format("15:04:05")),
		sm.row("Uptime", formatDuration(time.Since(conn.StartedAt))),
	}

	if sm.status != nil && sm.status.PID > 0 {
		connRows = append(connRows, sm.row("PID", fmt.Sprintf("%d", sm.status.PID)))
	}

	connCard := lipgloss.JoinVertical(lipgloss.Left,
		append([]string{cardTitleStyle.Render("Connection")}, connRows...)...,
	)
	sections = append(sections, connCard)

	// Stats card.
	if sm.stats != nil {
		statsRows := []string{
			sm.row("Upload", formatBytes(sm.stats.TotalUpload)),
			sm.row("Download", formatBytes(sm.stats.TotalDownload)),
			sm.row("Up Speed", formatBytes(sm.stats.UploadSpeed)+"/s"),
			sm.row("Down Speed", formatBytes(sm.stats.DownloadSpeed)+"/s"),
		}
		if sm.stats.ActiveConns > 0 {
			statsRows = append(statsRows, sm.row("Connections", fmt.Sprintf("%d", sm.stats.ActiveConns)))
		}

		statsCard := lipgloss.JoinVertical(lipgloss.Left,
			append([]string{cardTitleStyle.Render("Traffic")}, statsRows...)...,
		)
		sections = append(sections, statsCard)
	}

	// Layout: side by side if wide enough.
	w := sm.width - 6
	if w < 30 {
		w = 30
	}

	if len(sections) == 2 && sm.width > 80 {
		halfW := (w - 4) / 2
		left := cardStyle.Width(halfW).Render(sections[0])
		right := cardStyle.Width(halfW).Render(sections[1])
		return lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
	}

	var rendered []string
	for _, s := range sections {
		rendered = append(rendered, cardStyle.Width(w).Render(s))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rendered...)
}

func (sm *statusModel) row(label, value string) string {
	return cardLabelStyle.Render(label+":") + " " + cardValueStyle.Render(value)
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm %ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func formatBytes(b uint64) string {
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// padRight pads s to width with spaces.
func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
