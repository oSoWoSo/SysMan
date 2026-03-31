// Command pkgman-tui runs the Package Manager plugin as a standalone Bubbletea TUI.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"codeberg.org/oSoWoSo/SysMan/src/pkgman"
)

func main() {
	pkgman.InitI18n()
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Println(pkgman.Usage)
			os.Exit(0)
		}
	}
	prog := tea.NewProgram(pkgman.NewTuiModel(), tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
