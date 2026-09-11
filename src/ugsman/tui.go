package ugsman

import (
	"fmt"
	"os"
	"strings"
	"unicode"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"codeberg.org/oSoWoSo/SysMan/src/common"
	serman "codeberg.org/oSoWoSo/SysMan/src/serman"
	zone "github.com/lrstanley/bubblezone/v2"
)

// ── Styles ────────────────────────────────────────────────────────────

var (
	tuiSelected = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#005577"))
	tuiError    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
	tuiOK       = lipgloss.NewStyle().Foreground(lipgloss.Color("#55FF55"))
	tuiDanger   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
	tuiSubtle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#9B9B9B"))
)

// ── Model ─────────────────────────────────────────────────────────────

type tuiTab int

const (
	tabUsers tuiTab = iota
	tabGroups
)

type tuiDialog int

const (
	dialogNone tuiDialog = iota
	dialogAddUser
	dialogAddGroup
	dialogPassword
	dialogUserProps
)

type tuiModel struct {
	id            string
	tab           tuiTab
	users         []User
	groups        []Group
	showSystem    bool
	showAbout     bool
	cursor        int
	status        string
	statusOK      bool
	width         int
	height        int
	confirmAction string // "", "del_user", "del_group"
	confirmArg    string // user login or group name
	dialogType    tuiDialog
	dialogInputs  []string
	dialogCursor  int
	dialogLabel   string
}

