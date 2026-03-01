// Package xbpssrc provides an xbps-src template manager as an embeddable plugin.
//
// It scans $XBPS_DISTDIR/srcpkgs/ (default ~/void/srcpkgs/) and exposes
// build, lint, checksum, bump, install, and clean operations for each template.
package xbpssrc

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
)

// DefaultDistDir signals that the directory should be resolved at runtime.
const DefaultDistDir = ""

// Template represents a single xbps-src package template (one srcpkgs/<name>/ directory).
type Template struct {
	Name string

	// Per-session operation status (in-memory only, not persisted).
	Bumped, Checksummed, Linted, Built, Installed bool
}

// TemplateMeta holds human-readable metadata read from srcpkgs/<name>/template.
type TemplateMeta struct {
	Version  string
	Desc     string
	Homepage string
}

// ── Directory resolution ──────────────────────────────────────────────

// ResolveDistDir returns the effective void-packages directory.
// Priority: argument → $XBPS_DISTDIR → ~/void.
func ResolveDistDir(dir string) string {
	if dir != "" {
		return dir
	}
	if d := os.Getenv("XBPS_DISTDIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, "void")
}

// ── Template loading ──────────────────────────────────────────────────

// LoadTemplates scans srcpkgs/ in distDir and returns a sorted list of templates.
// Returns nil if the directory cannot be read.
func LoadTemplates(distDir string) []Template {
	srcDir := filepath.Join(ResolveDistDir(distDir), "srcpkgs")
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return nil
	}
	var out []Template
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, Template{Name: e.Name()})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ReadMeta parses version, short_desc, and homepage from srcpkgs/<name>/template.
// Missing fields are returned as empty strings.
func ReadMeta(distDir, name string) TemplateMeta {
	var m TemplateMeta
	path := filepath.Join(ResolveDistDir(distDir), "srcpkgs", name, "template")
	f, err := os.Open(path)
	if err != nil {
		return m
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		k, v, ok := strings.Cut(scanner.Text(), "=")
		if !ok {
			continue
		}
		v = strings.Trim(v, `"`)
		switch k {
		case "version":
			m.Version = v
		case "short_desc":
			m.Desc = v
		case "homepage":
			m.Homepage = v
		}
	}
	return m
}

// ── Disk space ───────────────────────────────────────────────────────

// DiskInfo returns human-readable free/total disk space for the filesystem
// that contains distDir. Returns an empty string on error.
func DiskInfo(distDir string) string {
	dir := ResolveDistDir(distDir)
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return ""
	}
	avail := stat.Bavail * uint64(stat.Bsize) //nolint:unconvert
	total := stat.Blocks * uint64(stat.Bsize) //nolint:unconvert
	return fmt.Sprintf("%s free / %s total", humanBytes(avail), humanBytes(total))
}

// humanBytes converts a byte count to a human-readable IEC string (KiB, MiB, …).
func humanBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// ── Command execution ─────────────────────────────────────────────────

// RunXbps executes a command with distDir as the working directory.
// Returns combined stdout+stderr output and any error.
func RunXbps(distDir string, args ...string) (string, error) {
	dir := ResolveDistDir(distDir)
	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// OpenEditor opens srcpkgs/<name>/template in $EDITOR (or xdg-open as fallback).
// The editor is launched detached (non-blocking).
func OpenEditor(distDir, name string) {
	path := filepath.Join(ResolveDistDir(distDir), "srcpkgs", name, "template")
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "xdg-open"
	}
	cmd := exec.Command(editor, path) //nolint:gosec
	_ = cmd.Start()
}

// OpenBrowser opens a URL in the default browser (non-blocking).
func OpenBrowser(url string) {
	cmd := exec.Command("xdg-open", url) //nolint:gosec
	_ = cmd.Start()
}
