//go:build !tui_only

package vmman

import (
	"fmt"
	"image/color"
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

var (
	colorRunning = color.NRGBA{R: 0x22, G: 0xaa, B: 0x55, A: 0xff}
	colorStopped = color.NRGBA{R: 0xff, G: 0x55, B: 0x55, A: 0xff}
	colorMuted   = color.NRGBA{R: 0x55, G: 0x5a, B: 0x60, A: 0xff}
)

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
		return colorRunning
	case theme.ColorNameError:
		return colorStopped
	}
	return th.Theme.Color(name, variant)
}

type guiApp struct {
	win        fyne.Window
	backend    Backend
	vms        []VM
	selected   int
	searchText string
	filter     FilterMode

	vmList      *widget.List
	detailName  *widget.Label
	detailState *widget.Label
	detailPID   *widget.Label
	detailPort  *widget.Label
	btnBoot     *common.HoverableButton
	btnKill     *common.HoverableButton
	btnConnect  *common.HoverableButton
	btnAbout    *common.HoverableButton
	statusBar   *common.StatusBar
	countLabel  *widget.Label
}

func (s *guiApp) filtered() []VM {
	return Filter(s.vms, s.filter, s.searchText,
		func(vm VM) bool { return vm.Running },
		func(vm VM, q string) bool {
			return strings.Contains(strings.ToLower(vm.Name), q)
		},
	)
}

func (s *guiApp) reload() {
	fyne.Do(func() {
		s.vms = s.backend.List()
		s.vmList.Refresh()
		s.updateCount()
		list := s.filtered()
		if s.selected >= 0 && s.selected < len(list) {
			s.showDetail(list[s.selected])
		} else {
			s.selected = -1
			s.clearDetail()
		}
	})
}

func (s *guiApp) updateCount() {
	running := 0
	for _, vm := range s.vms {
		if vm.Running {
			running++
		}
	}
	s.countLabel.SetText(fmt.Sprintf(t("stats.fmt"), running, len(s.vms), len(s.filtered())))
}

func (s *guiApp) clearDetail() {
	s.detailName.SetText(t("detail.empty"))
	s.detailState.SetText(t("detail.empty"))
	s.detailPID.SetText(t("detail.empty"))
	s.detailPort.SetText(t("detail.empty"))
	s.btnBoot.Disable()
	s.btnKill.Disable()
	s.btnConnect.Disable()
}

func (s *guiApp) showDetail(vm VM) {
	s.detailName.SetText(vm.Name)
	if vm.Running {
		s.detailState.SetText(t("state.running"))
		s.detailState.Importance = widget.SuccessImportance
		s.btnBoot.Disable()
		s.btnKill.Enable()
		s.btnConnect.Enable()
		if vm.PID > 0 {
			s.detailPID.SetText(fmt.Sprintf("%d", vm.PID))
		}
		if vm.SPICEPort > 0 {
			s.detailPort.SetText(fmt.Sprintf("%d", vm.SPICEPort))
		}
	} else {
		s.detailState.SetText(t("state.stopped"))
		s.detailState.Importance = widget.DangerImportance
		s.btnBoot.Enable()
		s.btnKill.Disable()
		s.btnConnect.Disable()
		s.detailPID.SetText(t("detail.empty"))
		s.detailPort.SetText(t("detail.empty"))
	}
}

