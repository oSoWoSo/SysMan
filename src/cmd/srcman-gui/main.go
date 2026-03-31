// Command srcman-gui runs the xbps-src template manager as a standalone Fyne GUI.
package main

import (
	"fmt"
	"os"

	"codeberg.org/oSoWoSo/SysMan/src/srcman"
)

func main() {
	srcman.InitI18n()
	mode := "auto"
	distDir := os.Getenv("XBPS_DISTDIR")
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--help", "-h":
			fmt.Println(srcman.Usage)
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
		fmt.Fprintln(os.Stderr, "srcman: no display available, falling back to TUI")
		mode = "tui"
	}

	switch mode {
	case "tui":
		srcman.RunTUI(distDir)
	default:
		srcman.RunGUI(distDir)
	}
}
