package api

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ── isGRCColor ─────────────────────────────────────────────────────────

func TestIsGRCColor_Valid(t *testing.T) {
	for _, name := range []string{"red", "yel", "cya", "grn", "mag", "blk", "whi", "blu"} {
		if !isGRCColor(name) {
			t.Errorf("isGRCColor(%q) = false, want true", name)
		}
	}
}

func TestIsGRCColor_ValidUppercase(t *testing.T) {
	for _, name := range []string{"RED", "YEL", "CYA", "GRN", "MAG", "BLK", "WHI", "BLU"} {
		if !isGRCColor(name) {
			t.Errorf("isGRCColor(%q) = false, want true", name)
		}
	}
}

func TestIsGRCColor_ValidWithSpaces(t *testing.T) {
	// isGRCColor trims space — used for full-format col 21-23 which may have trailing space
	if !isGRCColor("blu") {
		t.Error("isGRCColor(\"blu\") = false")
	}
}

func TestIsGRCColor_Invalid(t *testing.T) {
	for _, name := range []string{"", "green", "blue", "yellow", "cyan", "xyz", "123"} {
		if isGRCColor(name) {
			t.Errorf("isGRCColor(%q) = true, want false", name)
		}
	}
}

// ── isMatchType ────────────────────────────────────────────────────────

func TestIsMatchType_Valid(t *testing.T) {
	for _, mt := range []string{"s", "r", "R", "c", "t", " "} {
		if !isMatchType(mt) {
			t.Errorf("isMatchType(%q) = false, want true", mt)
		}
	}
}

func TestIsMatchType_Invalid(t *testing.T) {
	for _, mt := range []string{"", "x", "S", "rr", "regex"} {
		if isMatchType(mt) {
			t.Errorf("isMatchType(%q) = true, want false", mt)
		}
	}
}

// ── compilePattern ─────────────────────────────────────────────────────

func TestCompilePattern_String(t *testing.T) {
	re, err := compilePattern("s", "foo.bar")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !re.MatchString("foo.bar") {
		t.Error("should match literal 'foo.bar'")
	}
	if re.MatchString("fooXbar") {
		t.Error("should not match 'fooXbar' (dot is literal)")
	}
}

func TestCompilePattern_Regex(t *testing.T) {
	re, err := compilePattern("r", "err.*:")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !re.MatchString("error: something") {
		t.Error("should match 'error: something'")
	}
	if re.MatchString("ERR: something") {
		t.Error("r should be case-sensitive")
	}
}

func TestCompilePattern_RegexCaseInsensitive(t *testing.T) {
	re, err := compilePattern("R", "error")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !re.MatchString("ERROR") {
		t.Error("R should match uppercase ERROR")
	}
	if !re.MatchString("error") {
		t.Error("R should match lowercase error")
	}
}

func TestCompilePattern_Char(t *testing.T) {
	re, err := compilePattern("c", "abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !re.MatchString("x a y") {
		t.Error("should match line containing 'a'")
	}
	if !re.MatchString("xbx") {
		t.Error("should match line containing 'b'")
	}
	if re.MatchString("xyz") {
		t.Error("should not match line with none of a/b/c")
	}
}

func TestCompilePattern_CharSpecial(t *testing.T) {
	// [*] is the char set '*', '[', ']' — the '[' and ']' must be quoted
	re, err := compilePattern("c", "[*]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !re.MatchString("[installed]") {
		t.Error("should match line containing '['")
	}
	if !re.MatchString("foo*bar") {
		t.Error("should match line containing '*'")
	}
}

