// Command ugman-gui is a standalone GUI for user and group management.
package main

import (
	"fmt"
	"os"

	"codeberg.org/oSoWoSo/SysMan/src/ugsman"
)

func main() {
	mode := "auto"
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--help", "-h":
			fmt.Println(ugsman.Usage)
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
		fmt.Fprintln(os.Stderr, "ugman: no display available, falling back to TUI")
		mode = "tui"
	}

	switch mode {
	case "tui":
		ugsman.RunTUI()
	default:
		ugsman.RunGUI()
	}
}
