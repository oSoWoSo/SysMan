// Command sysmanager is a demo system manager that embeds multiple plugins.
//
// Built-in plugins (always available):
//   - Services        (svman runit service manager)
//   - Packages        (xbps package manager)
//   - Templates       (xbps-src template manager)
//   - System Info     (sysinfo)
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
	"strings"

	"image/color"

	"codeberg.org/oSoWoSo/SysMan/src/common"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"codeberg.org/oSoWoSo/SysMan/src/api"
	"codeberg.org/oSoWoSo/SysMan/src/infman"
	"codeberg.org/oSoWoSo/SysMan/src/pkgman"
	serman "codeberg.org/oSoWoSo/SysMan/src/serman"
	"codeberg.org/oSoWoSo/SysMan/src/srcman"
	"codeberg.org/oSoWoSo/SysMan/src/ugsman"
	"codeberg.org/oSoWoSo/SysMan/src/vmsman"
)

func main() {
	serman.InitI18n()

	serviceDir := os.Getenv("SERVICEDIR")
	if serviceDir == "" {
		serviceDir = serman.DefaultServiceDir
	}
	serviceDestDir := os.Getenv("SERVICEDESTDIR")
	if serviceDestDir == "" {
		serviceDestDir = serman.DefaultServiceDestDir
	}

	mode := "auto"
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

	hasDisplay := os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""

	if mode == "auto" {
		if hasDisplay {
			mode = "gui"
		} else {
			mode = "tui"
		}
	}

	// Explicit --gui with no display falls back to TUI.
	if mode == "gui" && !hasDisplay {
		fmt.Fprintln(os.Stderr, "sysmanager: no display available, falling back to TUI")
		mode = "tui"
	}

	// Built-in plugins — always present, no rebuild needed for these.
	// Create a shared status bar for all plugins.
	statusBar := common.NewStatusBar()
	plugins := []api.PluginIF{
		infman.New(),
		pkgman.New(),
		srcman.New(""),
		serman.New(serviceDir, serviceDestDir),
		ugsman.New(),
		vmman.New(vmman.DefaultVMDir),
	}

	// Set the shared status bar on all plugins.
	for _, p := range plugins {
		p.SetStatusBar(statusBar)
	}

	// Dynamic plugins — loaded from PLUGIN_DIR (default: ./plugins/).
	// Drop a compiled .so here and restart; no rebuild of sysmanager required.
	extra := loadDynamic(pluginDir())
	plugins = append(plugins, extra...)

	// Set shared status bar on dynamic plugins too.
	for _, p := range extra {
		p.SetStatusBar(statusBar)
	}

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
	common.SetWindowIcon(win)

	// Build content panels for each plugin + settings.
	contents := make([]fyne.CanvasObject, len(plugins))
	for i, p := range plugins {
		contents[i] = p.Content(win)
	}
	settingsContent := buildSettingsContent(win)
	allContent := append(contents, settingsContent)

	// Show only the active panel.
	show := func(idx int) {
		for i, c := range allContent {
			if i == idx {
				c.Show()
			} else {
				c.Hide()
			}
		}
	}

	// Hide all except first on startup.
	for i, c := range allContent {
		if i != 0 {
			c.Hide()
		}
	}
	stack := container.NewStack(allContent...)

	// Build tab bar: plugin buttons left, spacer, settings icon right.
	var tabBtns []*widget.Button
	barItems := make([]fyne.CanvasObject, 0, len(plugins)+2)
	for i, p := range plugins {
		idx := i
		btn := widget.NewButton(p.Name(), func() {
			show(idx)
			for j, b := range tabBtns {
				if j == idx {
					b.Importance = widget.HighImportance
				} else {
					b.Importance = widget.LowImportance
				}
				b.Refresh()
			}
		})
		if i == 0 {
			btn.Importance = widget.HighImportance
		} else {
			btn.Importance = widget.LowImportance
		}
		tabBtns = append(tabBtns, btn)
		barItems = append(barItems, btn)
	}
	settingsIdx := len(plugins)
	btnSettingsIcon := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		show(settingsIdx)
		for _, b := range tabBtns {
			b.Importance = widget.LowImportance
			b.Refresh()
		}
	})
	btnSettingsIcon.Importance = widget.LowImportance
	barItems = append(barItems, layout.NewSpacer(), btnSettingsIcon)

	tabBar := container.NewHBox(barItems...)
	tabBar = container.NewBorder(nil, nil, nil, nil, tabBar) // full width

	win.SetContent(container.NewBorder(
		container.NewVBox(container.NewPadded(tabBar), widget.NewSeparator()),
		nil, nil, nil,
		stack,
	))
	win.Resize(fyne.NewSize(1024, 768))
	win.SetMaster()
	win.ShowAndRun()
}

