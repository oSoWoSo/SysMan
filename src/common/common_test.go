package common

import (
	"image/color"
	"strings"
	"testing"
)

//item is a test struct for Filter tests.
type testItem struct {
	name    string
	status string // "running", "stopped", etc.
}

func TestFilter_ModeAll(t *testing.T) {
	items := []testItem{
		{name: "a", status: "running"},
		{name: "b", status: "stopped"},
		{name: "c", status: "running"},
	}
	isMatched := func(i testItem) bool { return i.status == "running" }
	matchesSearch := func(i testItem, q string) bool { return strings.Contains(i.name, q) }

	// mode 0 = show all
	got := Filter(items, 0, "", isMatched, matchesSearch)
	if len(got) != 3 {
		t.Errorf("Filter mode 0: got %d items, want 3", len(got))
	}
}

func TestFilter_ModeMatched(t *testing.T) {
	items := []testItem{
		{name: "a", status: "running"},
		{name: "b", status: "stopped"},
		{name: "c", status: "running"},
	}
	isMatched := func(i testItem) bool { return i.status == "running" }
	matchesSearch := func(i testItem, q string) bool { return strings.Contains(i.name, q) }

	// mode 1 = show only matched (running)
	got := Filter(items, 1, "", isMatched, matchesSearch)
	if len(got) != 2 {
		t.Errorf("Filter mode 1: got %d items, want 2", len(got))
	}
	if got[0].name != "a" || got[1].name != "c" {
		t.Errorf("Filter mode 1: unexpected items %v", got)
	}
}

func TestFilter_ModeUnmatched(t *testing.T) {
	items := []testItem{
		{name: "a", status: "running"},
		{name: "b", status: "stopped"},
		{name: "c", status: "running"},
	}
	isMatched := func(i testItem) bool { return i.status == "running" }
	matchesSearch := func(i testItem, q string) bool { return strings.Contains(i.name, q) }

	// mode 2 = show only unmatched (stopped)
	got := Filter(items, 2, "", isMatched, matchesSearch)
	if len(got) != 1 {
		t.Errorf("Filter mode 2: got %d items, want 1", len(got))
	}
	if got[0].name != "b" {
		t.Errorf("Filter mode 2: got %v, want b", got[0].name)
	}
}

func TestFilter_WithSearchQuery(t *testing.T) {
	items := []testItem{
		{name: "alpha", status: "running"},
		{name: "beta", status: "stopped"},
		{name: "gamma", status: "running"},
	}
	isMatched := func(i testItem) bool { return i.status == "running" }
	matchesSearch := func(i testItem, q string) bool {
		return strings.Contains(i.name, q) // match items containing the search query
	}

	// mode 0 with search query for "beta" (only beta matches)
	got := Filter(items, 0, "beta", isMatched, matchesSearch)
	if len(got) != 1 {
		t.Errorf("Filter with search: got %d items, want 1", len(got))
	}
	if len(got) > 0 && got[0].name != "beta" {
		t.Errorf("Filter with search: got %v, want beta", got[0].name)
	}
}

