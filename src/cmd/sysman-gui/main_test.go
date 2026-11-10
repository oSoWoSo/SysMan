package main

import (
	"testing"

	"charm.land/bubbletea/v2"
	"codeberg.org/oSoWoSo/SysMan/src/common"
)

func TestKeyHintsToggleOnF1AndCtrlH(t *testing.T) {
	common.SetTUIKeyHintsVisible(false)
	t.Cleanup(func() { common.SetTUIKeyHintsVisible(false) })

	model := sysManagerModel{}
	if !model.toggleKeyHints(tea.KeyPressMsg(tea.Key{Code: tea.KeyF1})) {
		t.Fatal("F1 should toggle key hints")
	}
	if !common.TUIKeyHintsVisible() {
		t.Fatal("first F1 press should show key hints")
	}
	model.toggleKeyHints(tea.KeyPressMsg(tea.Key{Code: tea.KeyF1}))
	if common.TUIKeyHintsVisible() {
		t.Fatal("second F1 press should hide key hints")
	}

	model.toggleKeyHints(tea.KeyPressMsg(tea.Key{Code: 'h', Mod: tea.ModCtrl}))
	if !common.TUIKeyHintsVisible() {
		t.Fatal("Ctrl+H should show key hints")
	}
	model.toggleKeyHints(tea.KeyPressMsg(tea.Key{Code: 'h', Mod: tea.ModCtrl}))
	if common.TUIKeyHintsVisible() {
		t.Fatal("second Ctrl+H press should hide key hints")
	}
}
