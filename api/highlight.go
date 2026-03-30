package api

import (
	"bufio"
	"image/color"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"github.com/fsnotify/fsnotify"
)

// ── Config file path ──────────────────────────────────────────────────

func highlightConfigPath() string {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(cfg, "svman", "highlight.conf")
}

// ── Default config (grc-compatible format) ────────────────────────────

// defaultConfig is written to ~/.config/svman/highlight.conf on first run.
// Format: each non-comment, non-blank line has fixed columns:
//
//	COL A T PATTERN
//	  COL : red yel cya grn mag blk whi blu
//	  A   : ' '=normal  b=bold  u=underline
//	  T   : s=string  r=regex(case-sensitive)  R=regex(insensitive)  c=char(any)
//	  PATTERN: rest of line (one space separator after T)
const defaultConfig = `#        1         2         3         4         5
#2345678901234567890123456789012345678901234567890123456789
# HTML COLOR         COL A N T STRING or REGULAR EXPRESSION
#################### ### # # # ############################
#Where:
#  HTML COLOR - Standard HTML Color name (ignored in GUI output)
#  COL        - Console color: red, yel, cya, grn, mag, blk, whi, blu
#  A          - Attribute: ' '=normal  b=bold  (u/r/k ignored in GUI)
#  N          - Number of matches: ' '/0=all  1-9=count (ignored here)
#  T          - Match type: s=string  r=regex  R=regex(insensitive)  c=chars
#
Blue                 blu     s running
Blue                 blu     s Compiling
Blue                 blu     s note:
Cyan                 cya     s test
Green                grn     s xbps-src:
Green                grn b   s ==>
Green                grn b   s =>
Green                grn     s Found
Green                grn     s TESTING
Green                grn     s installing host dependencies:
Green                grn     s installing target dependencies:
Green                grn     s MB/s
Green                grn     s  ok
Green                grn     s Finished
Green                grn     s passed
Green                grn     s PASS
Green                grn     s packages registered
Green                grn     s Stripped executable:
Green                grn     s Stripped position-independent executable:
Green                grn     s cmd:
Green                grn     s SONAME:
Green                grn     s index:
Green                grn     c [*]
Red                  red b   s ERROR
Red                  red b   s error:
Red                  red     s Removing
Red                  red b   s unresolved
Red                  red b   s Transaction aborted
Red                  red b   s broken
Red                  red b   s failures:
Red                  red b   s FAILED
Red                  red b   s failed
Red                  red     s ->
Red                  red     s <->
Red                  red     s Do you want to continue? [Y/n]
Red                  red     s Size to download
Yellow               yel     s SLOW
Yellow               yel     s configured,
Yellow               yel b   s continue
Yellow               yel b   s ignored
Yellow               yel b   s Warning:
Yellow               yel b   s warning:
Yellow               yel b   s WARNING:
`

// ── Parser ────────────────────────────────────────────────────────────

type parsedRule struct {
	re   *regexp.Regexp
	col  color.Color
	bold bool
}

// parseConfig parses grc-format highlight rules.
//
// Two line formats are supported:
//
//	Full grc (fixed columns, HTML COLOR in col 1-20):
//	  "Blue                 blu b   s  pattern"
//	  col 21-23 = COL, col 25 = A, col 27 = N (ignored), col 29 = T, col 31+ = PATTERN
//
//	Short (no HTML COLOR prefix):
//	  "blu b s pattern"   or   "blu   s pattern"
//	  fields[0]=COL  fields[1]=A  fields[2]=T  fields[3…]=PATTERN
//	  (A and N may be collapsed so fields[1] may be T directly)
//
// Lines starting with '#' or blank are skipped.
func parseConfig(data string) []parsedRule {
	var rules []parsedRule
	scanner := bufio.NewScanner(strings.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		var colName, attr, matchType, pattern string

		// Detect full grc format: line is long enough and col 21-23 (0-based) is a known colour.
		if len(line) >= 32 && isGRCColor(line[21:24]) {
			// Fixed-column layout (0-based, matching grc spec with 1-based col numbers):
			//  0-20  : HTML color name (ignored)
			// 21-23  : COL (3 chars)
			// 24     : space
			// 25     : A attribute
			// 26     : space
			// 27     : N (ignored)
			// 28     : space
			// 29     : T
			// 30     : space
			// 31+    : PATTERN
			colName = line[21:24]
			attr = string(line[25])
			matchType = string(line[29])
			if len(line) <= 31 {
				continue
			}
			pattern = strings.TrimLeft(line[30:], " ")
		} else {
			// Short format: whitespace-separated fields
			// COL [A] T PATTERN   (A may be absent / collapsed with spaces)
			fields := strings.Fields(line)
			if len(fields) < 3 {
				continue
			}
			colName = fields[0]
			if !isGRCColor(colName) {
				continue
			}
			// Determine if second field is an attribute or a match-type.
			if isMatchType(fields[1]) {
				// "col T pattern…"  (no explicit A)
				attr = " "
				matchType = fields[1]
				pattern = strings.Join(fields[2:], " ")
			} else {
				// "col A T pattern…"
				if len(fields) < 4 {
					continue
				}
				attr = fields[1]
				matchType = fields[2]
				pattern = strings.Join(fields[3:], " ")
			}
		}

		if !isMatchType(matchType) {
			continue
		}

		col := grcColor(colName)
		bold := attr == "b"

		re, err := compilePattern(matchType, pattern)
		if err != nil {
			continue
		}
		rules = append(rules, parsedRule{re: re, col: col, bold: bold})
	}
	return rules
}

