package api

import (
	"os"
	"os/exec"
)

// FindElevator returns the first available privilege-escalation binary
// in the order: pkexec, doas, sudo.
// Returns "" when already running as root (EUID == 0).
func FindElevator() string {
	if os.Getuid() == 0 {
		return ""
	}
	for _, bin := range []string{"pkexec", "doas", "sudo"} {
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
