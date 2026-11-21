package vmman

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// SetPickerOptions applies options to a picker selector and shows/hides its row.
func SetPickerOptions(selector *widget.Select, row fyne.CanvasObject, options []string) {
	selector.ClearSelected()
	selector.SetOptions(options)
	if len(options) > 0 {
		selector.SetSelected(options[0])
	}
	if len(options) > 1 {
		row.Show()
	} else {
		row.Hide()
	}
}

// PickerNeedsInput reports whether the picker requires user input.
func PickerNeedsInput(architectures, releases, editions []string) bool {
	return len(architectures) > 1 || len(releases) > 1 || len(editions) > 1
}
