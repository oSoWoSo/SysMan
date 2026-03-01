// Command srcman-gui runs the xbps-src template manager as a standalone Fyne GUI.
package main

import (
	"os"

	xbpssrc "codeberg.org/oSoWoSo/SysMan/xbps-src"
)

func main() {
	xbpssrc.RunGUI(os.Getenv("XBPS_DISTDIR"))
}
