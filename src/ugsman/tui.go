package ugsman

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Styles ────────────────────────────────────────────────────────────

var (
	tuiHeader   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00DDFF"))
	tuiSelected = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#005577"))
	tuiHelp     = lipgloss.NewStyle().Foreground(lipgloss.Color("#585858"))
	tuiError    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
	tuiOK       = lipgloss.NewStyle().Foreground(lipgloss.Color("#55FF55"))
)

// ── Model ─────────────────────────────────────────────────────────────

type tuiTab int

const (
	tabUsers tuiTab = iota
	tabGroups
)

type tuiModel struct {
	tab        tuiTab
	users      []User
	groups     []Group
	showSystem bool
	cursor     int
	status     string
	statusOK   bool
	width      int
	height     int
}

// NewTuiModel returns an initialised tea.Model for the usergroups plugin.
func NewTuiModel() tea.Model {
	return tuiModel{
		users:  LoadUsers(false),
		groups: LoadGroups(),
		cursor: 0,
	}
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.tab = tabUsers
			m.cursor = 0
		case "2":
			m.tab = tabGroups
			m.cursor = 0
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			m.cursor++
		case "s":
			m.showSystem = !m.showSystem
			m.users = LoadUsers(m.showSystem)
			m.cursor = 0
		case "r":
			m.users = LoadUsers(m.showSystem)
			m.groups = LoadGroups()
			m.cursor = 0
			m.status = "Refreshed"
			m.statusOK = true
		}

		// clamp cursor
		last := m.listLen() - 1
		if last < 0 {
			last = 0
		}
		if m.cursor > last {
			m.cursor = last
		}
	}
	return m, nil
}

func (m tuiModel) listLen() int {
	if m.tab == tabUsers {
		return len(m.users)
	}
	return len(m.groups)
}

// overhead: tab bar + separator + col header + blank + status/help = 5 lines
const tuiOverhead = 5

func (m tuiModel) listHeight() int {
	h := m.height - tuiOverhead
	if h < 4 {
		h = 4
	}
	return h
}

func (m tuiModel) View() string {
	var sb strings.Builder

	// Tab bar
	u := t("tui.tab.users")
	gr := t("tui.tab.groups")
	if m.tab == tabUsers {
		u = tuiHeader.Render(u)
	}
	if m.tab == tabGroups {
		gr = tuiHeader.Render(gr)
	}
	sb.WriteString(fmt.Sprintf("  %s   %s\n", u, gr))
	w := m.width
	if w < 10 {
		w = 10
	}
	sb.WriteString(strings.Repeat("─", w) + "\n")

	// Content
	if m.tab == tabUsers {
		sb.WriteString(m.viewUsers())
	} else {
		sb.WriteString(m.viewGroups())
	}

	// Status / help
	sb.WriteString("\n")
	if m.status != "" {
		style := tuiOK
		if !m.statusOK {
			style = tuiError
		}
		sb.WriteString(style.Render(m.status))
	} else {
		help := t("tui.help")
		sb.WriteString(tuiHelp.Render(help))
	}

	return sb.String()
}

func (m tuiModel) viewUsers() string {
	if len(m.users) == 0 {
		return "  " + t("tui.none.users") + "\n"
	}
	var sb strings.Builder
	header := fmt.Sprintf("  %-20s %6s  %-20s  %-16s  %s",
		t("col.login"), t("col.uid"), t("col.fullname"), t("col.group"), t("col.home"))
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render(header) + "\n")

	lh := m.listHeight()
	start := 0
	if m.cursor >= lh {
		start = m.cursor - lh + 1
	}
	shown := 0
	for i := start; i < len(m.users) && shown < lh; i++ {
		u := m.users[i]
		line := fmt.Sprintf("  %-20s %6d  %-20s  %-16s  %s",
			u.Login, u.UID, u.Name, u.Primary, u.Home)
		if i == m.cursor {
			sb.WriteString(tuiSelected.Render(line) + "\n")
		} else {
			sb.WriteString(line + "\n")
		}
		shown++
	}
	return sb.String()
}

func (m tuiModel) viewGroups() string {
	if len(m.groups) == 0 {
		return "  " + t("tui.none.groups") + "\n"
	}
	var sb strings.Builder
	header := fmt.Sprintf("  %-20s %6s  %s", t("col.name"), t("col.gid"), t("col.members"))
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render(header) + "\n")

	lh := m.listHeight()
	start := 0
	if m.cursor >= lh {
		start = m.cursor - lh + 1
	}
	shown := 0
	for i := start; i < len(m.groups) && shown < lh; i++ {
		gr := m.groups[i]
		members := strings.Join(gr.Members, ", ")
		line := fmt.Sprintf("  %-20s %6d  %s", gr.Name, gr.GID, members)
		if i == m.cursor {
			sb.WriteString(tuiSelected.Render(line) + "\n")
		} else {
			sb.WriteString(line + "\n")
		}
		shown++
	}
	return sb.String()
}

// RunTUI runs the Users & Groups manager as a standalone Bubbletea TUI.
func RunTUI() {
	prog := tea.NewProgram(NewTuiModel(), tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
