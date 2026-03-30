//go:build !tui_only

package xbpspkg

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"image/color"

	"codeberg.org/oSoWoSo/SysMan/api"
	svman "codeberg.org/oSoWoSo/SysMan/plugin"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Content builds the Fyne widget tree for embedding in a parent application.
// Implements api.PluginIF.
func (p *Plugin) Content(win fyne.Window) fyne.CanvasObject {
	g := &pkgGuiApp{win: win, backend: p.backend}
	return g.buildContent(false)
}

// RunGUI runs the package manager as a standalone Fyne application with a header.
func RunGUI() {
	a := app.New()
	win := a.NewWindow(t("app.window"))
	g := &pkgGuiApp{win: win, backend: NewXbpsBackend()}
	win.SetContent(g.buildContent(true))
	win.Resize(fyne.NewSize(900, 620))
	win.SetMaster()
	win.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		if e.Name == fyne.KeyEscape {
			a.Quit()
		}
	})
	win.ShowAndRun()
}

// ── GUI state ──────────────────────────────────────────────────────────

type pkgGuiApp struct {
	win      fyne.Window
	backend  PkgBackend
	packages []Package
	search   string
	filter   pkgFilter
	selected int

	pkgList       *widget.List
	detailName    *widget.Label
	detailVer     *widget.Label
	detailDesc    *widget.Label
	detailHome    *widget.Hyperlink
	detailInstall *widget.Label
	outputRich    *widget.RichText
	outputScroll  *container.Scroll
	highlighter   *api.Highlighter
	statusBar     *widget.Label
	btnInstall    *widget.Button
	btnRemove     *widget.Button
}

func (g *pkgGuiApp) showAbout() {
	title := canvas.NewText(t("app.title"), color.NRGBA{R: 0x00, G: 0xb8, B: 0xd4, A: 0xff})
	title.TextSize = 26
	title.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	subtitle := canvas.NewText(t("app.subtitle"), color.NRGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff})
	subtitle.TextSize = 12
	infoForm := widget.NewForm(
		widget.NewFormItem(t("about.version"), widget.NewLabel(svman.Version)),
		widget.NewFormItem(t("about.author"), widget.NewLabel(svman.AppAuthor)),
		widget.NewFormItem(t("about.license"), widget.NewLabel(svman.AppLicense)),
	)
	repoURL, _ := url.Parse(svman.AppURL)
	link := widget.NewHyperlink(svman.AppURL, repoURL)
	descLabel := widget.NewLabel(t("about.description"))
	descLabel.Wrapping = fyne.TextWrapWord
	content := container.NewVBox(
		container.NewCenter(title),
		container.NewCenter(subtitle),
		widget.NewSeparator(),
		infoForm,
		container.NewCenter(link),
		widget.NewSeparator(),
		descLabel,
	)
	d := dialog.NewCustom(t("btn.about"), t("btn.close"), content, g.win)
	d.Show()
}

func (g *pkgGuiApp) filtered() []Package {
	return Filter(g.packages, g.filter, g.search,
		func(p Package) bool { return p.Installed },
		func(p Package, q string) bool {
			return strings.Contains(strings.ToLower(p.Name), q) ||
				strings.Contains(strings.ToLower(p.ShortDesc), q)
		},
	)
}

func (g *pkgGuiApp) reload() {
	g.packages = g.backend.List()
	g.selected = -1
	g.pkgList.Refresh()
	g.clearDetail()
	g.statusBar.SetText(fmt.Sprintf(t("pkg.count"), len(g.packages)))
}

// reloadAndReselect reloads the package list and re-selects the package named
// prevName (if still present), refreshing buttons to reflect the new installed state.
func (g *pkgGuiApp) reloadAndReselect(prevName string) {
	g.packages = g.backend.List()
	g.selected = -1
	g.pkgList.Refresh()
	if prevName != "" {
		list := g.filtered()
		for i, pkg := range list {
			if pkg.Name == prevName {
				g.selected = i
				g.pkgList.Select(i)
				g.showDetail(prevName)
				break
			}
		}
	}
	if g.selected == -1 {
		g.clearDetail()
	}
	g.statusBar.SetText(fmt.Sprintf(t("pkg.count"), len(g.packages)))
}

func (g *pkgGuiApp) selectedName() string {
	list := g.filtered()
	if g.selected < 0 || g.selected >= len(list) {
		return ""
	}
	return list[g.selected].Name
}

