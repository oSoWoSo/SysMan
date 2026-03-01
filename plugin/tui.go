package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Styles ───────────────────────────────────────────────────────────

// Color palette — adaptive to light/dark terminal themes.
var (
	tsubtleColor = lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#585858"}
	thighlight   = lipgloss.AdaptiveColor{Light: "#00AABB", Dark: "#00DDFF"}
	tdanger      = lipgloss.AdaptiveColor{Light: "#CC3333", Dark: "#FF5555"}
	tsuccess     = lipgloss.AdaptiveColor{Light: "#22AA55", Dark: "#44DD77"}
	twarn        = lipgloss.AdaptiveColor{Light: "#BB8800", Dark: "#FFCC00"}
	// Component styles
	ttitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(thighlight).Padding(0, 1).MarginBottom(1)
	tsectionStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#444444", Dark: "#AAAAAA"})
	tselectedStyle = lipgloss.NewStyle().Bold(true).Foreground(thighlight).Background(lipgloss.AdaptiveColor{Light: "#DDFAFF", Dark: "#003344"}).Padding(0, 1)
	tnormalStyle   = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#333333", Dark: "#CCCCCC"}).Padding(0, 1)
	// Service status and feedback styles
	tenabledBadge  = lipgloss.NewStyle().Foreground(tsuccess).Bold(true)
	tdisabledBadge = lipgloss.NewStyle().Foreground(tsubtleColor)
	tstatusOk      = lipgloss.NewStyle().Foreground(tsuccess).Italic(true)
	tstatusErr     = lipgloss.NewStyle().Foreground(tdanger).Bold(true)
	thelpStyle     = lipgloss.NewStyle().Foreground(tsubtleColor)
	tdividerStyle  = lipgloss.NewStyle().Foreground(tsubtleColor)
	twarnStyle     = lipgloss.NewStyle().Foreground(twarn)
	tcolumnStyle   = lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.RoundedBorder()).BorderForeground(tsubtleColor)
	// Filters — active only when underlined + color
	tfilterActive   = lipgloss.NewStyle().Bold(true).Foreground(thighlight).Padding(0, 1).Underline(true)
	tfilterInactive = lipgloss.NewStyle().Foreground(tsubtleColor).Padding(0, 1)
)

// ── Filter ───────────────────────────────────────────────────────────

// tuiFilter represents the current filter state for the service list.
type tuiFilter int

const (
	tuiFilterAll      tuiFilter = iota // show all services
	tuiFilterEnabled                   // show only enabled services
	tuiFilterDisabled                  // show only disabled services
)

// label returns the translated label for the filter state.
func (f tuiFilter) label() string {
	switch f {
	case tuiFilterEnabled:
		return t("filter.enabled")
	case tuiFilterDisabled:
		return t("filter.disabled")
	default:
		return t("filter.all")
	}
}

// ── Model ────────────────────────────────────────────────────────────

// tuiModel holds the state of the TUI application.
type tuiModel struct {
	backend    Backend         // service manager backend (runit, openrc, …)
	services   []Service       // all loaded services
	cursor     int             // selected item index in filtered list
	filter     tuiFilter       // current filter (all/enabled/disabled)
	search     textinput.Model // search input field
	searchMode bool            // true when user is typing search query
	status     string          // status/error message
	statusErr  bool            // true if status is an error
	svStatus   ServiceStatus   // live runtime status of selected service
	svStatName string          // service name svStatus was fetched for
	width      int             // terminal width
	height     int             // terminal height
}

// Messages for async operations.
type tuiReloadMsg struct{}
type tuiErrMsg struct{ err error }
type tuiStatusMsg struct{ msg string }
type tuiSvStatusMsg struct {
	name   string
	status ServiceStatus
}
type tuiSvOpDoneMsg struct {
	name   string
	action string // key into status.* translations
}

