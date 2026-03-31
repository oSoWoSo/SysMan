// Package serman provides a runit service manager plugin.
//
// Standalone use (GUI or TUI):
//
//	plugin.RunGUI(serviceDir, serviceDestDir)
//	plugin.RunTUI(serviceDir, serviceDestDir)
//
// Embedded use in a parent Fyne application:
//
//	p := plugin.New(serviceDir, serviceDestDir)
//	content := p.Content(win)          // fyne.CanvasObject — place in any container
//
// Embedded use in a parent Bubbletea application:
//
//	p := plugin.New(serviceDir, serviceDestDir)
//	model := p.Model()                 // tea.Model — wrap in your own tea.Program
//
// Custom backend (openrc, s6, systemd, …):
//
//	type MyBackend struct{}
//	func (b *MyBackend) List() []plugin.Service { … }
//	// … implement remaining Backend methods …
//	p := plugin.NewWithBackend(&MyBackend{})
package serman

import (
	"codeberg.org/oSoWoSo/SysMan/src/common"
	tea "github.com/charmbracelet/bubbletea"
)

// Plugin is the embeddable svman component.
// Construct one with New() or NewWithBackend(); then call Content() or Model().
type Plugin struct {
	backend   Backend
	statusBar *common.StatusBar
}

// Name returns the plugin display name used in system manager tabs.
// Implements api.PluginIF.
func (p *Plugin) Name() string { return t("tab.name") }

// New creates a Plugin using the default runit backend.
// Use DefaultServiceDir and DefaultServiceDestDir for the standard paths.
func New(serviceDir, serviceDestDir string) *Plugin {
	return NewRunit(serviceDir, serviceDestDir)
}

// NewRunit creates a Plugin backed by runit using the given service directories.
// Equivalent to NewWithBackend(NewRunitBackend(serviceDir, serviceDestDir)).
func NewRunit(serviceDir, serviceDestDir string) *Plugin {
	return &Plugin{backend: NewRunitBackend(serviceDir, serviceDestDir)}
}

// NewWithBackend creates a Plugin using a custom Backend implementation.
// Use this to add support for openrc, s6, systemd, or any other service manager.
func NewWithBackend(b Backend) *Plugin { return &Plugin{backend: b} }

// SetStatusBar sets a shared status bar for tooltips and messages.
// Implements api.PluginIF.
func (p *Plugin) SetStatusBar(statusBar *common.StatusBar) {
	p.statusBar = statusBar
}

// Model returns an initialized Bubbletea tea.Model for TUI embedding.
// Wrap it in your own tea.Program to control program options.
//
// InitI18n() must be called once before Model() if you are not using RunTUI().
func (p *Plugin) Model() tea.Model {
	return NewTuiModel(p.backend)
}
