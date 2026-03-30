//go:build !tui_only

package plugin

import (
	"fmt"
	"image/color"
	"net/url"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ── Colors for detail ────────────────────────────────────────────────

var (
	colorEnabled  = color.NRGBA{R: 0x2e, G: 0xcc, B: 0x71, A: 0xff} // green
	colorDisabled = color.NRGBA{R: 0xff, G: 0x55, B: 0x55, A: 0xff} // red
	colorMuted    = color.NRGBA{R: 0x55, G: 0x5a, B: 0x60, A: 0xff} // grey
)

// ── Custom theme ─────────────────────────────────────────────────────

type darkIndustrialTheme struct{ fyne.Theme }

func (th darkIndustrialTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0x14, G: 0x17, B: 0x1a, A: 0xff}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0xd8, G: 0xdc, B: 0xe0, A: 0xff}
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0x00, G: 0xb8, B: 0xd4, A: 0xff}
	case theme.ColorNameButton:
		return color.NRGBA{R: 0x1e, G: 0x23, B: 0x29, A: 0xff}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 0x1e, G: 0x23, B: 0x29, A: 0xff}
	case theme.ColorNameDisabled:
		return colorMuted
	case theme.ColorNameHover:
		return color.NRGBA{R: 0x00, G: 0xb8, B: 0xd4, A: 0x22}
	case theme.ColorNameSelection:
		return color.NRGBA{R: 0x00, G: 0x7a, B: 0x8e, A: 0x55}
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 0x2e, G: 0x34, B: 0x3b, A: 0xff}
	case theme.ColorNameSuccess:
		return colorEnabled
	case theme.ColorNameError:
		return colorDisabled
	}
	return th.Theme.Color(name, variant)
}

// ── Filter ───────────────────────────────────────────────────────────

type filterMode int

const (
	filterAll filterMode = iota
	filterEnabled
	filterDisabled
)

// ── Detail state widget ──────────────────────────────────────────────

type stateLabel struct {
	widget.Label
}

func newStateLabel() *stateLabel {
	l := &stateLabel{}
	l.ExtendBaseWidget(l)
	l.TextStyle = fyne.TextStyle{Bold: true}
	return l
}

type detailStateWidget struct {
	icon  *widget.Icon
	label *stateLabel
	box   *fyne.Container
}

func newDetailStateWidget() *detailStateWidget {
	icon := widget.NewIcon(theme.NewDisabledResource(theme.RadioButtonIcon()))
	lbl := newStateLabel()
	lbl.SetText(t("detail.empty"))
	lbl.Importance = widget.LowImportance
	return &detailStateWidget{
		icon:  icon,
		label: lbl,
		box:   container.NewHBox(icon, lbl),
	}
}

func (d *detailStateWidget) setEnabled(enabled bool) {
	if enabled {
		d.icon.SetResource(theme.NewSuccessThemedResource(theme.ConfirmIcon()))
		d.label.SetText(t("state.enabled"))
		d.label.Importance = widget.SuccessImportance
	} else {
		d.icon.SetResource(theme.NewErrorThemedResource(theme.CancelIcon()))
		d.label.SetText(t("state.disabled"))
		d.label.Importance = widget.DangerImportance
	}
	d.label.Refresh()
}

func (d *detailStateWidget) clear() {
	d.icon.SetResource(theme.NewDisabledResource(theme.RadioButtonIcon()))
	d.label.SetText(t("detail.empty"))
	d.label.Importance = widget.LowImportance
	d.label.Refresh()
}

// ── App state ────────────────────────────────────────────────────────

type guiApp struct {
	win            fyne.Window
	serviceDir     string
	serviceDestDir string

	services   []Service
	selected   int
	searchText string
	filter     filterMode

	serviceList *widget.List
	detailName  *widget.Label
	detailState *detailStateWidget
	detailSrc   *widget.Label
	detailDst   *widget.Label
	btnEnable   *widget.Button
	btnDisable  *widget.Button
	statusBar   *widget.Label
	countLabel  *widget.Label
}

