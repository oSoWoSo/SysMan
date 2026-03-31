//go:build !tui_only

package vmman

import (
	"os"
	"path/filepath"
	"strings"

	"codeberg.org/oSoWoSo/SysMan/src/common"
	"fyne.io/fyne/v2"
	tea "github.com/charmbracelet/bubbletea"
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
	return g.buildContent()
}

// Model returns the TUI model.
func (p *Plugin) Model() tea.Model {
	return NewTuiModel(NewQEMUBackend(p.resolveVMDir()))
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

func (p *Plugin) resolveVMDir() string {
	if p.vmDir != "" && p.vmDir != DefaultVMDir {
		return expandHome(p.vmDir)
	}
	cfg := common.LoadSysManConfig()
	if cfg.Vmsman.VMDir != "" {
		return expandHome(cfg.Vmsman.VMDir)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, DefaultVMDir)
}
