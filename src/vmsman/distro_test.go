package vmman

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"
)

func testApp(t *testing.T) *DistroApp {
	t.Helper()
	t.Setenv("APPIMAGE", "")
	root := t.TempDir()
	a := &DistroApp{ctx: context.Background(), source: filepath.Join(root, "source tools's"), executable: filepath.Join(root, "dh's frontend"), configDir: filepath.Join(root, "config"), cfg: DistroConfig{VMs: filepath.Join(root, "VM directory's")}}
	a.icons = filepath.Join(a.configDir, "icons")
	for _, p := range []string{a.source, a.cfg.VMs, a.icons, filepath.Join(a.configDir, "ready"), filepath.Join(a.configDir, "supported")} {
		if err := os.MkdirAll(p, 0755); err != nil {
			t.Fatal(err)
		}
	}
	return a
}

func put(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0755); err != nil {
		t.Fatal(err)
	}
}

func fixture(t *testing.T, a *DistroApp) {
	t.Helper()
	put(t, filepath.Join(a.source, "qget"), `#!/bin/bash
set -eu
if [[ $1 == --list-os ]]; then printf 'fixture\nno-editions\n'; exit; fi
if [[ $1 != --show ]]; then printf '%s\n' "$PWD" "$@" > "$DOWNLOAD_LOG"; exit; fi
arch=amd64
release=''
name=${!#}
while (($#)); do
 case $1 in --arch) arch=$2;shift;; --release) release=$2;shift;; esac
 shift
done
printf 'Archs: amd64 arm64\n'
if [[ $arch == amd64 ]]; then echo 'Releases: old new-lts'; else echo 'Releases: arm-only'; fi
if [[ $name != no-editions ]]; then
 if [[ $release == new-lts ]]; then printf "EditionsList:\nBrazilian Portuguese\nDeveloper's Edition\n";
 elif [[ $release == arm-only ]]; then printf 'EditionsList:\nconsole\n'; fi
fi
echo 'Description: picker fixture'
`)
}

func TestParseMetadataPorted(t *testing.T) {
	m := parseMetadata("\nDemo OS\nArchs: amd64 arm64\nReleases: old new\nEditions: Brazilian Portuguese desktop\nEditionsList:\nBrazilian Portuguese\ndesktop\nRSS: https://example.test\nDescription: A: description\n")
	if !reflect.DeepEqual(m, Metadata{Name: "Demo OS", Description: "A: description", Architectures: []string{"amd64", "arm64"}, Releases: []string{"old", "new"}, Editions: []string{"Brazilian Portuguese", "desktop"}}) {
		t.Fatalf("%+v", m)
	}
}

func TestTemplatePathQuotingPorted(t *testing.T) {
	a := testApp(t)
	os.MkdirAll(filepath.Join(a.source, "templates"), 0755)
	put(t, filepath.Join(a.source, "templates", "ubuntu-budgie"), `DESCRIPTION="Description for $OS"; ICON=budgie.svg`)
	d, i := a.templateInfo("ubuntu-budgie-24.04")
	if d != "Description for ubuntu-budgie" || i != "budgie.svg" {
		t.Fatalf("%q %q", d, i)
	}
}

func TestDesktopQuotePorted(t *testing.T) {
	cases := map[string]string{"plain": `"plain"`, "with space": `"with space"`, "100%": `"100%%"`, "a\\b": `"a\\\\b"`, "a\"$`": `"a\\"\\$\\` + "`" + `"`}
	for input, want := range cases {
		if got := desktopQuote(input); got != want {
			t.Errorf("%q: got %q want %q", input, got, want)
		}
	}
}

func TestDesktopPublicationPorted(t *testing.T) {
	a := testApp(t)
	target := filepath.Join(a.configDir, "ready", "single-ready.desktop")
	for i := 0; i < 10; i++ {
		if err := a.desktop("ready", "single-ready", "single-ready", "a\ncomment", "true"); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(b), "[Desktop Entry]\n") || !strings.Contains(string(b), "Comment=a\\ncomment\n") || !strings.Contains(string(b), "Exec=true\n") {
			t.Fatal(string(b))
		}
		st, _ := os.Stat(target)
		if st.Mode().Perm() != 0755 {
			t.Fatal(st.Mode())
		}
	}
	files, _ := filepath.Glob(filepath.Join(a.configDir, "ready", ".dh-entry-*"))
	if len(files) != 0 {
		t.Fatal(files)
	}
	for _, bad := range []string{"../escape", "a/b", "evil\nExec=bad", "-option"} {
		if a.desktop("ready", bad, bad, "", "") == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
}

