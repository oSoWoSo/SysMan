// Package xbpspkg provides a package manager plugin with an extensible backend.
// The default backend targets Void Linux (xbps), but any package manager can be
// supported by implementing the PkgBackend interface.
package pkgman

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"strings"

	"codeberg.org/oSoWoSo/SysMan/src/api"
	"golang.org/x/term"
)

// Usage is the --help text for pkgman.
const Usage = "pkgman [-g|-t]\n\nOptions:\n  -g, --gui   GUI (default)\n  -t, --tui   TUI\n  -h, --help  show this help\n\nEnvironment:\n  SYSMAN_LANG  language override (e.g. cs)"

// isTTY reports whether stdout is connected to a terminal.
func isTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// Package represents a single package entry.
type Package struct {
	Name      string
	Installed bool   // true when the package is currently installed
	ShortDesc string // short description, may be populated lazily
}

// FilterMode represents the filter state for package lists.
type FilterMode int

const (
	FilterAll FilterMode = iota
	FilterInstalled
	FilterAvailable
)

// Filter filters items by state and search query.
func Filter[T any](
	items []T,
	mode FilterMode,
	search string,
	isInstalled func(T) bool,
	matchesSearch func(T, string) bool,
) []T {
	var out []T
	q := strings.ToLower(search)
	for _, item := range items {
		switch mode {
		case FilterInstalled:
			if !isInstalled(item) {
				continue
			}
		case FilterAvailable:
			if isInstalled(item) {
				continue
			}
		}
		if q != "" && !matchesSearch(item, q) {
			continue
		}
		out = append(out, item)
	}
	return out
}

// PackageDetail holds extended metadata for a single package.
type PackageDetail struct {
	Name         string
	Version      string
	ShortDesc    string
	Homepage     string
	License      string
	Maintainer   string
	Architecture string
}

// ── Backend interface ─────────────────────────────────────────────────
//
// PkgBackend abstracts package manager operations so the UI layer is
// independent of the underlying tool (xbps, apt, dnf, pacman, apk, …).
// Implement this interface and pass it to NewTuiModelWithBackend /
// NewGuiAppWithBackend to add support for another package manager.

// PkgBackend is the contract every package manager backend must satisfy.
type PkgBackend interface {
	// Name returns a short human-readable identifier, e.g. "xbps", "apt".
	Name() string
	// List returns all available packages with their installed state.
	List() []Package
	// Detail fetches extended metadata for one package by name.
	Detail(name string) PackageDetail
	// Install installs the named packages.
	// w receives real-time output lines (nil = discard). Returns combined output and error.
	Install(names []string, w io.Writer) (string, error)
	// Remove removes the named packages.
	// w receives real-time output lines (nil = discard). Returns combined output and error.
	Remove(names []string, w io.Writer) (string, error)
	// Update syncs the repository index and upgrades all installed packages.
	// w receives real-time output lines (nil = discard). Returns combined output and error.
	Update(w io.Writer) (string, error)
	// OpenURL opens a URL in the system browser (non-blocking, best-effort).
	OpenURL(url string)
}

// ── xbps backend ─────────────────────────────────────────────────────

// XbpsBackend implements PkgBackend for Void Linux using the xbps toolset.
type XbpsBackend struct{}

// NewXbpsBackend returns an XbpsBackend. It is the default backend used
// when the plugin is constructed with New().
func NewXbpsBackend() *XbpsBackend { return &XbpsBackend{} }

func (b *XbpsBackend) Name() string                                        { return "xbps" }
func (b *XbpsBackend) List() []Package                                     { return LoadPackages() }
func (b *XbpsBackend) Detail(name string) PackageDetail                    { return QueryDetail(name) }
func (b *XbpsBackend) Install(names []string, w io.Writer) (string, error) { return Install(names, w) }
func (b *XbpsBackend) Remove(names []string, w io.Writer) (string, error)  { return Remove(names, w) }
func (b *XbpsBackend) Update(w io.Writer) (string, error)                  { return Update(w) }
func (b *XbpsBackend) OpenURL(url string)                                  { OpenBrowser(url) }

// installedSet returns a set of pkgnames that are currently installed locally.
// Uses xbps-query --list-pkgs (local only, no remote needed).
func installedSet() map[string]bool {
	out, err := exec.Command("xbps-query", "--list-pkgs").Output()
	if err != nil {
		return nil
	}
	set := make(map[string]bool)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		// Format: "ii pkgname-ver   description"
		if len(line) < 4 || !strings.HasPrefix(line, "ii ") {
			continue
		}
		fields := strings.Fields(line[3:])
		if len(fields) == 0 {
			continue
		}
		set[pkgnameFromFull(fields[0])] = true
	}
	return set
}

