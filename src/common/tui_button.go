package common

import (
	"fmt"
	"image/color"
	"strings"

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
	// TBtnHintStyle styles the keyboard hint text after buttons.
	TBtnHintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).MarginLeft(1)
	// TBtnSepStyle styles button separators.
	TBtnSepStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
)

// TBtn renders a single clickable button with zone marking.
// zonePrefix is the model's zone prefix (model.id).
// zoneKey is a unique key for this button within the model.
// label is the visible button text.
// keyHint is the keyboard shortcut hint (e.g. "s") shown after the button.
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

	btn := lipgloss.NewStyle().
		Bold(true).
		Foreground(fg).
		Background(bg).
		Padding(0, 1).
		MarginRight(1).
		Render(label)

	hint := TBtnHintStyle.Render(fmt.Sprintf("[%s]", keyHint))

	return zone.Mark(zonePrefix+zoneKey, btn) + hint
}

// TBtnBar renders a row of buttons separated by spaces.
func TBtnBar(btns ...string) string {
	return strings.Join(btns, " ")
}

// EnsureZone initializes the global bubblezone manager.
// Call this once in main() or RunTUI() before creating tea.Program.
func EnsureZone() {
	zone.NewGlobal()
}
