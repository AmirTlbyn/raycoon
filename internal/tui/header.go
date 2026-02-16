package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var tabNames = []string{"Groups", "Configs", "Status", "Settings"}

func renderHeader(activeTab int, connected bool, connecting bool, configName string, width int) string {
	// Logo.
	logo := logoStyle.Render("RAYCOON")

	// Status pill.
	var pill string
	switch {
	case connecting:
		pill = connectingPillStyle.Render(" CONNECTING ")
	case connected:
		label := " CONNECTED "
		if configName != "" {
			label = fmt.Sprintf(" %s ", configName)
		}
		pill = connectedPillStyle.Render(label)
	default:
		pill = disconnectedPillStyle.Render(" DISCONNECTED ")
	}

	// Tabs.
	var tabs []string
	for i, name := range tabNames {
		if i == activeTab {
			tabs = append(tabs, activeTabStyle.Render(name))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(name))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...)

	// First row: logo + pill right-aligned.
	pillWidth := lipgloss.Width(pill)
	logoWidth := lipgloss.Width(logo)
	gap := width - logoWidth - pillWidth
	if gap < 1 {
		gap = 1
	}
	topRow := logo + strings.Repeat(" ", gap) + pill

	// Separator.
	sep := lipgloss.NewStyle().
		Foreground(colorBorder).
		Render(strings.Repeat("─", max(width, 0)))

	return lipgloss.JoinVertical(lipgloss.Left, topRow, tabBar, sep)
}

func renderFooter(helpText string, width int) string {
	sep := lipgloss.NewStyle().
		Foreground(colorBorder).
		Render(strings.Repeat("─", max(width, 0)))
	return lipgloss.JoinVertical(lipgloss.Left, sep, helpBarStyle.Render(helpText))
}

func renderHelpBar(showFull bool) string {
	if showFull {
		return renderFullHelp()
	}
	return renderShortHelp()
}

func renderShortHelp() string {
	bindings := keys.ShortHelp()
	var parts []string
	for _, b := range bindings {
		if !b.Enabled() {
			continue
		}
		k := helpKeyStyle.Render(b.Help().Key)
		d := helpDescStyle.Render(b.Help().Desc)
		parts = append(parts, k+" "+d)
	}
	return strings.Join(parts, helpSepStyle.Render(" | "))
}

func renderFullHelp() string {
	groups := keys.FullHelp()
	var lines []string
	for _, group := range groups {
		var parts []string
		for _, b := range group {
			if !b.Enabled() {
				continue
			}
			k := helpKeyStyle.Render(b.Help().Key)
			d := helpDescStyle.Render(b.Help().Desc)
			parts = append(parts, k+" "+d)
		}
		lines = append(lines, strings.Join(parts, helpSepStyle.Render("  ")))
	}
	return strings.Join(lines, "\n")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
