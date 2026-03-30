// Package plugin provides the svman runit service manager as an embeddable component.
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
package plugin

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Plugin is the embeddable svman component.
// Construct one with New(); then call Content() or Model() depending on the UI framework.
type Plugin struct {
	serviceDir     string
	serviceDestDir string
}

// Name returns the plugin display name used in system manager tabs.
// Implements api.PluginIF.
func (p *Plugin) Name() string { return "Services" }

// New creates a Plugin for the given service directories.
// Use DefaultServiceDir and DefaultServiceDestDir for the standard runit paths.
func New(serviceDir, serviceDestDir string) *Plugin {
	return &Plugin{
		serviceDir:     serviceDir,
		serviceDestDir: serviceDestDir,
	}
}

// Model returns an initialized Bubbletea tea.Model for TUI embedding.
// Wrap it in your own tea.Program to control program options.
//
// InitI18n() must be called once before Model() if you are not using RunTUI().
func (p *Plugin) Model() tea.Model {
	return NewTuiModel(p.serviceDir, p.serviceDestDir)
}
