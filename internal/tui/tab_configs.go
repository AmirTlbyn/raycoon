package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"raycoon/internal/storage"
	"raycoon/internal/storage/models"
)

type configsModel struct {
	table   table.Model
	configs []*models.Config
	width   int
	height  int

	// Filter state.
	filterGroupID   *int64
	filterGroupName string

	// Testing state.
	testingSingle bool
	testingBatch  bool
	batchProgress progress.Model
	batchCurrent  int
	batchTotal    int
}

func newConfigsModel() configsModel {
	cols := []table.Column{
		{Title: "ID", Width: 5},
		{Title: "Name", Width: 25},
		{Title: "Protocol", Width: 10},
		{Title: "Address", Width: 25},
		{Title: "Latency", Width: 10},
		{Title: "Group", Width: 15},
	}

	t := table.New(
		table.WithColumns(cols),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(colorBorder).
		BorderBottom(true).
		Bold(true).
		Foreground(colorPurple)
	s.Selected = s.Selected.
		Foreground(colorFg).
		Background(lipgloss.AdaptiveColor{Light: "#E8E0F0", Dark: "#2A1A3E"}).
		Bold(true)
	t.SetStyles(s)

	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithoutPercentage(),
	)

	return configsModel{
		table:         t,
		batchProgress: p,
	}
}

func (cm *configsModel) setSize(w, h int) {
	cm.width = w
	cm.height = h
	cm.adjustTableHeight()

	// Adjust column widths proportionally.
	if w > 100 {
		cm.table.SetColumns([]table.Column{
			{Title: "ID", Width: 5},
			{Title: "Name", Width: w/4 - 5},
			{Title: "Protocol", Width: 10},
			{Title: "Address", Width: w/4 - 2},
			{Title: "Latency", Width: 10},
			{Title: "Group", Width: w/4 - 8},
		})
	}
	cm.batchProgress.Width = w - 4
}

// adjustTableHeight sets the table height based on how many overhead lines are
// actually rendered (filter info line and/or testing indicator line).
// table.SetHeight(h) already accounts for the header rows internally, so we
// only need to subtract the lines we render above the table.
func (cm *configsModel) adjustTableHeight() {
	overhead := 0
	if cm.filterGroupName != "" {
		overhead++
	}
	if cm.testingSingle || cm.testingBatch {
		overhead++
	}
	th := cm.height - overhead
	if th < 1 {
		th = 1
	}
	cm.table.SetHeight(th)
}

func (cm *configsModel) setConfigs(configs []*models.Config, store storage.Storage) {
	cm.configs = configs
	ctx := context.Background()

	rows := make([]table.Row, len(configs))
	for i, c := range configs {
		// Get latency.
		latStr := "-"
		lat, err := store.GetLatestLatency(ctx, c.ID)
		if err == nil && lat != nil && lat.Success && lat.LatencyMS != nil {
			latStr = fmt.Sprintf("%dms", *lat.LatencyMS)
		} else if err == nil && lat != nil && !lat.Success {
			latStr = "fail"
		}

		// Get group name.
		groupName := ""
		group, err := store.GetGroup(ctx, c.GroupID)
		if err == nil {
			groupName = group.Name
		}

		rows[i] = table.Row{
			fmt.Sprintf("%d", c.ID),
			truncate(c.Name, 30),
			c.Protocol,
			fmt.Sprintf("%s:%d", c.Address, c.Port),
			latStr,
			groupName,
		}
	}
	cm.table.SetRows(rows)
	cm.table.GotoTop() // always start at row 0 when new data arrives
}

func (cm *configsModel) selectedConfig() *models.Config {
	idx := cm.table.Cursor()
	if idx >= 0 && idx < len(cm.configs) {
		return cm.configs[idx]
	}
	return nil
}

func (cm *configsModel) updateProgress(msg latencyTestProgressMsg) {
	cm.batchCurrent = msg.current
	cm.batchTotal = msg.total
}

func (cm *configsModel) Update(msg tea.Msg, root *Model) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Connect) || key.Matches(msg, keys.Enter):
			cfg := cm.selectedConfig()
			if cfg != nil && !root.connecting {
				root.connecting = true
				if root.connected {
					// Disconnect first, then connect to new config.
					return tea.Sequence(
						disconnect(root.store, root.coreMgr),
						func() tea.Msg { return connectStartedMsg{} },
						connectToConfig(root.store, root.coreMgr, cfg),
					)
				}
				return connectToConfig(root.store, root.coreMgr, cfg)
			}

		case key.Matches(msg, keys.TestSingle):
			cfg := cm.selectedConfig()
			if cfg != nil && !cm.testingSingle && !cm.testingBatch {
				cm.testingSingle = true
				cm.adjustTableHeight()
				strategy, _, timeoutMS := root.getLatencySettings()
				return testSingleLatency(root.store, cfg, strategy, timeoutMS)
			}

		case key.Matches(msg, keys.TestBatch):
			if len(cm.configs) > 0 && !cm.testingBatch && !cm.testingSingle {
				cm.testingBatch = true
				cm.batchCurrent = 0
				cm.batchTotal = len(cm.configs)
				cm.adjustTableHeight()
				strategy, workers, timeoutMS := root.getLatencySettings()
				return testBatchLatency(root.store, cm.configs, root.program, strategy, workers, timeoutMS)
			}

		case key.Matches(msg, keys.Back):
			if cm.filterGroupID != nil {
				cm.filterGroupID = nil
				cm.filterGroupName = ""
				cm.adjustTableHeight()
				return loadConfigs(root.store, nil)
			}
		}
	}

	var cmd tea.Cmd
	cm.table, cmd = cm.table.Update(msg)
	return cmd
}

func (cm *configsModel) View(s spinner.Model) string {
	var b strings.Builder

	// Filter info.
	if cm.filterGroupName != "" {
		b.WriteString(dimStyle.Render(fmt.Sprintf("Filtered by: %s (esc to clear)", cm.filterGroupName)))
		b.WriteString("\n")
	}

	// Testing indicator.
	if cm.testingSingle {
		b.WriteString(s.View() + " Testing latency...\n")
	} else if cm.testingBatch {
		pct := 0.0
		if cm.batchTotal > 0 {
			pct = float64(cm.batchCurrent) / float64(cm.batchTotal)
		}
		b.WriteString(fmt.Sprintf("%s Testing %d/%d ", s.View(), cm.batchCurrent, cm.batchTotal))
		b.WriteString(cm.batchProgress.ViewAs(pct))
		b.WriteString("\n")
	}

	// Table.
	b.WriteString(cm.table.View())

	return forceHeight(b.String(), cm.width, cm.height)
}

// Color the latency cell in the rendered view.
func formatLatency(ms int) string {
	return latencyStyle(ms).Render(fmt.Sprintf("%dms", ms))
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-1] + "~"
}