func isGRCColor(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "red", "yel", "cya", "grn", "mag", "blk", "whi", "blu":
		return true
	}
	return false
}

func isMatchType(s string) bool {
	switch s {
	case "s", "r", "R", "c", "t", " ":
		return true
	}
	return false
}

func compilePattern(matchType, pattern string) (*regexp.Regexp, error) {
	switch matchType {
	case "s": // literal string
		return regexp.Compile(regexp.QuoteMeta(pattern))
	case "r", " ": // regex case-sensitive (space = default = r)
		return regexp.Compile(pattern)
	case "R": // regex case-insensitive
		return regexp.Compile("(?i)" + pattern)
	case "c": // match any character from the set
		var parts []string
		for _, ch := range pattern {
			parts = append(parts, regexp.QuoteMeta(string(ch)))
		}
		return regexp.Compile(strings.Join(parts, "|"))
	case "t": // Unix timestamp regex (treat as plain regex, no time conversion in GUI)
		return regexp.Compile(pattern)
	}
	return nil, nil
}

// ── Colour map ────────────────────────────────────────────────────────

func grcColor(name string) color.Color {
	switch strings.ToLower(name) {
	case "red":
		return color.RGBA{R: 0xFF, G: 0x55, B: 0x55, A: 0xFF}
	case "yel":
		return color.RGBA{R: 0xFF, G: 0xCC, B: 0x00, A: 0xFF}
	case "cya":
		return color.RGBA{R: 0x00, G: 0xDD, B: 0xFF, A: 0xFF}
	case "grn":
		return color.RGBA{R: 0x44, G: 0xDD, B: 0x77, A: 0xFF}
	case "mag":
		return color.RGBA{R: 0xFF, G: 0x55, B: 0xFF, A: 0xFF}
	case "blk":
		return color.RGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xFF}
	case "whi":
		return color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	case "blu":
		return color.RGBA{R: 0x55, G: 0x99, B: 0xFF, A: 0xFF}
	}
	return color.White
}

// ── Highlighter ───────────────────────────────────────────────────────

// Highlighter applies colour rules to lines of text.
// It watches ~/.config/svman/highlight.conf and reloads rules automatically
// whenever the file is saved, with no restart required.
type Highlighter struct {
	mu      sync.RWMutex
	rules   []parsedRule
	watcher *fsnotify.Watcher // nil when watching is unavailable
}

// NewHighlighter loads rules from ~/.config/svman/highlight.conf (grc format).
// If the file does not exist it is created with built-in defaults.
// A background goroutine watches the file for changes and reloads rules live.
func NewHighlighter() *Highlighter {
	h := &Highlighter{}
	h.reload()
	h.watch()
	return h
}

// reload reads the config file (writing defaults if absent) and replaces the rules.
func (h *Highlighter) reload() {
	rules := parseConfig(loadConfigData())
	h.mu.Lock()
	h.rules = rules
	h.mu.Unlock()
}

