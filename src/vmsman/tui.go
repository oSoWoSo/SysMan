package vmman

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textarea"
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
	tselectedStyle = common.TSelectedStyle
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
	confirmAction string
	confirmArg    string

	createMode    bool
	createStep    int
	createName    string
	createGuestOS string
	createISO     string
	createMemory  string
	createCores   string
	createSSHUser string
	createInput   textinput.Model

	editMode     string // "config", "ssh_user", "settings"
	editVM       *VM
	editInput    textinput.Model
	configEditor textarea.Model

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
type tuiRefreshMsg struct{}

var (
	tkeyUp       = key.NewBinding(key.WithKeys("up", "k"))
	tkeyDown     = key.NewBinding(key.WithKeys("down", "j"))
	tkeyConnect  = key.NewBinding(key.WithKeys("c"))
	tkeyBoot     = key.NewBinding(key.WithKeys("b"))
	tkeyKill     = key.NewBinding(key.WithKeys("K"))
	tkeyNew      = key.NewBinding(key.WithKeys("n"))
	tkeyEdit     = key.NewBinding(key.WithKeys("e"))
	tkeyLog      = key.NewBinding(key.WithKeys("l"))
	tkeySSHUser  = key.NewBinding(key.WithKeys("u"))
	tkeySettings = key.NewBinding(key.WithKeys("s"))
	tkeyQuit     = key.NewBinding(key.WithKeys("q", "ctrl+c", "esc"))
	tkeySearch   = key.NewBinding(key.WithKeys("/"))
	tkeyEsc      = key.NewBinding(key.WithKeys("esc"))
	tkeyEnter    = key.NewBinding(key.WithKeys("enter"))
	tkeyFilter   = key.NewBinding(key.WithKeys("tab"))
	tkeyReload   = key.NewBinding(key.WithKeys("r"))
	tkeyAbout    = key.NewBinding(key.WithKeys("?"))
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

	ei := textinput.New()
	ei.CharLimit = 64
	ei.SetWidth(48)
	ei.SetStyles(st)
	ei.Prompt = "> "

	ce := textarea.New()
	ce.SetWidth(72)
	ce.SetHeight(20)
	// Check if VM directory exists and report any errors
	status := ""
	statusErr := false
	if errMsg := CheckVMDir(b.VMDir()); errMsg != "" {
		status = errMsg
		statusErr = true
	}

	return tuiModel{
		id:           "vmsman:" + zone.NewPrefix(),
		backend:      b,
		vms:          b.List(),
		search:       ti,
		createInput:  ci,
		editInput:    ei,
		configEditor: ce,
		logSink:      &tuiLogBuffer{},
		status:       status,
		statusErr:    statusErr,
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

func (m tuiModel) Init() tea.Cmd {
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return tuiRefreshMsg{} })
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok && common.ToggleTUIKeyHints(key) {
		return m, nil
	}
	if wheel, ok := msg.(tea.MouseWheelMsg); ok && !m.logMode && m.editMode == "" {
		keyCode := tea.KeyDown
		if wheel.Button == tea.MouseWheelUp {
			keyCode = tea.KeyUp
		}
		return m.Update(tea.KeyPressMsg{Code: keyCode})
	}
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
		if m.editMode != "" {
			if msg.String() == "esc" {
				m.editMode = ""
				m.editVM = nil
				m.editInput.Blur()
				return m, nil
			}
			if m.editMode == "config" {
				if msg.String() == "esc" {
					m.editMode, m.editVM = "", nil
					m.configEditor.Blur()
					return m, nil
				}
				if msg.String() == "ctrl+s" {
					vm := m.editVM
					content := m.configEditor.Value()
					m.editMode = ""
					m.configEditor.Blur()
					if vm != nil {
						if err := m.backend.WriteConfig(vm, content); err != nil {
							m.status, m.statusErr = t("status.err")+err.Error(), true
						} else {
							m.status, m.statusErr = fmt.Sprintf(t("status.saved"), vm.Name), false
							return m, func() tea.Msg { return tuiVMMsg{vms: m.backend.List()} }
						}
					}
					return m, nil
				}
				var cmd tea.Cmd
				m.configEditor, cmd = m.configEditor.Update(msg)
				return m, cmd
			}
			if msg.String() == "enter" {
				value := strings.TrimSpace(m.editInput.Value())
				mode, vm := m.editMode, m.editVM
				m.editMode = ""
				m.editInput.Blur()
				if mode == "ssh_user" && vm != nil {
					if value != "" && !ValidVMName(value) {
						m.status, m.statusErr = t("status.err")+t("new.err_name"), true
					} else if err := SetSSHUser(vm, value); err != nil {
						m.status, m.statusErr = t("status.err")+err.Error(), true
					} else {
						m.status, m.statusErr = fmt.Sprintf(t("status.saved"), vm.Name), false
						return m, func() tea.Msg { return tuiVMMsg{vms: m.backend.List()} }
					}
				} else if mode == "settings" {
					cfg := common.LoadSysManConfig()
					cfg.Vmsman.VMDir = value
					if err := common.SaveSysManConfig(cfg); err != nil {
						m.status, m.statusErr = t("status.err")+err.Error(), true
					} else {
						m.status, m.statusErr = t("status.settings_saved"), false
					}
				}
				return m, nil
			}
			var cmd tea.Cmd
			m.editInput, cmd = m.editInput.Update(msg)
			return m, cmd
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
				content, err := m.backend.ReadConfig(vm)
				if err != nil {
					m.status, m.statusErr = t("status.err")+err.Error(), true
					return m, nil
				}
				m.editMode, m.editVM = "config", vm
				m.configEditor.SetValue(content)
				m.configEditor.Focus()
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
		case key.Matches(msg, tkeySSHUser):
			if vm := m.selectedVM(); vm != nil {
				m.editMode, m.editVM = "ssh_user", vm
				m.editInput.SetValue(vm.SSHUser)
				m.editInput.Placeholder = t("ssh_user.empty")
				m.editInput.Focus()
				return m, textinput.Blink
			}
		case key.Matches(msg, tkeySettings):
			cfg := common.LoadSysManConfig()
			m.editMode = "settings"
			m.editInput.SetValue(cfg.Vmsman.VMDir)
			m.editInput.Placeholder = DefaultVMDir
			m.editInput.Focus()
			return m, textinput.Blink

		case key.Matches(msg, tkeyAbout):
			m.showAbout = true
			return m, nil
		}

	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			break
		}
		if m.createMode {
			if zone.Get(m.id + "a_create_cancel").InBounds(msg) {
				m.createMode = false
				m.createInput.Blur()
				return m, nil
			}
			if zone.Get(m.id + "a_create_next").InBounds(msg) {
				var cmd tea.Cmd
				m, cmd = m.updateCreate(tea.KeyPressMsg{Code: tea.KeyEnter})
				return m, cmd
			}
		}
		if m.editMode != "" {
			if zone.Get(m.id + "a_edit_save").InBounds(msg) {
				if m.editMode == "config" && m.editVM != nil {
					if err := m.backend.WriteConfig(m.editVM, m.configEditor.Value()); err != nil {
						m.status, m.statusErr = t("status.err")+err.Error(), true
					} else {
						m.status, m.statusErr = fmt.Sprintf(t("status.saved"), m.editVM.Name), false
						m.editMode, m.editVM = "", nil
						return m, func() tea.Msg { return tuiVMMsg{vms: m.backend.List()} }
					}
				} else {
					return m, func() tea.Msg { return tea.KeyPressMsg{Code: tea.KeyEnter} }
				}
				return m, nil
			}
			if zone.Get(m.id + "a_edit_cancel").InBounds(msg) {
				m.editMode, m.editVM = "", nil
				m.configEditor.Blur()
				m.editInput.Blur()
				return m, nil
			}
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
			if vm := m.selectedVM(); vm != nil {
				content, err := m.backend.ReadConfig(vm)
				if err != nil {
					m.status, m.statusErr = t("status.err")+err.Error(), true
					return m, nil
				}
				m.editMode, m.editVM = "config", vm
				m.configEditor.SetValue(content)
				m.configEditor.Focus()
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
		if zone.Get(m.id + "a_ssh_user").InBounds(msg) {
			if vm := m.selectedVM(); vm != nil {
				m.editMode, m.editVM = "ssh_user", vm
				m.editInput.SetValue(vm.SSHUser)
				m.editInput.Placeholder = t("ssh_user.empty")
				m.editInput.Focus()
				return m, textinput.Blink
			}
		}
		if zone.Get(m.id + "a_settings").InBounds(msg) {
			cfg := common.LoadSysManConfig()
			m.editMode = "settings"
			m.editInput.SetValue(cfg.Vmsman.VMDir)
			m.editInput.Placeholder = DefaultVMDir
			m.editInput.Focus()
			return m, textinput.Blink
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

	case tuiRefreshMsg:
		m.vms = m.backend.List()
		m = m.clampCursor()
		return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg { return tuiRefreshMsg{} })
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
func (m tuiModel) View() tea.View { return m.view(true) }

// EmbeddedView returns the unscanned view for composition inside sysman-tui.
func (m tuiModel) EmbeddedView() tea.View { return m.view(false) }

func (m tuiModel) view(scan bool) tea.View {
	content := m.render()
	if scan {
		content = zone.Scan(content)
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// listOverhead returns the number of non-list lines consumed by the layout.
func (m tuiModel) listOverhead() int {
	overhead := 16 // reserve space for the module header, pane borders, and footer
	if m.width > 0 && m.width < 60 {
		overhead = 17 // compact detail adds one line
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

// bottomBar places controls at the left and the status at the right, like the GUI footer.
func (m tuiModel) bottomBar() string {
	if m.confirmAction != "" {
		var prompt string
		switch m.confirmAction {
		case "kill":
			prompt = tdangerStyle.Render("  ⚠ " + fmt.Sprintf(t("confirm.kill"), m.confirmArg) + " [y/N]")
		}
		return prompt + "\n  " + thelpStyle.Render(t("confirm.hint"))
	}
	status := ""
	if m.status != "" {
		if m.statusErr {
			status = tstatusErr.Render(m.status)
		} else {
			status = tstatusOk.Render(m.status)
		}
	}
	settings := common.TBtn(m.id, "a_settings", t("btn.settings"), "s", false)
	about := common.TBtn(m.id, "a_about", t("btn.about"), "?", false)
	controls := []string{settings, about}
	if !common.TUIEmbedded() {
		controls = append(controls, common.TBtn(m.id, "a_quit", t("action.quit"), "q", false))
	}
	return common.TUIFooter(common.TBtnBar(controls...), status, m.width)
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
	if vm != nil {
		buttons = append(buttons, common.TBtn(m.id, "a_ssh_user", t("btn.ssh_user"), "u", false))
	}
	return common.TBtnBar(buttons...)
}

func (m tuiModel) render() string {
	return m.renderContent()
}
func (m tuiModel) renderContent() string {
	if m.editMode == "config" {
		return "\n" + ttitleStyle.Render(fmt.Sprintf(t("edit.title"), m.editVM.Name)) + "\n\n" +
			m.configEditor.View() + "\n\n" +
			common.TBtnBar(common.TBtn(m.id, "a_edit_save", t("btn.save"), "ctrl+s", false),
				common.TBtn(m.id, "a_edit_cancel", t("btn.cancel"), "esc", false)) + "\n"
	}
	if m.editMode != "" {
		title, prompt := t("settings.title"), t("settings.vm_dir")
		if m.editMode == "ssh_user" && m.editVM != nil {
			title, prompt = fmt.Sprintf(t("ssh_user.title"), m.editVM.Name), t("btn.ssh_user")
		}
		return "\n" + ttitleStyle.Render(title) + "\n\n" +
			tsectionStyle.Render(prompt+": ") + m.editInput.View() + "\n\n" +
			common.TBtnBar(common.TBtn(m.id, "a_edit_save", t("btn.save"), "enter", false),
				common.TBtn(m.id, "a_edit_cancel", t("btn.cancel"), "esc", false)) + "\n"
	}
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
		// Keep the selected row on a single line and aligned with unselected rows.
		contentWidth := colWidth - 6
		if contentWidth < 8 {
			contentWidth = 8
		}
		if lipgloss.Width(line) > contentWidth {
			name := vm.Name
			for lipgloss.Width(fmt.Sprintf("%s %s…", badge, name)) > contentWidth && len(name) > 0 {
				name = name[:len(name)-1]
			}
			line = fmt.Sprintf("%s %s…", badge, name)
		}
		if i == m.cursor {
			lsb.WriteString(zone.Mark(m.id+vm.Name, tselectedStyle.Width(colWidth-2).Render(line)) + "\n")
		} else {
			lsb.WriteString(zone.Mark(m.id+vm.Name, lipgloss.NewStyle().Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#333333"), Dark: lipgloss.Color("#CCCCCC")}).Render(line)) + "\n")
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
		return "\n" + title + "\n" + topBarStr + "\n" + listContent + "\n" + compact + sep + "\n" + help + "\n"
	}

	title := ttitleStyle.Render(t("app.title") + " - " + t("app.subtitle"))
	rightCol := tcolumnStyle.Width(colWidth).Render(m.buildDetail(list))
	cols := lipgloss.JoinHorizontal(lipgloss.Top, tcolumnStyle.Width(colWidth).Render(
		tsectionStyle.Render(t("vms.header")+" "+stats+"\n"+topBarStr+"\n"+searchRow)+"\n"+listContent), " ", rightCol)
	return "\n" + title + "\n" + cols + "\n" + sep + "\n" + help + "\n"
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
		switch m.createStep {
		case 1:
			if value == "" {
				value = "linux"
			}
			m.createGuestOS = value
			m.createStep = 2
			m.createInput.SetValue("")
			m.createInput.Prompt = t("new.ssh_user") + " > "
			m.createInput.Placeholder = t("new.ssh_user_ph")
		case 2:
			m.createSSHUser = value
			m.createStep = 3
			m.createInput.SetValue("")
			m.createInput.Prompt = t("new.iso") + " > "
			m.createInput.Placeholder = t("new.iso_ph")
		case 3:
			m.createISO = value
			m.createStep = 4
			m.createInput.SetValue("")
			m.createInput.Prompt = t("new.memory") + " > "
			m.createInput.Placeholder = "2048"
		case 4:
			m.createMemory = value
			m.createStep = 5
			m.createInput.SetValue("")
			m.createInput.Prompt = t("new.cores") + " > "
			m.createInput.Placeholder = "2"
		case 5:
			m.createCores = value
			name, guestOS, sshUser, iso := m.createName, m.createGuestOS, m.createSSHUser, m.createISO
			memory, cores := 0, 0
			_, _ = fmt.Sscanf(m.createMemory, "%d", &memory)
			_, _ = fmt.Sscanf(m.createCores, "%d", &cores)
			m.createMode = false
			m.createStep = 0
			m.createInput.SetValue("")
			m.createInput.Blur()
			return m, func() tea.Msg {
				cfg := VMCreateConfig{Name: name, GuestOS: guestOS, SSHUser: sshUser, ISO: iso, MemoryMB: memory, CPUCores: cores}
				if err := m.backend.Create(cfg); err != nil {
					return tuiErrMsg{err}
				}
				return tuiStatusMsg{fmt.Sprintf(t("status.created"), name)}
			}
		}
		m.createInput.Focus()
		return m, nil
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
	case 3, 4, 5:
		line = tsectionStyle.Render(fmt.Sprintf("%s: %s", t("new.name"), m.createName)) + "\n" +
			tsectionStyle.Render(fmt.Sprintf("%s: %s", t("new.guest_os"), m.createGuestOS)) + "\n" +
			tsectionStyle.Render(fmt.Sprintf("%s: %s", t("new.ssh_user"), m.createSSHUser)) + "\n"
		if m.createStep > 3 {
			line += tsectionStyle.Render(fmt.Sprintf("%s: %s", t("new.iso"), m.createISO)) + "\n"
		}
		if m.createStep > 4 {
			line += tsectionStyle.Render(fmt.Sprintf("%s: %s", t("new.memory"), m.createMemory)) + "\n"
		}
		prompt := map[int]string{3: t("new.iso"), 4: t("new.memory"), 5: t("new.cores")}[m.createStep]
		line += "\n" + tsectionStyle.Render(prompt) + " " + lipgloss.NewStyle().Foreground(thighlight).Render(m.createInput.View())
	}
	return "\n" + title + "\n\n" + line + "\n\n" +
		common.TBtnBar(common.TBtn(m.id, "a_create_next", t("btn.next"), "enter", false),
			common.TBtn(m.id, "a_create_cancel", t("btn.cancel"), "esc", false)) + "\n"
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
