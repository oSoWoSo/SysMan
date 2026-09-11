package serman

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsSymlink_RegularFile(t *testing.T) {
	dir := t.TempDir()
	regular := filepath.Join(dir, "regular")
	if err := os.WriteFile(regular, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if isSymlink(regular) {
		t.Error("regular file should not be detected as symlink")
	}
}

func TestIsSymlink_Symlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if !isSymlink(link) {
		t.Error("symlink should be detected as symlink")
	}
}

func TestIsSymlink_NonExistent(t *testing.T) {
	if isSymlink("/nonexistent/path/xyz") {
		t.Error("non-existent path should not be detected as symlink")
	}
}

func TestIsSymlink_Directory(t *testing.T) {
	dir := t.TempDir()
	if isSymlink(dir) {
		t.Error("plain directory should not be detected as symlink")
	}
}

func TestLoadServices_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	dest := t.TempDir()
	svcs := LoadServices(dir, dest)
	if len(svcs) != 0 {
		t.Errorf("expected 0 services in empty dir, got %d", len(svcs))
	}
}

func TestLoadServices_NonExistentDir(t *testing.T) {
	svcs := LoadServices("/nonexistent/path", "/nonexistent/dest")
	if svcs != nil {
		t.Errorf("expected nil for non-existent dir, got %v", svcs)
	}
}

