// Command sysman-tui runs the system manager aggregator as a standalone Bubbletea TUI
// with clickable tabs for switching between all modules.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"codeberg.org/oSoWoSo/SysMan/src/infman"
	"codeberg.org/oSoWoSo/SysMan/src/pkgman"
	serman "codeberg.org/oSoWoSo/SysMan/src/serman"
	"codeberg.org/oSoWoSo/SysMan/src/srcman"
	"codeberg.org/oSoWoSo/SysMan/src/ugsman"
	"codeberg.org/oSoWoSo/SysMan/src/vmsman"
	zone "github.com/lrstanley/bubblezone/v2"
)

func main() {
	serman.InitI18n()
	pkgman.InitI18n()
	srcman.InitI18n()
	infman.InitI18n()
	ugsman.InitI18n()
	vmman.InitI18n()

	serviceDir := os.Getenv("SERVICEDIR")
	if serviceDir == "" {
		serviceDir = serman.DefaultServiceDir
	}
	serviceDestDir := os.Getenv("SERVICEDESTDIR")
	if serviceDestDir == "" {
		serviceDestDir = serman.DefaultServiceDestDir
	}

	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Println("sysman-tui\n\nTabs: 1-9 or click to switch\nQuit: ctrl+c")
			os.Exit(0)
		}
	}

	tabs := []tabEntry{
		{name: "Services", model: serman.NewTuiModel(serman.NewRunitBackend(serviceDir, serviceDestDir))},
		{name: "Packages", model: pkgman.NewTuiModel()},
		{name: "Templates", model: srcman.NewTuiModel("")},
		{name: "System Info", model: infman.NewTuiModel()},
		{name: "Users & Groups", model: ugsman.NewTuiModel()},
		{name: "VMs", model: vmman.NewTuiModel(vmman.NewQEMUBackend(vmman.DefaultVMDir))},
	}

	m := newTabModel(tabs)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// ── Tab model ────────────────────────────────────────────────────────

type tabEntry struct {
	name  string
	model tea.Model
}

type tabModel struct {
	id     string
	tabs   []tabEntry
	active int
}

func newTabModel(tabs []tabEntry) tabModel {
	zone.NewGlobal()
	return tabModel{id: zone.NewPrefix(), tabs: tabs}
}

func (m tabModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, t := range m.tabs {
		if cmd := t.model.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (m tabModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		if key.String() == "ctrl+c" {
			return m, tea.Quit
		}
		for i := range m.tabs {
			if key.String() == fmt.Sprintf("%d", i+1) {
				m.active = i
				return m, nil
			}
		}
	}

	// Handle mouse clicks on tabs.
	if mouse, ok := msg.(tea.MouseClickMsg); ok && mouse.Button == tea.MouseLeft {
		for i, t := range m.tabs {
			if zone.Get(m.id + t.name).InBounds(mouse) {
				if i != m.active {
					m.active = i
					return m, nil
				}
			}
		}
	}

	// Non-key events go to all plugins (window resize, async results).
	if _, isKey := msg.(tea.KeyPressMsg); !isKey {
		var cmds []tea.Cmd
		for i, t := range m.tabs {
			updated, cmd := t.model.Update(msg)
			m.tabs[i].model = updated
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)
	}

	// Key events go only to the active plugin.
	updated, cmd := m.tabs[m.active].model.Update(msg)
	m.tabs[m.active].model = updated
	return m, cmd
}

// ── Styles ───────────────────────────────────────────────────────────

var (
	tuiTabActive   = lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.Color("#00DDFF")).Padding(0, 1)
	tuiTabInactive = lipgloss.NewStyle().Foreground(lipgloss.Color("#585858")).Padding(0, 1)
	tuiTabHelp     = lipgloss.NewStyle().Foreground(lipgloss.Color("#585858"))
	tuiTabBar      = lipgloss.NewStyle().BorderBottom(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#585858"))
)

func (m tabModel) View() tea.View {
	var tabLabels []string
	for i, t := range m.tabs {
		label := fmt.Sprintf("%d %s", i+1, t.name)
		if i == m.active {
			tabLabels = append(tabLabels, zone.Mark(m.id+t.name, tuiTabActive.Render(label)))
		} else {
			tabLabels = append(tabLabels, zone.Mark(m.id+t.name, tuiTabInactive.Render(label)))
		}
	}
	bar := lipgloss.JoinHorizontal(lipgloss.Top, tabLabels...)
	bar += "  " + tuiTabHelp.Render("1-9: switch  click: switch  ctrl+c: quit")
	v := tea.NewView(zone.Scan(tuiTabBar.Render(bar) + "\n" + m.tabs[m.active].model.View().Content))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
