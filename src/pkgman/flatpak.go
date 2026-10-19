package pkgman

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"strings"
)

// FlatpakBackend implements PkgBackend for Flatpak applications.
type FlatpakBackend struct{}

// NewFlatpakBackend creates a FlatpakBackend.
func NewFlatpakBackend() *FlatpakBackend { return &FlatpakBackend{} }

// Name returns "flatpak".
func (b *FlatpakBackend) Name() string { return "flatpak" }

// Reload is a no-op for flatpak.
func (b *FlatpakBackend) Reload() {}

// List returns available flatpak applications.
func (b *FlatpakBackend) List() []Package {
	installed := flatpakInstalledSet()
	var pkgs []Package
	out, err := exec.Command("flatpak", "remote-ls", "flathub", "--columns=application,name").Output()
	if err != nil {
		for id := range installed {
			pkgs = append(pkgs, Package{Name: id, DisplayName: id, Installed: true, ShortDesc: "Flatpak", Backend: b})
		}
		return pkgs
	}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		id := fields[0]
		name := strings.Join(fields[1:], " ")
		pkgs = append(pkgs, Package{
			Name:        id,
			DisplayName: name,
			ShortDesc:   name,
			Installed:   installed[id],
			Backend:     b,
		})
	}
	return pkgs
}

// Detail returns metadata for a flatpak application.
func (b *FlatpakBackend) Detail(name string) PackageDetail {
	d := PackageDetail{Name: name, Repository: "flathub", Homepage: "https://flathub.org/apps/" + name}
	cmd := exec.Command("flatpak", "info", name)
	cmd.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
	out, err := cmd.Output()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			line = strings.TrimSpace(line)
			if v, ok := strings.CutPrefix(line, "Version:"); ok {
				d.Version = strings.TrimSpace(v)
			}
			if v, ok := strings.CutPrefix(line, "Description:"); ok {
				d.ShortDesc = strings.TrimSpace(v)
			}
			if v, ok := strings.CutPrefix(line, "Installed Size:"); ok {
				d.InstalledSize = strings.TrimSpace(v)
			}
		}
		if d.ShortDesc == "" {
			for _, line := range strings.Split(string(out), "\n") {
				line = strings.TrimSpace(line)
				if line != "" {
					d.ShortDesc = line
					break
				}
			}
		}
	}
	return d
}

// Install installs flatpak applications.
func (b *FlatpakBackend) Install(names []string, w io.Writer) (string, error) {
	args := append([]string{"install", "-y", "flathub"}, names...)
	return runCmdStream(w, "flatpak", args...)
}

// Remove removes flatpak applications.
func (b *FlatpakBackend) Remove(names []string, w io.Writer) (string, error) {
	args := append([]string{"uninstall", "-y"}, names...)
	return runCmdStream(w, "flatpak", args...)
}

// Update updates all flatpak applications.
func (b *FlatpakBackend) Update(w io.Writer) (string, error) {
	return runCmdStream(w, "flatpak", "update", "-y")
}

// OpenURL opens a URL in the browser.
func (b *FlatpakBackend) OpenURL(u string) {
	OpenBrowser(u)
}

func flatpakInstalledSet() map[string]bool {
	set := make(map[string]bool)
	out, err := exec.Command("flatpak", "list", "--app", "--columns=application").Output()
	if err != nil {
		return set
	}
	for _, line := range strings.Split(string(out), "\n") {
		if id := strings.TrimSpace(line); id != "" {
			set[id] = true
		}
	}
	return set
}

func runCmdStream(w io.Writer, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
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
	_ = pw.Close()
	var buf strings.Builder
	sc := bufio.NewScanner(pr)
	for sc.Scan() {
		line := sc.Text()
		buf.WriteString(line + "\n")
		if w != nil {
			_, _ = io.WriteString(w, line+"\n")
		}
	}
	_ = pr.Close()
	err = cmd.Wait()
	return strings.TrimSpace(buf.String()), err
}
