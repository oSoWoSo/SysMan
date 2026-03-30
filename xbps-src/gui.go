//go:build !tui_only

package xbpssrc

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"image/color"

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

// ── App state ─────────────────────────────────────────────────────────

type xbpsGuiApp struct {
	win       fyne.Window
	distDir   string
	cfg       SrcmanConfig
	templates []Template
	selected  int
	search    string

	templateList *widget.List
	detailName   *widget.Label
	detailVer    *widget.Label
	detailDesc   *widget.Label
	output       *outputPanel
	statusBar    *widget.Label

	btnBuild   *widget.Button
	btnInstall *widget.Button
	btnClean   *widget.Button
	buildMode  string // "" = default, "-Q" = with tests, "-C" = with confpkg
	btnBack    *widget.Button
	btnFwd     *widget.Button

	// always-visible inline editor
	editorEntry   *focusEntry
	editorPath    string          // path of the file currently loaded
	editorTop     *fyne.Container // toolbar+separators
	editorBtnSave *widget.Button
	editorBtnLint *widget.Button
	editorBtnSum  *widget.Button
	editorBtnBump *widget.Button
	editorTitle   *widget.Label
	outerSplit    *container.Split

	logHistory []string // past command outputs (oldest first)
	logHistIdx int      // index while browsing; -1 = showing logLive
	logLive    string   // current (latest) output, not yet in history

	buildCancel context.CancelFunc // non-nil while build is running
}

func (g *xbpsGuiApp) filtered() []Template {
	return Filter(g.templates, g.search)
}

func (g *xbpsGuiApp) reload() {
	g.templates = LoadTemplates(g.distDir)
	g.templateList.Refresh()
}

func (g *xbpsGuiApp) showDetail(name string) {
	meta := ReadMeta(g.distDir, name)
	g.detailName.SetText(name)
	g.detailVer.SetText(meta.Version)
	g.detailDesc.SetText(meta.Desc)
	g.btnBuild.Enable()
	g.btnInstall.Enable()
	g.btnClean.Enable()
	g.loadEditorFile(name)
}

func (g *xbpsGuiApp) clearDetail() {
	g.detailName.SetText("—")
	g.detailVer.SetText("—")
	g.detailDesc.SetText("—")
	g.btnBuild.Disable()
	g.btnInstall.Disable()
	g.btnClean.Disable()
	g.editorPath = ""
	g.editorEntry.SetText("")
}

func (g *xbpsGuiApp) setStatus(msg string) {
	g.statusBar.SetText(msg)
}

func (g *xbpsGuiApp) showAbout() {
	title := canvas.NewText(t("app.title"), color.NRGBA{R: 0x00, G: 0xb8, B: 0xd4, A: 0xff})
	title.TextSize = 26
	title.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	subtitle := canvas.NewText(t("app.subtitle"), color.NRGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff})
	subtitle.TextSize = 12
	boldLabel := func(s string) *widget.Label {
		l := widget.NewLabel(s)
		l.TextStyle = fyne.TextStyle{Bold: true}
		return l
	}
	repoURL, _ := url.Parse(svman.AppURL)
	link := widget.NewHyperlink(svman.AppURL, repoURL)
	descLabel := widget.NewLabel(t("about.description"))
	descLabel.Wrapping = fyne.TextWrapWord
	content := container.NewVBox(
		container.NewCenter(title),
		container.NewCenter(subtitle),
		widget.NewSeparator(),
		boldLabel(t("about.version")), widget.NewLabel(svman.Version),
		boldLabel(t("about.author")), widget.NewLabel(svman.AppAuthor),
		boldLabel(t("about.license")), widget.NewLabel(svman.AppLicense),
		container.NewCenter(link),
		widget.NewSeparator(),
		descLabel,
	)
	d := dialog.NewCustom(t("btn.about"), t("btn.close"), content, g.win)
	d.Show()
}

