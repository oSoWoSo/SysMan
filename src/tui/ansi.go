// Package tui provides TUI helpers for SysMan plugins.
package tui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"

	"codeberg.org/oSoWoSo/SysMan/src/sysinfo"
)

// ansiColoredSegment is a RichTextSegment that renders text in a specific color.
type ansiColoredSegment struct {
	text string
	col  color.Color
}

func (s *ansiColoredSegment) Inline() bool { return false }

func (s *ansiColoredSegment) Textual() string { return s.text }

func (s *ansiColoredSegment) Select(_, _ fyne.Position) {}

func (s *ansiColoredSegment) SelectedText() string { return "" }

func (s *ansiColoredSegment) Unselect() {}

func (s *ansiColoredSegment) Visual() fyne.CanvasObject {
	t := canvas.NewText(s.text, s.col)
	t.TextStyle = fyne.TextStyle{Monospace: true}
	return t
}

func (s *ansiColoredSegment) Update(o fyne.CanvasObject) {
	t := o.(*canvas.Text)
	t.Text = s.text
	t.Color = s.col
	t.Refresh()
}

// HasAnsiCodes checks if text contains ANSI SGR escape sequences.
func HasAnsiCodes(text string) bool {
	return sysinfo.AnsiRe.MatchString(text)
}

// AnsiToRichSegments converts text with ANSI SGR codes into Fyne RichText segments.
// It parses ANSI escape sequences and creates colored segments while preserving
// the original text layout (including newlines).
func AnsiToRichSegments(text string) []widget.RichTextSegment {
	if text == "" {
		return nil
	}

	var segs []widget.RichTextSegment
	lines := strings.Split(text, "\n")

	for lineIdx, line := range lines {
		lineSegs := parseAnsiLine(line)
		segs = append(segs, lineSegs...)

		// Add newline separator between lines (except after the last line)
		if lineIdx < len(lines)-1 {
			segs = append(segs, &widget.TextSegment{
				Text:  "\n",
				Style: widget.RichTextStyle{Inline: true},
			})
		}
	}

	return segs
}

// parseAnsiLine converts a single line with ANSI codes to RichText segments.
func parseAnsiLine(line string) []widget.RichTextSegment {
	if !sysinfo.AnsiRe.MatchString(line) {
		// No ANSI codes - return plain text segment
		if line == "" {
			return []widget.RichTextSegment{
				&widget.TextSegment{
					Text:  " ",
					Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Monospace: true}},
				},
			}
		}
		return []widget.RichTextSegment{
			&widget.TextSegment{
				Text:  line,
				Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Monospace: true}},
			},
		}
	}

	// Split by ANSI codes
	parts := sysinfo.AnsiRe.Split(line, -1)
	codes := sysinfo.AnsiRe.FindAllString(line, -1)

	var segs []widget.RichTextSegment
	curColor := color.Color(sysinfo.DefaultFG)

	for i, part := range parts {
		// Process ANSI code BEFORE this text part (if any)
		if i < len(codes) {
			code := codes[i]
			if c, ok := sysinfo.ParseSeq(code); ok {
				curColor = c
			} else {
				// Reset or unknown code - revert to default
				curColor = sysinfo.DefaultFG
			}
		}

		// Add text segment with current color
		if part != "" {
			segs = append(segs, &ansiColoredSegment{text: part, col: curColor})
		}
	}

	if len(segs) == 0 {
		return []widget.RichTextSegment{
			&widget.TextSegment{
				Text:  " ",
				Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Monospace: true}},
			},
		}
	}

	return segs
}
