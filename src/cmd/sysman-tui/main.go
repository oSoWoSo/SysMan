// Command sysman-tui runs the system manager aggregator as a standalone Bubbletea TUI.
// Currently uses serman as the default TUI.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"codeberg.org/oSoWoSo/SysMan/src/serman"
)

func main() {
	serman.InitI18n()

	serviceDir := os.Getenv("SERVICEDIR")
	if serviceDir == "" {
		serviceDir = serman.DefaultServiceDir
	}
	serviceDestDir := os.Getenv("SERVICEDESTDIR")
	if serviceDestDir == "" {
		serviceDestDir = serman.DefaultServiceDestDir
	}

	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Println(serman.Usage)
			os.Exit(0)
		}
	}
	prog := tea.NewProgram(serman.NewTuiModel(serman.NewRunitBackend(serviceDir, serviceDestDir)), tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
