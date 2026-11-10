package srcman

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
	"codeberg.org/oSoWoSo/SysMan/src/common"
	serman "codeberg.org/oSoWoSo/SysMan/src/serman"
	zone "github.com/lrstanley/bubblezone/v2"
)

// ── Styles ────────────────────────────────────────────────────────────

var (
	xSubtle    = compat.AdaptiveColor{Light: lipgloss.Color("#9B9B9B"), Dark: lipgloss.Color("#585858")}
	xHighlight = compat.AdaptiveColor{Light: lipgloss.Color("#006688"), Dark: lipgloss.Color("#00DDFF")}
	xDanger    = compat.AdaptiveColor{Light: lipgloss.Color("#CC3333"), Dark: lipgloss.Color("#FF5555")}
	xSuccess   = compat.AdaptiveColor{Light: lipgloss.Color("#22AA55"), Dark: lipgloss.Color("#44DD77")}
	xWarn      = compat.AdaptiveColor{Light: lipgloss.Color("#BB8800"), Dark: lipgloss.Color("#FFCC00")}

	xTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(xHighlight)
	xSelectedStyle = common.TSelectedStyle.Padding(0, 1)
	xNormalStyle   = lipgloss.NewStyle().Padding(0, 1)
	xSubtleStyle   = lipgloss.NewStyle().Foreground(xSubtle)
	xDangerStyle   = lipgloss.NewStyle().Foreground(xDanger).Bold(true)
	xSuccessStyle  = lipgloss.NewStyle().Foreground(xSuccess)
	xWarnStyle     = lipgloss.NewStyle().Foreground(xWarn)
)

// ── Messages ──────────────────────────────────────────────────────────

type xbpsReloadMsg struct{}

type xbpsDoneMsg struct {
	action   string
	template string // template name that was processed (empty for global actions)
	output   string
	err      error
}

// ── Model ─────────────────────────────────────────────────────────────

type xbpsModel struct {
	id             string
	distDir        string
	templates      []Template
	cursor         int
	search         textinput.Model
	searchMode     bool
	output         string
	status         string
	statusErr      bool
	running        bool
	showAbout      bool
	width          int
	height         int
	buildQ         bool
	buildC         bool
	ctx            context.Context
	cancel         context.CancelFunc
	editor         textarea.Model
	editorPath     string
	editorMode     bool
	find           textinput.Model
	findMode       bool
	findIndex      int
	history        []string
	historyIdx     int
	cfg            Config
	settings       bool
	settingsIdx    int
	settingsFields []textinput.Model
	outputSelect   bool
	outputMenu     bool
	outputLine     int
	outputWord     int
}

// NewTuiModel creates an initialized xbps TUI model.
func NewTuiModel(distDir string) tea.Model {
	zone.NewGlobal()
	ti := textinput.New()
	ti.Placeholder = t("tui.search.placeholder")
	find := textinput.New()
	find.Placeholder = t("find.placeholder")
	editor := textarea.New()
	cfg := LoadConfig()
	m := xbpsModel{id: "srcman:" + zone.NewPrefix(), distDir: distDir, search: ti, find: find, editor: editor, cfg: cfg, historyIdx: -1}
	m.templates = LoadTemplates(distDir)
	m.loadEditorFile(m.selectedName())
	return m
}

func (m xbpsModel) filtered() []Template {
	return Filter(m.templates, m.search.Value())
}