func TestCompilePattern_Time(t *testing.T) {
	re, err := compilePattern("t", `\d+`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !re.MatchString("1234567890") {
		t.Error("t type should work as plain regex")
	}
}

func TestCompilePattern_SpaceDefault(t *testing.T) {
	re, err := compilePattern(" ", "warn")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !re.MatchString("warning: foo") {
		t.Error("space type should work as plain regex")
	}
}

// ── grcColor ───────────────────────────────────────────────────────────

func TestGRCColor_AllColors(t *testing.T) {
	cases := []struct {
		name string
		col  color.RGBA
	}{
		{"red", color.RGBA{R: 0xFF, G: 0x55, B: 0x55, A: 0xFF}},
		{"yel", color.RGBA{R: 0xFF, G: 0xCC, B: 0x00, A: 0xFF}},
		{"cya", color.RGBA{R: 0x00, G: 0xDD, B: 0xFF, A: 0xFF}},
		{"grn", color.RGBA{R: 0x44, G: 0xDD, B: 0x77, A: 0xFF}},
		{"mag", color.RGBA{R: 0xFF, G: 0x55, B: 0xFF, A: 0xFF}},
		{"blk", color.RGBA{R: 0x44, G: 0x44, B: 0x44, A: 0xFF}},
		{"whi", color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}},
		{"blu", color.RGBA{R: 0x55, G: 0x99, B: 0xFF, A: 0xFF}},
	}
	for _, tc := range cases {
		got := grcColor(tc.name)
		if got != tc.col {
			t.Errorf("grcColor(%q) = %v, want %v", tc.name, got, tc.col)
		}
	}
}

func TestGRCColor_Unknown(t *testing.T) {
	got := grcColor("unknown")
	if got != color.White {
		t.Errorf("grcColor(\"unknown\") = %v, want white", got)
	}
}

// ── parseConfig ────────────────────────────────────────────────────────

func TestParseConfig_Empty(t *testing.T) {
	rules := parseConfig("")
	if len(rules) != 0 {
		t.Errorf("expected 0 rules, got %d", len(rules))
	}
}

func TestParseConfig_Comments(t *testing.T) {
	cfg := `
# this is a comment
  # indented comment

`
	rules := parseConfig(cfg)
	if len(rules) != 0 {
		t.Errorf("expected 0 rules from comments-only config, got %d", len(rules))
	}
}

func TestParseConfig_ShortFormat_StringMatch(t *testing.T) {
	cfg := "grn s running\n"
	rules := parseConfig(cfg)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if !rules[0].re.MatchString("running") {
		t.Error("rule should match 'running'")
	}
	if rules[0].col != grcColor("grn") {
		t.Error("wrong color")
	}
	if rules[0].bold {
		t.Error("should not be bold")
	}
}

func TestParseConfig_ShortFormat_BoldAttribute(t *testing.T) {
	cfg := "red b s ERROR\n"
	rules := parseConfig(cfg)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if !rules[0].bold {
		t.Error("should be bold")
	}
}

func TestParseConfig_ShortFormat_MultipleRules(t *testing.T) {
	cfg := "grn s running\nred b s ERROR\nyel s WARNING\n"
	rules := parseConfig(cfg)
	if len(rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(rules))
	}
}

func TestParseConfig_ShortFormat_PatternWithSpaces(t *testing.T) {
	cfg := "red s Do you want to continue? [Y/n]\n"
	rules := parseConfig(cfg)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if !rules[0].re.MatchString("Do you want to continue? [Y/n]") {
		t.Error("rule should match multi-word pattern")
	}
}

func TestParseConfig_ShortFormat_InvalidColor(t *testing.T) {
	cfg := "green s running\n"
	rules := parseConfig(cfg)
	if len(rules) != 0 {
		t.Errorf("expected 0 rules for unknown color 'green', got %d", len(rules))
	}
}

func TestParseConfig_ShortFormat_TooFewFields(t *testing.T) {
	cfg := "grn\n"
	rules := parseConfig(cfg)
	if len(rules) != 0 {
		t.Errorf("expected 0 rules for too-few-fields line, got %d", len(rules))
	}
}

func TestParseConfig_FullGRCFormat(t *testing.T) {
	// Full grc format: HTML COLOR (0-20), COL (21-23), sp(24), A(25), sp(26), N(27), sp(28), T(29), sp(30), PATTERN(31+)
	// "Blue                 blu     s running"
	//  0         1         2         3
	//  01234567890123456789012345678901234567
	cfg := "Blue                 blu     s running\n"
	rules := parseConfig(cfg)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d (line: %q)", len(rules), cfg)
	}
	if !rules[0].re.MatchString("running") {
		t.Error("rule should match 'running'")
	}
	if rules[0].col != grcColor("blu") {
		t.Error("wrong color")
	}
}

