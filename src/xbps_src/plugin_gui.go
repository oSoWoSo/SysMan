//go:build !tui_only

package xbpssrc

import "fyne.io/fyne/v2"

// Content builds the Fyne widget tree for embedding in a parent application.
// win is the parent window (used by the Fyne framework for sizing/dialogs).
// Implements api.PluginIF.
func (p *Plugin) Content(win fyne.Window) fyne.CanvasObject {
	g := &xbpsGuiApp{
		win:      win,
		distDir:  p.distDir,
		cfg:      LoadConfig(),
		selected: -1,
	}
	g.templates = LoadTemplates(p.distDir)
	return g.buildContent()
}