func TestHasAnsiCodes_PlainText(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"hello world", false},
		{"", false},
		{"plain text without codes", false},
		{"\x1b[0m", true},
		{"\x1b[32m green \x1b[0m", true},
		{"\x1b[38;5;196m", true},
		{"\x1b[38;2;255;128;64m", true},
	}

	for _, tc := range tests {
		got := HasAnsiCodes(tc.input)
		if got != tc.want {
			t.Errorf("HasAnsiCodes(%q): got %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestParseSeq_StandardColors(t *testing.T) {
	tests := []struct {
		seq      string
		wantOk   bool
		wantR   uint8
		wantG   uint8
		wantB   uint8
	}{
		// Standard colors (30-37)
		{"\x1b[30m", true, 0x1c, 0x1c, 0x1c},
		{"\x1b[31m", true, 0xcc, 0x33, 0x33},
		{"\x1b[32m", true, 0x22, 0xaa, 0x55},
		{"\x1b[33m", true, 0xbb, 0x88, 0x00},
		{"\x1b[34m", true, 0x33, 0x66, 0xcc},
		{"\x1b[35m", true, 0x99, 0x33, 0xcc},
		{"\x1b[36m", true, 0x00, 0x99, 0xaa},
		{"\x1b[37m", true, 0xcc, 0xcc, 0xcc},
		// Bright colors (90-97)
		{"\x1b[90m", true, 0x55, 0x55, 0x55},
		{"\x1b[91m", true, 0xff, 0x55, 0x55},
		{"\x1b[92m", true, 0x44, 0xdd, 0x77},
		{"\x1b[93m", true, 0xff, 0xcc, 0x00},
		{"\x1b[94m", true, 0x55, 0x88, 0xff},
		{"\x1b[95m", true, 0xcc, 0x55, 0xff},
		{"\x1b[96m", true, 0x00, 0xdd, 0xff},
		{"\x1b[97m", true, 0xff, 0xff, 0xff},
		// Reset
		{"\x1b[0m", false, 0, 0, 0},
		{"\x1b[m", false, 0, 0, 0},
	}

	for _, tc := range tests {
		got, ok := ParseSeq(tc.seq)
		if ok != tc.wantOk {
			t.Errorf("ParseSeq(%q): ok = %v, want %v", tc.seq, ok, tc.wantOk)
			continue
		}
		if ok {
			rgba := color.NRGBA{}
			if got != rgba {
				r, g, b, _ := got.RGBA()
				if uint8(r>>8) != tc.wantR || uint8(g>>8) != tc.wantG || uint8(b>>8) != tc.wantB {
					t.Errorf("ParseSeq(%q): got (%v,%v,%v), want (%02x,%02x,%02x)",
						tc.seq, uint8(r>>8), uint8(g>>8), uint8(b>>8), tc.wantR, tc.wantG, tc.wantB)
				}
			}
		}
	}
}

func TestParseSeq_256Color(t *testing.T) {
	// Test xterm 256-color palette (38;5;n)
	tests := []struct {
		seq      string
		wantIdx int
		wantOk  bool
	}{
		{"\x1b[38;5;0m", 0, true},
		{"\x1b[38;5;15m", 15, true},
		{"\x1b[38;5;16m", 16, true},
		{"\x1b[38;5;231m", 231, true},
		{"\x1b[38;5;255m", 255, true},
		{"\x1b[38;5;256m", 256, false}, // out of range
	}

	for _, tc := range tests {
		got, ok := ParseSeq(tc.seq)
		if ok != tc.wantOk {
			t.Errorf("ParseSeq(%q): ok = %v, want %v", tc.seq, ok, tc.wantOk)
			continue
		}
		if ok {
			want := Ansi256Palette[tc.wantIdx]
			if got != want {
				t.Errorf("ParseSeq(%q): got %v, want %v", tc.seq, got, want)
			}
		}
	}
}

func TestParseSeq_RGB(t *testing.T) {
	// Test 24-bit RGB mode (38;2;R;G;B)
	tests := []struct {
		seq    string
		wantR uint8
		wantG uint8
		wantB uint8
		wantOk bool
	}{
		{"\x1b[38;2;255;128;64m", 255, 128, 64, true},
		{"\x1b[38;2;0;0;0m", 0, 0, 0, true},
		{"\x1b[38;2;255;255;255m", 255, 255, 255, true},
		{"\x1b[38;2;128;64;32m", 128, 64, 32, true},
	}

	for _, tc := range tests {
		got, ok := ParseSeq(tc.seq)
		if ok != tc.wantOk {
			t.Errorf("ParseSeq(%q): ok = %v, want %v", tc.seq, ok, tc.wantOk)
			continue
		}
		if ok {
			r, g, b, _ := got.RGBA()
			if uint8(r>>8) != tc.wantR || uint8(g>>8) != tc.wantG || uint8(b>>8) != tc.wantB {
				t.Errorf("ParseSeq(%q): got (%v,%v,%v), want (%02x,%02x,%02x)",
					tc.seq, uint8(r>>8), uint8(g>>8), uint8(b>>8), tc.wantR, tc.wantG, tc.wantB)
			}
		}
	}
}