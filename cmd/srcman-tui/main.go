// Command srcman-tui runs the xbps-src template manager as a standalone Bubbletea TUI.
package main

import (
	"os"

	xbpssrc "codeberg.org/oSoWoSo/SysMan/xbps-src"
)

func main() {
	xbpssrc.RunTUI(os.Getenv("XBPS_DISTDIR"))
}
