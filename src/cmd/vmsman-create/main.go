// Command vmsman-create is a small standalone VM creator used by generated
// supported .desktop files. It replaces the old YAD picker with a native
// Go/Fyne interface.
package main

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"

	"codeberg.org/oSoWoSo/SysMan/src/common"
	vmman "codeberg.org/oSoWoSo/SysMan/src/vmsman"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func main() {
	vmman.InitI18n()

	osID := ""
	if len(os.Args) >= 3 && os.Args[1] == "create" {
		osID = os.Args[2]
	} else if len(os.Args) >= 2 {
		osID = os.Args[1]
	}
	if osID == "" {
		fmt.Fprintln(os.Stderr, "usage: vmsman-create create <os>")
		os.Exit(1)
	}

	cfg := common.LoadSysManConfig()
	distroCfg := vmman.DistroConfig{
		VMs:       cfg.Vmsman.VMDir,
		SourceDir: cfg.Vmsman.SourceDir,
		RAMDisk:   cfg.Vmsman.RAMDisk,
		RAMDir:    cfg.Vmsman.RAMDir,
		ISOs:      cfg.Vmsman.ISODir,
		Templates: cfg.Vmsman.Templates,
		Terminal:  cfg.Vmsman.Terminal,
	}
	a, err := vmman.NewDistroApp(context.Background(), distroCfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fyneApp := app.NewWithID("org.osowoso.vmsman-create")
	w := fyneApp.NewWindow("Create VM - " + osID)
	w.Resize(fyne.NewSize(560, 420))

	status := widget.NewLabel("Loading choices...")
	status.Wrapping = fyne.TextWrapWord
	status.Selectable = true

	arch := widget.NewSelect(nil, nil)
	release := widget.NewSelect(nil, nil)
	edition := widget.NewSelect(nil, nil)

	archRow := container.NewBorder(nil, nil, widget.NewLabel("Architecture"), nil, arch)
	releaseRow := container.NewBorder(nil, nil, widget.NewLabel("Release"), nil, release)
	editionRow := container.NewBorder(nil, nil, widget.NewLabel("Edition"), nil, edition)

	downloadBtn := widget.NewButton("Download / create VM", nil)
	downloadBtn.Disable()
	cancelBtn := widget.NewButton("Cancel", func() { w.Close() })

	var selectedArch, selectedRelease string
	var generation int

	var update func(int)
	update = func(level int) {
		generation++
		req := generation
		downloadBtn.Disable()
		arch.Disable()
		release.Disable()
		edition.Disable()
		status.SetText("Reading metadata...")
		switch level {
		case 0:
			selectedArch = ""
			selectedRelease = ""
		case 1:
			selectedRelease = ""
		}
		go func() {
			m, err := a.Query(osID, selectedArch, selectedRelease)
			fyne.Do(func() {
				if generation != req {
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
					vmman.SetPickerOptions(arch, archRow, m.Architectures)
				case 1:
					vmman.SetPickerOptions(release, releaseRow, m.Releases)
				default:
					vmman.SetPickerOptions(edition, editionRow, m.Editions)
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
					return
				}
				status.SetText(m.Description)
				selectedArch = arch.Selected
				selectedRelease = release.Selected
				if arch.Selected != "" && release.Selected != "" {
					downloadBtn.Enable()
					if !vmman.PickerNeedsInput(arch.Options, release.Options, edition.Options) {
						downloadBtn.OnTapped()
					}
				}
			})
		}()
	}

	arch.OnChanged = func(v string) {
		selectedArch = v
		update(1)
	}
	release.OnChanged = func(v string) {
		selectedRelease = v
		update(2)
	}
	edition.OnChanged = func(_ string) {
	}

	var chosen bool
	var chosenArch, chosenRelease, chosenEdition string
	downloadBtn.OnTapped = func() {
		chosenArch, chosenRelease, chosenEdition = arch.Selected, release.Selected, edition.Selected
		chosen = true
		w.Close()
	}

	update(0)

	w.SetContent(container.NewBorder(nil, container.NewHBox(cancelBtn, downloadBtn), nil, nil, container.NewVBox(archRow, releaseRow, editionRow, status)))
	w.ShowAndRun()

	if chosen {
		runDownload(a, osID, chosenArch, chosenRelease, chosenEdition)
	}
}

func runDownload(a *vmman.DistroApp, osID, arch, release, edition string) {
	fyneApp := app.NewWithID("org.osowoso.vmsman-create")
	w := fyneApp.NewWindow("Creating " + osID)
	w.Resize(fyne.NewSize(760, 500))
	output := widget.NewMultiLineEntry()
	output.Disable()
	scroll := container.NewVScroll(output)
	progress := widget.NewProgressBar()
	progress.SetValue(0)
	var text string
	cancelBtn := widget.NewButton("Close", func() { w.Close() })
	w.SetContent(container.NewBorder(nil, container.NewVBox(cancelBtn, progress), nil, nil, scroll))
	w.Show()

	progressRe := regexp.MustCompile(`(\d+(?:\.\d+)?)%`)
	go func() {
		err := a.Download(osID, arch, release, edition, func(p []byte) {
			fyne.Do(func() {
				text += string(p)
				if len(text) > 128*1024 {
					text = text[len(text)-128*1024:]
				}
				if m := progressRe.FindAllStringSubmatch(string(p), -1); len(m) > 0 {
					if pct, perr := strconv.ParseFloat(m[len(m)-1][1], 64); perr == nil {
						progress.SetValue(pct / 100)
					}
				}
				output.SetText(text)
				scroll.ScrollToBottom()
			})
		})
		fyne.Do(func() {
			progress.SetValue(1)
			if err != nil {
				vmman.ShowCopyableError(w, err)
			} else {
				dialog.ShowInformation("Done", "VM created.", w)
			}
			cancelBtn.Disable()
		})
	}()
	fyneApp.Run()
}

// expose methods? DistroApp fields unexported; add exported wrappers.
