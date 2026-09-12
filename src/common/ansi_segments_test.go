//go:build !tui_only

package common

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// renderedRows renders the given segment list and returns how many visual
// rows it produces. Fyne renders phantom blank rows for mis-handled newline
// segments, so the row count is a good proxy for correct line spacing.
func renderedRows(t *testing.T, segs []widget.RichTextSegment) int {
	t.Helper()
	rt := widget.NewRichText(segs...)
	rt.Wrapping = fyne.TextWrapOff
	r := rt.CreateRenderer()
	r.Layout(fyne.NewSize(800, 600))
	return len(r.Objects())
}

func TestAnsiToRichSegments_LineSpacing(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	cases := []struct {
		name string
		text string
		want int
	}{
		{"singleLine", "just one line", 1},
		{"threeLines", "foo\nbar\nbaz", 3},
		{"trailingNewline", "foo\nbar\nbaz\n", 3},
		{"emptyMiddle", "a\n\nb", 3},
		{"twoEmptyMiddle", "a\n\n\nb", 4},
		{"trailingDouble", "a\nb\n", 2},
		{"ansiMultiline", "\x1b[31mfoo\n\x1b[32mbar", 2},
		{"ansiMidEmpty", "\x1b[31mfoo\n\n\x1b[32mbar", 3},
	}
	for _, c := range cases {
		got := renderedRows(t, AnsiToRichSegments(c.text))
		if got != c.want {
			t.Errorf("%s: %q rendered %d rows, want %d", c.name, c.text, got, c.want)
		}
	}
}
