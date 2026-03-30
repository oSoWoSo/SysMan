//go:build !tui_only

package xbpspkg

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"codeberg.org/oSoWoSo/SysMan/api"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Content builds the Fyne widget tree for embedding in a parent application.
// Implements api.PluginIF.
func (p *Plugin) Content(_ fyne.Window) fyne.CanvasObject {
	g := &pkgGuiApp{backend: p.backend}
	return g.buildContent(false)
}

// RunGUI runs the package manager as a standalone Fyne application with a header.
func RunGUI() {
	a := app.New()
	win := a.NewWindow("xbps packages")
	g := &pkgGuiApp{backend: NewXbpsBackend()}
	win.SetContent(g.buildContent(true))
	win.Resize(fyne.NewSize(900, 620))
	win.SetMaster()
	win.ShowAndRun()
}

// ── GUI state ──────────────────────────────────────────────────────────

type pkgGuiApp struct {
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

func (g *pkgGuiApp) filtered() []Package {
	q := strings.ToLower(g.search)
	var out []Package
	for _, pkg := range g.packages {
		switch g.filter {
		case filterInstalled:
			if !pkg.Installed {
				continue
			}
		case filterAvailable:
			if pkg.Installed {
				continue
			}
		}
		if q != "" && !strings.Contains(strings.ToLower(pkg.Name), q) &&
			!strings.Contains(strings.ToLower(pkg.ShortDesc), q) {
			continue
		}
		out = append(out, pkg)
	}
	return out
}

func (g *pkgGuiApp) reload() {
	g.packages = g.backend.List()
	g.selected = -1
	g.pkgList.Refresh()
	g.clearDetail()
	g.statusBar.SetText(fmt.Sprintf("%d packages", len(g.packages)))
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
	g.statusBar.SetText(fmt.Sprintf("%d packages", len(g.packages)))
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
			g.detailInstall.SetText("installed")
			g.btnInstall.Disable()
			g.btnRemove.Enable()
		} else {
			g.detailInstall.SetText("not installed")
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
	g.filter = filterAll

	// ── Search ────────────────────────────────────────────────────────
	search := widget.NewEntry()
	search.SetPlaceHolder("Search packages…")
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
		case filterInstalled:
			btnFilterInstalled.Importance = widget.HighImportance
		case filterAvailable:
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
	btnFilterAll = widget.NewButton("All", func() { applyFilter(filterAll) })
	btnFilterInstalled = widget.NewButton("Installed", func() { applyFilter(filterInstalled) })
	btnFilterAvailable = widget.NewButton("Available", func() { applyFilter(filterAvailable) })
	filterRow := container.NewHBox(btnFilterAll, btnFilterInstalled, btnFilterAvailable)

	// ── Package list ──────────────────────────────────────────────────
	g.pkgList = widget.NewList(
		func() int { return len(g.filtered()) },
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("package-placeholder")
			lbl.TextStyle = fyne.TextStyle{Monospace: true}
			return lbl
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			list := g.filtered()
			if id >= len(list) {
				return
			}
			pkg := list[id]
			lbl := obj.(*widget.Label)
			if pkg.Installed {
				lbl.SetText("* " + pkg.Name)
			} else {
				lbl.SetText("  " + pkg.Name)
			}
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
		widget.NewFormItem("Name", g.detailName),
		widget.NewFormItem("Status", g.detailInstall),
		widget.NewFormItem("Version", g.detailVer),
		widget.NewFormItem("Description", g.detailDesc),
		widget.NewFormItem("Homepage", g.detailHome),
	)

	// ── Action buttons ────────────────────────────────────────────────
	g.btnInstall = widget.NewButtonWithIcon("Install", theme.DownloadIcon(), func() {
		if name := g.selectedName(); name != "" {
			g.runOp("install "+name, func(w io.Writer) (string, error) { return g.backend.Install([]string{name}, w) })
		}
	})
	g.btnInstall.Importance = widget.HighImportance

	g.btnRemove = widget.NewButtonWithIcon("Remove", theme.DeleteIcon(), func() {
		if name := g.selectedName(); name != "" {
			g.runOp("remove "+name, func(w io.Writer) (string, error) { return g.backend.Remove([]string{name}, w) })
		}
	})
	g.btnRemove.Importance = widget.DangerImportance

	btnUpdate := widget.NewButtonWithIcon("Update all", theme.UploadIcon(), func() {
		g.runOp("update", func(w io.Writer) (string, error) { return g.backend.Update(w) })
	})
	btnUpdate.Importance = widget.MediumImportance

	btnReload := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		g.reload()
	})
	btnReload.Importance = widget.LowImportance

	actionRow := container.NewHBox(g.btnInstall, g.btnRemove, layout.NewSpacer(), btnUpdate)

	// ── Status bar ────────────────────────────────────────────────────
	g.statusBar = widget.NewLabel("Loading…")
	g.statusBar.TextStyle = fyne.TextStyle{Italic: true, Monospace: true}

	statusBar := container.NewHBox(btnReload, g.statusBar, layout.NewSpacer())

	rightTop := container.NewVBox(detailForm, widget.NewSeparator(), actionRow, widget.NewSeparator())
	rightPanel := container.NewBorder(rightTop, nil, nil, nil, g.outputScroll)

	split := container.NewHSplit(
		container.NewPadded(leftPanel),
		container.NewPadded(rightPanel),
	)
	split.SetOffset(0.38)

	highlightFilter(filterAll) // highlight "All" button (detail widgets now initialized)
	g.clearDetail()

	// Load packages asynchronously.
	go g.reload()

	var header fyne.CanvasObject
	if showHeader {
		title := widget.NewLabel("xbps packages")
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
