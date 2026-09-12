package serman

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
	"codeberg.org/oSoWoSo/SysMan/src/common"
	zone "github.com/lrstanley/bubblezone/v2"
)

// ── Styles ───────────────────────────────────────────────────────────

// Color palette — adaptive to light/dark terminal themes.
var (
	tsubtleColor = compat.AdaptiveColor{Light: lipgloss.Color("#9B9B9B"), Dark: lipgloss.Color("#585858")}
	thighlight   = compat.AdaptiveColor{Light: lipgloss.Color("#00AABB"), Dark: lipgloss.Color("#00DDFF")}
	tdanger      = compat.AdaptiveColor{Light: lipgloss.Color("#CC3333"), Dark: lipgloss.Color("#FF5555")}
	tsuccess     = compat.AdaptiveColor{Light: lipgloss.Color("#22AA55"), Dark: lipgloss.Color("#44DD77")}
	// Component styles
	ttitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(thighlight).Padding(0, 1).MarginBottom(1)
	tsectionStyle  = lipgloss.NewStyle().Bold(true).Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#444444"), Dark: lipgloss.Color("#AAAAAA")})
	tselectedStyle = lipgloss.NewStyle().Bold(true).Foreground(thighlight).Background(compat.AdaptiveColor{Light: lipgloss.Color("#DDFAFF"), Dark: lipgloss.Color("#003344")}).Padding(0, 1)
	tnormalStyle   = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#333333"), Dark: lipgloss.Color("#CCCCCC")}).Padding(0, 1)
	// Service status and feedback styles
	tenabledBadge  = lipgloss.NewStyle().Foreground(tsuccess).Bold(true)
	tdisabledBadge = lipgloss.NewStyle().Foreground(tsubtleColor)
	tstatusOk      = lipgloss.NewStyle().Foreground(tsuccess).Italic(true)
	tstatusErr     = lipgloss.NewStyle().Foreground(tdanger).Bold(true)
	tdangerStyle   = lipgloss.NewStyle().Foreground(tdanger)
	thelpStyle     = lipgloss.NewStyle().Foreground(tsubtleColor)
	tdividerStyle  = lipgloss.NewStyle().Foreground(tsubtleColor)
	tcolumnStyle   = lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.RoundedBorder()).BorderForeground(tsubtleColor)
)

// ── Filter ───────────────────────────────────────────────────────────

// tuiFilter is an alias for FilterMode for the TUI.
type tuiFilter = FilterMode

// label returns the translated label for the filter state.
func (f tuiFilter) label() string {
	switch f {
	case FilterEnabled:
		return t("filter.enabled")
	case FilterDisabled:
		return t("filter.disabled")
	default:
		return t("filter.all")
	}
}

// ── Model ────────────────────────────────────────────────────────────

// tuiModel holds the state of the TUI application.
type tuiModel struct {
	id            string                   // bubblezone ID prefix for clickable elements
	backend       Backend                  // service manager backend (runit, openrc, …)
	services      []Service                // all loaded services
	cursor        int                      // selected item index in filtered list
	filter        tuiFilter                // current filter (all/enabled/disabled)
	scopeFilter   ScopeFilter              // current scope filter (all/system/user)
	search        textinput.Model          // search input field
	searchMode    bool                     // true when user is typing search query
	status        string                   // status/error message
	statusErr     bool                     // true if status is an error
	svStatus      ServiceStatus            // live runtime status of selected service
	svStatKey     string                   // service key svStatus was fetched for
	svStatusAll   map[string]ServiceStatus // batch status cache for all services (keyed by Service.Key())
	showAbout     bool                     // true when about screen is shown
	width         int                      // terminal width
	height        int                      // terminal height
	confirmAction string                   // pending confirmation action: "", "disable", "stop", "kill"
	confirmSvc    Service                  // service for the pending action
}

