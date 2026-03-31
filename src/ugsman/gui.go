//go:build !tui_only

package ugsman

import (
	"fmt"
	"strings"

	"codeberg.org/oSoWoSo/SysMan/src/common"
	serman "codeberg.org/oSoWoSo/SysMan/src/serman"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ── GUI state ─────────────────────────────────────────────────────────

type ugApp struct {
	win        fyne.Window
	users      []User
	groups     []Group
	showSystem bool
	statusBar  *common.StatusBar

	// Users tab
	userTable    *widget.Table
	selectedUser int // index into g.users, -1 = none

	// Groups tab
	groupTable    *widget.Table
	selectedGroup int // index into g.groups, -1 = none
}

func (g *ugApp) setStatus(msg string) { g.statusBar.SetText(msg) }

func (g *ugApp) showAbout() {
	common.ShowAbout(common.AboutConfig{
		Win:       g.win,
		Title:     t("app.title"),
		Subtitle:  t("app.subtitle"),
		Version:   serman.Version,
		Author:    serman.AppAuthor,
		License:   serman.AppLicense,
		URL:       serman.AppURL,
		DialogBtn: t("btn.about"),
		CloseBtn:  t("btn.close"),
	})
}

func (g *ugApp) refresh() {
	g.users = LoadUsers(g.showSystem)
	g.groups = LoadGroups()
	g.selectedUser = -1
	g.selectedGroup = -1
	g.userTable.Refresh()
	g.groupTable.Refresh()
}

// ── Users table ───────────────────────────────────────────────────────

func userCols() []string {
	return []string{t("col.login"), t("col.uid"), t("col.fullname"), t("col.group"), t("col.home")}
}

func (g *ugApp) buildUserTable() *widget.Table {
	t := widget.NewTable(
		func() (int, int) { return len(g.users) + 1, len(userCols()) },
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("placeholder")
			lbl.TextStyle = fyne.TextStyle{Monospace: true}
			return lbl
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id.Row == 0 {
				lbl.TextStyle = fyne.TextStyle{Bold: true}
				lbl.SetText(userCols()[id.Col])
				return
			}
			lbl.TextStyle = fyne.TextStyle{Monospace: true}
			u := g.users[id.Row-1]
			switch id.Col {
			case 0:
				lbl.SetText(u.Login)
			case 1:
				lbl.SetText(fmt.Sprintf("%d", u.UID))
			case 2:
				lbl.SetText(u.Name)
			case 3:
				lbl.SetText(u.Primary)
			case 4:
				lbl.SetText(u.Home)
			}
		},
	)
	t.SetColumnWidth(0, 120)
	t.SetColumnWidth(1, 60)
	t.SetColumnWidth(2, 160)
	t.SetColumnWidth(3, 120)
	t.SetColumnWidth(4, 240)

	t.OnSelected = func(id widget.TableCellID) {
		if id.Row == 0 {
			g.selectedUser = -1
			return
		}
		g.selectedUser = id.Row - 1
		g.showUserPropsDialog()
	}
	g.userTable = t
	return t
}

// ── Groups table ──────────────────────────────────────────────────────

func groupCols() []string {
	return []string{t("col.name"), t("col.gid"), t("col.members")}
}

func (g *ugApp) buildGroupTable() *widget.Table {
	t := widget.NewTable(
		func() (int, int) { return len(g.groups) + 1, len(groupCols()) },
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("placeholder")
			lbl.TextStyle = fyne.TextStyle{Monospace: true}
			return lbl
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id.Row == 0 {
				lbl.TextStyle = fyne.TextStyle{Bold: true}
				lbl.SetText(groupCols()[id.Col])
				return
			}
			lbl.TextStyle = fyne.TextStyle{Monospace: true}
			gr := g.groups[id.Row-1]
			switch id.Col {
			case 0:
				lbl.SetText(gr.Name)
			case 1:
				lbl.SetText(fmt.Sprintf("%d", gr.GID))
			case 2:
				lbl.SetText(strings.Join(gr.Members, ", "))
			}
		},
	)
	t.SetColumnWidth(0, 160)
	t.SetColumnWidth(1, 60)
	t.SetColumnWidth(2, 400)

	t.OnSelected = func(id widget.TableCellID) {
		if id.Row == 0 {
			g.selectedGroup = -1
			return
		}
		g.selectedGroup = id.Row - 1
		g.showEditGroupMembersDialog()
	}
	g.groupTable = t
	return t
}

