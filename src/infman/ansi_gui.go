//go:build !tui_only

package infman

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

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
