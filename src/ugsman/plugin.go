// Package usergroups provides a Users & Groups management plugin
// for the system manager. It wraps standard shadow-utils commands
// (useradd, userdel, usermod, chpasswd, groupadd, groupdel) via sudo.
package ugsman

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Plugin implements api.PluginIF.
type Plugin struct{}

// New creates a Plugin.
func New() *Plugin { return &Plugin{} }

// Name returns the display name shown in tabs.
func (p *Plugin) Name() string { return t("tab.name") }

// Model returns a Bubbletea tea.Model for TUI embedding.
func (p *Plugin) Model() tea.Model { return NewTuiModel() }

// Content is defined in gui.go (build tag !tui_only).