// showSelectionMenu shows a popup menu for the selected output text.
func (g *xbpsGuiApp) showSelectionMenu(sel string, pos fyne.Position) {
	sel = strings.TrimSpace(sel)
	if sel == "" {
		return
	}

	items := []*fyne.MenuItem{
		fyne.NewMenuItem(t("menu.add.hostmakedepends"), func() { g.addToDeps(sel, "hostmakedepends") }),
		fyne.NewMenuItem(t("menu.add.makedepends"), func() { g.addToDeps(sel, "makedepends") }),
		fyne.NewMenuItem(t("menu.add.depends"), func() { g.addToDeps(sel, "depends") }),
		fyne.NewMenuItem(t("menu.add.checkdepends"), func() { g.addToDeps(sel, "checkdepends") }),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem(t("menu.xlocate"), func() {
			g.setStatus(fmt.Sprintf("xlocate %s…", sel))
			go func() {
				out, err := RunXlocate(sel)
				if err != nil {
					g.pushLog(fmt.Sprintf("xlocate %s: %s\n%s", sel, err.Error(), out))
					g.setStatus(fmt.Sprintf("✗ xlocate %s", sel))
				} else {
					if out == "" {
						out = "(no results)"
					}
					g.pushLog(out)
					g.setStatus(fmt.Sprintf("✓ xlocate %s", sel))
				}
			}()
		}),
		fyne.NewMenuItem(t("menu.xbpsquery"), func() {
			g.setStatus(fmt.Sprintf("xbps-query -Rs %s…", sel))
			go func() {
				out, err := RunXbpsStream("", nil, "xbps-query", "-Rs", sel)
				if err != nil && out == "" {
					g.pushLog(fmt.Sprintf("xbps-query -Rs %s: %s\n", sel, err.Error()))
					g.setStatus(fmt.Sprintf("✗ xbps-query %s", sel))
				} else {
					if out == "" {
						out = "(no results)"
					}
					g.pushLog(out)
					g.setStatus(fmt.Sprintf("✓ xbps-query -Rs %s", sel))
				}
			}()
		}),
		fyne.NewMenuItemSeparator(),
	}

	items = append(items,
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem(t("menu.websearch"), func() {
			OpenBrowser(g.cfg.SearchEngine + url.QueryEscape(sel))
		}),
	)

	menu := fyne.NewMenu("", items...)
	widget.ShowPopUpMenuAtPosition(menu, g.win.Canvas(), pos)
}

// addToDeps inserts pkgName into the named dependency variable in the editor.
func (g *xbpsGuiApp) addToDeps(pkgName, field string) {
	if g.editorPath == "" {
		g.setStatus(t("status.no_template"))
		return
	}
	text := g.editorEntry.Text
	re := regexp.MustCompile(`(?m)^(` + regexp.QuoteMeta(field) + `=")([^"]*)(")`)
	if re.MatchString(text) {
		text = re.ReplaceAllStringFunc(text, func(m string) string {
			parts := re.FindStringSubmatch(m)
			if len(parts) != 4 {
				return m
			}
			existing := strings.TrimSpace(parts[2])
			if existing == "" {
				return parts[1] + pkgName + parts[3]
			}
			return parts[1] + existing + " " + pkgName + parts[3]
		})
	} else {
		depFields := []string{"hostmakedepends=", "makedepends=", "depends=", "checkdepends="}
		lines := strings.Split(text, "\n")
		shortDescLine := -1
		lastDepLine := -1
		for i, l := range lines {
			if strings.HasPrefix(l, "short_desc=") {
				shortDescLine = i
			}
			for _, df := range depFields {
				if strings.HasPrefix(l, df) {
					lastDepLine = i
				}
			}
		}
		insertAt := len(lines)
		if shortDescLine >= 0 {
			insertAt = shortDescLine
		} else if lastDepLine >= 0 {
			insertAt = lastDepLine + 1
		}
		newLine := field + `="` + pkgName + `"`
		lines = append(lines[:insertAt], append([]string{newLine}, lines[insertAt:]...)...)
		text = strings.Join(lines, "\n")
	}
	g.editorEntry.SetText(text)
	if err := os.WriteFile(g.editorPath, []byte(text), 0o644); err != nil { //nolint:gosec
		g.setStatus(t("status.save_err") + err.Error())
		return
	}
	g.setStatus(fmt.Sprintf(t("status.added"), pkgName, field))
}

func (g *xbpsGuiApp) selectedName() string {
	list := g.filtered()
	if g.selected < 0 || g.selected >= len(list) {
		return ""
	}
	return list[g.selected].Name
}

func (g *xbpsGuiApp) loadEditorFile(name string) {
	path := filepath.Join(ResolveDistDir(g.distDir), "srcpkgs", name, "template")
	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		g.setStatus(t("status.read_err") + err.Error())
		g.editorPath = ""
		g.editorEntry.SetText("")
		return
	}
	g.editorPath = path
	g.editorEntry.SetText(string(data))
}

