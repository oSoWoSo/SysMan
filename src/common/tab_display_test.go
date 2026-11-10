package common

import "testing"

func TestFormatTabLabel(t *testing.T) {
	tests := []struct {
		name      string
		showNames bool
		showIcons bool
		want      string
	}{
		{"Packages", true, false, "Packages"},
		{"Packages", false, true, "📦"},
		{"Packages", true, true, "📦 Packages"},
		{"Packages", false, false, ""},
	}
	for _, tt := range tests {
		if got := FormatTabLabel(tt.name, tt.showNames, tt.showIcons); got != tt.want {
			t.Errorf("FormatTabLabel(%q, %t, %t) = %q, want %q", tt.name, tt.showNames, tt.showIcons, got, tt.want)
		}
	}
}