func TestAppImageEntriesPorted(t *testing.T) {
	a := testApp(t)
	fixture(t, a)
	image := filepath.Join(t.TempDir(), "Distro Hopper.AppImage")
	put(t, image, "image")
	t.Setenv("APPIMAGE", image)
	put(t, filepath.Join(a.cfg.VMs, "fixture.conf"), "disk_img=x")
	if err := a.renewSupported(nil); err != nil {
		t.Fatal(err)
	}
	for kind, action := range map[string]string{"supported": "create", "ready": "run"} {
		data, err := os.ReadFile(filepath.Join(a.configDir, kind, "fixture.desktop"))
		if err != nil {
			t.Fatal(err)
		}
		var want string
		if kind == "supported" {
			want = desktopQuote("vmsman-create") + " " + action + " " + desktopQuote("fixture")
		} else {
			want = desktopQuote(image) + " " + action + " " + desktopQuote("fixture")
		}
		if !strings.Contains(string(data), "Exec="+want+"\n") {
			t.Fatal(string(data))
		}
	}
}

func TestReadyPorted(t *testing.T) {
	a := testApp(t)
	put(t, filepath.Join(a.cfg.VMs, "ubuntu-24.04.conf"), "disk_img=x")
	put(t, filepath.Join(a.cfg.VMs, "distrohopper.conf"), "")
	put(t, filepath.Join(a.configDir, "ready", "stale.desktop"), "stale")
	if err := a.renewReady(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(a.configDir, "ready", "ubuntu-24.04.desktop"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "--vm "+desktopQuote(filepath.Join(a.cfg.VMs, "ubuntu-24.04.conf"))) {
		t.Fatal(string(b))
	}
	entries, _ := os.ReadDir(filepath.Join(a.configDir, "ready"))
	if len(entries) != 1 {
		t.Fatal(entries)
	}
}

func TestPickerTuplesPorted(t *testing.T) {
	a := testApp(t)
	fixture(t, a)
	m, err := a.query("fixture", "arm64", "")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.Releases, []string{"arm-only"}) {
		t.Fatal(m)
	}
	m, err = a.query("fixture", "amd64", "new-lts")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.Editions, []string{"Brazilian Portuguese", "Developer's Edition"}) {
		t.Fatal(m)
	}
	m, err = a.query("no-editions", "amd64", "old")
	if err != nil || len(m.Editions) != 0 {
		t.Fatalf("%+v %v", m, err)
	}
}

func TestDownloadPorted(t *testing.T) {
	a := testApp(t)
	fixture(t, a)
	log := filepath.Join(t.TempDir(), "download.log")
	t.Setenv("DOWNLOAD_LOG", log)
	// fixture qget doesn't use DOWNLOAD_LOG unless modified; use env in qget? Our fixture writes to DOWNLOAD_LOG env.
	for _, tuple := range [][4]string{{"fixture", "amd64", "new-lts", "Brazilian Portuguese"}, {"fixture", "amd64", "new-lts", "Developer's Edition"}, {"fixture", "arm64", "arm-only", "console"}, {"no-editions", "amd64", "old", ""}} {
		if err := a.download(tuple[0], tuple[1], tuple[2], tuple[3], nil); err != nil {
			t.Fatal(err)
		}
		b, _ := os.ReadFile(log)
		want := a.cfg.VMs + "\n--arch\n" + tuple[1] + "\n" + tuple[0] + "\n" + tuple[2] + "\n"
		if tuple[3] != "" {
			want += tuple[3] + "\n"
		}
		if string(b) != want {
			t.Fatalf("got %q want %q", b, want)
		}
		os.Remove(log)
	}
	for _, tuple := range [][4]string{{"fixture", "arm64", "new-lts", "Brazilian Portuguese"}, {"fixture", "amd64", "new-lts", "missing"}, {"no-editions", "amd64", "old", "unexpected"}, {"fixture", "unknown", "old", ""}} {
		if err := a.download(tuple[0], tuple[1], tuple[2], tuple[3], nil); err == nil {
			t.Fatalf("accepted %v", tuple)
		}
	}
	if _, err := os.Stat(log); err == nil {
		t.Fatal("invalid selection downloaded")
	}
}

