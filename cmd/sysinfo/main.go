// Command sysinfo runs the System Info plugin as a standalone application.
package main

import (
	"fmt"
	"os"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	tea "github.com/charmbracelet/bubbletea"

	"codeberg.org/oSoWoSo/SysMan/sysinfo"
)

func main() {
	mode := "auto"
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--tui", "-t":
			mode = "tui"
		case "--gui", "-g":
			mode = "gui"
		case "--help", "-h":
			fmt.Println(sysinfo.Usage)
			os.Exit(0)
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

	if mode == "gui" && !hasDisplay {
		fmt.Fprintln(os.Stderr, "infoman: no display available, falling back to TUI")
		mode = "tui"
	}

	p := sysinfo.New()

	switch mode {
	case "tui":
		prog := tea.NewProgram(p.Model(), tea.WithAltScreen())
		if _, err := prog.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
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
