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
	tuiSelected = common.TSelectedStyle
	tuiError    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
	tuiOK       = lipgloss.NewStyle().Foreground(lipgloss.Color("#55FF55"))
	tuiDanger   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
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
	dialogGroupMembers
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
	removeHome    bool
	dialogType    tuiDialog
	dialogInputs  []string
	dialogCursor  int
	dialogLabel   string
	memberUsers   []User
	memberChecks  []bool
}

// NewTuiModel returns an initialised tea.Model for the usergroups plugin.
func NewTuiModel() tea.Model {
	zone.NewGlobal()
	return tuiModel{
		id:     "ugsman:" + zone.NewPrefix(),
		users:  LoadUsers(false),
		groups: LoadGroups(),
		cursor: 0,
	}
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok && common.ToggleTUIKeyHints(key) {
		return m, nil
	}
	if wheel, ok := msg.(tea.MouseWheelMsg); ok {
		keyCode := tea.KeyDown
		if wheel.Button == tea.MouseWheelUp {
			keyCode = tea.KeyUp
		}
		return m.Update(tea.KeyPressMsg{Code: keyCode})
	}
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
		if m.tab == tabUsers && zone.Get(m.id+"a_system").InBounds(msg) {
			m.showSystem = !m.showSystem
			m.users = LoadUsers(m.showSystem)
			m.cursor = 0
			return m, nil
		}
		if m.confirmAction != "" {
			if zone.Get(m.id + "confirm_yes").InBounds(msg) {
				m.executeConfirm()
				return m, nil
			}
			if zone.Get(m.id + "confirm_no").InBounds(msg) {
				m.confirmAction = ""
				m.confirmArg = ""
				m.removeHome = false
				return m, nil
			}
			if zone.Get(m.id + "confirm_home").InBounds(msg) {
				m.removeHome = !m.removeHome
				return m, nil
			}
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
					m.removeHome = false
				} else {
					m.status = t("status.no_user")
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
				} else {
					m.status = t("status.no_group")
				}
				return m, nil
			}
			if zone.Get(m.id + "a_members").InBounds(msg) {
				m.openDialog(dialogGroupMembers)
				return m, nil
			}
		}
		if m.dialogType == dialogGroupMembers {
			for i, user := range m.memberUsers {
				if zone.Get(m.id + "member_" + user.Login).InBounds(msg) {
					m.memberChecks[i] = !m.memberChecks[i]
					m.dialogCursor = i
					return m, nil
				}
			}
		}
		if m.dialogType != dialogNone {
			for i := range m.dialogFieldLabels() {
				if zone.Get(m.id + fmt.Sprintf("field_%d", i)).InBounds(msg) {
					m.dialogCursor = i
					return m, nil
				}
			}
			if zone.Get(m.id + "dialog_apply").InBounds(msg) {
				m.submitDialog()
				return m, nil
			}
			if zone.Get(m.id + "dialog_cancel").InBounds(msg) {
				m.closeDialog()
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
					m.openDialog(dialogUserProps)
					return m, nil
				}
			}
		} else {
			for i := start; i < len(m.groups); i++ {
				if zone.Get(m.id + m.groups[i].Name).InBounds(msg) {
					m.cursor = i
					m.openDialog(dialogGroupMembers)
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
			if m.dialogType == dialogGroupMembers {
				switch msg.String() {
				case "esc":
					m.closeDialog()
				case "enter":
					m.submitDialog()
				case "up", "k":
					if m.dialogCursor > 0 {
						m.dialogCursor--
					}
				case "down", "j", "tab":
					if m.dialogCursor < len(m.memberUsers)-1 {
						m.dialogCursor++
					}
				case " ", "space":
					if m.dialogCursor >= 0 && m.dialogCursor < len(m.memberChecks) {
						m.memberChecks[m.dialogCursor] = !m.memberChecks[m.dialogCursor]
					}
				}
				return m, nil
			}
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
					if unicode.IsPrint(r) && m.dialogCursor < len(m.dialogInputs) {
						m.dialogInputs[m.dialogCursor] += string(r)
					}
				}
				return m, nil
			}
		}

		if m.confirmAction != "" {
			switch msg.String() {
			case "y":
				m.executeConfirm()
			case "h":
				if m.confirmAction == "del_user" {
					m.removeHome = !m.removeHome
				}
			default:
				m.confirmAction = ""
				m.confirmArg = ""
				m.removeHome = false
			}
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
			if m.tab == tabUsers {
				m.showSystem = !m.showSystem
				m.users = LoadUsers(m.showSystem)
				m.cursor = 0
			}
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
			if m.tab == tabUsers {
				if m.cursor >= 0 && m.cursor < len(m.users) {
					m.confirmAction = "del_user"
					m.confirmArg = m.users[m.cursor].Login
					m.removeHome = false
				} else {
					m.status = t("status.no_user")
				}
			} else if m.cursor >= 0 && m.cursor < len(m.groups) {
				m.confirmAction = "del_group"
				m.confirmArg = m.groups[m.cursor].Name
			} else {
				m.status = t("status.no_group")
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
				m.openDialog(dialogGroupMembers)
			}
		case "enter":
			if m.tab == tabUsers {
				m.openDialog(dialogUserProps)
			} else {
				m.openDialog(dialogGroupMembers)
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
func (m *tuiModel) executeConfirm() {
	action, name, removeHome := m.confirmAction, m.confirmArg, m.removeHome
	m.confirmAction, m.confirmArg, m.removeHome = "", "", false
	switch action {
	case "del_user":
		output, err := DeleteUser(name, removeHome)
		if err != nil {
			m.status, m.statusOK = t("status.userdel_err")+output+err.Error(), false
		} else {
			m.status, m.statusOK = fmt.Sprintf(t("status.user_deleted"), name), true
		}
		m.users = LoadUsers(m.showSystem)
	case "del_group":
		output, err := DeleteGroup(name)
		if err != nil {
			m.status, m.statusOK = t("status.groupdel_err")+output+err.Error(), false
		} else {
			m.status, m.statusOK = fmt.Sprintf(t("status.group_deleted"), name), true
		}
		m.groups = LoadGroups()
	}
	m.cursor = 0
}

func (m *tuiModel) openDialog(d tuiDialog) {
	m.dialogType = d
	m.dialogCursor = 0
	switch d {
	case dialogAddUser:
		m.dialogInputs = []string{"", "", t("user.add.shell_ph")}
		m.dialogLabel = ""
	case dialogAddGroup:
		m.dialogInputs = []string{""}
		m.dialogLabel = ""
	case dialogPassword:
		m.dialogInputs = []string{"", ""}
		if m.cursor >= 0 && m.cursor < len(m.users) {
			m.dialogLabel = m.users[m.cursor].Login
		} else {
			m.status = t("status.no_user")
			m.dialogType = dialogNone
		}
	case dialogUserProps:
		if m.cursor >= 0 && m.cursor < len(m.users) {
			u := m.users[m.cursor]
			m.dialogInputs = []string{u.Name, u.Shell}
			m.dialogLabel = u.Login
		} else {
			m.status = t("status.no_user")
			m.dialogType = dialogNone
		}
	case dialogGroupMembers:
		if m.cursor >= 0 && m.cursor < len(m.groups) {
			group := m.groups[m.cursor]
			m.dialogLabel = group.Name
			m.memberUsers = LoadUsers(true)
			m.memberChecks = make([]bool, len(m.memberUsers))
			for i, user := range m.memberUsers {
				for _, member := range group.Members {
					if user.Login == member {
						m.memberChecks[i] = true
						break
					}
				}
			}
		} else {
			m.status = t("status.no_group")
			m.dialogType = dialogNone
		}
	}
}

func (m *tuiModel) closeDialog() {
	m.dialogType = dialogNone
	m.dialogInputs = nil
	m.dialogCursor = 0
	m.dialogLabel = ""
	m.memberUsers = nil
	m.memberChecks = nil
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
	case dialogGroupMembers:
		return fmt.Sprintf(t("group.members.title"), m.dialogLabel)
	}
	return ""
}

func (m *tuiModel) submitDialog() {
	switch m.dialogType {
	case dialogAddUser:
		login := m.dialogInputs[0]
		if login == "" {
			m.status, m.statusOK = t("status.useradd_err")+t("dialog.login_required"), false
			m.closeDialog()
			return
		}
		output, err := AddUser(login, m.dialogInputs[1], m.dialogInputs[2])
		if err != nil {
			m.status, m.statusOK = t("status.useradd_err")+output+err.Error(), false
		} else {
			m.status, m.statusOK = fmt.Sprintf(t("status.user_created"), login), true
			m.users = LoadUsers(m.showSystem)
			m.cursor = 0
		}
	case dialogAddGroup:
		name := m.dialogInputs[0]
		if name == "" {
			m.status, m.statusOK = t("status.groupadd_err")+t("dialog.group_required"), false
			m.closeDialog()
			return
		}
		output, err := AddGroup(name)
		if err != nil {
			m.status, m.statusOK = t("status.groupadd_err")+output+err.Error(), false
		} else {
			m.status, m.statusOK = fmt.Sprintf(t("status.group_created"), name), true
			m.groups = LoadGroups()
			m.cursor = 0
		}
	case dialogPassword:
		if m.dialogInputs[0] != m.dialogInputs[1] {
			m.status, m.statusOK = t("status.passwd_mismatch"), false
			m.closeDialog()
			return
		}
		output, err := SetPassword(m.dialogLabel, m.dialogInputs[0])
		if err != nil {
			m.status, m.statusOK = t("status.chpasswd_err")+output+err.Error(), false
		} else {
			m.status, m.statusOK = fmt.Sprintf(t("status.passwd_changed"), m.dialogLabel), true
		}
	case dialogUserProps:
		output, err := SetUserProps(m.dialogLabel, m.dialogInputs[0], m.dialogInputs[1])
		if err != nil {
			m.status, m.statusOK = t("status.usermod_err")+output+err.Error(), false
		} else {
			m.status, m.statusOK = fmt.Sprintf(t("status.user_updated"), m.dialogLabel), true
			m.users = LoadUsers(m.showSystem)
		}
	case dialogGroupMembers:
		var members []string
		for i, user := range m.memberUsers {
			if i < len(m.memberChecks) && m.memberChecks[i] {
				members = append(members, user.Login)
			}
		}
		output, err := SetGroupMembers(m.dialogLabel, members)
		if err != nil {
			m.status, m.statusOK = t("status.groupmod_err")+output+err.Error(), false
		} else {
			m.status, m.statusOK = fmt.Sprintf(t("status.group_updated"), m.dialogLabel), true
			m.groups = LoadGroups()
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
func checkMark(checked bool) string {
	if checked {
		return "x"
	}
	return " "
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
			common.TBtn(m.id, "a_system", fmt.Sprintf("[%s] %s", checkMark(m.showSystem), t("chk.system_users")), "s", false),
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

// statusBar mirrors the GUI footer: About on the left and status on the right.
func (m tuiModel) statusBar() string {
	if m.confirmAction != "" {
		prompt := ""
		switch m.confirmAction {
		case "del_user":
			prompt = tuiDanger.Render("  ⚠ " + fmt.Sprintf(t("confirm.del_user"), m.confirmArg))
			home := common.TBtn(m.id, "confirm_home", fmt.Sprintf("[%s] %s", checkMark(m.removeHome), t("tui.remove_home")), "h", false)
			yes := common.TBtn(m.id, "confirm_yes", t("btn.delete"), "y", false)
			no := common.TBtn(m.id, "confirm_no", t("btn.cancel"), "n", false)
			return prompt + "\n  " + home + " " + yes + " " + no
		case "del_group":
			prompt = tuiDanger.Render("  ⚠ " + fmt.Sprintf(t("confirm.del_group"), m.confirmArg))
			yes := common.TBtn(m.id, "confirm_yes", t("btn.delete"), "y", false)
			no := common.TBtn(m.id, "confirm_no", t("btn.cancel"), "n", false)
			return prompt + "\n  " + yes + " " + no
		}
	}
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
	aboutBtn := common.TBtn(m.id, "a_about", t("btn.about"), "?", false)
	right := statusStyle.Render(msg)
	if !common.TUIEmbedded() {
		right += "  " + common.TBtn(m.id, "a_quit", t("tui.btn.quit"), "q", false)
	}
	return common.TUIFooter(aboutBtn, right, m.width)
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
	headers := []string{t("col.login"), t("col.uid"), t("col.fullname"), t("col.group"), t("col.home")}
	rows := make([][]string, len(m.users))
	for i, u := range m.users {
		rows[i] = []string{u.Login, fmt.Sprint(u.UID), u.Name, u.Primary, u.Home}
	}
	widths := tuiColumnWidths(m.width, headers, rows, 2)
	var sb strings.Builder
	sb.WriteString(tuiTableRow(headers, widths, 2, true) + "\n")

	lh := m.listHeight()
	start := 0
	if m.cursor >= lh {
		start = m.cursor - lh + 1
	}
	shown := 0
	for i := start; i < len(m.users) && shown < lh; i++ {
		line := tuiTableRow(rows[i], widths, 2, false)
		if i == m.cursor {
			sb.WriteString(zone.Mark(m.id+m.users[i].Login, tuiSelected.Render(line)) + "\n")
		} else {
			sb.WriteString(zone.Mark(m.id+m.users[i].Login, line) + "\n")
		}
		shown++
	}
	return sb.String()
}

func (m tuiModel) viewGroups() string {
	if len(m.groups) == 0 {
		return "  " + t("tui.none.groups") + "\n"
	}
	headers := []string{t("col.name"), t("col.gid"), t("col.members")}
	rows := make([][]string, len(m.groups))
	for i, gr := range m.groups {
		rows[i] = []string{gr.Name, fmt.Sprint(gr.GID), strings.Join(gr.Members, ", ")}
	}
	widths := tuiColumnWidths(m.width, headers, rows, 2)
	var sb strings.Builder
	sb.WriteString(tuiTableRow(headers, widths, 2, true) + "\n")

	lh := m.listHeight()
	start := 0
	if m.cursor >= lh {
		start = m.cursor - lh + 1
	}
	shown := 0
	for i := start; i < len(m.groups) && shown < lh; i++ {
		line := tuiTableRow(rows[i], widths, 2, false)
		if i == m.cursor {
			sb.WriteString(zone.Mark(m.id+m.groups[i].Name, tuiSelected.Render(line)) + "\n")
		} else {
			sb.WriteString(zone.Mark(m.id+m.groups[i].Name, line) + "\n")
		}
		shown++
	}
	return sb.String()
}

// tuiColumnWidths sizes columns to fit the terminal while preserving every field.
func tuiColumnWidths(terminalWidth int, headers []string, rows [][]string, indent int) []int {
	n := len(headers)
	widths := make([]int, n)
	maxWidths := make([]int, n)
	for i, header := range headers {
		maxWidths[i] = lipgloss.Width(header)
	}
	for _, row := range rows {
		for i, value := range row {
			if w := lipgloss.Width(value); w > maxWidths[i] {
				maxWidths[i] = w
			}
		}
	}

	gap := 2
	if terminalWidth < indent+n+2*(n-1) {
		gap = 1
		indent = 0
	}
	available := terminalWidth - indent - gap*(n-1)
	if available < n {
		available = n
	}
	remaining := available
	for i := range widths {
		widths[i] = 1
		remaining--
	}
	for remaining > 0 {
		progress := false
		for i := range widths {
			if widths[i] < maxWidths[i] && remaining > 0 {
				widths[i]++
				remaining--
				progress = true
			}
		}
		if !progress {
			break
		}
	}
	return widths
}

func tuiTableRow(values []string, widths []int, indent int, header bool) string {
	gap := 2
	if indent == 0 {
		gap = 1
	}
	var sb strings.Builder
	if indent > 0 {
		sb.WriteString(strings.Repeat(" ", indent))
	}
	for i, value := range values {
		text := tuiTruncate(value, widths[i])
		if header {
			text = lipgloss.NewStyle().Bold(true).Render(text)
		}
		sb.WriteString(text)
		if i < len(values)-1 {
			if padding := widths[i] - lipgloss.Width(tuiTruncate(value, widths[i])); padding > 0 {
				sb.WriteString(strings.Repeat(" ", padding))
			}
			sb.WriteString(strings.Repeat(" ", gap))
		}
	}
	return sb.String()
}

func tuiTruncate(value string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	if width == 1 {
		return "…"
	}
	limit := width - lipgloss.Width("…")
	var sb strings.Builder
	used := 0
	for _, r := range value {
		runeWidth := lipgloss.Width(string(r))
		if used+runeWidth > limit {
			break
		}
		sb.WriteRune(r)
		used += runeWidth
	}
	return sb.String() + "…"
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

func (m tuiModel) renderDialog() string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00AABB"))
	labelStyle := lipgloss.NewStyle().Bold(true)
	focusStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#005577"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9B9B9B"))
	var sb strings.Builder
	sb.WriteString("\n  " + titleStyle.Render(m.dialogTitle()) + "\n\n")
	if m.dialogType == dialogGroupMembers {
		maxRows := m.height - 8
		if maxRows < 1 {
			maxRows = 1
		}
		start := 0
		if m.dialogCursor >= maxRows {
			start = m.dialogCursor - maxRows + 1
		}
		for i := start; i < len(m.memberUsers) && i < start+maxRows; i++ {
			user := m.memberUsers[i]
			line := fmt.Sprintf("    [%s] %s (%d)", checkMark(m.memberChecks[i]), user.Login, user.UID)
			if i == m.dialogCursor {
				line = focusStyle.Render(line)
			}
			sb.WriteString(zone.Mark(m.id+"member_"+user.Login, line) + "\n")
		}
		sb.WriteString("\n    " + hintStyle.Render(t("dialog.members_hint")) + "\n")
	} else {
		for i, label := range m.dialogFieldLabels() {
			value := ""
			if i < len(m.dialogInputs) {
				value = m.dialogInputs[i]
			}
			if m.dialogType == dialogPassword {
				value = strings.Repeat("*", len(value))
			}
			line := "    " + labelStyle.Render(label+": ") + "[" + value + "_]"
			if i == m.dialogCursor {
				line = "    " + focusStyle.Render(label+": ") + focusStyle.Render("["+value+"_]")
			}
			sb.WriteString(zone.Mark(m.id+fmt.Sprintf("field_%d", i), line) + "\n")
		}
		sb.WriteString("\n    " + hintStyle.Render(t("dialog.hint")) + "\n")
	}
	apply := common.TBtn(m.id, "dialog_apply", t("btn.apply"), "enter", false)
	cancel := common.TBtn(m.id, "dialog_cancel", t("btn.cancel"), "esc", false)
	sb.WriteString("    " + apply + " " + cancel + "\n")
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
