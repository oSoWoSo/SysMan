package main

import (
	"fmt"
	"os"

	"codeberg.org/oSoWoSo/svman/plugin"
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
	mode := "gui"
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--tui", "-t":
			mode = "tui"
		case "--gui", "-g":
			mode = "gui"
		case "--help", "-h":
			fmt.Printf(`svman — %s

%s:
  svman [--gui]   %s
  svman --tui     %s

%s:
  SERVICEDIR      %s  (default: /etc/sv)
  SERVICEDESTDIR  %s  (default: /var/service)
  SVMAN_LANG      %s  (cs, en)
`,
				plugin.T["app.subtitle"],
				"Usage", "GUI (default)", "TUI terminal",
				"Environment",
				"service directory",
				"enabled services directory",
				"language override",
			)
			os.Exit(0)
		}
	}

	// Launch the selected UI mode.
	switch mode {
	case "tui":
		plugin.RunTUI(serviceDir, serviceDestDir)
	default:
		plugin.RunGUI(serviceDir, serviceDestDir)
	}
}
