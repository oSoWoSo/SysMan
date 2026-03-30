// Command xbps is a standalone Void Linux package manager TUI.
// It provides package search, install, and removal via xbps-query/xbps-install/xbps-remove.
package main

import xbpspkg "codeberg.org/oSoWoSo/SysMan/xbps-pkg"

func main() {
	xbpspkg.RunTUI()
}
