// Command infoman-gui runs the System Info plugin as a standalone Fyne GUI.
package main

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"codeberg.org/oSoWoSo/SysMan/src/infman"
)

func main() {
	mode := "auto"
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--help", "-h":
			fmt.Println(infman.Usage)
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
		fmt.Fprintln(os.Stderr, "infoman: no display available, falling back to TUI")
		mode = "tui"
	}

	switch mode {
	case "tui":
		infman.RunTUI()
	default:
		p := infman.New()
		a := app.New()
		win := a.NewWindow(p.Name())
		win.SetContent(p.Content(win))
		win.Resize(fyne.NewSize(420, 300))
		win.SetMaster()
		win.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
			if e.Name == fyne.KeyEscape {
				a.Quit()
			}
		})
		win.ShowAndRun()
	}
}