func (m xbpsModel) clampCursor() xbpsModel {
	list := m.filtered()
	if len(list) == 0 {
		m.cursor = 0
		return m
	}
	if m.cursor >= len(list) {
		m.cursor = len(list) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	return m
}

func (m xbpsModel) selectedName() string {
	list := m.filtered()
	if m.cursor < 0 || m.cursor >= len(list) {
		return ""
	}
	return list[m.cursor].Name
}

func (m xbpsModel) CapturesGlobalKeys() bool {
	return m.editorMode || m.searchMode || m.findMode || m.settings || m.outputSelect
}

func (m *xbpsModel) loadEditorFile(name string) {
	if name == "" {
		m.editorPath = ""
		m.editor.SetValue("")
		m.editor.Blur()
		m.editorMode = false
		return
	}
	path := filepath.Join(ResolveDistDir(m.distDir), "srcpkgs", name, "template")
	if path == m.editorPath {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		m.editorPath = ""
		m.editor.SetValue("")
		m.status = t("status.read_err") + err.Error()
		m.statusErr = true
		return
	}
	m.editorPath = path
	m.editor.SetValue(string(data))
}

func (m xbpsModel) saveEditor() xbpsModel {
	if m.editorPath == "" {
		return m
	}
	if err := os.WriteFile(m.editorPath, []byte(m.editor.Value()), 0o644); err != nil {
		m.status = t("status.save_err") + err.Error()
		m.statusErr = true
	} else {
		m.status = t("status.save_ok")
		m.statusErr = false
	}
	return m
}

// ── tea.Model ─────────────────────────────────────────────────────────

func (m xbpsModel) Init() tea.Cmd { return nil }

func (m xbpsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok && common.ToggleTUIKeyHints(key) {
		return m, nil
	}
	if wheel, ok := msg.(tea.MouseWheelMsg); ok && !m.editorMode {
		keyCode := tea.KeyDown
		if wheel.Button == tea.MouseWheelUp {
			keyCode = tea.KeyUp
		}
		return m.Update(tea.KeyPressMsg{Code: keyCode})
	}
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		editorWidth := msg.Width - 8
		if msg.Width >= 100 {
			editorWidth = max(msg.Width*26/100-4, 1)
		}
		m.editor.SetWidth(max(editorWidth, 1))
		m.editor.SetHeight(max(msg.Height-12, 1))
		return m, nil

	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			break
		}
		if zone.Get(m.id + "a_editor_save").InBounds(msg) {
			return m.saveEditor(), nil
		}
		if zone.Get(m.id+"editor_area").InBounds(msg) && m.editorPath != "" {
			m.editorMode = true
			focusCmd := m.editor.Focus()
			var updateCmd tea.Cmd
			m.editor, updateCmd = m.editor.Update(msg)
			return m, tea.Batch(focusCmd, updateCmd)
		}
		// Status bar buttons.
		if zone.Get(m.id + "a_quit").InBounds(msg) {
			return m, tea.Quit
		}
		if zone.Get(m.id + "a_about").InBounds(msg) {
			m.showAbout = true
			return m, nil
		}
		if zone.Get(m.id + "a_copy").InBounds(msg) {
			if m.output != "" {
				if err := common.CopyToClipboard(m.output); err != nil {
					m.status = t("action.copy_err")
					m.statusErr = true
				} else {
					m.status = t("action.copied")
					m.statusErr = false
				}
			}
			return m, nil
		}
		if zone.Get(m.id + "a_settings_save").InBounds(msg) {
			return m.handleSettingsKey(tea.KeyPressMsg{Code: tea.KeyEnter})
		}
		if zone.Get(m.id + "a_settings_cancel").InBounds(msg) {
			m.settings = false
			return m, nil
		}
		if zone.Get(m.id + "a_find").InBounds(msg) {
			m.findMode = true
			m.find.Focus()
			return m, nil
		}
		if zone.Get(m.id + "a_settings").InBounds(msg) {
			m.settingsFields = make([]textinput.Model, 3)
			vals := []string{m.cfg.DistDir, m.cfg.SearchEngine, m.cfg.ForkURL}
			for i := range vals {
				m.settingsFields[i] = textinput.New()
				m.settingsFields[i].SetValue(vals[i])
			}
			m.settings = true
			return m, nil
		}
		if zone.Get(m.id + "a_reload").InBounds(msg) {
			return m, func() tea.Msg { return xbpsReloadMsg{} }
		}
		if zone.Get(m.id + "a_output_select").InBounds(msg) {
			m.startOutputSelection()
			return m, nil
		}
		if m.outputMenu {
			for i, field := range []string{"hostmakedepends", "makedepends", "depends", "checkdepends"} {
				if zone.Get(m.id + fmt.Sprintf("a_outdep_%d", i)).InBounds(msg) {
					return m.outputDependency(field)
				}
			}
			if zone.Get(m.id + "a_out_xlocate").InBounds(msg) {
				return m.outputCommand("xlocate")
			}
			if zone.Get(m.id + "a_out_query").InBounds(msg) {
				return m.outputCommand("xbps-query")
			}
			if zone.Get(m.id + "a_out_web").InBounds(msg) {
				OpenBrowser(m.cfg.SearchEngine + url.QueryEscape(m.selectedOutputWord()))
				m.outputMenu = false
				m.outputSelect = false
				return m, nil
			}
		}
		if zone.Get(m.id + "a_back").InBounds(msg) {
			if len(m.history) > 0 {
				if m.historyIdx == -1 {
					m.historyIdx = len(m.history) - 1
				} else if m.historyIdx > 0 {
					m.historyIdx--
				}
				m.output = m.history[m.historyIdx]
			}
			return m, nil
		}
		if zone.Get(m.id + "a_forward").InBounds(msg) {
			if m.historyIdx >= 0 {
				m.historyIdx++
				if m.historyIdx >= len(m.history) {
					m.historyIdx = -1
				} else {
					m.output = m.history[m.historyIdx]
				}
			}
			return m, nil
		}
		if zone.Get(m.id + "a_find_prev").InBounds(msg) {
			m.findNext(-1)
			return m, nil
		}
		if zone.Get(m.id + "a_find_next").InBounds(msg) {
			m.findNext(1)
			return m, nil
		}
		if zone.Get(m.id + "a_find_close").InBounds(msg) {
			m.findMode = false
			m.find.Blur()
			return m, nil
		}
		// Detail panel buttons — global actions (no template required).
		// Build button — handled outside !m.running so cancellation works.
		if zone.Get(m.id + "a_build").InBounds(msg) {
			if m.running {
				if m.cancel != nil {
					m.cancel()
				}
				m.running = false
				m.status = t("tui.status.build_cancelled")
				return m, nil
			}
			name := m.selectedName()
			if name == "" {
				return m, nil
			}
			m.running = true
			m.status = fmt.Sprintf(t("tui.status.building"), name)
			m.ctx, m.cancel = context.WithCancel(context.Background())
			dir := ResolveDistDir(m.distDir)
			args := []string{"./xbps-src", "pkg"}
			if m.buildQ {
				args = append(args, "-Q")
			}
			if m.buildC {
				args = append(args, "-C")
			}
			args = append(args, name)
			return m, xbpsCmd(m.ctx, "build", name, dir, args...)
		}
		if !m.running {
			if zone.Get(m.id + "a_bootstrap_update").InBounds(msg) {
				m.running = true
				m.status = t("tui.status.bootstrap")
				dir := ResolveDistDir(m.distDir)
				return m, xbpsCmd(context.Background(), "bootstrap-update", "", dir, "./xbps-src", "bootstrap-update")
			}
			// Template actions — require a selected template.
			name := m.selectedName()
			if name != "" {
				dir := ResolveDistDir(m.distDir)
				if zone.Get(m.id + "a_homepage").InBounds(msg) {
					meta := ReadMeta(m.distDir, name)
					if meta.Homepage != "" {
						OpenBrowser(meta.Homepage)
					}
					return m, nil
				}
				if zone.Get(m.id + "a_repology").InBounds(msg) {
					OpenBrowser("https://repology.org/projects/?search=" + name)
					return m, nil
				}
				if zone.Get(m.id + "a_lint").InBounds(msg) {
					m.running = true
					m.status = fmt.Sprintf(t("tui.status.linting"), name)
					return m, xbpsCmd(context.Background(), "lint", name, dir, "xlint", name)
				}
				if zone.Get(m.id + "a_sum").InBounds(msg) {
					m.running = true
					m.status = fmt.Sprintf(t("tui.status.checksum"), name)
					return m, xbpsCmd(context.Background(), "checksum", name, dir, "xgensum", "-i", name)
				}
				if zone.Get(m.id + "a_bump").InBounds(msg) {
					m.running = true
					m.status = fmt.Sprintf(t("tui.status.bumping"), name)
					return m, xbpsCmd(context.Background(), "bump", name, dir, "xxautobump", name)
				}
				if zone.Get(m.id + "a_install").InBounds(msg) {
					m.running = true
					m.status = fmt.Sprintf(t("tui.status.installing"), name)
					return m, xbpsCmd(context.Background(), "install", name, dir, "xi", name)
				}
				if zone.Get(m.id + "a_clean").InBounds(msg) {
					m.running = true
					m.status = fmt.Sprintf(t("tui.status.cleaning"), name)
					return m, xbpsCmd(context.Background(), "clean", name, dir, "./xbps-src", "clean", name)
				}
			}
		}
		// Check list items.
		list := m.filtered()
		listHeight := max(m.height-13, 4)
		start := 0
		if m.cursor >= listHeight {
			start = m.cursor - listHeight + 1
		}
		for i := start; i < len(list); i++ {
			if zone.Get(m.id + list[i].Name).InBounds(msg) {
				m.cursor = i
				m.editorMode = false
				m.editor.Blur()
				m.loadEditorFile(list[i].Name)
				return m, nil
			}
		}

	case xbpsReloadMsg:
		// Preserve per-session status across reloads.
		oldStatus := make(map[string]Template, len(m.templates))
		for _, t := range m.templates {
			oldStatus[t.Name] = t
		}
		fresh := LoadTemplates(m.distDir)
		for i, t := range fresh {
			if old, ok := oldStatus[t.Name]; ok {
				fresh[i] = old // restore status fields
			}
		}
		m.templates = fresh
		m = m.clampCursor()
		m.loadEditorFile(m.selectedName())
		return m, nil

	case xbpsDoneMsg:
		if m.output != "" {
			m.history = append(m.history, m.output)
		}
		m.historyIdx = -1
		m.running = false
		m.output = msg.output
		if msg.err != nil {
			m.status = fmt.Sprintf("%s failed: %s", msg.action, msg.err.Error())
			m.statusErr = true
		} else {
			m.status = fmt.Sprintf("%s OK", msg.action)
			m.statusErr = false
			if msg.template != "" {
				for i, tmpl := range m.templates {
					if tmpl.Name == msg.template {
						switch msg.action {
						case "build":
							m.templates[i].Built = true
						case "lint":
							m.templates[i].Linted = true
						case "checksum":
							m.templates[i].Checksummed = true
						case "bump":
							m.templates[i].Bumped = true
						case "install":
							m.templates[i].Installed = true
						}
						break
					}
				}
			}
		}
		return m, nil
	case tea.KeyPressMsg:
		if m.outputSelect {
			return m.handleOutputSelectKey(msg)
		}
		if m.searchMode {
			previous := m.selectedName()
			updated, cmd := m.handleSearchKey(msg)
			m = updated.(xbpsModel)
			if m.selectedName() != previous {
				m.loadEditorFile(m.selectedName())
			}
			return m, cmd
		}
		if m.editorMode {
			return m.handleEditorKey(msg)
		}
		if m.findMode {
			return m.handleFindKey(msg)
		}
		if m.settings {
			updated, cmd := m.handleSettingsKey(msg)
			m = updated.(xbpsModel)
			m.loadEditorFile(m.selectedName())
			return m, cmd
		}
		if m.showAbout {
			m.showAbout = false
			return m, nil
		}
		updated, cmd := m.handleKey(msg)
		m = updated.(xbpsModel)
		m.loadEditorFile(m.selectedName())
		return m, cmd
	}
	if m.searchMode {
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		m = m.clampCursor()
		m.loadEditorFile(m.selectedName())
		return m, cmd
	}
	return m, nil
}

