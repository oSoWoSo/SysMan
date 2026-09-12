//go:build !tui_only

package vmman

import (
	"fmt"
	"image/color"
	"strings"

	"codeberg.org/oSoWoSo/SysMan/src/common"
	"fyne.io/fyne/v2"
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
	vmDir      string
	vms        []VM
	selected   int
	searchText string
	filter     FilterMode

	vmList      *widget.List
	detailName  *widget.Label
	detailState *widget.Label
	detailPID   *widget.Label
	detailPort  *widget.Label
	detailSSH   *widget.Label
	btnBoot     *common.HoverableButton
	btnKill     *common.HoverableButton
	btnConnect  *common.HoverableButton
	btnNew      *common.HoverableButton
	btnEdit     *common.HoverableButton
	btnLog      *common.HoverableButton
	btnSSHUser  *common.HoverableButton
	btnAbout    *common.HoverableButton
	statusBar   *common.StatusBar
	countLabel  *widget.Label
	logScroll   *container.Scroll
	logText     *widget.RichText
	logBuf      strings.Builder
	prevLogVM   string
	root        fyne.CanvasObject
	bottomBar   fyne.CanvasObject
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
		// Report an error if the VM directory is missing or inaccessible,
		// but leave any existing status message (hover text, action results) untouched.
		if errMsg := CheckVMDir(s.backend.VMDir()); errMsg != "" {
			s.statusBar.SetText(errMsg)
		}

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
	s.detailSSH.SetText(t("detail.empty"))
	s.btnBoot.Disable()
	s.btnKill.Disable()
	s.btnConnect.Disable()
	s.btnEdit.Disable()
	s.btnLog.Disable()
	s.btnSSHUser.Disable()
}

func (s *guiApp) showDetail(vm VM) {
	s.detailName.SetText(vm.Name)
	if vm.Running {
		s.detailState.Importance = widget.SuccessImportance
		s.detailState.SetText(t("state.running"))
		s.btnBoot.Disable()
		s.btnKill.Enable()
		s.btnConnect.Enable()
		if vm.PID > 0 {
			s.detailPID.SetText(fmt.Sprintf("%d", vm.PID))
		}
		if vm.SPICEPort > 0 {
			s.detailPort.SetText(fmt.Sprintf("%d", vm.SPICEPort))
		}
		if vm.SSHPort > 0 {
			s.detailSSH.SetText(fmt.Sprintf("%s@localhost:%d", sshUser(vm), vm.SSHPort))
		}
	} else {
		s.detailState.Importance = widget.DangerImportance
		s.detailState.SetText(t("state.stopped"))
		s.btnBoot.Enable()
		s.btnKill.Disable()
		s.btnConnect.Disable()
		s.detailPID.SetText(t("detail.empty"))
		s.detailPort.SetText(t("detail.empty"))
		s.detailSSH.SetText(t("detail.empty"))
	}
	s.btnEdit.Enable()
	s.btnLog.Enable()
	s.btnSSHUser.Enable()
	s.showLogFor(vm.Name)
}

// appendLog appends a line to the log view. Safe to call from goroutines.
func (s *guiApp) appendLog(line string) {
	fyne.Do(func() {
		s.logBuf.WriteString(line)
		s.logText.Segments = common.AnsiToRichSegments(s.logBuf.String())
		s.logText.Refresh()
		s.logScroll.ScrollToBottom()
	})
}

// showLogFor loads the on-disk <name>.log file into the log view.
func (s *guiApp) showLogFor(name string) {
	if name == s.prevLogVM {
		return
	}
	s.prevLogVM = name
	s.logBuf.Reset()
	s.logBuf.WriteString(readVMLog(s.backend.VMDir(), name))
	s.logText.Segments = common.AnsiToRichSegments(s.logBuf.String())
	s.logText.Refresh()
	s.logScroll.ScrollToBottom()
}

