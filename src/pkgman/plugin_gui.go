//go:build !tui_only

package pkgman

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"image/color"

	"codeberg.org/oSoWoSo/SysMan/src/api"
	"codeberg.org/oSoWoSo/SysMan/src/common"
	serman "codeberg.org/oSoWoSo/SysMan/src/serman"
	"fyne.io/fyne/v2"
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
	a := common.NewApp(t("app.window"))
	win := a.NewWindow(t("app.window"))
	common.SetWindowIcon(win)
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

// ── Queue ──────────────────────────────────────────────────────────────

// QueueEntry represents a single package operation in the batch queue.
type QueueEntry struct {
	Name   string
	Action string // "install" or "remove"
}

// ── GUI state ──────────────────────────────────────────────────────────

type pkgGuiApp struct {
	win      fyne.Window
	backend  PkgBackend
	packages []Package
	search   string
	filter   pkgFilter
	selected int
	queue    []QueueEntry

	btnAppImage *common.HoverableButton
	appImageOn  bool

	pkgList       *widget.List
	detailName    *widget.Label
	detailVer     *widget.Label
	detailDesc    *widget.Label
	detailHome    *widget.Hyperlink
	detailInstall *widget.Label
	detailRepo    *widget.Label
	detailSize    *widget.Label
	outputRich    *widget.RichText
	outputScroll  *container.Scroll
	btnCopy       *common.HoverableButton
	highlighter   *api.Highlighter
	statusBar     *common.StatusBar
	queueLabel    *widget.Label
	queueContent  *fyne.Container
	queueScroll   *container.Scroll
	lastOutput    string
}

func (g *pkgGuiApp) doUI(fn func()) {
	fyne.Do(fn)
}

func (g *pkgGuiApp) showAbout() {
	title := canvas.NewText(t("app.title"), color.NRGBA{R: 0x00, G: 0xb8, B: 0xd4, A: 0xff})
	title.TextSize = 26
	title.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	subtitle := canvas.NewText(t("app.subtitle"), color.NRGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff})
	subtitle.TextSize = 12
	infoForm := widget.NewForm(
		widget.NewFormItem(t("about.version"), widget.NewLabel(serman.Version)),
		widget.NewFormItem(t("about.author"), widget.NewLabel(serman.AppAuthor)),
		widget.NewFormItem(t("about.license"), widget.NewLabel(serman.AppLicense)),
	)
	repoURL, _ := url.Parse(serman.AppURL)
	link := widget.NewHyperlink(serman.AppURL, repoURL)
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
	g.backend.Reload()
	g.selected = -1
	g.clearDetail()
	g.statusBar.SetText(t("pkg.loading"))
	go func() {
		pkgs := g.backend.List()
		g.doUI(func() {
			g.packages = pkgs
			g.pkgList.Refresh()
			g.statusBar.SetText(fmt.Sprintf(t("pkg.count"), len(g.packages)))
		})
	}()
}

// reloadAndReselect reloads the package list and re-selects the package named
// prevName (if still present), refreshing buttons to reflect the new installed state.
func (g *pkgGuiApp) reloadAndReselect(prevName string) {
	g.backend.Reload()
	g.selected = -1
	g.clearDetail()
	g.statusBar.SetText(t("pkg.loading"))
	go func() {
		pkgs := g.backend.List()
		g.doUI(func() {
			g.packages = pkgs
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
		})
	}()
}

func (g *pkgGuiApp) selectedName() string {
	list := g.filtered()
	if g.selected < 0 || g.selected >= len(list) {
		return ""
	}
	return list[g.selected].Name
}

func (g *pkgGuiApp) showDetail(name string) {
	g.detailName.SetText(name)
	g.detailVer.SetText("…")
	g.detailDesc.SetText("…")
	g.detailHome.Hide()
	g.detailRepo.SetText("…")
	g.detailSize.SetText("…")

	list := g.filtered()
	if g.selected >= 0 && g.selected < len(list) {
		pkg := list[g.selected]
		if pkg.Installed {
			g.detailInstall.SetText(t("pkg.installed"))
		} else {
			g.detailInstall.SetText(t("pkg.not_installed"))
		}
	}

	go func() {
		d := g.backend.Detail(name)
		g.doUI(func() {
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
			if d.Repository != "" {
				g.detailRepo.SetText(d.Repository)
			} else {
				g.detailRepo.SetText("—")
			}
			size := ""
			if d.InstalledSize != "" {
				size = d.InstalledSize
			} else if d.FilenameSize != "" {
				size = d.FilenameSize
			}
			if size != "" {
				g.detailSize.SetText(size)
			} else {
				g.detailSize.SetText("—")
			}
		})
	}()
}

