package srcman

import (
	"testing"
)

func TestHumanBytes(t *testing.T) {
	tests := []struct {
		input uint64
		want  string
	}{
		{0, "0 B"},
		{1023, "1023 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{1048576, "1.0 MiB"},
		{1572864, "1.5 MiB"},
		{1073741824, "1.0 GiB"},
		{1610612736, "1.5 GiB"},
		{1099511627776, "1.0 TiB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := humanBytes(tt.input); got != tt.want {
				t.Errorf("humanBytes(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestResolveDistDir_Default(t *testing.T) {
	// When XBPS_DISTDIR not set, should use default ~/void
	got := ResolveDistDir("")
	if got == "" {
		t.Error("ResolveDistDir('') returned empty string")
	}
}

func TestFilter_Templates(t *testing.T) {
	templates := []Template{
		{Name: "vim"},
		{Name: "bash"},
		{Name: "neovim"},
	}

	// Test empty search returns all
	filtered := Filter(templates, "")
	if len(filtered) != 3 {
		t.Errorf("Filter(, '') = %d, want 3", len(filtered))
	}

	// Test search filter
	filtered = Filter(templates, "vim")
	if len(filtered) != 2 {
		t.Errorf("Filter(, 'vim') = %d, want 2", len(filtered))
	}
}
