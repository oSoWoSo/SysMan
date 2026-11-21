//go:build !tui_only

package vmman

import (
	"context"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

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
	distro     *DistroApp
	vms        []VM
	selected   int
	searchText string
	filter     FilterMode

	// download tab
	supportedSystems []OperatingSystem
	filteredSystems  []OperatingSystem
	busy             bool
	workers          sync.WaitGroup
	viewMode         string
	supportedGrid    *fyne.Container
	supportedIcons   *container.Scroll
	supportedList    *widget.List
	iconCache        map[string]fyne.Resource
	downloadSearch   *widget.Entry

	vmList      *widget.List
	vmGrid      *fyne.Container
	vmIcons     *container.Scroll
	detailName  *widget.Label
	detailState *widget.Label
	detailPID   *widget.Label
	detailPort  *widget.Label
	detailSSH   *widget.Label
	btnBoot     *common.HoverableButton
	btnKill     *common.HoverableButton
	btnConnect  *common.HoverableButton
	btnCustom   *common.HoverableButton
	btnEdit     *common.HoverableButton
	btnLog      *common.HoverableButton
	btnSSHUser  *common.HoverableButton
	btnDelete   *common.HoverableButton
	btnAbout    *common.HoverableButton
	statusBar   *common.StatusBar
	countLabel  *widget.Label
	logScroll   *container.Scroll
	logText     *widget.RichText
	logBuf      strings.Builder
	prevLogVM   string
	root        fyne.CanvasObject
	bottomBar   fyne.CanvasObject
	stopRefresh chan struct{}
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
		s.refreshVMGrid()
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

// startAutoRefresh polls VM state periodically so VMs started or stopped
// outside vmsman (or moments after a boot) show up without manual reloads.
func (s *guiApp) startAutoRefresh() {
	if s.stopRefresh != nil {
		close(s.stopRefresh)
	}
	s.stopRefresh = make(chan struct{})
	go func() {
		t := time.NewTicker(3 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				s.reload()
			case <-s.stopRefresh:
				return
			}
		}
	}()
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
	s.btnDelete.Disable()
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
	s.btnDelete.Enable()
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

	filterAll := common.NewHoverableButtonText(t("filter.all"), t("tooltip.vmsman.filter_all"), s.statusBar, func() { s.applyFilter(FilterAll) })
	filterRunning := common.NewHoverableButtonText(t("filter.running"), t("tooltip.vmsman.filter_running"), s.statusBar, func() { s.applyFilter(FilterRunning) })
	filterStopped := common.NewHoverableButtonText(t("filter.stopped"), t("tooltip.vmsman.filter_stopped"), s.statusBar, func() { s.applyFilter(FilterStopped) })
	s.countLabel = widget.NewLabel("")
	filterRow := container.NewHBox(filterAll, filterRunning, filterStopped, layout.NewSpacer(), s.countLabel)
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
	s.btnBoot.Importance = widget.SuccessImportance
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

	s.btnDelete = common.NewHoverableButton(t("btn.delete"), theme.DeleteIcon(), t("tooltip.vmsman.delete"), s.statusBar, func() {
		vm := s.selectedVM()
		if vm == nil {
			return
		}
		dialog.ShowConfirm(t("confirm.title"), fmt.Sprintf("Delete %s?", vm.Name), func(ok bool) {
			if !ok {
				return
			}
			if vm.Running {
				if err := s.backend.Kill(vm); err != nil {
					s.setStatus(t("status.err") + err.Error())
				}
			}
			// Remove config file and known state files.
			_ = os.Remove(vm.Config)
			vmDir := s.backend.VMDir()
			for _, name := range []string{vm.Name + ".qcow2", vm.Name + ".log", vm.Name + ".pid", vm.Name + ".ports", vm.Name + ".spice", vm.Name + ".sock"} {
				_ = os.Remove(filepath.Join(vmDir, name))
			}
			// Remove per-VM state directory if present.
			_ = os.RemoveAll(filepath.Join(vmDir, vm.Name))
			s.setStatus(fmt.Sprintf("Deleted %s", vm.Name))
			s.selected = -1
			s.reload()
		}, s.win)
	})
	s.btnDelete.Importance = widget.DangerImportance
	s.btnDelete.Disable()

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

	cfg := common.LoadSysManConfig()
	btnSettings := widget.NewButtonWithIcon("", theme.SettingsIcon(), func() {
		common.ShowSettingsDialog(s.win, "vmsman - virtual machines manager settings", []common.SettingsField{
			{Label: "VM dir", Value: cfg.Vmsman.VMDir, Placeholder: DefaultVMDir},
			{Label: "Source dir", Value: cfg.Vmsman.SourceDir, Placeholder: ""},
			{Label: "RAM disk", Value: cfg.Vmsman.RAMDisk, Placeholder: "no", Type: "bool"},
			{Label: "RAM dir", Value: cfg.Vmsman.RAMDir, Placeholder: "/dev/shm/quickemu"},
			{Label: "ISO dir", Value: cfg.Vmsman.ISODir, Placeholder: ""},
			{Label: "Templates", Value: cfg.Vmsman.Templates, Placeholder: "no", Type: "bool"},
			{Label: "Display mode", Value: cfg.Vmsman.DisplayMode, Placeholder: "Combined", Type: "select", Options: []string{viewCombined, viewIcons, viewText}},
		}, func(values map[string]string) {
			c := common.LoadSysManConfig()
			c.Vmsman.VMDir = values["VM dir"]
			c.Vmsman.SourceDir = values["Source dir"]
			c.Vmsman.RAMDisk = values["RAM disk"]
			c.Vmsman.RAMDir = values["RAM dir"]
			c.Vmsman.ISODir = values["ISO dir"]
			c.Vmsman.Templates = values["Templates"]
			c.Vmsman.DisplayMode = values["Display mode"]
			if err := common.SaveSysManConfig(c); err != nil {
				common.ShowSettingsError(s.win, err)
				return
			}
			// Apply settings live so VM lists and downloads use the new paths.
			newVM := ResolveVMDir(values["VM dir"])
			if newVM == "" {
				newVM = ResolveVMDir("")
			}
			s.vmDir = newVM
			s.backend = NewQEMUBackend(newVM)
			d, _ := NewDistroApp(context.Background(), DistroConfig{
				VMs:       newVM,
				SourceDir: values["Source dir"],
				RAMDisk:   values["RAM disk"],
				RAMDir:    values["RAM dir"],
				ISOs:      values["ISO dir"],
				Templates: values["Templates"],
			})
			if d != nil {
				s.distro = d
			}
			s.viewMode = validView(values["Display mode"])
			s.setVMView(s.viewMode)
			s.setView(s.viewMode)
			s.reload()
		})
	})
	btnSettings.Importance = widget.LowImportance

	buttonRow := container.NewHBox(s.btnBoot, s.btnKill, s.btnConnect, s.btnEdit, s.btnLog, s.btnSSHUser, s.btnDelete, layout.NewSpacer())

	detailTop := container.NewVBox(
		detailForm,
		widget.NewSeparator(),
		buttonRow,
	)

	s.logText = widget.NewRichText()
	s.logText.Wrapping = fyne.TextWrapOff
	s.logScroll = container.NewScroll(s.logText)
	rightPanel := container.NewBorder(detailTop, nil, nil, nil, s.logScroll)

	cfg2 := common.LoadSysManConfig()
	if s.viewMode == "" {
		s.viewMode = validView(cfg2.Vmsman.DisplayMode)
		if s.viewMode == "" {
			s.viewMode = validView(fyne.CurrentApp().Preferences().StringWithFallback("display-mode", viewCombined))
		}
	}

	s.vmGrid = container.NewGridWrap(fyne.NewSize(132, 120))
	s.vmIcons = container.NewVScroll(s.vmGrid)
	vmView := container.NewStack(s.vmList, s.vmIcons)

	leftTop := container.NewVBox(search, filterRow, widget.NewSeparator())
	leftPanel := container.NewBorder(leftTop, nil, nil, nil, vmView)

	s.setVMView(s.viewMode)

	split := container.NewHSplit(container.NewPadded(leftPanel), container.NewPadded(rightPanel))
	split.SetOffset(0.42)

	statusBarRow := container.NewHBox(btnSettings, s.btnAbout, layout.NewSpacer(), s.statusBar)
	s.bottomBar = container.NewVBox(widget.NewSeparator(), statusBarRow)

	s.root = split

	s.applyFilter(FilterAll)

	s.startAutoRefresh()

	return s.root
}