func (g *pkgGuiApp) clearDetail() {
	g.detailName.SetText("—")
	g.detailVer.SetText("—")
	g.detailDesc.SetText("—")
	g.detailInstall.SetText("—")
	g.detailHome.Hide()
	g.detailRepo.SetText("—")
	g.detailSize.SetText("—")
}

// streamWriter is an io.Writer that appends each line to the outputRich widget
// with syntax highlighting and scrolls to the bottom after each update.
type streamWriter struct {
	app *pkgGuiApp
	buf strings.Builder
}

func (sw *streamWriter) Write(p []byte) (int, error) {
	sw.buf.Write(p)
	sw.app.doUI(func() {
		sw.app.setOutput(sw.buf.String())
	})
	return len(p), nil
}

// setOutput renders text into outputRich with highlighting and scrolls to bottom.
func (g *pkgGuiApp) setOutput(text string) {
	g.lastOutput = text
	var segs []widget.RichTextSegment
	if common.HasAnsiCodes(text) {
		segs = common.AnsiToRichSegments(text)
	} else {
		segs = g.highlighter.RichSegments(text)
	}
	g.outputRich.Segments = segs
	g.outputRich.Refresh()
	g.outputScroll.ScrollToBottom()
}

func (g *pkgGuiApp) runOp(label string, fn func(w io.Writer) (string, error)) {
	prevName := g.selectedName()
	g.statusBar.SetText(fmt.Sprintf("Running: %s…", label))
	g.outputRich.Segments = nil
	g.outputRich.Refresh()
	go func() {
		sw := &streamWriter{app: g}
		out, err := fn(sw)
		g.doUI(func() {
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
		})
	}()
}

// ── Queue ──────────────────────────────────────────────────────────────

func (g *pkgGuiApp) toggleQueue() {
	name := g.selectedName()
	if name == "" {
		return
	}
	for i, e := range g.queue {
		if e.Name == name {
			g.queue = append(g.queue[:i], g.queue[i+1:]...)
			g.statusBar.SetText(fmt.Sprintf("Removed %s from queue", name))
			g.refreshQueue()
			return
		}
	}
	action := "install"
	if g.selected >= 0 && g.selected < len(g.packages) && g.packages[g.selected].Installed {
		action = "remove"
	}
	g.queue = append(g.queue, QueueEntry{Name: name, Action: action})
	g.statusBar.SetText(fmt.Sprintf("Added %s to queue (%s)", name, action))
	g.refreshQueue()
}

func (g *pkgGuiApp) refreshQueue() {
	n := len(g.queue)
	if n == 0 {
		g.queueLabel.SetText("")
		g.queueContent.Objects = nil
		g.queueContent.Refresh()
	} else {
		g.queueLabel.SetText(fmt.Sprintf(t("pkg.queue_count"), n))
		var lines []string
		for _, e := range g.queue {
			symbol := "[+]"
			if e.Action == "remove" {
				symbol = "[-]"
			}
			lines = append(lines, fmt.Sprintf("%s %s", symbol, e.Name))
		}
		lbl := widget.NewLabel(strings.Join(lines, "\n"))
		lbl.TextStyle = fyne.TextStyle{Monospace: true}
		lbl.Wrapping = fyne.TextWrapOff
		g.queueContent.Objects = []fyne.CanvasObject{lbl}
		g.queueContent.Refresh()
	}
}

func (g *pkgGuiApp) clearQueue() {
	g.queue = nil
	g.refreshQueue()
}

