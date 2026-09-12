//go:build !tui_only

// Package common provides shared helpers for SysMan plugins.
package common

import (
	"image/color"
	"regexp"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

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

	levels := []uint8{0, 95, 135, 175, 215, 255}
	for i := 16; i < 232; i++ {
		n := i - 16
		b := levels[n%6]
		g := levels[(n/6)%6]
		r := levels[(n/36)%6]
		Ansi256Palette[i] = color.NRGBA{R: r, G: g, B: b, A: 0xff}
	}

	for i := 232; i < 256; i++ {
		v := uint8(8 + (i-232)*10)
		Ansi256Palette[i] = color.NRGBA{R: v, G: v, B: v, A: 0xff}
	}
}

// ParseSeq parses an ANSI SGR sequence (e.g. "\x1b[38;5;196m") and returns
// the resulting foreground color and whether it was recognized.
func ParseSeq(seq string) (color.NRGBA, bool) {
	inner := seq[2 : len(seq)-1]
	if inner == "" || inner == "0" {
		return color.NRGBA{}, false
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
	if len(nums) >= 3 && nums[0] == 38 && nums[1] == 5 {
		idx := nums[2]
		if idx >= 0 && idx < 256 {
			return Ansi256Palette[idx], true
		}
	}
	if len(nums) >= 5 && nums[0] == 38 && nums[1] == 2 {
		return color.NRGBA{R: uint8(nums[2]), G: uint8(nums[3]), B: uint8(nums[4]), A: 0xff}, true
	}
	last := nums[len(nums)-1]
	if c, ok := Ansi16[last]; ok {
		return c, true
	}
	return color.NRGBA{}, false
}

// ansiRun is a single coloured text fragment within one log line.
type ansiRun struct {
	text string
	col  color.Color
}

// ansiBlockSegment is a RichTextSegment that renders one log line as its own
// paragraph row. It is deliberately NOT inline: inline segments that carry
// the line break get merged by Fyne's richtext renderer into phantom blank
// rows, so every line is rendered as a separate non-inline block instead.
type ansiBlockSegment struct {
	runs []ansiRun
}

func (s *ansiBlockSegment) Inline() bool { return false }

func (s *ansiBlockSegment) Textual() string {
	var b strings.Builder
	for _, r := range s.runs {
		b.WriteString(r.text)
	}
	return b.String()
}

func (s *ansiBlockSegment) Select(_, _ fyne.Position) {}

func (s *ansiBlockSegment) SelectedText() string { return "" }

func (s *ansiBlockSegment) Unselect() {}

func (s *ansiBlockSegment) Visual() fyne.CanvasObject {
	if len(s.runs) == 1 {
		t := canvas.NewText(s.runs[0].text, s.runs[0].col)
		t.TextStyle = fyne.TextStyle{Monospace: true}
		return t
	}
	objs := make([]fyne.CanvasObject, 0, len(s.runs))
	for _, r := range s.runs {
		t := canvas.NewText(r.text, r.col)
		t.TextStyle = fyne.TextStyle{Monospace: true}
		objs = append(objs, t)
	}
	return container.NewHBox(objs...)
}

func (s *ansiBlockSegment) Update(_ fyne.CanvasObject) {}

// HasAnsiCodes checks if text contains ANSI SGR escape sequences.
func HasAnsiCodes(text string) bool {
	return AnsiRe.MatchString(text)
}

// AnsiToRichSegments converts text with ANSI SGR codes into Fyne RichText
// segments. Each line becomes one non-inline block segment, so lines are
// rendered one per row without the phantom blank rows Fyne inserts when
// newlines are carried by inline segments.
func AnsiToRichSegments(text string) []widget.RichTextSegment {
	if text == "" {
		return nil
	}

	var segs []widget.RichTextSegment
	lines := strings.Split(text, "\n")

	for lineIdx, line := range lines {
		seg := parseAnsiLine(line)
		if len(seg.runs) == 0 {
			// A genuinely empty line renders as one blank row. A trailing
			// newline (empty last line) is skipped to avoid a phantom row.
			if lineIdx < len(lines)-1 {
				segs = append(segs, &ansiBlockSegment{
					runs: []ansiRun{{text: "", col: theme.Color(theme.ColorNameForeground)}},
				})
			}
			continue
		}
		segs = append(segs, seg)
	}

	return segs
}

// parseAnsiLine converts a single line into one block segment, keeping the
// colour per ANSI run. Lines without ANSI codes keep the theme foreground.
func parseAnsiLine(line string) *ansiBlockSegment {
	if !AnsiRe.MatchString(line) {
		if line == "" {
			return &ansiBlockSegment{}
		}
		return &ansiBlockSegment{
			runs: []ansiRun{{text: line, col: theme.Color(theme.ColorNameForeground)}},
		}
	}

	parts := AnsiRe.Split(line, -1)
	codes := AnsiRe.FindAllString(line, -1)

	var runs []ansiRun
	curColor := color.Color(DefaultFG)

	for i, part := range parts {
		if i < len(codes) {
			code := codes[i]
			if c, ok := ParseSeq(code); ok {
				curColor = c
			} else {
				curColor = DefaultFG
			}
		}

		if part != "" {
			runs = append(runs, ansiRun{text: part, col: curColor})
		}
	}

	return &ansiBlockSegment{runs: runs}
}
