// Package usergroups provides a Users & Groups management plugin
// for the system manager. It wraps standard shadow-utils commands
// (useradd, userdel, usermod, chpasswd, groupadd, groupdel) via sudo.
package usergroups

import (
	"fyne.io/fyne/v2"
	tea "github.com/charmbracelet/bubbletea"
)

// Plugin implements api.PluginIF.
type Plugin struct{}

// New creates a Plugin.
func New() *Plugin { return &Plugin{} }

// Name returns the display name shown in tabs.
func (p *Plugin) Name() string { return "Users & Groups" }

// Model returns a Bubbletea tea.Model for TUI embedding.
func (p *Plugin) Model() tea.Model { return NewTuiModel() }

// Content builds the Fyne widget tree.
// Defined in gui.go (build tag !tui_only).
// The method signature is declared here so the TUI-only build sees it.
var _ = (*Plugin)(nil) // ensure Plugin is used

// tui_only stub — Content is defined in gui.go for GUI builds.
// For tui_only builds we need a no-op to satisfy the interface.
// The interface is defined in api/plugin.go and requires Content.
// We handle this with build tags: gui.go provides Content for !tui_only.

// Ensure *Plugin satisfies fyne.Window-independent parts at compile time.
var _ interface {
	Name() string
	Model() tea.Model
	Content(fyne.Window) fyne.CanvasObject
} = (*Plugin)(nil)