func TestParseConfig_FullGRCFormat_Bold(t *testing.T) {
	// "Red                  red b   s ERROR"
	//  0         1         2         3
	//  0123456789012345678901234567890123456
	cfg := "Red                  red b   s ERROR\n"
	rules := parseConfig(cfg)
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d (line: %q)", len(rules), cfg)
	}
	if !rules[0].bold {
		t.Error("should be bold (attr='b' at position 25)")
	}
	if rules[0].col != grcColor("red") {
		t.Error("wrong color")
	}
}

func TestParseConfig_FullGRCFormat_ColumnPositions(t *testing.T) {
	// Exact column layout (0-based):
	//  0-20 : HTML color name (21 chars including trailing spaces)
	// 21-23 : COL
	//    24 : space
	//    25 : A attribute (' ' or 'b')
	//    26 : space
	//    27 : N count (ignored, always ' ')
	//    28 : space
	//    29 : T match type
	//    30 : space
	//   31+ : PATTERN
	line := "Blue                 blu b   s test pattern"
	//       0         1         2         3         4
	//       0123456789012345678901234567890123456789012

	if line[21:24] != "blu" {
		t.Fatalf("test setup: col not at 21-23, got %q", line[21:24])
	}
	if line[25] != 'b' {
		t.Fatalf("test setup: attr not at 25, got %q", string(line[25]))
	}
	if line[27] != ' ' {
		t.Fatalf("test setup: N not at 27, got %q", string(line[27]))
	}
	if line[29] != 's' {
		t.Fatalf("test setup: type not at 29, got %q", string(line[29]))
	}

	rules := parseConfig(line + "\n")
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if !rules[0].bold {
		t.Error("should be bold")
	}
	if !rules[0].re.MatchString("test pattern") {
		t.Error("should match 'test pattern'")
	}
}

func TestParseConfig_DefaultConfig(t *testing.T) {
	// Default config must parse without errors and produce rules.
	rules := parseConfig(defaultConfig)
	if len(rules) == 0 {
		t.Error("default config should produce at least one rule")
	}
}

func TestParseConfig_DefaultConfig_KnownRules(t *testing.T) {
	rules := parseConfig(defaultConfig)

	// Build a quick lookup: line → (color name, bold)
	type match struct {
		col  color.Color
		bold bool
	}
	check := func(line string) (match, bool) {
		for _, r := range rules {
			if r.re.MatchString(line) {
				return match{r.col, r.bold}, true
			}
		}
		return match{}, false
	}

	cases := []struct {
		line string
		col  color.Color
		bold bool
	}{
		{"ERROR occurred", grcColor("red"), true},
		{"error: oops", grcColor("red"), true},
		{"Warning: something", grcColor("yel"), true},
		{"running service", grcColor("blu"), false},
		{"==> building", grcColor("grn"), true},
	}
	for _, tc := range cases {
		m, ok := check(tc.line)
		if !ok {
			t.Errorf("no rule matched %q", tc.line)
			continue
		}
		if m.col != tc.col {
			t.Errorf("line %q: wrong color %v, want %v", tc.line, m.col, tc.col)
		}
		if m.bold != tc.bold {
			t.Errorf("line %q: bold=%v, want %v", tc.line, m.bold, tc.bold)
		}
	}
}

// ── Highlighter.matchLine ──────────────────────────────────────────────

func TestHighlighter_MatchLine_NoMatch(t *testing.T) {
	h := &Highlighter{rules: parseConfig("grn s running\n")}
	col, bold := h.matchLine("nothing here")
	if col != nil || bold {
		t.Error("should not match unrelated line")
	}
}

func TestHighlighter_MatchLine_Match(t *testing.T) {
	h := &Highlighter{rules: parseConfig("grn s running\n")}
	col, bold := h.matchLine("service running")
	if col == nil {
		t.Fatal("expected a color match")
	}
	if col != grcColor("grn") {
		t.Errorf("wrong color: %v", col)
	}
	if bold {
		t.Error("should not be bold")
	}
}

func TestHighlighter_MatchLine_FirstRuleWins(t *testing.T) {
	cfg := "grn s foo\nred s foo\n"
	h := &Highlighter{rules: parseConfig(cfg)}
	col, _ := h.matchLine("foo bar")
	if col != grcColor("grn") {
		t.Error("first matching rule should win")
	}
}

// ── Highlighter.RichSegments ───────────────────────────────────────────

func TestRichSegments_Empty(t *testing.T) {
	h := &Highlighter{}
	segs := h.RichSegments("")
	// empty string → one empty line
	if len(segs) == 0 {
		t.Error("expected at least one segment for empty string")
	}
}