func (m xbpsModel) handleSearchKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.Code {
	case tea.KeyEsc:
		m.searchMode = false
		m.search.SetValue("")
		m.search.Blur()
		m = m.clampCursor()
		return m, nil
	case tea.KeyEnter:
		m.searchMode = false
		m.search.Blur()
		m = m.clampCursor()
		return m, nil
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(key)
	m = m.clampCursor()
	return m, cmd
}

func (m xbpsModel) handleKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	list := m.filtered()

	switch key.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "?":
		m.showAbout = true
		return m, nil

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil

	case "down", "j":
		if m.cursor < len(list)-1 {
			m.cursor++
		}
		return m, nil

	case "/":
		m.searchMode = true
		m.search.Focus()
		return m, nil

	case "esc":
		if m.search.Value() != "" {
			m.search.SetValue("")
			m = m.clampCursor()
		}
		return m, nil

	case "r":
		return m, func() tea.Msg { return xbpsReloadMsg{} }
	case "f":
		m.findMode = true
		m.findIndex = -1
		m.find.Focus()
		return m, nil
	case "n":
		m.findNext(1)
		return m, nil
	case "N":
		m.findNext(-1)
		return m, nil
	case "v":
		m.startOutputSelection()
		return m, nil
	case "[":
		if len(m.history) > 0 {
			if m.historyIdx == -1 {
				m.historyIdx = len(m.history) - 1
			} else if m.historyIdx > 0 {
				m.historyIdx--
			}
			m.output = m.history[m.historyIdx]
		}
		return m, nil
	case "]":
		if m.historyIdx >= 0 {
			m.historyIdx++
			if m.historyIdx >= len(m.history) {
				m.historyIdx = -1
			} else {
				m.output = m.history[m.historyIdx]
			}
		}
		return m, nil
	case "S":
		m.cfg = LoadConfig()
		m.settingsFields = make([]textinput.Model, 3)
		vals := []string{m.cfg.DistDir, m.cfg.SearchEngine, m.cfg.ForkURL}
		for i := range vals {
			m.settingsFields[i] = textinput.New()
			m.settingsFields[i].SetValue(vals[i])
		}
		m.settings = true
		return m, nil

	case "1": // toggle -Q (tests)
		m.buildQ = !m.buildQ
		return m, nil

	case "2": // toggle -C (confpkg)
		m.buildC = !m.buildC
		return m, nil

	case "y": // copy output to clipboard
		if m.output != "" {
			if err := common.CopyToClipboard(m.output); err != nil {
				m.status = t("action.copy_err")
				m.statusErr = true
			} else {
				m.status = t("action.copied")
				m.statusErr = false
			}
			return m, nil
		}

	case "u": // bootstrap-update — no template required
		if m.running {
			return m, nil
		}
		m.running = true
		m.status = t("tui.status.bootstrap")
		dir := ResolveDistDir(m.distDir)
		return m, xbpsCmd(context.Background(), "bootstrap-update", "", dir, "./xbps-src", "bootstrap-update")
	}

	// Build key — handled before running check so cancellation works.
	if key.String() == "b" {
		if m.running {
			if m.cancel != nil {
				m.cancel()
			}
			m.running = false
			m.status = t("tui.status.build_cancelled")
			return m, nil
		}
		name := m.selectedName()
		if name == "" {
			return m, nil
		}
		m.running = true
		m.status = fmt.Sprintf(t("tui.status.building"), name)
		m.ctx, m.cancel = context.WithCancel(context.Background())
		dir := ResolveDistDir(m.distDir)
		args := []string{"./xbps-src", "pkg"}
		if m.buildQ {
			args = append(args, "-Q")
		}
		if m.buildC {
			args = append(args, "-C")
		}
		args = append(args, name)
		return m, xbpsCmd(m.ctx, "build", name, dir, args...)
	}

	// Template actions — require a selected template and no running command.
	if m.running {
		return m, nil
	}
	name := m.selectedName()
	if name == "" {
		return m, nil
	}
	dir := ResolveDistDir(m.distDir)

	switch key.String() {
	case "l":
		m.running = true
		m.status = fmt.Sprintf(t("tui.status.linting"), name)
		return m, xbpsCmd(context.Background(), "lint", name, dir, "xlint", name)

	case "s":
		m.running = true
		m.status = fmt.Sprintf(t("tui.status.checksum"), name)
		return m, xbpsCmd(context.Background(), "checksum", name, dir, "xgensum", "-i", name)

	case "a":
		m.running = true
		m.status = fmt.Sprintf(t("tui.status.bumping"), name)
		return m, xbpsCmd(context.Background(), "bump", name, dir, "xxautobump", name)

	case "i":
		m.running = true
		m.status = fmt.Sprintf(t("tui.status.installing"), name)
		return m, xbpsCmd(context.Background(), "install", name, dir, "xi", name)

	case "c":
		m.running = true
		m.status = fmt.Sprintf(t("tui.status.cleaning"), name)
		return m, xbpsCmd(context.Background(), "clean", name, dir, "./xbps-src", "clean", name)

	case "e":
		if m.editorPath != "" {
			m.editorMode = true
			return m, m.editor.Focus()
		}
		return m, nil

	case "h":
		meta := ReadMeta(m.distDir, name)
		if meta.Homepage != "" {
			OpenBrowser(meta.Homepage)
		}
		return m, nil

	case "p":
		OpenBrowser("https://repology.org/projects/?search=" + name)
		return m, nil
	}

	return m, nil
}

