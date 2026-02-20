package tui

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"raycoon/internal/storage"
	"raycoon/internal/storage/models"
)

// groupItem implements list.Item for the groups list.
type groupItem struct {
	group       *models.Group
	configCount int
	hasSub      bool
}

func (i groupItem) Title() string       { return i.group.Name }
func (i groupItem) FilterValue() string { return i.group.Name }
func (i groupItem) Description() string {
	parts := []string{fmt.Sprintf("%d configs", i.configCount)}
	if i.hasSub {
		parts = append(parts, "sub")
		if i.group.AutoUpdate {
			parts = append(parts, "auto")
		}
	}
	if i.group.IsGlobal {
		parts = append(parts, "global")
	}
	return strings.Join(parts, " | ")
}

// groupItemDelegate renders each group item.
type groupItemDelegate struct{}

func (d groupItemDelegate) Height() int                             { return 2 }
func (d groupItemDelegate) Spacing() int                            { return 0 }
func (d groupItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d groupItemDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	gi, ok := item.(groupItem)
	if !ok {
		return
	}

	isSelected := index == m.Index()

	title := gi.Title()
	desc := gi.Description()

	if isSelected {
		title = lipgloss.NewStyle().Bold(true).Foreground(colorPurple).Render("> " + title)
		desc = lipgloss.NewStyle().Foreground(colorDimFg).PaddingLeft(2).Render(desc)
	} else {
		title = lipgloss.NewStyle().Foreground(colorFg).Render("  " + title)
		desc = lipgloss.NewStyle().Foreground(colorDimFg).PaddingLeft(2).Render(desc)
	}

	fmt.Fprintf(w, "%s\n%s", title, desc)
}

// groupsModel manages the groups tab.
type groupsModel struct {
	list     list.Model
	groups   []*models.Group
	width    int
	height   int
	updating bool
}

func newGroupsModel() groupsModel {
	l := list.New(nil, groupItemDelegate{}, 0, 0)
	l.Title = "Groups"
	l.SetShowHelp(false)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = titleStyle
	l.Styles.FilterPrompt = lipgloss.NewStyle().Foreground(colorPurple)
	l.Styles.FilterCursor = lipgloss.NewStyle().Foreground(colorPurple)

	return groupsModel{list: l}
}

func (gm *groupsModel) setSize(w, h int) {
	gm.width = w
	gm.height = h
	gm.list.SetSize(w, h)
}

func (gm *groupsModel) setGroups(groups []*models.Group, store storage.Storage) {
	gm.groups = groups
	items := make([]list.Item, len(groups))
	ctx := context.Background()

	for i, g := range groups {
		filter := storage.ConfigFilter{GroupID: &g.ID}
		configs, _ := store.GetAllConfigs(ctx, filter)
		hasSub := g.SubscriptionURL != nil && *g.SubscriptionURL != ""
		items[i] = groupItem{
			group:       g,
			configCount: len(configs),
			hasSub:      hasSub,
		}
	}
	gm.list.SetItems(items)
}

func (gm *groupsModel) selectedGroup() *models.Group {
	item := gm.list.SelectedItem()
	if item == nil {
		return nil
	}
	gi, ok := item.(groupItem)
	if !ok {
		return nil
	}
	return gi.group
}

func (gm *groupsModel) Update(msg tea.Msg, root *Model) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// When filtering, pass all keys to list.
		if gm.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			gm.list, cmd = gm.list.Update(msg)
			return cmd
		}

		switch {
		case key.Matches(msg, keys.Enter):
			g := gm.selectedGroup()
			if g != nil {
				// Switch to configs tab filtered by this group.
				root.activeTab = tabConfigs
				root.configsTab.filterGroupID = &g.ID
				root.configsTab.filterGroupName = g.Name
				root.configsTab.adjustTableHeight() // filter row now visible
				root.configsTab.table.GotoTop()     // reset cursor so no row appears "out of range"
				return tea.Batch(
					loadConfigs(root.store, &g.ID),
					func() tea.Msg { return tea.ClearScreen() },
				)
			}

		case key.Matches(msg, keys.Update):
			g := gm.selectedGroup()
			if g != nil && !gm.updating && g.SubscriptionURL != nil && *g.SubscriptionURL != "" {
				gm.updating = true
				return updateSubscriptionWithManager(root.subMgr, g.ID)
			}

		case key.Matches(msg, keys.Search):
			gm.list.SetFilteringEnabled(true)
			// Trigger filter mode.
			var cmd tea.Cmd
			gm.list, cmd = gm.list.Update(msg)
			return cmd
		}
	}

	var cmd tea.Cmd
	gm.list, cmd = gm.list.Update(msg)
	return cmd
}

func (gm *groupsModel) View(s spinner.Model) string {
	if gm.updating {
		// Pad to full height so the footer stays pinned at the bottom.
		return forceHeight(s.View()+" Updating subscription...", gm.width, gm.height)
	}
	return forceHeight(gm.list.View(), gm.width, gm.height)
}
