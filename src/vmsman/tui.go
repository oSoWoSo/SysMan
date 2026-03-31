package vmman

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	tsubtleColor = lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#585858"}
	thighlight   = lipgloss.AdaptiveColor{Light: "#00AABB", Dark: "#00DDFF"}
	tdanger      = lipgloss.AdaptiveColor{Light: "#CC3333", Dark: "#FF5555"}
	tGreen       = lipgloss.AdaptiveColor{Light: "#22AA55", Dark: "#44DD77"}
	twarn        = lipgloss.AdaptiveColor{Light: "#BB8800", Dark: "#FFCC00"}

	ttitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(thighlight).Padding(0, 1).MarginBottom(1)
	tsectionStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.AdaptiveColor{Light: "#444444", Dark: "#AAAAAA"})
	tselectedStyle = lipgloss.NewStyle().Bold(true).Foreground(thighlight).Background(lipgloss.AdaptiveColor{Light: "#DDFAFF", Dark: "#003344"}).Padding(0, 1)
	tnormalStyle   = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#333333", Dark: "#CCCCCC"}).Padding(0, 1)

	trunningBadge = lipgloss.NewStyle().Foreground(tGreen).Bold(true)
	tstoppedBadge = lipgloss.NewStyle().Foreground(tsubtleColor)
	tstatusOk     = lipgloss.NewStyle().Foreground(tGreen).Italic(true)
	tstatusErr    = lipgloss.NewStyle().Foreground(tdanger).Bold(true)
	thelpStyle    = lipgloss.NewStyle().Foreground(tsubtleColor)
	tdividerStyle = lipgloss.NewStyle().Foreground(tsubtleColor)
	twarnStyle    = lipgloss.NewStyle().Foreground(twarn)
	tcolumnStyle  = lipgloss.NewStyle().Padding(0, 1).Border(lipgloss.RoundedBorder()).BorderForeground(tsubtleColor)

	tfilterActive   = lipgloss.NewStyle().Bold(true).Foreground(thighlight).Padding(0, 1).Underline(true)
	tfilterInactive = lipgloss.NewStyle().Foreground(tsubtleColor).Padding(0, 1)
)

type vmmanFilter = FilterMode

func (f vmmanFilter) label() string {
	switch f {
	case FilterRunning:
		return t("filter.running")
	case FilterStopped:
		return t("filter.stopped")
	default:
		return t("filter.all")
	}
}

type tuiModel struct {
	backend    Backend
	vms        []VM
	cursor     int
	filter     vmmanFilter
	search     textinput.Model
	searchMode bool
	status     string
	statusErr  bool
	width      int
	height     int
}

type tuiVMMsg struct{ vms []VM }
type tuiErrMsg struct{ err error }
type tuiStatusMsg struct{ msg string }
type tuiOpDoneMsg struct{ name, action string }

var (
	tkeyUp      = key.NewBinding(key.WithKeys("up", "k"))
	tkeyDown    = key.NewBinding(key.WithKeys("down", "j"))
	tkeyToggle  = key.NewBinding(key.WithKeys("enter", " "))
	tkeyConnect = key.NewBinding(key.WithKeys("c"))
	tkeyBoot    = key.NewBinding(key.WithKeys("b"))
	tkeyKill    = key.NewBinding(key.WithKeys("k"))
	tkeyQuit    = key.NewBinding(key.WithKeys("q", "ctrl+c", "esc"))
	tkeySearch  = key.NewBinding(key.WithKeys("/"))
	tkeyEsc     = key.NewBinding(key.WithKeys("esc"))
	tkeyEnter   = key.NewBinding(key.WithKeys("enter"))
	tkeyFilter  = key.NewBinding(key.WithKeys("tab"))
)