func (s *guiApp) filtered() []Service {
	var out []Service
	for _, svc := range s.services {
		switch s.filter {
		case filterEnabled:
			if !svc.Enabled {
				continue
			}
		case filterDisabled:
			if svc.Enabled {
				continue
			}
		}
		if s.searchText != "" && !strings.Contains(strings.ToLower(svc.Name), strings.ToLower(s.searchText)) {
			continue
		}
		out = append(out, svc)
	}
	return out
}

func (s *guiApp) reload() {
	s.services = LoadServices(s.serviceDir, s.serviceDestDir)
	s.serviceList.Refresh()
	s.updateCount()
	list := s.filtered()
	if s.selected >= 0 && s.selected < len(list) {
		s.showDetail(list[s.selected])
	} else {
		s.selected = -1
		s.clearDetail()
	}
}

func (s *guiApp) updateCount() {
	enabled := 0
	for _, svc := range s.services {
		if svc.Enabled {
			enabled++
		}
	}
	s.countLabel.SetText(fmt.Sprintf(t("count.fmt"), enabled, len(s.services), len(s.filtered())))
}

func (s *guiApp) clearDetail() {
	s.detailName.SetText(t("detail.empty"))
	s.detailState.clear()
	s.detailSrc.SetText(t("detail.empty"))
	s.detailDst.SetText(t("detail.empty"))
	s.btnEnable.Disable()
	s.btnDisable.Disable()
}

func (s *guiApp) showDetail(svc Service) {
	s.detailName.SetText(svc.Name)
	s.detailState.setEnabled(svc.Enabled)
	s.detailSrc.SetText(filepath.Join(s.serviceDir, svc.Name))
	s.detailDst.SetText(filepath.Join(s.serviceDestDir, svc.Name))
	s.btnEnable.Enable()
	s.btnDisable.Enable()
	if svc.Enabled {
		s.btnEnable.Disable()
	} else {
		s.btnDisable.Disable()
	}
}

func (s *guiApp) setStatus(msg string) {
	s.statusBar.SetText(msg)
}

// showAbout displays the About dialog with app metadata.
func (s *guiApp) showAbout() {
	title := canvas.NewText("svman", color.NRGBA{R: 0x00, G: 0xb8, B: 0xd4, A: 0xff})
	title.TextSize = 26
	title.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}

	subtitle := canvas.NewText(t("app.subtitle"), colorMuted)
	subtitle.TextSize = 12

	infoForm := widget.NewForm(
		widget.NewFormItem(t("about.version"), widget.NewLabel(Version)),
		widget.NewFormItem(t("about.author"), widget.NewLabel(AppAuthor)),
		widget.NewFormItem(t("about.license"), widget.NewLabel(AppLicense)),
	)

	repoURL, _ := url.Parse(AppURL)
	link := widget.NewHyperlink(AppURL, repoURL)

	content := container.NewVBox(
		container.NewCenter(title),
		container.NewCenter(subtitle),
		widget.NewSeparator(),
		infoForm,
		container.NewCenter(link),
	)

	d := dialog.NewCustom(t("menu.about"), t("btn.close"), content, s.win)
	d.Show()
}

