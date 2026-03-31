// Command serman-gui runs the Service Manager plugin as a standalone Fyne GUI.
package main

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"codeberg.org/oSoWoSo/SysMan/src/common"
	"codeberg.org/oSoWoSo/SysMan/src/serman"
)

func main() {
	mode := "auto"
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--help", "-h":
			fmt.Println(serman.Usage)
			os.Exit(0)
		case "--tui", "-t":
			mode = "tui"
		case "--gui", "-g":
			mode = "gui"
		}
	}

	hasDisplay := os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""

	if mode == "auto" {
		if hasDisplay {
			mode = "gui"
		} else {
			mode = "tui"
		}
	}

	// Explicit --gui with no display falls back to TUI.
	if mode == "gui" && !hasDisplay {
		fmt.Fprintln(os.Stderr, "serman: no display available, falling back to TUI")
		mode = "tui"
	}

	serviceDir := os.Getenv("SERVICEDIR")
	if serviceDir == "" {
		serviceDir = serman.DefaultServiceDir
	}
	serviceDestDir := os.Getenv("SERVICEDESTDIR")
	if serviceDestDir == "" {
		serviceDestDir = serman.DefaultServiceDestDir
	}

	switch mode {
	case "tui":
		serman.RunTUI(serviceDir, serviceDestDir)
	default:
		p := serman.New(serviceDir, serviceDestDir)
		a := app.New()
		win := a.NewWindow(p.Name())
		common.SetWindowIcon(win)
		win.SetContent(p.Content(win))
		win.Resize(fyne.NewSize(760, 520))
		win.SetMaster()
		win.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
			if e.Name == fyne.KeyEscape {
				a.Quit()
			}
		})
		win.ShowAndRun()
	}
}