// NewTuiModel returns an initialised tea.Model for the usergroups plugin.
func NewTuiModel() tea.Model {
	zone.NewGlobal()
	return tuiModel{
		id:     zone.NewPrefix(),
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

	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			break
		}
		// Check tab clicks.
		if zone.Get(m.id + "tab_users").InBounds(msg) {
			m.tab = tabUsers
			m.cursor = 0
			return m, nil
		}
		if zone.Get(m.id + "tab_groups").InBounds(msg) {
			m.tab = tabGroups
			m.cursor = 0
			return m, nil
		}
		// Check toolbar action clicks.
		if zone.Get(m.id + "a_quit").InBounds(msg) {
			return m, tea.Quit
		}
		if zone.Get(m.id + "a_about").InBounds(msg) {
			m.showAbout = true
			return m, nil
		}
		if zone.Get(m.id + "a_refresh").InBounds(msg) {
			m.refreshAll()
			return m, nil
		}
		// Users toolbar.
		if m.tab == tabUsers {
			if zone.Get(m.id + "a_add_user").InBounds(msg) {
				m.openDialog(dialogAddUser)
				return m, nil
			}
			if zone.Get(m.id + "a_del_user").InBounds(msg) {
				if m.cursor >= 0 && m.cursor < len(m.users) {
					m.confirmAction = "del_user"
					m.confirmArg = m.users[m.cursor].Login
				}
				return m, nil
			}
			if zone.Get(m.id + "a_props_user").InBounds(msg) {
				m.openDialog(dialogUserProps)
				return m, nil
			}
			if zone.Get(m.id + "a_passwd_user").InBounds(msg) {
				m.openDialog(dialogPassword)
				return m, nil
			}
		}
		// Groups toolbar.
		if m.tab == tabGroups {
			if zone.Get(m.id + "a_add_group").InBounds(msg) {
				m.openDialog(dialogAddGroup)
				return m, nil
			}
			if zone.Get(m.id + "a_del_group").InBounds(msg) {
				if m.cursor >= 0 && m.cursor < len(m.groups) {
					m.confirmAction = "del_group"
					m.confirmArg = m.groups[m.cursor].Name
				}
				return m, nil
			}
			if zone.Get(m.id + "a_members").InBounds(msg) {
				m.status = t("tui.action.members")
				m.statusOK = true
				return m, nil
			}
		}
		// Check list item clicks.
		lh := m.listHeight()
		start := 0
		if m.cursor >= lh {
			start = m.cursor - lh + 1
		}
		if m.tab == tabUsers {
			for i := start; i < len(m.users); i++ {
				if zone.Get(m.id + m.users[i].Login).InBounds(msg) {
					m.cursor = i
					return m, nil
				}
			}
		} else {
			for i := start; i < len(m.groups); i++ {
				if zone.Get(m.id + m.groups[i].Name).InBounds(msg) {
					m.cursor = i
					return m, nil
				}
			}
		}

	case tea.KeyPressMsg:
		if m.showAbout {
			m.showAbout = false
			return m, nil
		}

		// Handle dialog input
		if m.dialogType != dialogNone {
			switch msg.String() {
			case "esc":
				m.closeDialog()
				return m, nil
			case "enter":
				m.submitDialog()
				return m, nil
			case "tab", "down", "j":
				labels := m.dialogFieldLabels()
				if len(labels) > 0 {
					m.dialogCursor = (m.dialogCursor + 1) % len(labels)
				}
				return m, nil
			case "up", "k":
				labels := m.dialogFieldLabels()
				if len(labels) > 0 {
					m.dialogCursor = (m.dialogCursor - 1 + len(labels)) % len(labels)
				}
				return m, nil
			case "backspace":
				if m.dialogCursor < len(m.dialogInputs) && len(m.dialogInputs[m.dialogCursor]) > 0 {
					m.dialogInputs[m.dialogCursor] = m.dialogInputs[m.dialogCursor][:len(m.dialogInputs[m.dialogCursor])-1]
				}
				return m, nil
			default:
				if msg.String() == "ctrl+u" {
					if m.dialogCursor < len(m.dialogInputs) {
						m.dialogInputs[m.dialogCursor] = ""
					}
					return m, nil
				}
				for _, r := range msg.String() {
					if unicode.IsPrint(r) {
						if m.dialogCursor < len(m.dialogInputs) {
							m.dialogInputs[m.dialogCursor] += string(r)
						}
					}
				}
				return m, nil
			}
		}

		// Handle confirmation dialog
		if m.confirmAction != "" {
			if msg.String() == "y" {
				action := m.confirmAction
				name := m.confirmArg
				m.confirmAction = ""
				m.confirmArg = ""
				switch action {
				case "del_user":
					output, err := DeleteUser(name, false)
					if err != nil {
						m.status = t("status.userdel_err") + output + err.Error()
						m.statusOK = false
					} else {
						m.status = fmt.Sprintf(t("status.user_deleted"), name)
						m.statusOK = true
					}
					m.users = LoadUsers(m.showSystem)
					m.cursor = 0
					return m, nil
				case "del_group":
					output, err := DeleteGroup(name)
					if err != nil {
						m.status = t("status.groupdel_err") + output + err.Error()
						m.statusOK = false
					} else {
						m.status = fmt.Sprintf(t("status.group_deleted"), name)
						m.statusOK = true
					}
					m.groups = LoadGroups()
					m.cursor = 0
					return m, nil
				}
			}
			m.confirmAction = ""
			m.confirmArg = ""
			return m, nil
		}

		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "?":
			m.showAbout = true
			return m, nil
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
			m.refreshAll()
		case "a":
			// Add user/group depending on active tab.
			if m.tab == tabUsers {
				m.openDialog(dialogAddUser)
			} else {
				m.openDialog(dialogAddGroup)
			}
		case "d":
			// Delete user/group depending on active tab.
			if m.tab == tabUsers {
				if m.cursor >= 0 && m.cursor < len(m.users) {
					m.confirmAction = "del_user"
					m.confirmArg = m.users[m.cursor].Login
				}
			} else {
				if m.cursor >= 0 && m.cursor < len(m.groups) {
					m.confirmAction = "del_group"
					m.confirmArg = m.groups[m.cursor].Name
				}
			}
		case "p":
			if m.tab == tabUsers {
				m.openDialog(dialogUserProps)
			}
		case "w":
			if m.tab == tabUsers {
				m.openDialog(dialogPassword)
			}
		case "m":
			if m.tab == tabGroups {
				m.status = t("tui.action.members")
				m.statusOK = true
			}
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

func (m *tuiModel) refreshAll() {
	m.users = LoadUsers(m.showSystem)
	m.groups = LoadGroups()
	m.cursor = 0
	m.status = t("status.refreshed")
	m.statusOK = true
}

func (m *tuiModel) openDialog(d tuiDialog) {
	m.dialogType = d
	m.dialogCursor = 0
	switch d {
	case dialogAddUser:
		m.dialogInputs = []string{"", "", ""}
		m.dialogLabel = ""
	case dialogAddGroup:
		m.dialogInputs = []string{""}
		m.dialogLabel = ""
	case dialogPassword:
		m.dialogInputs = []string{"", ""}
		if m.cursor >= 0 && m.cursor < len(m.users) {
			m.dialogLabel = m.users[m.cursor].Login
		}
	case dialogUserProps:
		if m.cursor >= 0 && m.cursor < len(m.users) {
			u := m.users[m.cursor]
			m.dialogInputs = []string{u.Name, u.Shell}
			m.dialogLabel = u.Login
		} else {
			m.dialogType = dialogNone
			return
		}
	}
}

func (m *tuiModel) closeDialog() {
	m.dialogType = dialogNone
	m.dialogInputs = nil
	m.dialogCursor = 0
	m.dialogLabel = ""
}

func (m *tuiModel) dialogFieldLabels() []string {
	switch m.dialogType {
	case dialogAddUser:
		return []string{t("dialog.login"), t("dialog.fullname"), t("dialog.shell")}
	case dialogAddGroup:
		return []string{t("dialog.group_name")}
	case dialogPassword:
		return []string{t("dialog.password_new"), t("dialog.password_confirm")}
	case dialogUserProps:
		return []string{t("dialog.fullname"), t("dialog.shell")}
	}
	return nil
}

func (m *tuiModel) dialogTitle() string {
	switch m.dialogType {
	case dialogAddUser:
		return t("dialog.add_user")
	case dialogAddGroup:
		return t("dialog.add_group")
	case dialogPassword:
		return fmt.Sprintf(t("dialog.password")+" [%s]", m.dialogLabel)
	case dialogUserProps:
		return fmt.Sprintf(t("dialog.user_props")+" [%s]", m.dialogLabel)
	}
	return ""
}

func (m *tuiModel) submitDialog() {
	switch m.dialogType {
	case dialogAddUser:
		login := m.dialogInputs[0]
		if login == "" {
			m.status = t("status.useradd_err") + "login required"
			m.statusOK = false
			m.closeDialog()
			return
		}
		fullName := m.dialogInputs[1]
		shell := m.dialogInputs[2]
		output, err := AddUser(login, fullName, shell)
		if err != nil {
			m.status = t("status.useradd_err") + output + err.Error()
			m.statusOK = false
		} else {
			m.status = fmt.Sprintf(t("status.user_created"), login)
			m.statusOK = true
			m.users = LoadUsers(m.showSystem)
			m.cursor = 0
		}
	case dialogAddGroup:
		name := m.dialogInputs[0]
		if name == "" {
			m.status = t("status.groupadd_err") + "name required"
			m.statusOK = false
			m.closeDialog()
			return
		}
		output, err := AddGroup(name)
		if err != nil {
			m.status = t("status.groupadd_err") + output + err.Error()
			m.statusOK = false
		} else {
			m.status = fmt.Sprintf(t("status.group_created"), name)
			m.statusOK = true
			m.groups = LoadGroups()
			m.cursor = 0
		}
	case dialogPassword:
		if m.dialogInputs[0] != m.dialogInputs[1] {
			m.status = t("status.passwd_mismatch")
			m.statusOK = false
			m.closeDialog()
			return
		}
		output, err := SetPassword(m.dialogLabel, m.dialogInputs[0])
		if err != nil {
			m.status = t("status.chpasswd_err") + output + err.Error()
			m.statusOK = false
		} else {
			m.status = fmt.Sprintf(t("status.passwd_changed"), m.dialogLabel)
			m.statusOK = true
		}
	case dialogUserProps:
		output, err := SetUserProps(m.dialogLabel, m.dialogInputs[0], m.dialogInputs[1])
		if err != nil {
			m.status = t("status.usermod_err") + output + err.Error()
			m.statusOK = false
		} else {
			m.status = fmt.Sprintf(t("status.user_updated"), m.dialogLabel)
			m.statusOK = true
			m.users = LoadUsers(m.showSystem)
		}
	}
	m.closeDialog()
}

func (m tuiModel) listLen() int {
	if m.tab == tabUsers {
		return len(m.users)
	}
	return len(m.groups)
}

// overhead: tab bar + separator + toolbar + col header + status bar = 5 lines
const tuiOverhead = 5

func (m tuiModel) listHeight() int {
	h := m.height - tuiOverhead
	if h < 4 {
		h = 4
	}
	return h
}

func (m tuiModel) topBar() string {
	usersBtn := common.TBtn(m.id, "tab_users", t("tui.tab.users"), "1", m.tab == tabUsers)
	groupsBtn := common.TBtn(m.id, "tab_groups", t("tui.tab.groups"), "2", m.tab == tabGroups)
	return common.TBtnBar(usersBtn, groupsBtn)
}

// toolbar returns the action buttons for the currently active tab.
func (m tuiModel) toolbar() string {
	if m.tab == tabUsers {
		return common.TBtnBar(
			common.TBtn(m.id, "a_add_user", t("tui.btn.add_user"), "a", false),
			common.TBtn(m.id, "a_del_user", t("tui.btn.delete"), "d", false),
			common.TBtn(m.id, "a_props_user", t("tui.btn.properties"), "p", false),
			common.TBtn(m.id, "a_passwd_user", t("tui.btn.password"), "w", false),
			common.TBtn(m.id, "a_refresh", t("tui.action.refresh"), "r", false),
		)
	}
	return common.TBtnBar(
		common.TBtn(m.id, "a_add_group", t("tui.btn.add_group"), "a", false),
		common.TBtn(m.id, "a_del_group", t("tui.btn.delete"), "d", false),
		common.TBtn(m.id, "a_members", t("tui.btn.members"), "m", false),
		common.TBtn(m.id, "a_refresh", t("tui.action.refresh"), "r", false),
	)
}

// statusBar renders the bottom status bar: status message on the left, quit on the right.
func (m tuiModel) statusBar() string {
	if m.confirmAction != "" {
		var prompt string
		switch m.confirmAction {
		case "del_user":
			prompt = tuiDanger.Render("  ⚠ " + fmt.Sprintf(t("confirm.del_user"), m.confirmArg) + " [y/N]")
		case "del_group":
			prompt = tuiDanger.Render("  ⚠ " + fmt.Sprintf(t("confirm.del_group"), m.confirmArg) + " [y/N]")
		}
		return prompt + "\n  " + tuiSubtle.Render(t("confirm.hint"))
	}
	quitBtn := common.TBtn(m.id, "a_quit", t("tui.btn.quit"), "q", false)
	aboutBtn := common.TBtn(m.id, "a_about", t("btn.about"), "?", false)

	msg := m.status
	if msg == "" {
		msg = t("tui.status.ready")
	}
	var statusStyle lipgloss.Style
	if m.statusOK {
		statusStyle = tuiOK
	} else if m.status != "" {
		statusStyle = tuiError
	} else {
		statusStyle = lipgloss.NewStyle()
	}

	left := statusStyle.Render(msg)
	return left + strings.Repeat(" ", max(0, m.width-len(msg)-len(t("tui.btn.quit"))-len(t("btn.about"))-14)) + aboutBtn + " " + quitBtn
}

// View implements tea.Model; renders the TUI in the alt screen.
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
	if m.dialogType != dialogNone {
		return m.renderDialog()
	}
	var sb strings.Builder

	// Tab bar
	sb.WriteString(m.topBar() + "\n")
	w := m.width
	if w < 10 {
		w = 10
	}
	sb.WriteString(strings.Repeat("─", w) + "\n")

	// Toolbar
	sb.WriteString(m.toolbar() + "\n")

	// Content
	if m.tab == tabUsers {
		sb.WriteString(m.viewUsers())
	} else {
		sb.WriteString(m.viewGroups())
	}

	// Status bar at bottom
	sb.WriteString("\n" + m.statusBar())

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
			sb.WriteString(zone.Mark(m.id+u.Login, tuiSelected.Render(line)) + "\n")
		} else {
			sb.WriteString(zone.Mark(m.id+u.Login, line) + "\n")
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
			sb.WriteString(zone.Mark(m.id+gr.Name, tuiSelected.Render(line)) + "\n")
		} else {
			sb.WriteString(zone.Mark(m.id+gr.Name, line) + "\n")
		}
		shown++
	}
	return sb.String()
}

