// Command vmsman-gui runs the VM Manager plugin as a standalone Fyne GUI.
package main

import (
	"fmt"
	"os"

	vmman "codeberg.org/oSoWoSo/SysMan/src/vmsman"
)

func main() {
	mode := "auto"
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--help", "-h":
			fmt.Println(vmman.Usage)
			os.Exit(0)
		case "--tui", "-t":
			mode = "tui"
		case "--gui", "-g":
			mode = "gui"
		}
	}

	hasDisplay := os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""

	if mode == "auto" {
		if hasDisplay {
			mode = "gui"
		} else {
			mode = "tui"
		}
	}

	// Explicit --gui with no display falls back to TUI.
	if mode == "gui" && !hasDisplay {
		fmt.Fprintln(os.Stderr, "vmsman: no display available, falling back to TUI")
		mode = "tui"
	}

	vmDir := os.Getenv("VMDIR")
	if vmDir == "" {
		vmDir = vmman.DefaultVMDir
	}

	switch mode {
	case "tui":
		vmman.RunTUI(vmDir)
	default:
		vmman.RunGUI(vmDir)
	}
}
