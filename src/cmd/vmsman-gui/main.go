// Command vmsman-gui runs the VM Manager plugin as a standalone Fyne GUI.
package main

import (
	"fmt"
	"os"

	vmman "codeberg.org/oSoWoSo/SysMan/src/vmsman"
)

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--help" || arg == "-h" {
			fmt.Println(vmman.Usage)
			os.Exit(0)
		}
	}

	vmDir := os.Getenv("VMDIR")
	if vmDir == "" {
		vmDir = vmman.DefaultVmDir
	}

	vmman.RunGUI(vmDir)
}
