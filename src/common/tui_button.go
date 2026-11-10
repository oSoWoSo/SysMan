package common

import (
	"image/color"
	"strings"
	"sync/atomic"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
	zone "github.com/lrstanley/bubblezone/v2"
)

// ── TUI Button styles ────────────────────────────────────────────────

var (
	// TBtnActiveBg is the background color for active/selected buttons.
	TBtnActiveBg = compat.AdaptiveColor{Light: lipgloss.Color("#006688"), Dark: lipgloss.Color("#0055AA")}
	// TBtnActiveFg is the foreground color for active/selected buttons.
	TBtnActiveFg = lipgloss.Color("#FFFFFF")
	// TBtnInactiveBg is the background color for inactive buttons.
	TBtnInactiveBg = compat.AdaptiveColor{Light: lipgloss.Color("#E0E0E0"), Dark: lipgloss.Color("#333333")}
	// TBtnInactiveFg is the foreground color for inactive buttons.
	TBtnInactiveFg = compat.AdaptiveColor{Light: lipgloss.Color("#333333"), Dark: lipgloss.Color("#AAAAAA")}
	// TBtnSepStyle styles button separators.
	TBtnSepStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	// TSelectedStyle is the shared selected-row style, matching active TUI buttons.
	TSelectedStyle = lipgloss.NewStyle().Bold(true).Foreground(TBtnActiveFg).Background(TBtnActiveBg)
)

var (
	tuiKeyHintsVisible atomic.Bool
	tuiEmbedded        atomic.Bool
)

func init() {
	tuiKeyHintsVisible.Store(false)
}

// SetTUIKeyHintsVisible controls whether TUI buttons show their keyboard shortcut.
func SetTUIKeyHintsVisible(visible bool) {
	tuiKeyHintsVisible.Store(visible)
}

// TUIKeyHintsVisible reports whether TUI keyboard shortcuts are currently shown.
func TUIKeyHintsVisible() bool {
	return tuiKeyHintsVisible.Load()
}

// ToggleTUIKeyHints handles the global shortcut that toggles visible TUI key hints.
func ToggleTUIKeyHints(key tea.KeyPressMsg) bool {
	switch key.String() {
	case "f1", "ctrl+h":
		tuiKeyHintsVisible.Store(!tuiKeyHintsVisible.Load())
		return true
	default:
		return false
	}
}

// SetTUIEmbedded marks module views as being hosted inside the sysman TUI.
func SetTUIEmbedded(embedded bool) {
	tuiEmbedded.Store(embedded)
}

// TUIEmbedded reports whether module views are hosted inside the sysman TUI.
func TUIEmbedded() bool {
	return tuiEmbedded.Load()
}

// TBtn renders a single clickable button with zone marking.
// zonePrefix is the model's zone prefix (model.id).
// zoneKey is a unique key for this button within the model.
// label is the visible button text.
// keyHint is the keyboard shortcut hint (e.g. "s") shown while requested.
// active indicates whether this button is currently selected/active.
func TBtn(zonePrefix, zoneKey, label, keyHint string, active bool) string {
	var bg, fg color.Color
	if active {
		bg = TBtnActiveBg
		fg = TBtnActiveFg
	} else {
		bg = TBtnInactiveBg
		fg = TBtnInactiveFg
	}

	text := label
	if keyHint != "" && tuiKeyHintsVisible.Load() {
		text += " [" + keyHint + "]"
	}
	btn := lipgloss.NewStyle().
		Bold(true).
		Foreground(fg).
		Background(bg).
		Padding(0, 1).
		MarginRight(1).
		Render(text)

	return zone.Mark(zonePrefix+zoneKey, btn)
}

// TBtnBar renders a row of buttons separated by spaces.
func TBtnBar(btns ...string) string {
	return strings.Join(btns, " ")
}

// TUIFooter places controls on the left and status text on the right.
func TUIFooter(controls, status string, width int) string {
	gap := width - lipgloss.Width(controls) - lipgloss.Width(status)
	if gap < 2 {
		gap = 2
	}
	return controls + strings.Repeat(" ", gap) + status
}

// EnsureZone initializes the global bubblezone manager.
// Call this once in main() or RunTUI() before creating tea.Program.
func EnsureZone() {
	zone.NewGlobal()
}
