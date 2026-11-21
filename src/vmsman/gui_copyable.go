//go:build !tui_only

package vmman

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// ShowCopyableError displays err in a dialog whose message can be selected and
// copied, so users can share the full text of failures (e.g. Create VM errors).
func ShowCopyableError(parent fyne.Window, err error) {
	showCopyableError(parent, err)
}

// showCopyableError displays err in a dialog whose message can be selected and
// copied, so users can share the full text of failures (e.g. Create VM errors).
func showCopyableError(parent fyne.Window, err error) {
	if err == nil {
		return
	}
	msg := err.Error()
	entry := widget.NewMultiLineEntry()
	entry.SetText(msg)
	entry.Disable()
	entry.Wrapping = fyne.TextWrapWord
	scroll := container.NewVScroll(entry)
	scroll.SetMinSize(fyne.NewSize(480, 120))
	copyBtn := widget.NewButton("Copy", func() {
		fyne.CurrentApp().Clipboard().SetContent(msg)
	})
	content := container.NewBorder(nil, container.NewHBox(copyBtn), nil, nil, scroll)
	d := dialog.NewCustom("Error", "OK", content, parent)
	d.Show()
}
