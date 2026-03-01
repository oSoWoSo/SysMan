// Command sysinfo runs the System Info plugin as a standalone application.
// Use --gui (default) for the Fyne GUI or --tui for the Bubbletea TUI.
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
	mode := "gui"
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--tui", "-t":
			mode = "tui"
		case "--gui", "-g":
			mode = "gui"
		case "--help", "-h":
			fmt.Println("sysinfo [--gui|--tui]")
			os.Exit(0)
		}
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
		win.ShowAndRun()
	}
}
