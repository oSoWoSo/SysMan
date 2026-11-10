package common

import (
	"strings"
	"testing"
)

func TestTBtnShowsShortcutOnlyWhenEnabled(t *testing.T) {
	EnsureZone()
	SetTUIKeyHintsVisible(false)
	t.Cleanup(func() { SetTUIKeyHintsVisible(true) })

	button := TBtn("test", "save", "Save", "ctrl+s", false)
	if strings.Contains(button, "ctrl+s") {
		t.Fatal("shortcut should be hidden when key hints are disabled")
	}

	SetTUIKeyHintsVisible(true)
	button = TBtn("test", "save", "Save", "ctrl+s", false)
	if !strings.Contains(button, "Save [ctrl+s]") {
		t.Fatal("shortcut should appear inside the button when key hints are enabled")
	}
}
