//go:build !tui_only

package xbpssrc

import (
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

	// always-visible inline editor
	editorEntry *widget.Entry
	editorPath  string // path of the file currently loaded
}

func (g *xbpsGuiApp) filtered() []Template {
	q := strings.ToLower(g.search)
	if q == "" {
		return g.templates
	}
	var out []Template
	for _, tmpl := range g.templates {
		if strings.Contains(strings.ToLower(tmpl.Name), q) {
			out = append(out, tmpl)
		}
	}
	return out
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
	infoForm := widget.NewForm(
		widget.NewFormItem(t("about.version"), widget.NewLabel(svman.Version)),
		widget.NewFormItem(t("about.author"), widget.NewLabel(svman.AppAuthor)),
		widget.NewFormItem(t("about.license"), widget.NewLabel(svman.AppLicense)),
	)
	repoURL, _ := url.Parse(svman.AppURL)
	link := widget.NewHyperlink(svman.AppURL, repoURL)
	content := container.NewVBox(
		container.NewCenter(title),
		container.NewCenter(subtitle),
		widget.NewSeparator(),
		infoForm,
		container.NewCenter(link),
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
					g.setOutput(fmt.Sprintf("xlocate %s: %s\n%s", sel, err.Error(), out))
					g.setStatus(fmt.Sprintf("✗ xlocate %s", sel))
				} else {
					if out == "" {
						out = "(no results)"
					}
					g.setOutput(out)
					g.setStatus(fmt.Sprintf("✓ xlocate %s", sel))
				}
			}()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem(t("menu.websearch"), func() {
			OpenBrowser(g.cfg.SearchEngine + url.QueryEscape(sel))
		}),
	}

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

func (g *xbpsGuiApp) setOutput(text string) {
	g.output.SetText(text)
}

func (g *xbpsGuiApp) runCmd(label string, args ...string) {
	pkg := g.selectedName()
	statusLabel := label
	if pkg != "" {
		statusLabel = fmt.Sprintf("%s %s", label, pkg)
	}
	g.setStatus(fmt.Sprintf(t("status.running"), statusLabel))
	g.output.SetText("")
	go func() {
		w := writerFunc(func(p []byte) (int, error) {
			g.output.Append(string(p))
			return len(p), nil
		})
		_, err := RunXbpsStream(g.distDir, w, args...)
		if g.output.plain.Len() == 0 {
			if err != nil {
				g.setOutput(fmt.Sprintf(t("status.error"), err.Error()))
			} else {
				g.setOutput(statusLabel + " OK")
			}
		}
		if err != nil {
			g.setStatus(fmt.Sprintf(t("status.failed"), statusLabel, err.Error()))
		} else {
			g.setStatus(fmt.Sprintf(t("status.ok"), statusLabel))
		}
	}()
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

// ── Build widget tree ─────────────────────────────────────────────────

func (g *xbpsGuiApp) buildContent() fyne.CanvasObject {
	g.editorEntry = widget.NewMultiLineEntry()
	g.editorEntry.TextStyle = fyne.TextStyle{Monospace: true}
	g.editorEntry.Wrapping = fyne.TextWrapOff

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

	leftPanel := container.NewBorder(
		container.NewVBox(search, widget.NewSeparator()),
		nil, nil, nil,
		g.templateList,
	)

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

	g.btnBuild = widget.NewButtonWithIcon(t("btn.build"), theme.MediaPlayIcon(), func() {
		name := g.selectedName()
		if name == "" {
			return
		}
		g.runCmd("build", "./xbps-src", "pkg", name)
	})
	g.btnBuild.Importance = widget.HighImportance

	g.btnInstall = widget.NewButtonWithIcon(t("btn.install"), theme.DownloadIcon(), func() {
		if name := g.selectedName(); name != "" {
			g.runCmd("install", "xi", name)
		}
	})

	g.btnClean = widget.NewButtonWithIcon(t("btn.clean"), theme.DeleteIcon(), func() {
		if name := g.selectedName(); name != "" {
			g.runCmd("clean", "./xbps-src", "clean", name)
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
	actionRow2 := container.NewHBox(g.btnBuild, layout.NewSpacer(), g.btnInstall, g.btnClean)

	g.output = newOutputPanel(func(sel string, pos fyne.Position) { g.showSelectionMenu(sel, pos) })
	g.output.SetMinSize(fyne.NewSize(0, 280))

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
	statusBar := container.NewHBox(btnAbout, btnReload, btnFind, g.statusBar, layout.NewSpacer(), dirLabel)

	topSection := container.NewVBox(
		detailForm,
		widget.NewSeparator(),
		actionRow1,
		actionRow2,
		widget.NewSeparator(),
	)
	leftMainPanel := container.NewBorder(topSection, nil, nil, nil, g.output.CanvasObject())

	mainSplit := container.NewHSplit(
		container.NewPadded(leftPanel),
		container.NewPadded(leftMainPanel),
	)
	mainSplit.SetOffset(0.32)

	g.clearDetail()

	btnEditorSave := widget.NewButtonWithIcon(t("btn.save"), theme.DocumentSaveIcon(), func() {
		if g.editorPath == "" {
			return
		}
		if err := os.WriteFile(g.editorPath, []byte(g.editorEntry.Text), 0o644); err != nil { //nolint:gosec
			g.setStatus(t("status.save_err") + err.Error())
			return
		}
		g.setStatus(t("status.save_ok"))
	})
	btnEditorSave.Importance = widget.HighImportance

	btnEditorLint := widget.NewButtonWithIcon(t("btn.lint"), theme.WarningIcon(), func() {
		if name := g.selectedName(); name != "" {
			g.runCmd("lint", "xlint", name)
		}
	})

	btnEditorSum := widget.NewButtonWithIcon(t("btn.sum"), theme.ConfirmIcon(), func() {
		if name := g.selectedName(); name != "" {
			g.runCmd("checksum", "xgensum", "-i", name)
		}
	})

	btnEditorBump := widget.NewButtonWithIcon(t("btn.bump"), theme.MoveUpIcon(), func() {
		if name := g.selectedName(); name != "" {
			g.runCmd("bump", "xxautobump", name)
		}
	})

	editorTitle := widget.NewLabel(t("editor.title"))
	editorTitle.TextStyle = fyne.TextStyle{Bold: true}

	editorToolbar := container.NewHBox(
		editorTitle,
		layout.NewSpacer(),
		btnEditorLint, btnEditorSum, btnEditorBump,
		widget.NewSeparator(),
		btnEditorSave,
	)

	editorPanel := container.NewBorder(
		container.NewVBox(widget.NewSeparator(), container.NewPadded(editorToolbar), widget.NewSeparator()),
		nil, nil, nil,
		container.NewScroll(g.editorEntry),
	)

	outerSplit := container.NewHSplit(mainSplit, editorPanel)
	outerSplit.SetOffset(0.45)

	return container.NewBorder(
		nil,
		container.NewVBox(widget.NewSeparator(), container.NewPadded(statusBar)),
		nil, nil,
		outerSplit,
	)
}

// RunGUI runs the xbps plugin as a standalone Fyne application.
func RunGUI(distDir string) {
	a := app.New()
	win := a.NewWindow(t("app.window"))
	g := &xbpsGuiApp{
		win:      win,
		distDir:  distDir,
		cfg:      LoadConfig(),
		selected: -1,
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