func NewTuiModel(b Backend) tea.Model {
	ti := textinput.New()
	ti.Placeholder = t("search.placeholder")
	ti.CharLimit = 64
	ti.Width = 28
	ti.PromptStyle = lipgloss.NewStyle().Foreground(thighlight)
	ti.Prompt = "/ "
	return tuiModel{
		backend: b,
		vms:     b.List(),
		search:  ti,
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

func (m tuiModel) currentName() string {
	list := m.filtered()
	if m.cursor < 0 || m.cursor >= len(list) {
		return ""
	}
	return list[m.cursor].Name
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

	case tea.KeyMsg:
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
			if vm != nil && vm.Running && vm.SPICEPort > 0 {
				return m, func() tea.Msg {
					err := ConnectToVM(vm.SPICEPort, "remote-viewer")
					if err != nil {
						return tuiErrMsg{err}
					}
					return tuiStatusMsg{t("status.connected")}
				}
			}
		case key.Matches(msg, tkeyBoot):
			vm := m.selectedVM()
			if vm != nil && !vm.Running {
				return m, vmmanBackendCmd(m.backend, vm, "boot")
			}
		case key.Matches(msg, tkeyKill):
			vm := m.selectedVM()
			if vm != nil && vm.Running {
				return m, vmmanBackendCmd(m.backend, vm, "kill")
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
	}

	return m, nil
}

func (m tuiModel) View() string {
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

	filters := []vmmanFilter{FilterAll, FilterRunning, FilterStopped}
	filterRow := ""
	for _, f := range filters {
		if f == m.filter {
			filterRow += tfilterActive.Render(f.label())
		} else {
			filterRow += tfilterInactive.Render(f.label())
		}
		filterRow += " "
	}

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

	stats := tstoppedBadge.Render(fmt.Sprintf(t("stats.fmt"), runningTotal, len(m.vms), len(list)))

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

	start := 0
	if m.cursor >= listHeight {
		start = m.cursor - listHeight + 1
	}

	var lsb strings.Builder
	if start > 0 {
		lsb.WriteString(thelpStyle.Render(fmt.Sprintf("  ↑ %d", start)) + "\n")
	}
	shown := 0
	for i := start; i < len(list) && shown < listHeight; i++ {
		vm := list[i]
		var badge string
		if vm.Running {
			badge = trunningBadge.Render("[▶]")
		} else {
			badge = tstoppedBadge.Render("[■]")
		}
		line := fmt.Sprintf("%s %s", badge, vm.Name)
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
		listContent = tnormalStyle.Render(t("vms.none"))
	}

	statusLine := ""
	if m.status != "" {
		if m.statusErr {
			statusLine = "\n" + tstatusErr.Render(m.status)
		} else {
			statusLine = "\n" + tstatusOk.Render(m.status)
		}
	}

	var helpText string
	switch {
	case m.searchMode:
		helpText = t("help.search")
	case narrow:
		helpText = "↑↓=nav  c=connect  b=boot  k=kill  /=search  q=quit"
	default:
		helpText = t("help.normal")
	}

	sep := tdividerStyle.Render(strings.Repeat("─", sepWidth))
	help := thelpStyle.Render(helpText)

	if narrow {
		title := ttitleStyle.Render(t("app.title"))
		compact := m.compactDetail(list)
		return "\n" + title + "\n" + listContent + "\n" + compact + sep + statusLine + "\n" + help + "\n"
	}

	title := ttitleStyle.Render(t("app.title") + " - " + t("app.subtitle"))
	rightCol := tcolumnStyle.Width(colWidth).Render(m.buildDetail(list))
	cols := lipgloss.JoinHorizontal(lipgloss.Top, tcolumnStyle.Width(colWidth).Render(
		tsectionStyle.Render(t("vms.header")+" "+stats+"\n\n"+filterRow+"\n"+searchRow+"\n\n")+listContent), " ", rightCol)
	return "\n" + title + "\n" + cols + "\n" + sep + statusLine + "\n" + help + "\n"
}

func (m tuiModel) buildDetail(list []VM) string {
	if len(list) == 0 || m.cursor >= len(list) {
		return ""
	}
	vm := list[m.cursor]
	var stateStr, actionStr string
	if vm.Running {
		stateStr = trunningBadge.Render("[▶] " + t("state.running"))
		actionStr = twarnStyle.Render(t("action.connect"))
	} else {
		stateStr = tstoppedBadge.Render("[■] " + t("state.stopped"))
		actionStr = lipgloss.NewStyle().Foreground(tGreen).Bold(true).Render(t("action.boot"))
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
	}
	detail += "\n" + tnormalStyle.Render("action:  ") + actionStr
	if vm.Running {
		detail += "\n\n" + thelpStyle.Render("c=connect  k=kill")
	} else {
		detail += "\n\n" + thelpStyle.Render("b=boot")
	}
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
		line += tstoppedBadge.Render(fmt.Sprintf(" pid %d", vm.PID))
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

func RunTUI(vmDir string) {
	InitI18n()
	b := NewQEMUBackend(vmDir)
	p := tea.NewProgram(NewTuiModel(b), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