// buildContent builds the full widget tree and returns it as a CanvasObject.
// It does NOT call SetContent on the window — the caller is responsible for that.
// This allows the panel to be embedded in a parent application.
func (s *guiApp) buildContent() fyne.CanvasObject {
	// ── Header ───────────────────────────────────────────────────────
	titleText := canvas.NewText(t("app.title"), color.NRGBA{R: 0x00, G: 0xb8, B: 0xd4, A: 0xff})
	titleText.TextSize = 22
	titleText.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	subtitleText := canvas.NewText(t("app.subtitle"), colorMuted)
	subtitleText.TextSize = 11
	header := container.NewVBox(
		container.NewPadded(container.NewVBox(titleText, subtitleText)),
		widget.NewSeparator(),
	)

	// ── Search ───────────────────────────────────────────────────────
	search := widget.NewEntry()
	search.SetPlaceHolder(t("search.placeholder"))
	search.OnChanged = func(q string) {
		s.searchText = q
		s.selected = -1
		s.serviceList.Refresh()
		s.updateCount()
		s.clearDetail()
	}

	s.countLabel = widget.NewLabel("")
	s.countLabel.TextStyle = fyne.TextStyle{Italic: true}

	// ── Service list ─────────────────────────────────────────────────
	s.serviceList = widget.NewList(
		func() int { return len(s.filtered()) },
		func() fyne.CanvasObject {
			icon := widget.NewIcon(theme.CancelIcon())
			nameLbl := widget.NewLabel("service-placeholder")
			nameLbl.TextStyle = fyne.TextStyle{Monospace: true}
			return container.NewHBox(icon, nameLbl)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			list := s.filtered()
			if id >= len(list) {
				return
			}
			svc := list[id]
			row := obj.(*fyne.Container)
			icon := row.Objects[0].(*widget.Icon)
			nameLbl := row.Objects[1].(*widget.Label)
			nameLbl.SetText(svc.Name)
			if svc.Enabled {
				icon.SetResource(theme.NewSuccessThemedResource(theme.ConfirmIcon()))
			} else {
				icon.SetResource(theme.NewErrorThemedResource(theme.CancelIcon()))
			}
		},
	)
	s.serviceList.OnSelected = func(id widget.ListItemID) {
		s.selected = id
		list := s.filtered()
		if id < len(list) {
			s.showDetail(list[id])
		}
	}

	// ── Filter toggle buttons ────────────────────────────────────────
	var btnFilterAll, btnFilterEnabled, btnFilterDisabled *widget.Button

	applyFilter := func(f filterMode) {
		s.filter = f
		s.selected = -1
		s.serviceList.Refresh()
		s.updateCount()
		s.clearDetail()
		btnFilterAll.Importance = widget.MediumImportance
		btnFilterEnabled.Importance = widget.MediumImportance
		btnFilterDisabled.Importance = widget.MediumImportance
		switch f {
		case filterEnabled:
			btnFilterEnabled.Importance = widget.HighImportance
		case filterDisabled:
			btnFilterDisabled.Importance = widget.HighImportance
		default:
			btnFilterAll.Importance = widget.HighImportance
		}
		btnFilterAll.Refresh()
		btnFilterEnabled.Refresh()
		btnFilterDisabled.Refresh()
	}

	btnFilterAll = widget.NewButton(t("filter.all"), func() { applyFilter(filterAll) })
	btnFilterEnabled = widget.NewButton(t("filter.enabled"), func() { applyFilter(filterEnabled) })
	btnFilterDisabled = widget.NewButton(t("filter.disabled"), func() { applyFilter(filterDisabled) })
	filterRow := container.NewHBox(btnFilterAll, btnFilterEnabled, btnFilterDisabled)

	// ── Detail panel ─────────────────────────────────────────────────
	s.detailName = widget.NewLabel(t("detail.empty"))
	s.detailName.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	s.detailState = newDetailStateWidget()
	s.detailSrc = widget.NewLabel(t("detail.empty"))
	s.detailSrc.Wrapping = fyne.TextWrapBreak
	s.detailDst = widget.NewLabel(t("detail.empty"))
	s.detailDst.Wrapping = fyne.TextWrapBreak

	detailForm := widget.NewForm(
		widget.NewFormItem(t("detail.name"), s.detailName),
		widget.NewFormItem(t("detail.state"), s.detailState.box),
		widget.NewFormItem(t("detail.source"), s.detailSrc),
		widget.NewFormItem(t("detail.symlink"), s.detailDst),
	)

	// ── Action buttons ───────────────────────────────────────────────
	s.btnEnable = widget.NewButtonWithIcon(t("btn.enable"), theme.ConfirmIcon(), func() {
		list := s.filtered()
		if s.selected < 0 || s.selected >= len(list) {
			return
		}
		svc := list[s.selected]
		if err := EnableService(s.serviceDir, s.serviceDestDir, svc.Name); err != nil {
			dialog.ShowError(err, s.win)
			s.setStatus(t("status.err") + err.Error())
		} else {
			s.setStatus(fmt.Sprintf(t("status.enabled"), svc.Name))
			s.reload()
			s.serviceList.Select(s.selected)
		}
	})
	s.btnEnable.Importance = widget.HighImportance

	s.btnDisable = widget.NewButtonWithIcon(t("btn.disable"), theme.DeleteIcon(), func() {
		list := s.filtered()
		if s.selected < 0 || s.selected >= len(list) {
			return
		}
		svc := list[s.selected]
		dialog.ShowConfirm(
			t("confirm.title"),
			fmt.Sprintf(t("confirm.disable"), svc.Name),
			func(ok bool) {
				if !ok {
					return
				}
				if err := DisableService(s.serviceDestDir, svc.Name); err != nil {
					dialog.ShowError(err, s.win)
					s.setStatus(t("status.err") + err.Error())
				} else {
					s.setStatus(fmt.Sprintf(t("status.disabled"), svc.Name))
					s.reload()
					s.serviceList.Select(s.selected)
				}
			}, s.win)
	})
	s.btnDisable.Importance = widget.DangerImportance

	btnReload := widget.NewButtonWithIcon(t("btn.reload"), theme.ViewRefreshIcon(), func() {
		s.reload()
		s.setStatus(t("status.reloaded"))
	})

	s.btnEnable.Disable()
	s.btnDisable.Disable()
	buttons := container.NewHBox(s.btnEnable, s.btnDisable, layout.NewSpacer(), btnReload)

	// ── Status bar ───────────────────────────────────────────────────
	s.statusBar = widget.NewLabel("")
	s.statusBar.TextStyle = fyne.TextStyle{Italic: true, Monospace: true}
	// About button — info icon at the bottom-left corner
	btnAbout := widget.NewButtonWithIcon("", theme.InfoIcon(), func() { s.showAbout() })
	btnAbout.Importance = widget.LowImportance
	statusBar := container.NewHBox(btnAbout, s.statusBar)

	// ── Dir info ─────────────────────────────────────────────────────
	dirInfo := widget.NewLabel(fmt.Sprintf("SERVICEDIR=%s\nSERVICEDESTDIR=%s", s.serviceDir, s.serviceDestDir))
	dirInfo.TextStyle = fyne.TextStyle{Monospace: true}

	detailTitle := canvas.NewText(t("detail.title"), color.NRGBA{R: 0x00, G: 0xb8, B: 0xd4, A: 0xcc})
	detailTitle.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	configTitle := canvas.NewText(t("config.title"), colorMuted)
	configTitle.TextStyle = fyne.TextStyle{Monospace: true}

	rightPanel := container.NewVBox(
		detailTitle,
		widget.NewSeparator(),
		detailForm,
		widget.NewSeparator(),
		buttons,
		layout.NewSpacer(),
		widget.NewSeparator(),
		configTitle,
		dirInfo,
	)

	leftTop := container.NewVBox(search, filterRow, s.countLabel, widget.NewSeparator())
	leftPanel := container.NewBorder(leftTop, nil, nil, nil, s.serviceList)

	split := container.NewHSplit(container.NewPadded(leftPanel), container.NewPadded(rightPanel))
	split.SetOffset(0.42)

	root := container.NewBorder(
		header,
		container.NewVBox(widget.NewSeparator(), container.NewPadded(statusBar)),
		nil, nil,
		split,
	)

	// Set default filter (after all widgets are initialized)
	applyFilter(filterAll)

	return root
}

// ── Standalone runner ────────────────────────────────────────────────

// RunGUI runs svman as a standalone Fyne GUI application.
func RunGUI(serviceDir, serviceDestDir string) {
	InitI18n()
	a := app.New()
	a.Settings().SetTheme(darkIndustrialTheme{theme.DefaultTheme()})
	win := a.NewWindow(t("app.window"))
	g := &guiApp{
		win:            win,
		serviceDir:     serviceDir,
		serviceDestDir: serviceDestDir,
		selected:       -1,
	}
	g.services = LoadServices(serviceDir, serviceDestDir)
	win.SetContent(g.buildContent())
	win.Resize(fyne.NewSize(860, 560))
	win.SetMaster()
	win.ShowAndRun()
}