func buildSettingsContent(win fyne.Window) fyne.CanvasObject {
	cfg := common.LoadSysManConfig()

	sermanServiceDir := newFormEntry(cfg.Serman.ServiceDir, serman.DefaultServiceDir)
	sermanServiceDestDir := newFormEntry(cfg.Serman.ServiceDestDir, serman.DefaultServiceDestDir)

	srcmanDistDir := newFormEntry(cfg.Srcman.DistDir, "")
	srcmanSearchEngine := newFormEntry(cfg.Srcman.SearchEngine, "https://duckduckgo.com/?q=")

	vmsmanVMDir := newFormEntry(cfg.Vmsman.VMDir, vmman.DefaultVMDir)

	btnSave := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		cfg.Serman.ServiceDir = strings.TrimSpace(sermanServiceDir.Text)
		cfg.Serman.ServiceDestDir = strings.TrimSpace(sermanServiceDestDir.Text)
		cfg.Srcman.DistDir = strings.TrimSpace(srcmanDistDir.Text)
		cfg.Srcman.SearchEngine = strings.TrimSpace(srcmanSearchEngine.Text)
		cfg.Vmsman.VMDir = strings.TrimSpace(vmsmanVMDir.Text)
		if err := common.SaveSysManConfig(cfg); err != nil {
			dialog.ShowError(fmt.Errorf("save config: %w", err), win)
		}
	})
	btnSave.Importance = widget.HighImportance

	titleStyle := canvas.NewText("Settings", color.NRGBA{R: 0x00, G: 0xb8, B: 0xd4, A: 0xff})
	titleStyle.TextStyle = fyne.TextStyle{Bold: true}
	titleStyle.TextSize = 20

	dividerColor := color.NRGBA{R: 0x2e, G: 0x34, B: 0x3b, A: 0xff}
	headerColor := color.NRGBA{R: 0x00, G: 0xb8, B: 0xd4, A: 0xff}

	headerSerman := canvas.NewText("Serman", headerColor)
	headerSerman.TextStyle = fyne.TextStyle{Bold: true}
	headerSerman.TextSize = 14

	headerSrcman := canvas.NewText("Srcman", headerColor)
	headerSrcman.TextStyle = fyne.TextStyle{Bold: true}
	headerSrcman.TextSize = 14

	headerVmsman := canvas.NewText("Vmsman", headerColor)
	headerVmsman.TextStyle = fyne.TextStyle{Bold: true}
	headerVmsman.TextSize = 14

	sep1 := canvas.NewRectangle(dividerColor)
	sep1.SetMinSize(fyne.NewSize(0, 1))
	sep2 := canvas.NewRectangle(dividerColor)
	sep2.SetMinSize(fyne.NewSize(0, 1))
	sep3 := canvas.NewRectangle(dividerColor)
	sep3.SetMinSize(fyne.NewSize(0, 1))

	formSerman := widget.NewForm(
		widget.NewFormItem("Service Dir", sermanServiceDir),
		widget.NewFormItem("Service Dest Dir", sermanServiceDestDir),
	)

	formSrcman := widget.NewForm(
		widget.NewFormItem("void-packages dir", srcmanDistDir),
		widget.NewFormItem("Search engine URL", srcmanSearchEngine),
	)

	formVmsman := widget.NewForm(
		widget.NewFormItem("VM dir", vmsmanVMDir),
	)

	return container.NewVBox(
		container.NewPadded(titleStyle),
		container.NewPadded(sep1),
		container.NewPadded(headerSerman),
		container.NewPadded(formSerman),
		container.NewPadded(sep2),
		container.NewPadded(headerSrcman),
		container.NewPadded(formSrcman),
		container.NewPadded(sep3),
		container.NewPadded(headerVmsman),
		container.NewPadded(formVmsman),
		layout.NewSpacer(),
		container.NewPadded(btnSave),
	)
}

func newFormEntry(value, placeholder string) *widget.Entry {
	e := widget.NewEntry()
	e.SetPlaceHolder(placeholder)
	e.SetText(value)
	return e
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
		// 1, 2, … switch tabs (up to 9 plugins).
		for i := range m.plugins {
			if key.String() == fmt.Sprintf("%d", i+1) {
				m.active = i
				return m, nil
			}
		}
	}

	newModels := make([]tea.Model, len(m.models))
	copy(newModels, m.models)

	// Window resize and async result messages go to all plugins so background
	// commands (e.g. package list loading) complete regardless of which tab is active.
	if _, isKey := msg.(tea.KeyMsg); !isKey {
		var cmds []tea.Cmd
		for i, mdl := range newModels {
			updated, cmd := mdl.Update(msg)
			newModels[i] = updated
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		m.models = newModels
		return m, tea.Batch(cmds...)
	}

	// Key events go only to the active plugin.
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
		label := fmt.Sprintf("%d %s", i+1, p.Name())
		if i == m.active {
			bar += tuiTabActive.Render(label)
		} else {
			bar += tuiTabInactive.Render(label)
		}
	}
	bar += "  " + tuiTabHelp.Render("1-9: switch tab  ctrl+c: quit")
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
