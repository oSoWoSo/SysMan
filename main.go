package main

import (
	"fmt"
	"os"

	"codeberg.org/oSoWoSo/SysMan/src/plugin"
)

// main is the application entry point.
// It initializes translations, parses command-line arguments,
// reads environment configuration, and launches the selected UI mode.
func main() {
	plugin.InitI18n()

	// Read service directories from environment or use defaults.
	serviceDir := os.Getenv("SERVICEDIR")
	if serviceDir == "" {
		serviceDir = plugin.DefaultServiceDir
	}
	serviceDestDir := os.Getenv("SERVICEDESTDIR")
	if serviceDestDir == "" {
		serviceDestDir = plugin.DefaultServiceDestDir
	}

	// Parse command-line arguments and select UI mode.
	mode := "auto"
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--tui", "-t":
			mode = "tui"
		case "--gui", "-g":
			mode = "gui"
		case "--help", "-h":
			fmt.Println(plugin.Usage)
			os.Exit(0)
		}
	}

	hasDisplay := os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""

	// Auto-detect: prefer GUI when a display server is available.
	if mode == "auto" {
		if hasDisplay {
			mode = "gui"
		} else {
			mode = "tui"
		}
	}

	// Explicit --gui with no display falls back to TUI.
	if mode == "gui" && !hasDisplay {
		fmt.Fprintln(os.Stderr, "svman: no display available, falling back to TUI")
		mode = "tui"
	}

	// Launch the selected UI mode.
	switch mode {
	case "tui":
		plugin.RunTUI(serviceDir, serviceDestDir)
	default:
		plugin.RunGUI(serviceDir, serviceDestDir)
	}
}
