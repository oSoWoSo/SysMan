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
	"fyne.io/fyne/v2"
	tea "github.com/charmbracelet/bubbletea"
)

// Plugin is the embeddable svman component.
// Construct one with New(); then call Content() or Model() depending on the UI framework.
type Plugin struct {
	serviceDir     string
	serviceDestDir string
}

// New creates a Plugin for the given service directories.
// Use DefaultServiceDir and DefaultServiceDestDir for the standard runit paths.
func New(serviceDir, serviceDestDir string) *Plugin {
	return &Plugin{
		serviceDir:     serviceDir,
		serviceDestDir: serviceDestDir,
	}
}

// Content builds and returns the Fyne widget tree for embedding as a tab or panel.
// win is the parent window used for dialogs (About, error, confirm).
//
// InitI18n() must be called once before Content() if you are not using RunGUI().
func (p *Plugin) Content(win fyne.Window) fyne.CanvasObject {
	g := &guiApp{
		win:            win,
		serviceDir:     p.serviceDir,
		serviceDestDir: p.serviceDestDir,
		selected:       -1,
	}
	g.services = LoadServices(p.serviceDir, p.serviceDestDir)
	return g.buildContent()
}

// Model returns an initialized Bubbletea tea.Model for TUI embedding.
// Wrap it in your own tea.Program to control program options.
//
// InitI18n() must be called once before Model() if you are not using RunTUI().
func (p *Plugin) Model() tea.Model {
	return NewTuiModel(p.serviceDir, p.serviceDestDir)
}

// ShowAbout displays the About dialog attached to win.
func (p *Plugin) ShowAbout(win fyne.Window) {
	g := &guiApp{win: win, serviceDir: p.serviceDir, serviceDestDir: p.serviceDestDir}
	g.showAbout()
}
