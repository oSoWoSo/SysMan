// Package pkgman provides a package manager plugin.
// The default backend targets Void Linux (xbps), but any package manager can be
// supported by implementing the PkgBackend interface.
package pkgman

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"codeberg.org/oSoWoSo/SysMan/src/api"
	"codeberg.org/oSoWoSo/SysMan/src/common"
	"golang.org/x/term"
)

// Usage is the --help text for pkgman.
const Usage = "pkgman [-g|-t]\n\nOptions:\n  -g, --gui   GUI (default)\n  -t, --tui   TUI\n  -h, --help  show this help\n\nEnvironment:\n  SYSMAN_LANG  language override (e.g. cs)"

// QueueEntry represents a single package operation in the batch queue.
type QueueEntry struct {
	Name   string
	Action string // "install" or "remove"
}

// buildOps splits queue entries into install/remove name lists. The optional
// selected package is appended (install when not installed, remove when
// installed) only when it is not already present in the queue, so Apply can act
// on the current selection even without an explicit queue entry.
func buildOps(queue []QueueEntry, selected *Package) (installs, removes []string) {
	for _, e := range queue {
		switch e.Action {
		case "install":
			installs = append(installs, e.Name)
		case "remove":
			removes = append(removes, e.Name)
		}
	}
	if selected == nil || selected.Name == "" {
		return installs, removes
	}
	for _, e := range queue {
		if e.Name == selected.Name {
			return installs, removes
		}
	}
	if selected.Installed {
		removes = append(removes, selected.Name)
	} else {
		installs = append(installs, selected.Name)
	}
	return installs, removes
}

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
	// FilterAll selects all packages.
	FilterAll FilterMode = iota
	// FilterInstalled selects only installed packages.
	FilterInstalled
	// FilterAvailable selects only available (not installed) packages.
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
	return common.Filter(items, int(mode), search, isInstalled, matchesSearch)
}

// PackageDetail holds extended metadata for a single package.
type PackageDetail struct {
	Name          string
	Version       string
	ShortDesc     string
	Homepage      string
	License       string
	Maintainer    string
	Architecture  string
	Repository    string
	InstalledSize string
	FilenameSize  string
	RunDeps       []string
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
	// Reload invalidates any cached state so the next List() returns fresh data.
	Reload()
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

// Name returns "xbps".
func (b *XbpsBackend) Name() string { return "xbps" }

// Reload is a no-op for xbps (packages are always fresh from xbps-query).
func (b *XbpsBackend) Reload() {}

// List returns all available packages.
func (b *XbpsBackend) List() []Package { return LoadPackages() }

// Detail returns package details by name.
func (b *XbpsBackend) Detail(name string) PackageDetail { return QueryDetail(name) }

// Install installs packages by name.
func (b *XbpsBackend) Install(names []string, w io.Writer) (string, error) { return Install(names, w) }

// Remove removes packages by name.
func (b *XbpsBackend) Remove(names []string, w io.Writer) (string, error) { return Remove(names, w) }

// Update updates all packages.
func (b *XbpsBackend) Update(w io.Writer) (string, error) { return Update(w) }

// OpenURL opens a URL in the browser.
func (b *XbpsBackend) OpenURL(url string) { OpenBrowser(url) }

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

// QueryDetail fetches extended metadata for a single package via xbps-query -R -v.
func QueryDetail(name string) PackageDetail {
	out, err := exec.Command("xbps-query", "-R", "-v", name).Output() //nolint:gosec
	d := PackageDetail{Name: name}
	if err != nil {
		return d
	}
	inRunDeps := false
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "\t") {
			if inRunDeps {
				d.RunDeps = append(d.RunDeps, strings.TrimSpace(line))
			}
			continue
		}
		inRunDeps = false
		k, v, ok := strings.Cut(line, ": ")
		if !ok {
			// Handle "key:" (no value, e.g. run_depends:)
			if strings.HasSuffix(line, ":") {
				k = strings.TrimSuffix(line, ":")
				v = ""
			} else {
				continue
			}
		}
		switch strings.TrimSpace(k) {
		case "pkgname":
			d.Name = strings.TrimSpace(v)
		case "pkgver":
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
		case "repository":
			d.Repository = strings.TrimSpace(v)
		case "installed_size":
			d.InstalledSize = strings.TrimSpace(v)
		case "filename-size":
			d.FilenameSize = strings.TrimSpace(v)
		case "run_depends":
			inRunDeps = true
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
		_ = pw.Close()
		_ = pr.Close()
		return "", err
	}
	_ = pw.Close() // close write end in parent so scanner reaches EOF
	var buf strings.Builder
	scanner := bufio.NewScanner(pr)
	for scanner.Scan() {
		line := scanner.Text()
		buf.WriteString(line + "\n")
		if w != nil {
			_, _ = io.WriteString(w, line+"\n")
		}
	}
	_ = pr.Close()
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

// ── AppImage directory (appman/am config) ────────────────────────────

// readAppIconfig returns the first non-empty line of an appman/am config file,
// which holds the user's AppImage applications directory.
func readAppIconfig(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

// DefaultAppImageDir returns the AppImage applications directory inherited from
// the appman/am configuration, falling back to ~/Applications.
// Relative config paths are resolved against the home directory, matching the
// behaviour of appman itself.
func DefaultAppImageDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	for _, p := range []string{
		filepath.Join(home, ".config", "appman", "appman-config"),
		filepath.Join(home, ".config", "AM", "appman-config"),
	} {
		if d := readAppIconfig(p); d != "" {
			if !filepath.IsAbs(d) && home != "" {
				return filepath.Join(home, d)
			}
			return d
		}
	}
	if home != "" {
		return filepath.Join(home, "Applications")
	}
	return ""
}

// effectiveAppImageDir resolves the AppImage directory: an explicit
// configuration override wins, otherwise the appman/am-configured default.
func effectiveAppImageDir(override string) string {
	if override = strings.TrimSpace(override); override != "" {
		return override
	}
	return DefaultAppImageDir()
}

// scanInstalledApps lists installed AM/AppMan apps under dir. For each depth-1
// entry it detects both layouts:
//   - a subdirectory containing a `remove` script (the AM per-app layout; the
//     AppImage binary itself carries no .AppImage suffix), and
//   - a lone file ending in .AppImage (the --launcher layout).
//
// Returns lowercased app name → path. nil when the directory cannot be read.
func scanInstalledApps(dir string) map[string]string {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	apps := make(map[string]string)
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		if e.IsDir() {
			if _, err := os.Stat(filepath.Join(path, "remove")); err != nil {
				continue
			}
			if e.Name() != "" {
				apps[strings.ToLower(e.Name())] = path
			}
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".appimage") {
			continue
		}
		base := strings.TrimSuffix(name, filepath.Ext(name))
		if base != "" {
			apps[strings.ToLower(base)] = path
		}
	}
	return apps
}