// Messages for async operations.
type tuiReloadMsg struct{}
type tuiErrMsg struct{ err error }
type tuiStatusMsg struct{ msg string }
type tuiSvOpDoneMsg struct {
	name   string
	action string // key into status.* translations
}
type tuiSvAllStatusMsg struct {
	statuses map[string]ServiceStatus
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
	tkeyScope    = key.NewBinding(key.WithKeys("z"))
	tkeyStart    = key.NewBinding(key.WithKeys("s"))
	tkeyStop     = key.NewBinding(key.WithKeys("x"))
	tkeyRestart  = key.NewBinding(key.WithKeys("t"))
	tkeyHup      = key.NewBinding(key.WithKeys("l"))
	tkeyPause    = key.NewBinding(key.WithKeys("p"))
	tkeyContinue = key.NewBinding(key.WithKeys("c"))
	tkeyKill     = key.NewBinding(key.WithKeys("K"))
	tkeyAbout    = key.NewBinding(key.WithKeys("?"))
)

// NewTuiModel creates and initializes a new TUI model with services loaded.
// Exported so a system manager can embed the model in its own tea.Program.
func NewTuiModel(b Backend) tea.Model {
	zone.NewGlobal()
	ti := textinput.New()
	ti.Placeholder = t("search.placeholder")
	ti.CharLimit = 64
	ti.SetWidth(28)
	st := textinput.DefaultStyles(compat.HasDarkBackground)
	st.Focused.Prompt = lipgloss.NewStyle().Foreground(thighlight)
	st.Blurred.Prompt = lipgloss.NewStyle().Foreground(thighlight)
	ti.SetStyles(st)
	ti.Prompt = "/ "
	return tuiModel{
		id:          zone.NewPrefix(),
		backend:     b,
		services:    b.List(),
		search:      ti,
		scopeFilter: DefaultScopeFilter(b.ScopeDirs()),
	}
}

