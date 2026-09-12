package vmman

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
	"codeberg.org/oSoWoSo/SysMan/src/common"
	zone "github.com/lrstanley/bubblezone/v2"
)

var (
	tsubtleColor = compat.AdaptiveColor{Light: lipgloss.Color("#9B9B9B"), Dark: lipgloss.Color("#585858")}
	thighlight   = compat.AdaptiveColor{Light: lipgloss.Color("#00AABB"), Dark: lipgloss.Color("#00DDFF")}
	tdanger      = compat.AdaptiveColor{Light: lipgloss.Color("#CC3333"), Dark: lipgloss.Color("#FF5555")}
	tGreen       = compat.AdaptiveColor{Light: lipgloss.Color("#22AA55"), Dark: lipgloss.Color("#44DD77")}

	ttitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(thighlight).Padding(0, 1).MarginBottom(1)
	tsectionStyle  = lipgloss.NewStyle().Bold(true).Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#444444"), Dark: lipgloss.Color("#AAAAAA")})
	tselectedStyle = lipgloss.NewStyle().Bold(true).Foreground(thighlight).Background(compat.AdaptiveColor{Light: lipgloss.Color("#DDFAFF"), Dark: lipgloss.Color("#003344")}).Padding(0, 1)
	tnormalStyle   = lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#333333"), Dark: lipgloss.Color("#CCCCCC")}).Padding(0, 1)

	trunningBadge = lipgloss.NewStyle().Foreground(tGreen).Bold(true)
	tstoppedBadge = lipgloss.NewStyle().Foreground(tdanger).Bold(true)
	tstatusOk     = lipgloss.NewStyle().Foreground(tGreen).Italic(true)
	tstatusErr    = lipgloss.NewStyle().Foreground(tdanger).Bold(true)
	tdangerStyle  = lipgloss.NewStyle().Foreground(tdanger)
	thelpStyle    = lipgloss.NewStyle().Foreground(tsubtleColor)
	tdividerStyle = lipgloss.NewStyle().Foreground(tsubtleColor)
	tcolumnStyle  = lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.RoundedBorder()).BorderForeground(tsubtleColor)
)

type vmmanFilter = FilterMode

type tuiModel struct {
	id            string
	backend       Backend
	vms           []VM
	cursor        int
	filter        vmmanFilter
	search        textinput.Model
	searchMode    bool
	status        string
	statusErr     bool
	showAbout     bool
	width         int
	height        int
	confirmAction string // "", "kill"
	confirmArg    string // VM name

	// create-VM prompts
	createMode   bool
	createStep   int // 0 = name, 1 = guest os, 2 = ssh user
	createName   string
	createGuestOS string
	createInput  textinput.Model

	// log screen
	logMode bool
	logBuf  string
	logSink *tuiLogBuffer
}

// tuiLogBuffer accumulates log lines written from stream goroutines.
type tuiLogBuffer struct {
	mu  sync.Mutex
	buf strings.Builder
}

// Add appends a line to the buffer (safe from any goroutine).
func (b *tuiLogBuffer) Add(line string) {
	b.mu.Lock()
	b.buf.WriteString(line)
	b.mu.Unlock()
}

// Drain returns and clears everything currently buffered.
func (b *tuiLogBuffer) Drain() string {
	b.mu.Lock()
	s := b.buf.String()
	b.buf.Reset()
	b.mu.Unlock()
	return s
}

const maxTuiLogBytes = 512 * 1024

type tuiVMMsg struct{ vms []VM }
type tuiErrMsg struct{ err error }
type tuiStatusMsg struct{ msg string }
type tuiTickMsg struct{}

var (
	tkeyUp      = key.NewBinding(key.WithKeys("up", "k"))
	tkeyDown    = key.NewBinding(key.WithKeys("down", "j"))
	tkeyConnect = key.NewBinding(key.WithKeys("c"))
	tkeyBoot    = key.NewBinding(key.WithKeys("b"))
	tkeyKill    = key.NewBinding(key.WithKeys("K"))
	tkeyNew     = key.NewBinding(key.WithKeys("n"))
	tkeyEdit    = key.NewBinding(key.WithKeys("e"))
	tkeyLog     = key.NewBinding(key.WithKeys("l"))
	tkeyQuit    = key.NewBinding(key.WithKeys("q", "ctrl+c", "esc"))
	tkeySearch  = key.NewBinding(key.WithKeys("/"))
	tkeyEsc     = key.NewBinding(key.WithKeys("esc"))
	tkeyEnter   = key.NewBinding(key.WithKeys("enter"))
	tkeyFilter  = key.NewBinding(key.WithKeys("tab"))
	tkeyReload  = key.NewBinding(key.WithKeys("r"))
	tkeyAbout   = key.NewBinding(key.WithKeys("?"))
)

