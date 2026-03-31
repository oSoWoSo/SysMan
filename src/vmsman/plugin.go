//go:build !tui_only

package vmman

import (
	"os"
	"path/filepath"
	"strings"

	"codeberg.org/oSoWoSo/SysMan/src/config"
	"fyne.io/fyne/v2"
	tea "github.com/charmbracelet/bubbletea"
)

type Plugin struct {
	vmDir string
}

func New(vmDir string) *Plugin {
	return &Plugin{vmDir: vmDir}
}

func (p *Plugin) Name() string { return t("tab.name") }

func (p *Plugin) Content(win fyne.Window) fyne.CanvasObject {
	g := &guiApp{win: win, backend: NewQEMUBackend(p.resolveVmDir())}
	return g.buildContent()
}

func (p *Plugin) Model() tea.Model {
	return NewTuiModel(NewQEMUBackend(p.resolveVmDir()))
}

func (p *Plugin) resolveVmDir() string {
	if p.vmDir != "" && p.vmDir != DefaultVmDir {
		if strings.HasPrefix(p.vmDir, "~/") {
			home, err := os.UserHomeDir()
			if err == nil {
				return filepath.Join(home, p.vmDir[2:])
			}
		}
		return p.vmDir
	}
	cfg := config.LoadSysManConfig()
	if cfg.Vmsman.VmDir != "" {
		if strings.HasPrefix(cfg.Vmsman.VmDir, "~/") {
			home, err := os.UserHomeDir()
			if err == nil {
				return filepath.Join(home, cfg.Vmsman.VmDir[2:])
			}
		}
		return cfg.Vmsman.VmDir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, DefaultVmDir)
}
