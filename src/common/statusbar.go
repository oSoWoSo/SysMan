package common

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// StatusBar is a simple status bar widget that displays text messages.
// It can be shared across modules in sysman or used standalone.
type StatusBar struct {
	widget.Label
}

// NewStatusBar creates a new status bar widget.
func NewStatusBar() *StatusBar {
	bar := &StatusBar{}
	bar.ExtendBaseWidget(bar)
	bar.Alignment = fyne.TextAlignLeading
	return bar
}

// SetText sets the status bar text.
func (s *StatusBar) SetText(text string) {
	s.Label.SetText(text)
}

// Clear clears the status bar text.
func (s *StatusBar) Clear() {
	s.Label.SetText("")
}
