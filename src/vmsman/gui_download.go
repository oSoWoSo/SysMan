//go:build !tui_only

package vmman

import (
	"context"
	"fmt"
	"strings"

	"codeberg.org/oSoWoSo/SysMan/src/common"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// buildDownloadContent builds the "Download VM" tab.
func (s *guiApp) buildDownloadContent() fyne.CanvasObject {
	if s.iconCache == nil {
		s.iconCache = make(map[string]fyne.Resource)
	}
	if s.distro == nil {
		d, _ := NewDistroApp(context.Background(), DistroConfig{VMs: s.vmDir})
		s.distro = d
	}

	search := widget.NewEntry()
	search.SetPlaceHolder("Search operating systems")
	s.downloadSearch = search

	s.btnCustom = common.NewHoverableButton(t("btn.custom"), theme.ContentAddIcon(), t("tooltip.vmsman.custom"), s.statusBar, func() {
		s.showCreateView()
	})
	s.btnCustom.Importance = widget.SuccessImportance
	search.OnChanged = func(q string) {
		s.filteredSystems = nil
		for _, system := range s.supportedSystems {
			if containsFold(system.Name, q) || containsFold(system.ID, q) {
				s.filteredSystems = append(s.filteredSystems, system)
			}
		}
		s.refreshDownloadGrids()
	}

	renewBtn := widget.NewButton("renew", func() {
		s.refreshDownloads(true, "")
	})

	cfg := common.LoadSysManConfig()
	if s.viewMode == "" {
		s.viewMode = validView(cfg.Vmsman.DisplayMode)
	}

	s.supportedList = widget.NewList(func() int {
		list := s.filteredSystems
		if list == nil {
			list = s.supportedSystems
		}
		return len(list)
	}, func() fyne.CanvasObject { return widget.NewLabel("") }, func(id widget.ListItemID, o fyne.CanvasObject) {
		list := s.filteredSystems
		if list == nil {
			list = s.supportedSystems
		}
		if id < len(list) {
			o.(*widget.Label).SetText(list[id].Name)
		}
	})
	s.supportedList.OnSelected = func(id widget.ListItemID) {
		list := s.filteredSystems
		if list == nil {
			list = s.supportedSystems
		}
		if id < len(list) {
			s.supportedList.UnselectAll()
			s.picker(list[id])
		}
	}

	s.supportedGrid = container.NewGridWrap(fyne.NewSize(132, 120))
	s.supportedIcons = container.NewVScroll(s.supportedGrid)

	supportedView := container.NewStack(s.supportedList, s.supportedIcons)

	s.setView(s.viewMode)
	s.refreshDownloadGrids()
	s.refreshDownloads(false, "")

	top := container.NewVBox(container.NewBorder(nil, nil, container.NewHBox(s.btnCustom, renewBtn), nil, search))
	return container.NewBorder(top, nil, nil, nil, supportedView)
}

func containsFold(s, q string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(q))
}

const (
	viewIcons    = "Icons only"
	viewText     = "Text only"
	viewCombined = "Combined"
)

func validView(mode string) string {
	switch mode {
	case viewIcons, viewText, viewCombined:
		return mode
	}
	return viewCombined
}

func (s *guiApp) setView(mode string) {
	s.viewMode = validView(mode)
	cfg := common.LoadSysManConfig()
	cfg.Vmsman.DisplayMode = s.viewMode
	_ = common.SaveSysManConfig(cfg)
	if s.supportedGrid == nil || s.supportedIcons == nil {
		return
	}
	if s.viewMode == viewText {
		s.supportedIcons.Hide()
		s.supportedList.Show()
	} else {
		s.supportedList.Hide()
		s.supportedIcons.Show()
	}
	s.refreshDownloadGrids()
}

// refreshDownloads loads (and optionally regenerates) supported/ready VM lists.
func (s *guiApp) refreshDownloads(regenerate bool, createOS string) {
	s.busy = true
	s.workers.Add(1)
	if s.statusBar != nil {
		s.statusBar.SetText("Renewing...")
	}
	go func() {
		defer s.workers.Done()
		d := s.distro
		var err error
		if regenerate {
			err = d.RenewSupported(func(done, total int, name string) {
				s.setStatus(fmt.Sprintf("Renewing %d/%d: %s", done, total, name))
			})
		} else {
			err = d.RenewReady()
		}
		supported, sysErr := d.Systems()
		if sysErr != nil && err == nil {
			err = sysErr
		}
		fyne.Do(func() {
			if err != nil {
				s.setStatus(t("status.err") + err.Error())
			} else {
				s.setStatus("Done")
			}
			s.supportedSystems = supported
			s.filteredSystems = append([]OperatingSystem(nil), supported...)
			s.refreshDownloadGrids()
			s.busy = false
		})
	}()
	_ = createOS
}

