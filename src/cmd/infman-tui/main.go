// Command infoman-tui runs the System Info plugin as a standalone Bubbletea TUI.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"codeberg.org/oSoWoSo/SysMan/src/common"
	"codeberg.org/oSoWoSo/SysMan/src/infman"
)

func main() {
	infman.InitI18n()
	common.EnsureZone()
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Println(infman.Usage)
			os.Exit(0)
		}
	}
	prog := tea.NewProgram(infman.NewTuiModel())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
