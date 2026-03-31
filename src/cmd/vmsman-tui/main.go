// Command vmsman-tui runs the VM Manager plugin as a standalone Bubbletea TUI.
package main

import (
	"fmt"
	"os"

	vmman "codeberg.org/oSoWoSo/SysMan/src/vmsman"
)

func main() {
	vmman.InitI18n()
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Println(vmman.Usage)
			os.Exit(0)
		}
	}

	vmDir := os.Getenv("VMDIR")
	if vmDir == "" {
		vmDir = vmman.DefaultVMDir
	}

	vmman.RunTUI(vmDir)
}