func (m xbpsModel) handleEditorKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if key.String() == "esc" {
		m.editorMode = false
		m.editor.Blur()
		return m, nil
	}
	if key.String() == "ctrl+s" {
		return m.saveEditor(), nil
	}
	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(key)
	return m, cmd
}

func (m xbpsModel) handleFindKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc", "enter":
		m.findMode = false
		m.find.Blur()
		return m, nil
	case "ctrl+n", "down":
		m.findNext(1)
		return m, nil
	case "ctrl+p", "up":
		m.findNext(-1)
		return m, nil
	}
	var cmd tea.Cmd
	m.find, cmd = m.find.Update(key)
	return m, cmd
}

func (m *xbpsModel) findNext(direction int) {
	query := m.find.Value()
	if query == "" || m.output == "" {
		return
	}
	positions := []int{}
	for at := 0; at < len(m.output); {
		i := strings.Index(strings.ToLower(m.output[at:]), strings.ToLower(query))
		if i < 0 {
			break
		}
		at += i
		positions = append(positions, at)
		at += len(query)
	}
	if len(positions) == 0 {
		return
	}
	if direction > 0 {
		m.findIndex = (m.findIndex + 1) % len(positions)
	} else {
		m.findIndex--
		if m.findIndex < 0 {
			m.findIndex = len(positions) - 1
		}
	}
}

