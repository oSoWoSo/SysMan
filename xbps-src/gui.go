//go:build !tui_only

package xbpssrc

import (
	"fmt"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ── App state ─────────────────────────────────────────────────────────

type xbpsGuiApp struct {
	win      fyne.Window
	distDir  string
	templates []Template
	selected  int
	search    string

	templateList *widget.List
	detailName   *widget.Label
	detailVer    *widget.Label
	detailDesc   *widget.Label
	outputEntry  *widget.Entry
	statusBar    *widget.Label

	btnBuild    *widget.Button
	btnLint     *widget.Button
	btnChecksum *widget.Button
	btnBump     *widget.Button
	btnInstall  *widget.Button
	btnClean    *widget.Button
	btnEdit     *widget.Button
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
	g.btnLint.Enable()
	g.btnChecksum.Enable()
	g.btnBump.Enable()
	g.btnInstall.Enable()
	g.btnClean.Enable()
	g.btnEdit.Enable()
}

func (g *xbpsGuiApp) clearDetail() {
	g.detailName.SetText("—")
	g.detailVer.SetText("—")
	g.detailDesc.SetText("—")
	g.btnBuild.Disable()
	g.btnLint.Disable()
	g.btnChecksum.Disable()
	g.btnBump.Disable()
	g.btnInstall.Disable()
	g.btnClean.Disable()
	g.btnEdit.Disable()
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

// runCmd runs a command in a goroutine and updates the output + status.
func (g *xbpsGuiApp) runCmd(label string, args ...string) {
	g.setStatus(fmt.Sprintf("Running: %s…", label))
	g.outputEntry.SetText("")
	go func() {
		out, err := RunXbps(g.distDir, args...)
		g.outputEntry.SetText(out)
		if err != nil {
			g.setStatus(fmt.Sprintf("✗ %s failed: %s", label, err.Error()))
		} else {
			g.setStatus(fmt.Sprintf("✓ %s OK", label))
		}
	}()
}

// ── Build widget tree ─────────────────────────────────────────────────

func (g *xbpsGuiApp) buildContent() fyne.CanvasObject {
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

	g.btnLint = widget.NewButton("Lint", func() {
		if name := g.selectedName(); name != "" {
			g.runCmd("lint", "xlint", name)
		}
	})

	g.btnChecksum = widget.NewButton("Checksum", func() {
		if name := g.selectedName(); name != "" {
			g.runCmd("checksum", "xgensum", "-i", name)
		}
	})

	g.btnBump = widget.NewButton("Bump", func() {
		if name := g.selectedName(); name != "" {
			g.runCmd("bump", "xxautobump", name)
		}
	})

	g.btnInstall = widget.NewButton("Install", func() {
		if name := g.selectedName(); name != "" {
			g.runCmd("install", "xi", name)
		}
	})

	g.btnClean = widget.NewButtonWithIcon("Clean", theme.DeleteIcon(), func() {
		if name := g.selectedName(); name != "" {
			g.runCmd("clean", "./xbps-src", "clean", name)
		}
	})

	g.btnEdit = widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() {
		if name := g.selectedName(); name != "" {
			OpenEditor(g.distDir, name)
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

	actionRow1 := container.NewHBox(g.btnBuild, g.btnLint, g.btnChecksum, g.btnBump)
	actionRow2 := container.NewHBox(g.btnInstall, g.btnClean, g.btnEdit, btnHomepage, btnRepology,
		layout.NewSpacer(), btnBootstrap)

	// ── Output area ───────────────────────────────────────────────────
	g.outputEntry = widget.NewMultiLineEntry()
	g.outputEntry.Disable()
	g.outputEntry.Wrapping = fyne.TextWrapBreak

	outputScroll := container.NewScroll(g.outputEntry)
	outputScroll.SetMinSize(fyne.NewSize(0, 170))

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

	// ── Layout ────────────────────────────────────────────────────────
	rightPanel := container.NewVBox(
		detailForm,
		widget.NewSeparator(),
		actionRow1,
		actionRow2,
		widget.NewSeparator(),
		outputScroll,
	)

	split := container.NewHSplit(
		container.NewPadded(leftPanel),
		container.NewPadded(rightPanel),
	)
	split.SetOffset(0.32)

	g.clearDetail()

	return container.NewBorder(
		nil,
		container.NewVBox(widget.NewSeparator(), container.NewPadded(statusBar)),
		nil, nil,
		split,
	)
}

// RunGUI runs the xbps plugin as a standalone Fyne application.
func RunGUI(distDir string) {
	a := app.New()
	win := a.NewWindow("XBPS Templates")
	g := &xbpsGuiApp{
		win:      win,
		distDir:  distDir,
		selected: -1,
	}
	g.templates = LoadTemplates(distDir)
	win.SetContent(g.buildContent())
	win.Resize(fyne.NewSize(940, 640))
	win.SetMaster()
	win.ShowAndRun()
}
