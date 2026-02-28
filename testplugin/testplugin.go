// Package testplugin is a demo system manager plugin that displays basic system information.
// It implements api.PluginIF and can be used:
//   - Standalone: via cmd/testplugin (GUI or TUI)
//   - Embedded: via cmd/sysmanager (as a static built-in plugin)
//   - Dynamic: via pluginentry/testplugin compiled as a .so
package testplugin

import (
	"fmt"
	"os"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	tea "github.com/charmbracelet/bubbletea"
)

// Plugin displays basic system information (hostname, OS, arch, CPUs, Go version).
type Plugin struct{}

// New returns a new testplugin Plugin.
func New() *Plugin { return &Plugin{} }

// Name returns the plugin display name.
// Implements api.PluginIF.
func (p *Plugin) Name() string { return "System Info" }

// Content builds the Fyne widget tree showing system information.
// Implements api.PluginIF.
func (p *Plugin) Content(_ fyne.Window) fyne.CanvasObject {
	hostname, _ := os.Hostname()
	return container.NewVBox(
		widget.NewRichTextFromMarkdown("## System Info"),
		widget.NewSeparator(),
		widget.NewLabel(fmt.Sprintf("Hostname : %s", hostname)),
		widget.NewLabel(fmt.Sprintf("OS       : %s", runtime.GOOS)),
		widget.NewLabel(fmt.Sprintf("Arch     : %s", runtime.GOARCH)),
		widget.NewLabel(fmt.Sprintf("CPUs     : %d", runtime.NumCPU())),
		widget.NewLabel(fmt.Sprintf("Go       : %s", runtime.Version())),
	)
}

// Model returns a Bubbletea tea.Model showing system information.
// Implements api.PluginIF.
func (p *Plugin) Model() tea.Model {
	hostname, _ := os.Hostname()
	return sysInfoModel{hostname: hostname}
}

// sysInfoModel is the TUI model for the testplugin.
type sysInfoModel struct {
	hostname string
}

func (m sysInfoModel) Init() tea.Cmd { return nil }

func (m sysInfoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m sysInfoModel) View() string {
	return fmt.Sprintf(
		"\n  System Info\n  %s\n\n  Hostname : %s\n  OS       : %s\n  Arch     : %s\n  CPUs     : %d\n  Go       : %s\n\n  q / ctrl+c: quit\n",
		"───────────────────────────",
		m.hostname, runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), runtime.Version(),
	)
}
