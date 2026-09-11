// Command svman-tui is a TUI-only build of svman with no Fyne/OpenGL dependency.
//
// Build:
//
//	CGO_ENABLED=0 go build -tags tui_only -o svman-tui ./cmd/svman-tui/
package main

import (
	"fmt"
	"os"

	"codeberg.org/oSoWoSo/SysMan/src/serman"
)

func main() {
	serman.InitI18n()

	system, user := serman.ResolveScopes()

	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Println(serman.Usage)
			os.Exit(0)
		}
	}

	serman.RunTUI(system, user)
}
