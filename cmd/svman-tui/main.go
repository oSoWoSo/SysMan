// Command svman-tui is a TUI-only build of svman with no Fyne/OpenGL dependency.
// It can be cross-compiled with CGO_ENABLED=0 for any supported platform.
//
// Build:
//
//	CGO_ENABLED=0 go build -tags tui_only -o svman-tui ./cmd/svman-tui/
package main

import (
	"fmt"
	"os"

	"codeberg.org/oSoWoSo/svman/plugin"
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
			fmt.Printf("svman-tui — runit service manager (TUI)\n\nUsage: svman-tui\n\nEnvironment:\n  SERVICEDIR      service directory (default: /etc/sv)\n  SERVICEDESTDIR  enabled services directory (default: /var/service)\n  SVMAN_LANG      language override (cs, en)\n")
			os.Exit(0)
		}
	}

	plugin.RunTUI(serviceDir, serviceDestDir)
}
