// Command infoman-tui runs the System Info plugin as a standalone Bubbletea TUI.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"codeberg.org/oSoWoSo/SysMan/sysinfo"
)

func main() {
	prog := tea.NewProgram(sysinfo.New().Model(), tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