// NewTuiModel creates a new TUI model.
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

	ci := textinput.New()
	ci.CharLimit = 64
	ci.SetWidth(28)
	ci.SetStyles(st)
	ci.Prompt = t("new.name") + " > "

	// Check if VM directory exists and report any errors
	status := ""
	statusErr := false
	if errMsg := CheckVMDir(b.VMDir()); errMsg != "" {
		status = errMsg
		statusErr = true
	}

	return tuiModel{
		id:          zone.NewPrefix(),
		backend:     b,
		vms:         b.List(),
		search:      ti,
		createInput: ci,
		logSink:     &tuiLogBuffer{},
		status:      status,
		statusErr:   statusErr,
	}
}

func (m tuiModel) filtered() []VM {
	return Filter(m.vms, m.filter, m.search.Value(),
		func(vm VM) bool { return vm.Running },
		func(vm VM, q string) bool { return strings.Contains(strings.ToLower(vm.Name), q) },
	)
}

func (m tuiModel) clampCursor() tuiModel {
	list := m.filtered()
	if len(list) == 0 {
		m.cursor = 0
	} else if m.cursor >= len(list) {
		m.cursor = len(list) - 1
	}
	return m
}

func (m tuiModel) selectedVM() *VM {
	list := m.filtered()
	if m.cursor < 0 || m.cursor >= len(list) {
		return nil
	}
	return &list[m.cursor]
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyPressMsg:
		if m.logMode {
			m.logBuf = appendLog(m.logBuf, m.logSink.Drain())
			switch msg.String() {
			case "esc", "q":
				m.logMode = false
				return m, nil
			case "ctrl+c":
				return m, tea.Quit
			default:
				return m, nil
			}
		}

		if m.createMode {
			return m.updateCreate(msg)
		}

		if m.searchMode {
			switch {
			case key.Matches(msg, tkeyEsc):
				m.search.SetValue("")
				m.search.Blur()
				m.searchMode = false
				m.cursor = 0
				return m, nil
			case key.Matches(msg, tkeyEnter):
				m.search.Blur()
				m.cursor = 0
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

		if m.showAbout {
			m.showAbout = false
			return m, nil
		}

		// Handle confirmation dialog
		if m.confirmAction != "" {
			if msg.String() == "y" {
				vm := m.selectedVM()
				if vm != nil {
					action := m.confirmAction
					m.confirmAction = ""
					m.confirmArg = ""
					if action == "kill" {
						return m, vmmanBackendCmd(m.backend, vm, "kill")
					}
				}
			}
			m.confirmAction = ""
			m.confirmArg = ""
			return m, nil
		}

		switch {
		case key.Matches(msg, tkeyQuit):
			return m, tea.Quit
		case key.Matches(msg, tkeySearch):
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
		case key.Matches(msg, tkeyConnect):
			vm := m.selectedVM()
			if vm != nil && vm.Running && (vm.SPICEPort > 0 || vm.SSHPort > 0) {
				return m, connectCmd(vm)
			}
		case key.Matches(msg, tkeyBoot):
			vm := m.selectedVM()
			if vm != nil && !vm.Running {
				return m, m.bootLogCmd(vm)
			}
		case key.Matches(msg, tkeyKill):
			vm := m.selectedVM()
			if vm != nil && vm.Running {
				m.confirmAction = "kill"
				m.confirmArg = vm.Name
				return m, nil
			}
		case key.Matches(msg, tkeyNew):
			m.createMode = true
			m.createStep = 0
			m.createName = ""
			m.createInput.SetValue("")
			m.createInput.Prompt = t("new.name") + " > "
			m.createInput.Placeholder = t("new.name_ph")
			m.createInput.Focus()
			return m, textinput.Blink
		case key.Matches(msg, tkeyEdit):
			vm := m.selectedVM()
			if vm != nil {
				OpenEditor(m.backend.VMDir(), vm.Name)
				m.status = fmt.Sprintf(t("status.saved"), vm.Name)
				m.statusErr = false
				return m, nil
			}
		case key.Matches(msg, tkeyLog):
			vm := m.selectedVM()
			if vm != nil {
				m.logBuf = readVMLog(m.backend.VMDir(), vm.Name)
				m.logMode = true
				return m, m.logTick()
			}
		case key.Matches(msg, tkeyReload):
			return m, func() tea.Msg {
				return tuiVMMsg{vms: m.backend.List()}
			}
		case key.Matches(msg, tkeyAbout):
			m.showAbout = true
			return m, nil
		}

	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			break
		}
		// Check filter tabs.
		for _, f := range []vmmanFilter{FilterAll, FilterRunning, FilterStopped} {
			var zoneKey string
			switch f {
			case FilterRunning:
				zoneKey = "f_running"
			case FilterStopped:
				zoneKey = "f_stopped"
			default:
				zoneKey = "f_all"
			}
			if zone.Get(m.id + zoneKey).InBounds(msg) {
				m.filter = f
				m.cursor = 0
				return m, nil
			}
		}
		// Check bottom bar buttons (actions).
		if zone.Get(m.id + "a_boot").InBounds(msg) {
			vm := m.selectedVM()
			if vm != nil && !vm.Running {
				return m, m.bootLogCmd(vm)
			}
		}
		if zone.Get(m.id + "a_new").InBounds(msg) {
			m.createMode = true
			m.createStep = 0
			m.createName = ""
			m.createInput.SetValue("")
			m.createInput.Prompt = t("new.name") + " > "
			m.createInput.Placeholder = t("new.name_ph")
			m.createInput.Focus()
			return m, textinput.Blink
		}
		if zone.Get(m.id + "a_edit").InBounds(msg) {
			vm := m.selectedVM()
			if vm != nil {
				OpenEditor(m.backend.VMDir(), vm.Name)
				m.status = fmt.Sprintf(t("status.saved"), vm.Name)
				m.statusErr = false
				return m, nil
			}
		}
		if zone.Get(m.id + "a_log").InBounds(msg) {
			vm := m.selectedVM()
			if vm != nil {
				m.logBuf = readVMLog(m.backend.VMDir(), vm.Name)
				m.logMode = true
				return m, m.logTick()
			}
		}
		if zone.Get(m.id + "a_kill").InBounds(msg) {
			vm := m.selectedVM()
			if vm != nil && vm.Running {
				m.confirmAction = "kill"
				m.confirmArg = vm.Name
				return m, nil
			}
		}
		if zone.Get(m.id + "a_connect").InBounds(msg) {
			vm := m.selectedVM()
			if vm != nil && vm.Running && (vm.SPICEPort > 0 || vm.SSHPort > 0) {
				return m, connectCmd(vm)
			}
		}
		if zone.Get(m.id + "a_reload").InBounds(msg) {
			return m, func() tea.Msg {
				return tuiVMMsg{vms: m.backend.List()}
			}
		}
		if zone.Get(m.id + "a_quit").InBounds(msg) {
			return m, tea.Quit
		}
		if zone.Get(m.id + "a_about").InBounds(msg) {
			m.showAbout = true
			return m, nil
		}
		// Check list items.
		list := m.filtered()
		start := m.scrollStart()
		for i := start; i < len(list); i++ {
			if zone.Get(m.id + list[i].Name).InBounds(msg) {
				m.cursor = i
				return m, nil
			}
		}

	case tuiVMMsg:
		m.vms = msg.vms
		m = m.clampCursor()
	case tuiErrMsg:
		m.status = msg.err.Error()
		m.statusErr = true
	case tuiStatusMsg:
		m.status = msg.msg
		m.statusErr = false
		return m, func() tea.Msg { return tuiVMMsg{vms: m.backend.List()} }
	case tuiTickMsg:
		m.logBuf = appendLog(m.logBuf, m.logSink.Drain())
		if m.logMode {
			return m, m.logTick()
		}
		return m, nil
	}

	return m, nil
}