// Key bindings for TUI navigation and actions.
var (
	tkeyUp       = key.NewBinding(key.WithKeys("up", "k"))
	tkeyDown     = key.NewBinding(key.WithKeys("down", "j"))
	tkeyToggle   = key.NewBinding(key.WithKeys("enter", " "))
	tkeyReload   = key.NewBinding(key.WithKeys("r"))
	tkeyQuit     = key.NewBinding(key.WithKeys("q", "ctrl+c", "esc"))
	tkeySearch   = key.NewBinding(key.WithKeys("/"))
	tkeyEsc      = key.NewBinding(key.WithKeys("esc"))
	tkeyEnter    = key.NewBinding(key.WithKeys("enter"))
	tkeyFilter   = key.NewBinding(key.WithKeys("tab"))
	tkeyStart    = key.NewBinding(key.WithKeys("s"))
	tkeyStop     = key.NewBinding(key.WithKeys("x"))
	tkeyRestart  = key.NewBinding(key.WithKeys("t"))
	tkeyHup      = key.NewBinding(key.WithKeys("l"))
	tkeyPause    = key.NewBinding(key.WithKeys("p"))
	tkeyContinue = key.NewBinding(key.WithKeys("c"))
	tkeyKill     = key.NewBinding(key.WithKeys("K"))
)

// NewTuiModel creates and initializes a new TUI model with services loaded.
// Exported so a system manager can embed the model in its own tea.Program.
func NewTuiModel(b Backend) tea.Model {
	ti := textinput.New()
	ti.Placeholder = t("search.placeholder")
	ti.CharLimit = 64
	ti.Width = 28
	ti.PromptStyle = lipgloss.NewStyle().Foreground(thighlight)
	ti.Prompt = "/ "
	return tuiModel{
		backend:  b,
		services: b.List(),
		search:   ti,
	}
}

// filtered returns the service list filtered by current filter and search query.
func (m tuiModel) filtered() []Service {
	var out []Service
	q := strings.ToLower(m.search.Value())
	for _, svc := range m.services {
		switch m.filter {
		case tuiFilterEnabled:
			if !svc.Enabled {
				continue
			}
		case tuiFilterDisabled:
			if svc.Enabled {
				continue
			}
		}
		if q != "" && !strings.Contains(strings.ToLower(svc.Name), q) {
			continue
		}
		out = append(out, svc)
	}
	return out
}

// clampCursor ensures the cursor position is valid for the current filtered list.
func (m tuiModel) clampCursor() tuiModel {
	list := m.filtered()
	if len(list) == 0 {
		m.cursor = 0
	} else if m.cursor >= len(list) {
		m.cursor = len(list) - 1
	}
	return m
}

// currentName returns the name of the currently selected service, or "".
func (m tuiModel) currentName() string {
	list := m.filtered()
	if m.cursor < 0 || m.cursor >= len(list) {
		return ""
	}
	return list[m.cursor].Name
}

// selectedEnabled returns the currently selected service if it is enabled, else nil.
func (m tuiModel) selectedEnabled() *Service {
	list := m.filtered()
	if m.cursor < 0 || m.cursor >= len(list) {
		return nil
	}
	svc := list[m.cursor]
	if !svc.Enabled {
		return nil
	}
	return &svc
}

// Init implements tea.Model; returns no initial command.
func (m tuiModel) Init() tea.Cmd { return nil }

