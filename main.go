package main

import (
	"fmt"
	"os"
)

// ── Utilities ────────────────────────────────────────────────────────

// errorWriter returns the standard error stream for logging errors.
func errorWriter() *os.File { return os.Stderr }

// ── Main ─────────────────────────────────────────────────────────────

// main is the application entry point.
// It initializes translations, parses command-line arguments,
// reads environment configuration, and launches the selected UI mode.
func main() {
	initI18n()

	// Read service directories from environment or use defaults ────────
	serviceDir := getEnv("SERVICEDIR", defaultServiceDir)
	serviceDestDir := getEnv("SERVICEDESTDIR", defaultServiceDestDir)

	// Parse command-line arguments and select UI mode ──────────────────
	mode := "gui"
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--tui", "-t":
			mode = "tui"
		case "--gui", "-g":
			mode = "gui"
		case "--help", "-h":
			// Display help message with usage and environment variables
			fmt.Printf(`svman — %s

%s:
  svman [--gui]   %s
  svman --tui     %s

%s:
  SERVICEDIR      %s  (default: /etc/sv)
  SERVICEDESTDIR  %s  (default: /var/service)
  SVMAN_LANG      %s  (cs, en)
`,
				t("app.subtitle"),
				"Usage", "GUI (default)", "TUI terminal",
				"Environment",
				"service directory",
				"enabled services directory",
				"language override",
			)
			os.Exit(0)
		}
	}

	// Launch the selected UI mode ─────────────────────────────────────
	switch mode {
	case "tui":
		runTUI(serviceDir, serviceDestDir)
	default:
		runGUI(serviceDir, serviceDestDir)
	}
}