func TestRichSegments_SingleLine_NoMatch(t *testing.T) {
	h := &Highlighter{rules: parseConfig("grn s running\n")}
	segs := h.RichSegments("no match here")
	// 1 text segment (no newline appended for single line)
	if len(segs) != 1 {
		t.Errorf("single unmatched line: expected 1 segment, got %d", len(segs))
	}
}

func TestRichSegments_MultiLine_NewlineSeparators(t *testing.T) {
	h := &Highlighter{}
	text := "line1\nline2\nline3"
	segs := h.RichSegments(text)
	// 3 lines + 2 newline segments = 5
	if len(segs) != 5 {
		t.Errorf("3 lines: expected 5 segments, got %d", len(segs))
	}
}

func TestRichSegments_ColoredSegmentType(t *testing.T) {
	h := &Highlighter{rules: parseConfig("red b s ERROR\n")}
	segs := h.RichSegments("ERROR: something failed")
	if len(segs) == 0 {
		t.Fatal("expected segments")
	}
	cs, ok := segs[0].(*coloredSegment)
	if !ok {
		t.Fatalf("expected *coloredSegment, got %T", segs[0])
	}
	if cs.col != grcColor("red") {
		t.Errorf("wrong color: %v", cs.col)
	}
	if !cs.bold {
		t.Error("should be bold")
	}
}

// ── ParseHexColor ──────────────────────────────────────────────────────

func TestParseHexColor_Valid(t *testing.T) {
	cases := []struct {
		input string
		want  color.RGBA
	}{
		{"#ff5555", color.RGBA{R: 0xFF, G: 0x55, B: 0x55, A: 0xFF}},
		{"#000000", color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF}},
		{"#ffffff", color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}},
		{"#AABBCC", color.RGBA{R: 0xAA, G: 0xBB, B: 0xCC, A: 0xFF}},
		{"ff5555", color.RGBA{R: 0xFF, G: 0x55, B: 0x55, A: 0xFF}}, // without #
	}
	for _, tc := range cases {
		got := ParseHexColor(tc.input)
		if got != tc.want {
			t.Errorf("ParseHexColor(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestParseHexColor_Invalid(t *testing.T) {
	for _, s := range []string{"#xyz", "#12345", "#1234567", "", "nope"} {
		got := ParseHexColor(s)
		if got != color.White {
			t.Errorf("ParseHexColor(%q) = %v, want white", s, got)
		}
	}
}

// ── Live reload ────────────────────────────────────────────────────────

func TestHighlighter_LiveReload(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "highlight.conf")

	// Write initial config: grn matches "ok"
	if err := os.WriteFile(cfgPath, []byte("grn s ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Override the config path via env so highlightConfigPath() returns our temp file.
	t.Setenv("XDG_CONFIG_HOME", dir)
	// Adjust sub-dir to match highlightConfigPath() → <cfgDir>/svman/highlight.conf
	svmanDir := filepath.Join(dir, "svman")
	if err := os.MkdirAll(svmanDir, 0o755); err != nil {
		t.Fatal(err)
	}
	realPath := filepath.Join(svmanDir, "highlight.conf")
	if err := os.WriteFile(realPath, []byte("grn s ok\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	h := NewHighlighter()
	defer h.Close()

	// Initial rule: "ok" → green.
	col, _ := h.matchLine("everything ok")
	if col != grcColor("grn") {
		t.Fatalf("initial: expected grn, got %v", col)
	}
	col2, _ := h.matchLine("error here")
	if col2 != nil {
		t.Fatal("initial: 'error here' should not match")
	}

	// Overwrite config: now red matches "error".
	if err := os.WriteFile(realPath, []byte("red b s error\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Wait up to 2 s for the watcher goroutine to reload.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		c, _ := h.matchLine("error here")
		if c == grcColor("red") {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	col3, bold := h.matchLine("error here")
	if col3 != grcColor("red") {
		t.Errorf("after reload: expected red, got %v", col3)
	}
	if !bold {
		t.Error("after reload: expected bold")
	}
	// Old rule should be gone.
	col4, _ := h.matchLine("everything ok")
	if col4 != nil {
		t.Error("after reload: 'ok' should no longer match")
	}
}