// Update implements tea.Model; processes keyboard input, window resizes, and async messages.
func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyMsg:
		if m.searchMode {
			switch {
			case key.Matches(msg, tkeyEsc):
				m.search.SetValue("")
				m.search.Blur()
				m.searchMode = false
				m.cursor = 0
				m.svStatName = m.currentName()
				return m, m.fetchStatusCmd()
			case key.Matches(msg, tkeyEnter):
				m.search.Blur()
				m.searchMode = false
				m.cursor = 0
				m.svStatName = m.currentName()
				return m, m.fetchStatusCmd()
			case key.Matches(msg, tkeyUp):
				if m.cursor > 0 {
					m.cursor--
				}
			case key.Matches(msg, tkeyDown):
				list := m.filtered()
				if m.cursor < len(list)-1 {
					m.cursor++
				}
			default:
				var cmd tea.Cmd
				prev := m.search.Value()
				m.search, cmd = m.search.Update(msg)
				if m.search.Value() != prev {
					m.cursor = 0
				}
				return m, cmd
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, tkeyQuit):
			return m, tea.Quit
		case key.Matches(msg, tkeySearch):
			m.searchMode = true
			m.search.Focus()
			return m, textinput.Blink
		case key.Matches(msg, tkeyFilter):
			m.filter = (m.filter + 1) % 3
			m.cursor = 0
			m.svStatName = m.currentName()
			return m, m.fetchStatusCmd()
		case key.Matches(msg, tkeyUp):
			if m.cursor > 0 {
				m.cursor--
				m.svStatName = m.currentName()
				return m, m.fetchStatusCmd()
			}
		case key.Matches(msg, tkeyDown):
			list := m.filtered()
			if m.cursor < len(list)-1 {
				m.cursor++
				m.svStatName = m.currentName()
				return m, m.fetchStatusCmd()
			}
		case key.Matches(msg, tkeyToggle):
			list := m.filtered()
			if len(list) == 0 || m.cursor >= len(list) {
				break
			}
			svc := list[m.cursor]
			if svc.Enabled {
				return m, tuiBackendCmd(m.backend.Disable, svc.Name, "disabled")
			}
			return m, tuiBackendCmd(m.backend.Enable, svc.Name, "enabled")
		case key.Matches(msg, tkeyReload):
			return m, func() tea.Msg { return tuiReloadMsg{} }
		case key.Matches(msg, tkeyStart):
			if svc := m.selectedEnabled(); svc != nil {
				return m, tuiBackendCmd(m.backend.Start, svc.Name, "started")
			}
		case key.Matches(msg, tkeyStop):
			if svc := m.selectedEnabled(); svc != nil {
				return m, tuiBackendCmd(m.backend.Stop, svc.Name, "stopped")
			}
		case key.Matches(msg, tkeyRestart):
			if svc := m.selectedEnabled(); svc != nil {
				return m, tuiBackendCmd(m.backend.Restart, svc.Name, "restarted")
			}
		case key.Matches(msg, tkeyHup):
			if svc := m.selectedEnabled(); svc != nil {
				return m, tuiBackendCmd(m.backend.Reload, svc.Name, "hupped")
			}
		case key.Matches(msg, tkeyPause):
			if svc := m.selectedEnabled(); svc != nil {
				return m, tuiBackendCmd(m.backend.Pause, svc.Name, "paused")
			}
		case key.Matches(msg, tkeyContinue):
			if svc := m.selectedEnabled(); svc != nil {
				return m, tuiBackendCmd(m.backend.Continue, svc.Name, "continued")
			}
		case key.Matches(msg, tkeyKill):
			if svc := m.selectedEnabled(); svc != nil {
				return m, tuiBackendCmd(m.backend.Kill, svc.Name, "killed")
			}
		}

	case tuiReloadMsg:
		m.services = m.backend.List()
		m = m.clampCursor()
		return m, m.fetchStatusCmd()

	case tuiStatusMsg:
		m.status = msg.msg
		m.statusErr = false
		return m, func() tea.Msg { return tuiReloadMsg{} }

	case tuiSvStatusMsg:
		if msg.name == m.svStatName {
			m.svStatus = msg.status
		}

	case tuiSvOpDoneMsg:
		m.status = fmt.Sprintf(t("status."+msg.action), msg.name)
		m.statusErr = false
		return m, m.fetchStatusCmd()

	case tuiErrMsg:
		m.status = msg.err.Error()
		m.statusErr = true
	}
	return m, nil
}

// fetchStatusCmd returns a command that fetches sv status for the currently selected service.
// Returns nil if no enabled service is selected.
func (m tuiModel) fetchStatusCmd() tea.Cmd {
	list := m.filtered()
	if m.cursor < 0 || m.cursor >= len(list) {
		return nil
	}
	svc := list[m.cursor]
	if !svc.Enabled {
		return nil
	}
	name := svc.Name
	return func() tea.Msg {
		return tuiSvStatusMsg{name: name, status: m.backend.Status(name)}
	}
}

// tuiBackendCmd returns an async command that calls a backend method and emits the result.
// action must be a key suffix in the "status.*" translations (e.g. "enabled", "started").
func tuiBackendCmd(fn func(string) error, name, action string) tea.Cmd {
	return func() tea.Msg {
		if err := fn(name); err != nil {
			return tuiErrMsg{err}
		}
		return tuiSvOpDoneMsg{name: name, action: action}
	}
}

