package common

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// HoverableButton is a button with hover status text.
type HoverableButton struct {
	widget.Button
	StatusText string
	statusBar  *StatusBar
}

// NewHoverableButton creates a new HoverableButton.
func NewHoverableButton(label string, icon fyne.Resource, statusText string, statusBar *StatusBar, tapped func()) *HoverableButton {
	b := &HoverableButton{
		StatusText: statusText,
		statusBar:  statusBar,
	}
	b.Text = label
	b.Icon = icon
	b.OnTapped = tapped
	b.ExtendBaseWidget(b)
	return b
}

// NewHoverableButtonText creates a text button without icon that shows status text on hover.
func NewHoverableButtonText(label string, statusText string, statusBar *StatusBar, tapped func()) *HoverableButton {
	b := &HoverableButton{
		StatusText: statusText,
		statusBar:  statusBar,
	}
	b.Text = label
	b.OnTapped = tapped
	b.ExtendBaseWidget(b)
	return b
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

// MouseMoved satisfies the desktop.Hoverable interface.
func (b *HoverableButton) MouseMoved(_ *desktop.MouseEvent) {}

// HoverableCheck is a check widget with hover status text.
type HoverableCheck struct {
	widget.Check
	StatusText string
	statusBar  *StatusBar
}

// NewHoverableCheck creates a new HoverableCheck.
func NewHoverableCheck(label string, statusText string, statusBar *StatusBar, changed func(bool)) *HoverableCheck {
	c := &HoverableCheck{
		StatusText: statusText,
		statusBar:  statusBar,
	}
	c.Text = label
	c.OnChanged = changed
	c.ExtendBaseWidget(c)
	return c
}

// MouseIn handles mouse in event.
func (c *HoverableCheck) MouseIn(_ *desktop.MouseEvent) {
	if c.statusBar != nil && c.StatusText != "" {
		c.statusBar.SetText(c.StatusText)
	}
}

// MouseOut handles mouse out event.
func (c *HoverableCheck) MouseOut() {
	if c.statusBar != nil {
		c.statusBar.SetText("")
	}
}

// MouseMoved satisfies the desktop.Hoverable interface.
func (c *HoverableCheck) MouseMoved(_ *desktop.MouseEvent) {}
