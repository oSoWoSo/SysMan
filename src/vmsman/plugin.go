//go:build !tui_only

package vmman

import (
	"context"
	"os"

	tea "charm.land/bubbletea/v2"
	"codeberg.org/oSoWoSo/SysMan/src/common"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Plugin is the VMman plugin.
type Plugin struct {
	vmDir     string
	statusBar *common.StatusBar
	gui       *guiApp
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
	vmDir := p.resolveVMDir()
	g := &guiApp{win: win, backend: NewQEMUBackend(vmDir), vmDir: vmDir}
	p.gui = g
	if p.statusBar != nil {
		g.statusBar = p.statusBar
	}
	cfg := common.LoadSysManConfig()
	if cfg.Vmsman.VMDir == "" {
		cfg.Vmsman.VMDir = vmDir
	}
	d, _ := NewDistroApp(context.Background(), DistroConfig{
		VMs:       cfg.Vmsman.VMDir,
		SourceDir: cfg.Vmsman.SourceDir,
		RAMDisk:   cfg.Vmsman.RAMDisk,
		RAMDir:    cfg.Vmsman.RAMDir,
		ISOs:      cfg.Vmsman.ISODir,
		Templates: cfg.Vmsman.Templates,
		Terminal:  cfg.Vmsman.Terminal,
	})
	g.distro = d
	g.vms = g.backend.List()
	g.iconCache = make(map[string]fyne.Resource)
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Run VM", theme.ComputerIcon(), g.buildContent()),
		container.NewTabItemWithIcon("Create VM", theme.DownloadIcon(), g.buildDownloadContent()),
	)
	if g.bottomBar == nil {
		g.bottomBar = container.NewVBox(widget.NewSeparator(), container.NewHBox(layout.NewSpacer(), g.statusBar))
	}
	return container.NewBorder(nil, g.bottomBar, nil, nil, tabs)
}

// Model returns the TUI model.
func (p *Plugin) Model() tea.Model {
	return NewTuiModel(NewQEMUBackend(p.resolveVMDir()))
}

// ApplySettings applies vmsman settings (from a unified settings dialog) to the
// running plugin instance, mirroring the standalone vmsman settings dialog.
func (p *Plugin) ApplySettings(values map[string]string) {
	if p.gui == nil {
		return
	}
	g := p.gui
	newVM := ResolveVMDir(values["VM dir"])
	if newVM == "" {
		newVM = ResolveVMDir("")
	}
	g.vmDir = newVM
	g.backend = NewQEMUBackend(newVM)
	d, _ := NewDistroApp(context.Background(), DistroConfig{
		VMs:       newVM,
		SourceDir: values["Source dir"],
		RAMDisk:   values["RAM disk"],
		RAMDir:    values["RAM dir"],
		ISOs:      values["ISO dir"],
		Templates: values["Templates"],
	})
	if d != nil {
		g.distro = d
	}
	g.reload()
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
