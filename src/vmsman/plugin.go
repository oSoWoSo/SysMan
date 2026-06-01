//go:build !tui_only

package vmman

import (
	"os"

	tea "charm.land/bubbletea/v2"
	"codeberg.org/oSoWoSo/SysMan/src/common"
	"fyne.io/fyne/v2"
)

// Plugin is the VMman plugin.
type Plugin struct {
	vmDir     string
	statusBar *common.StatusBar
}

// New creates a new Plugin.
func New(vmDir string) *Plugin {
	return &Plugin{vmDir: vmDir}
}

// Name returns the plugin name.
func (p *Plugin) Name() string { return t("tab.name") }

// SetStatusBar sets a shared status bar for tooltips and messages.
// Implements api.PluginIF.
// SetStatusBar sets the status bar.
func (p *Plugin) SetStatusBar(statusBar *common.StatusBar) {
	p.statusBar = statusBar
}

// Content returns the GUI content.
func (p *Plugin) Content(win fyne.Window) fyne.CanvasObject {
	g := &guiApp{win: win, backend: NewQEMUBackend(p.resolveVMDir())}
	if p.statusBar != nil {
		g.statusBar = p.statusBar
	}
	g.vms = g.backend.List()
	return g.buildContent()
}

// Model returns the TUI model.
func (p *Plugin) Model() tea.Model {
	return NewTuiModel(NewQEMUBackend(p.resolveVMDir()))
}

func (p *Plugin) resolveVMDir() string {
	// First check VMDIR environment variable (highest priority)
	if vmDir := os.Getenv("VMDIR"); vmDir != "" {
		return ResolveVMDir(vmDir)
	}
	// Then check if a specific vmDir was passed that differs from default
	if p.vmDir != "" && p.vmDir != DefaultVMDir {
		return ResolveVMDir(p.vmDir)
	}
	// Then check config file
	cfg := common.LoadSysManConfig()
	if cfg.Vmsman.VMDir != "" {
		return ResolveVMDir(cfg.Vmsman.VMDir)
	}
	// Finally fall back to default ~/vm
	return ResolveVMDir("")
}