// View implements tea.Model; renders the TUI in the alt screen.
func (m tuiModel) View() tea.View {
	v := tea.NewView(zone.Scan(m.render()))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// listOverhead returns the number of non-list lines consumed by the layout.
func (m tuiModel) listOverhead() int {
	overhead := 14 // blank(1)+title(1)+topbar(1)+col-borders(2)+header(4)+sep(1)+bottombar(1)+trailing(2)
	if m.width > 0 && m.width < 60 {
		overhead = 15 // +1 for compact-detail line
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
	return common.TBtnBar(
		common.TBtn(m.id, "f_all", t("filter.all"), "tab", m.filter == FilterAll),
		common.TBtn(m.id, "f_running", t("filter.running"), "tab", m.filter == FilterRunning),
		common.TBtn(m.id, "f_stopped", t("filter.stopped"), "tab", m.filter == FilterStopped),
	)
}

// bottomBar renders the status bar at the bottom of the TUI: status message + quit.
func (m tuiModel) bottomBar() string {
	if m.confirmAction != "" {
		var prompt string
		switch m.confirmAction {
		case "kill":
			prompt = tdangerStyle.Render("  ⚠ " + fmt.Sprintf(t("confirm.kill"), m.confirmArg) + " [y/N]")
		}
		return prompt + "\n  " + thelpStyle.Render(t("confirm.hint"))
	}
	left := ""
	if m.status != "" {
		if m.statusErr {
			left = tstatusErr.Render(m.status)
		} else {
			left = tstatusOk.Render(m.status)
		}
	}
	quit := common.TBtn(m.id, "a_quit", t("action.quit"), "q", false)
	about := common.TBtn(m.id, "a_about", t("btn.about"), "?", false)
	if m.width > 0 {
		gap := m.width - lipgloss.Width(left) - lipgloss.Width(quit) - lipgloss.Width(about) - 6
		if gap > 0 {
			return left + strings.Repeat(" ", gap) + about + " " + quit
		}
	}
	return left + "  " + about + " " + quit
}

// detailButtons renders action buttons inside the right panel.
func (m tuiModel) detailButtons() string {
	vm := m.selectedVM()
	buttons := []string{
		common.TBtn(m.id, "a_new", t("btn.new"), "n", false),
	}
	if vm != nil {
		buttons = append(buttons, common.TBtn(m.id, "a_edit", t("btn.edit"), "e", false))
		buttons = append(buttons, common.TBtn(m.id, "a_log", t("btn.log"), "l", false))
	}
	if vm != nil && !vm.Running {
		buttons = append(buttons, common.TBtn(m.id, "a_boot", t("btn.boot"), "b", false))
	}
	if vm != nil && vm.Running {
		buttons = append(buttons, common.TBtn(m.id, "a_kill", t("btn.kill"), "K", false))
	}
	if vm != nil && vm.Running && (vm.SPICEPort > 0 || vm.SSHPort > 0) {
		buttons = append(buttons, common.TBtn(m.id, "a_connect", t("btn.connect"), "c", false))
	}
	buttons = append(buttons, common.TBtn(m.id, "a_reload", t("btn.reload"), "r", false))
	return common.TBtnBar(buttons...)
}

func (m tuiModel) render() string {
	if m.showAbout {
		return m.renderAbout()
	}
	if m.logMode {
		return m.renderLog()
	}
	if m.createMode {
		return m.renderCreate()
	}
	narrow := m.width > 0 && m.width < 60
	list := m.filtered()

	runningTotal := 0
	for _, vm := range m.vms {
		if vm.Running {
			runningTotal++
		}
	}

	sepWidth := 70
	if m.width > 0 {
		sepWidth = m.width - 2
		if sepWidth < 10 {
			sepWidth = 10
		}
	}

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
	topBarStr := m.topBar()

	var searchRow string
	if m.searchMode {
		searchRow = lipgloss.NewStyle().Foreground(thighlight).Render(m.search.View()) +
			thelpStyle.Render("  "+t("search.active"))
	} else if m.search.Value() != "" {
		searchRow = thelpStyle.Render("/ "+m.search.Value()) +
			lipgloss.NewStyle().Foreground(tdanger).Render("  "+t("search.clear"))
	} else {
		searchRow = thelpStyle.Render(t("search.hint"))
	}

	stats := thelpStyle.Render(fmt.Sprintf(t("stats.fmt"), runningTotal, len(m.vms), len(list)))

	overhead := m.listOverhead()
	listHeight := 8
	if m.height > 0 {
		listHeight = m.height - overhead
		if listHeight < 3 {
			listHeight = 3
		}
	}

	start := m.scrollStart()

	var lsb strings.Builder
	if start > 0 {
		lsb.WriteString(thelpStyle.Render(fmt.Sprintf("  ↑ %d", start)) + "\n")
	}
	shown := 0
	for i := start; i < len(list) && shown < listHeight; i++ {
		vm := list[i]
		var badge string
		if vm.Running {
			badge = "[▶]"
		} else {
			badge = "[■]"
		}
		line := fmt.Sprintf("%s %s", badge, vm.Name)
		if i == m.cursor {
			lsb.WriteString(zone.Mark(m.id+vm.Name, tselectedStyle.Width(colWidth-4).Render(line)) + "\n")
		} else {
			lsb.WriteString(zone.Mark(m.id+vm.Name, tnormalStyle.Render(line)) + "\n")
		}
		shown++
	}
	if remaining := len(list) - (start + shown); remaining > 0 {
		lsb.WriteString(thelpStyle.Render(fmt.Sprintf("  ↓ %d", remaining)) + "\n")
	}
	listContent := lsb.String()
	if listContent == "" {
		listContent = tnormalStyle.Render(t("vms.none"))
	}

	sep := tdividerStyle.Render(strings.Repeat("─", sepWidth))
	help := m.bottomBar()

	if narrow {
		title := ttitleStyle.Render(t("app.title"))
		compact := m.compactDetail(list)
		return "\n" + topBarStr + "\n" + title + "\n" + listContent + "\n" + compact + sep + "\n" + help + "\n"
	}

	title := ttitleStyle.Render(t("app.title") + " - " + t("app.subtitle"))
	rightCol := tcolumnStyle.Width(colWidth).Render(m.buildDetail(list))
	cols := lipgloss.JoinHorizontal(lipgloss.Top, tcolumnStyle.Width(colWidth).Render(
		tsectionStyle.Render(t("vms.header")+" "+stats+"\n\n"+topBarStr+"\n"+searchRow+"\n\n")+listContent), " ", rightCol)
	return "\n" + topBarStr + "\n" + title + "\n" + cols + "\n" + sep + "\n" + help + "\n"
}

func (m tuiModel) buildDetail(list []VM) string {
	if len(list) == 0 || m.cursor >= len(list) {
		return ""
	}
	vm := list[m.cursor]
	var stateStr string
	if vm.Running {
		stateStr = trunningBadge.Render("[▶] " + t("state.running"))
	} else {
		stateStr = tstoppedBadge.Render("[■] " + t("state.stopped"))
	}

	detail := tsectionStyle.Render(t("detail.header")) + "\n\n" +
		tnormalStyle.Render(t("detail.name")+":   "+vm.Name) + "\n" +
		tnormalStyle.Render(t("detail.state")+":    ") + stateStr + "\n"
	if vm.Running {
		if vm.PID > 0 {
			detail += tnormalStyle.Render(t("detail.pid")+":      "+fmt.Sprintf("%d", vm.PID)) + "\n"
		}
		if vm.SPICEPort > 0 {
			detail += tnormalStyle.Render(t("detail.spice")+":  "+fmt.Sprintf("%d", vm.SPICEPort)) + "\n"
		}
		if vm.SSHPort > 0 {
			detail += tnormalStyle.Render(t("detail.ssh")+":     "+fmt.Sprintf("%s@localhost:%d", sshUser(vm), vm.SSHPort)) + "\n"
		}
	}
	detail += "\n\n" + m.detailButtons()
	return detail
}

func (m tuiModel) compactDetail(list []VM) string {
	if len(list) == 0 || m.cursor >= len(list) {
		return ""
	}
	vm := list[m.cursor]
	var stateStr string
	if vm.Running {
		stateStr = trunningBadge.Render("[▶]")
	} else {
		stateStr = tstoppedBadge.Render("[■]")
	}
	line := " " + tnormalStyle.Render(vm.Name) + " " + stateStr
	if vm.Running && vm.PID > 0 {
		line += thelpStyle.Render(fmt.Sprintf(" pid %d", vm.PID))
	}
	return line + "\n"
}

func vmmanBackendCmd(b Backend, vm *VM, action string) tea.Cmd {
	return func() tea.Msg {
		var err error
		switch action {
		case "boot":
			err = b.Boot(vm)
		case "kill":
			err = b.Kill(vm)
		}
		if err != nil {
			return tuiErrMsg{err}
		}
		return tuiStatusMsg{t("status." + action)}
	}
}

// connectCmd connects to a VM, preferring SPICE and falling back to SSH.
func connectCmd(vm *VM) tea.Cmd {
	return func() tea.Msg {
		var err error
		if vm.SPICEPort > 0 {
			err = ConnectToVM(vm.SPICEPort, "remote-viewer")
		} else if vm.SSHPort > 0 {
			err = ConnectToVMSSH(vm.SSHPort, vm.SSHUser)
		}
		if err != nil {
			return tuiErrMsg{err}
		}
		return tuiStatusMsg{t("status.connected")}
	}
}

// appendLog appends text to a bounded log buffer.
func appendLog(buf, text string) string {
	buf += text
	if len(buf) > maxTuiLogBytes {
		buf = buf[len(buf)-maxTuiLogBytes:]
	}
	return buf
}

// logTick schedules periodic log drain while the log screen is open.
func (m tuiModel) logTick() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg { return tuiTickMsg{} })
}

