package srcman

import (
	"os"
	"path/filepath"
	"testing"

	"charm.land/bubbles/v2/textarea"
)

func TestInlineEditorLoadsAndSavesSelectedTemplate(t *testing.T) {
	distDir := t.TempDir()
	templateDir := filepath.Join(distDir, "srcpkgs", "fixture")
	if err := os.MkdirAll(templateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(templateDir, "template")
	const original = "pkgname=fixture\nversion=1\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	model := xbpsModel{distDir: distDir, editor: textarea.New()}
	model.loadEditorFile("fixture")
	if model.editor.Value() != original {
		t.Fatalf("loaded template = %q, want %q", model.editor.Value(), original)
	}

	const updated = "pkgname=fixture\nversion=2\n"
	model.editor.SetValue(updated)
	model = model.saveEditor()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != updated {
		t.Fatalf("saved template = %q, want %q", got, updated)
	}
}
