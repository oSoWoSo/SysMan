package pkgman

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// NixBackend implements PkgBackend using the Nix profile and nixpkgs search commands.
type NixBackend struct{}

// NewNixBackend returns a Nix package backend.
func NewNixBackend() *NixBackend { return &NixBackend{} }

// Name returns "nix".
func (b *NixBackend) Name() string { return "nix" }

// Reload is a no-op for nix.
func (b *NixBackend) Reload() {}

// List returns available nix packages.
func (b *NixBackend) List() []Package {
	if path := nixEnvPath(); path != "" {
		installed := nixEnvInstalledSet(path)
		out, err := exec.Command(path, "-f", "<nixpkgs>", "-qaP", ".*").Output()
		if err == nil {
			return parseNixEnvPackages(string(out), installed, b)
		}
		return nixInstalledPackages(installed)
	}
	installed := nixInstalledSet()
	out, err := exec.Command("nix", "search", "nixpkgs", ".*", "--json").Output()
	if err != nil {
		return nixInstalledPackages(installed)
	}
	var entries map[string]nixSearchEntry
	if json.Unmarshal(out, &entries) != nil {
		return nixInstalledPackages(installed)
	}
	packages := make([]Package, 0, len(entries))
	for attr, entry := range entries {
		name := entry.Pname
		if name == "" {
			name = lastAttr(attr)
		}
		packages = append(packages, Package{
			Name:      name,
			Installed: installed[name],
			ShortDesc: entry.Description,
			Backend:   b,
		})
	}
	return packages
}

// Detail returns metadata for a nix package.
func (b *NixBackend) Detail(name string) PackageDetail {
	detail := PackageDetail{Name: name, Repository: "nixpkgs"}
	if path := nixEnvPath(); path != "" {
		out, err := exec.Command(path, "-f", "<nixpkgs>", "-qaA", name).Output()
		if err == nil {
			_, detail.Version = nixNameVersion(strings.TrimSpace(string(out)))
		}
		return detail
	}
	out, err := exec.Command("nix", "search", "nixpkgs", "^"+name+"$", "--json").Output()
	if err != nil {
		return detail
	}
	var entries map[string]nixSearchEntry
	if json.Unmarshal(out, &entries) != nil {
		return detail
	}
	for _, entry := range entries {
		detail.ShortDesc = entry.Description
		detail.Version = entry.Version
		break
	}
	return detail
}

// Install installs nix packages.
func (b *NixBackend) Install(names []string, w io.Writer) (string, error) {
	if path := nixEnvPath(); path != "" {
		args := append([]string{path, "-f", "<nixpkgs>", "-iA"}, names...)
		return runCommand(w, args...)
	}
	args := append([]string{"nix", "profile", "install"}, nixAttrs(names)...)
	return runCommand(w, args...)
}

// Remove removes nix packages.
func (b *NixBackend) Remove(names []string, w io.Writer) (string, error) {
	if path := nixEnvPath(); path != "" {
		return runCommand(w, append([]string{path, "-e"}, names...)...)
	}
	return runCommand(w, append([]string{"nix", "profile", "remove"}, names...)...)
}

// Update updates nix packages.
func (b *NixBackend) Update(w io.Writer) (string, error) {
	if path := nixEnvPath(); path != "" {
		out, err := runCommand(w, "nix-channel", "--update")
		if err != nil {
			return out, err
		}
		next, err := runCommand(w, path, "-u")
		return strings.TrimSpace(out + "\n" + next), err
	}
	return runCommand(w, "nix", "profile", "upgrade", "*")
}

// OpenURL opens a URL in the browser.
func (b *NixBackend) OpenURL(url string) { OpenBrowser(url) }

