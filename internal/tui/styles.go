package tui

import "github.com/charmbracelet/lipgloss"

// Adaptive colors that work on light and dark terminals.
var (
	colorPurple    = lipgloss.AdaptiveColor{Light: "#7B2FBE", Dark: "#B97EFF"}
	colorGreen     = lipgloss.AdaptiveColor{Light: "#04B575", Dark: "#04B575"}
	colorRed       = lipgloss.AdaptiveColor{Light: "#FF4672", Dark: "#FF4672"}
	colorAmber     = lipgloss.AdaptiveColor{Light: "#FF8C00", Dark: "#FFA500"}
	colorSubtle    = lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"}
	colorHighlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#B97EFF"}
	colorFg        = lipgloss.AdaptiveColor{Light: "#1A1A2E", Dark: "#FFFDF5"}
	colorDimFg     = lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"}
	colorBorder    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
)

// Header styles.
var (
	logoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPurple).
			PaddingRight(2)

	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPurple).
			Underline(true).
			Padding(0, 2)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(colorDimFg).
				Padding(0, 2)

	tabGapStyle = lipgloss.NewStyle().
			Foreground(colorSubtle).
			PaddingRight(1)
)

// Connection status pill styles.
var (
	connectedPillStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(colorGreen).
				Padding(0, 1)

	disconnectedPillStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(colorRed).
				Padding(0, 1)

	connectingPillStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(colorAmber).
				Padding(0, 1)
)

// Footer / help bar styles.
var (
	helpBarStyle = lipgloss.NewStyle().
			Foreground(colorDimFg).
			Padding(0, 1)

	helpKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPurple)

	helpDescStyle = lipgloss.NewStyle().
			Foreground(colorDimFg)

	helpSepStyle = lipgloss.NewStyle().
			Foreground(colorSubtle)
)

// General content styles.
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPurple).
			MarginBottom(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(colorAmber)

	dimStyle = lipgloss.NewStyle().
			Foreground(colorDimFg)

	// For status cards / dashboard.
	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2)

	cardTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPurple).
			MarginBottom(1)

	cardLabelStyle = lipgloss.NewStyle().
			Foreground(colorDimFg).
			Width(14)

	cardValueStyle = lipgloss.NewStyle().
			Foreground(colorFg)
)

// Latency color coding.
func latencyStyle(ms int) lipgloss.Style {
	switch {
	case ms < 100:
		return lipgloss.NewStyle().Foreground(colorGreen)
	case ms < 500:
		return lipgloss.NewStyle().Foreground(colorAmber)
	default:
		return lipgloss.NewStyle().Foreground(colorRed)
	}
}

// Spinner style.
var spinnerStyle = lipgloss.NewStyle().Foreground(colorPurple)

// Notification styles.
var (
	notifSuccessStyle = lipgloss.NewStyle().
				Foreground(colorGreen).
				Bold(true).
				Padding(0, 1)

	notifErrorStyle = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true).
			Padding(0, 1)
)
