package srcman

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Styles ────────────────────────────────────────────────────────────

var (
	xSubtle    = lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#585858"}
	xHighlight = lipgloss.AdaptiveColor{Light: "#006688", Dark: "#00DDFF"}
	xDanger    = lipgloss.AdaptiveColor{Light: "#CC3333", Dark: "#FF5555"}
	xSuccess   = lipgloss.AdaptiveColor{Light: "#22AA55", Dark: "#44DD77"}
	xWarn      = lipgloss.AdaptiveColor{Light: "#BB8800", Dark: "#FFCC00"}

	xTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(xHighlight)
	xSelectedStyle = lipgloss.NewStyle().Bold(true).Foreground(xHighlight).
			Background(lipgloss.AdaptiveColor{Light: "#DDF5FF", Dark: "#003344"}).Padding(0, 1)
	xNormalStyle  = lipgloss.NewStyle().Padding(0, 1)
	xSubtleStyle  = lipgloss.NewStyle().Foreground(xSubtle)
	xDangerStyle  = lipgloss.NewStyle().Foreground(xDanger).Bold(true)
	xSuccessStyle = lipgloss.NewStyle().Foreground(xSuccess)
	xWarnStyle    = lipgloss.NewStyle().Foreground(xWarn)
	xHelpStyle    = lipgloss.NewStyle().Foreground(xSubtle)
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
	distDir    string
	templates  []Template
	cursor     int
	search     textinput.Model
	searchMode bool
	output     string // last command output
	status     string
	statusErr  bool
	running    bool
	width      int
	height     int
}

// NewTuiModel creates an initialized xbps TUI model.
func NewTuiModel(distDir string) tea.Model {
	ti := textinput.New()
	ti.Placeholder = t("tui.search.placeholder")
	m := xbpsModel{
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

	case tea.KeyMsg:
		if m.searchMode {
			return m.handleSearchKey(msg)
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

func (m xbpsModel) handleSearchKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
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

func (m xbpsModel) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	list := m.filtered()

	switch key.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

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

	case "u": // bootstrap-update — no template required
		if m.running {
			return m, nil
		}
		m.running = true
		m.status = t("tui.status.bootstrap")
		dir := ResolveDistDir(m.distDir)
		return m, xbpsCmd("bootstrap-update", "", dir, "./xbps-src", "bootstrap-update")
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
	case "b":
		m.running = true
		m.status = fmt.Sprintf(t("tui.status.building"), name)
		return m, xbpsCmd("build", name, dir, "./xbps-src", "pkg", name)

	case "l":
		m.running = true
		m.status = fmt.Sprintf(t("tui.status.linting"), name)
		return m, xbpsCmd("lint", name, dir, "xlint", name)

	case "s":
		m.running = true
		m.status = fmt.Sprintf(t("tui.status.checksum"), name)
		return m, xbpsCmd("checksum", name, dir, "xgensum", "-i", name)

	case "a":
		m.running = true
		m.status = fmt.Sprintf(t("tui.status.bumping"), name)
		return m, xbpsCmd("bump", name, dir, "xxautobump", name)

	case "i":
		m.running = true
		m.status = fmt.Sprintf(t("tui.status.installing"), name)
		return m, xbpsCmd("install", name, dir, "xi", name)

	case "c":
		m.running = true
		m.status = fmt.Sprintf(t("tui.status.cleaning"), name)
		return m, xbpsCmd("clean", name, dir, "./xbps-src", "clean", name)

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
func xbpsCmd(action, templateName, dir string, args ...string) tea.Cmd {
	return func() tea.Msg {
		out, err := RunXbps(dir, args...)
		return xbpsDoneMsg{action: action, template: templateName, output: out, err: err}
	}
}

// ── View ──────────────────────────────────────────────────────────────

func (m xbpsModel) View() string {
	w := m.width
	if w <= 0 {
		w = 60
	}
	narrow := w < 80
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
			sb.WriteString(xSelectedStyle.Render(line) + "\n")
		} else {
			sb.WriteString(xNormalStyle.Render(line) + "\n")
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

	// ── Selected template detail ──────────────────────────────────────
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
	sb.WriteString(divider + "\n")

	// ── Help ──────────────────────────────────────────────────────────
	switch {
	case m.running:
		sb.WriteString(xWarnStyle.Render("  "+t("tui.running")) + "\n")
	case narrow:
		sb.WriteString(xHelpStyle.Render("  "+t("tui.help.simple")) + "\n")
	default:
		sb.WriteString(xHelpStyle.Render("  "+t("tui.help.full")) + "\n")
	}

	// ── Command output (last lines) ───────────────────────────────────
	if m.output != "" {
		sb.WriteString(divider + "\n")
		lines := strings.Split(m.output, "\n")
		if len(lines) > 5 {
			lines = lines[len(lines)-5:]
		}
		for _, l := range lines {
			sb.WriteString(xSubtleStyle.Render("  "+l) + "\n")
		}
	}

	// ── Status bar ────────────────────────────────────────────────────
	sb.WriteString(divider + "\n")
	if m.status != "" {
		if m.statusErr {
			sb.WriteString(xDangerStyle.Render("  ✗ "+m.status) + "\n")
		} else {
			sb.WriteString(xSuccessStyle.Render("  ✓ "+m.status) + "\n")
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

// RunTUI runs the xbps plugin as a standalone Bubbletea application.
func RunTUI(distDir string) {
	p := tea.NewProgram(NewTuiModel(distDir), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
