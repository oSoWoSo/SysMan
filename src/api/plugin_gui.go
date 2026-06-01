//go:build !tui_only

package api

import (
	"codeberg.org/oSoWoSo/SysMan/src/common"
	"fyne.io/fyne/v2"
)

// PluginGUI extends PluginIF with GUI-specific methods.
// This interface is only available when building without the tui_only tag.
type PluginGUI interface {
	PluginIF

	// Content builds the Fyne widget tree for embedding as a tab or panel.
	Content(win fyne.Window) fyne.CanvasObject

	// SetStatusBar sets a shared status bar for tooltips and messages.
	SetStatusBar(statusBar *common.StatusBar)
}
