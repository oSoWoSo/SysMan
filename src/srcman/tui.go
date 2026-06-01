package srcman

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

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
	xSelectedStyle = lipgloss.NewStyle().Bold(true).Foreground(xHighlight).
			Background(compat.AdaptiveColor{Light: lipgloss.Color("#DDF5FF"), Dark: lipgloss.Color("#003344")}).Padding(0, 1)
	xNormalStyle  = lipgloss.NewStyle().Padding(0, 1)
	xSubtleStyle  = lipgloss.NewStyle().Foreground(xSubtle)
	xDangerStyle  = lipgloss.NewStyle().Foreground(xDanger).Bold(true)
	xSuccessStyle = lipgloss.NewStyle().Foreground(xSuccess)
	xWarnStyle    = lipgloss.NewStyle().Foreground(xWarn)
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

// xbpsModel is the Bubbletea model for the xbps-src plugin.
type xbpsModel struct {
	id         string
	distDir    string
	templates  []Template
	cursor     int
	search     textinput.Model
	searchMode bool
	output     string // last command output
	status     string
	statusErr  bool
	running    bool
	showAbout  bool
	width      int
	height     int
	buildQ     bool // build with tests (-Q flag)
	buildC     bool // build with confpkg (-C flag)
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewTuiModel creates an initialized xbps TUI model.
func NewTuiModel(distDir string) tea.Model {
	zone.NewGlobal()
	ti := textinput.New()
	ti.Placeholder = t("tui.search.placeholder")
	m := xbpsModel{
		id:      zone.NewPrefix(),
		distDir: distDir,
		search:  ti,
	}
	m.templates = LoadTemplates(distDir)
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

// ── tea.Model ─────────────────────────────────────────────────────────

func (m xbpsModel) Init() tea.Cmd { return nil }

func (m xbpsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			break
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
		if zone.Get(m.id + "a_reload").InBounds(msg) {
			return m, func() tea.Msg { return xbpsReloadMsg{} }
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
				if zone.Get(m.id + "a_edit").InBounds(msg) {
					OpenEditor(m.distDir, name)
					return m, nil
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
		return m, nil

	case xbpsDoneMsg:
		m.running = false
		m.output = msg.output
		if msg.err != nil {
			m.status = fmt.Sprintf("%s failed: %s", msg.action, msg.err.Error())
			m.statusErr = true
		} else {
			m.status = fmt.Sprintf("%s OK", msg.action)
			m.statusErr = false
			// Mark template status on success.
			if msg.template != "" {
				for i, t := range m.templates {
					if t.Name == msg.template {
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
		if m.searchMode {
			return m.handleSearchKey(msg)
		}
		if m.showAbout {
			m.showAbout = false
			return m, nil
		}
		return m.handleKey(msg)
	}

	if m.searchMode {
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
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
		OpenEditor(m.distDir, name)
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

// xbpsCmd returns a tea.Cmd that runs a command and emits xbpsDoneMsg.
func xbpsCmd(ctx context.Context, action, templateName, dir string, args ...string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		return xbpsDoneMsg{action: action, template: templateName, output: string(out), err: err}
	}
}

// bottomBar renders a status bar at the very bottom: status message + quit button.
func (m xbpsModel) bottomBar() string {
	status := ""
	if m.status != "" {
		if m.statusErr {
			status = xDangerStyle.Render("  ✗ " + m.status)
		} else {
			status = xSuccessStyle.Render("  ✓ " + m.status)
		}
	}
	quit := common.TBtn(m.id, "a_quit", t("btn.quit"), "q", false)
	about := common.TBtn(m.id, "a_about", t("btn.about"), "?", false)
	reload := common.TBtn(m.id, "a_reload", t("btn.reload"), "r", false)
	var copyBtn string
	if m.output != "" {
		copyBtn = " " + common.TBtn(m.id, "a_copy", t("btn.copy"), "y", false)
	}
	return status + "  " + copyBtn + " " + reload + " " + about + " " + quit
}

// detailButtons renders the action buttons inside the detail panel (right side),
// below the template info. Shows global actions always, template actions only when a template is selected.
func (m xbpsModel) detailButtons() string {
	hasTemplate := m.selectedName() != ""
	var btns []string

	// Global actions (always visible)
	btns = append(btns,
		common.TBtn(m.id, "a_bootstrap_update", t("btn.bootstrap"), "u", false),
		common.TBtn(m.id, "a_homepage", t("btn.homepage"), "h", hasTemplate),
		common.TBtn(m.id, "a_repology", t("btn.repology"), "p", hasTemplate),
	)

	// Template actions (only if a template is selected)
	if hasTemplate {
		btns = append(btns, "\n")
		buildLabel := t("btn.build")
		if m.running {
			buildLabel = t("btn.stop")
		}
		btns = append(btns,
			common.TBtn(m.id, "a_build", buildLabel, "b", false),
			common.TBtn(m.id, "a_lint", t("btn.lint"), "l", false),
			common.TBtn(m.id, "a_sum", t("btn.sum"), "s", false),
			common.TBtn(m.id, "a_bump", t("btn.bump"), "a", false),
			common.TBtn(m.id, "a_install", t("btn.install"), "i", false),
			common.TBtn(m.id, "a_clean", t("btn.clean"), "c", false),
			common.TBtn(m.id, "a_edit", t("btn.edit"), "e", false),
		)
	}

	return common.TBtnBar(btns...)
}

// ── View ──────────────────────────────────────────────────────────────

// View implements tea.Model; renders the TUI in the alt screen.
func (m xbpsModel) View() tea.View {
	v := tea.NewView(zone.Scan(m.render()))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m xbpsModel) render() string {
	if m.showAbout {
		return m.renderAbout()
	}
	w := m.width
	if w <= 0 {
		w = 60
	}
	divider := xSubtleStyle.Render(strings.Repeat("─", w))

	// ── Title ─────────────────────────────────────────────────────────
	dir := ResolveDistDir(m.distDir)
	title := xTitleStyle.Render("xbps-src") + "  " + xSubtleStyle.Render(dir)
	if disk := DiskInfo(m.distDir); disk != "" {
		title += "  " + xWarnStyle.Render("💾 "+disk)
	}
	if m.searchMode {
		title += "  " + m.search.View()
	} else if m.search.Value() != "" {
		title += "  " + xSubtleStyle.Render("/"+m.search.Value())
	}

	var sb strings.Builder
	sb.WriteString(title + "\n")
	sb.WriteString(divider + "\n")

	// ── Template list ─────────────────────────────────────────────────
	list := m.filtered()
	// Reserve lines: divider×3 + help + status + detail(3) + output(max 5)
	listHeight := max(m.height-13, 4)

	// Scroll window: keep cursor visible.
	start := 0
	if m.cursor >= listHeight {
		start = m.cursor - listHeight + 1
	}

	if start > 0 {
		sb.WriteString(xSubtleStyle.Render(fmt.Sprintf("  ↑ %d", start)) + "\n")
	}
	shown := 0
	for i := start; i < len(list) && shown < listHeight; i++ {
		tmpl := list[i]
		// Status badges (B=built L=linted S=sum A=bumped I=installed)
		badge := buildBadge(tmpl)
		line := fmt.Sprintf("%-28s %s", tmpl.Name, badge)
		if i == m.cursor {
			sb.WriteString(zone.Mark(m.id+tmpl.Name, xSelectedStyle.Render(line)) + "\n")
		} else {
			sb.WriteString(zone.Mark(m.id+tmpl.Name, xNormalStyle.Render(line)) + "\n")
		}
		shown++
	}
	if remaining := len(list) - (start + shown); remaining > 0 {
		sb.WriteString(xSubtleStyle.Render(fmt.Sprintf("  ↓ %d", remaining)) + "\n")
	}
	if len(list) == 0 {
		sb.WriteString(xSubtleStyle.Render("  "+t("tui.none")) + "\n")
	}
	sb.WriteString(divider + "\n")

	if name := m.selectedName(); name != "" {
		meta := ReadMeta(m.distDir, name)
		detail := xTitleStyle.Render(name)
		if meta.Version != "" {
			detail += "  " + xSubtleStyle.Render("v"+meta.Version)
		}
		sb.WriteString("  " + detail + "\n")
		if meta.Desc != "" {
			sb.WriteString("  " + xSubtleStyle.Render(meta.Desc) + "\n")
		}
	}
	// Build flags display
	var flags []string
	if m.buildQ {
		flags = append(flags, "-Q (tests)")
	}
	if m.buildC {
		flags = append(flags, "-C (confpkg)")
	}
	if len(flags) > 0 {
		sb.WriteString("  " + xWarnStyle.Render(t("tui.build_flags")+": "+strings.Join(flags, "  ")) + "\n")
	}
	// Action buttons inside the detail panel (global + template)
	sb.WriteString(m.detailButtons() + "\n")
	sb.WriteString(divider + "\n")

	// ── Status bar (bottom) ──────────────────────────────────────
	sb.WriteString(m.bottomBar() + "\n")

	// ── Command output (last lines) ───────────────────────────────────
	if m.output != "" {
		sb.WriteString(divider + "\n")
		outputLines := max(m.height/3, 5)
		lines := strings.Split(m.output, "\n")
		if len(lines) > outputLines {
			lines = lines[len(lines)-outputLines:]
		}
		for _, l := range lines {
			sb.WriteString(xSubtleStyle.Render("  "+l) + "\n")
		}
	}

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
