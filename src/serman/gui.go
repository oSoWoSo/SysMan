//go:build !tui_only

package serman

import (
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"

	"codeberg.org/oSoWoSo/SysMan/src/common"
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

// filterMode is an alias for FilterMode for the GUI.
type filterMode = FilterMode

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
	win     fyne.Window
	backend Backend

	services    []Service
	statusCache map[string]ServiceStatus // populated by reload(), read by showDetail
	selected    int
	searchText  string
	filter      filterMode

	serviceList   *widget.List
	detailName    *widget.Label
	detailState   *detailStateWidget
	detailRunning *widget.Label
	detailPID     *widget.Label
	detailUptime  *widget.Label
	detailSrc     *widget.Label
	detailDst     *widget.Label
	btnEnable     *common.HoverableButton
	btnDisable    *common.HoverableButton
	btnStart      *common.HoverableButton
	btnStop       *common.HoverableButton
	btnRestart    *common.HoverableButton
	btnHup        *common.HoverableButton
	btnPause      *common.HoverableButton
	btnContinue   *common.HoverableButton
	btnKill       *common.HoverableButton
	statusBar     *common.StatusBar
	countLabel    *widget.Label
}

func (s *guiApp) filtered() []Service {
	return Filter(s.services, s.filter, s.searchText,
		func(svc Service) bool { return svc.Enabled },
		func(svc Service, q string) bool {
			return strings.Contains(strings.ToLower(svc.Name), q)
		},
	)
}

func (s *guiApp) reload() {
	s.services = s.backend.List()
	// Collect names of all enabled services and fetch their status in one
	// elevated call so the user is prompted for a password only once.
	var enabledNames []string
	for _, svc := range s.services {
		if svc.Enabled {
			enabledNames = append(enabledNames, svc.Name)
		}
	}
	s.statusCache = s.backend.StatusAll(enabledNames)
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
	s.detailRunning.SetText(t("detail.empty"))
	s.detailPID.SetText(t("detail.empty"))
	s.detailUptime.SetText(t("detail.empty"))
	s.detailSrc.SetText(t("detail.empty"))
	s.detailDst.SetText(t("detail.empty"))
	s.btnEnable.Disable()
	s.btnDisable.Disable()
	s.btnStart.Disable()
	s.btnStop.Disable()
	s.btnRestart.Disable()
	s.btnHup.Disable()
	s.btnPause.Disable()
	s.btnContinue.Disable()
	s.btnKill.Disable()
}

func (s *guiApp) showDetail(svc Service) {
	s.detailName.SetText(svc.Name)
	s.detailState.setEnabled(svc.Enabled)
	svcDir, destDir := s.backend.Dirs()
	s.detailSrc.SetText(filepath.Join(svcDir, svc.Name))
	s.detailDst.SetText(filepath.Join(destDir, svc.Name))
	s.btnEnable.Enable()
	s.btnDisable.Enable()
	if svc.Enabled {
		s.btnEnable.Disable()
		s.btnStart.Enable()
		s.btnStop.Enable()
		s.btnRestart.Enable()
		s.btnHup.Enable()
		s.btnPause.Enable()
		s.btnContinue.Enable()
		s.btnKill.Enable()
		// Use cached status — populated by reload() in one elevated call.
		if st, ok := s.statusCache[svc.Name]; ok {
			s.updateRunningStatus(st)
		} else {
			s.detailRunning.SetText(t("detail.empty"))
			s.detailPID.SetText(t("detail.empty"))
			s.detailUptime.SetText(t("detail.empty"))
		}
	} else {
		s.btnDisable.Disable()
		s.btnStart.Disable()
		s.btnStop.Disable()
		s.btnRestart.Disable()
		s.btnHup.Disable()
		s.btnPause.Disable()
		s.btnContinue.Disable()
		s.btnKill.Disable()
		s.detailRunning.SetText(t("detail.empty"))
		s.detailPID.SetText(t("detail.empty"))
		s.detailUptime.SetText(t("detail.empty"))
	}
}

// refreshOneStatus fetches the current status for name, updates the cache,
// and refreshes the running-status display.  Used after control actions
// (start/stop/restart/…) where only one service changes.
func (s *guiApp) refreshOneStatus(name string) {
	st := s.backend.Status(name)
	fyne.Do(func() {
		if s.statusCache == nil {
			s.statusCache = make(map[string]ServiceStatus)
		}
		s.statusCache[name] = st
		s.updateRunningStatus(st)
	})
}