func (s *guiApp) selectedVM() *VM {
	list := s.filtered()
	if s.selected < 0 || s.selected >= len(list) {
		return nil
	}
	return &list[s.selected]
}

// setVMView toggles the Run VM tab between list and icon views.
func (s *guiApp) setVMView(mode string) {
	s.viewMode = validView(mode)
	fyne.CurrentApp().Preferences().SetString("display-mode", s.viewMode)
	cfg := common.LoadSysManConfig()
	cfg.Vmsman.DisplayMode = s.viewMode
	_ = common.SaveSysManConfig(cfg)
	if s.vmGrid == nil || s.vmIcons == nil || s.vmList == nil {
		return
	}
	if s.viewMode == viewText {
		s.vmIcons.Hide()
		s.vmList.Show()
	} else {
		s.vmList.Hide()
		s.vmIcons.Show()
		s.refreshVMGrid()
	}
}

// refreshVMGrid rebuilds the Run VM icon grid.
func (s *guiApp) refreshVMGrid() {
	if s.vmGrid == nil {
		return
	}
	list := s.filtered()
	objects := make([]fyne.CanvasObject, 0, len(list))
	for i, vm := range list {
		i := i
		vm := vm
		icon := theme.ComputerIcon()
		if vm.Running {
			icon = theme.NewSuccessThemedResource(theme.ComputerIcon())
		}
		var label string
		if vm.Running {
			label = "[▶] " + vm.Name
		} else {
			label = "[■] " + vm.Name
		}
		tile := newIconTile(label, icon, s.viewMode == viewCombined, func() {
			s.selected = i
			s.showDetail(vm)
		})
		objects = append(objects, tile)
	}
	s.vmGrid.Objects = objects
	s.vmGrid.Refresh()
}