func (g *pkgGuiApp) applyQueue() {
	if len(g.queue) == 0 {
		return
	}
	var installs, removes []string
	for _, e := range g.queue {
		switch e.Action {
		case "install":
			installs = append(installs, e.Name)
		case "remove":
			removes = append(removes, e.Name)
		}
	}
	g.queue = nil
	g.refreshQueue()

	prevName := g.selectedName()
	g.statusBar.SetText(fmt.Sprintf(t("pkg.queue_running"), len(installs)+len(removes)))
	g.outputRich.Segments = nil
	g.outputRich.Refresh()

	go func() {
		var allOutput strings.Builder
		var firstErr error
		sw := &streamWriter{app: g}
		if len(installs) > 0 {
			out, err := g.backend.Install(installs, sw)
			if out != "" {
				allOutput.WriteString(out + "\n")
			}
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}
		if len(removes) > 0 {
			out, err := g.backend.Remove(removes, sw)
			if out != "" {
				allOutput.WriteString(out + "\n")
			}
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}
		g.doUI(func() {
			if allOutput.Len() > 0 {
				g.setOutput(strings.TrimSpace(allOutput.String()))
			}
			if firstErr != nil {
				g.statusBar.SetText(fmt.Sprintf("✗ queue failed: %s", firstErr.Error()))
			} else {
				g.statusBar.SetText(t("pkg.queue_ok"))
			}
			g.reloadAndReselect(prevName)
		})
	}()
}

