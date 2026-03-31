package pkgman

import (
	"testing"
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
