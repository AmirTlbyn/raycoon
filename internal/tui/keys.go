package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Quit       key.Binding
	Help       key.Binding
	TabNext    key.Binding
	TabPrev    key.Binding
	Enter      key.Binding
	Back       key.Binding
	Connect    key.Binding
	Disconnect key.Binding
	TestSingle key.Binding
	TestBatch  key.Binding
	Update     key.Binding
	Refresh    key.Binding
	Search     key.Binding
}

var keys = keyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	TabNext: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next tab"),
	),
	TabPrev: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev tab"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Back: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "back"),
	),
	Connect: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "connect"),
	),
	Disconnect: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "disconnect"),
	),
	TestSingle: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "test latency"),
	),
	TestBatch: key.NewBinding(
		key.WithKeys("T"),
		key.WithHelp("T", "test all"),
	),
	Update: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "update sub"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
}

// ShortHelp returns a compact list for the help bar.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.TabNext, k.Connect, k.Disconnect, k.TestSingle, k.Refresh, k.Help, k.Quit}
}

// FullHelp returns grouped bindings for the expanded help view.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.TabNext, k.TabPrev, k.Enter, k.Back},
		{k.Connect, k.Disconnect, k.TestSingle, k.TestBatch},
		{k.Update, k.Refresh, k.Search},
		{k.Help, k.Quit},
	}
}
