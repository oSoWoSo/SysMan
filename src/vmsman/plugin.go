//go:build !tui_only

package vmman

import (
	"os"
	"path/filepath"
	"strings"

	commonconfig "codeberg.org/oSoWoSo/SysMan/src/common"
	"codeberg.org/oSoWoSo/SysMan/src/common"
	"fyne.io/fyne/v2"
	tea "github.com/charmbracelet/bubbletea"
)

type Plugin struct {
	vmDir     string
	statusBar *common.StatusBar
}

func New(vmDir string) *Plugin {
	return &Plugin{vmDir: vmDir}
}

func (p *Plugin) Name() string { return t("tab.name") }

// SetStatusBar sets a shared status bar for tooltips and messages.
// Implements api.PluginIF.
func (p *Plugin) SetStatusBar(statusBar *common.StatusBar) {
	p.statusBar = statusBar
}

func (p *Plugin) Content(win fyne.Window) fyne.CanvasObject {
	g := &guiApp{win: win, backend: NewQEMUBackend(p.resolveVmDir())}
	return g.buildContent()
}

func (p *Plugin) Model() tea.Model {
	return NewTuiModel(NewQEMUBackend(p.resolveVmDir()))
}

func expandHome(p string) string {
	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}

func (p *Plugin) resolveVmDir() string {
	if p.vmDir != "" && p.vmDir != DefaultVmDir {
		return expandHome(p.vmDir)
	}
	cfg := commonconfig.LoadSysManConfig()
	if cfg.Vmsman.VmDir != "" {
		return expandHome(cfg.Vmsman.VmDir)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, DefaultVmDir)
}
