//go:build !tui_only

package srcman

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"codeberg.org/oSoWoSo/SysMan/src/common"
)

// outputPanel is a selectable, scrollable output area with an inline find bar.
//
// It uses a disabled multiline Entry for native OS-level text selection and
// right-click support. When ANSI escape codes are detected, it switches to
// RichText for proper color rendering. Right-clicking while text is selected
// fires onSecondaryTap so the caller can show a context menu.
type outputPanel struct {
	canvas    fyne.Canvas
	outer     fyne.CanvasObject // border: entryContainer + find bar
	statusBar *common.StatusBar

	entry *selEntry
	plain strings.Builder

	// RichText for ANSI content
	rich       *widget.RichText
	richScroll *container.Scroll

	// Container for switching between Entry and RichText
	content *fyne.Container

	// find bar
	findEntry   *widget.Entry
	findLabel   *widget.Label
	findBar     *fyne.Container
	findMatches []int // byte offsets of matches
	findIdx     int
	findQuery   string
}

func newOutputPanel(canvas fyne.Canvas, statusBar *common.StatusBar, onSecondaryTap func(sel string, pos fyne.Position)) *outputPanel {
	p := &outputPanel{canvas: canvas, statusBar: statusBar, findIdx: -1}

	// widget.Entry has its own internal scroll — no container.Scroll needed.
	p.entry = newSelEntry(onSecondaryTap)

	// RichText for ANSI content with scroll container
	p.rich = widget.NewRichText()
	p.rich.Wrapping = fyne.TextWrapOff
	p.richScroll = container.NewScroll(p.rich)
	p.richScroll.SetMinSize(fyne.NewSize(0, 100))

	// Content stack - we'll show/hide based on ANSI detection
	p.content = container.NewStack(p.entry, p.richScroll)
	// Start with Entry visible
	p.richScroll.Hide()

	// ── Find bar (hidden by default) ──────────────────────────────────
	p.findEntry = widget.NewEntry()
	p.findEntry.SetPlaceHolder(t("find.placeholder"))
	p.findEntry.OnChanged = func(q string) { p.search(q) }

	p.findLabel = widget.NewLabel("")
	p.findLabel.TextStyle = fyne.TextStyle{Monospace: true}

	btnPrev := common.NewHoverableButton("", theme.NavigateBackIcon(), t("tooltip.find_prev"), p.statusBar, func() { p.stepMatch(-1) })
	btnPrev.Importance = widget.LowImportance
	btnNext := common.NewHoverableButton("", theme.NavigateNextIcon(), t("tooltip.find_next"), p.statusBar, func() { p.stepMatch(+1) })
	btnNext.Importance = widget.LowImportance
	btnClose := common.NewHoverableButton("", theme.CancelIcon(), t("tooltip.find_close"), p.statusBar, func() { p.HideFind() })
	btnClose.Importance = widget.LowImportance

	findRight := container.NewHBox(btnPrev.Button, btnNext.Button, p.findLabel, btnClose.Button)
	findEntryWrap := container.New(layout.NewGridWrapLayout(fyne.NewSize(240, 36)), p.findEntry)
	p.findBar = container.NewHBox(findEntryWrap, findRight)
	p.findBar.Hide()

	p.outer = container.NewBorder(nil, p.findBar, nil, nil, p.content)
	return p
}

// SetMinSize sets the minimum size of the output entry.
func (p *outputPanel) SetMinSize(size fyne.Size) {
	p.entry.SetMinSize(size)
}

// CanvasObject returns the outer widget to embed in layouts.
func (p *outputPanel) CanvasObject() fyne.CanvasObject {
	return p.outer
}

// ShowFind shows the find bar and focuses the search entry.
func (p *outputPanel) ShowFind() {
	p.findBar.Show()
	p.findEntry.FocusGained()
}

// HideFind hides the find bar.
func (p *outputPanel) HideFind() {
	p.findBar.Hide()
	p.findQuery = ""
	p.findMatches = nil
	p.findIdx = -1
	p.findLabel.SetText("")
}

// scrollToBottom moves the cursor to the last line, triggering the widget's
// internal scroll to follow.
func (p *outputPanel) scrollToBottom() {
	lines := strings.Count(p.plain.String(), "\n")
	if lines < 0 {
		lines = 0
	}
	// Scroll the active widget
	if p.richScroll.Visible() {
		p.richScroll.ScrollToBottom()
	} else {
		p.entry.CursorRow = lines
		p.entry.Refresh()
	}
}