// renderAbout renders the about screen.
func (m tuiModel) renderAbout() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00AABB")).Render(t("app.title") + " - " + t("app.subtitle"))
	version := lipgloss.NewStyle().Render(t("about.version") + ": " + serman.Version)
	author := lipgloss.NewStyle().Render(t("about.author") + ": " + serman.AppAuthor)
	license := lipgloss.NewStyle().Render(t("about.license") + ": " + serman.AppLicense)
	url := lipgloss.NewStyle().Render(serman.AppURL)
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#9B9B9B")).Render(t("about.hint"))
	return "\n" + title + "\n\n" + version + "\n" + author + "\n" + license + "\n" + url + "\n\n" + hint + "\n"
}

// renderDialog renders a form dialog overlay.
func (m tuiModel) renderDialog() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00AABB"))
	labelStyle := lipgloss.NewStyle().Bold(true)
	focusStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#005577"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9B9B9B"))

	var sb strings.Builder
	sb.WriteString("\n  " + titleStyle.Render(m.dialogTitle()) + "\n\n")

	labels := m.dialogFieldLabels()
	for i, label := range labels {
		value := ""
		if i < len(m.dialogInputs) {
			value = m.dialogInputs[i]
		}
		fieldLine := "    " + labelStyle.Render(label+": ") + "[" + value + "_]"
		if i == m.dialogCursor {
			fieldLine = "    " + focusStyle.Render(label+": ") + focusStyle.Render("["+value+"_]")
		}
		sb.WriteString(fieldLine + "\n")
	}

	sb.WriteString("\n    " + hintStyle.Render(t("dialog.hint")) + "\n")
	return sb.String()
}

// RunTUI runs the Users & Groups manager as a standalone Bubbletea TUI.
func RunTUI() {
	common.EnsureZone()
	prog := tea.NewProgram(NewTuiModel())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