// pushLog commits current live output to history and shows text as new live output.
func (g *xbpsGuiApp) pushLog(text string) {
	g.commitLiveToHistory()
	g.logHistIdx = -1
	g.logLive = text
	g.output.SetText(text)
	g.updateNavBtns()
}

// commitLiveToHistory moves logLive into history (if non-empty).
func (g *xbpsGuiApp) commitLiveToHistory() {
	if g.logLive != "" {
		g.logHistory = append(g.logHistory, g.logLive)
		g.logLive = ""
	}
}

// logBack navigates one step back in output history.
// History:  [0, 1, 2, ...N-1]  live=logLive
// At live (-1): show history[N-1], set idx=N-1
// At idx>0:     show history[idx-1], set idx--
func (g *xbpsGuiApp) logBack() {
	if len(g.logHistory) == 0 {
		return
	}
	if g.logHistIdx == -1 {
		g.logHistIdx = len(g.logHistory) - 1
	} else if g.logHistIdx > 0 {
		g.logHistIdx--
	} else {
		return // already at oldest
	}
	g.output.SetText(g.logHistory[g.logHistIdx])
	g.updateNavBtns()
}

// logForward navigates one step forward (toward live output).
// At idx < N-1: show history[idx+1]
// At idx == N-1: show live output, set idx=-1
func (g *xbpsGuiApp) logForward() {
	if g.logHistIdx == -1 {
		return // already at live
	}
	g.logHistIdx++
	if g.logHistIdx >= len(g.logHistory) {
		g.logHistIdx = -1
		g.output.SetText(g.logLive)
	} else {
		g.output.SetText(g.logHistory[g.logHistIdx])
	}
	g.updateNavBtns()
}

// updateNavBtns shows/hides the back and forward navigation buttons.
func (g *xbpsGuiApp) updateNavBtns() {
	if g.btnBack == nil || g.btnFwd == nil {
		return
	}
	// canBack: history exists and we're not at the oldest entry
	canBack := len(g.logHistory) > 0 && (g.logHistIdx == -1 || g.logHistIdx > 0)
	// canFwd: we're browsing (not at live output)
	canFwd := g.logHistIdx != -1
	if canBack {
		g.btnBack.Show()
	} else {
		g.btnBack.Hide()
	}
	if canFwd {
		g.btnFwd.Show()
	} else {
		g.btnFwd.Hide()
	}
}

func (g *xbpsGuiApp) setOutput(text string) {
	g.output.SetText(text)
}

// checksumLineWidth is the target editor width in Fyne dp units for a
// checksum= line (9 chars key + 64 hex chars + newline = 74 chars monospace ~9 dp/char).
const checksumLineWidth float32 = 74 * 9

func (g *xbpsGuiApp) setEditorFocused(focused bool) {
	if focused {
		g.editorBtnSave.SetText(t("btn.save"))
		g.editorBtnLint.SetText(t("btn.lint"))
		g.editorBtnSum.SetText(t("btn.sum"))
		g.editorBtnBump.SetText(t("btn.bump"))
		// Set offset so editor is exactly checksumLineWidth wide.
		total := g.win.Canvas().Size().Width
		if total > checksumLineWidth {
			g.outerSplit.SetOffset(float64(1 - checksumLineWidth/total))
		} else {
			g.outerSplit.SetOffset(0.5)
		}
	} else {
		g.editorBtnSave.SetText("")
		g.editorBtnLint.SetText("")
		g.editorBtnSum.SetText("")
		g.editorBtnBump.SetText("")
		g.outerSplit.SetOffset(0.92)
	}
	g.outerSplit.Refresh()
}

func (g *xbpsGuiApp) setBuildRunning(running bool) {
	if running {
		g.btnBuild.SetText(t("btn.stop"))
		g.btnBuild.SetIcon(theme.MediaStopIcon())
		g.btnBuild.Importance = widget.DangerImportance
	} else {
		g.btnBuild.SetText(t("btn.build"))
		g.btnBuild.SetIcon(theme.MediaPlayIcon())
		g.btnBuild.Importance = widget.HighImportance
		g.buildCancel = nil
	}
}

func (g *xbpsGuiApp) runCmd(label string, args ...string) {
	g.runCmdCtx(false, label, args...)
}