func (s *guiApp) buildContent() fyne.CanvasObject {
	if s.statusBar == nil {
		s.statusBar = common.NewStatusBar()
	}
	s.statusBar.TextStyle = fyne.TextStyle{Italic: true, Monospace: true}

	search := widget.NewEntry()
	search.SetPlaceHolder(t("search.placeholder"))
	search.OnChanged = func(text string) {
		s.searchText = text
		s.vmList.Refresh()
	}

	s.btnNew = common.NewHoverableButton(t("btn.new"), theme.ContentAddIcon(), t("tooltip.vmsman.new"), s.statusBar, func() {
		s.showCreateView()
	})
	s.btnNew.Importance = widget.SuccessImportance

	filterAll := common.NewHoverableButtonText(t("filter.all"), t("tooltip.vmsman.filter_all"), s.statusBar, func() { s.applyFilter(FilterAll) })
	filterRunning := common.NewHoverableButtonText(t("filter.running"), t("tooltip.vmsman.filter_running"), s.statusBar, func() { s.applyFilter(FilterRunning) })
	filterStopped := common.NewHoverableButtonText(t("filter.stopped"), t("tooltip.vmsman.filter_stopped"), s.statusBar, func() { s.applyFilter(FilterStopped) })
	filterRow := container.NewHBox(s.btnNew, filterAll, filterRunning, filterStopped)

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

	s.detailName = widget.NewLabel(t("detail.empty"))
	s.detailName.Selectable = true
	s.detailState = widget.NewLabel(t("detail.empty"))
	s.detailState.Selectable = true
	s.detailPID = widget.NewLabel(t("detail.empty"))
	s.detailPID.Selectable = true
	s.detailPort = widget.NewLabel(t("detail.empty"))
	s.detailPort.Selectable = true
	s.detailSSH = widget.NewLabel(t("detail.empty"))
	s.detailSSH.Selectable = true

	detailForm := container.NewVBox(
		container.NewHBox(widget.NewLabel(t("detail.name")+":"), layout.NewSpacer(), s.detailName),
		container.NewHBox(widget.NewLabel(t("detail.state")+":"), layout.NewSpacer(), s.detailState),
		container.NewHBox(widget.NewLabel(t("detail.pid")+":"), layout.NewSpacer(), s.detailPID),
		container.NewHBox(widget.NewLabel(t("detail.spice")+":"), layout.NewSpacer(), s.detailPort),
		container.NewHBox(widget.NewLabel(t("detail.ssh")+":"), layout.NewSpacer(), s.detailSSH),
	)

	s.btnBoot = common.NewHoverableButton(t("btn.boot"), theme.MediaPlayIcon(), t("tooltip.vmsman.boot"), s.statusBar, func() {
		vm := s.selectedVM()
		if vm != nil {
			// Reset the log view so boot output starts fresh.
			s.prevLogVM = ""
			s.logBuf.Reset()
			s.logText.Segments = nil
			s.logText.Refresh()
			go func() {
				if err := s.backend.BootStream(vm, s.appendLog); err != nil {
					s.setStatus(t("status.err") + err.Error())
				} else {
					s.setStatus(t("status.boot"))
					go s.reload()
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
		if vm != nil {
			if vm.SPICEPort > 0 {
				if err := ConnectToVM(vm.SPICEPort, "remote-viewer"); err != nil {
					s.setStatus(t("status.err") + err.Error())
					return
				}
			} else if vm.SSHPort > 0 {
				if err := ConnectToVMSSH(vm.SSHPort, vm.SSHUser); err != nil {
					s.setStatus(t("status.err") + err.Error())
					return
				}
			} else {
				return
			}
			s.setStatus(t("status.connected"))
		}
	})
	s.btnConnect.Importance = widget.SuccessImportance
	s.btnConnect.Disable()

	s.btnEdit = common.NewHoverableButton(t("btn.edit"), theme.DocumentCreateIcon(), t("tooltip.vmsman.edit"), s.statusBar, func() {
		vm := s.selectedVM()
		if vm != nil {
			s.showEditConfigDialog(vm)
		}
	})
	s.btnEdit.Disable()

	s.btnLog = common.NewHoverableButton(t("btn.log"), theme.DocumentIcon(), t("tooltip.vmsman.log"), s.statusBar, func() {
		if vm := s.selectedVM(); vm != nil {
			s.showLogFor(vm.Name)
		}
	})
	s.btnLog.Disable()

	s.btnSSHUser = common.NewHoverableButton(t("btn.ssh_user"), theme.AccountIcon(), t("tooltip.vmsman.ssh_user"), s.statusBar, func() {
		if vm := s.selectedVM(); vm != nil {
			s.showSSHUserDialog(vm)
		}
	})
	s.btnSSHUser.Disable()

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

	btnSettings := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		common.ShowSettingsDialog(s.win, t("app.window"), []common.SettingsField{
			{Label: "VM dir", Value: s.vmDir, Placeholder: DefaultVMDir},
		}, func(values map[string]string) {
			cfg := common.LoadSysManConfig()
			cfg.Vmsman.VMDir = values["VM dir"]
			if err := common.SaveSysManConfig(cfg); err != nil {
				common.ShowSettingsError(s.win, err)
			}
		})
	})
	btnSettings.Importance = widget.LowImportance

	buttonRow := container.NewHBox(s.btnBoot, s.btnKill, s.btnConnect, s.btnEdit, s.btnLog, s.btnSSHUser, layout.NewSpacer())

	detailTop := container.NewVBox(
		detailForm,
		widget.NewSeparator(),
		buttonRow,
	)

	s.logText = widget.NewRichText()
	s.logText.Wrapping = fyne.TextWrapOff
	s.logScroll = container.NewScroll(s.logText)
	rightPanel := container.NewBorder(detailTop, nil, nil, nil, s.logScroll)

	leftTop := container.NewVBox(search, filterRow, s.countLabel, widget.NewSeparator())
	leftPanel := container.NewBorder(leftTop, nil, nil, nil, s.vmList)

	split := container.NewHSplit(container.NewPadded(leftPanel), container.NewPadded(rightPanel))
	split.SetOffset(0.42)

	statusBarRow := container.NewHBox(btnSettings, s.btnAbout, layout.NewSpacer(), s.statusBar)
	s.bottomBar = container.NewVBox(widget.NewSeparator(), statusBarRow)

	s.root = container.NewBorder(
		nil,
		s.bottomBar,
		nil, nil,
		split,
	)

	s.applyFilter(FilterAll)

	return s.root
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

// showCreateView switches the whole window to the "create VM" form. Cancel or
// a successful create returns to the main view.
func (s *guiApp) showCreateView() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder(t("new.name_ph"))
	guestOSEntry := widget.NewEntry()
	guestOSEntry.SetPlaceHolder(t("new.guest_os_ph"))
	isoEntry := widget.NewEntry()
	isoEntry.SetPlaceHolder(t("new.iso_ph"))
	memoryEntry := widget.NewEntry()
	memoryEntry.SetPlaceHolder("2048")
	coresEntry := widget.NewEntry()
	coresEntry.SetPlaceHolder("2")
	sshUserEntry := widget.NewEntry()
	sshUserEntry.SetPlaceHolder(t("new.ssh_user_ph"))

	form := &widget.Form{
		Items: []*widget.FormItem{
			widget.NewFormItem(t("new.name"), nameEntry),
			widget.NewFormItem(t("new.guest_os"), guestOSEntry),
			widget.NewFormItem(t("new.iso"), isoEntry),
			widget.NewFormItem(t("new.memory"), memoryEntry),
			widget.NewFormItem(t("new.cores"), coresEntry),
			widget.NewFormItem(t("new.ssh_user"), sshUserEntry),
		},
		SubmitText: t("btn.create"),
		CancelText: t("btn.cancel"),
		OnSubmit: func() {
			name := strings.TrimSpace(nameEntry.Text)
			if name == "" {
				s.setStatus(t("status.err") + t("new.err_name"))
				return
			}
			cfg := VMCreateConfig{
				Name:    name,
				GuestOS: strings.TrimSpace(guestOSEntry.Text),
				ISO:     strings.TrimSpace(isoEntry.Text),
				SSHUser: strings.TrimSpace(sshUserEntry.Text),
			}
			if cfg.GuestOS == "" {
				cfg.GuestOS = "linux"
			}
			_, _ = fmt.Sscanf(memoryEntry.Text, "%d", &cfg.MemoryMB)
			_, _ = fmt.Sscanf(coresEntry.Text, "%d", &cfg.CPUCores)
			if err := s.backend.Create(cfg); err != nil {
				s.setStatus(t("status.err") + err.Error())
				return
			}
			s.setStatus(fmt.Sprintf(t("status.created"), cfg.Name))
			s.showMainView()
			s.reload()
		},
		OnCancel: func() { s.showMainView() },
	}

	title := canvas.NewText(t("new.title"), color.NRGBA{R: 0x00, G: 0xb8, B: 0xd4, A: 0xff})
	title.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}

	center := container.NewPadded(container.NewVBox(title, widget.NewSeparator(), form))
	s.win.SetContent(container.NewBorder(nil, s.bottomBar, nil, nil, center))
}

