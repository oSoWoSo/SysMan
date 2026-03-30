package usergroups

import (
	"fmt"
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

func (m tuiModel) View() string {
	var sb strings.Builder

	// Tab bar
	u := "1 Users"
	gr := "2 Groups"
	if m.tab == tabUsers {
		u = tuiHeader.Render(u)
	}
	if m.tab == tabGroups {
		gr = tuiHeader.Render(gr)
	}
	sb.WriteString(fmt.Sprintf("  %s   %s\n", u, gr))
	sb.WriteString(strings.Repeat("─", m.width) + "\n")

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
		help := "↑/↓: navigate   r: refresh   s: toggle system users   1/2: switch tab"
		sb.WriteString(tuiHelp.Render(help))
	}

	return sb.String()
}

func (m tuiModel) viewUsers() string {
	if len(m.users) == 0 {
		return "  (no users)\n"
	}
	var sb strings.Builder
	header := fmt.Sprintf("  %-20s %6s  %-20s  %-16s  %s",
		"Login", "UID", "Full Name", "Group", "Home")
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render(header) + "\n")

	for i, u := range m.users {
		line := fmt.Sprintf("  %-20s %6d  %-20s  %-16s  %s",
			u.Login, u.UID, u.Name, u.Primary, u.Home)
		if i == m.cursor {
			sb.WriteString(tuiSelected.Render(line) + "\n")
		} else {
			sb.WriteString(line + "\n")
		}
	}
	return sb.String()
}

func (m tuiModel) viewGroups() string {
	if len(m.groups) == 0 {
		return "  (no groups)\n"
	}
	var sb strings.Builder
	header := fmt.Sprintf("  %-20s %6s  %s", "Name", "GID", "Members")
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render(header) + "\n")

	for i, gr := range m.groups {
		members := strings.Join(gr.Members, ", ")
		line := fmt.Sprintf("  %-20s %6d  %s", gr.Name, gr.GID, members)
		if i == m.cursor {
			sb.WriteString(tuiSelected.Render(line) + "\n")
		} else {
			sb.WriteString(line + "\n")
		}
	}
	return sb.String()
}
