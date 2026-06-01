package pkgman

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
)

const (
	appImageCatalogURL = "https://raw.githubusercontent.com/ivan-hc/AM/main/programs/x86_64-apps"
)

// appImageEntry holds parsed info from a single line of "am -f --byname" output.
// Format: ◆ name | version ✓ | type | size
type appImageEntry struct {
	Name    string
	Version string
	Type    string // "appimage", "appimage*", "dynamický-binární", …
	Size    string
}

// AppImageBackend implements PkgBackend for AppImage packages managed via AM/AppMan.
type AppImageBackend struct {
	mu      sync.Mutex
	catalog []Package         // HTTP catalog: name + description (all available apps)
	apps    map[string]appImageEntry // from am -f --byname (installed apps only)
}

func NewAppImageBackend() *AppImageBackend {
	return &AppImageBackend{}
}

func (b *AppImageBackend) Name() string { return "appimage" }

func (b *AppImageBackend) hasAM() bool {
	_, err := exec.LookPath("am")
	if err == nil {
		return true
	}
	_, err = exec.LookPath("appman")
	return err == nil
}

func (b *AppImageBackend) amCmd() string {
	if _, err := exec.LookPath("am"); err == nil {
		return "am"
	}
	return "appman"
}

// parseAMOutput parses the full "am -f --byname" table output.
// Returns a map of lowercase app name → parsed entry.
// All entries in this output are installed apps.
func parseAMOutput(data []byte) map[string]appImageEntry {
	apps := make(map[string]appImageEntry)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "◆ ") {
			continue
		}
		idx := strings.Index(line, "◆ ")
		if idx < 0 {
			continue
		}
		rest := strings.TrimSpace(line[idx+2:])
		parts := strings.Split(rest, "|")
		if len(parts) < 1 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		if name == "" {
			continue
		}
		entry := appImageEntry{Name: name}
		if len(parts) >= 2 {
			verField := strings.TrimSpace(parts[1])
			// Version may have " ✓" suffix meaning up-to-date
			entry.Version = strings.TrimSuffix(verField, " ✓")
			entry.Version = strings.TrimSpace(entry.Version)
			// If version was just "✓", it means installed but version unknown
			if entry.Version == "✓" {
				entry.Version = ""
			}
		}
		if len(parts) >= 3 {
			entry.Type = strings.TrimSpace(parts[2])
		}
		if len(parts) >= 4 {
			entry.Size = strings.TrimSpace(parts[3])
		}
		apps[strings.ToLower(name)] = entry
	}
	return apps
}

// queryInstalled runs am/appman -f --byname and returns installed apps with metadata.
func (b *AppImageBackend) queryInstalled() map[string]appImageEntry {
	if !b.hasAM() {
		return nil
	}
	out, err := exec.Command(b.amCmd(), "-f", "--byname").Output()
	if err != nil {
		return nil
	}
	return parseAMOutput(out)
}

// Reload clears all cached state so the next List() re-fetches everything.
func (b *AppImageBackend) Reload() {
	b.mu.Lock()
	b.catalog = nil
	b.apps = nil
	b.mu.Unlock()
}

// loadCatalog fetches the remote HTTP catalog if not yet cached.
func (b *AppImageBackend) loadCatalog() []Package {
	b.mu.Lock()
	if b.catalog != nil {
		defer b.mu.Unlock()
		return b.catalog
	}
	b.mu.Unlock()

	resp, err := http.Get(appImageCatalogURL)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var pkgs []Package
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "◆ ") {
			continue
		}
		rest := strings.TrimPrefix(line, "◆ ")
		parts := strings.SplitN(rest, " : ", 2)
		name := strings.TrimSpace(parts[0])
		desc := ""
		if len(parts) > 1 {
			desc = strings.TrimSpace(parts[1])
		}
		if name == "" {
			continue
		}
		pkgs = append(pkgs, Package{
			Name:      name,
			ShortDesc: desc,
		})
	}

	b.mu.Lock()
	b.catalog = pkgs
	b.mu.Unlock()
	return pkgs
}