// View implements tea.Model; renders the entire TUI layout.
func (m tuiModel) View() string {
	// narrow: switch to single-column layout when terminal is too slim for two columns.
	narrow := m.width > 0 && m.width < 60

	list := m.filtered()
	enabledTotal := 0
	for _, s := range m.services {
		if s.Enabled {
			enabledTotal++
		}
	}

	// Separator width — fill the terminal, fallback to 70.
	sepWidth := 70
	if m.width > 0 {
		sepWidth = m.width - 2
		if sepWidth < 10 {
			sepWidth = 10
		}
	}

	// Column width: single column when narrow, two equal columns otherwise.
	colWidth := 36
	if m.width > 0 {
		if narrow {
			colWidth = m.width - 4
			if colWidth < 20 {
				colWidth = 20
			}
		} else {
			colWidth = (m.width - 8) / 2
			if colWidth < 24 {
				colWidth = 24
			}
		}
	}

	// Filter tabs — active one highlighted.
	filters := []tuiFilter{tuiFilterAll, tuiFilterEnabled, tuiFilterDisabled}
	filterRow := ""
	for _, f := range filters {
		if f == m.filter {
			filterRow += tfilterActive.Render(f.label())
		} else {
			filterRow += tfilterInactive.Render(f.label())
		}
		filterRow += " "
	}

	// Search — always 1 line.
	var searchRow string
	switch {
	case m.searchMode:
		searchRow = lipgloss.NewStyle().Foreground(thighlight).Render(m.search.View()) +
			thelpStyle.Render("  "+t("search.active"))
	case m.search.Value() != "":
		searchRow = thelpStyle.Render("/ "+m.search.Value()) +
			lipgloss.NewStyle().Foreground(tdanger).Render("  "+t("search.clear"))
	default:
		searchRow = thelpStyle.Render(t("search.hint"))
	}

	// Stats — enabled/total and filtered count.
	stats := tdisabledBadge.Render(fmt.Sprintf(t("stats.fmt"), enabledTotal, len(m.services), len(list)))

	// Height budget: blank(1)+title(1)+col-borders(2)+header-lines(5)+sep(1)+help(1)+trailing(2) ≈ 13.
	// Narrow adds 1 for compact-detail line.
	overhead := 13
	if narrow {
		overhead = 14
	}
	listHeight := 8
	if m.height > 0 {
		listHeight = m.height - overhead
		if listHeight < 3 {
			listHeight = 3
		}
	}

	// Scroll window — keep cursor visible.
	start := 0
	if m.cursor >= listHeight {
		start = m.cursor - listHeight + 1
	}

	// Service list with scroll indicators.
	var lsb strings.Builder
	if start > 0 {
		lsb.WriteString(thelpStyle.Render(fmt.Sprintf("  ↑ %d", start)) + "\n")
	}
	shown := 0
	for i := start; i < len(list) && shown < listHeight; i++ {
		svc := list[i]
		var badge string
		if svc.Enabled {
			badge = tenabledBadge.Render("[*]")
		} else {
			badge = tdisabledBadge.Render("[ ]")
		}
		line := fmt.Sprintf("%s %s", badge, svc.Name)
		if i == m.cursor {
			lsb.WriteString(tselectedStyle.Width(colWidth-4).Render(line) + "\n")
		} else {
			lsb.WriteString(tnormalStyle.Render(line) + "\n")
		}
		shown++
	}
	if remaining := len(list) - (start + shown); remaining > 0 {
		lsb.WriteString(thelpStyle.Render(fmt.Sprintf("  ↓ %d", remaining)) + "\n")
	}
	listContent := lsb.String()
	if listContent == "" {
		listContent = tnormalStyle.Render(t("services.none"))
	}

	svcDir, _ := m.backend.Dirs()
	leftHeader := tsectionStyle.Render(t("services.header")+svcDir) + "\n" +
		stats + "\n" +
		filterRow + "\n" +
		searchRow + "\n\n"
	leftCol := tcolumnStyle.Width(colWidth).Render(leftHeader + listContent)

	// Status line.
	statusLine := ""
	if m.status != "" {
		if m.statusErr {
			statusLine = "\n" + tstatusErr.Render(t("status.err")+m.status)
		} else {
			statusLine = "\n" + tstatusOk.Render(m.status)
		}
	}

	// Help bar — abbreviated on narrow terminals.
	var helpText string
	switch {
	case m.searchMode:
		helpText = t("help.search")
	case narrow:
		helpText = "↑↓=nav  enter=toggle  s/x/t=sv  /=search  q=quit"
	default:
		helpText = t("help.normal")
	}

	sep := tdividerStyle.Render(strings.Repeat("─", sepWidth))
	help := thelpStyle.Render(helpText)

	if narrow {
		// Single-column: compact 1-line detail below the list.
		title := ttitleStyle.Render(t("app.title"))
		compact := m.compactDetail(list)
		return "\n" + title + "\n" + leftCol + "\n" + compact + sep + statusLine + "\n" + help + "\n"
	}

	// Wide two-column layout.
	title := ttitleStyle.Render(t("app.title") + " - " + t("app.subtitle"))
	rightCol := tcolumnStyle.Width(colWidth).Render(m.buildDetail(list))
	cols := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, " ", rightCol)
	return "\n" + title + "\n" + cols + "\n" + sep + statusLine + "\n" + help + "\n"
}

