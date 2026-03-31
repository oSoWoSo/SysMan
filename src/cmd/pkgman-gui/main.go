// Command pkgman-gui is a standalone GUI for the xbps package manager.
package main

import (
	"fmt"
	"os"

	"codeberg.org/oSoWoSo/SysMan/src/pkgman"
)

func main() {
	pkgman.InitI18n()
	mode := "auto"
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--help", "-h":
			fmt.Println(pkgman.Usage)
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
		fmt.Fprintln(os.Stderr, "pkgman: no display available, falling back to TUI")
		mode = "tui"
	}

	switch mode {
	case "tui":
		pkgman.RunTUI()
	default:
		pkgman.RunGUI()
	}
}
