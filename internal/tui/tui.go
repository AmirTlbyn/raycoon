package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"raycoon/internal/config/parser"
	"raycoon/internal/core"
	"raycoon/internal/storage"
	"raycoon/internal/storage/models"
	"raycoon/internal/subscription"
)

// Tab indices.
const (
	tabGroups   = 0
	tabConfigs  = 1
	tabStatus   = 2
	tabSettings = 3
	tabCount    = 4
)

// Model is the root BubbleTea model.
type Model struct {
	// Dependencies.
	store   storage.Storage
	coreMgr *core.Manager
	subMgr  *subscription.Manager
	program *tea.Program

	// Dimensions.
	width  int
	height int

	// Navigation.
	activeTab int
	showHelp  bool

	// Connection state.
	connected    bool
	connecting   bool
	activeConn   *models.ActiveConnection
	activeConfig *models.Config

	// Tab models.
	groupsTab   groupsModel
	configsTab  configsModel
	statusTab   statusModel
	settingsTab settingsModel

	// Notification.
	notification    string
	notificationErr bool
	notifVersion    int

	// Spinner for async operations.
	spinner spinner.Model
}

// Deps holds all dependencies injected into the TUI.
type Deps struct {
	Storage storage.Storage
	CoreMgr *core.Manager
	Parser  *parser.Registry
	SubMgr  *subscription.Manager
}