// picker opens the architecture/release/edition chooser for a supported OS.
func (s *guiApp) picker(system OperatingSystem) {
	if s.busy {
		dialog.ShowInformation("Busy", "Wait for the current operation to finish.", s.win)
		return
	}
	s.statusBar.SetText("Reading choices for " + system.Name + "...")
	w := fyne.CurrentApp().NewWindow("Create VM - " + system.Name)
	w.Resize(fyne.NewSize(560, 330))
	ctx, cancel := context.WithCancel(context.Background())
	_ = ctx
	status := widget.NewLabel("Loading architectures...")
	status.Wrapping = fyne.TextWrapWord
	status.Selectable = true
	arch := widget.NewSelect(nil, nil)
	release := widget.NewSelect(nil, nil)
	edition := widget.NewSelect(nil, nil)
	archRow := container.NewBorder(nil, nil, widget.NewLabel("Architecture"), nil, arch)
	releaseRow := container.NewBorder(nil, nil, widget.NewLabel("Release"), nil, release)
	editionRow := container.NewBorder(nil, nil, widget.NewLabel("Edition"), nil, edition)
	download := widget.NewButton("Download / create VM", nil)
	download.Disable()
	cancelBtn := widget.NewButton("Cancel", func() { w.Close() })

	var generation int
	var closed bool
	w.SetOnClosed(func() { closed = true; cancel() })

	var update func(int)
	update = func(level int) {
		generation++
		req := generation
		download.Disable()
		arch.Disable()
		release.Disable()
		edition.Disable()
		status.SetText("Reading metadata...")
		selectedArch := arch.Selected
		selectedRelease := release.Selected
		switch level {
		case 0:
			selectedArch = ""
			selectedRelease = ""
		case 1:
			selectedRelease = ""
		}
		s.workers.Add(1)
		go func() {
			defer s.workers.Done()
			m, err := s.distro.Query(system.ID, selectedArch, selectedRelease)
			fyne.Do(func() {
				if closed || generation != req {
					return
				}
				if err != nil {
					status.SetText(err.Error())
					arch.Enable()
					if len(release.Options) > 0 {
						release.Enable()
					}
					return
				}
				switch level {
				case 0:
					SetPickerOptions(arch, archRow, m.Architectures)
				case 1:
					SetPickerOptions(release, releaseRow, m.Releases)
				default:
					SetPickerOptions(edition, editionRow, m.Editions)
					edition.Enable()
				}
				arch.Enable()
				if level == 0 {
					update(1)
					return
				}
				if level == 1 && len(m.Releases) > 0 {
					update(2)
					return
				}
				release.Enable()
				if len(release.Options) == 0 {
					status.SetText("No releases available for this architecture.")
					w.Show()
					return
				}
				status.SetText(m.Description)
				if arch.Selected != "" && release.Selected != "" {
					download.Enable()
					if !PickerNeedsInput(arch.Options, release.Options, edition.Options) {
						download.OnTapped()
						return
					}
				}
				w.Show()
			})
		}()
	}

	arch.OnChanged = func(string) { update(1) }
	release.OnChanged = func(string) { update(2) }
	download.OnTapped = func() {
		sa, sr, se := arch.Selected, release.Selected, edition.Selected
		w.Close()
		s.download(system, sa, sr, se)
	}

	w.SetContent(container.NewBorder(nil, container.NewHBox(cancelBtn, download), nil, nil, container.NewVBox(archRow, releaseRow, editionRow, status)))
	update(0)
	w.Show()
}

// download runs qget for the chosen tuple and shows output.
func (s *guiApp) download(system OperatingSystem, arch, release, edition string) {
	if s.busy {
		dialog.ShowInformation("Busy", "Wait for the current operation to finish.", s.win)
		return
	}
	logWindow := fyne.CurrentApp().NewWindow("Creating " + system.Name)
	logWindow.Resize(fyne.NewSize(760, 500))
	output := widget.NewMultiLineEntry()
	output.Disable()
	scroll := container.NewVScroll(output)
	var text string
	cancelBtn := widget.NewButton("Cancel", func() { logWindow.Close() })
	logWindow.SetContent(container.NewBorder(nil, cancelBtn, nil, nil, scroll))
	logWindow.Show()

	s.busy = true
	s.workers.Add(1)
	go func() {
		defer s.workers.Done()
		err := s.distro.Download(system.ID, arch, release, edition, func(p []byte) {
			fyne.Do(func() {
				text += string(p)
				if len(text) > 128*1024 {
					text = text[len(text)-128*1024:]
				}
				output.SetText(text)
				scroll.ScrollToBottom()
			})
		})
		fyne.Do(func() {
			s.busy = false
			if err != nil {
				showCopyableError(s.win, err)
			} else {
				s.statusBar.SetText("Done")
			}
			s.refreshDownloads(false, "")
			cancelBtn.Disable()
		})
	}()
}

// refreshDownloadGrids rebuilds icon grids.
func (s *guiApp) refreshDownloadGrids() {
	build := func(grid *fyne.Container, systems []OperatingSystem, action func(OperatingSystem)) {
		objects := make([]fyne.CanvasObject, 0, len(systems))
		for _, system := range systems {
			system := system
			icon := s.iconCache[system.Icon]
			if icon == nil {
				icon = theme.ComputerIcon()
				if res, err := fyne.LoadResourceFromPath(system.Icon); err == nil {
					icon = res
				}
				s.iconCache[system.Icon] = icon
			}
			tile := newIconTile(system.Name, icon, s.viewMode == viewCombined, func() { action(system) })
			objects = append(objects, tile)
		}
		grid.Objects = objects
		grid.Refresh()
	}
	list := s.filteredSystems
	if list == nil {
		list = s.supportedSystems
	}
	if s.supportedGrid != nil {
		build(s.supportedGrid, list, func(system OperatingSystem) {
			s.picker(system)
		})
	}
}