// bootLogCmd starts a VM and streams its quickemu output into the log buffer.
func (m tuiModel) bootLogCmd(vm *VM) tea.Cmd {
	return func() tea.Msg {
		if err := m.backend.BootStream(vm, func(line string) { m.logSink.Add(line) }); err != nil {
			return tuiErrMsg{err}
		}
		return tuiStatusMsg{t("status.boot")}
	}
}

// updateCreate handles keyboard input during the create-VM prompts.
func (m tuiModel) updateCreate(msg tea.KeyPressMsg) (tuiModel, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.createMode = false
		m.createStep = 0
		m.createInput.SetValue("")
		m.createInput.Blur()
		return m, nil
	case "enter":
		value := strings.TrimSpace(m.createInput.Value())
		if m.createStep == 0 {
			if value == "" {
				return m, nil
			}
			m.createName = value
			m.createStep = 1
			m.createInput.SetValue("")
			m.createInput.Prompt = t("new.guest_os") + " > "
			m.createInput.Placeholder = t("new.guest_os_ph")
			m.createInput.Focus()
			return m, nil
		}
		if m.createStep == 1 {
			if value == "" {
				value = "linux"
			}
			m.createGuestOS = value
			m.createStep = 2
			m.createInput.SetValue("")
			m.createInput.Prompt = t("new.ssh_user") + " > "
			m.createInput.Placeholder = t("new.ssh_user_ph")
			m.createInput.Focus()
			return m, nil
		}
		name := m.createName
		guestOS := m.createGuestOS
		sshUser := strings.TrimSpace(value)
		m.createMode = false
		m.createStep = 0
		m.createInput.SetValue("")
		m.createInput.Prompt = "/ "
		m.createInput.Blur()
		return m, func() tea.Msg {
			if err := m.backend.Create(VMCreateConfig{Name: name, GuestOS: guestOS, SSHUser: sshUser}); err != nil {
				return tuiErrMsg{err}
			}
			return tuiStatusMsg{fmt.Sprintf(t("status.created"), name)}
		}
	default:
		var cmd tea.Cmd
		m.createInput, cmd = m.createInput.Update(msg)
		return m, cmd
	}
}