// scanAllInstalledApps merges scanInstalledApps over several roots. Earlier
// roots win on name conflicts. A top-level `am` directory is skipped when it
// belongs to a system-wide am installation (its own install dir under /opt).
func scanAllInstalledApps(dirs ...string) map[string]string {
	apps := make(map[string]string)
	for i, dir := range dirs {
		for name, path := range scanInstalledApps(dir) {
			if i > 0 && name == "am" {
				continue
			}
			if _, exists := apps[name]; !exists {
				apps[name] = path
			}
		}
	}
	if len(apps) == 0 {
		return nil
	}
	return apps
}

// ── xbps user repositories (custom repos) ────────────────────────────

// defaultReposFile is the name of the managed repos file inside /etc/xbps.d.
const defaultReposFile = "sysman-repos.conf"

// uniqueSortedRepos trims, drops empties, sorts, and dedupes a repo list.
func uniqueSortedRepos(repos []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(repos))
	for _, r := range repos {
		r = strings.TrimSpace(r)
		if r == "" || seen[r] {
			continue
		}
		seen[r] = true
		out = append(out, r)
	}
	sort.Strings(out)
	return out
}

// reposConfContent renders repository lines for the managed xbps.d conff file.
func reposConfContent(repos []string) string {
	repos = uniqueSortedRepos(repos)
	if len(repos) == 0 {
		return ""
	}
	var b strings.Builder
	for _, r := range repos {
		b.WriteString("repository=" + r + "\n")
	}
	return b.String()
}

// loadReposConf parses `repository=` lines from the managed conff file in dir.
func loadReposConf(dir, file string) []string {
	data, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return nil
	}
	var repos []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if v, ok := strings.CutPrefix(line, "repository="); ok {
			if v = strings.TrimSpace(v); v != "" {
				repos = append(repos, v)
			}
		}
	}
	return repos
}

// writeReposConf writes the managed conff file into dir (unprivileged; works on
// any writable dir, e.g. a temp dir for tests).
func writeReposConf(dir, file string, repos []string) error {
	if dir == "" || file == "" {
		return fmt.Errorf("repos config dir/file not set")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, file+".tmp*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.WriteString(reposConfContent(repos)); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, filepath.Join(dir, file))
}

// syncSystemRepos applies the used-repo set to /etc/xbps.d/<file> with privilege
// escalation (install -Dm644 into the root-owned repo dir).
func syncSystemRepos(repos []string, file string) error {
	if file == "" {
		file = defaultReposFile
	}
	tmp, err := os.CreateTemp("", "sysman-repos-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.WriteString(reposConfContent(repos)); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	_ = tmp.Close()
	defer os.Remove(tmpName) //nolint:errcheck

	dst := filepath.Join("/etc/xbps.d", file)
	_, err = runElevated(nil, []string{"install", "-Dm644", tmpName, dst})
	return err
}
