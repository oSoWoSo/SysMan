// Command xbps is a standalone xbps-src template manager (GUI + TUI).
package main

import (
	"fmt"
	"os"

	xbpssrc "codeberg.org/oSoWoSo/SysMan/src/xbps_src"
)

func main() {
	distDir := os.Getenv("XBPS_DISTDIR")

	mode := "auto"
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--tui", "-t":
			mode = "tui"
		case "--gui", "-g":
			mode = "gui"
		case "--help", "-h":
			fmt.Println(xbpssrc.Usage)
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
		fmt.Fprintln(os.Stderr, "srcman: no display available, falling back to TUI")
		mode = "tui"
	}

	switch mode {
	case "tui":
		xbpssrc.RunTUI(distDir)
	default:
		xbpssrc.RunGUI(distDir)
	}
}
