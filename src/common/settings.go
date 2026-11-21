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
	"fyne.io/fyne/v2/widget"
)

// SettingsField defines a single field in the settings dialog.
type SettingsField struct {
	Label       string
	Value       string   // current value from config
	Placeholder string   // placeholder text (e.g. default value)
	Type        string   // "entry" (default), "select", or "bool"
	Options     []string // options for "select" fields
}

// ShowSettingsDialog shows a modal settings dialog for a single module.
// The saveFn is called with a map of label → trimmed value when the user clicks Save.
func ShowSettingsDialog(win fyne.Window, moduleName string, fields []SettingsField, saveFn func(values map[string]string)) {
	getters := make(map[string]func() string)
	var items []*widget.FormItem

	for _, f := range fields {
		switch f.Type {
		case "bool":
			check := widget.NewCheck("", nil)
			check.SetChecked(f.Value == "yes" || f.Value == "true" || f.Value == "1")
			getters[f.Label] = func() string {
				if check.Checked {
					return "yes"
				}
				return "no"
			}
			items = append(items, widget.NewFormItem(f.Label, check))
		case "select":
			sel := widget.NewSelect(f.Options, nil)
			sel.SetSelected(f.Value)
			getters[f.Label] = func() string { return sel.Selected }
			items = append(items, widget.NewFormItem(f.Label, sel))
		default:
			e := widget.NewEntry()
			e.SetPlaceHolder(f.Placeholder)
			e.SetText(f.Value)
			getters[f.Label] = func() string { return strings.TrimSpace(e.Text) }
			items = append(items, widget.NewFormItem(f.Label, e))
		}
	}

	form := widget.NewForm(items...)

	titleText := moduleName
	if !strings.HasSuffix(strings.ToLower(strings.TrimSpace(moduleName)), "settings") {
		titleText = moduleName + " Settings"
	}
	title := canvas.NewText(titleText, color.NRGBA{R: 0x00, G: 0xb8, B: 0xd4, A: 0xff})
	title.TextStyle = fyne.TextStyle{Bold: true}
	title.TextSize = 18

	content := container.NewVBox(
		container.NewPadded(title),
		container.NewPadded(form),
	)

	d := dialog.NewCustomConfirm("Settings", "Save", "Close", container.NewPadded(container.NewVScroll(content)), func(ok bool) {
		if !ok {
			return
		}
		values := make(map[string]string)
		for _, f := range fields {
			values[f.Label] = getters[f.Label]()
		}
		if saveFn != nil {
			saveFn(values)
		}
	}, win)
	d.Resize(fyne.NewSize(520, 620))
	d.Show()
}

// ShowSettingsError shows an error dialog for settings save failures.
func ShowSettingsError(win fyne.Window, err error) {
	dialog.ShowError(fmt.Errorf("save config: %w", err), win)
}