// renderContent updates the output based on whether ANSI codes are present.
func (p *outputPanel) renderContent() {
	content := p.plain.String()
	if common.HasAnsiCodes(content) {
		// Switch to RichText for ANSI content
		p.entry.Hide()
		p.richScroll.Show()
		p.rich.Segments = common.AnsiToRichSegments(content)
		p.rich.Refresh()
	} else {
		// Use plain Entry for non-ANSI content
		p.richScroll.Hide()
		p.entry.Show()
		p.entry.SetText(content)
	}
}

// Append appends text to the output. It is thread-safe and can be called
// from goroutines.
func (p *outputPanel) Append(text string) {
	p.plain.WriteString(text)
	if p.canvas != nil {
		fyne.Do(func() {
			p.renderContent()
			p.scrollToBottom()
		})
	}
}

// SetText replaces the entire output content.
func (p *outputPanel) SetText(text string) {
	p.plain.Reset()
	p.plain.WriteString(text)
	p.renderContent()
	p.scrollToBottom()
}

// search finds all occurrences of q (case-insensitive).
func (p *outputPanel) search(q string) {
	p.findQuery = q
	p.findMatches = nil
	p.findIdx = -1

	if q != "" {
		lower := strings.ToLower(p.plain.String())
		lq := strings.ToLower(q)
		for i := 0; i <= len(lower)-len(lq); {
			idx := strings.Index(lower[i:], lq)
			if idx < 0 {
				break
			}
			p.findMatches = append(p.findMatches, i+idx)
			i += idx + len(lq)
		}
		if len(p.findMatches) > 0 {
			p.findIdx = 0
		}
	}
	p.updateFindLabel()
}

// stepMatch moves to the next (+1) or previous (-1) match.
func (p *outputPanel) stepMatch(dir int) {
	if len(p.findMatches) == 0 {
		return
	}
	p.findIdx = (p.findIdx + dir + len(p.findMatches)) % len(p.findMatches)
	p.updateFindLabel()
}

func (p *outputPanel) updateFindLabel() {
	if len(p.findMatches) == 0 {
		if p.findQuery != "" {
			p.findLabel.SetText(t("find.no_matches"))
		} else {
			p.findLabel.SetText("")
		}
		return
	}
	p.findLabel.SetText(fmt.Sprintf("%d / %d", p.findIdx+1, len(p.findMatches)))
}

// ── selEntry — selectable disabled entry ──────────────────────────────

// selEntry is a disabled multiline Entry that supports native text selection
// and fires onSecondaryTap on right-click when text is selected.
type selEntry struct {
	widget.Entry
	onSecondaryTap func(sel string, pos fyne.Position)
	minSize        fyne.Size
}

func (e *selEntry) MinSize() fyne.Size {
	base := e.Entry.MinSize()
	return fyne.NewSize(
		max32(base.Width, e.minSize.Width),
		max32(base.Height, e.minSize.Height),
	)
}

func max32(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func (e *selEntry) SetMinSize(size fyne.Size) {
	e.minSize = size
	e.Refresh()
}

func newSelEntry(onSecondaryTap func(sel string, pos fyne.Position)) *selEntry {
	e := &selEntry{onSecondaryTap: onSecondaryTap}
	e.ExtendBaseWidget(e)
	e.MultiLine = true
	e.Wrapping = fyne.TextWrapOff
	e.TextStyle = fyne.TextStyle{Monospace: true}
	// Do NOT call Disable() — that breaks selection.
	// We block editing by overriding TypedRune/TypedKey below.
	return e
}

// TypedRune blocks character input so the entry behaves as read-only.
func (e *selEntry) TypedRune(_ rune) {}

// TypedKey allows navigation and selection keys but blocks editing keys.
func (e *selEntry) TypedKey(key *fyne.KeyEvent) {
	switch key.Name {
	case fyne.KeyBackspace, fyne.KeyDelete, fyne.KeyReturn, fyne.KeyTab:
		// block editing
	default:
		e.Entry.TypedKey(key)
	}
}

// TappedSecondary intercepts right-click: if text is selected show our
// context menu; otherwise fall through to the default (Copy etc.).
func (e *selEntry) TappedSecondary(ev *fyne.PointEvent) {
	sel := e.SelectedText()
	if sel != "" && e.onSecondaryTap != nil {
		e.onSecondaryTap(sel, ev.AbsolutePosition)
		return
	}
	e.Entry.TappedSecondary(ev)
}
