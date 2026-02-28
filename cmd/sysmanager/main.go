// Command sysmanager is a demo system manager that embeds multiple plugins.
//
// Built-in plugins (always available):
//   - Services  (svman runit service manager)
//   - System Info (testplugin)
//
// Dynamic plugins (optional, loaded from PLUGIN_DIR or ./plugins/):
//
//	Each .so file must export:  func New() api.PluginIF
//	Build with:  go build -buildmode=plugin -o plugins/foo.so ./pluginentry/foo/
//
// Usage:
//
//	sysmanager [--gui] [--tui]
package main

import (
	"fmt"
	"os"
	"path/filepath"
	goplugin "plugin"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"codeberg.org/oSoWoSo/svman/api"
	svman "codeberg.org/oSoWoSo/svman/plugin"
	"codeberg.org/oSoWoSo/svman/testplugin"
)

func main() {
	svman.InitI18n()

	serviceDir := os.Getenv("SERVICEDIR")
	if serviceDir == "" {
		serviceDir = svman.DefaultServiceDir
	}
	serviceDestDir := os.Getenv("SERVICEDESTDIR")
	if serviceDestDir == "" {
		serviceDestDir = svman.DefaultServiceDestDir
	}

	mode := "gui"
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--tui", "-t":
			mode = "tui"
		case "--gui", "-g":
			mode = "gui"
		case "--help", "-h":
			fmt.Printf("sysmanager [--gui|--tui]\n\nPlugin dir: %s\n", pluginDir())
			os.Exit(0)
		}
	}

	// Built-in plugins — always present, no rebuild needed for these.
	plugins := []api.PluginIF{
		svman.New(serviceDir, serviceDestDir),
		testplugin.New(),
	}

	// Dynamic plugins — loaded from PLUGIN_DIR (default: ./plugins/).
	// Drop a compiled .so here and restart; no rebuild of sysmanager required.
	extra := loadDynamic(pluginDir())
	plugins = append(plugins, extra...)

	switch mode {
	case "tui":
		runTUI(plugins)
	default:
		runGUI(plugins)
	}
}

// pluginDir returns the directory to scan for dynamic .so plugins.
func pluginDir() string {
	if d := os.Getenv("PLUGIN_DIR"); d != "" {
		return d
	}
	return "./plugins"
}

// ── Dynamic loading ───────────────────────────────────────────────────

// loadDynamic opens every *.so in dir and calls its exported New() function.
// Files that fail to load or have the wrong symbol signature are skipped with a warning.
// Returns nil (not an error) when the directory does not exist.
func loadDynamic(dir string) []api.PluginIF {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "sysmanager: reading plugin dir: %v\n", err)
		return nil
	}

	var loaded []api.PluginIF
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".so" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		p, err := goplugin.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "sysmanager: load %s: %v\n", e.Name(), err)
			continue
		}
		sym, err := p.Lookup("New")
		if err != nil {
			fmt.Fprintf(os.Stderr, "sysmanager: %s: missing New symbol\n", e.Name())
			continue
		}
		newFn, ok := sym.(func() api.PluginIF)
		if !ok {
			fmt.Fprintf(os.Stderr, "sysmanager: %s: New has wrong signature\n", e.Name())
			continue
		}
		loaded = append(loaded, newFn())
		fmt.Printf("sysmanager: loaded plugin %s (%s)\n", newFn().Name(), e.Name())
	}
	return loaded
}

// ── GUI ───────────────────────────────────────────────────────────────

func runGUI(plugins []api.PluginIF) {
	a := app.New()
	win := a.NewWindow("System Manager")

	tabs := make([]*container.TabItem, len(plugins))
	for i, p := range plugins {
		tabs[i] = container.NewTabItem(p.Name(), p.Content(win))
	}

	win.SetContent(container.NewAppTabs(tabs...))
	win.Resize(fyne.NewSize(940, 640))
	win.SetMaster()
	win.ShowAndRun()
}

// ── TUI ───────────────────────────────────────────────────────────────

type sysManagerModel struct {
	plugins []api.PluginIF
	models  []tea.Model
	active  int
}

func newSysManagerModel(plugins []api.PluginIF) sysManagerModel {
	models := make([]tea.Model, len(plugins))
	for i, p := range plugins {
		models[i] = p.Model()
	}
	return sysManagerModel{plugins: plugins, models: models}
}

func (m sysManagerModel) Init() tea.Cmd {
	var cmds []tea.Cmd
	for _, mdl := range m.models {
		if cmd := mdl.Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return tea.Batch(cmds...)
}

func (m sysManagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		if key.String() == "ctrl+c" {
			return m, tea.Quit
		}
		// F1, F2, … switch tabs (up to 9 plugins).
		for i := range m.plugins {
			if key.String() == fmt.Sprintf("f%d", i+1) {
				m.active = i
				return m, nil
			}
		}
	}

	// Forward all other input to the active plugin's model.
	newModels := make([]tea.Model, len(m.models))
	copy(newModels, m.models)
	updated, cmd := newModels[m.active].Update(msg)
	newModels[m.active] = updated
	m.models = newModels
	return m, cmd
}

var (
	tuiTabActive   = lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.Color("#00DDFF")).Padding(0, 1)
	tuiTabInactive = lipgloss.NewStyle().Foreground(lipgloss.Color("#585858")).Padding(0, 1)
	tuiTabHelp     = lipgloss.NewStyle().Foreground(lipgloss.Color("#585858"))
	tuiTabBar      = lipgloss.NewStyle().BorderBottom(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#585858"))
)

func (m sysManagerModel) View() string {
	bar := ""
	for i, p := range m.plugins {
		label := fmt.Sprintf("F%d %s", i+1, p.Name())
		if i == m.active {
			bar += tuiTabActive.Render(label)
		} else {
			bar += tuiTabInactive.Render(label)
		}
	}
	bar += "  " + tuiTabHelp.Render("ctrl+c: quit")
	return tuiTabBar.Render(bar) + "\n" + m.models[m.active].View()
}

func runTUI(plugins []api.PluginIF) {
	m := newSysManagerModel(plugins)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