func (m xbpsModel) handleSettingsKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.String() {
	case "esc":
		m.settings = false
		return m, nil
	case "tab", "down", "enter":
		if m.settingsIdx < len(m.settingsFields)-1 {
			m.settingsIdx++
		} else {
			m.cfg.DistDir = strings.TrimSpace(m.settingsFields[0].Value())
			m.cfg.SearchEngine = strings.TrimSpace(m.settingsFields[1].Value())
			m.cfg.ForkURL = strings.TrimSpace(m.settingsFields[2].Value())
			if err := SaveConfig(m.cfg); err != nil {
				m.status = t("status.save_err") + err.Error()
				m.statusErr = true
			} else {
				m.distDir = m.cfg.DistDir
				m.templates = LoadTemplates(m.distDir)
				m.status = t("status.save_ok")
				m.statusErr = false
			}
			m.settings = false
		}
		return m, nil
	case "shift+tab", "up":
		if m.settingsIdx > 0 {
			m.settingsIdx--
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.settingsFields[m.settingsIdx], cmd = m.settingsFields[m.settingsIdx].Update(key)
	return m, cmd
}
func (m *xbpsModel) startOutputSelection() {
	if m.output == "" {
		return
	}
	m.outputSelect = true
	m.outputMenu = false
	lines := m.outputTextLines()
	m.outputLine = len(lines) - 1
	for m.outputLine > 0 && len(strings.Fields(lines[m.outputLine])) == 0 {
		m.outputLine--
	}
	m.outputWord = 0
}

func (m xbpsModel) outputTextLines() []string { return strings.Split(m.output, "\n") }

func (m xbpsModel) selectedOutputWord() string {
	lines := m.outputTextLines()
	if m.outputLine < 0 || m.outputLine >= len(lines) {
		return ""
	}
	words := strings.Fields(lines[m.outputLine])
	if m.outputWord < 0 || m.outputWord >= len(words) {
		return ""
	}
	return strings.Trim(words[m.outputWord], "[](){}<>:,;\"'`")
}

func (m xbpsModel) handleOutputSelectKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.outputMenu {
		switch key.String() {
		case "esc":
			m.outputMenu = false
			return m, nil
		case "1":
			return m.outputDependency("hostmakedepends")
		case "2":
			return m.outputDependency("makedepends")
		case "3":
			return m.outputDependency("depends")
		case "4":
			return m.outputDependency("checkdepends")
		case "5":
			return m.outputCommand("xlocate")
		case "6":
			return m.outputCommand("xbps-query")
		case "7":
			OpenBrowser(m.cfg.SearchEngine + url.QueryEscape(m.selectedOutputWord()))
			m.outputMenu = false
			m.outputSelect = false
			return m, nil
		}
		return m, nil
	}
	switch key.String() {
	case "esc", "v":
		m.outputSelect = false
		return m, nil
	case "enter":
		if m.selectedOutputWord() != "" {
			m.outputMenu = true
		}
		return m, nil
	case "up", "k":
		if m.outputLine > 0 {
			m.outputLine--
			m.outputWord = 0
		}
	case "down", "j":
		if m.outputLine < len(m.outputTextLines())-1 {
			m.outputLine++
			m.outputWord = 0
		}
	case "left", "h":
		if m.outputWord > 0 {
			m.outputWord--
		}
	case "right", "l":
		words := strings.Fields(m.outputTextLines()[m.outputLine])
		if len(words) > 0 && m.outputWord < len(words)-1 {
			m.outputWord++
		}
	}
	return m, nil
}

func (m xbpsModel) outputDependency(field string) (tea.Model, tea.Cmd) {
	pkg := m.selectedOutputWord()
	name := m.selectedName()
	if pkg == "" {
		m.outputMenu = false
		return m, nil
	}
	if name == "" {
		m.status = t("status.no_template")
		m.statusErr = true
		m.outputMenu = false
		return m, nil
	}
	path := filepath.Join(ResolveDistDir(m.distDir), "srcpkgs", name, "template")
	data, err := os.ReadFile(path)
	if err != nil {
		m.status = t("status.read_err") + err.Error()
		m.statusErr = true
		m.outputMenu = false
		return m, nil
	}
	text := addDependency(string(data), pkg, field)
	if err = os.WriteFile(path, []byte(text), 0o644); err != nil {
		m.status = t("status.save_err") + err.Error()
		m.statusErr = true
		m.outputMenu = false
		return m, nil
	}
	m.editorPath = path
	m.editor.SetValue(text)
	m.status = fmt.Sprintf(t("status.added"), pkg, field)
	m.statusErr = false
	m.outputMenu = false
	m.outputSelect = false
	return m, nil
}

