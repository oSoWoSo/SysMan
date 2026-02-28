//go:build !tui_only

package plugin

import "fyne.io/fyne/v2"

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

// ShowAbout displays the About dialog attached to win.
func (p *Plugin) ShowAbout(win fyne.Window) {
	g := &guiApp{win: win, serviceDir: p.serviceDir, serviceDestDir: p.serviceDestDir}
	g.showAbout()
}