// showMainView restores the main content after the create view.
func (s *guiApp) showMainView() {
	if s.root == nil {
		return
	}
	s.win.SetContent(s.root)
}

// showEditConfigDialog opens an editor for the selected VM's .conf file.
func (s *guiApp) showEditConfigDialog(vm *VM) {
	content, err := s.backend.ReadConfig(vm)
	if err != nil {
		s.setStatus(t("status.err") + err.Error())
		return
	}
	editor := widget.NewMultiLineEntry()
	editor.SetText(content)
	editor.Wrapping = fyne.TextWrapOff
	editor.TextStyle = fyne.TextStyle{Monospace: true}
	scroll := container.NewScroll(editor)
	scroll.SetMinSize(fyne.NewSize(520, 380))

	d := dialog.NewCustomConfirm(fmt.Sprintf(t("edit.title"), vm.Name), t("btn.save"), t("btn.cancel"), scroll,
		func(ok bool) {
			if !ok {
				return
			}
			if err := s.backend.WriteConfig(vm, editor.Text); err != nil {
				s.setStatus(t("status.err") + err.Error())
				return
			}
			s.setStatus(fmt.Sprintf(t("status.saved"), vm.Name))
			s.reload()
		}, s.win)
	d.Resize(fyne.NewSize(560, 440))
	d.Show()
}

