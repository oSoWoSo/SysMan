// Command pkgman-tui is a standalone TUI for the xbps package manager.
package main

import (
	"fmt"
	"os"

	xbpspkg "codeberg.org/oSoWoSo/SysMan/src/xbps_pkg"
)

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Println(xbpspkg.Usage)
			os.Exit(0)
		}
	}
	xbpspkg.RunTUI()
}