// renderLog renders the fullscreen log screen for the selected VM.
func (m tuiModel) renderLog() string {
	vm := m.selectedVM()
	name := ""
	if vm != nil {
		name = vm.Name
	}
	title := ttitleStyle.Render(fmt.Sprintf(t("log.title"), name) + " - " + t("log.hint"))
	body := m.logBuf
	if body == "" {
		body = tnormalStyle.Render(t("log.empty"))
	}
	return "\n" + title + "\n\n" + body + "\n"
}

// renderCreate renders the create-VM prompt screen.
func (m tuiModel) renderCreate() string {
	title := ttitleStyle.Render(t("new.title"))
	var line string
	switch m.createStep {
	case 0:
		line = tsectionStyle.Render(t("new.name")) + " " + lipgloss.NewStyle().Foreground(thighlight).Render(m.createInput.View()) + "\n" + thelpStyle.Render(t("log.hint"))
	case 1:
		line = tsectionStyle.Render(fmt.Sprintf("%s: %s", t("new.name"), m.createName)) + "\n\n" +
			tsectionStyle.Render(t("new.guest_os")) + " " + lipgloss.NewStyle().Foreground(thighlight).Render(m.createInput.View()) +
			thelpStyle.Render("  ("+t("new.guest_os_ph")+")")
	case 2:
		line = tsectionStyle.Render(fmt.Sprintf("%s: %s", t("new.name"), m.createName)) + "\n" +
			tsectionStyle.Render(fmt.Sprintf("%s: %s", t("new.guest_os"), m.createGuestOS)) + "\n\n" +
			tsectionStyle.Render(t("new.ssh_user")) + " " + lipgloss.NewStyle().Foreground(thighlight).Render(m.createInput.View()) +
			thelpStyle.Render("  ("+t("new.ssh_user_ph")+")")
	}
	return "\n" + title + "\n\n" + line + "\n"
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

// RunTUI runs the TUI.
func RunTUI(vmDir string) {
	InitI18n()
	b := NewQEMUBackend(vmDir)
	common.EnsureZone()
	p := tea.NewProgram(NewTuiModel(b))
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
