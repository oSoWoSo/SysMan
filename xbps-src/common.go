// Package xbpssrc provides an xbps-src template manager as an embeddable plugin.
//
// It scans $XBPS_DISTDIR/srcpkgs/ (default ~/void/srcpkgs/) and exposes
// build, lint, checksum, bump, install, and clean operations for each template.
package xbpssrc

import (
	"bufio"
	"fmt"
	"io"
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

// LoadTemplates scans srcpkgs/ in distDir and returns templates sorted by
// the modification time of their template file (newest first).
// Returns nil if the directory cannot be read.
func LoadTemplates(distDir string) []Template {
	srcDir := filepath.Join(ResolveDistDir(distDir), "srcpkgs")
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return nil
	}
	type entry struct {
		name    string
		modTime int64 // Unix nano
	}
	var raw []entry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		tplPath := filepath.Join(srcDir, e.Name(), "template")
		if info, err := os.Stat(tplPath); err == nil {
			raw = append(raw, entry{e.Name(), info.ModTime().UnixNano()})
		} else {
			raw = append(raw, entry{e.Name(), 0})
		}
	}
	sort.Slice(raw, func(i, j int) bool { return raw[i].modTime > raw[j].modTime })
	out := make([]Template, len(raw))
	for i, r := range raw {
		out[i] = Template{Name: r.name}
	}
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
	return RunXbpsStream(distDir, nil, args...)
}

// RunXbpsStream executes a command with distDir as the working directory,
// streaming each output line to w in real-time (pass nil to skip streaming).
// Returns combined output and any error.
func RunXbpsStream(distDir string, w io.Writer, args ...string) (string, error) {
	dir := ResolveDistDir(distDir)
	pr, pw, err := os.Pipe()
	if err != nil {
		return "", err
	}
	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec
	cmd.Dir = dir
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		return "", err
	}
	pw.Close()
	var buf strings.Builder
	scanner := bufio.NewScanner(pr)
	for scanner.Scan() {
		line := scanner.Text()
		buf.WriteString(line + "\n")
		if w != nil {
			_, _ = io.WriteString(w, line+"\n")
		}
	}
	pr.Close()
	err = cmd.Wait()
	return strings.TrimSpace(buf.String()), err
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

// RunXlocate runs "xlocate <query>" and returns its output.
func RunXlocate(query string) (string, error) {
	return RunXbpsStream("", nil, "xlocate", query)
}