// NewModel creates a new root Model.
func NewModel(deps Deps) *Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = spinnerStyle

	return &Model{
		store:       deps.Storage,
		coreMgr:     deps.CoreMgr,
		subMgr:      deps.SubMgr,
		activeTab:   tabGroups,
		spinner:     s,
		groupsTab:   newGroupsModel(),
		configsTab:  newConfigsModel(),
		statusTab:   newStatusModel(),
		settingsTab: newSettingsModel(),
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		loadGroups(m.store),
		loadConfigs(m.store, nil),
		loadActiveConnection(m.store),
		loadSettings(m.store),
		m.spinner.Tick,
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	prevNotifVersion := m.notifVersion

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		ch := m.contentHeight()
		m.groupsTab.setSize(msg.Width, ch)
		m.configsTab.setSize(msg.Width, ch)
		m.statusTab.setSize(msg.Width, ch)
		m.settingsTab.setSize(msg.Width, ch)
		return m, nil

	case tea.KeyMsg:
		if cmd := m.handleGlobalKey(msg); cmd != nil {
			return m, cmd
		}

	// Data loading.
	case groupsLoadedMsg:
		if msg.err == nil {
			m.groupsTab.setGroups(msg.groups, m.store)
		}
	case configsLoadedMsg:
		if msg.err == nil {
			m.configsTab.setConfigs(msg.configs, m.store)
		}
	case settingsLoadedMsg:
		if msg.err == nil {
			m.settingsTab.setSettings(msg.settings)
		}
	case activeConnLoadedMsg:
		m.activeConn = msg.conn
		m.activeConfig = msg.config
		m.connected = msg.conn != nil && msg.config != nil
		if m.connected {
			cmds = append(cmds, statusTick())
		}

	// Connection.
	case connectStartedMsg:
		m.connecting = true
	case connectResultMsg:
		m.connecting = false
		if msg.err != nil {
			m.setNotification(fmt.Sprintf("Connect failed: %v", msg.err), true)
		} else {
			m.connected = true
			m.activeConfig = msg.config
			m.setNotification(fmt.Sprintf("Connected to %s", msg.config.Name), false)
			cmds = append(cmds, loadActiveConnection(m.store), statusTick())
		}
	case disconnectResultMsg:
		m.connecting = false
		if msg.err != nil {
			m.setNotification(fmt.Sprintf("Disconnect failed: %v", msg.err), true)
		} else {
			m.connected = false
			m.activeConn = nil
			m.activeConfig = nil
			m.setNotification("Disconnected", false)
		}

	// Status polling.
	case statusTickMsg:
		if m.connected && m.activeTab == tabStatus {
			cmds = append(cmds, pollStatus(m.coreMgr))
		}
		if m.connected {
			cmds = append(cmds, statusTick())
		}
	case statusResultMsg:
		m.statusTab.updateStatus(msg)
		if !msg.running && m.connected {
			m.connected = false
			m.setNotification("Connection lost - core process stopped", true)
		}

	// Latency.
	case latencyTestProgressMsg:
		m.configsTab.updateProgress(msg)
	case latencyTestDoneMsg:
		m.configsTab.testingBatch = false
		if msg.err != nil {
			m.setNotification(fmt.Sprintf("Batch test failed: %v", msg.err), true)
		} else {
			m.setNotification(
				fmt.Sprintf("Tested %d: %d ok, %d failed",
					msg.batch.Tested, msg.batch.Succeeded, msg.batch.Failed), false)
		}
		cmds = append(cmds, loadConfigs(m.store, m.configsTab.filterGroupID))
	case singleLatencyDoneMsg:
		m.configsTab.testingSingle = false
		if msg.result.Latency.Success {
			m.setNotification(
				fmt.Sprintf("%s: %dms", msg.result.Config.Name, *msg.result.Latency.LatencyMS), false)
		} else {
			m.setNotification(fmt.Sprintf("%s: failed", msg.result.Config.Name), true)
		}
		cmds = append(cmds, loadConfigs(m.store, m.configsTab.filterGroupID))

	// Subscription.
	case subUpdateResultMsg:
		m.groupsTab.updating = false
		if msg.err != nil {
			m.setNotification(fmt.Sprintf("Update failed: %v", msg.err), true)
		} else {
			m.setNotification(
				fmt.Sprintf("Updated %s: +%d -%d",
					msg.result.GroupName, msg.result.Added, msg.result.Removed), false)
			cmds = append(cmds, loadGroups(m.store), loadConfigs(m.store, m.configsTab.filterGroupID))
		}

	// Settings.
	case settingSavedMsg:
		if msg.err != nil {
			m.setNotification(fmt.Sprintf("Save failed: %v", msg.err), true)
		} else {
			m.setNotification(fmt.Sprintf("Saved %s", msg.key), false)
		}

	// Notification.
	case clearNotificationMsg:
		if msg.version == m.notifVersion {
			m.notification = ""
			m.notificationErr = false
		}
	}

	// Spinner.
	if m.connecting || m.configsTab.testingSingle || m.configsTab.testingBatch || m.groupsTab.updating {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Schedule notification auto-clear when a new notification was set.
	if m.notifVersion > prevNotifVersion && m.notification != "" {
		cmds = append(cmds, clearNotification(4*time.Second, m.notifVersion))
	}

	// Delegate to active tab.
	switch m.activeTab {
	case tabGroups:
		cmds = append(cmds, m.groupsTab.Update(msg, m))
	case tabConfigs:
		cmds = append(cmds, m.configsTab.Update(msg, m))
	case tabStatus:
		cmds = append(cmds, m.statusTab.Update(msg, m))
	case tabSettings:
		cmds = append(cmds, m.settingsTab.Update(msg, m))
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	connName := ""
	if m.activeConfig != nil {
		connName = m.activeConfig.Name
	}
	header := renderHeader(m.activeTab, m.connected, m.connecting, connName, m.width)

	var content string
	switch m.activeTab {
	case tabGroups:
		content = m.groupsTab.View(m.spinner)
	case tabConfigs:
		content = m.configsTab.View(m.spinner)
	case tabStatus:
		content = m.statusTab.View(m.connected, m.activeConfig, m.activeConn)
	case tabSettings:
		content = m.settingsTab.View()
	}

	var notif string
	if m.notification != "" {
		if m.notificationErr {
			notif = notifErrorStyle.Render("! " + m.notification)
		} else {
			notif = notifSuccessStyle.Render("* " + m.notification)
		}
	}

	helpText := renderHelpBar(m.showHelp)
	footer := renderFooter(helpText, m.width)

	parts := []string{header}
	if notif != "" {
		parts = append(parts, notif)
	}
	parts = append(parts, content, footer)
	output := lipgloss.JoinVertical(lipgloss.Left, parts...)

	// Force exactly m.height lines to prevent BubbleTea rendering drift.
	return forceHeight(output, m.width, m.height)
}

// forceHeight ensures the string has exactly `height` lines, each padded to `width`.
// This prevents BubbleTea from leaving ghost lines when switching tabs.
func forceHeight(s string, width, height int) string {
	lines := strings.Split(s, "\n")
	// Truncate excess lines.
	if len(lines) > height {
		lines = lines[:height]
	}
	// Pad missing lines with blank space.
	blank := strings.Repeat(" ", width)
	for len(lines) < height {
		lines = append(lines, blank)
	}
	return strings.Join(lines, "\n")
}

func (m *Model) contentHeight() int {
	overhead := 5
	if m.showHelp {
		overhead += 3
	}
	h := m.height - overhead
	if h < 1 {
		h = 1
	}
	return h
}

func (m *Model) handleGlobalKey(msg tea.KeyMsg) tea.Cmd {
	// Don't intercept when settings editing or search active.
	if m.activeTab == tabSettings && m.settingsTab.editing {
		return nil
	}

	switch {
	case key.Matches(msg, keys.Quit):
		return tea.Quit

	case key.Matches(msg, keys.Help):
		m.showHelp = !m.showHelp
		return nil

	case key.Matches(msg, keys.TabNext):
		m.activeTab = (m.activeTab + 1) % tabCount
		if m.activeTab == tabStatus && m.connected {
			return pollStatus(m.coreMgr)
		}
		return nil

	case key.Matches(msg, keys.TabPrev):
		m.activeTab = (m.activeTab - 1 + tabCount) % tabCount
		if m.activeTab == tabStatus && m.connected {
			return pollStatus(m.coreMgr)
		}
		return nil

	case key.Matches(msg, keys.Disconnect):
		if m.connected && !m.connecting {
			m.connecting = true
			return disconnect(m.store, m.coreMgr)
		}
		return nil

	case key.Matches(msg, keys.Refresh):
		return tea.Batch(
			loadGroups(m.store),
			loadConfigs(m.store, m.configsTab.filterGroupID),
			loadActiveConnection(m.store),
			loadSettings(m.store),
		)
	}

	return nil
}

func (m *Model) setNotification(text string, isErr bool) {
	m.notification = text
	m.notificationErr = isErr
	m.notifVersion++
}

func (m *Model) getLatencySettings() (string, int64, int64) {
	strategy := "tcp"
	var workers int64 = 10
	var timeoutMS int64 = 5000

	settings := m.settingsTab.settings
	if v, ok := settings["latency_test_strategy"]; ok {
		strategy = v
	}
	if v, ok := settings["latency_test_workers"]; ok {
		var n int64
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			workers = n
		}
	}
	if v, ok := settings["latency_test_timeout"]; ok {
		var n int64
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			timeoutMS = n
		}
	}
	return strategy, workers, timeoutMS
}

// NewProgram creates a bubbletea program with alt screen.
func NewProgram(deps Deps) *tea.Program {
	m := NewModel(deps)
	p := tea.NewProgram(m, tea.WithAltScreen())
	m.program = p
	return p
}