// filtered returns the service list filtered by current filter and search query.
func (m tuiModel) filtered() []Service {
	return filterScoped(m.services, m.filter, m.search.Value(), m.scopeFilter)
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

// current returns the currently selected service, or nil.
func (m tuiModel) current() *Service {
	list := m.filtered()
	if m.cursor < 0 || m.cursor >= len(list) {
		return nil
	}
	svc := list[m.cursor]
	return &svc
}

// syncStatKey refreshes svStatKey from the current selection, tolerating an empty list.
func (m tuiModel) syncStatKey() tuiModel {
	if svc := m.current(); svc != nil {
		m.svStatKey = svc.Key()
	} else {
		m.svStatKey = ""
	}
	return m
}

// switchScope swaps the scope view between the system and user scopes.
// No status is fetched here — statuses require privileges and are only
// refreshed on an explicit reload or after a service change.
func (m tuiModel) switchScope() tuiModel {
	if m.scopeFilter.User {
		m.scopeFilter = ScopeFilter{System: true}
	} else {
		m.scopeFilter = ScopeFilter{User: true}
	}
	m.cursor = 0
	m.svStatusAll = nil
	m.svStatus = ServiceStatus{}
	m.svStatKey = ""
	return m.syncStatKey()
}

// selectedEnabled returns the currently selected service if it is enabled, else nil.
func (m tuiModel) selectedEnabled() *Service {
	svc := m.current()
	if svc == nil || !svc.Enabled {
		return nil
	}
	return svc
}

// zoneID returns the bubblezone ID for a service list row.
func (m tuiModel) zoneID(svc Service) string { return m.id + svc.Key() }

// activeZones returns the bubblezone ID prefix for list rows in the TUI.
func (m tuiModel) activeZones() string { return m.id }

// multiScope reports whether the given list spans more than one scope.
func (m tuiModel) multiScope(list []Service) bool {
	if len(list) < 2 {
		return false
	}
	scopes := make(map[Scope]bool)
	for _, svc := range list {
		scopes[svc.Scope] = true
	}
	return len(scopes) > 1
}

// leftDir returns a compact service-directory summary for the list header.
func (m tuiModel) leftDir() string {
	var dirs []string
	for _, sd := range m.backend.ScopeDirs() {
		dirs = append(dirs, sd.ServiceDir)
	}
	return strings.Join(dirs, " | ")
}

// Init implements tea.Model; returns no initial command.
func (m tuiModel) Init() tea.Cmd { return nil }

// handleSearchMode handles keyboard input when the search field is active.
func (m tuiModel) handleSearchMode(msg tea.KeyPressMsg) (tuiModel, tea.Cmd) {
	switch {
	case key.Matches(msg, tkeyEsc):
		m.search.SetValue("")
		m.search.Blur()
		m.searchMode = false
		m.cursor = 0
		m = m.syncStatKey()
		return m, nil
	case key.Matches(msg, tkeyEnter):
		m.search.Blur()
		m.searchMode = false
		m.cursor = 0
		m = m.syncStatKey()
		return m, nil
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

// Update implements tea.Model; processes keyboard input, window resizes, and async messages.
func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyPressMsg:
		if m.searchMode {
			return m.handleSearchMode(msg)
		}
		if m.showAbout {
			m.showAbout = false
			return m, nil
		}

		// Handle confirmation dialog
		if m.confirmAction != "" {
			if msg.String() == "y" {
				svc := m.confirmSvc
				action := m.confirmAction
				m.confirmAction = ""
				m.confirmSvc = Service{}
				switch action {
				case "disable":
					return m, tuiBackendCmd(m.backend.Disable, svc, "disabled")
				case "stop":
					return m, tuiBackendCmd(m.backend.Stop, svc, "stopped")
				case "kill":
					return m, tuiBackendCmd(m.backend.Kill, svc, "killed")
				}
			}
			m.confirmAction = ""
			m.confirmSvc = Service{}
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
			m = m.syncStatKey()
			return m, nil
		case key.Matches(msg, tkeyScope):
			m = m.switchScope()
			return m, nil
		case key.Matches(msg, tkeyUp):
			if m.cursor > 0 {
				m.cursor--
				m = m.syncStatKey()
				return m, nil
			}
		case key.Matches(msg, tkeyDown):
			list := m.filtered()
			if m.cursor < len(list)-1 {
				m.cursor++
				m = m.syncStatKey()
				return m, nil
			}
		case key.Matches(msg, tkeyToggle):
			list := m.filtered()
			if len(list) == 0 || m.cursor >= len(list) {
				break
			}
			svc := list[m.cursor]
			if svc.Enabled {
				m.confirmAction = "disable"
				m.confirmSvc = svc
				return m, nil
			}
			return m, tuiBackendCmd(m.backend.Enable, svc, "enabled")
		case key.Matches(msg, tkeyReload):
			return m, func() tea.Msg { return tuiReloadMsg{} }
		case key.Matches(msg, tkeyStart):
			if svc := m.selectedEnabled(); svc != nil {
				return m, tuiBackendCmd(m.backend.Start, *svc, "started")
			}
		case key.Matches(msg, tkeyStop):
			if svc := m.selectedEnabled(); svc != nil {
				m.confirmAction = "stop"
				m.confirmSvc = *svc
				return m, nil
			}
		case key.Matches(msg, tkeyRestart):
			if svc := m.selectedEnabled(); svc != nil {
				return m, tuiBackendCmd(m.backend.Restart, *svc, "restarted")
			}
		case key.Matches(msg, tkeyHup):
			if svc := m.selectedEnabled(); svc != nil {
				return m, tuiBackendCmd(m.backend.Reload, *svc, "hupped")
			}
		case key.Matches(msg, tkeyPause):
			if svc := m.selectedEnabled(); svc != nil {
				return m, tuiBackendCmd(m.backend.Pause, *svc, "paused")
			}
		case key.Matches(msg, tkeyContinue):
			if svc := m.selectedEnabled(); svc != nil {
				return m, tuiBackendCmd(m.backend.Continue, *svc, "continued")
			}
		case key.Matches(msg, tkeyKill):
			if svc := m.selectedEnabled(); svc != nil {
				m.confirmAction = "kill"
				m.confirmSvc = *svc
				return m, nil
			}
		case key.Matches(msg, tkeyAbout):
			m.showAbout = true
			return m, nil
		}

	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			break
		}
		// Check top bar buttons (filter tabs).
		for _, f := range []tuiFilter{FilterAll, FilterEnabled, FilterDisabled} {
			if zone.Get(m.id + "f_" + f.label()).InBounds(msg) {
				m.filter = f
				m.cursor = 0
				m = m.syncStatKey()
				return m, nil
			}
		}
		// Check top bar scope toggle button (system ↔ user).
		if zone.Get(m.id + "s_scope").InBounds(msg) {
			m = m.switchScope()
			return m, nil
		}
		// Check detail panel buttons (action buttons).
		if zone.Get(m.id + "a_quit").InBounds(msg) {
			return m, tea.Quit
		}
		if zone.Get(m.id + "a_about").InBounds(msg) {
			m.showAbout = true
			return m, nil
		}
		if zone.Get(m.id + "a_reload").InBounds(msg) {
			return m, func() tea.Msg { return tuiReloadMsg{} }
		}
		if zone.Get(m.id + "a_enable").InBounds(msg) {
			list := m.filtered()
			if len(list) > 0 && m.cursor < len(list) {
				svc := list[m.cursor]
				if !svc.Enabled {
					return m, tuiBackendCmd(m.backend.Enable, svc, "enabled")
				}
			}
		}
		if zone.Get(m.id + "a_disable").InBounds(msg) {
			list := m.filtered()
			if len(list) > 0 && m.cursor < len(list) {
				svc := list[m.cursor]
				if svc.Enabled {
					m.confirmAction = "disable"
					m.confirmSvc = svc
					return m, nil
				}
			}
		}
		if zone.Get(m.id + "a_start").InBounds(msg) {
			if svc := m.selectedEnabled(); svc != nil {
				return m, tuiBackendCmd(m.backend.Start, *svc, "started")
			}
		}
		if zone.Get(m.id + "a_stop").InBounds(msg) {
			if svc := m.selectedEnabled(); svc != nil {
				m.confirmAction = "stop"
				m.confirmSvc = *svc
				return m, nil
			}
		}
		if zone.Get(m.id + "a_restart").InBounds(msg) {
			if svc := m.selectedEnabled(); svc != nil {
				return m, tuiBackendCmd(m.backend.Restart, *svc, "restarted")
			}
		}
		if zone.Get(m.id + "a_hup").InBounds(msg) {
			if svc := m.selectedEnabled(); svc != nil {
				return m, tuiBackendCmd(m.backend.Reload, *svc, "hupped")
			}
		}
		if zone.Get(m.id + "a_pause").InBounds(msg) {
			if svc := m.selectedEnabled(); svc != nil {
				return m, tuiBackendCmd(m.backend.Pause, *svc, "paused")
			}
		}
		if zone.Get(m.id + "a_continue").InBounds(msg) {
			if svc := m.selectedEnabled(); svc != nil {
				return m, tuiBackendCmd(m.backend.Continue, *svc, "continued")
			}
		}
		if zone.Get(m.id + "a_kill").InBounds(msg) {
			if svc := m.selectedEnabled(); svc != nil {
				m.confirmAction = "kill"
				m.confirmSvc = *svc
				return m, nil
			}
		}
		// Check list items.
		list := m.filtered()
		active := m.activeZones()
		start := m.scrollStart()
		for i := start; i < len(list); i++ {
			key := list[i].Key()
			if zone.Get(active + key).InBounds(msg) {
				m.cursor = i
				m = m.syncStatKey()
				return m, nil
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

	case tuiSvAllStatusMsg:
		m.svStatusAll = msg.statuses
		// Also update selected service status
		if svc := m.current(); svc != nil {
			if st, ok := msg.statuses[svc.Key()]; ok {
				m.svStatus = st
				m.svStatKey = svc.Key()
			}
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

// fetchStatusCmd returns a command that fetches sv status for all enabled
// services matching the active scope filter in one call per scope. A
// user-only scope filter never triggers elevated status queries.
func (m tuiModel) fetchStatusCmd() tea.Cmd {
	b := m.backend
	var enabled []Service
	for _, svc := range m.services {
		if svc.Enabled && m.scopeFilter.matches(svc.Scope) {
			enabled = append(enabled, svc)
		}
	}
	if len(enabled) == 0 {
		return nil
	}
	return func() tea.Msg {
		return tuiSvAllStatusMsg{statuses: b.StatusAll(enabled)}
	}
}

// tuiBackendCmd returns an async command that calls a backend method and emits the result.
// action must be a key suffix in the "status.*" translations (e.g. "enabled", "started").
func tuiBackendCmd(fn func(Service) error, svc Service, action string) tea.Cmd {
	return func() tea.Msg {
		if err := fn(svc); err != nil {
			return tuiErrMsg{err}
		}
		return tuiSvOpDoneMsg{name: svc.Name, action: action}
	}
}

// listOverhead returns the number of non-list lines consumed by the layout.
func (m tuiModel) listOverhead() int {
	overhead := 15 // blank(1)+title(1)+topbar(1)+col-borders(2)+header(4)+sep(1)+status(1)+bottombar(1)+trailing(2)
	if m.width > 0 && m.width < 60 {
		overhead = 16 // +1 for compact-detail line
	}
	return overhead
}

// scrollStart returns the first visible item index for the current cursor and terminal height.
func (m tuiModel) scrollStart() int {
	listHeight := 8
	if m.height > 0 {
		listHeight = m.height - m.listOverhead()
		if listHeight < 3 {
			listHeight = 3
		}
	}
	start := 0
	if m.cursor >= listHeight {
		start = m.cursor - listHeight + 1
	}
	return start
}

// topBar renders the filter button bar at the top of the TUI.
func (m tuiModel) topBar() string {
	btns := []string{
		common.TBtn(m.id, "f_all", t("filter.all"), "tab", m.filter == FilterAll),
		common.TBtn(m.id, "f_enabled", t("filter.enabled"), "tab", m.filter == FilterEnabled),
		common.TBtn(m.id, "f_disabled", t("filter.disabled"), "tab", m.filter == FilterDisabled),
	}
	if len(m.backend.ScopeDirs()) > 1 {
		btns = append(btns,
			common.TBtn(m.id, "s_scope", t("filter.scope_user"), "z", m.scopeFilter.User),
		)
	}
	return common.TBtnBar(btns...)
}

// bottomBar renders the status bar at the bottom of the TUI (matching GUI layout).
func (m tuiModel) bottomBar() string {
	if m.confirmAction != "" {
		var prompt string
		switch m.confirmAction {
		case "disable":
			prompt = tdangerStyle.Render("  ⚠ " + fmt.Sprintf(t("confirm.disable"), m.confirmSvc.Name) + " [y/N]")
		case "stop":
			prompt = tdangerStyle.Render("  ⚠ " + fmt.Sprintf(t("confirm.stop"), m.confirmSvc.Name) + " [y/N]")
		case "kill":
			prompt = tdangerStyle.Render("  ⚠ " + fmt.Sprintf(t("confirm.kill"), m.confirmSvc.Name) + " [y/N]")
		}
		return prompt + "\n  " + thelpStyle.Render(t("confirm.hint"))
	}
	// Status message
	statusText := ""
	if m.status != "" {
		if m.statusErr {
			statusText = tstatusErr.Render(t("status.err") + m.status)
		} else {
			statusText = tstatusOk.Render(m.status)
		}
	}
	// Quit and About buttons on the right
	aboutBtn := common.TBtn(m.id, "a_about", t("btn.about"), "?", false)
	quitBtn := common.TBtn(m.id, "a_quit", t("action.quit"), "q", false)
	return statusText + "  " + aboutBtn + " " + quitBtn
}

// detailButtons renders the action buttons inside the right panel (matching GUI layout).
func (m tuiModel) detailButtons() string {
	svc := m.selectedEnabled()
	var toggleRow, controlRow string

	// Toggle row: Enable / Disable
	toggleRow = common.TBtnBar(
		common.TBtn(m.id, "a_enable", t("btn.enable"), "enter", false),
		common.TBtn(m.id, "a_disable", t("btn.disable"), "enter", false),
	)

	// Control row: Start Stop Restart HUP Pause Continue Kill | Reload
	if svc != nil {
		controlRow = common.TBtnBar(
			common.TBtn(m.id, "a_start", t("btn.start"), "s", false),
			common.TBtn(m.id, "a_stop", t("btn.stop"), "x", false),
			common.TBtn(m.id, "a_restart", t("btn.restart"), "t", false),
			common.TBtn(m.id, "a_hup", t("btn.hup"), "l", false),
			common.TBtn(m.id, "a_pause", t("btn.pause"), "p", false),
			common.TBtn(m.id, "a_continue", t("btn.continue"), "c", false),
			common.TBtn(m.id, "a_kill", t("btn.kill"), "K", false),
			common.TBtn(m.id, "a_reload", t("btn.reload"), "r", false),
		)
	} else {
		controlRow = thelpStyle.Render("  " + t("help.select_service"))
	}

	return toggleRow + "\n" + controlRow
}

// View implements tea.Model; renders the entire TUI layout in the alt screen.
func (m tuiModel) View() tea.View {
	v := tea.NewView(zone.Scan(m.render()))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m tuiModel) render() string {
	if m.showAbout {
		return m.renderAbout()
	}
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

	// Top button bar — filter tabs.
	topBar := m.topBar()

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

	// Height budget: see listOverhead() for the formula.
	overhead := m.listOverhead()
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

	// Service list with scroll indicators and scope headers.
	var lsb strings.Builder
	if start > 0 {
		lsb.WriteString(thelpStyle.Render(fmt.Sprintf("  ↑ %d", start)) + "\n")
	}
	multi := m.multiScope(list)
	last := Scope("")
	shown := 0
	for i := start; i < len(list) && shown < listHeight; i++ {
		svc := list[i]
		if multi && svc.Scope != last {
			lsb.WriteString(tsectionStyle.Render("  "+t("group."+string(svc.Scope))) + "\n")
			last = svc.Scope
		}
		var badge string
		if svc.Enabled {
			badge = tenabledBadge.Render("[*]")
		} else {
			badge = tdisabledBadge.Render("[ ]")
		}
		line := fmt.Sprintf("%s %s", badge, svc.Name)
		if i == m.cursor {
			lsb.WriteString(zone.Mark(m.zoneID(svc), tselectedStyle.Width(colWidth-4).Render(line)) + "\n")
		} else {
			lsb.WriteString(zone.Mark(m.zoneID(svc), tnormalStyle.Render(line)) + "\n")
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

	svcDir := m.leftDir()
	leftHeader := tsectionStyle.Render(t("services.header")+svcDir) + "\n" +
		stats + "\n" +
		searchRow + "\n\n"
	leftCol := tcolumnStyle.Width(colWidth).Render(leftHeader + listContent)

	sep := tdividerStyle.Render(strings.Repeat("─", sepWidth))
	help := m.bottomBar()

	if narrow {
		// Single-column: compact 1-line detail below the list.
		title := ttitleStyle.Render(t("app.title"))
		compact := m.compactDetail(list)
		return "\n" + topBar + "\n" + title + "\n" + leftCol + "\n" + compact + "\n" + sep + "\n" + help + "\n"
	}

	// Wide two-column layout.
	title := ttitleStyle.Render(t("app.title") + " - " + t("app.subtitle"))
	rightCol := tcolumnStyle.Width(colWidth).Render(m.buildDetail(list))
	cols := lipgloss.JoinHorizontal(lipgloss.Top, leftCol, " ", rightCol)
	return "\n" + topBar + "\n" + title + "\n" + cols + "\n" + sep + "\n" + help + "\n"
}

// buildDetail renders the full detail panel for the right column (wide layout).
// Matches the GUI: detail form + action buttons + config section.
func (m tuiModel) buildDetail(list []Service) string {
	if len(list) == 0 || m.cursor >= len(list) {
		return thelpStyle.Render("  " + t("help.select_service"))
	}
	svc := list[m.cursor]
	var stateStr string
	if svc.Enabled {
		stateStr = tenabledBadge.Render("[*] " + t("state.enabled"))
	} else {
		stateStr = tdisabledBadge.Render("[ ] " + t("state.disabled"))
	}

	runningStr := ""
	if svc.Enabled {
		var st ServiceStatus
		if m.svStatusAll != nil {
			st = m.svStatusAll[svc.Key()]
		} else if m.svStatKey == svc.Key() {
			st = m.svStatus
		}
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

	// Detail form (matching GUI layout)
	detail := tsectionStyle.Render(t("detail.header")) + "\n\n" +
		tnormalStyle.Render(t("detail.name")+":   "+svc.Name) + "\n" +
		tnormalStyle.Render(t("detail.state")+":    ") + stateStr + "\n"
	if runningStr != "" {
		detail += tnormalStyle.Render(t("detail.running")+": ") + runningStr + "\n"
	}
	svcDir2, destDir2 := m.backend.Dirs(svc)
	detail += tnormalStyle.Render(t("detail.scope")+":   "+t("scope."+string(svc.Scope))) + "\n" +
		tnormalStyle.Render(t("detail.source")+":   "+filepath.Join(svcDir2, svc.Name)) + "\n" +
		tnormalStyle.Render(t("detail.symlink")+": "+filepath.Join(destDir2, svc.Name)) + "\n"

	// Separator
	detail += "\n" + tdividerStyle.Render(strings.Repeat("─", 30)) + "\n"

	// Action buttons (matching GUI: toggle row + control row)
	detail += "\n" + m.detailButtons() + "\n"

	// Separator
	detail += "\n" + tdividerStyle.Render(strings.Repeat("─", 30)) + "\n"

	// Config section (matching GUI)
	detail += "\n" + tsectionStyle.Render(t("config.title")) + "\n"
	for _, sd := range m.backend.ScopeDirs() {
		if sd.Scope == ScopeUser {
			detail += tnormalStyle.Render("USER_SERVICEDIR="+sd.ServiceDir) + "\n" +
				tnormalStyle.Render("USER_SERVICEDESTDIR="+sd.DestDir) + "\n"
		} else {
			detail += tnormalStyle.Render("SERVICEDIR="+sd.ServiceDir) + "\n" +
				tnormalStyle.Render("SERVICEDESTDIR="+sd.DestDir) + "\n"
		}
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
	if svc.Enabled {
		var st ServiceStatus
		if m.svStatusAll != nil {
			st = m.svStatusAll[svc.Key()]
		} else if m.svStatKey == svc.Key() {
			st = m.svStatus
		}
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

// renderAbout renders the about screen.
func (m tuiModel) renderAbout() string {
	title := ttitleStyle.Render(t("app.title") + " - " + t("app.subtitle"))
	version := tnormalStyle.Render(t("about.version") + ": " + Version)
	author := tnormalStyle.Render(t("about.author") + ": " + AppAuthor)
	license := tnormalStyle.Render(t("about.license") + ": " + AppLicense)
	url := tnormalStyle.Render(AppURL)
	hint := thelpStyle.Render(t("about.hint"))
	return "\n" + title + "\n\n" + version + "\n" + author + "\n" + license + "\n" + url + "\n\n" + hint + "\n"
}

// ── Standalone runner ────────────────────────────────────────────────

// RunTUI runs svman as a standalone fullscreen TUI application.
func RunTUI(system, user ScopeDir) {
	InitI18n()
	common.EnsureZone()
	p := tea.NewProgram(NewTuiModel(NewScopedRunitBackend(system, user)))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
