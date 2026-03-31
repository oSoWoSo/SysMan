// Command srcman-tui runs the xbps-src template manager as a standalone Bubbletea TUI.
package main

import (
	"fmt"
	"os"

	"codeberg.org/oSoWoSo/SysMan/src/srcman"
)

func main() {
	srcman.InitI18n()
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Println(srcman.Usage)
			os.Exit(0)
		}
	}
	srcman.RunTUI(os.Getenv("XBPS_DISTDIR"))
}
