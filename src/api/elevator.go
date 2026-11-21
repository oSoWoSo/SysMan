package api

import (
	"os"
	"os/exec"

	"golang.org/x/term"
)

// FindElevator returns the first available privilege-escalation binary.
// When stdin is a terminal, it prefers console-based helpers so TUI
// authentication stays in the terminal instead of opening a GUI polkit agent.
// Returns "" when already running as root (EUID == 0).
func FindElevator() string {
	if os.Getuid() == 0 {
		return ""
	}

	bins := []string{"pkexec", "doas", "sudo"}
	// Prefer a GUI polkit agent when a display is available, even if stdin
	// happens to be a terminal (e.g. launched from a terminal emulator).
	hasDisplay := os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
	if term.IsTerminal(int(os.Stdin.Fd())) && !hasDisplay {
		bins = []string{"doas", "sudo"}
	}
	for _, bin := range bins {
		if p, err := exec.LookPath(bin); err == nil && p != "" {
			return bin
		}
	}
	return "sudo" // fallback — will fail with a meaningful error
}

// Elevate prepends the elevator binary to args when not running as root.
// Safe to pass directly to exec.Command(args[0], args[1:]...).
func Elevate(args ...string) []string {
	el := FindElevator()
	if el == "" {
		return args
	}
	return append([]string{el}, args...)
}