// loadApps loads the installed app metadata from am/appman (cached).
func (b *AppImageBackend) loadApps() map[string]appImageEntry {
	b.mu.Lock()
	if b.apps != nil {
		defer b.mu.Unlock()
		return b.apps
	}
	b.mu.Unlock()

	apps := b.queryInstalled()

	b.mu.Lock()
	b.apps = apps
	b.mu.Unlock()
	return apps
}

// List returns the full app list with current installed state.
// HTTP catalog provides all available apps; am output provides installed state + metadata.
func (b *AppImageBackend) List() []Package {
	catalog := b.loadCatalog()
	apps := b.loadApps()

	pkgs := make([]Package, len(catalog))
	for i, p := range catalog {
		pkgs[i] = p
		if apps != nil {
			if entry, ok := apps[strings.ToLower(p.Name)]; ok {
				pkgs[i].Installed = true
				if pkgs[i].ShortDesc == "" && entry.Type != "" {
					pkgs[i].ShortDesc = entry.Type
				}
			}
		}
	}
	return pkgs
}

// Detail returns metadata for a single app. Installed apps get full info from am;
// non-installed apps get catalog description + homepage only.
func (b *AppImageBackend) Detail(name string) PackageDetail {
	d := PackageDetail{
		Name:       name,
		Repository: "AppImage",
		Homepage:   fmt.Sprintf("https://portable-linux-apps.github.io/apps/%s.html", name),
	}

	// Fill description from catalog
	catalog := b.loadCatalog()
	for _, p := range catalog {
		if strings.EqualFold(p.Name, name) {
			d.ShortDesc = p.ShortDesc
			break
		}
	}

	// Fill version/size/type from am if installed
	apps := b.loadApps()
	if apps != nil {
		if entry, ok := apps[strings.ToLower(name)]; ok {
			d.Version = entry.Version
			d.Architecture = entry.Type
			d.InstalledSize = entry.Size
		}
	}

	return d
}

func (b *AppImageBackend) Install(names []string, w io.Writer) (string, error) {
	if !b.hasAM() {
		return "", fmt.Errorf("%s", t("appimage.am_missing"))
	}
	cmd := b.amCmd()
	var output strings.Builder
	for _, name := range names {
		c := exec.Command(cmd, "-i", name)
		if w != nil {
			c.Stdout = w
			c.Stderr = w
		} else if isTTY() {
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
		}
		if err := c.Run(); err != nil {
			return output.String(), fmt.Errorf("install %s: %w", name, err)
		}
		output.WriteString(fmt.Sprintf("Installed %s\n", name))
	}
	return output.String(), nil
}

func (b *AppImageBackend) Remove(names []string, w io.Writer) (string, error) {
	if !b.hasAM() {
		return "", fmt.Errorf("%s", t("appimage.am_missing"))
	}
	cmd := b.amCmd()
	var output strings.Builder
	for _, name := range names {
		c := exec.Command(cmd, "-r", name)
		if w != nil {
			c.Stdout = w
			c.Stderr = w
		} else if isTTY() {
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
		}
		if err := c.Run(); err != nil {
			return output.String(), fmt.Errorf("remove %s: %w", name, err)
		}
		output.WriteString(fmt.Sprintf("Removed %s\n", name))
	}
	return output.String(), nil
}

func (b *AppImageBackend) Update(w io.Writer) (string, error) {
	if !b.hasAM() {
		return "", fmt.Errorf("%s", t("appimage.am_missing"))
	}
	cmd := b.amCmd()
	c := exec.Command(cmd, "-u")
	if w != nil {
		c.Stdout = w
		c.Stderr = w
	} else if isTTY() {
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
	}
	return "", c.Run()
}

func (b *AppImageBackend) OpenURL(url string) {
	OpenBrowser(url)
}
