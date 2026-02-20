package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// settingKind distinguishes free-text settings from choice-based settings.
type settingKind int

const (
	settingText   settingKind = iota // Free-text input (ports, numbers).
	settingChoice                    // Cycle through predefined options.
)

// settingDef defines a setting's display metadata.
type settingDef struct {
	key         string
	label       string
	description string
	defaultVal  string
	kind        settingKind
	choices     []string // Only for settingChoice.
}

var settingDefs = []settingDef{
	{key: "vpn_mode", label: "VPN Mode", description: "How traffic is routed", defaultVal: "proxy", kind: settingChoice, choices: []string{"proxy", "tun"}},
	{key: "proxy_port", label: "SOCKS Port", description: "SOCKS5 proxy port", defaultVal: "1080", kind: settingText},
	{key: "http_proxy_port", label: "HTTP Port", description: "HTTP proxy port", defaultVal: "1081", kind: settingText},
	{key: "active_core", label: "Core", description: "Proxy core engine", defaultVal: "xray", kind: settingChoice, choices: []string{"xray", "singbox"}},
	{key: "latency_test_workers", label: "Test Workers", description: "Concurrent latency test workers", defaultVal: "10", kind: settingText},
	{key: "latency_test_timeout", label: "Test Timeout", description: "Latency test timeout (ms)", defaultVal: "5000", kind: settingText},
	{key: "latency_test_strategy", label: "Test Strategy", description: "Latency test method", defaultVal: "tcp", kind: settingChoice, choices: []string{"tcp", "http"}},
}

type settingsModel struct {
	settings map[string]string
	cursor   int
	editing  bool
	input    textinput.Model
	width    int
	height   int
}

func newSettingsModel() settingsModel {
	ti := textinput.New()
	ti.CharLimit = 64
	ti.Prompt = "> "
	ti.PromptStyle = lipgloss.NewStyle().Foreground(colorPurple)
	ti.TextStyle = lipgloss.NewStyle().Foreground(colorFg)

	return settingsModel{
		settings: make(map[string]string),
		input:    ti,
	}
}

func (sm *settingsModel) setSize(w, h int) {
	sm.width = w
	sm.height = h
	sm.input.Width = w / 2
}

func (sm *settingsModel) setSettings(s map[string]string) {
	sm.settings = s
}

func (sm *settingsModel) currentDef() settingDef {
	if sm.cursor >= 0 && sm.cursor < len(settingDefs) {
		return settingDefs[sm.cursor]
	}
	return settingDefs[0]
}

func (sm *settingsModel) currentValue() string {
	def := sm.currentDef()
	if v, ok := sm.settings[def.key]; ok {
		return v
	}
	return def.defaultVal
}

// choiceIndex returns the current index in the choices slice for a choice setting.
func (sm *settingsModel) choiceIndex(def settingDef) int {
	val := sm.currentValue()
	for i, c := range def.choices {
		if c == val {
			return i
		}
	}
	return 0
}

func (sm *settingsModel) Update(msg tea.Msg, root *Model) tea.Cmd {
	if sm.editing {
		return sm.updateEditing(msg, root)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		def := sm.currentDef()

		switch msg.String() {
		case "up", "k":
			if sm.cursor > 0 {
				sm.cursor--
			}
		case "down", "j":
			if sm.cursor < len(settingDefs)-1 {
				sm.cursor++
			}
		case "enter":
			if def.kind == settingChoice {
				// Cycle to next choice on enter.
				return sm.cycleChoice(root, 1)
			}
			// Text setting: open editor.
			sm.editing = true
			sm.input.SetValue(sm.currentValue())
			sm.input.Focus()
			return textinput.Blink
		case "left", "h":
			if def.kind == settingChoice {
				return sm.cycleChoice(root, -1)
			}
		case "right", "l":
			if def.kind == settingChoice {
				return sm.cycleChoice(root, 1)
			}
		}
	}
	return nil
}

// cycleChoice moves to the next/prev choice and saves it.
func (sm *settingsModel) cycleChoice(root *Model, dir int) tea.Cmd {
	def := sm.currentDef()
	idx := sm.choiceIndex(def)
	idx = (idx + dir + len(def.choices)) % len(def.choices)
	val := def.choices[idx]
	sm.settings[def.key] = val
	return saveSetting(root.store, def.key, val)
}

func (sm *settingsModel) updateEditing(msg tea.Msg, root *Model) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Back):
			sm.editing = false
			sm.input.Blur()
			return nil
		case msg.String() == "enter":
			sm.editing = false
			sm.input.Blur()
			def := sm.currentDef()
			val := sm.input.Value()
			sm.settings[def.key] = val
			return saveSetting(root.store, def.key, val)
		}
	}

	var cmd tea.Cmd
	sm.input, cmd = sm.input.Update(msg)
	return cmd
}

func (sm *settingsModel) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Settings"))
	b.WriteString("\n\n")

	for i, def := range settingDefs {
		isSelected := i == sm.cursor

		val := def.defaultVal
		if v, ok := sm.settings[def.key]; ok {
			val = v
		}

		var line string
		if isSelected {
			label := lipgloss.NewStyle().Bold(true).Foreground(colorPurple).Width(18).Render("> " + def.label)
			if sm.editing {
				line = label + sm.input.View()
			} else if def.kind == settingChoice {
				line = label + sm.renderChoices(def, val)
			} else {
				value := lipgloss.NewStyle().Foreground(colorFg).Render(val)
				line = label + value
			}
		} else {
			label := lipgloss.NewStyle().Foreground(colorFg).Width(18).Render("  " + def.label)
			if def.kind == settingChoice {
				value := lipgloss.NewStyle().Foreground(colorDimFg).Render(val)
				line = label + value
			} else {
				value := lipgloss.NewStyle().Foreground(colorDimFg).Render(val)
				line = label + value
			}
		}

		b.WriteString(line + "\n")

		// Show description for selected item.
		if isSelected && !sm.editing {
			hint := def.description
			if def.kind == settingChoice {
				hint += "  (enter/arrows to change)"
			} else {
				hint += fmt.Sprintf("  (enter to edit, default: %s)", def.defaultVal)
			}
			b.WriteString(lipgloss.NewStyle().
				Foreground(colorDimFg).
				PaddingLeft(2).
				Render("  "+hint) + "\n")
		}
	}

	return forceHeight(b.String(), sm.width, sm.height)
}

// renderChoices renders the choice selector with the active choice highlighted.
func (sm *settingsModel) renderChoices(def settingDef, current string) string {
	var parts []string
	for _, c := range def.choices {
		if c == current {
			parts = append(parts, lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPurple).
				Render("["+c+"]"))
		} else {
			parts = append(parts, lipgloss.NewStyle().
				Foreground(colorDimFg).
				Render(" "+c+" "))
		}
	}
	return strings.Join(parts, " ")
}
