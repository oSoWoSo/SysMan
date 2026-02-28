package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// ── Constants ────────────────────────────────────────────────────────

// Version is set at build time via -ldflags "-X main.Version=<tag>".
var Version = "dev"

// App metadata used in the About dialog.
const (
	appAuthor  = "oSoWoSo"
	appLicense = "MIT"
	appURL     = "https://codeberg.org/oSoWoSo/svman"
)

// Default directories for service definitions and enabled services.
const (
	defaultServiceDir     = "/etc/sv"      // service definition directory
	defaultServiceDestDir = "/var/service" // enabled services symlink directory
)

// ── Types ────────────────────────────────────────────────────────────

// Service represents a single service with its name and enabled state.
type Service struct {
	Name    string // service name (directory name)
	Enabled bool   // true if symlink exists in destination directory
}

// ── Utilities ────────────────────────────────────────────────────────

// getEnv retrieves an environment variable, returning fallback if not set.
func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// isSymlink checks whether the given path is a symbolic link.
// Returns false if the path does not exist or cannot be accessed.
func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

// ── Loading ──────────────────────────────────────────────────────────

// loadServices scans the service directory and returns a sorted list of services.
// Each service's enabled state is determined by checking for a symlink
// in the destination directory.
// Returns nil if the service directory cannot be read.
func loadServices(serviceDir, destDir string) []Service {
	entries, err := os.ReadDir(serviceDir)
	if err != nil {
		return nil
	}
	var svcs []Service
	for _, e := range entries {
		// skip non-directory entries
		info, err := os.Stat(filepath.Join(serviceDir, e.Name()))
		if err != nil || !info.IsDir() {
			continue
		}
		svcs = append(svcs, Service{
			Name:    e.Name(),
			Enabled: isSymlink(filepath.Join(destDir, e.Name())),
		})
	}
	// sort services alphabetically by name
	sort.Slice(svcs, func(i, j int) bool { return svcs[i].Name < svcs[j].Name })
	return svcs
}

// ── Service Control ─────────────────────────────────────────────────

// enableService creates a symlink from the service source to the destination,
// enabling the service. Uses sudo to handle permission requirements.
// Returns an error if the symlink creation fails.
func enableService(serviceDir, destDir, name string) error {
	src := filepath.Join(serviceDir, name)
	dst := filepath.Join(destDir, name)
	out, err := exec.Command("sudo", "ln", "-s", src, dst).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}

// disableService removes the symlink from the destination directory,
// disabling the service. Uses sudo to handle permission requirements.
// Returns an error if the symlink removal fails.
func disableService(destDir, name string) error {
	dst := filepath.Join(destDir, name)
	out, err := exec.Command("sudo", "rm", dst).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}