func (g *pkgGuiApp) showDetail(name string) {
	// Set name immediately; fetch details asynchronously to avoid blocking the UI.
	g.detailName.SetText(name)
	g.detailVer.SetText("…")
	g.detailDesc.SetText("…")
	g.detailHome.Hide()

	// Enable/disable buttons based on installed status already known from the list.
	list := g.filtered()
	if g.selected >= 0 && g.selected < len(list) {
		pkg := list[g.selected]
		if pkg.Installed {
			g.detailInstall.SetText(t("pkg.installed"))
			g.btnInstall.Disable()
			g.btnRemove.Enable()
		} else {
			g.detailInstall.SetText(t("pkg.not_installed"))
			g.btnInstall.Enable()
			g.btnRemove.Disable()
		}
	}

	go func() {
		d := g.backend.Detail(name)
		if d.Version != "" {
			g.detailVer.SetText(d.Version)
		} else {
			g.detailVer.SetText("—")
		}
		if d.ShortDesc != "" {
			g.detailDesc.SetText(d.ShortDesc)
		} else {
			g.detailDesc.SetText("—")
		}
		if d.Homepage != "" {
			if u, err := url.Parse(d.Homepage); err == nil {
				g.detailHome.SetText(d.Homepage)
				g.detailHome.SetURL(u)
				g.detailHome.Show()
			}
		}
	}()
}

func (g *pkgGuiApp) clearDetail() {
	g.detailName.SetText("—")
	g.detailVer.SetText("—")
	g.detailDesc.SetText("—")
	g.detailInstall.SetText("—")
	g.detailHome.Hide()
	g.btnInstall.Disable()
	g.btnRemove.Disable()
}

// streamWriter is an io.Writer that appends each line to the outputRich widget
// with syntax highlighting and scrolls to the bottom after each update.
type streamWriter struct {
	app *pkgGuiApp
	buf strings.Builder
}

func (sw *streamWriter) Write(p []byte) (int, error) {
	sw.buf.Write(p)
	sw.app.setOutput(sw.buf.String())
	return len(p), nil
}

// setOutput renders text into outputRich with highlighting and scrolls to bottom.
func (g *pkgGuiApp) setOutput(text string) {
	segs := g.highlighter.RichSegments(text)
	g.outputRich.Segments = segs
	g.outputRich.Refresh()
	g.outputScroll.ScrollToBottom()
}

func (g *pkgGuiApp) runOp(label string, fn func(w io.Writer) (string, error)) {
	// Remember selected package name so we can re-select after reload.
	prevName := g.selectedName()
	g.statusBar.SetText(fmt.Sprintf("Running: %s…", label))
	g.outputRich.Segments = nil
	g.outputRich.Refresh()
	go func() {
		sw := &streamWriter{app: g}
		out, err := fn(sw)
		// If fn returned output that wasn't streamed (e.g. TTY mode), show it.
		if out != "" {
			g.setOutput(out)
		} else if err != nil && sw.buf.Len() == 0 {
			g.setOutput(err.Error())
		}
		if err != nil {
			g.statusBar.SetText(fmt.Sprintf("✗ %s failed: %s", label, err.Error()))
		} else {
			g.statusBar.SetText(fmt.Sprintf("✓ %s OK", label))
		}
		g.reloadAndReselect(prevName)
	}()
}

