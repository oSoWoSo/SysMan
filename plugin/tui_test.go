package plugin

import (
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// newTestModel builds a tuiModel suitable for unit tests.
// It uses empty-dir backend so Dirs() and List() are safe to call without real paths.
func newTestModel(services []Service) tuiModel {
	ti := textinput.New()
	return tuiModel{
		backend:  NewRunitBackend("", ""),
		services: services,
		search:   ti,
	}
}

// ── filtered ─────────────────────────────────────────────────────────

func TestFiltered_All(t *testing.T) {
	m := newTestModel([]Service{
		{Name: "alpha", Enabled: true},
		{Name: "beta", Enabled: false},
	})
	got := m.filtered()
	if len(got) != 2 {
		t.Errorf("expected 2 services with filterAll, got %d", len(got))
	}
}

func TestFiltered_Enabled(t *testing.T) {
	m := newTestModel([]Service{
		{Name: "alpha", Enabled: true},
		{Name: "beta", Enabled: false},
	})
	m.filter = tuiFilterEnabled
	got := m.filtered()
	if len(got) != 1 || got[0].Name != "alpha" {
		t.Errorf("expected only 'alpha', got %v", got)
	}
}

func TestFiltered_Disabled(t *testing.T) {
	m := newTestModel([]Service{
		{Name: "alpha", Enabled: true},
		{Name: "beta", Enabled: false},
	})
	m.filter = tuiFilterDisabled
	got := m.filtered()
	if len(got) != 1 || got[0].Name != "beta" {
		t.Errorf("expected only 'beta', got %v", got)
	}
}

func TestFiltered_SearchMatchesSubstring(t *testing.T) {
	m := newTestModel([]Service{
		{Name: "nginx", Enabled: true},
		{Name: "sshd", Enabled: false},
	})
	m.search.SetValue("ng")
	got := m.filtered()
	if len(got) != 1 || got[0].Name != "nginx" {
		t.Errorf("expected only 'nginx', got %v", got)
	}
}

func TestFiltered_SearchCaseInsensitive(t *testing.T) {
	m := newTestModel([]Service{
		{Name: "NGINX", Enabled: true},
		{Name: "sshd", Enabled: false},
	})
	m.search.SetValue("nginx")
	got := m.filtered()
	if len(got) != 1 || got[0].Name != "NGINX" {
		t.Errorf("expected 'NGINX', got %v", got)
	}
}

func TestFiltered_SearchNoMatch(t *testing.T) {
	m := newTestModel([]Service{
		{Name: "nginx", Enabled: true},
	})
	m.search.SetValue("zzz")
	got := m.filtered()
	if len(got) != 0 {
		t.Errorf("expected 0 matches, got %d", len(got))
	}
}

func TestFiltered_EmptyServices(t *testing.T) {
	m := newTestModel(nil)
	got := m.filtered()
	if len(got) != 0 {
		t.Errorf("expected 0 services, got %d", len(got))
	}
}

func TestFiltered_EnabledFilterWithSearch(t *testing.T) {
	m := newTestModel([]Service{
		{Name: "nginx", Enabled: true},
		{Name: "ntpd", Enabled: true},
		{Name: "sshd", Enabled: false},
	})
	m.filter = tuiFilterEnabled
	m.search.SetValue("ng")
	got := m.filtered()
	if len(got) != 1 || got[0].Name != "nginx" {
		t.Errorf("expected only 'nginx', got %v", got)
	}
}

// ── clampCursor ──────────────────────────────────────────────────────

func TestClampCursor_EmptyList(t *testing.T) {
	m := newTestModel(nil)
	m.cursor = 5
	m = m.clampCursor()
	if m.cursor != 0 {
		t.Errorf("expected cursor 0 for empty list, got %d", m.cursor)
	}
}

func TestClampCursor_BeyondEnd(t *testing.T) {
	m := newTestModel([]Service{{Name: "svc"}})
	m.cursor = 10
	m = m.clampCursor()
	if m.cursor != 0 {
		t.Errorf("expected cursor clamped to 0 (last index), got %d", m.cursor)
	}
}

func TestClampCursor_ValidPosition(t *testing.T) {
	m := newTestModel([]Service{{Name: "a"}, {Name: "b"}, {Name: "c"}})
	m.cursor = 1
	m = m.clampCursor()
	if m.cursor != 1 {
		t.Errorf("expected cursor unchanged at 1, got %d", m.cursor)
	}
}

// ── Init ─────────────────────────────────────────────────────────────

func TestInit_ReturnsNil(t *testing.T) {
	m := newTestModel(nil)
	if cmd := m.Init(); cmd != nil {
		t.Error("expected Init to return nil")
	}
}

// ── Update — messages ────────────────────────────────────────────────

func TestUpdate_WindowSize(t *testing.T) {
	m := newTestModel(nil)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	tm := updated.(tuiModel)
	if tm.width != 120 || tm.height != 40 {
		t.Errorf("expected 120×40, got %d×%d", tm.width, tm.height)
	}
}

func TestUpdate_ReloadMsg_RefreshesServices(t *testing.T) {
	m := newTestModel([]Service{{Name: "stale"}})
	m.backend = NewRunitBackend(t.TempDir(), t.TempDir())

	updated, _ := m.Update(tuiReloadMsg{})
	tm := updated.(tuiModel)
	// temp dir has no service subdirs, so services should be empty after reload
	if len(tm.services) != 0 {
		t.Errorf("expected 0 services after reload, got %d", len(tm.services))
	}
}

func TestUpdate_StatusMsg_SetsStatusAndTriggersReload(t *testing.T) {
	m := newTestModel(nil)
	updated, cmd := m.Update(tuiStatusMsg{msg: "done"})
	tm := updated.(tuiModel)
	if tm.status != "done" {
		t.Errorf("expected status 'done', got %q", tm.status)
	}
	if tm.statusErr {
		t.Error("expected statusErr false")
	}
	if cmd == nil {
		t.Error("expected a follow-up reload command")
	}
}

func TestUpdate_ErrMsg_SetsErrorStatus(t *testing.T) {
	m := newTestModel(nil)
	updated, _ := m.Update(tuiErrMsg{err: errors.New("boom")})
	tm := updated.(tuiModel)
	if tm.status != "boom" {
		t.Errorf("expected status 'boom', got %q", tm.status)
	}
	if !tm.statusErr {
		t.Error("expected statusErr true")
	}
}

// ── Update — keyboard ────────────────────────────────────────────────

func TestUpdate_KeyDown_MovesCursor(t *testing.T) {
	m := newTestModel([]Service{{Name: "a"}, {Name: "b"}, {Name: "c"}})
	m.cursor = 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	tm := updated.(tuiModel)
	if tm.cursor != 1 {
		t.Errorf("expected cursor 1 after 'j', got %d", tm.cursor)
	}
}

func TestUpdate_KeyDown_ClampedAtEnd(t *testing.T) {
	m := newTestModel([]Service{{Name: "a"}, {Name: "b"}})
	m.cursor = 1
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	tm := updated.(tuiModel)
	if tm.cursor != 1 {
		t.Errorf("expected cursor clamped at 1, got %d", tm.cursor)
	}
}

func TestUpdate_KeyUp_MovesCursor(t *testing.T) {
	m := newTestModel([]Service{{Name: "a"}, {Name: "b"}})
	m.cursor = 1
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	tm := updated.(tuiModel)
	if tm.cursor != 0 {
		t.Errorf("expected cursor 0 after 'k', got %d", tm.cursor)
	}
}

func TestUpdate_KeyUp_ClampedAtStart(t *testing.T) {
	m := newTestModel([]Service{{Name: "a"}, {Name: "b"}})
	m.cursor = 0
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	tm := updated.(tuiModel)
	if tm.cursor != 0 {
		t.Errorf("expected cursor clamped at 0, got %d", tm.cursor)
	}
}

func TestUpdate_KeyTab_CyclesFilter(t *testing.T) {
	m := newTestModel(nil)

	// All → Enabled
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	tm := updated.(tuiModel)
	if tm.filter != tuiFilterEnabled {
		t.Errorf("expected filterEnabled after tab, got %d", tm.filter)
	}

	// Enabled → Disabled
	updated, _ = tm.Update(tea.KeyMsg{Type: tea.KeyTab})
	tm = updated.(tuiModel)
	if tm.filter != tuiFilterDisabled {
		t.Errorf("expected filterDisabled after second tab, got %d", tm.filter)
	}

	// Disabled → All
	updated, _ = tm.Update(tea.KeyMsg{Type: tea.KeyTab})
	tm = updated.(tuiModel)
	if tm.filter != tuiFilterAll {
		t.Errorf("expected filterAll after third tab, got %d", tm.filter)
	}
}

func TestUpdate_KeySlash_EntersSearchMode(t *testing.T) {
	m := newTestModel(nil)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	tm := updated.(tuiModel)
	if !tm.searchMode {
		t.Error("expected searchMode true after '/'")
	}
}

func TestUpdate_KeyQ_Quits(t *testing.T) {
	m := newTestModel(nil)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("expected quit command after 'q'")
	}
}