func addDependency(text, pkg, field string) string {
	re := regexp.MustCompile(`(?m)^(` + regexp.QuoteMeta(field) + `=")([^"]*)(")`)
	if re.MatchString(text) {
		return re.ReplaceAllStringFunc(text, func(s string) string {
			p := re.FindStringSubmatch(s)
			if len(p) != 4 {
				return s
			}
			old := strings.TrimSpace(p[2])
			if old == "" {
				return p[1] + pkg + p[3]
			}
			return p[1] + old + " " + pkg + p[3]
		})
	}
	fields := []string{"hostmakedepends=", "makedepends=", "depends=", "checkdepends="}
	lines := strings.Split(text, "\n")
	desc, last := -1, -1
	for i, line := range lines {
		if strings.HasPrefix(line, "short_desc=") {
			desc = i
		}
		for _, f := range fields {
			if strings.HasPrefix(line, f) {
				last = i
			}
		}
	}
	at := len(lines)
	if desc >= 0 {
		at = desc
	} else if last >= 0 {
		at = last + 1
	}
	lines = append(lines[:at], append([]string{field + `="` + pkg + `"`}, lines[at:]...)...)
	return strings.Join(lines, "\n")
}

func (m xbpsModel) outputCommand(which string) (tea.Model, tea.Cmd) {
	query := m.selectedOutputWord()
	m.outputSelect = false
	m.outputMenu = false
	if query == "" {
		return m, nil
	}
	if which == "xlocate" {
		return m, func() tea.Msg {
			out, err := RunXlocate(query)
			return xbpsDoneMsg{action: "xlocate", output: out, err: err}
		}
	}
	return m, func() tea.Msg {
		out, err := RunXbpsStream("", nil, "xbps-query", "-Rs", query)
		return xbpsDoneMsg{action: "xbps-query", output: out, err: err}
	}
}

// xbpsCmd returns a tea.Cmd that runs a command and emits xbpsDoneMsg.
func xbpsCmd(ctx context.Context, action, templateName, dir string, args ...string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		return xbpsDoneMsg{action: action, template: templateName, output: string(out), err: err}
	}
}

// bottomBar places controls at the left and the status at the right, like the GUI footer.
func (m xbpsModel) bottomBar() string {
	status := ""
	if m.status != "" {
		if m.statusErr {
			status = xDangerStyle.Render("  ✗ " + m.status)
		} else {
			status = xSuccessStyle.Render("  ✓ " + m.status)
		}
	}
	about := common.TBtn(m.id, "a_about", t("btn.about"), "?", false)
	reload := common.TBtn(m.id, "a_reload", t("btn.reload"), "r", false)
	find := common.TBtn(m.id, "a_find", t("btn.find"), "f", false)
	settings := common.TBtn(m.id, "a_settings", t("tui.settings.short"), "S", false)
	buttons := []string{settings, about, reload, find}
	if len(m.history) > 0 {
		buttons = append(buttons,
			common.TBtn(m.id, "a_back", t("tui.history.back"), "[", false),
			common.TBtn(m.id, "a_forward", t("tui.history.forward"), "]", false),
		)
	}
	if !common.TUIEmbedded() {
		buttons = append(buttons, common.TBtn(m.id, "a_quit", t("btn.quit"), "q", false))
	}
	return common.TUIFooter(common.TBtnBar(buttons...), status, m.width)
}

// detailButtons renders the actions aligned with the selected template details.
func (m xbpsModel) detailButtons() string {
	hasTemplate := m.selectedName() != ""
	buttons := []string{
		common.TBtn(m.id, "a_bootstrap_update", t("btn.bootstrap"), "u", false),
		common.TBtn(m.id, "a_homepage", t("btn.homepage"), "h", hasTemplate),
		common.TBtn(m.id, "a_repology", t("btn.repology"), "p", hasTemplate),
	}
	if hasTemplate {
		buildLabel := t("btn.build")
		if m.running {
			buildLabel = t("btn.stop")
		}
		buttons = append(buttons,
			"\n",
			common.TBtn(m.id, "a_build", buildLabel, "b", false),
			common.TBtn(m.id, "a_clean", t("btn.clean"), "c", false),
			common.TBtn(m.id, "a_install", t("btn.install"), "i", false),
		)
	}
	return common.TBtnBar(buttons...)
}

func (m xbpsModel) editorToolbar() string {
	return common.TBtnBar(
		common.TBtn(m.id, "a_lint", t("btn.lint"), "l", false),
		common.TBtn(m.id, "a_sum", t("btn.sum"), "s", false),
		common.TBtn(m.id, "a_bump", t("btn.bump"), "a", false),
		common.TBtn(m.id, "a_editor_save", t("btn.save"), "ctrl+s", false),
	)
}

// ── View ──────────────────────────────────────────────────────────────

// View implements tea.Model; renders the TUI in the alt screen.
func (m xbpsModel) View() tea.View { return m.view(true) }

// EmbeddedView returns the unscanned view for composition inside sysman-tui.
func (m xbpsModel) EmbeddedView() tea.View { return m.view(false) }

