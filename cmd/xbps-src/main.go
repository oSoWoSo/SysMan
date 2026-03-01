// Command xbps is a standalone xbps-src template manager (GUI + TUI).
//
// Usage:
//
//	xbps           # GUI (default)
//	xbps --gui     # GUI explicitly
//	xbps --tui     # TUI (terminal)
//
// Environment:
//
//	XBPS_DISTDIR   void-packages directory (default: ~/void)
package main

import (
	"fmt"
	"os"

	xbpssrc "codeberg.org/oSoWoSo/svman/xbps-src"
)

func main() {
	distDir := os.Getenv("XBPS_DISTDIR") // xbpssrc resolves ~/void if empty

	mode := "gui"
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--tui", "-t":
			mode = "tui"
		case "--gui", "-g":
			mode = "gui"
		case "--help", "-h":
			fmt.Printf("xbps — xbps-src template manager\n\nUsage:\n  xbps [--gui|--tui]\n\nEnvironment:\n  XBPS_DISTDIR  void-packages directory (default: ~/void)\n")
			os.Exit(0)
		}
	}

	switch mode {
	case "tui":
		xbpssrc.RunTUI(distDir)
	default:
		xbpssrc.RunGUI(distDir)
	}
}
