//go:build !tui_only

package xbpssrc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"codeberg.org/oSoWoSo/SysMan/api"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ── App state ─────────────────────────────────────────────────────────

type xbpsGuiApp struct {
	win       fyne.Window
	distDir   string
	templates []Template
	selected  int
	search    string

	templateList *widget.List
	detailName   *widget.Label
	detailVer    *widget.Label
	detailDesc   *widget.Label
	outputRich   *widget.RichText
	outputScroll *container.Scroll
	highlighter  *api.Highlighter
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
	for _, t := range g.templates {
		if strings.Contains(strings.ToLower(t.Name), q) {
			out = append(out, t)
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

func (g *xbpsGuiApp) selectedName() string {
	list := g.filtered()
	if g.selected < 0 || g.selected >= len(list) {
		return ""
	}
	return list[g.selected].Name
}

// loadEditorFile reads the template file for name into the editor entry.
func (g *xbpsGuiApp) loadEditorFile(name string) {
	path := filepath.Join(ResolveDistDir(g.distDir), "srcpkgs", name, "template")
	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		g.setStatus("✗ cannot read template: " + err.Error())
		g.editorPath = ""
		g.editorEntry.SetText("")
		return
	}
	g.editorPath = path
	g.editorEntry.SetText(string(data))
}

// setOutput renders text into outputRich with syntax highlighting and scrolls to bottom.
func (g *xbpsGuiApp) setOutput(text string) {
	segs := g.highlighter.RichSegments(text)
	g.outputRich.Segments = segs
	g.outputRich.Refresh()
	g.outputScroll.ScrollToBottom()
}

// runCmd runs a command in a goroutine and streams output with highlighting.
// pkg is shown in status messages; pass "" to omit it.
func (g *xbpsGuiApp) runCmd(label string, args ...string) {
	pkg := g.selectedName()
	statusLabel := label
	if pkg != "" {
		statusLabel = fmt.Sprintf("%s %s", label, pkg)
	}
	g.setStatus(fmt.Sprintf("Running: %s…", statusLabel))
	g.outputRich.Segments = nil
	g.outputRich.Refresh()
	go func() {
		var buf strings.Builder
		w := writerFunc(func(p []byte) (int, error) {
			buf.Write(p)
			g.setOutput(buf.String())
			return len(p), nil
		})
		out, err := RunXbpsStream(g.distDir, w, args...)
		if out == "" {
			if err != nil {
				g.setOutput(fmt.Sprintf("error: %s", err.Error()))
			} else if buf.Len() == 0 {
				g.setOutput(fmt.Sprintf("%s OK", statusLabel))
			}
		}
		if err != nil {
			g.setStatus(fmt.Sprintf("✗ %s failed: %s", statusLabel, err.Error()))
		} else {
			g.setStatus(fmt.Sprintf("✓ %s OK", statusLabel))
		}
	}()
}

// writerFunc adapts a function to io.Writer.
type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

// ── Build widget tree ─────────────────────────────────────────────────

func (g *xbpsGuiApp) buildContent() fyne.CanvasObject {
	// ── Editor entry must be initialised before clearDetail() is called ──
	g.editorEntry = widget.NewMultiLineEntry()
	g.editorEntry.TextStyle = fyne.TextStyle{Monospace: true}
	g.editorEntry.Wrapping = fyne.TextWrapOff

	// ── Search bar ────────────────────────────────────────────────────
	search := widget.NewEntry()
	search.SetPlaceHolder("Search templates…")
	search.OnChanged = func(q string) {
		g.search = q
		g.selected = -1
		g.templateList.Refresh()
		g.clearDetail()
	}

	// ── Template list ─────────────────────────────────────────────────
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

	// ── Detail panel ──────────────────────────────────────────────────
	g.detailName = widget.NewLabel("—")
	g.detailName.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	g.detailVer = widget.NewLabel("—")
	g.detailDesc = widget.NewLabel("—")
	g.detailDesc.Wrapping = fyne.TextWrapBreak

	detailForm := widget.NewForm(
		widget.NewFormItem("Name", g.detailName),
		widget.NewFormItem("Version", g.detailVer),
		widget.NewFormItem("Description", g.detailDesc),
	)

	// ── Action buttons ────────────────────────────────────────────────
	g.btnBuild = widget.NewButtonWithIcon("Build", theme.MediaPlayIcon(), func() {
		name := g.selectedName()
		if name == "" {
			return
		}
		g.runCmd("build", "./xbps-src", "pkg", name)
	})
	g.btnBuild.Importance = widget.HighImportance

	g.btnInstall = widget.NewButton("install", func() {
		if name := g.selectedName(); name != "" {
			g.runCmd("install", "xi", name)
		}
	})

	g.btnClean = widget.NewButtonWithIcon("Clean", theme.DeleteIcon(), func() {
		if name := g.selectedName(); name != "" {
			g.runCmd("clean", "./xbps-src", "clean", name)
		}
	})

	btnHomepage := widget.NewButtonWithIcon("Homepage", theme.HomeIcon(), func() {
		name := g.selectedName()
		if name == "" {
			return
		}
		meta := ReadMeta(g.distDir, name)
		if meta.Homepage != "" {
			OpenBrowser(meta.Homepage)
		}
	})

	btnRepology := widget.NewButton("Repology", func() {
		if name := g.selectedName(); name != "" {
			OpenBrowser("https://repology.org/projects/?search=" + name)
		}
	})

	btnBootstrap := widget.NewButtonWithIcon("Bootstrap Update", theme.ViewRefreshIcon(), func() {
		g.runCmd("bootstrap-update", "./xbps-src", "bootstrap-update")
	})
	btnBootstrap.Importance = widget.LowImportance

	actionRow1 := container.NewHBox(btnBootstrap, layout.NewSpacer(), btnHomepage, btnRepology)
	actionRow2 := container.NewHBox(g.btnBuild, layout.NewSpacer(), g.btnInstall, g.btnClean)

	// ── Output area ───────────────────────────────────────────────────
	g.highlighter = api.NewHighlighter()
	g.outputRich = widget.NewRichText()
	g.outputRich.Wrapping = fyne.TextWrapBreak
	g.outputScroll = container.NewScroll(g.outputRich)
	g.outputScroll.SetMinSize(fyne.NewSize(0, 280))

	// ── Status bar ────────────────────────────────────────────────────
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
		g.setStatus("Reloaded")
	})
	btnReload.Importance = widget.LowImportance

	statusBar := container.NewHBox(btnReload, g.statusBar, layout.NewSpacer(), dirLabel)

	// ── Left main panel ───────────────────────────────────────────────
	topSection := container.NewVBox(
		detailForm,
		widget.NewSeparator(),
		actionRow1,
		actionRow2,
		widget.NewSeparator(),
	)
	leftMainPanel := container.NewBorder(topSection, nil, nil, nil, g.outputScroll)

	mainSplit := container.NewHSplit(
		container.NewPadded(leftPanel),
		container.NewPadded(leftMainPanel),
	)
	mainSplit.SetOffset(0.32)

	g.clearDetail()

	// ── Inline editor panel (always visible) ─────────────────────────
	btnEditorSave := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		if g.editorPath == "" {
			return
		}
		if err := os.WriteFile(g.editorPath, []byte(g.editorEntry.Text), 0o644); err != nil { //nolint:gosec
			g.setStatus("✗ save failed: " + err.Error())
			return
		}
		g.setStatus("✓ saved")
	})
	btnEditorSave.Importance = widget.HighImportance

	btnEditorLint := widget.NewButton("Lint", func() {
		if name := g.selectedName(); name != "" {
			g.runCmd("lint", "xlint", name)
		}
	})

	btnEditorSum := widget.NewButton("Sum", func() {
		if name := g.selectedName(); name != "" {
			g.runCmd("checksum", "xgensum", "-i", name)
		}
	})

	btnEditorBump := widget.NewButton("Bump", func() {
		if name := g.selectedName(); name != "" {
			g.runCmd("bump", "xxautobump", name)
		}
	})

	editorTitle := widget.NewLabel("Template editor")
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

	// ── Outer split: main | editor ────────────────────────────────────
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
	win := a.NewWindow("Templates")
	g := &xbpsGuiApp{
		win:      win,
		distDir:  distDir,
		selected: -1,
	}
	g.templates = LoadTemplates(distDir)
	win.SetContent(g.buildContent())
	win.Resize(fyne.NewSize(1400, 700))
	win.SetMaster()
	win.ShowAndRun()
}