func (g *pkgGuiApp) buildContent(showHeader bool) fyne.CanvasObject {
	g.filter = FilterAll

	// ── Status bar (must be created before any HoverableButton) ──────
	if g.statusBar == nil {
		g.statusBar = common.NewStatusBar()
	}
	g.statusBar.TextStyle = fyne.TextStyle{Italic: true, Monospace: true}

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
	var btnFilterAll, btnFilterInstalled, btnFilterAvailable *common.HoverableButton
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
	btnFilterAll = common.NewHoverableButtonText(t("filter.all"), t("tooltip.pkgman.filter_all"), g.statusBar, func() { applyFilter(FilterAll) })
	btnFilterInstalled = common.NewHoverableButtonText(t("filter.installed"), t("tooltip.pkgman.filter_installed"), g.statusBar, func() { applyFilter(FilterInstalled) })
	btnFilterAvailable = common.NewHoverableButtonText(t("filter.available"), t("tooltip.pkgman.filter_available"), g.statusBar, func() { applyFilter(FilterAvailable) })

	// ── AppImage toggle button ───────────────────────────────────────
	g.btnAppImage = common.NewHoverableButtonText(t("filter.appimage"), t("tooltip.pkgman.backend"), g.statusBar, func() {
		g.appImageOn = !g.appImageOn
		if g.appImageOn {
			g.backend = NewAppImageBackend()
			g.btnAppImage.Importance = widget.HighImportance
		} else {
			g.backend = NewXbpsBackend()
			g.btnAppImage.Importance = widget.MediumImportance
		}
		g.btnAppImage.Refresh()
		g.reload()
	})
	g.btnAppImage.Importance = widget.MediumImportance

	filterRow := container.NewHBox(btnFilterAll, btnFilterInstalled, btnFilterAvailable, widget.NewSeparator(), g.btnAppImage)

	// ── Package list ──────────────────────────────────────────────────
	installedColor := color.RGBA{R: 0x44, G: 0xDD, B: 0x77, A: 0xFF} // grn
	g.pkgList = widget.NewList(
		func() int { return len(g.filtered()) },
		func() fyne.CanvasObject {
			star := canvas.NewText("*", installedColor)
			star.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}
			name := canvas.NewText("package-placeholder", theme.Color(theme.ColorNameForeground))
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

	g.btnCopy = common.NewHoverableButton(t("btn.copy"), theme.ContentCopyIcon(), t("tooltip.common.copy"), g.statusBar, func() {
		fyne.CurrentApp().Clipboard().SetContent(g.lastOutput)
		g.statusBar.SetText(t("status.copied"))
	})

	// ── Detail panel ──────────────────────────────────────────────────
	g.detailName = widget.NewLabel("—")
	g.detailName.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	g.detailName.Selectable = true
	g.detailVer = widget.NewLabel("—")
	g.detailVer.Selectable = true
	g.detailDesc = widget.NewLabel("—")
	g.detailDesc.Wrapping = fyne.TextWrapBreak
	g.detailDesc.Selectable = true
	g.detailInstall = widget.NewLabel("—")
	g.detailInstall.Selectable = true
	g.detailHome = widget.NewHyperlink("", nil)
	g.detailHome.Hide()
	g.detailRepo = widget.NewLabel("—")
	g.detailRepo.Selectable = true
	g.detailSize = widget.NewLabel("—")
	g.detailSize.Selectable = true

	detailForm := widget.NewForm(
		widget.NewFormItem(t("detail.name"), g.detailName),
		widget.NewFormItem(t("detail.status"), g.detailInstall),
		widget.NewFormItem(t("detail.version"), g.detailVer),
		widget.NewFormItem(t("detail.desc"), g.detailDesc),
		widget.NewFormItem(t("detail.homepage"), g.detailHome),
		widget.NewFormItem(t("detail.repository"), g.detailRepo),
		widget.NewFormItem(t("detail.size"), g.detailSize),
	)

	// ── Action buttons ────────────────────────────────────────────────
	btnToggle := common.NewHoverableButtonText(t("btn.toggle_queue"), t("tooltip.pkgman.toggle_queue"), g.statusBar, func() {
		g.toggleQueue()
	})
	btnToggle.Importance = widget.HighImportance

	btnApply := common.NewHoverableButton(t("btn.queue_apply"), theme.ConfirmIcon(), t("tooltip.pkgman.queue_apply"), g.statusBar, func() {
		g.applyQueue()
	})
	btnApply.Importance = widget.HighImportance

	btnClear := common.NewHoverableButton(t("btn.queue_clear"), theme.CancelIcon(), t("tooltip.pkgman.queue_clear"), g.statusBar, func() {
		g.clearQueue()
	})
	btnClear.Importance = widget.LowImportance

	btnUpdate := common.NewHoverableButton(t("btn.update_all"), theme.UploadIcon(), t("tooltip.pkgman.update_all"), g.statusBar, func() {
		g.runOp("update", func(w io.Writer) (string, error) { return g.backend.Update(w) })
	})
	btnUpdate.Importance = widget.MediumImportance

	btnReload := common.NewHoverableButton("", theme.ViewRefreshIcon(), t("tooltip.pkgman.reload"), g.statusBar, func() {
		g.reload()
	})
	btnReload.Importance = widget.LowImportance

	actionRow := container.NewHBox(btnToggle, btnApply, btnClear, layout.NewSpacer(), btnUpdate)

	// ── Queue panel ──────────────────────────────────────────────────
	g.queueLabel = widget.NewLabel("")
	g.queueLabel.TextStyle = fyne.TextStyle{Bold: true}

	g.queueContent = container.NewVBox()
	g.queueScroll = container.NewVScroll(g.queueContent)

	btnAbout := common.NewHoverableButton("", theme.InfoIcon(), t("tooltip.pkgman.about"), g.statusBar, func() { g.showAbout() })
	btnAbout.Importance = widget.LowImportance
	statusBar := container.NewHBox(btnAbout, btnReload, layout.NewSpacer(), g.statusBar)

	rightTop := container.NewVBox(detailForm, widget.NewSeparator(), actionRow, widget.NewSeparator())
	outputToolbar := container.NewHBox(g.btnCopy, layout.NewSpacer())
	queueArea := container.NewBorder(g.queueLabel, nil, nil, nil, g.queueScroll)
	outputArea := container.NewBorder(outputToolbar, nil, nil, nil, g.outputScroll)
	queueOutputSplit := container.NewVSplit(queueArea, outputArea)
	queueOutputSplit.SetOffset(0.15)
	rightPanel := container.NewBorder(rightTop, nil, nil, nil, queueOutputSplit)

	split := container.NewHSplit(
		container.NewPadded(leftPanel),
		container.NewPadded(rightPanel),
	)
	split.SetOffset(0.38)

	highlightFilter(FilterAll)
	g.clearDetail()

	// Load packages asynchronously.
	go func() {
		packages := g.backend.List()
		fyne.Do(func() {
			g.packages = packages
			g.selected = -1
			g.pkgList.Refresh()
			g.clearDetail()
			g.statusBar.SetText(fmt.Sprintf(t("pkg.count"), len(g.packages)))
		})
	}()

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