func (g *xbpsGuiApp) runCmdCtx(cancellable bool, label string, args ...string) {
	pkg := g.selectedName()
	statusLabel := label
	if pkg != "" {
		statusLabel = fmt.Sprintf("%s %s", label, pkg)
	}
	g.setStatus(fmt.Sprintf(t("status.running"), statusLabel))
	// Save current live output to history, then start fresh.
	g.commitLiveToHistory()
	g.logHistIdx = -1
	g.logLive = ""
	g.output.SetText("")
	g.updateNavBtns()

	ctx := context.Background()
	if cancellable {
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(context.Background())
		g.buildCancel = cancel
		g.setBuildRunning(true)
	}

	go func() {
		w := writerFunc(func(p []byte) (int, error) {
			g.output.Append(string(p))
			return len(p), nil
		})
		_, err := RunXbpsPtyCtx(ctx, g.distDir, w, args...)
		if cancellable {
			g.setBuildRunning(false)
		}
		if g.output.plain.Len() == 0 {
			if err != nil {
				g.setOutput(fmt.Sprintf(t("status.error"), err.Error()))
			} else {
				g.setOutput(statusLabel + " OK")
			}
		}
		// Snapshot the completed output as the new live buffer.
		g.logLive = g.output.plain.String()
		g.updateNavBtns()
		if err != nil {
			if ctx.Err() != nil {
				g.setStatus(fmt.Sprintf(t("status.failed"), statusLabel, t("btn.stop")))
			} else {
				g.setStatus(fmt.Sprintf(t("status.failed"), statusLabel, err.Error()))
			}
		} else {
			g.setStatus(fmt.Sprintf(t("status.ok"), statusLabel))
		}
	}()
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

// focusEntry is a plain multiline Entry used as the inline template editor.
// onFocus is called with true on focus gained, false on focus lost.
type focusEntry struct {
	widget.Entry
	onFocus func(bool)
}

func (e *focusEntry) FocusGained() {
	e.Entry.FocusGained()
	if e.onFocus != nil {
		e.onFocus(true)
	}
}

func (e *focusEntry) FocusLost() {
	e.Entry.FocusLost()
	if e.onFocus != nil {
		e.onFocus(false)
	}
}

func newFocusEntry(onFocus func(bool)) *focusEntry {
	e := &focusEntry{onFocus: onFocus}
	e.ExtendBaseWidget(e)
	e.MultiLine = true
	e.Wrapping = fyne.TextWrapOff
	e.TextStyle = fyne.TextStyle{Monospace: true}
	return e
}

// ── Build widget tree ─────────────────────────────────────────────────

func (g *xbpsGuiApp) buildContent() fyne.CanvasObject {
	g.editorEntry = newFocusEntry(nil)

	search := widget.NewEntry()
	search.SetPlaceHolder(t("search.placeholder"))
	search.OnChanged = func(q string) {
		prevName := g.selectedName()
		g.search = q
		list := g.filtered()
		g.templateList.Refresh()

		if len(list) == 1 {
			g.selected = 0
			g.templateList.Select(0)
			g.showDetail(list[0].Name)
			return
		}

		if prevName != "" {
			for i, tmpl := range list {
				if tmpl.Name == prevName {
					g.selected = i
					g.templateList.Select(i)
					return
				}
			}
		}

		g.selected = -1
		g.clearDetail()
	}

	g.templateList = widget.NewList(
		func() int { return len(g.filtered()) },
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("template-placeholder")
			lbl.TextStyle = fyne.TextStyle{Monospace: true}
			return lbl
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			list := g.filtered()
			if id < len(list) {
				obj.(*widget.Label).SetText(list[id].Name)
			}
		},
	)
	g.templateList.OnSelected = func(id widget.ListItemID) {
		g.selected = id
		list := g.filtered()
		if id < len(list) {
			g.showDetail(list[id].Name)
		}
	}

	// Fix the list height to show exactly 4 rows; output fills the rest below.
	const listRowH float32 = 38
	listScroll := container.NewVScroll(g.templateList)
	listScroll.SetMinSize(fyne.NewSize(0, listRowH*4))

	g.detailName = widget.NewLabel("—")
	g.detailName.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	g.detailVer = widget.NewLabel("—")
	g.detailDesc = widget.NewLabel("—")
	g.detailDesc.Wrapping = fyne.TextWrapBreak

	detailForm := widget.NewForm(
		widget.NewFormItem(t("detail.name"), g.detailName),
		widget.NewFormItem(t("detail.version"), g.detailVer),
		widget.NewFormItem(t("detail.desc"), g.detailDesc),
	)

	// Build options: -Q (with tests), -C (confpkg) - can be used together
	checkQ := widget.NewCheck("-Q", func(checked bool) {
		if checked {
			g.buildMode = g.buildMode + " -Q"
		} else {
			g.buildMode = strings.ReplaceAll(g.buildMode, " -Q", "")
		}
		g.buildMode = strings.TrimSpace(g.buildMode)
	})
	checkC := widget.NewCheck("-C", func(checked bool) {
		if checked {
			g.buildMode = g.buildMode + " -C"
		} else {
			g.buildMode = strings.ReplaceAll(g.buildMode, " -C", "")
		}
		g.buildMode = strings.TrimSpace(g.buildMode)
	})

	// Build button
	g.btnBuild = widget.NewButtonWithIcon(t("btn.build"), theme.MediaPlayIcon(), func() {
		if g.buildCancel != nil {
			g.buildCancel()
			return
		}
		name := g.selectedName()
		if name == "" {
			return
		}
		args := []string{"./xbps-src", "pkg"}
		// Add flags if any are checked
		if strings.Contains(g.buildMode, "-Q") {
			args = append(args, "-Q")
		}
		if strings.Contains(g.buildMode, "-C") {
			args = append(args, "-C")
		}
		args = append(args, name)
		g.runCmdCtx(true, "build", args...)
	})
	g.btnBuild.Importance = widget.HighImportance

	// Clean button
	g.btnClean = widget.NewButtonWithIcon(t("btn.clean"), theme.DeleteIcon(), func() {
		if name := g.selectedName(); name != "" {
			g.runCmd("clean", "./xbps-src", "clean", name)
		}
	})

	// Install button - moved after Clean
	g.btnInstall = widget.NewButtonWithIcon(t("btn.install"), theme.DownloadIcon(), func() {
		if name := g.selectedName(); name != "" {
			g.runCmd("install", "xi", name)
		}
	})

	btnHomepage := widget.NewButtonWithIcon(t("btn.homepage"), theme.HomeIcon(), func() {
		name := g.selectedName()
		if name == "" {
			return
		}
		meta := ReadMeta(g.distDir, name)
		if meta.Homepage != "" {
			OpenBrowser(meta.Homepage)
		}
	})

	btnRepology := widget.NewButtonWithIcon(t("btn.repology"), theme.SearchIcon(), func() {
		if name := g.selectedName(); name != "" {
			OpenBrowser("https://repology.org/projects/?search=" + name)
		}
	})

	btnBootstrap := widget.NewButtonWithIcon(t("btn.bootstrap"), theme.ViewRefreshIcon(), func() {
		g.runCmd("bootstrap-update", "./xbps-src", "bootstrap-update")
	})
	btnBootstrap.Importance = widget.LowImportance

	actionRow1 := container.NewHBox(btnBootstrap, layout.NewSpacer(), btnHomepage, btnRepology)
	actionRow2 := container.NewHBox(checkQ, checkC, g.btnBuild, layout.NewSpacer(), g.btnClean, g.btnInstall)

	g.output = newOutputPanel(func(sel string, pos fyne.Position) { g.showSelectionMenu(sel, pos) })

	g.statusBar = widget.NewLabel("")
	g.statusBar.TextStyle = fyne.TextStyle{Italic: true, Monospace: true}

	diskText := fmt.Sprintf("XBPS_DISTDIR=%s", filepath.Clean(ResolveDistDir(g.distDir)))
	if disk := DiskInfo(g.distDir); disk != "" {
		diskText += "  💾 " + disk
	}
	dirLabel := widget.NewLabel(diskText)
	dirLabel.TextStyle = fyne.TextStyle{Monospace: true}

	btnReload := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), func() {
		g.reload()
		g.setStatus(t("status.reloaded"))
	})
	btnReload.Importance = widget.LowImportance

	btnFind := widget.NewButtonWithIcon("", theme.SearchIcon(), func() { g.output.ShowFind() })
	btnFind.Importance = widget.LowImportance
	btnAbout := widget.NewButtonWithIcon("", theme.InfoIcon(), func() { g.showAbout() })
	btnAbout.Importance = widget.LowImportance

	g.btnBack = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() { g.logBack() })
	g.btnBack.Importance = widget.LowImportance
	g.btnBack.Hide()
	g.btnFwd = widget.NewButtonWithIcon("", theme.NavigateNextIcon(), func() { g.logForward() })
	g.btnFwd.Importance = widget.LowImportance
	g.btnFwd.Hide()

	statusBar := container.NewHBox(btnAbout, btnReload, btnFind, g.statusBar, layout.NewSpacer(), g.btnBack, g.btnFwd, layout.NewSpacer(), dirLabel)

	// Top split: left = search + list (4 rows), right = detail + buttons.
	leftTop := container.NewVBox(search, widget.NewSeparator(), listScroll)
	rightTop := container.NewVBox(detailForm, widget.NewSeparator(), actionRow1, actionRow2)

	topSplit := container.NewHSplit(
		container.NewPadded(leftTop),
		container.NewPadded(rightTop),
	)
	topSplit.SetOffset(0.45)

	// Output terminal spans full width below the split.
	mainPanel := container.NewBorder(topSplit, nil, nil, nil, g.output.CanvasObject())

	g.clearDetail()

	g.editorBtnSave = widget.NewButtonWithIcon(t("btn.save"), theme.DocumentSaveIcon(), func() {
		if g.editorPath == "" {
			return
		}
		if err := os.WriteFile(g.editorPath, []byte(g.editorEntry.Text), 0o644); err != nil { //nolint:gosec
			g.setStatus(t("status.save_err") + err.Error())
			return
		}
		g.setStatus(t("status.save_ok"))
	})
	g.editorBtnSave.Importance = widget.HighImportance

	g.editorBtnLint = widget.NewButtonWithIcon(t("btn.lint"), theme.WarningIcon(), func() {
		if name := g.selectedName(); name != "" {
			g.runCmd("lint", "xlint", name)
		}
	})

	g.editorBtnSum = widget.NewButtonWithIcon(t("btn.sum"), theme.ConfirmIcon(), func() {
		if name := g.selectedName(); name != "" {
			g.runCmd("checksum", "xgensum", "-i", name)
		}
	})

	g.editorBtnBump = widget.NewButtonWithIcon(t("btn.bump"), theme.MoveUpIcon(), func() {
		if name := g.selectedName(); name != "" {
			g.runCmd("bump", "xxautobump", name)
		}
	})

	g.editorTitle = widget.NewLabel(t("editor.title"))
	g.editorTitle.TextStyle = fyne.TextStyle{Bold: true}

	editorToolbar := container.NewHBox(
		g.editorTitle,
		layout.NewSpacer(),
		g.editorBtnLint, g.editorBtnSum, g.editorBtnBump,
		widget.NewSeparator(),
		g.editorBtnSave,
	)

	g.editorTop = container.NewVBox(widget.NewSeparator(), container.NewPadded(editorToolbar), widget.NewSeparator())

	editorPanel := container.NewBorder(
		g.editorTop,
		nil, nil, nil,
		container.NewScroll(g.editorEntry),
	)

	// Initial state: icon-only (unfocused).
	g.editorBtnSave.SetText("")
	g.editorBtnLint.SetText("")
	g.editorBtnSum.SetText("")
	g.editorBtnBump.SetText("")
	g.outerSplit = container.NewHSplit(mainPanel, editorPanel)
	g.outerSplit.SetOffset(0.92)
	// Wire up focus callback now that outerSplit exists.
	g.editorEntry.onFocus = g.setEditorFocused

	return container.NewBorder(
		nil,
		container.NewVBox(widget.NewSeparator(), container.NewPadded(statusBar)),
		nil, nil,
		g.outerSplit,
	)
}

// RunGUI runs the xbps plugin as a standalone Fyne application.
func RunGUI(distDir string) {
	a := app.New()
	win := a.NewWindow(t("app.window"))
	g := &xbpsGuiApp{
		win:       win,
		distDir:   distDir,
		cfg:       LoadConfig(),
		selected:  -1,
		buildMode: "",
	}
	g.templates = LoadTemplates(distDir)
	win.SetContent(g.buildContent())
	win.Resize(fyne.NewSize(1400, 700))
	win.SetMaster()
	win.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		if e.Name == fyne.KeyEscape {
			a.Quit()
		}
	})
	win.ShowAndRun()
}
