package common

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type HoverableButton struct {
	*widget.Button
	StatusText string
	statusBar  *widget.Label
}

func NewHoverableButton(label string, icon fyne.Resource, statusText string, statusBar *widget.Label, tapped func()) *HoverableButton {
	btn := widget.NewButtonWithIcon(label, icon, tapped)
	return &HoverableButton{
		Button:     btn,
		StatusText: statusText,
		statusBar:  statusBar,
	}
}

func (b *HoverableButton) MouseIn(e *desktop.MouseEvent) {
	if b.statusBar != nil && b.StatusText != "" {
		b.statusBar.SetText(b.StatusText)
	}
}

func (b *HoverableButton) MouseOut() {
	if b.statusBar != nil {
		b.statusBar.SetText("")
	}
}