func (s *guiApp) buildContent() fyne.CanvasObject {
	header := canvas.NewText("VMman - Viewer Manager", color.NRGBA{R: 0x00, G: 0xb8, B: 0xd4, A: 0xff})
	header.TextStyle = fyne.TextStyle{Bold: true}

	search := widget.NewEntry()
	search.SetPlaceHolder(t("search.placeholder"))
	search.OnChanged = func(text string) {
		s.searchText = text
		s.vmList.Refresh()
	}

	filterAll := common.NewHoverableButtonText(t("filter.all"), t("tooltip.vmsman.filter_all"), s.statusBar, func() { s.applyFilter(FilterAll) })
	filterRunning := common.NewHoverableButtonText(t("filter.running"), t("tooltip.vmsman.filter_running"), s.statusBar, func() { s.applyFilter(FilterRunning) })
	filterStopped := common.NewHoverableButtonText(t("filter.stopped"), t("tooltip.vmsman.filter_stopped"), s.statusBar, func() { s.applyFilter(FilterStopped) })
	filterRow := container.NewHBox(filterAll, filterRunning, filterStopped)

	s.countLabel = widget.NewLabel("")
	s.countLabel.Alignment = fyne.TextAlignCenter

	s.vmList = widget.NewList(
		func() int { return len(s.filtered()) },
		func() fyne.CanvasObject {
			return widget.NewLabel("VM Name")
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			if lbl, ok := obj.(*widget.Label); ok {
				vm := s.filtered()[i]
				if vm.Running {
					lbl.SetText("[▶] " + vm.Name)
				} else {
					lbl.SetText("[■] " + vm.Name)
				}
			}
		},
	)
	s.vmList.OnSelected = func(id widget.ListItemID) {
		s.selected = id
		list := s.filtered()
		if id < len(list) {
			s.showDetail(list[id])
		}
	}

	detailTitle := canvas.NewText(t("detail.header"), color.NRGBA{R: 0x00, G: 0xb8, B: 0xd4, A: 0xff})
	detailTitle.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}

	s.detailName = widget.NewLabel(t("detail.empty"))
	s.detailState = widget.NewLabel(t("detail.empty"))
	s.detailPID = widget.NewLabel(t("detail.empty"))
	s.detailPort = widget.NewLabel(t("detail.empty"))

	detailForm := container.NewVBox(
		widget.NewLabel(t("detail.name")+":"), s.detailName,
		widget.NewLabel(t("detail.state")+":"), s.detailState,
		widget.NewLabel(t("detail.pid")+":"), s.detailPID,
		widget.NewLabel(t("detail.spice")+":"), s.detailPort,
	)

	s.statusBar = common.NewStatusBar()
	s.statusBar.TextStyle = fyne.TextStyle{Italic: true, Monospace: true}

	s.btnBoot = common.NewHoverableButton(t("btn.boot"), theme.MediaPlayIcon(), t("tooltip.vmsman.boot"), s.statusBar, func() {
		vm := s.selectedVM()
		if vm != nil {
			go func() {
				if err := s.backend.Boot(vm); err != nil {
					s.setStatus(t("status.err") + err.Error())
				} else {
					s.setStatus(t("status.boot"))
					s.reload()
				}
			}()
		}
	})
	s.btnBoot.Disable()

	s.btnKill = common.NewHoverableButton(t("btn.kill"), theme.DeleteIcon(), t("tooltip.vmsman.kill"), s.statusBar, func() {
		vm := s.selectedVM()
		if vm != nil {
			dialog.ShowConfirm(t("confirm.title"), fmt.Sprintf("Kill %s?", vm.Name), func(ok bool) {
				if ok {
					go func() {
						if err := s.backend.Kill(vm); err != nil {
							s.setStatus(t("status.err") + err.Error())
						} else {
							s.setStatus(t("status.kill"))
							s.reload()
						}
					}()
				}
			}, s.win)
		}
	})
	s.btnKill.Importance = widget.DangerImportance
	s.btnKill.Disable()

	s.btnConnect = common.NewHoverableButton(t("btn.connect"), theme.NewSuccessThemedResource(theme.ConfirmIcon()), t("tooltip.vmsman.connect"), s.statusBar, func() {
		vm := s.selectedVM()
		if vm != nil && vm.SPICEPort > 0 {
			if err := ConnectToVM(vm.SPICEPort, "remote-viewer"); err != nil {
				s.setStatus(t("status.err") + err.Error())
			} else {
				s.setStatus(t("status.connected"))
			}
		}
	})
	s.btnConnect.Importance = widget.SuccessImportance
	s.btnConnect.Disable()

	s.btnAbout = common.NewHoverableButton("", theme.InfoIcon(), t("tooltip.vmsman.about"), s.statusBar, func() {
		common.ShowAbout(common.AboutConfig{
			Win:       s.win,
			Title:     t("app.title"),
			Subtitle:  t("app.subtitle"),
			Version:   Version,
			Author:    AppAuthor,
			License:   AppLicense,
			URL:       AppURL,
			DialogBtn: t("btn.about"),
			CloseBtn:  t("btn.close"),
		})
	})
	s.btnAbout.Importance = widget.LowImportance

	buttonRow := container.NewHBox(s.btnBoot, s.btnKill, s.btnConnect, layout.NewSpacer())

	rightPanel := container.NewVBox(
		detailTitle,
		widget.NewSeparator(),
		detailForm,
		widget.NewSeparator(),
		buttonRow,
		layout.NewSpacer(),
		s.statusBar,
	)

	leftTop := container.NewVBox(header, widget.NewSeparator(), search, filterRow, s.countLabel, widget.NewSeparator())
	leftPanel := container.NewBorder(leftTop, nil, nil, nil, s.vmList)

	split := container.NewHSplit(container.NewPadded(leftPanel), container.NewPadded(rightPanel))
	split.SetOffset(0.42)

	statusBarRow := container.NewHBox(s.btnAbout, layout.NewSpacer(), s.statusBar)

	return container.NewBorder(
		nil,
		container.NewVBox(widget.NewSeparator(), statusBarRow),
		nil, nil,
		split,
	)
}

func (s *guiApp) buildContentWithHeader() fyne.CanvasObject {
	return container.NewBorder(
		nil, nil, nil, nil,
		s.buildContent(),
	)
}

func (s *guiApp) selectedVM() *VM {
	list := s.filtered()
	if s.selected < 0 || s.selected >= len(list) {
		return nil
	}
	return &list[s.selected]
}

func (s *guiApp) applyFilter(mode FilterMode) {
	s.filter = mode
	s.selected = -1
	s.vmList.Refresh()
	s.updateCount()
	s.clearDetail()
}

func (s *guiApp) setStatus(msg string) {
	fyne.Do(func() {
		s.statusBar.SetText(msg)
	})
}

// RunGUI runs the GUI.
func RunGUI(vmDir string) {
	InitI18n()
	a := app.New()
	a.Settings().SetTheme(darkIndustrialTheme{theme.DefaultTheme()})
	win := a.NewWindow(t("app.window"))
	common.SetWindowIcon(win)
	b := NewQEMUBackend(vmDir)
	g := &guiApp{
		win:     win,
		backend: b,
	}
	g.vms = b.List()
	win.SetContent(g.buildContentWithHeader())
	win.Resize(fyne.NewSize(860, 560))
	win.SetMaster()
	win.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		if e.Name == fyne.KeyEscape {
			a.Quit()
		}
	})
	win.ShowAndRun()
}