// ── Dialogs ───────────────────────────────────────────────────────────

func (g *ugApp) showAddUserDialog() {
	loginEntry := widget.NewEntry()
	loginEntry.SetPlaceHolder(t("user.add.login_ph"))
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder(t("user.add.fullname_ph"))
	shellEntry := widget.NewEntry()
	shellEntry.SetText(t("user.add.shell_ph"))

	form := dialog.NewForm(t("user.add.title"), t("btn.add"), t("btn.cancel"),
		[]*widget.FormItem{
			widget.NewFormItem(t("user.add.login"), loginEntry),
			widget.NewFormItem(t("user.add.fullname"), nameEntry),
			widget.NewFormItem(t("user.add.shell"), shellEntry),
		},
		func(ok bool) {
			if !ok || loginEntry.Text == "" {
				return
			}
			out, err := AddUser(loginEntry.Text, nameEntry.Text, shellEntry.Text)
			if err != nil {
				g.setStatus(t("status.useradd_err") + err.Error() + " " + out)
			} else {
				g.setStatus(fmt.Sprintf(t("status.user_created"), loginEntry.Text))
				g.refresh()
			}
		}, g.win)
	form.Show()
}

func (g *ugApp) showDeleteUserDialog() {
	if g.selectedUser < 0 || g.selectedUser >= len(g.users) {
		g.setStatus(t("status.no_user"))
		return
	}
	u := g.users[g.selectedUser]
	removeHome := widget.NewCheck(t("user.delete.rmhome"), nil)
	dialog.ShowCustomConfirm(
		t("user.delete.title"),
		t("btn.delete"), t("btn.cancel"),
		container.NewVBox(
			widget.NewLabel(fmt.Sprintf(t("user.delete.confirm"), u.Login, u.UID)),
			removeHome,
		),
		func(ok bool) {
			if !ok {
				return
			}
			out, err := DeleteUser(u.Login, removeHome.Checked)
			if err != nil {
				g.setStatus(t("status.userdel_err") + err.Error() + " " + out)
			} else {
				g.setStatus(fmt.Sprintf(t("status.user_deleted"), u.Login))
				g.refresh()
			}
		}, g.win)
}

func (g *ugApp) showUserPropsDialog() {
	if g.selectedUser < 0 || g.selectedUser >= len(g.users) {
		g.setStatus(t("status.no_user"))
		return
	}
	u := g.users[g.selectedUser]
	nameEntry := widget.NewEntry()
	nameEntry.SetText(u.Name)
	shellEntry := widget.NewEntry()
	shellEntry.SetText(u.Shell)

	form := dialog.NewForm(fmt.Sprintf(t("user.props.title"), u.Login), t("btn.apply"), t("btn.cancel"),
		[]*widget.FormItem{
			widget.NewFormItem(t("user.props.fullname"), nameEntry),
			widget.NewFormItem(t("user.props.shell"), shellEntry),
		},
		func(ok bool) {
			if !ok {
				return
			}
			out, err := SetUserProps(u.Login, nameEntry.Text, shellEntry.Text)
			if err != nil {
				g.setStatus(t("status.usermod_err") + err.Error() + " " + out)
			} else {
				g.setStatus(fmt.Sprintf(t("status.user_updated"), u.Login))
				g.refresh()
			}
		}, g.win)
	form.Show()
}

func (g *ugApp) showChangePasswordDialog() {
	if g.selectedUser < 0 || g.selectedUser >= len(g.users) {
		g.setStatus(t("status.no_user"))
		return
	}
	u := g.users[g.selectedUser]
	pwEntry := widget.NewPasswordEntry()
	pw2Entry := widget.NewPasswordEntry()

	form := dialog.NewForm(fmt.Sprintf(t("user.passwd.title"), u.Login), t("btn.apply"), t("btn.cancel"),
		[]*widget.FormItem{
			widget.NewFormItem(t("user.passwd.new"), pwEntry),
			widget.NewFormItem(t("user.passwd.confirm"), pw2Entry),
		},
		func(ok bool) {
			if !ok {
				return
			}
			if pwEntry.Text != pw2Entry.Text {
				g.setStatus(t("status.passwd_mismatch"))
				return
			}
			out, err := SetPassword(u.Login, pwEntry.Text)
			if err != nil {
				g.setStatus(t("status.chpasswd_err") + err.Error() + " " + out)
			} else {
				g.setStatus(fmt.Sprintf(t("status.passwd_changed"), u.Login))
			}
		}, g.win)
	form.Show()
}

