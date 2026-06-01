//go:build !tui_only

package infman

import (
	tea "charm.land/bubbletea/v2"
	"codeberg.org/oSoWoSo/SysMan/src/common"
	serman "codeberg.org/oSoWoSo/SysMan/src/serman"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	zone "github.com/lrstanley/bubblezone/v2"
)

// Plugin displays basic system information (hostname, OS, arch, CPUs, Go version).
type Plugin struct {
	statusBar *common.StatusBar
}

// New returns a new sysinfo Plugin.
func New() *Plugin { return &Plugin{} }

// Name returns the plugin display name.
// Implements api.PluginIF.
func (p *Plugin) Name() string { return t("tab.name") }

// SetStatusBar sets a shared status bar for tooltips and messages.
// Implements api.PluginIF.
func (p *Plugin) SetStatusBar(statusBar *common.StatusBar) {
	p.statusBar = statusBar
}

// Model returns a Bubbletea tea.Model showing system information.
// Implements api.PluginIF.
func (p *Plugin) Model() tea.Model {
	zone.NewGlobal()
	return sysInfoModel{id: zone.NewPrefix()}
}

// ── Logo ───────────────────────────────────────────────────────────────

// logoImage tries to load a distro logo PNG from well-known paths.
// Returns nil when no logo is found.
func logoImage() *canvas.Image {
	return common.LogoImage()
}

// ── GUI ────────────────────────────────────────────────────────────────

// buildNativeView constructs a Fyne widget showing key/value pairs
// with the theme primary color for keys and foreground for values.
func buildNativeView(entries []infoEntry) fyne.CanvasObject {
	const keyWidth float32 = 110
	const fontSize float32 = 14

	var rows []fyne.CanvasObject
	for _, e := range entries {
		key := canvas.NewText(e.Key, theme.Color(theme.ColorNamePrimary))
		key.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
		key.TextSize = fontSize

		val := canvas.NewText(e.Value, theme.Color(theme.ColorNameForeground))
		val.TextStyle = fyne.TextStyle{Monospace: true}
		val.TextSize = fontSize

		keyBox := container.New(&fixedWidthLayout{width: keyWidth}, key)
		row := container.NewHBox(keyBox, val)
		rows = append(rows, row)
	}
	return container.NewVBox(rows...)
}

// fixedWidthLayout is a minimal layout that forces its single child to a fixed width.
type fixedWidthLayout struct{ width float32 }

func (l *fixedWidthLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(l.width, 0)
}

func (l *fixedWidthLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(fyne.NewSize(l.width, size.Height))
	}
}

// showAbout displays the About dialog for infoman.
func showAbout(win fyne.Window) {
	common.ShowAbout(common.AboutConfig{
		Win:       win,
		Title:     t("app.title"),
		Subtitle:  t("app.subtitle"),
		Version:   serman.Version,
		Author:    serman.AppAuthor,
		License:   serman.AppLicense,
		URL:       serman.AppURL,
		DialogBtn: t("btn.about"),
		CloseBtn:  t("btn.close"),
	})
}

// Content builds the Fyne widget tree showing system information.
// Implements api.PluginIF.
func (p *Plugin) Content(win fyne.Window) fyne.CanvasObject {
	entries := collectNative()
	infoView := buildNativeView(entries)

	var inner fyne.CanvasObject
	if img := logoImage(); img != nil {
		logoCol := container.NewVBox(layout.NewSpacer(), img, layout.NewSpacer())
		inner = container.NewHBox(logoCol, infoView)
	} else {
		inner = infoView
	}

	scroll := container.NewScroll(inner)

	// Status bar for tooltips
	statusBar := p.statusBar
	if statusBar == nil {
		statusBar = common.NewStatusBar()
	}
	statusBar.TextStyle = fyne.TextStyle{Italic: true, Monospace: true}

	btnAbout := common.NewHoverableButton("", theme.InfoIcon(), t("tooltip.infman.about"), statusBar, func() { showAbout(win) })
	btnAbout.Importance = widget.LowImportance
	statusBarRow := container.NewHBox(btnAbout, layout.NewSpacer(), statusBar)
	statusBarPanel := container.NewVBox(
		widget.NewSeparator(),
		container.NewPadded(statusBarRow),
	)

	return container.NewBorder(nil, statusBarPanel, nil, nil, scroll)
}
