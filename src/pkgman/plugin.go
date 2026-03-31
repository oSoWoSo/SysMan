// Package xbpspkg provides an extensible package manager plugin.
//
// The default backend targets Void Linux (xbps). To support another package
// manager implement PkgBackend and use NewWithBackend:
//
//	type MyBackend struct{}
//	func (b *MyBackend) Name() string { return "apt" }
//	// … implement remaining PkgBackend methods …
//	p := xbpspkg.NewWithBackend(&MyBackend{})
//
// Standalone TUI use (xbps default):
//
//	xbpspkg.RunTUI()
//
// Embedded use in a parent Bubbletea application:
//
//	p := xbpspkg.New()
//	model := p.Model()   // tea.Model — wrap in your own tea.Program
package pkgman

import (
	"codeberg.org/oSoWoSo/SysMan/src/common"
	tea "github.com/charmbracelet/bubbletea"
)

// Plugin is the embeddable package-manager component.
type Plugin struct {
	backend   PkgBackend
	statusBar *common.StatusBar
}

// New creates a Plugin using the default xbps backend.
func New() *Plugin { return NewXbps() }

// NewXbps creates a Plugin backed by xbps (Void Linux).
// Equivalent to NewWithBackend(NewXbpsBackend()).
func NewXbps() *Plugin { return &Plugin{backend: NewXbpsBackend()} }

// NewWithBackend creates a Plugin using a custom PkgBackend implementation.
// Use this to add support for apt, dnf, pacman, apk, or any other package manager.
func NewWithBackend(b PkgBackend) *Plugin { return &Plugin{backend: b} }

// Name returns the plugin display name used in system manager tabs.
// Implements api.PluginIF.
func (p *Plugin) Name() string { return t("tab.name") }

// SetStatusBar sets a shared status bar for tooltips and messages.
// Implements api.PluginIF.
func (p *Plugin) SetStatusBar(statusBar *common.StatusBar) {
	p.statusBar = statusBar
}

// Model returns an initialized Bubbletea tea.Model for TUI embedding.
// Implements api.PluginIF.
func (p *Plugin) Model() tea.Model {
	return NewTuiModelWithBackend(p.backend)
}