type nixSearchEntry struct {
	Pname       string `json:"pname"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type nixProfile struct {
	Elements map[string]struct {
		AttrPath string `json:"attrPath"`
		Active   bool   `json:"active"`
	} `json:"elements"`
}

func nixInstalledSet() map[string]bool {
	out, err := exec.Command("nix", "profile", "list", "--json").Output()
	if err != nil {
		return nil
	}
	var profile nixProfile
	if json.Unmarshal(out, &profile) != nil {
		return nil
	}
	set := make(map[string]bool)
	for _, element := range profile.Elements {
		if element.Active {
			set[lastAttr(element.AttrPath)] = true
		}
	}
	return set
}

func nixEnvPath() string {
	if path, err := exec.LookPath("nix-env"); err == nil {
		return path
	}
	paths := []string{"/nix/var/nix/profiles/default/bin/nix-env"}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, home+"/.nix-profile/bin/nix-env")
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func nixEnvInstalledSet(path string) map[string]bool {
	out, err := exec.Command(path, "-q").Output()
	if err != nil {
		return nil
	}
	set := make(map[string]bool)
	for _, line := range strings.Split(string(out), "\n") {
		name, _ := nixNameVersion(strings.TrimSpace(line))
		if name != "" {
			set[name] = true
		}
	}
	return set
}

func parseNixEnvPackages(output string, installed map[string]bool, backend PkgBackend) []Package {
	var packages []Package
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 {
			continue
		}
		name := lastAttr(fields[0])
		if parsed, _ := nixNameVersion(name); parsed != "" {
			name = parsed
		}
		for _, field := range fields[1:] {
			if parsed, _ := nixNameVersion(field); parsed != "" {
				name = parsed
				break
			}
		}
		if name == "" {
			continue
		}
		packages = append(packages, Package{Name: name, Installed: installed[name], Backend: backend})
	}
	return packages
}

func nixNameVersion(value string) (string, string) {
	value = strings.TrimSpace(value)
	idx := strings.LastIndexByte(value, '-')
	if idx <= 0 || idx+1 >= len(value) || value[idx+1] < '0' || value[idx+1] > '9' {
		return value, ""
	}
	return value[:idx], value[idx+1:]
}

func nixInstalledPackages(installed map[string]bool) []Package {
	packages := make([]Package, 0, len(installed))
	for name := range installed {
		packages = append(packages, Package{Name: name, Installed: true})
	}
	return packages
}

func nixAttrs(names []string) []string {
	attrs := make([]string, len(names))
	for i, name := range names {
		attrs[i] = "nixpkgs#" + name
	}
	return attrs
}

func lastAttr(value string) string {
	if idx := strings.LastIndexByte(value, '.'); idx >= 0 {
		return value[idx+1:]
	}
	return value
}

// GuixBackend implements PkgBackend using the Guix package commands.
type GuixBackend struct{}

// NewGuixBackend returns a Guix package backend.
func NewGuixBackend() *GuixBackend { return &GuixBackend{} }

// Name returns "guix".
func (b *GuixBackend) Name() string { return "guix" }

// Reload is a no-op for guix.
func (b *GuixBackend) Reload() {}

// List returns available guix packages.
func (b *GuixBackend) List() []Package {
	installed := guixInstalledSet()
	out, err := exec.Command("guix", "package", "--list-available").Output()
	if err != nil {
		return guixInstalledPackages(installed)
	}
	return parseGuixPackages(string(out), installed, b)
}

// Detail returns metadata for a guix package.
func (b *GuixBackend) Detail(name string) PackageDetail {
	detail := PackageDetail{Name: name, Repository: "guix"}
	out, err := exec.Command("guix", "show", name).Output()
	if err != nil {
		return detail
	}
	for _, line := range strings.Split(string(out), "\n") {
		key, value, ok := strings.Cut(line, ": ")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "name":
			detail.Name = strings.TrimSpace(value)
		case "version":
			detail.Version = strings.TrimSpace(value)
		case "synopsis":
			detail.ShortDesc = strings.TrimSpace(value)
		case "home-page":
			detail.Homepage = strings.TrimSpace(value)
		case "license":
			detail.License = strings.TrimSpace(value)
		}
	}
	return detail
}

// Install installs guix packages.
func (b *GuixBackend) Install(names []string, w io.Writer) (string, error) {
	return runCommand(w, append([]string{"guix", "install"}, names...)...)
}

// Remove removes guix packages.
func (b *GuixBackend) Remove(names []string, w io.Writer) (string, error) {
	return runCommand(w, append([]string{"guix", "remove"}, names...)...)
}

// Update updates guix packages.
func (b *GuixBackend) Update(w io.Writer) (string, error) {
	return runCommand(w, "guix", "upgrade")
}

// OpenURL opens a URL in the browser.
func (b *GuixBackend) OpenURL(url string) { OpenBrowser(url) }

func guixInstalledSet() map[string]bool {
	out, err := exec.Command("guix", "package", "--list-installed").Output()
	if err != nil {
		return nil
	}
	set := make(map[string]bool)
	for _, packageInfo := range parseGuixLines(string(out)) {
		set[packageInfo.name] = true
	}
	return set
}

func guixInstalledPackages(installed map[string]bool) []Package {
	packages := make([]Package, 0, len(installed))
	for name := range installed {
		packages = append(packages, Package{Name: name, Installed: true})
	}
	return packages
}

type guixPackageInfo struct {
	name    string
	version string
	desc    string
}

func parseGuixPackages(output string, installed map[string]bool, backend PkgBackend) []Package {
	lines := parseGuixLines(output)
	packages := make([]Package, 0, len(lines))
	for _, info := range lines {
		packages = append(packages, Package{
			Name:      info.name,
			Installed: installed[info.name],
			ShortDesc: info.desc,
			Backend:   backend,
		})
	}
	return packages
}

func parseGuixLines(output string) []guixPackageInfo {
	var packages []guixPackageInfo
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 || strings.HasPrefix(fields[0], "-") {
			continue
		}
		name, version := fields[0], fields[1]
		descriptionStart := 3
		if at := strings.LastIndexByte(name, '@'); at > 0 {
			version = name[at+1:]
			name = name[:at]
			descriptionStart = 2
		}
		desc := ""
		if len(fields) > descriptionStart {
			desc = strings.Join(fields[descriptionStart:], " ")
		}
		packages = append(packages, guixPackageInfo{name: name, version: version, desc: desc})
	}
	return packages
}

func runCommand(w io.Writer, args ...string) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("empty package command")
	}
	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec
	if w == nil && isTTY() {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return "", cmd.Run()
	}
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		_ = pr.Close()
		_ = pw.Close()
		return "", err
	}
	var output strings.Builder
	done := make(chan struct{})
	go func() {
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			line := scanner.Text()
			output.WriteString(line + "\n")
			if w != nil {
				_, _ = io.WriteString(w, line+"\n")
			}
		}
		close(done)
	}()
	err := cmd.Wait()
	_ = pw.Close()
	<-done
	_ = pr.Close()
	return strings.TrimSpace(output.String()), err
}