// LoadPackages runs xbps-query -R --search '_' and returns the full list.
// Installed status is determined by cross-referencing with xbps-query --list-pkgs
// so that the local installation state is always accurate.
func LoadPackages() []Package {
	installed := installedSet()

	out, err := exec.Command("xbps-query", "-R", "--search", "_").Output()
	if err != nil {
		return nil
	}
	var pkgs []Package
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) < 5 {
			continue
		}
		// Format: "[*] pkgname-ver   short description"
		// or:     "[-] pkgname-ver   short description"
		rest := strings.TrimSpace(line[3:])
		fields := strings.SplitN(rest, " ", 2)
		if len(fields) == 0 {
			continue
		}
		name := pkgnameFromFull(fields[0])
		desc := ""
		if len(fields) > 1 {
			desc = strings.TrimSpace(fields[1])
		}
		pkgs = append(pkgs, Package{
			Name:      name,
			Installed: installed[name],
			ShortDesc: desc,
		})
	}
	return pkgs
}

// pkgnameFromFull strips the version suffix from "pkgname-version" strings.
// xbps version numbers always start with a digit; split on the last '-' that
// precedes a digit.
func pkgnameFromFull(full string) string {
	for i := len(full) - 1; i > 0; i-- {
		if full[i-1] == '-' && full[i] >= '0' && full[i] <= '9' {
			return full[:i-1]
		}
	}
	return full
}

// QueryDetail fetches extended metadata for a single package name via xbps-query --show.
func QueryDetail(name string) PackageDetail {
	out, err := exec.Command("xbps-query", "-R", "--show", name).Output() //nolint:gosec
	d := PackageDetail{Name: name}
	if err != nil {
		return d
	}
	for _, line := range strings.Split(string(out), "\n") {
		k, v, ok := strings.Cut(line, ": ")
		if !ok {
			continue
		}
		switch strings.TrimSpace(k) {
		case "pkgname":
			d.Name = strings.TrimSpace(v)
		case "pkgver":
			// "pkgver: vim-9.2.0_1" — strip pkgname prefix
			full := strings.TrimSpace(v)
			if idx := strings.LastIndex(full, "-"); idx > 0 {
				d.Version = full[idx+1:]
			} else {
				d.Version = full
			}
		case "short_desc":
			d.ShortDesc = strings.TrimSpace(v)
		case "homepage":
			d.Homepage = strings.TrimSpace(v)
		case "license":
			d.License = strings.TrimSpace(v)
		case "maintainer":
			d.Maintainer = strings.TrimSpace(v)
		case "architecture":
			d.Architecture = strings.TrimSpace(v)
		}
	}
	return d
}

// runElevated runs an elevated command. When stdout is a terminal the command
// inherits stdin/stdout/stderr so the user sees real-time output and can
// interact (e.g. answer prompts). When stdout is not a terminal (GUI mode)
// all output is streamed line-by-line to w.
// Returns combined output (empty in TTY mode) and any error.
func runElevated(w io.Writer, args []string) (string, error) {
	elevated := api.Elevate(args...) //nolint:gosec
	cmd := exec.Command(elevated[0], elevated[1:]...)
	// Pass through to terminal only when no GUI writer is provided and stdout is a TTY.
	if w == nil && isTTY() {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return "", cmd.Run()
	}
	// GUI / non-TTY: stream output in real-time.
	pr, pw, err := os.Pipe()
	if err != nil {
		return "", err
	}
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		pw.Close()
		pr.Close()
		return "", err
	}
	pw.Close() // close write end in parent so scanner reaches EOF
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

// Install installs packages using xbps-install with privilege escalation.
// In TTY mode output goes to the terminal; in GUI mode it streams to w.
func Install(names []string, w io.Writer) (string, error) {
	args := append([]string{"xbps-install", "-Sy"}, names...)
	return runElevated(w, args)
}

// Remove removes packages using xbps-remove with privilege escalation.
// In TTY mode output goes to the terminal; in GUI mode it streams to w.
func Remove(names []string, w io.Writer) (string, error) {
	args := append([]string{"xbps-remove", "-Ry"}, names...)
	return runElevated(w, args)
}

// Update runs xbps-install -Suvy to sync and upgrade all packages.
// In TTY mode output goes to the terminal; in GUI mode it streams to w.
func Update(w io.Writer) (string, error) {
	return runElevated(w, []string{"xbps-install", "-Suvy"})
}

// OpenBrowser opens a URL in the default browser (non-blocking).
func OpenBrowser(url string) {
	if url == "" {
		return
	}
	cmd := exec.Command("xdg-open", url) //nolint:gosec
	_ = cmd.Start()
}
