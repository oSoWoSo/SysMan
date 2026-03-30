// Command pkgman-gui is a standalone GUI for the xbps package manager.
package main

import (
	"fmt"
	"os"

	xbpspkg "codeberg.org/oSoWoSo/SysMan/xbps-pkg"
)

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Println(xbpspkg.Usage)
			os.Exit(0)
		}
	}
	xbpspkg.RunGUI()
}