// watch starts a background goroutine that re-reads the config on any write/create
// event on the highlight config file. Errors starting the watcher are silently
// ignored — highlighting still works, just without live reload.
func (h *Highlighter) watch() {
	path := highlightConfigPath()
	if path == "" {
		return
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return
	}
	// Watch the directory so we also catch atomic saves (write-rename).
	if err := w.Add(filepath.Dir(path)); err != nil {
		w.Close() //nolint:errcheck
		return
	}
	h.watcher = w
	go func() {
		defer w.Close() //nolint:errcheck
		for {
			select {
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if ev.Name != path {
					continue
				}
				if ev.Has(fsnotify.Write) || ev.Has(fsnotify.Create) {
					h.reload()
				}
			case _, ok := <-w.Errors:
				if !ok {
					return
				}
			}
		}
	}()
}

// Close stops the file watcher. Call when the Highlighter is no longer needed.
func (h *Highlighter) Close() {
	if h.watcher != nil {
		h.watcher.Close() //nolint:errcheck
	}
}

func loadConfigData() string {
	path := highlightConfigPath()
	if path == "" {
		return defaultConfig
	}
	raw, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		// Write defaults and use them.
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		_ = os.WriteFile(path, []byte(defaultConfig), 0o644)
		return defaultConfig
	}
	return string(raw)
}

// ── Rendering ─────────────────────────────────────────────────────────

// coloredSegment is a widget.RichTextSegment that renders one line in a fixed colour.
type coloredSegment struct {
	text string
	col  color.Color
	bold bool
}

func (s *coloredSegment) Inline() bool              { return false }
func (s *coloredSegment) Textual() string           { return s.text }
func (s *coloredSegment) Select(_, _ fyne.Position) {}
func (s *coloredSegment) SelectedText() string      { return "" }
func (s *coloredSegment) Unselect()                 {}

func (s *coloredSegment) Visual() fyne.CanvasObject {
	t := canvas.NewText(s.text, s.col)
	t.TextStyle.Bold = s.bold
	t.TextStyle.Monospace = true
	return t
}

func (s *coloredSegment) Update(o fyne.CanvasObject) {
	t := o.(*canvas.Text)
	t.Text = s.text
	t.Color = s.col
	t.TextStyle.Bold = s.bold
	t.Refresh()
}

// RichSegments converts a plain-text output string into RichText segments
// with syntax highlighting applied line by line.
func (h *Highlighter) RichSegments(text string) []widget.RichTextSegment {
	var segs []widget.RichTextSegment
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		col, bold := h.matchLine(line)
		var seg widget.RichTextSegment
		if col != nil {
			seg = &coloredSegment{text: line, col: col, bold: bold}
		} else {
			seg = &widget.TextSegment{
				Text:  line,
				Style: widget.RichTextStyle{TextStyle: fyne.TextStyle{Monospace: true}},
			}
		}
		segs = append(segs, seg)
		if i < len(lines)-1 {
			segs = append(segs, &widget.TextSegment{
				Text:  "\n",
				Style: widget.RichTextStyle{Inline: true},
			})
		}
	}
	return segs
}

func (h *Highlighter) matchLine(line string) (color.Color, bool) {
	h.mu.RLock()
	rules := h.rules
	h.mu.RUnlock()
	for _, r := range rules {
		if r.re.MatchString(line) {
			return r.col, r.bold
		}
	}
	return nil, false
}

// ── Hex colour helper (kept for potential future use) ─────────────────

func hexByte(s string) (uint8, bool) {
	var v uint8
	for _, c := range s {
		v <<= 4
		switch {
		case c >= '0' && c <= '9':
			v |= uint8(c - '0')
		case c >= 'a' && c <= 'f':
			v |= uint8(c-'a') + 10
		case c >= 'A' && c <= 'F':
			v |= uint8(c-'A') + 10
		default:
			return 0, false
		}
	}
	return v, true
}

// ParseHexColor parses "#rrggbb" into a color.Color.
func ParseHexColor(s string) color.Color {
	s = strings.TrimSpace(strings.TrimPrefix(s, "#"))
	if len(s) != 6 {
		return color.White
	}
	r, ok1 := hexByte(s[0:2])
	g, ok2 := hexByte(s[2:4])
	b, ok3 := hexByte(s[4:6])
	if !ok1 || !ok2 || !ok3 {
		return color.White
	}
	return color.RGBA{R: r, G: g, B: b, A: 0xFF}
}