func (g *pkgGuiApp) buildContent(showHeader bool) fyne.CanvasObject {
	g.filter = FilterAll

	// ── Search ────────────────────────────────────────────────────────
	search := widget.NewEntry()
	search.SetPlaceHolder(t("search.placeholder"))
	search.OnChanged = func(q string) {
		g.search = q
		g.selected = -1
		g.pkgList.Refresh()
		g.clearDetail()
	}

	// ── Filter buttons ────────────────────────────────────────────────
	var btnFilterAll, btnFilterInstalled, btnFilterAvailable *widget.Button
	highlightFilter := func(f pkgFilter) {
		btnFilterAll.Importance = widget.MediumImportance
		btnFilterInstalled.Importance = widget.MediumImportance
		btnFilterAvailable.Importance = widget.MediumImportance
		switch f {
		case FilterInstalled:
			btnFilterInstalled.Importance = widget.HighImportance
		case FilterAvailable:
			btnFilterAvailable.Importance = widget.HighImportance
		default:
			btnFilterAll.Importance = widget.HighImportance
		}
		btnFilterAll.Refresh()
		btnFilterInstalled.Refresh()
		btnFilterAvailable.Refresh()
	}
	applyFilter := func(f pkgFilter) {
		g.filter = f
		g.selected = -1
		g.pkgList.Refresh()
		g.clearDetail()
		highlightFilter(f)
	}
	btnFilterAll = widget.NewButton(t("filter.all"), func() { applyFilter(FilterAll) })
	btnFilterInstalled = widget.NewButton(t("filter.installed"), func() { applyFilter(FilterInstalled) })
	btnFilterAvailable = widget.NewButton(t("filter.available"), func() { applyFilter(FilterAvailable) })
	filterRow := container.NewHBox(btnFilterAll, btnFilterInstalled, btnFilterAvailable)

	// ── Package list ──────────────────────────────────────────────────
	installedColor := color.RGBA{R: 0x44, G: 0xDD, B: 0x77, A: 0xFF} // grn
	g.pkgList = widget.NewList(
		func() int { return len(g.filtered()) },
		func() fyne.CanvasObject {
			star := canvas.NewText("*", installedColor)
			star.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}
			name := canvas.NewText("package-placeholder", theme.ForegroundColor())
			name.TextStyle = fyne.TextStyle{Monospace: true}
			return container.NewHBox(star, name)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			list := g.filtered()
			if id >= len(list) {
				return
			}
			pkg := list[id]
			c := obj.(*fyne.Container)
			star := c.Objects[0].(*canvas.Text)
			name := c.Objects[1].(*canvas.Text)
			if pkg.Installed {
				star.Text = "*"
				star.Color = installedColor
			} else {
				star.Text = " "
				star.Color = color.Transparent
			}
			name.Text = pkg.Name
			star.Refresh()
			name.Refresh()
		},
	)
	g.pkgList.OnSelected = func(id widget.ListItemID) {
		g.selected = id
		list := g.filtered()
		if id < len(list) {
			g.showDetail(list[id].Name)
		}
	}

	leftPanel := container.NewBorder(
		container.NewVBox(search, filterRow, widget.NewSeparator()),
		nil, nil, nil,
		g.pkgList,
	)

	// ── Output area (must be init before clearDetail) ─────────────────
	g.highlighter = api.NewHighlighter()
	g.outputRich = widget.NewRichText()
	g.outputRich.Wrapping = fyne.TextWrapBreak
	g.outputScroll = container.NewScroll(g.outputRich)
	g.outputScroll.SetMinSize(fyne.NewSize(0, 200))

	// ── Detail panel ──────────────────────────────────────────────────
	g.detailName = widget.NewLabel("—")
	g.detailName.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	g.detailVer = widget.NewLabel("—")
	g.detailDesc = widget.NewLabel("—")
	g.detailDesc.Wrapping = fyne.TextWrapBreak
	g.detailInstall = widget.NewLabel("—")
	g.detailHome = widget.NewHyperlink("", nil)
	g.detailHome.Hide()

	detailForm := widget.NewForm(
		widget.NewFormItem(t("detail.name"), g.detailName),
		widget.NewFormItem(t("detail.status"), g.detailInstall),
		widget.NewFormItem(t("detail.version"), g.detailVer),
		widget.NewFormItem(t("detail.desc"), g.detailDesc),
		widget.NewFormItem(t("detail.homepage"), g.detailHome),
	)

	// ── Action buttons ────────────────────────────────────────────────
	g.btnInstall = widget.NewButtonWithIcon(t("btn.install"), theme.DownloadIcon(), func() {
		if name := g.selectedName(); name != "" {
			g.runOp("install "+name, func(w io.Writer) (string, error) { return g.backend.Install([]string{name}, w) })
		}
	})
	g.btnInstall.Importance = widget.HighImportance

	g.btnRemove = widget.NewButtonWithIcon(t("btn.remove"), theme.DeleteIcon(), func() {
		if name := g.selectedName(); name != "" {
			g.runOp("remove "+name, func(w io.Writer) (string, error) { return g.backend.Remove([]string{name}, w) })
		}
	})
	g.btnRemove.Importance = widget.DangerImportance

	btnUpdate := widget.NewButtonWithIcon(t("btn.update_all"), theme.UploadIcon(), func() {
		g.runOp("update", func(w io.Writer) (string, error) { return g.backend.Update(w) })
	})
	btnUpdate.Importance = widget.MediumImportance

	btnReload := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		g.reload()
	})
	btnReload.Importance = widget.LowImportance

	actionRow := container.NewHBox(g.btnInstall, g.btnRemove, layout.NewSpacer(), btnUpdate)

	// ── Status bar ────────────────────────────────────────────────────
	g.statusBar = widget.NewLabel(t("pkg.loading"))
	g.statusBar.TextStyle = fyne.TextStyle{Italic: true, Monospace: true}

	btnAbout := widget.NewButtonWithIcon("", theme.InfoIcon(), func() { g.showAbout() })
	btnAbout.Importance = widget.LowImportance
	statusBar := container.NewHBox(btnAbout, btnReload, g.statusBar, layout.NewSpacer())

	rightTop := container.NewVBox(detailForm, widget.NewSeparator(), actionRow, widget.NewSeparator())
	rightPanel := container.NewBorder(rightTop, nil, nil, nil, g.outputScroll)

	split := container.NewHSplit(
		container.NewPadded(leftPanel),
		container.NewPadded(rightPanel),
	)
	split.SetOffset(0.38)

	highlightFilter(FilterAll) // highlight "All" button (detail widgets now initialized)
	g.clearDetail()

	// Load packages asynchronously.
	go g.reload()

	var header fyne.CanvasObject
	if showHeader {
		title := widget.NewLabel(t("app.window"))
		title.TextStyle = fyne.TextStyle{Bold: true}
		header = container.NewVBox(container.NewPadded(title), widget.NewSeparator())
	}

	return container.NewBorder(
		header,
		container.NewVBox(widget.NewSeparator(), container.NewPadded(statusBar)),
		nil, nil,
		split,
	)
}