func (m xbpsModel) view(scan bool) tea.View {
	content := m.render()
	if scan {
		content = zone.Scan(content)
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m xbpsModel) renderOutputSelection() string {
	lines := m.outputTextLines()
	var b strings.Builder
	b.WriteString(xTitleStyle.Render(t("tui.output.select")) + "\n")
	if m.outputMenu {
		pkg := m.selectedOutputWord()
		b.WriteString(fmt.Sprintf(t("tui.output.selected"), pkg) + "\n")
		actions := []string{t("menu.add.hostmakedepends"), t("menu.add.makedepends"), t("menu.add.depends"), t("menu.add.checkdepends"), t("menu.xlocate"), t("menu.xbpsquery"), t("menu.websearch")}
		ids := []string{"a_outdep_0", "a_outdep_1", "a_outdep_2", "a_outdep_3", "a_out_xlocate", "a_out_query", "a_out_web"}
		for i, label := range actions {
			b.WriteString(zone.Mark(m.id+ids[i], common.TBtn(m.id, ids[i], label, fmt.Sprint(i+1), false)) + " ")
		}
		b.WriteString("\n" + xSubtleStyle.Render(t("tui.output.menu.help")))
		return b.String()
	}
	b.WriteString(xSubtleStyle.Render(t("tui.output.select.help")) + "\n")
	start := max(0, m.outputLine-5)
	end := min(len(lines), start+11)
	for i := start; i < end; i++ {
		prefix := "  "
		if i == m.outputLine {
			prefix = "› "
		}
		words := strings.Fields(lines[i])
		line := lines[i]
		if i == m.outputLine && m.outputWord >= 0 && m.outputWord < len(words) {
			line = strings.Replace(line, words[m.outputWord], xSelectedStyle.Render(words[m.outputWord]), 1)
		}
		b.WriteString(prefix + line + "\n")
	}
	b.WriteString("\n" + xSubtleStyle.Render(t("tui.output.current")+": "+m.selectedOutputWord()))
	return b.String()
}

func (m xbpsModel) render() string {
	if m.showAbout {
		return m.renderAbout()
	}
	if m.outputSelect {
		return m.renderOutputSelection()
	}
	if m.settings {
		var b strings.Builder
		b.WriteString(xTitleStyle.Render(t("tui.settings.title")) + "\n")
		labels := []string{t("tui.settings.distdir"), t("tui.settings.search"), t("tui.settings.fork")}
		for i, f := range m.settingsFields {
			b.WriteString(labels[i] + ": " + f.View() + "\n")
		}
		b.WriteString(common.TBtn(m.id, "a_settings_save", t("btn.save"), "enter", false) + "  " + common.TBtn(m.id, "a_settings_cancel", t("btn.close"), "esc", false))
		return b.String()
	}
	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height
	if h <= 0 {
		h = 24
	}

	// ── Workspace header ──────────────────────────────────────────────
	dir := ResolveDistDir(m.distDir)
	title := xTitleStyle.Render("xbps-src") + "  " + xSubtleStyle.Render(dir)
	if disk := DiskInfo(m.distDir); disk != "" {
		title += "  " + xWarnStyle.Render("💾 "+disk)
	}

	// Keep the search field with the list it filters.
	listTitle := t("tui.templates")
	if m.searchMode {
		listTitle += "  " + m.search.View()
	} else if m.search.Value() != "" {
		listTitle += "  " + xSubtleStyle.Render("/"+m.search.Value())
	}

	list := m.filtered()
	sideBySide := w >= 100
	contentHeight := max(h-8, 5)
	outputHeight := max(h/3-3, 2)
	detailHeight := max(contentHeight-outputHeight-2, 4)
	leftWidth := w - 4
	rightWidth := w - 4
	editorWidth := w - 4
	if sideBySide {
		editorWidth = max(w*26/100, 24)
		mainWidth := w - editorWidth - 2
		leftWidth = max(mainWidth*32/100, 22)
		rightWidth = max(mainWidth-leftWidth-3, 24)
	}

	// ── Template list pane ────────────────────────────────────────────
	var listBody strings.Builder
	listBody.WriteString(xTitleStyle.Render(listTitle) + "\n")
	listHeight := max(contentHeight-3, 1)
	start := 0
	if m.cursor >= listHeight {
		start = m.cursor - listHeight + 1
	}
	if start > 0 {
		listBody.WriteString(xSubtleStyle.Render(fmt.Sprintf("↑ %d", start)) + "\n")
	}
	shown := 0
	for i := start; i < len(list) && shown < listHeight; i++ {
		tmpl := list[i]
		nameWidth := max(leftWidth-11, 1)
		name := tmpl.Name
		if len(name) > nameWidth {
			name = name[:nameWidth-1] + "…"
		}
		line := fmt.Sprintf("%-*s %s", nameWidth, name, buildBadge(tmpl))
		if i == m.cursor {
			listBody.WriteString(zone.Mark(m.id+tmpl.Name, xSelectedStyle.Render(line)) + "\n")
		} else {
			listBody.WriteString(zone.Mark(m.id+tmpl.Name, xNormalStyle.Render(line)) + "\n")
		}
		shown++
	}
	if remaining := len(list) - (start + shown); remaining > 0 {
		listBody.WriteString(xSubtleStyle.Render(fmt.Sprintf("↓ %d", remaining)) + "\n")
	}
	if len(list) == 0 {
		listBody.WriteString(xSubtleStyle.Render(t("tui.none")) + "\n")
	}
	leftPane := lipgloss.NewStyle().
		Width(leftWidth).
		Height(contentHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(xSubtle).
		Padding(0, 1).
		Render(listBody.String())

	// ── Selected template detail/actions pane ─────────────────────────
	var detail strings.Builder
	detail.WriteString(xTitleStyle.Render(t("tui.details")) + "\n")
	if name := m.selectedName(); name != "" {
		meta := ReadMeta(m.distDir, name)
		detail.WriteString(xTitleStyle.Render(name))
		if meta.Version != "" {
			detail.WriteString("  " + xSubtleStyle.Render("v"+meta.Version))
		}
		detail.WriteString("\n")
		if meta.Desc != "" {
			detail.WriteString(xSubtleStyle.Render(meta.Desc) + "\n")
		}
	}
	if m.findMode || m.find.Value() != "" {
		detail.WriteString(t("btn.find") + ": " + m.find.View() + "  " +
			common.TBtn(m.id, "a_find_prev", t("tooltip.srcman.find_prev"), "N", false) + " " +
			common.TBtn(m.id, "a_find_next", t("tooltip.srcman.find_next"), "n", false) + "  " +
			common.TBtn(m.id, "a_find_close", t("tooltip.srcman.find_close"), "esc", false) + "\n")
	}
	var flags []string
	if m.buildQ {
		flags = append(flags, "-Q (tests)")
	}
	if m.buildC {
		flags = append(flags, "-C (confpkg)")
	}
	if len(flags) > 0 {
		detail.WriteString(xWarnStyle.Render(t("tui.build_flags")+": "+strings.Join(flags, "  ")) + "\n")
	}
	detail.WriteString(m.detailButtons())
	rightPane := lipgloss.NewStyle().
		Width(rightWidth).
		Height(detailHeight).
		MaxHeight(detailHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(xSubtle).
		Padding(0, 1).
		Render(detail.String())

	var outputBody strings.Builder
	if m.output != "" {
		outputBody.WriteString(common.TBtnBar(
			common.TBtn(m.id, "a_copy", t("btn.copy"), "y", false),
			common.TBtn(m.id, "a_output_select", t("tui.output.select.short"), "v", false),
		) + "\n")
	}
	outputBody.WriteString(xTitleStyle.Render(t("tui.command_output")) + "\n")
	if m.output != "" {
		lines := strings.Split(m.output, "\n")
		maxLines := max(outputHeight-5, 1)
		if len(lines) > maxLines {
			lines = lines[len(lines)-maxLines:]
		}
		for _, line := range lines {
			outputBody.WriteString(xSubtleStyle.Render(line) + "\n")
		}
	}
	outputPane := lipgloss.NewStyle().
		Width(rightWidth).
		Height(outputHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(xSubtle).
		Padding(0, 1).
		Render(outputBody.String())
	centerColumn := lipgloss.JoinVertical(lipgloss.Left, rightPane, outputPane)

	editor := m.editor
	editor.SetWidth(max(editorWidth-4, 1))
	editor.SetHeight(max(contentHeight-4, 1))
	editorContent := xTitleStyle.Render(t("editor.title")) + "\n" +
		m.editorToolbar() + "\n" +
		zone.Mark(m.id+"editor_area", editor.View())
	editorPane := lipgloss.NewStyle().
		Width(editorWidth).
		Height(contentHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(xSubtle).
		Padding(0, 1).
		Render(editorContent)

	var workspace string
	if sideBySide {
		workspace = lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " ", centerColumn, " ", editorPane)
	} else {
		workspace = lipgloss.JoinVertical(lipgloss.Left, leftPane, centerColumn, editorPane)
	}

	var sb strings.Builder
	sb.WriteString(title + "\n")
	sb.WriteString(workspace + "\n")

	sb.WriteString(m.bottomBar() + "\n")
	return sb.String()
}

// buildBadge returns a compact status string like "[B L S A I]" for the template.
func buildBadge(tmpl Template) string {
	badge := func(flag bool, letter string) string {
		if flag {
			return xSuccessStyle.Render(letter)
		}
		return xSubtleStyle.Render("·")
	}
	return badge(tmpl.Built, "B") +
		badge(tmpl.Linted, "L") +
		badge(tmpl.Checksummed, "S") +
		badge(tmpl.Bumped, "A") +
		badge(tmpl.Installed, "I")
}

// renderAbout renders the about screen.
func (m xbpsModel) renderAbout() string {
	title := xTitleStyle.Render(t("app.title") + " - " + t("app.subtitle"))
	version := xNormalStyle.Render(t("about.version") + ": " + serman.Version)
	author := xNormalStyle.Render(t("about.author") + ": " + serman.AppAuthor)
	license := xNormalStyle.Render(t("about.license") + ": " + serman.AppLicense)
	url := xNormalStyle.Render(serman.AppURL)
	hint := xSubtleStyle.Render(t("about.hint"))
	return "\n" + title + "\n\n" + version + "\n" + author + "\n" + license + "\n" + url + "\n\n" + hint + "\n"
}

// RunTUI runs the xbps plugin as a standalone Bubbletea application.
func RunTUI(distDir string) {
	common.EnsureZone()
	p := tea.NewProgram(NewTuiModel(distDir))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
