// Command srcman-tui runs the xbps-src template manager as a standalone Bubbletea TUI.
package main

import (
	"fmt"
	"os"

	xbpssrc "codeberg.org/oSoWoSo/SysMan/src/xbps_src"
)

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Println(xbpssrc.Usage)
			os.Exit(0)
		}
	}
	xbpssrc.RunTUI(os.Getenv("XBPS_DISTDIR"))
}