func TestLoadServices_SkipsNonDirectories(t *testing.T) {
	dir := t.TempDir()
	dest := t.TempDir()

	// regular file — should be skipped
	if err := os.WriteFile(filepath.Join(dir, "notadir.conf"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// service directory — should be included
	if err := os.Mkdir(filepath.Join(dir, "myservice"), 0o755); err != nil {
		t.Fatal(err)
	}

	svcs := LoadServices(dir, dest)
	if len(svcs) != 1 {
		t.Fatalf("expected 1 service, got %d", len(svcs))
	}
	if svcs[0].Name != "myservice" {
		t.Errorf("expected 'myservice', got %q", svcs[0].Name)
	}
}

func TestLoadServices_EnabledViaSymlink(t *testing.T) {
	dir := t.TempDir()
	dest := t.TempDir()

	for _, name := range []string{"alpha", "beta"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// enable "alpha" via symlink in dest
	if err := os.Symlink(filepath.Join(dir, "alpha"), filepath.Join(dest, "alpha")); err != nil {
		t.Fatal(err)
	}

	svcs := LoadServices(dir, dest)
	if len(svcs) != 2 {
		t.Fatalf("expected 2 services, got %d", len(svcs))
	}
	// sorted: alpha, beta
	if svcs[0].Name != "alpha" || !svcs[0].Enabled {
		t.Errorf("expected alpha enabled, got %+v", svcs[0])
	}
	if svcs[1].Name != "beta" || svcs[1].Enabled {
		t.Errorf("expected beta disabled, got %+v", svcs[1])
	}
}

func TestLoadServices_SortedAlphabetically(t *testing.T) {
	dir := t.TempDir()
	dest := t.TempDir()

	for _, name := range []string{"zebra", "apple", "mango"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	svcs := LoadServices(dir, dest)
	if len(svcs) != 3 {
		t.Fatalf("expected 3 services, got %d", len(svcs))
	}
	expected := []string{"apple", "mango", "zebra"}
	for i, exp := range expected {
		if svcs[i].Name != exp {
			t.Errorf("index %d: expected %q, got %q", i, exp, svcs[i].Name)
		}
	}
}

func TestLoadServices_AllDisabledWhenDestEmpty(t *testing.T) {
	dir := t.TempDir()
	dest := t.TempDir()

	for _, name := range []string{"svc1", "svc2"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	svcs := LoadServices(dir, dest)
	for _, svc := range svcs {
		if svc.Enabled {
			t.Errorf("expected %q to be disabled", svc.Name)
		}
	}
}

// TestLoadServices_LiveDirOnly verifies that services living directly in the
// enabled-services directory (per-user runsvdir/turnstile layout) are found
// even when there is no separate definitions directory.
func TestLoadServices_LiveDirOnly(t *testing.T) {
	dest := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dest, "inline"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/srv/linked", filepath.Join(dest, "linked")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dest, "stray"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	svcs := LoadServices("/nonexistent/service", dest)
	if len(svcs) != 2 {
		t.Fatalf("expected 2 services from live dir, got %v", svcs)
	}
	for _, svc := range svcs {
		if svc.Name != "inline" && svc.Name != "linked" {
			t.Errorf("unexpected service %q", svc.Name)
		}
		if !svc.Enabled {
			t.Errorf("expected %q enabled (live dir entry)", svc.Name)
		}
	}
}

// TestLoadServices_DestEntryIncluded ensures live-dir entries not present in the
// definitions directory are still listed (and marked enabled) alongside
// definitions-dir services.
func TestLoadServices_DestEntryIncluded(t *testing.T) {
	dir := t.TempDir()
	dest := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "defs-only"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "live-src"), filepath.Join(dest, "enabled-only")); err != nil {
		t.Fatal(err)
	}

	svcs := LoadServices(dir, dest)
	if len(svcs) != 2 {
		t.Fatalf("expected 2 services, got %v", svcs)
	}
	if svcs[0].Name != "defs-only" || svcs[0].Enabled {
		t.Errorf("expected 'defs-only' disabled first, got %+v", svcs[0])
	}
	if svcs[1].Name != "enabled-only" || !svcs[1].Enabled {
		t.Errorf("expected 'enabled-only' enabled second, got %+v", svcs[1])
	}
}

// TestUserScopeActive verifies the user-scope activation rule: either the
// definitions directory or the live services directory is enough.
func TestUserScopeActive(t *testing.T) {
	defs := t.TempDir()
	live := t.TempDir()
	cases := []struct {
		name string
		dir  string
		dest string
		want bool
	}{
		{"definitions only", defs, "/nonexistent", true},
		{"live dir only", "/nonexistent", live, true},
		{"both", defs, live, true},
		{"neither", "/nonexistent", "/nonexistent", false},
	}
	for _, c := range cases {
		if got := userScopeActive(c.dir, c.dest); got != c.want {
			t.Errorf("%s: userScopeActive(%q, %q) = %v, want %v", c.name, c.dir, c.dest, got, c.want)
		}
	}
}

func TestServiceKey(t *testing.T) {
	svc := Service{Name: "sshd", Scope: ScopeSystem}
	if svc.Key() != "system/sshd" {
		t.Errorf("expected 'system/sshd', got %q", svc.Key())
	}
	svc.Scope = ScopeUser
	if svc.Key() != "user/sshd" {
		t.Errorf("expected 'user/sshd', got %q", svc.Key())
	}
}

func TestLoadServicesScoped_TagsScope(t *testing.T) {
	dir := t.TempDir()
	dest := t.TempDir()
	for _, name := range []string{"alpha", "beta"} {
		if err := os.Mkdir(filepath.Join(dir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	sd := ScopeDir{Scope: ScopeUser, ServiceDir: dir, DestDir: dest}
	svcs := LoadServicesScoped(sd)
	if len(svcs) != 2 {
		t.Fatalf("expected 2, got %d", len(svcs))
	}
	for _, svc := range svcs {
		if svc.Scope != ScopeUser {
			t.Errorf("expected ScopeUser for %q, got %q", svc.Name, svc.Scope)
		}
	}
}

func TestLoadServicesScoped_MissingDir(t *testing.T) {
	sd := ScopeDir{Scope: ScopeUser, ServiceDir: "/nonexistent/path", DestDir: "/nonexistent/dest"}
	svcs := LoadServicesScoped(sd)
	if svcs != nil {
		t.Errorf("expected nil for missing dir, got %v", svcs)
	}
}

func TestList_TwoScopes_SystemFirst(t *testing.T) {
	sysDir := t.TempDir()
	sysDest := t.TempDir()
	usrDir := t.TempDir()
	usrDest := t.TempDir()
	for _, name := range []string{"alpha", "bravo"} {
		if err := os.Mkdir(filepath.Join(sysDir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"charlie", "delta"} {
		if err := os.Mkdir(filepath.Join(usrDir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	b := NewScopedRunitBackend(
		ScopeDir{Scope: ScopeSystem, ServiceDir: sysDir, DestDir: sysDest, Elevated: true},
		ScopeDir{Scope: ScopeUser, ServiceDir: usrDir, DestDir: usrDest},
	)
	svcs := b.List()
	if len(svcs) != 4 {
		t.Fatalf("expected 4 services, got %d", len(svcs))
	}
	expected := []struct {
		name  string
		scope Scope
	}{
		{"alpha", ScopeSystem},
		{"bravo", ScopeSystem},
		{"charlie", ScopeUser},
		{"delta", ScopeUser},
	}
	for i, exp := range expected {
		if svcs[i].Name != exp.name {
			t.Errorf("index %d: expected name %q, got %q", i, exp.name, svcs[i].Name)
		}
		if svcs[i].Scope != exp.scope {
			t.Errorf("index %d: expected scope %q, got %q", i, exp.scope, svcs[i].Scope)
		}
	}
}

func TestNewScopedRunitBackend_OmitsEmptyUser(t *testing.T) {
	sys := ScopeDir{Scope: ScopeSystem, ServiceDir: DefaultServiceDir, DestDir: DefaultServiceDestDir, Elevated: true}
	user := ScopeDir{Scope: ScopeUser}
	b := NewScopedRunitBackend(sys, user)
	if len(b.Scopes) != 1 {
		t.Errorf("expected 1 scope (user omitted), got %d", len(b.Scopes))
	}
	if b.Scopes[0].Scope != ScopeSystem {
		t.Errorf("expected system scope, got %q", b.Scopes[0].Scope)
	}
}

func TestStatusAll_EmptyServices(t *testing.T) {
	b := NewRunitBackend(t.TempDir(), t.TempDir())
	result := b.StatusAll(nil)
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d entries", len(result))
	}
}

func TestDirs_RoutesByScope(t *testing.T) {
	sys := ScopeDir{Scope: ScopeSystem, ServiceDir: "/etc/sv", DestDir: "/var/service", Elevated: true}
	usr := ScopeDir{Scope: ScopeUser, ServiceDir: "/home/u/.config/service", DestDir: "/home/u/service"}
	b := NewScopedRunitBackend(sys, usr)
	svcDir, destDir := b.Dirs(Service{Name: "sshd", Scope: ScopeUser})
	if svcDir != usr.ServiceDir || destDir != usr.DestDir {
		t.Errorf("expected user dirs, got %q/%q", svcDir, destDir)
	}
	svcDir, destDir = b.Dirs(Service{Name: "sshd", Scope: ScopeSystem})
	if svcDir != sys.ServiceDir || destDir != sys.DestDir {
		t.Errorf("expected system dirs, got %q/%q", svcDir, destDir)
	}
}

func TestScopeFilter_EmptyMatchesEverything(t *testing.T) {
	var f ScopeFilter
	if f.Active() {
		t.Error("empty ScopeFilter should not be active")
	}
	if !f.matches(ScopeSystem) || !f.matches(ScopeUser) {
		t.Error("empty ScopeFilter should match every scope")
	}
}

func TestScopeFilter_SystemOnly(t *testing.T) {
	f := ScopeFilter{System: true}
	if !f.Active() {
		t.Error("System filter should be active")
	}
	if !f.matches(ScopeSystem) {
		t.Error("System filter should match system scope")
	}
	if f.matches(ScopeUser) {
		t.Error("System filter should not match user scope")
	}
}

func TestScopeFilter_UserOnly(t *testing.T) {
	f := ScopeFilter{User: true}
	if !f.matches(ScopeUser) {
		t.Error("User filter should match user scope")
	}
	if f.matches(ScopeSystem) {
		t.Error("User filter should not match system scope")
	}
}

func TestScopeFilter_BothMatchEverything(t *testing.T) {
	f := ScopeFilter{System: true, User: true}
	if !f.matches(ScopeSystem) || !f.matches(ScopeUser) {
		t.Error("ScopeFilter with both scopes should match everything")
	}
}

func TestScopeFilter_Toggle(t *testing.T) {
	var f ScopeFilter
	f = f.Toggle(ScopeSystem)
	if !f.System || f.User {
		t.Errorf("after toggling system, expected System only, got %+v", f)
	}
	f = f.Toggle(ScopeSystem)
	if f.System {
		t.Error("second toggle should disable System")
	}
	f = f.Toggle(ScopeUser)
	if !f.User || f.System {
		t.Errorf("after toggling user, expected User only, got %+v", f)
	}
}

func TestFilterScoped_AllModesRespectScope(t *testing.T) {
	services := []Service{
		{Name: "sys-a", Scope: ScopeSystem, Enabled: true},
		{Name: "sys-b", Scope: ScopeSystem, Enabled: true},
		{Name: "sys-c", Scope: ScopeSystem, Enabled: false},
		{Name: "usr-a", Scope: ScopeUser, Enabled: true},
		{Name: "usr-b", Scope: ScopeUser, Enabled: true},
		{Name: "usr-c", Scope: ScopeUser, Enabled: false},
	}

	check := func(t *testing.T, sf ScopeFilter, mode FilterMode, want []string) {
		t.Helper()
		got := filterScoped(services, mode, "", sf)
		if len(got) != len(want) {
			t.Fatalf("mode %d scope %+v: got %d services, want %d: %+v", mode, sf, len(got), len(want), got)
		}
		for i, svc := range got {
			if svc.Name != want[i] {
				t.Errorf("mode %d scope %+v: row %d = %s, want %s", mode, sf, i, svc.Name, want[i])
			}
		}
	}

	sys := ScopeFilter{System: true}
	usr := ScopeFilter{User: true}
	check(t, sys, FilterAll, []string{"sys-a", "sys-b", "sys-c"})
	check(t, sys, FilterEnabled, []string{"sys-a", "sys-b"})
	check(t, sys, FilterDisabled, []string{"sys-c"})
	check(t, usr, FilterAll, []string{"usr-a", "usr-b", "usr-c"})
	check(t, usr, FilterEnabled, []string{"usr-a", "usr-b"})
	check(t, usr, FilterDisabled, []string{"usr-c"})
}