func (s *guiApp) applyFilter(mode FilterMode) {
	s.filter = mode
	s.selected = -1
	s.vmList.Refresh()
	s.refreshVMGrid()
	s.updateCount()
	s.clearDetail()
}

func (s *guiApp) setStatus(msg string) {
	fyne.Do(func() {
		if s.statusBar != nil {
			s.statusBar.SetText(msg)
		}
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
	if env := os.Getenv("VMDIR"); env != "" {
		vmDir = ResolveVMDir(env)
	} else {
		cfg := common.LoadSysManConfig()
		if cfg.Vmsman.VMDir != "" {
			vmDir = ResolveVMDir(cfg.Vmsman.VMDir)
		} else if vmDir == "" {
			vmDir = ResolveVMDir("")
		} else {
			vmDir = ResolveVMDir(vmDir)
		}
	}
	b := NewQEMUBackend(vmDir)
	cfg := common.LoadSysManConfig()
	if cfg.Vmsman.VMDir == "" {
		cfg.Vmsman.VMDir = vmDir
	}
	d, _ := NewDistroApp(context.Background(), DistroConfig{
		VMs:       cfg.Vmsman.VMDir,
		SourceDir: cfg.Vmsman.SourceDir,
		RAMDisk:   cfg.Vmsman.RAMDisk,
		RAMDir:    cfg.Vmsman.RAMDir,
		ISOs:      cfg.Vmsman.ISODir,
		Templates: cfg.Vmsman.Templates,
		Terminal:  cfg.Vmsman.Terminal,
	})
	if d == nil || cfg.Vmsman.VMDir == "" {
		d, _ = NewDistroApp(context.Background(), DistroConfig{VMs: vmDir})
	}
	g := &guiApp{
		win:     win,
		backend: b,
		vmDir:   vmDir,
		distro:  d,
	}
	g.vms = b.List()
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("Run VM", theme.ComputerIcon(), g.buildContent()),
		container.NewTabItemWithIcon("Create VM", theme.DownloadIcon(), g.buildDownloadContent()),
	)
	if g.bottomBar == nil {
		g.bottomBar = container.NewVBox(widget.NewSeparator(), container.NewHBox(layout.NewSpacer(), g.statusBar))
	}
	g.root = container.NewBorder(nil, g.bottomBar, nil, nil, tabs)
	win.SetContent(g.root)
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