// ── Update — search mode ─────────────────────────────────────────────

func TestUpdate_SearchMode_EscExitsAndClears(t *testing.T) {
	m := newTestModel([]Service{{Name: "nginx"}})
	m.searchMode = true
	m.search.SetValue("ng")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	tm := updated.(tuiModel)
	if tm.searchMode {
		t.Error("expected searchMode false after Esc")
	}
	if tm.search.Value() != "" {
		t.Errorf("expected empty search after Esc, got %q", tm.search.Value())
	}
}

func TestUpdate_SearchMode_EnterExitsWithoutClearing(t *testing.T) {
	m := newTestModel([]Service{{Name: "nginx"}})
	m.searchMode = true
	m.search.SetValue("ng")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	tm := updated.(tuiModel)
	if tm.searchMode {
		t.Error("expected searchMode false after Enter")
	}
	if tm.search.Value() != "ng" {
		t.Errorf("expected search value preserved, got %q", tm.search.Value())
	}
}

// ── View ─────────────────────────────────────────────────────────────

func TestView_DoesNotPanic(t *testing.T) {
	m := newTestModel([]Service{
		{Name: "nginx", Enabled: true},
		{Name: "sshd", Enabled: false},
	})
	out := m.View()
	if out == "" {
		t.Error("expected non-empty View output")
	}
}