// showSSHUserDialog lets the user set the SSH login user for a single VM,
// stored as ssh_user="..." in the VM's config file.
func (s *guiApp) showSSHUserDialog(vm *VM) {
	entry := widget.NewEntry()
	entry.SetText(vm.SSHUser)
	entry.SetPlaceHolder(t("ssh_user.empty"))
	f := dialog.NewForm(fmt.Sprintf(t("ssh_user.title"), vm.Name), t("btn.save"), t("btn.cancel"),
		[]*widget.FormItem{widget.NewFormItem(t("btn.ssh_user"), entry)},
		func(ok bool) {
			if !ok {
				return
			}
			user := strings.TrimSpace(entry.Text)
			if user != "" && !ValidVMName(user) {
				s.setStatus(t("status.err") + t("new.err_name"))
				return
			}
			if err := SetSSHUser(vm, user); err != nil {
				s.setStatus(t("status.err") + err.Error())
				return
			}
			s.setStatus(fmt.Sprintf(t("status.saved"), vm.Name))
			s.reload()
		}, s.win)
	f.Resize(fyne.NewSize(420, 200))
	f.Show()
}

// RunGUI runs the GUI.
func RunGUI(vmDir string) {
	InitI18n()
	a := common.NewApp(t("app.window"))
	a.Settings().SetTheme(darkIndustrialTheme{theme.DefaultTheme()})
	win := a.NewWindow(t("app.window"))
	common.SetWindowIcon(win)
	b := NewQEMUBackend(vmDir)
	g := &guiApp{
		win:     win,
		backend: b,
		vmDir:   vmDir,
	}
	g.vms = b.List()
	win.SetContent(g.buildContentWithHeader())
	g.reload()
	win.Resize(fyne.NewSize(860, 560))
	win.SetMaster()
	win.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		if e.Name == fyne.KeyEscape {
			a.Quit()
		}
	})
	win.ShowAndRun()
}
