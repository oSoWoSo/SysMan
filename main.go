// Command sysman is a demo system manager.
package main

import (
	"fmt"
	"os"

	"codeberg.org/oSoWoSo/SysMan/src/serman"
)

// main is the application entry point.
// It initializes translations, parses command-line arguments,
// reads environment configuration, and launches the selected UI mode.
func main() {
	serman.InitI18n()

	// Resolve system and user service scopes from env, config, or defaults.
	system, user := serman.ResolveScopes()

	// Parse command-line arguments and select UI mode.
	mode := "auto"
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--tui", "-t":
			mode = "tui"
		case "--gui", "-g":
			mode = "gui"
		case "--help", "-h":
			fmt.Println(serman.Usage)
			os.Exit(0)
		}
	}

	hasDisplay := os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""

	// Auto-detect: prefer GUI when a display server is available.
	if mode == "auto" {
		if hasDisplay {
			mode = "gui"
		} else {
			mode = "tui"
		}
	}

	// Explicit --gui with no display falls back to TUI.
	if mode == "gui" && !hasDisplay {
		fmt.Fprintln(os.Stderr, "svman: no display available, falling back to TUI")
		mode = "tui"
	}

	// Launch the selected UI mode.
	switch mode {
	case "tui":
		serman.RunTUI(system, user)
	default:
		serman.RunGUI(system, user)
	}
}
