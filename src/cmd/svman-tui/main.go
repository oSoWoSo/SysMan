// Command svman-tui is a TUI-only build of svman with no Fyne/OpenGL dependency.
//
// Build:
//
//	CGO_ENABLED=0 go build -tags tui_only -o svman-tui ./cmd/svman-tui/
package main

import (
	"fmt"
	"os"

	"codeberg.org/oSoWoSo/SysMan/src/plugin"
)

func main() {
	plugin.InitI18n()

	serviceDir := os.Getenv("SERVICEDIR")
	if serviceDir == "" {
		serviceDir = plugin.DefaultServiceDir
	}
	serviceDestDir := os.Getenv("SERVICEDESTDIR")
	if serviceDestDir == "" {
		serviceDestDir = plugin.DefaultServiceDestDir
	}

	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Println(plugin.Usage)
			os.Exit(0)
		}
	}

	plugin.RunTUI(serviceDir, serviceDestDir)
}