func (g *ugApp) showAddGroupDialog() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder(t("group.add.name_ph"))
	form := dialog.NewForm(t("group.add.title"), t("btn.add"), t("btn.cancel"),
		[]*widget.FormItem{widget.NewFormItem(t("group.add.name"), nameEntry)},
		func(ok bool) {
			if !ok || nameEntry.Text == "" {
				return
			}
			out, err := AddGroup(nameEntry.Text)
			if err != nil {
				g.setStatus(t("status.groupadd_err") + err.Error() + " " + out)
			} else {
				g.setStatus(fmt.Sprintf(t("status.group_created"), nameEntry.Text))
				g.refresh()
			}
		}, g.win)
	form.Show()
}

func (g *ugApp) showEditGroupMembersDialog() {
	if g.selectedGroup < 0 || g.selectedGroup >= len(g.groups) {
		g.setStatus(t("status.no_group"))
		return
	}
	gr := g.groups[g.selectedGroup]

	// Build a set of current members for quick lookup.
	current := make(map[string]bool, len(gr.Members))
	for _, m := range gr.Members {
		current[m] = true
	}

	// All human users (UID >= 1000 + root) as candidate members.
	all := LoadUsers(true)
	checks := make([]*widget.Check, len(all))
	items := make([]fyne.CanvasObject, len(all))
	for i, u := range all {
		chk := widget.NewCheck(fmt.Sprintf("%s (%d)", u.Login, u.UID), nil)
		chk.Checked = current[u.Login]
		checks[i] = chk
		items[i] = chk
	}

	scroll := container.NewScroll(container.NewVBox(items...))
	scroll.SetMinSize(fyne.NewSize(300, 240))

	dialog.ShowCustomConfirm(
		fmt.Sprintf(t("group.members.title"), gr.Name),
		t("btn.apply"), t("btn.cancel"),
		scroll,
		func(ok bool) {
			if !ok {
				return
			}
			var members []string
			for i, chk := range checks {
				if chk.Checked {
					members = append(members, all[i].Login)
				}
			}
			out, err := SetGroupMembers(gr.Name, members)
			if err != nil {
				g.setStatus(t("status.groupmod_err") + err.Error() + " " + out)
			} else {
				g.setStatus(fmt.Sprintf(t("status.group_updated"), gr.Name))
				g.refresh()
			}
		}, g.win)
}

func (g *ugApp) showDeleteGroupDialog() {
	if g.selectedGroup < 0 || g.selectedGroup >= len(g.groups) {
		g.setStatus(t("status.no_group"))
		return
	}
	gr := g.groups[g.selectedGroup]
	dialog.ShowConfirm(t("group.delete.title"),
		fmt.Sprintf(t("group.delete.confirm"), gr.Name, gr.GID),
		func(ok bool) {
			if !ok {
				return
			}
			out, err := DeleteGroup(gr.Name)
			if err != nil {
				g.setStatus(t("status.groupdel_err") + err.Error() + " " + out)
			} else {
				g.setStatus(fmt.Sprintf(t("status.group_deleted"), gr.Name))
				g.refresh()
			}
		}, g.win)
}

// ── Build content ─────────────────────────────────────────────────────

