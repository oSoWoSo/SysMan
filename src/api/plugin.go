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
	"codeberg.org/oSoWoSo/SysMan/src/common"
	"fyne.io/fyne/v2"
	tea "github.com/charmbracelet/bubbletea"
)

// PluginIF is the contract between the system manager and each plugin.
// Plugins are usable both as standalone binaries (via Run* helpers) and
// as embedded components inside the system manager (via Content / Model).
type PluginIF interface {
	// Name returns the human-readable plugin name shown in tabs / headers.
	Name() string

	// Content builds the Fyne widget tree for embedding as a tab or panel.
	// win is the parent window used for dialogs.
	Content(win fyne.Window) fyne.CanvasObject

	// Model returns an initialized Bubbletea tea.Model for TUI embedding.
	Model() tea.Model

	// SetStatusBar sets a shared status bar for tooltips and messages.
	// If not called, each plugin creates its own status bar.
	// This allows the system manager to provide a unified status bar across all plugins.
	SetStatusBar(statusBar *common.StatusBar)
}
