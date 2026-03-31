package common

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// HoverableButton is a button with hover status text.
type HoverableButton struct {
	*widget.Button
	StatusText string
	statusBar  *StatusBar
}

// NewHoverableButton creates a new HoverableButton.
func NewHoverableButton(label string, icon fyne.Resource, statusText string, statusBar *StatusBar, tapped func()) *HoverableButton {
	btn := widget.NewButtonWithIcon(label, icon, tapped)
	return &HoverableButton{
		Button:     btn,
		StatusText: statusText,
		statusBar:  statusBar,
	}
}

// NewHoverableButtonText creates a text button without icon that shows status text on hover.
func NewHoverableButtonText(label string, statusText string, statusBar *StatusBar, tapped func()) *HoverableButton {
	btn := widget.NewButton(label, tapped)
	return &HoverableButton{
		Button:     btn,
		StatusText: statusText,
		statusBar:  statusBar,
	}
}

// MouseIn handles mouse in event.
func (b *HoverableButton) MouseIn(_ *desktop.MouseEvent) {
	if b.statusBar != nil && b.StatusText != "" {
		b.statusBar.SetText(b.StatusText)
	}
}

// MouseOut handles mouse out event.
func (b *HoverableButton) MouseOut() {
	if b.statusBar != nil {
		b.statusBar.SetText("")
	}
}
