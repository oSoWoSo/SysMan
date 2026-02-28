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
	serviceDir     string          // path to service definitions
	serviceDestDir string          // path to enabled services symlinks
	services       []Service       // all loaded services
	cursor         int             // selected item index in filtered list
	filter         tuiFilter       // current filter (all/enabled/disabled)
	search         textinput.Model // search input field
	searchMode     bool            // true when user is typing search query
	status         string          // status/error message
	statusErr      bool            // true if status is an error
	width          int             // terminal width
	height         int             // terminal height
}

// Messages for async operations.
type tuiReloadMsg struct{}
type tuiErrMsg struct{ err error }
type tuiStatusMsg struct{ msg string }

// Key bindings for TUI navigation and actions.
var (
	tkeyUp     = key.NewBinding(key.WithKeys("up", "k"))
	tkeyDown   = key.NewBinding(key.WithKeys("down", "j"))
	tkeyToggle = key.NewBinding(key.WithKeys("enter", " "))
	tkeyReload = key.NewBinding(key.WithKeys("r"))
	tkeyQuit   = key.NewBinding(key.WithKeys("q", "ctrl+c", "esc"))
	tkeySearch = key.NewBinding(key.WithKeys("/"))
	tkeyEsc    = key.NewBinding(key.WithKeys("esc"))
	tkeyEnter  = key.NewBinding(key.WithKeys("enter"))
	tkeyFilter = key.NewBinding(key.WithKeys("tab"))
)

// NewTuiModel creates and initializes a new TUI model with services loaded.
// Exported so a system manager can embed the model in its own tea.Program.
func NewTuiModel(serviceDir, serviceDestDir string) tea.Model {
	ti := textinput.New()
	ti.Placeholder = t("search.placeholder")
	ti.CharLimit = 64
	ti.Width = 28
	ti.PromptStyle = lipgloss.NewStyle().Foreground(thighlight)
	ti.Prompt = "/ "
	return tuiModel{
		serviceDir:     serviceDir,
		serviceDestDir: serviceDestDir,
		services:       LoadServices(serviceDir, serviceDestDir),
		search:         ti,
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
			case key.Matches(msg, tkeyEnter):
				m.search.Blur()
				m.searchMode = false
				m.cursor = 0
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
		case key.Matches(msg, tkeyUp):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, tkeyDown):
			list := m.filtered()
			if m.cursor < len(list)-1 {
				m.cursor++
			}
		case key.Matches(msg, tkeyToggle):
			list := m.filtered()
			if len(list) == 0 || m.cursor >= len(list) {
				break
			}
			svc := list[m.cursor]
			if svc.Enabled {
				return m, tuiDisableCmd(m.serviceDestDir, svc.Name)
			}
			return m, tuiEnableCmd(m.serviceDir, m.serviceDestDir, svc.Name)
		case key.Matches(msg, tkeyReload):
			return m, func() tea.Msg { return tuiReloadMsg{} }
		}

	case tuiReloadMsg:
		m.services = LoadServices(m.serviceDir, m.serviceDestDir)
		m = m.clampCursor()

	case tuiStatusMsg:
		m.status = msg.msg
		m.statusErr = false
		return m, func() tea.Msg { return tuiReloadMsg{} }

	case tuiErrMsg:
		m.status = msg.err.Error()
		m.statusErr = true
	}
	return m, nil
}

// tuiEnableCmd returns an async command that enables a service.
func tuiEnableCmd(serviceDir, destDir, name string) tea.Cmd {
	return func() tea.Msg {
		if err := EnableService(serviceDir, destDir, name); err != nil {
			return tuiErrMsg{err}
		}
		return tuiStatusMsg{fmt.Sprintf(t("status.enabled"), name)}
	}
}

// tuiDisableCmd returns an async command that disables a service.
func tuiDisableCmd(destDir, name string) tea.Cmd {
	return func() tea.Msg {
		if err := DisableService(destDir, name); err != nil {
			return tuiErrMsg{err}
		}
		return tuiStatusMsg{fmt.Sprintf(t("status.disabled"), name)}
	}
}

// View implements tea.Model; renders the entire TUI layout.
func (m tuiModel) View() string {
	title := ttitleStyle.Render(t("app.title") + " - " + t("app.subtitle"))

	list := m.filtered()
	enabledTotal := 0
	for _, s := range m.services {
		if s.Enabled {
			enabledTotal++
		}
	}

	// Calculate column width based on terminal width.
	colWidth := 36
	if m.width > 0 {
		colWidth = (m.width - 8) / 2
		if colWidth < 24 {
			colWidth = 24
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

	// Service list — each item with enabled/disabled badge.
	listContent := ""
	for i, svc := range list {
		var badge string
		if svc.Enabled {
			badge = tenabledBadge.Render("[*]")
		} else {
			badge = tdisabledBadge.Render("[ ]")
		}
		line := fmt.Sprintf("%s %s", badge, svc.Name)
		if i == m.cursor {
			listContent += tselectedStyle.Width(colWidth-4).Render(line) + "\n"
		} else {
			listContent += tnormalStyle.Render(line) + "\n"
		}
	}
	if listContent == "" {
		listContent = tnormalStyle.Render(t("services.none"))
	}

	leftHeader := tsectionStyle.Render(t("services.header")+m.serviceDir) + "\n" +
		stats + "\n" +
		filterRow + "\n" +
		searchRow + "\n\n"
	leftCol := tcolumnStyle.Width(colWidth).Render(leftHeader + listContent)

	// Detail panel — metadata and toggle action for selected item.
	detail := ""
	if len(list) > 0 && m.cursor < len(list) {
		svc := list[m.cursor]
		var stateStr, actionStr string
		if svc.Enabled {
			stateStr = tenabledBadge.Render("[*] " + t("state.enabled"))
			actionStr = twarnStyle.Render(t("action.disable"))
		} else {
			stateStr = tdisabledBadge.Render("[ ] " + t("state.disabled"))
			actionStr = tenabledBadge.Render(t("action.enable"))
		}
		detail = tsectionStyle.Render(t("detail.header")) + "\n\n" +
			tnormalStyle.Render(t("detail.name")+":   "+svc.Name) + "\n" +
			tnormalStyle.Render(t("detail.state")+":    ") + stateStr + "\n" +
			tnormalStyle.Render(t("detail.source")+":   "+filepath.Join(m.serviceDir, svc.Name)) + "\n" +
			tnormalStyle.Render(t("detail.symlink")+": "+filepath.Join(m.serviceDestDir, svc.Name)) + "\n\n" +
			tnormalStyle.Render("action:  ") + actionStr
	}
	rightCol := tcolumnStyle.Width(colWidth).Render(detail)

	cols := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, " ", rightCol)

	// Status line.
	statusLine := ""
	if m.status != "" {
		if m.statusErr {
			statusLine = "\n" + tstatusErr.Render(t("status.err")+m.status)
		} else {
			statusLine = "\n" + tstatusOk.Render(m.status)
		}
	}

	// Help bar.
	var helpText string
	if m.searchMode {
		helpText = t("help.search")
	} else {
		helpText = t("help.normal")
	}

	sep := tdividerStyle.Render(strings.Repeat("─", 70))
	help := thelpStyle.Render(helpText)

	return "\n" + title + "\n" + cols + "\n" + sep + statusLine + "\n" + help + "\n"
}

// ── Standalone runner ────────────────────────────────────────────────

// RunTUI runs svman as a standalone fullscreen TUI application.
func RunTUI(serviceDir, serviceDestDir string) {
	InitI18n()
	p := tea.NewProgram(NewTuiModel(serviceDir, serviceDestDir), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
	}
}
