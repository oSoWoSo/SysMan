package pkgman

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"codeberg.org/oSoWoSo/SysMan/src/common"
)

func TestPkgnameFromFull_Basic(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"vim-9.2.0_1", "vim"},
		{"bash-5.2.0", "bash"},
		{"glibc-2.38_1", "glibc"},
		{"no-version", "no-version"},
		{"pkg-1.0.0_2", "pkg"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := pkgnameFromFull(tt.input); got != tt.want {
				t.Errorf("pkgnameFromFull(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestPkgnameFromFull_VersionWithDash(t *testing.T) {
	// Test versions that start with digit after dash
	tests := []struct {
		input string
		want  string
	}{
		{"abc-123", "abc"},
		{"pkg-0.1.0", "pkg"},
		{"lib-2.3.4_5", "lib"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := pkgnameFromFull(tt.input); got != tt.want {
				t.Errorf("pkgnameFromFull(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFilterModes(t *testing.T) {
	// Verify FilterMode constants
	if FilterAll != 0 {
		t.Errorf("FilterAll = %d, want 0", FilterAll)
	}
	if FilterInstalled != 1 {
		t.Errorf("FilterInstalled = %d, want 1", FilterInstalled)
	}
	if FilterAvailable != 2 {
		t.Errorf("FilterAvailable = %d, want 2", FilterAvailable)
	}
}

func TestReadAppIconfig(t *testing.T) {
	if got := readAppIconfig(filepath.Join(t.TempDir(), "missing")); got != "" {
		t.Errorf("missing file: got %q, want empty", got)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "appman-config")
	if err := os.WriteFile(path, []byte("\n\n/home/user/.local/AppImages\nignored\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readAppIconfig(path); got != "/home/user/.local/AppImages" {
		t.Errorf("got %q, want first non-empty line", got)
	}
	if err := os.WriteFile(path, []byte("   \n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readAppIconfig(path); got != "" {
		t.Errorf("blank file: got %q, want empty", got)
	}
}

func TestDefaultAppImageDir_Sources(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	if got := DefaultAppImageDir(); got != filepath.Join(tmp, "Applications") {
		t.Fatalf("no config: got %q, want %q", got, filepath.Join(tmp, "Applications"))
	}
	appman := filepath.Join(tmp, ".config", "appman", "appman-config")
	if err := os.MkdirAll(filepath.Dir(appman), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(appman, []byte("/from/appman\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := DefaultAppImageDir(); got != "/from/appman" {
		t.Errorf("appman config: got %q, want /from/appman", got)
	}
	am := filepath.Join(tmp, ".config", "AM", "appman-config")
	if err := os.MkdirAll(filepath.Dir(am), 0o755); err != nil {
		t.Fatal(err)
	}
	os.Remove(appman)
	if err := os.WriteFile(am, []byte("  /from/am\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := DefaultAppImageDir(); got != "/from/am" {
		t.Errorf("AM config: got %q, want /from/am (trimmed)", got)
	}
}

func TestEffectiveAppImageDir(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if got := effectiveAppImageDir(""); got != DefaultAppImageDir() {
		t.Errorf("empty override: got %q, want default %q", got, DefaultAppImageDir())
	}
	if got := effectiveAppImageDir("  /custom/path  "); got != "/custom/path" {
		t.Errorf("override: got %q, want /custom/path", got)
	}
}

func TestScanInstalledApps(t *testing.T) {
	dir := t.TempDir()
	// AM per-app layout: subdir with a `remove` script marks an installed app.
	for _, name := range []string{"alma", "RustDesk"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name, "remove"), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Plain dir without a remove script is not an app.
	if err := os.Mkdir(filepath.Join(dir, "BACKUP"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Launcher layout: a lone .AppImage file is an app too.
	for _, f := range []string{"Firefox.AppImage", "chrome.Appimage", "note.txt"} {
		if err := os.WriteFile(filepath.Join(dir, f), []byte("x"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if got := scanInstalledApps(filepath.Join(t.TempDir(), "missing")); got != nil {
		t.Errorf("missing dir: got %v, want nil", got)
	}
	apps := scanInstalledApps(dir)
	if len(apps) != 4 {
		t.Fatalf("expected 4 apps, got %d: %v", len(apps), apps)
	}
	for name, want := range map[string]string{
		"alma":     filepath.Join(dir, "alma"),
		"rustdesk": filepath.Join(dir, "RustDesk"),
		"firefox":  filepath.Join(dir, "Firefox.AppImage"),
		"chrome":   filepath.Join(dir, "chrome.Appimage"),
	} {
		if apps[name] != want {
			t.Errorf("app %q: got %q, want %q", name, apps[name], want)
		}
	}
	if apps["backup"] != "" || apps["note"] != "" {
		t.Errorf("BACKUP/note.txt should not be apps: %v", apps)
	}
	if got := scanInstalledApps(t.TempDir()); got == nil || len(got) != 0 {
		t.Errorf("empty dir: got %v, want empty map", got)
	}
}

func TestScanAllInstalledApps(t *testing.T) {
	user := t.TempDir()
	sys := t.TempDir()
	for _, name := range []string{"alma", "rustdesk"} {
		if err := os.MkdirAll(filepath.Join(user, name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(user, name, "remove"), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// System-wide am installs apps under /opt, including am's own dir.
	for _, name := range []string{"steam", "am"} {
		if err := os.MkdirAll(filepath.Join(sys, name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(sys, name, "remove"), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	apps := scanAllInstalledApps(user, sys)
	if len(apps) != 3 {
		t.Fatalf("expected 3 apps, got %d: %v", len(apps), apps)
	}
	if apps["alma"] != filepath.Join(user, "alma") {
		t.Errorf("user root should win: %v", apps["alma"])
	}
	if apps["steam"] != filepath.Join(sys, "steam") {
		t.Errorf("system root missing: %v", apps["steam"])
	}
	if apps["am"] != "" {
		t.Errorf("system-wide am dir should be skipped: %v", apps["am"])
	}

	// Single unreadable root → nil.
	if got := scanAllInstalledApps(filepath.Join(t.TempDir(), "missing")); got != nil {
		t.Errorf("unreadable roots: got %v, want nil", got)
	}
}

func TestDefaultAppImageDir_RelativeConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	cfg := filepath.Join(tmp, ".config", "appman", "appman-config")
	if err := os.MkdirAll(filepath.Dir(cfg), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("AppImages\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := DefaultAppImageDir(); got != filepath.Join(tmp, "AppImages") {
		t.Errorf("relative config: got %q, want %q", got, filepath.Join(tmp, "AppImages"))
	}
}

func TestUniqueSortedRepos(t *testing.T) {
	got := uniqueSortedRepos([]string{" b", "a", "", " b", "A"})
	if len(got) != 3 {
		t.Fatalf("expected 3, got %v", got)
	}
	for i, exp := range []string{"A", "a", "b"} {
		if got[i] != exp {
			t.Errorf("index %d: got %q, want %q", i, got[i], exp)
		}
	}
}

func TestReposConfContent(t *testing.T) {
	repos := []string{"https://repo.example.com/current", "https://repo.example.com/x86_64"}
	content := reposConfContent(repos)
	want := "repository=https://repo.example.com/current\nrepository=https://repo.example.com/x86_64\n"
	if content != want {
		t.Errorf("content:\n%q\nwant:\n%q", content, want)
	}
	if got := reposConfContent([]string{"", " "}); got != "" {
		t.Errorf("empty repos: got %q, want empty", got)
	}
}

func TestLoadReposConf(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, defaultReposFile)
	if got := loadReposConf(dir, defaultReposFile); got != nil {
		t.Errorf("missing file: got %v, want nil", got)
	}
	if err := os.WriteFile(path, []byte("# comment\nrepository=https://one\n\nsomething=else\nrepository= https://two \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := loadReposConf(dir, defaultReposFile)
	if len(got) != 2 || got[0] != "https://one" || got[1] != "https://two" {
		t.Errorf("parsed repos: %v", got)
	}
}

func TestWriteReposConf_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	repos := []string{"https://two", "https://one"}
	if err := writeReposConf(dir, "custom.conf", repos); err != nil {
		t.Fatal(err)
	}
	got := loadReposConf(dir, "custom.conf")
	want := []string{"https://one", "https://two"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %q, want %q", i, got[i], want[i])
		}
	}
	sort.Strings(repos)
	wrote, err := os.ReadFile(filepath.Join(dir, "custom.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if string(wrote) != reposConfContent(repos) {
		t.Errorf("file content mismatch: %q", string(wrote))
	}
	// Overwrite (rename path) again to ensure it is idempotent.
	if err := writeReposConf(dir, "custom.conf", []string{"https://one"}); err != nil {
		t.Fatal(err)
	}
	if got := loadReposConf(dir, "custom.conf"); len(got) != 1 || got[0] != "https://one" {
		t.Errorf("after overwrite: %v", got)
	}
}

func TestSysmanConfigRoundTrip_PkgmanFields(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	cfg := common.LoadSysManConfig()
	cfg.Pkgman.AppImageDir = "/apps/appimages"
	cfg.Pkgman.Repos = []string{"https://a", "https://b"}
	cfg.Pkgman.ReposFile = "custom.conf"
	if err := common.SaveSysManConfig(cfg); err != nil {
		t.Fatal(err)
	}
	got := common.LoadSysManConfig()
	if got.Pkgman.AppImageDir != "/apps/appimages" {
		t.Errorf("AppImageDir: got %q", got.Pkgman.AppImageDir)
	}
	if got.Pkgman.ReposFile != "custom.conf" {
		t.Errorf("ReposFile: got %q", got.Pkgman.ReposFile)
	}
	if len(got.Pkgman.Repos) != 2 || got.Pkgman.Repos[0] != "https://a" || got.Pkgman.Repos[1] != "https://b" {
		t.Errorf("Repos: got %v", got.Pkgman.Repos)
	}

	// Empty values survive a save without clobbering.
	cfg2 := common.LoadSysManConfig()
	cfg2.Pkgman.AppImageDir = ""
	cfg2.Pkgman.ReposFile = ""
	cfg2.Pkgman.Repos = nil
	if err := common.SaveSysManConfig(cfg2); err != nil {
		t.Fatal(err)
	}
	got2 := common.LoadSysManConfig()
	if len(got2.Pkgman.Repos) != 2 || got2.Pkgman.AppImageDir != "/apps/appimages" || got2.Pkgman.ReposFile != "custom.conf" {
		t.Errorf("empty save clobbered Pkgman fields: %+v", got2.Pkgman)
	}
}

func TestBuildOps(t *testing.T) {
	t.Run("queue only", func(t *testing.T) {
		q := []QueueEntry{
			{Name: "vim", Action: "install"},
			{Name: "old", Action: "remove"},
		}
		installs, removes := buildOps(q, nil)
		if len(installs) != 1 || installs[0] != "vim" {
			t.Errorf("installs: %v", installs)
		}
		if len(removes) != 1 || removes[0] != "old" {
			t.Errorf("removes: %v", removes)
		}
	})

	t.Run("selection not queued is added", func(t *testing.T) {
		q := []QueueEntry{{Name: "vim", Action: "install"}}
		installs, removes := buildOps(q, &Package{Name: "firefox", Installed: false})
		if len(installs) != 2 || installs[1] != "firefox" {
			t.Errorf("installs: %v", installs)
		}
		if len(removes) != 0 {
			t.Errorf("removes: %v", removes)
		}
		installs2, removes2 := buildOps(nil, &Package{Name: "firefox", Installed: true})
		if len(installs2) != 0 {
			t.Errorf("installs2: %v", installs2)
		}
		if len(removes2) != 1 || removes2[0] != "firefox" {
			t.Errorf("removes2: %v", removes2)
		}
	})

	t.Run("already queued is not duplicated", func(t *testing.T) {
		q := []QueueEntry{{Name: "firefox", Action: "install"}}
		installs, removes := buildOps(q, &Package{Name: "firefox", Installed: false})
		if len(installs) != 1 || len(removes) != 0 {
			t.Errorf("installs: %v removes: %v", installs, removes)
		}
	})

	t.Run("empty", func(t *testing.T) {
		installs, removes := buildOps(nil, nil)
		if len(installs) != 0 || len(removes) != 0 {
			t.Errorf("installs: %v removes: %v", installs, removes)
		}
	})
}

func TestAppDirSize(t *testing.T) {
	if got := appDirSize(filepath.Join(t.TempDir(), "missing")); got != 0 {
		t.Errorf("missing path: got %d, want 0", got)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.bin"), []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "sub", "b.bin"), []byte("1234567890"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := appDirSize(dir); got != 15 {
		t.Errorf("dir size: got %d, want 15", got)
	}

	file := filepath.Join(dir, "a.bin")
	if got := appDirSize(file); got != 5 {
		t.Errorf("file size: got %d, want 5", got)
	}
}

func TestHumanSize(t *testing.T) {
	for n, want := range map[int64]string{
		0:         "0 B",
		512:       "512 B",
		1024:      "1.0 KB",
		1536:      "1.5 KB",
		1048576:   "1.0 MB",
		536870912: "512.0 MB",
	} {
		if got := humanSize(n); got != want {
			t.Errorf("humanSize(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestInstalledSizeFor_FromDir(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "alma")
	if err := os.MkdirAll(app, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "remove"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "bin"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}

	b := NewAppImageBackendWithDir(dir)
	if got := b.installedSizeFor("Alma"); got != "2.0 KB" {
		t.Errorf("installedSizeFor from dir: got %q, want 2.0 KB", got)
	}
	if got := b.installedSizeFor("unknown"); got != "" {
		t.Errorf("installedSizeFor unknown: got %q, want empty", got)
	}
}

func TestInstalledSizeFor_FallbackToAM(t *testing.T) {
	b := NewAppImageBackendWithDir(filepath.Join(t.TempDir(), "empty"))
	b.apps = map[string]appImageEntry{
		"firefox": {Name: "firefox", Size: "150 MB"},
	}
	if got := b.installedSizeFor("firefox"); got != "150 MB" {
		t.Errorf("installedSizeFor am fallback: got %q, want 150 MB", got)
	}
}

func TestAppImageDetailSize(t *testing.T) {
	dir := t.TempDir()
	app := filepath.Join(dir, "RustDesk")
	if err := os.MkdirAll(app, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "remove"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "bin"), make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	b := NewAppImageBackendWithDir(dir)
	b.catalog = []Package{} // avoid the HTTP catalog fetch in Detail()
	d := b.Detail("RustDesk")
	if d.InstalledSize != "4.0 KB" {
		t.Errorf("Detail size: got %q, want 4.0 KB", d.InstalledSize)
	}
	if d.Architecture != "AppImage" {
		t.Errorf("Detail architecture: got %q, want AppImage", d.Architecture)
	}
}

func TestAppImageDetailSize_NotInstalled(t *testing.T) {
	dir := t.TempDir()
	b := NewAppImageBackendWithDir(dir)
	b.catalog = []Package{} // avoid the HTTP catalog fetch in Detail()
	b.apps = map[string]appImageEntry{}
	if d := b.Detail("firefox"); d.InstalledSize != "" {
		t.Errorf("Detail size of non-installed app: got %q, want empty", d.InstalledSize)
	}
}
