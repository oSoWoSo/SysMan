// Package api defines the shared interface every system manager plugin must implement.
//
// Both static (compiled-in) and dynamic (.so) plugins use this interface.
// A dynamic plugin .so must export:
//
//	func New() api.PluginIF
//
// which the system manager locates via plugin.Lookup("New").
package api

import (
	tea "charm.land/bubbletea/v2"
)

// PluginIF is the contract between the system manager and each plugin.
// Plugins are usable both as standalone binaries (via Run* helpers) and
// as embedded components inside the system manager (via Content / Model).
type PluginIF interface {
	// Name returns the human-readable plugin name shown in tabs / headers.
	Name() string

	// Model returns an initialized Bubbletea tea.Model for TUI embedding.
	Model() tea.Model
}
