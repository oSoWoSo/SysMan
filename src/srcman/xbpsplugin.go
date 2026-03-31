// Package xbpssrc provides an xbps-src template manager as an embeddable component.
//
// Standalone use (GUI or TUI):
//
//	xbpssrc.RunGUI(distDir)
//	xbpssrc.RunTUI(distDir)
//
// Embedded use in a parent Fyne application:
//
//	p := xbpssrc.New(distDir)
//	content := p.Content(win)   // fyne.CanvasObject — place in any container
//
// Embedded use in a parent Bubbletea application:
//
//	p := xbpssrc.New(distDir)
//	model := p.Model()          // tea.Model — wrap in your own tea.Program
package srcman

import tea "github.com/charmbracelet/bubbletea"

// Plugin is the embeddable xbps-src template manager component.
// Construct one with New(); then call Content() or Model() depending on the UI framework.
type Plugin struct {
	distDir string
}

// New creates a Plugin for the given void-packages directory.
// Pass DefaultDistDir to resolve from $XBPS_DISTDIR or ~/void.
func New(distDir string) *Plugin {
	return &Plugin{distDir: distDir}
}

// Name returns the plugin display name used in system manager tabs.
// Implements api.PluginIF.
func (p *Plugin) Name() string { return t("tab.name") }

// Model returns an initialized Bubbletea tea.Model for TUI embedding.
// Implements api.PluginIF.
func (p *Plugin) Model() tea.Model {
	return NewTuiModel(p.distDir)
}