// buildDetail renders the full detail panel for the right column (wide layout).
func (m tuiModel) buildDetail(list []Service) string {
	if len(list) == 0 || m.cursor >= len(list) {
		return ""
	}
	svc := list[m.cursor]
	var stateStr, actionStr string
	if svc.Enabled {
		stateStr = tenabledBadge.Render("[*] " + t("state.enabled"))
		actionStr = twarnStyle.Render(t("action.disable"))
	} else {
		stateStr = tdisabledBadge.Render("[ ] " + t("state.disabled"))
		actionStr = tenabledBadge.Render(t("action.enable"))
	}

	runningStr := ""
	if svc.Enabled && m.svStatName == svc.Name {
		st := m.svStatus
		if st.Running {
			runningStr = tenabledBadge.Render("▶ " + t("state.running"))
			if st.PID > 0 {
				runningStr += tnormalStyle.Render(fmt.Sprintf("  pid %d", st.PID))
			}
			if st.Uptime != "" {
				runningStr += tdisabledBadge.Render("  " + st.Uptime)
			}
		} else if st.Raw != "" {
			runningStr = tdisabledBadge.Render("■ " + t("state.stopped"))
		}
	}

	detail := tsectionStyle.Render(t("detail.header")) + "\n\n" +
		tnormalStyle.Render(t("detail.name")+":   "+svc.Name) + "\n" +
		tnormalStyle.Render(t("detail.state")+":    ") + stateStr + "\n"
	if runningStr != "" {
		detail += tnormalStyle.Render(t("detail.running")+": ") + runningStr + "\n"
	}
	svcDir2, destDir2 := m.backend.Dirs()
	detail += tnormalStyle.Render(t("detail.source")+":   "+filepath.Join(svcDir2, svc.Name)) + "\n" +
		tnormalStyle.Render(t("detail.symlink")+": "+filepath.Join(destDir2, svc.Name)) + "\n\n" +
		tnormalStyle.Render("action:  ") + actionStr
	if svc.Enabled {
		detail += "\n\n" + thelpStyle.Render("s=start  x=stop  t=restart  l=hup")
	}
	return detail
}

// compactDetail renders a concise 1-line summary for narrow (single-column) layout.
func (m tuiModel) compactDetail(list []Service) string {
	if len(list) == 0 || m.cursor >= len(list) {
		return ""
	}
	svc := list[m.cursor]
	var stateStr string
	if svc.Enabled {
		stateStr = tenabledBadge.Render("[*]")
	} else {
		stateStr = tdisabledBadge.Render("[ ]")
	}
	line := " " + tnormalStyle.Render(svc.Name) + " " + stateStr
	if svc.Enabled && m.svStatName == svc.Name {
		st := m.svStatus
		if st.Running {
			line += " " + tenabledBadge.Render("▶")
			if st.PID > 0 {
				line += tdisabledBadge.Render(fmt.Sprintf(" pid %d", st.PID))
			}
		} else if st.Raw != "" {
			line += " " + tdisabledBadge.Render("■")
		}
	}
	return line + "\n"
}

// ── Standalone runner ────────────────────────────────────────────────

// RunTUI runs svman as a standalone fullscreen TUI application.
func RunTUI(serviceDir, serviceDestDir string) {
	InitI18n()
	p := tea.NewProgram(NewTuiModel(NewRunitBackend(serviceDir, serviceDestDir)), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
