package plugin

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