func TestView_EmptyServices_DoesNotPanic(t *testing.T) {
	m := newTestModel(nil)
	out := m.View()
	if out == "" {
		t.Error("expected non-empty View output even with no services")
	}
}

func TestView_WithErrorStatus(t *testing.T) {
	m := newTestModel(nil)
	m.status = "something went wrong"
	m.statusErr = true
	out := m.View()
	if !strings.Contains(out, "something went wrong") {
		t.Errorf("expected error status in View output, got:\n%s", out)
	}
}

func TestView_WithOkStatus(t *testing.T) {
	m := newTestModel(nil)
	m.status = "service enabled"
	out := m.View()
	if !strings.Contains(out, "service enabled") {
		t.Errorf("expected ok status in View output, got:\n%s", out)
	}
}

func TestView_SearchMode_DoesNotPanic(t *testing.T) {
	m := newTestModel([]Service{{Name: "nginx"}})
	m.searchMode = true
	out := m.View()
	if out == "" {
		t.Error("expected non-empty View output in search mode")
	}
}

func TestView_WithTerminalSize(t *testing.T) {
	m := newTestModel([]Service{{Name: "nginx"}, {Name: "sshd"}})
	m.width = 160
	m.height = 50
	out := m.View()
	if out == "" {
		t.Error("expected non-empty View output with terminal size set")
	}
}

// ── filter label ─────────────────────────────────────────────────────

func TestFilterLabel_All(t *testing.T) {
	f := tuiFilterAll
	if f.label() == "" {
		t.Error("expected non-empty label for filterAll")
	}
}

func TestFilterLabel_Enabled(t *testing.T) {
	f := tuiFilterEnabled
	if f.label() == "" {
		t.Error("expected non-empty label for filterEnabled")
	}
}

func TestFilterLabel_Disabled(t *testing.T) {
	f := tuiFilterDisabled
	if f.label() == "" {
		t.Error("expected non-empty label for filterDisabled")
	}
}
