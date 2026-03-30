// Package sysinfo — ANSI rendering helpers.
// Exported so external callers (e.g. a TUI shell or future plugins) can reuse
// the full-colour Fyne view and the fastfetch/neofetch runner.
package sysinfo

import (
	"image/color"
	"os/exec"
	"regexp"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

// RunFetch tries fastfetch then neofetch with the given args.
// Returns (output, true) on success, ("", false) when neither tool is found.
func RunFetch(args ...string) (string, bool) {
	for _, bin := range []string{"fastfetch", "neofetch"} {
		path, err := exec.LookPath(bin)
		if err != nil {
			continue
		}
		out, err := exec.Command(path, args...).Output() //nolint:gosec
		if err != nil {
			continue
		}
		return strings.TrimRight(string(out), "\n"), true
	}
	return "", false
}

// AnsiRe matches ANSI SGR escape sequences (e.g. \x1b[32m, \x1b[38;5;196m, \x1b[0m).
var AnsiRe = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// DefaultFG is the fallback foreground color for uncolored text.
var DefaultFG = color.NRGBA{R: 0xd8, G: 0xdc, B: 0xe0, A: 0xff}

// Ansi16 maps standard (30-37) and bright (90-97) SGR foreground codes to colors.
var Ansi16 = map[int]color.NRGBA{
	30: {0x1c, 0x1c, 0x1c, 0xff}, // black
	31: {0xcc, 0x33, 0x33, 0xff}, // red
	32: {0x22, 0xaa, 0x55, 0xff}, // green
	33: {0xbb, 0x88, 0x00, 0xff}, // yellow
	34: {0x33, 0x66, 0xcc, 0xff}, // blue
	35: {0x99, 0x33, 0xcc, 0xff}, // magenta
	36: {0x00, 0x99, 0xaa, 0xff}, // cyan
	37: {0xcc, 0xcc, 0xcc, 0xff}, // white
	90: {0x55, 0x55, 0x55, 0xff}, // bright black
	91: {0xff, 0x55, 0x55, 0xff}, // bright red
	92: {0x44, 0xdd, 0x77, 0xff}, // bright green
	93: {0xff, 0xcc, 0x00, 0xff}, // bright yellow
	94: {0x55, 0x88, 0xff, 0xff}, // bright blue
	95: {0xcc, 0x55, 0xff, 0xff}, // bright magenta
	96: {0x00, 0xdd, 0xff, 0xff}, // bright cyan
	97: {0xff, 0xff, 0xff, 0xff}, // bright white
}

// Ansi256Palette is the xterm 256-color palette.
var Ansi256Palette [256]color.NRGBA

func init() {
	// 0-15: standard and bright colors (same as Ansi16 table).
	std := []color.NRGBA{
		{0x1c, 0x1c, 0x1c, 0xff}, {0xcc, 0x33, 0x33, 0xff},
		{0x22, 0xaa, 0x55, 0xff}, {0xbb, 0x88, 0x00, 0xff},
		{0x33, 0x66, 0xcc, 0xff}, {0x99, 0x33, 0xcc, 0xff},
		{0x00, 0x99, 0xaa, 0xff}, {0xcc, 0xcc, 0xcc, 0xff},
		{0x55, 0x55, 0x55, 0xff}, {0xff, 0x55, 0x55, 0xff},
		{0x44, 0xdd, 0x77, 0xff}, {0xff, 0xcc, 0x00, 0xff},
		{0x55, 0x88, 0xff, 0xff}, {0xcc, 0x55, 0xff, 0xff},
		{0x00, 0xdd, 0xff, 0xff}, {0xff, 0xff, 0xff, 0xff},
	}
	copy(Ansi256Palette[:16], std)

	// 16-231: 6x6x6 color cube.
	levels := []uint8{0, 95, 135, 175, 215, 255}
	for i := 16; i < 232; i++ {
		n := i - 16
		b := levels[n%6]
		g := levels[(n/6)%6]
		r := levels[(n/36)%6]
		Ansi256Palette[i] = color.NRGBA{R: r, G: g, B: b, A: 0xff}
	}

	// 232-255: grayscale ramp.
	for i := 232; i < 256; i++ {
		v := uint8(8 + (i-232)*10)
		Ansi256Palette[i] = color.NRGBA{R: v, G: v, B: v, A: 0xff}
	}
}

// ParseSeq parses an ANSI SGR sequence (e.g. "\x1b[38;5;196m") and returns
// the resulting foreground color and whether it was recognized.
// Returns (zero, false) for reset/unknown sequences.
func ParseSeq(seq string) (color.NRGBA, bool) {
	inner := seq[2 : len(seq)-1] // strip \x1b[ and m
	if inner == "" || inner == "0" {
		return color.NRGBA{}, false // reset
	}
	parts := strings.Split(inner, ";")
	nums := make([]int, len(parts))
	for i, p := range parts {
		v := 0
		for _, ch := range p {
			if ch >= '0' && ch <= '9' {
				v = v*10 + int(ch-'0')
			}
		}
		nums[i] = v
	}
	// 256-color: ESC[38;5;Nm
	if len(nums) >= 3 && nums[0] == 38 && nums[1] == 5 {
		idx := nums[2]
		if idx >= 0 && idx < 256 {
			return Ansi256Palette[idx], true
		}
	}
	// 24-bit: ESC[38;2;R;G;Bm
	if len(nums) >= 5 && nums[0] == 38 && nums[1] == 2 {
		return color.NRGBA{R: uint8(nums[2]), G: uint8(nums[3]), B: uint8(nums[4]), A: 0xff}, true
	}
	// Standard 16 colors: use last number.
	last := nums[len(nums)-1]
	if c, ok := Ansi16[last]; ok {
		return c, true
	}
	return color.NRGBA{}, false
}

// BuildColoredView converts text with ANSI SGR codes into a scrollable Fyne
// widget using canvas.Text objects so arbitrary colors are preserved.
func BuildColoredView(text string) fyne.CanvasObject {
	const fontSize float32 = 13

	var rows []fyne.CanvasObject
	for _, line := range strings.Split(text, "\n") {
		parts := AnsiRe.Split(line, -1)
		codes := AnsiRe.FindAllString(line, -1)

		var cells []fyne.CanvasObject
		cur := color.Color(DefaultFG)

		for i, part := range parts {
			if part != "" {
				t := canvas.NewText(part, cur)
				t.TextStyle = fyne.TextStyle{Monospace: true}
				t.TextSize = fontSize
				cells = append(cells, t)
			}
			if i < len(codes) {
				if c, ok := ParseSeq(codes[i]); ok {
					cur = c
				} else {
					cur = DefaultFG
				}
			}
		}
		if len(cells) == 0 {
			// empty line — add a space to preserve height
			t := canvas.NewText(" ", DefaultFG)
			t.TextStyle = fyne.TextStyle{Monospace: true}
			t.TextSize = fontSize
			cells = append(cells, t)
		}
		rows = append(rows, container.NewHBox(cells...))
	}
	return container.NewScroll(container.NewVBox(rows...))
}
