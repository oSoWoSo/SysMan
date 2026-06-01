//go:build !tui_only

package common

import (
	"fmt"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// SettingsField defines a single field in the settings dialog.
type SettingsField struct {
	Label       string
	Value       string // current value from config
	Placeholder string // placeholder text (e.g. default value)
}

// ShowSettingsDialog shows a modal settings dialog for a single module.
// The saveFn is called with a map of label → trimmed value when the user clicks Save.
func ShowSettingsDialog(win fyne.Window, moduleName string, fields []SettingsField, saveFn func(values map[string]string)) {
	entries := make(map[string]*widget.Entry)
	var items []*widget.FormItem

	for _, f := range fields {
		e := widget.NewEntry()
		e.SetPlaceHolder(f.Placeholder)
		e.SetText(f.Value)
		entries[f.Label] = e
		items = append(items, widget.NewFormItem(f.Label, e))
	}

	form := widget.NewForm(items...)

	title := canvas.NewText(moduleName+" Settings", color.NRGBA{R: 0x00, G: 0xb8, B: 0xd4, A: 0xff})
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 18

 btnSave := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), func() {
		values := make(map[string]string)
		for _, f := range fields {
			values[f.Label] = strings.TrimSpace(entries[f.Label].Text)
		}
		if saveFn != nil {
			saveFn(values)
		}
	})
	btnSave.Importance = widget.HighImportance

	content := container.NewVBox(
		container.NewPadded(title),
		container.NewPadded(form),
		layout.NewSpacer(),
		container.NewPadded(btnSave),
	)

	d := dialog.NewCustom("Settings", "Close", content, win)
	d.Resize(fyne.NewSize(500, 400))
	d.Show()
}

// ShowSettingsError shows an error dialog for settings save failures.
func ShowSettingsError(win fyne.Window, err error) {
	dialog.ShowError(fmt.Errorf("save config: %w", err), win)
}