func TestRenewPorted(t *testing.T) {
	a := testApp(t)
	fixture(t, a)
	var events []int
	if err := a.renewSupported(func(done, total int, name string) {
		if total != 3 {
			t.Errorf("total=%d", total)
		}
		events = append(events, done)
	}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(events, []int{0, 1, 2, 3}) {
		t.Fatal(events)
	}
	for _, name := range []string{"fixture", "no-editions"} {
		b, err := os.ReadFile(filepath.Join(a.configDir, "supported", name+".desktop"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), desktopQuote("vmsman-create")+" create "+desktopQuote(name)) || strings.Contains(string(b), "yad") {
			t.Fatal(string(b))
		}
	}
	t.Setenv("DH_RENEW_JOBS", "0")
	if a.renewSupported(nil) == nil {
		t.Fatal("accepted invalid jobs")
	}
}

func TestRenewPreservesEntriesOnFailurePorted(t *testing.T) {
	a := testApp(t)
	fixture(t, a)
	path := filepath.Join(a.configDir, "supported", "fixture.desktop")
	put(t, path, "previous entry")
	put(t, filepath.Join(a.source, "qget"), "#!/bin/sh\nif [ \"$1\" = --list-os ]; then echo fixture; else exit 1; fi\n")
	if a.renewSupported(nil) == nil {
		t.Fatal("ignored worker failure")
	}
	b, _ := os.ReadFile(path)
	if string(b) != "previous entry" {
		t.Fatal(string(b))
	}
}

func TestWorkerPoolRefillsPorted(t *testing.T) {
	a := testApp(t)
	t.Setenv("DH_RENEW_JOBS", "2")
	barrier := filepath.Join(t.TempDir(), "third-started")
	t.Setenv("BARRIER", barrier)
	put(t, filepath.Join(a.source, "qget"), `#!/bin/bash
if [[ $1 == --list-os ]]; then printf 'slow\nfast\nthird\n'; exit; fi
name=${!#}
if [[ $name == slow ]]; then
 for ((i=0;i<100;i++)); do [[ -f $BARRIER ]] && break; sleep .03; done
 [[ -f $BARRIER ]] || exit 1
elif [[ $name == third ]]; then touch "$BARRIER"; fi
printf 'Archs: x86_64\nReleases: latest\nDescription: fixture\n'
`)
	if err := a.renewSupported(nil); err != nil {
		t.Fatal("pool waited in launch order:", err)
	}
}

func TestRenewLockPorted(t *testing.T) {
	a := testApp(t)
	fixture(t, a)
	path := filepath.Join(a.configDir, "renew.lock")
	lock, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatal(err)
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		t.Fatal(err)
	}
	if err := a.renewSupported(nil); err == nil {
		t.Fatal("concurrent regeneration accepted")
	}
	if !isFile(path) {
		t.Fatal("removed another worker's lock")
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_UN); err != nil {
		t.Fatal(err)
	}
	if err := a.renewSupported(nil); err != nil {
		t.Fatal("lock was not released:", err)
	}
}

func TestCancellationPorted(t *testing.T) {
	a := testApp(t)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	a.ctx = ctx
	put(t, filepath.Join(a.source, "qget"), "#!/bin/bash\nexec sleep 30\n")
	start := time.Now()
	if _, err := a.query("fixture", "", ""); err == nil {
		t.Fatal("ignored cancellation")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("cancellation too slow")
	}
}

func TestCancellationKillsChildrenPorted(t *testing.T) {
	a := testApp(t)
	marker := filepath.Join(t.TempDir(), "orphan-downloader")
	t.Setenv("CHILD_MARKER", marker)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	a.ctx = ctx
	put(t, filepath.Join(a.source, "qget"), "#!/bin/bash\n(sleep .4; touch \"$CHILD_MARKER\") &\nwait\n")
	if _, err := a.query("fixture", "", ""); err == nil {
		t.Fatal("ignored cancellation")
	}
	time.Sleep(500 * time.Millisecond)
	if isFile(marker) {
		t.Fatal("backend child survived cancellation")
	}
}

func TestPrettyNamesKeepTemplateIDsPorted(t *testing.T) {
	a := testApp(t)
	if err := os.MkdirAll(filepath.Join(a.source, "templates"), 0755); err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(a.source, "templates", "alpine-linux"), `PRETTY="Alpine Linux"; DESCRIPTION="Small and secure"; ICON=alpine-linux.svg`)
	log := filepath.Join(t.TempDir(), "download.log")
	t.Setenv("DOWNLOAD_LOG", log)
	put(t, filepath.Join(a.source, "qget"), `#!/bin/bash
set -eu
if [[ $1 == --list-os ]]; then echo alpine-linux; exit; fi
if [[ $1 == --show ]]; then
 [[ ${!#} == alpine-linux ]] || { echo 'wrong template ID' >&2; exit 1; }
 printf ' Alpine Linux\nArchs: amd64\nReleases: v3.24\nEditionsList:\nstandard\nDescription: Small and secure\n'
else
 [[ $3 == alpine-linux ]] || { echo 'pretty name passed as argument' >&2; exit 1; }
 printf '%s\n' "$@" > "$DOWNLOAD_LOG"
fi
`)
	systems, err := a.systems()
	if err != nil {
		t.Fatal(err)
	}
	if len(systems) != 1 || systems[0].ID != "alpine-linux" || systems[0].Name != "Alpine Linux" {
		t.Fatalf("%+v", systems)
	}
	if err := a.renewSupported(nil); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(a.configDir, "supported", "alpine-linux.desktop"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "Name=Alpine Linux\n") || !strings.Contains(string(b), " create "+desktopQuote("alpine-linux")) {
		t.Fatal(string(b))
	}
	if err := a.download(systems[0].ID, "amd64", "v3.24", "standard", nil); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(log)
	if string(b) != "--arch\namd64\nalpine-linux\nv3.24\nstandard\n" {
		t.Fatalf("backend argv: %q", b)
	}
	if _, err := a.query("Alpine Linux", "", ""); err == nil {
		t.Fatal("accepted pretty name as template ID")
	}
}
