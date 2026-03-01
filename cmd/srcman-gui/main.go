// Command srcman-gui runs the xbps-src template manager as a standalone Fyne GUI.
package main

import (
	"fmt"
	"os"

	xbpssrc "codeberg.org/oSoWoSo/SysMan/xbps-src"
)

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Println(xbpssrc.Usage)
			os.Exit(0)
		}
	}
	xbpssrc.RunGUI(os.Getenv("XBPS_DISTDIR"))
}