func (s *guiApp) updateRunningStatus(st ServiceStatus) {
	fyne.Do(func() {
		if st.Running {
			s.detailRunning.SetText(t("state.running"))
		} else {
			s.detailRunning.SetText(t("state.stopped"))
		}
		if st.PID > 0 {
			s.detailPID.SetText(fmt.Sprintf("%d", st.PID))
		} else {
			s.detailPID.SetText(t("detail.empty"))
		}
		if st.Uptime != "" {
			s.detailUptime.SetText(st.Uptime)
		} else {
			s.detailUptime.SetText(t("detail.empty"))
		}
	})
}

func (s *guiApp) setStatus(msg string) {
	fyne.Do(func() {
		s.statusBar.SetText(msg)
	})
}

// showAbout displays the About dialog with app metadata.
func (s *guiApp) showAbout() {
	common.ShowAbout(common.AboutConfig{
		Win:       s.win,
		Title:     "svman",
		Subtitle:  t("app.subtitle"),
		Version:   Version,
		Author:    AppAuthor,
		License:   AppLicense,
		URL:       AppURL,
		DialogBtn: t("menu.about"),
		CloseBtn:  t("btn.close"),
	})
}

// buildContent builds the full widget tree and returns it as a CanvasObject.
// showHeader controls whether the svman title/subtitle bar is rendered.
// Pass false when embedding in a parent app (e.g. sysmanager tab), true for standalone use.
func (s *guiApp) buildContent(showHeader bool) fyne.CanvasObject {
	// ── Status bar (must be created before any HoverableButton) ──────
	if s.statusBar == nil {
		s.statusBar = common.NewStatusBar()
	}
	s.statusBar.TextStyle = fyne.TextStyle{Italic: true, Monospace: true}

	// ── Header ───────────────────────────────────────────────────────
	var header fyne.CanvasObject
	if showHeader {
		titleText := canvas.NewText(t("app.title"), color.NRGBA{R: 0x00, G: 0xb8, B: 0xd4, A: 0xff})
		titleText.TextSize = 22
		titleText.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
		subtitleText := canvas.NewText(t("app.subtitle"), colorMuted)
		subtitleText.TextSize = 11
		header = container.NewVBox(
			container.NewPadded(container.NewVBox(titleText, subtitleText)),
			widget.NewSeparator(),
		)
	}

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
	var btnFilterAll, btnFilterEnabled, btnFilterDisabled *common.HoverableButton

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
		case FilterEnabled:
			btnFilterEnabled.Importance = widget.HighImportance
		case FilterDisabled:
			btnFilterDisabled.Importance = widget.HighImportance
		default:
			btnFilterAll.Importance = widget.HighImportance
		}
		btnFilterAll.Refresh()
		btnFilterEnabled.Refresh()
		btnFilterDisabled.Refresh()
	}

	btnFilterAll = common.NewHoverableButtonText(t("filter.all"), t("tooltip.serman.filter_all"), s.statusBar, func() { applyFilter(FilterAll) })
	btnFilterEnabled = common.NewHoverableButtonText(t("filter.enabled"), t("tooltip.serman.filter_enabled"), s.statusBar, func() { applyFilter(FilterEnabled) })
	btnFilterDisabled = common.NewHoverableButtonText(t("filter.disabled"), t("tooltip.serman.filter_disabled"), s.statusBar, func() { applyFilter(FilterDisabled) })
	filterRow := container.NewHBox(btnFilterAll, btnFilterEnabled, btnFilterDisabled)

	// ── Detail panel ─────────────────────────────────────────────────
	s.detailName = widget.NewLabel(t("detail.empty"))
	s.detailName.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	s.detailState = newDetailStateWidget()
	s.detailRunning = widget.NewLabel(t("detail.empty"))
	s.detailPID = widget.NewLabel(t("detail.empty"))
	s.detailPID.TextStyle = fyne.TextStyle{Monospace: true}
	s.detailUptime = widget.NewLabel(t("detail.empty"))
	s.detailSrc = widget.NewLabel(t("detail.empty"))
	s.detailSrc.Wrapping = fyne.TextWrapBreak
	s.detailDst = widget.NewLabel(t("detail.empty"))
	s.detailDst.Wrapping = fyne.TextWrapBreak

	detailForm := widget.NewForm(
		widget.NewFormItem(t("detail.name"), s.detailName),
		widget.NewFormItem(t("detail.state"), s.detailState.box),
		widget.NewFormItem(t("detail.running"), s.detailRunning),
		widget.NewFormItem(t("detail.pid"), s.detailPID),
		widget.NewFormItem(t("detail.uptime"), s.detailUptime),
		widget.NewFormItem(t("detail.source"), s.detailSrc),
		widget.NewFormItem(t("detail.symlink"), s.detailDst),
	)

	// ── Action buttons ───────────────────────────────────────────────
	s.btnEnable = common.NewHoverableButton(t("btn.enable"), theme.ConfirmIcon(), t("tooltip.serman.enable"), s.statusBar, func() {
		list := s.filtered()
		if s.selected < 0 || s.selected >= len(list) {
			return
		}
		svc := list[s.selected]
		if err := s.backend.Enable(svc.Name); err != nil {
			dialog.ShowError(err, s.win)
			s.setStatus(t("status.err") + err.Error())
		} else {
			s.setStatus(fmt.Sprintf(t("status.enabled"), svc.Name))
			s.reload()
			s.serviceList.Select(s.selected)
		}
	})
	s.btnEnable.Importance = widget.HighImportance

	s.btnDisable = common.NewHoverableButton(t("btn.disable"), theme.DeleteIcon(), t("tooltip.serman.disable"), s.statusBar, func() {
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
				if err := s.backend.Disable(svc.Name); err != nil {
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

	btnReload := common.NewHoverableButton(t("btn.reload"), theme.ViewRefreshIcon(), t("tooltip.serman.reload"), s.statusBar, func() {
		s.reload()
		if os.Getuid() != 0 {
			s.setStatus(t("status.reloaded") + " ⚠ " + t("status.no_root"))
		} else {
			s.setStatus(t("status.reloaded"))
		}
	})

	// ── sv control buttons ───────────────────────────────────────────
	s.btnStart = common.NewHoverableButton(t("btn.start"), theme.MediaPlayIcon(), t("tooltip.serman.start"), s.statusBar, func() {
		list := s.filtered()
		if s.selected < 0 || s.selected >= len(list) {
			return
		}
		name := list[s.selected].Name
		go func() {
			if err := s.backend.Start(name); err != nil {
				s.setStatus(t("status.err") + err.Error())
				return
			}
			s.setStatus(fmt.Sprintf(t("status.started"), name))
			s.refreshOneStatus(name)
		}()
	})
	s.btnStart.Importance = widget.SuccessImportance

	s.btnStop = common.NewHoverableButton(t("btn.stop"), theme.MediaStopIcon(), t("tooltip.serman.stop"), s.statusBar, func() {
		list := s.filtered()
		if s.selected < 0 || s.selected >= len(list) {
			return
		}
		name := list[s.selected].Name
		dialog.ShowConfirm(
			t("confirm.title"),
			fmt.Sprintf(t("confirm.stop"), name),
			func(ok bool) {
				if !ok {
					return
				}
				go func() {
					if err := s.backend.Stop(name); err != nil {
						s.setStatus(t("status.err") + err.Error())
						return
					}
					s.setStatus(fmt.Sprintf(t("status.stopped"), name))
					st := s.backend.Status(name)
					s.updateRunningStatus(st)
				}()
			}, s.win)
	})
	s.btnStop.Importance = widget.DangerImportance

	s.btnRestart = common.NewHoverableButton(t("btn.restart"), theme.ViewRefreshIcon(), t("tooltip.serman.restart"), s.statusBar, func() {
		list := s.filtered()
		if s.selected < 0 || s.selected >= len(list) {
			return
		}
		name := list[s.selected].Name
		go func() {
			if err := s.backend.Restart(name); err != nil {
				s.setStatus(t("status.err") + err.Error())
				return
			}
			s.setStatus(fmt.Sprintf(t("status.restarted"), name))
			s.refreshOneStatus(name)
		}()
	})

	s.btnHup = common.NewHoverableButton(t("btn.hup"), theme.MailSendIcon(), t("tooltip.serman.hup"), s.statusBar, func() {
		list := s.filtered()
		if s.selected < 0 || s.selected >= len(list) {
			return
		}
		name := list[s.selected].Name
		go func() {
			if err := s.backend.Reload(name); err != nil {
				s.setStatus(t("status.err") + err.Error())
				return
			}
			s.setStatus(fmt.Sprintf(t("status.hupped"), name))
			s.refreshOneStatus(name)
		}()
	})

	s.btnPause = common.NewHoverableButton(t("btn.pause"), theme.MediaPauseIcon(), t("tooltip.serman.pause"), s.statusBar, func() {
		list := s.filtered()
		if s.selected < 0 || s.selected >= len(list) {
			return
		}
		name := list[s.selected].Name
		go func() {
			if err := s.backend.Pause(name); err != nil {
				s.setStatus(t("status.err") + err.Error())
				return
			}
			s.setStatus(fmt.Sprintf(t("status.paused"), name))
			s.refreshOneStatus(name)
		}()
	})

	s.btnContinue = common.NewHoverableButton(t("btn.continue"), theme.MediaPlayIcon(), t("tooltip.serman.continue"), s.statusBar, func() {
		list := s.filtered()
		if s.selected < 0 || s.selected >= len(list) {
			return
		}
		name := list[s.selected].Name
		go func() {
			if err := s.backend.Continue(name); err != nil {
				s.setStatus(t("status.err") + err.Error())
				return
			}
			s.setStatus(fmt.Sprintf(t("status.continued"), name))
			s.refreshOneStatus(name)
		}()
	})

	s.btnKill = common.NewHoverableButton(t("btn.kill"), theme.DeleteIcon(), t("tooltip.serman.kill"), s.statusBar, func() {
		list := s.filtered()
		if s.selected < 0 || s.selected >= len(list) {
			return
		}
		name := list[s.selected].Name
		dialog.ShowConfirm(
			t("confirm.title"),
			fmt.Sprintf(t("confirm.kill"), name),
			func(ok bool) {
				if !ok {
					return
				}
				go func() {
					if err := s.backend.Kill(name); err != nil {
						s.setStatus(t("status.err") + err.Error())
						return
					}
					s.setStatus(fmt.Sprintf(t("status.killed"), name))
					st := s.backend.Status(name)
					s.updateRunningStatus(st)
				}()
			}, s.win)
	})
	s.btnKill.Importance = widget.DangerImportance

	s.btnEnable.Disable()
	s.btnDisable.Disable()
	s.btnStart.Disable()
	s.btnStop.Disable()
	s.btnRestart.Disable()
	s.btnHup.Disable()
	s.btnPause.Disable()
	s.btnContinue.Disable()
	s.btnKill.Disable()

	toggleRow := container.NewHBox(s.btnEnable, s.btnDisable)
	controlRow := container.NewHBox(s.btnStart, s.btnStop, s.btnRestart, s.btnHup, s.btnPause, s.btnContinue, s.btnKill, layout.NewSpacer(), btnReload)
	buttons := container.NewVBox(toggleRow, controlRow)

	// About button — info icon at the bottom-left corner
	btnAbout := common.NewHoverableButton("", theme.InfoIcon(), t("tooltip.serman.about"), s.statusBar, func() { s.showAbout() })
	btnAbout.Importance = widget.LowImportance
	statusBar := container.NewHBox(btnAbout, layout.NewSpacer(), s.statusBar)

	// ── Dir info ─────────────────────────────────────────────────────
	svcDir, destDir := s.backend.Dirs()
	dirInfo := widget.NewLabel(fmt.Sprintf("SERVICEDIR=%s\nSERVICEDESTDIR=%s", svcDir, destDir))
	dirInfo.TextStyle = fyne.TextStyle{Monospace: true}

	detailTitle := canvas.NewText(t("detail.title"), color.NRGBA{R: 0x00, G: 0xb8, B: 0xd4, A: 0xcc})
	detailTitle.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
	configTitle := canvas.NewText(t("config.title"), colorMuted)
	configTitle.TextStyle = fyne.TextStyle{Monospace: true}

	reloadHint := widget.NewLabel(t("config.reload_hint"))
	reloadHint.TextStyle = fyne.TextStyle{Italic: true}
	reloadHint.Importance = widget.WarningImportance

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
		widget.NewSeparator(),
		reloadHint,
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
	applyFilter(FilterAll)

	return root
}

// ── Standalone runner ────────────────────────────────────────────────

// RunGUI runs svman as a standalone Fyne GUI application.
func RunGUI(serviceDir, serviceDestDir string) {
	InitI18n()
	a := app.New()
	a.Settings().SetTheme(darkIndustrialTheme{theme.DefaultTheme()})
	win := a.NewWindow(t("app.window"))
	b := NewRunitBackend(serviceDir, serviceDestDir)
	g := &guiApp{
		win:      win,
		backend:  b,
		selected: -1,
	}
	g.services = b.List()
	win.SetContent(g.buildContent(true))
	win.Resize(fyne.NewSize(860, 560))
	win.SetMaster()
	win.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		if e.Name == fyne.KeyEscape {
			a.Quit()
		}
	})
	win.ShowAndRun()
}
