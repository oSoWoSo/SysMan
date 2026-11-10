package pkgman

import "testing"

func TestLastAttr(t *testing.T) {
	if got := lastAttr("nixpkgs.hello"); got != "hello" {
		t.Fatalf("lastAttr() = %q, want hello", got)
	}
}

func TestParseGuixLines(t *testing.T) {
	got := parseGuixLines("hello@2.12.1 x86_64-linux GNU Hello\n\n- header\n")
	if len(got) != 1 {
		t.Fatalf("got %d packages, want 1", len(got))
	}
	if got[0].name != "hello" || got[0].version != "2.12.1" || got[0].desc != "GNU Hello" {
		t.Fatalf("parsed package = %#v", got[0])
	}
}

func TestParseNixEnvPackages(t *testing.T) {
	installed := map[string]bool{"hello": true}
	got := parseNixEnvPackages("nixpkgs.hello hello-2.12.1\nnixpkgs.git git-2.45.1\n", installed, NewNixBackend())
	if len(got) != 2 {
		t.Fatalf("got %d packages, want 2", len(got))
	}
	if got[0].Name != "hello" || !got[0].Installed {
		t.Fatalf("first package = %#v", got[0])
	}
	if got[1].Name != "git" || got[1].Installed {
		t.Fatalf("second package = %#v", got[1])
	}
}