// Content builds the widget tree. Implements Plugin.Content.
func (p *Plugin) Content(win fyne.Window) fyne.CanvasObject {
	g := &ugApp{
		win:           win,
		showSystem:    false,
		selectedUser:  -1,
		selectedGroup: -1,
	}
	g.users = LoadUsers(false)
	g.groups = LoadGroups()

	// ── Status bar ────────────────────────────────────────────────────
	g.statusBar = common.NewStatusBar()
	g.statusBar.TextStyle = fyne.TextStyle{Italic: true, Monospace: true}

	// ── Users tab ─────────────────────────────────────────────────────
	userTable := g.buildUserTable()

	showSystemChk := widget.NewCheck(t("chk.system_users"), func(v bool) {
		g.showSystem = v
		g.users = LoadUsers(v)
		g.selectedUser = -1
		userTable.Refresh()
	})

	// Toolbar buttons — Users context
	btnAddUser := common.NewHoverableButton(t("btn.add"), theme.ContentAddIcon(), t("tooltip.add_user"), g.statusBar, func() {
		g.showAddUserDialog()
	})
	btnDelUser := common.NewHoverableButton(t("btn.delete"), theme.DeleteIcon(), t("tooltip.delete_user"), g.statusBar, func() {
		g.showDeleteUserDialog()
	})
	btnPropsUser := common.NewHoverableButton(t("btn.properties"), theme.DocumentCreateIcon(), t("tooltip.user_properties"), g.statusBar, func() {
		g.showUserPropsDialog()
	})
	btnPasswd := common.NewHoverableButton(t("btn.password"), theme.VisibilityIcon(), t("tooltip.change_password"), g.statusBar, func() {
		g.showChangePasswordDialog()
	})
	btnRefreshUsers := common.NewHoverableButton(t("btn.refresh"), theme.ViewRefreshIcon(), t("tooltip.refresh"), g.statusBar, func() {
		g.refresh()
		g.setStatus(t("status.refreshed"))
	})

	userToolbar := container.NewHBox(
		btnAddUser, btnDelUser, btnPropsUser, btnPasswd,
		layout.NewSpacer(),
		btnRefreshUsers,
	)

	usersTab := container.NewBorder(
		container.NewVBox(userToolbar, widget.NewSeparator()),
		container.NewPadded(showSystemChk),
		nil, nil,
		container.NewScroll(userTable),
	)

	// ── Groups tab ────────────────────────────────────────────────────
	groupTable := g.buildGroupTable()

	btnAddGroup := common.NewHoverableButton(t("btn.add"), theme.ContentAddIcon(), t("tooltip.add_group"), g.statusBar, func() {
		g.showAddGroupDialog()
	})
	btnDelGroup := common.NewHoverableButton(t("btn.delete"), theme.DeleteIcon(), t("tooltip.delete_group"), g.statusBar, func() {
		g.showDeleteGroupDialog()
	})
	btnEditMembers := common.NewHoverableButton(t("btn.members"), theme.AccountIcon(), t("tooltip.edit_members"), g.statusBar, func() {
		g.showEditGroupMembersDialog()
	})
	btnRefreshGroups := common.NewHoverableButton(t("btn.refresh"), theme.ViewRefreshIcon(), t("tooltip.refresh"), g.statusBar, func() {
		g.refresh()
		g.setStatus(t("status.refreshed"))
	})

	groupToolbar := container.NewHBox(
		btnAddGroup, btnDelGroup, btnEditMembers,
		layout.NewSpacer(),
		btnRefreshGroups,
	)

	groupsTab := container.NewBorder(
		container.NewVBox(groupToolbar, widget.NewSeparator()),
		nil, nil, nil,
		container.NewScroll(groupTable),
	)

	// ── Tabs ──────────────────────────────────────────────────────────
	tabs := container.NewAppTabs(
		container.NewTabItem(t("tab.users"), usersTab),
		container.NewTabItem(t("tab.groups"), groupsTab),
	)

	btnAbout := common.NewHoverableButton("", theme.InfoIcon(), t("tooltip.about"), g.statusBar, func() { g.showAbout() })
	btnAbout.Button.Importance = widget.LowImportance
	statusBar := container.NewVBox(
		widget.NewSeparator(),
		container.NewPadded(container.NewHBox(btnAbout, g.statusBar)),
	)

	return container.NewBorder(nil, statusBar, nil, nil, tabs)
}

// RunGUI runs the Users & Groups manager as a standalone Fyne application.
func RunGUI() {
	a := app.New()
	win := a.NewWindow(t("app.window"))
	common.SetWindowIcon(win)
	win.SetContent(New().Content(win))
	win.Resize(fyne.NewSize(760, 520))
	win.SetMaster()
	win.Canvas().SetOnTypedKey(func(e *fyne.KeyEvent) {
		if e.Name == fyne.KeyEscape {
			a.Quit()
		}
	})
	win.ShowAndRun()
}
