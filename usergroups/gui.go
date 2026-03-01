//go:build !tui_only

package usergroups

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
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
	statusBar  *widget.Label

	// Users tab
	userTable    *widget.Table
	selectedUser int // index into g.users, -1 = none

	// Groups tab
	groupTable    *widget.Table
	selectedGroup int // index into g.groups, -1 = none
}

func (g *ugApp) setStatus(msg string) { g.statusBar.SetText(msg) }

func (g *ugApp) refresh() {
	g.users = LoadUsers(g.showSystem)
	g.groups = LoadGroups()
	g.selectedUser = -1
	g.selectedGroup = -1
	g.userTable.Refresh()
	g.groupTable.Refresh()
}

// ── Users table ───────────────────────────────────────────────────────

var userCols = []string{"Login", "UID", "Full Name", "Group", "Home"}

func (g *ugApp) buildUserTable() *widget.Table {
	t := widget.NewTable(
		func() (int, int) { return len(g.users) + 1, len(userCols) },
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("placeholder")
			lbl.TextStyle = fyne.TextStyle{Monospace: true}
			return lbl
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id.Row == 0 {
				lbl.TextStyle = fyne.TextStyle{Bold: true}
				lbl.SetText(userCols[id.Col])
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
	}
	g.userTable = t
	return t
}

// ── Groups table ──────────────────────────────────────────────────────

var groupCols = []string{"Name", "GID", "Members"}

func (g *ugApp) buildGroupTable() *widget.Table {
	t := widget.NewTable(
		func() (int, int) { return len(g.groups) + 1, len(groupCols) },
		func() fyne.CanvasObject {
			lbl := widget.NewLabel("placeholder")
			lbl.TextStyle = fyne.TextStyle{Monospace: true}
			return lbl
		},
		func(id widget.TableCellID, obj fyne.CanvasObject) {
			lbl := obj.(*widget.Label)
			if id.Row == 0 {
				lbl.TextStyle = fyne.TextStyle{Bold: true}
				lbl.SetText(groupCols[id.Col])
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
	}
	g.groupTable = t
	return t
}

// ── Dialogs ───────────────────────────────────────────────────────────

func (g *ugApp) showAddUserDialog() {
	loginEntry := widget.NewEntry()
	loginEntry.SetPlaceHolder("login name")
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("Full Name")
	shellEntry := widget.NewEntry()
	shellEntry.SetText("/bin/bash")

	form := dialog.NewForm("Add User", "Add", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Login", loginEntry),
			widget.NewFormItem("Full Name", nameEntry),
			widget.NewFormItem("Shell", shellEntry),
		},
		func(ok bool) {
			if !ok || loginEntry.Text == "" {
				return
			}
			out, err := AddUser(loginEntry.Text, nameEntry.Text, shellEntry.Text)
			if err != nil {
				g.setStatus("✗ useradd: " + err.Error() + " " + out)
			} else {
				g.setStatus("✓ User " + loginEntry.Text + " created")
				g.refresh()
			}
		}, g.win)
	form.Show()
}

func (g *ugApp) showDeleteUserDialog() {
	if g.selectedUser < 0 || g.selectedUser >= len(g.users) {
		g.setStatus("No user selected")
		return
	}
	u := g.users[g.selectedUser]
	removeHome := widget.NewCheck("Remove home directory", nil)
	dialog.ShowCustomConfirm(
		"Delete User",
		"Delete", "Cancel",
		container.NewVBox(
			widget.NewLabel("Delete user: "+u.Login+" (UID "+fmt.Sprintf("%d", u.UID)+")"),
			removeHome,
		),
		func(ok bool) {
			if !ok {
				return
			}
			out, err := DeleteUser(u.Login, removeHome.Checked)
			if err != nil {
				g.setStatus("✗ userdel: " + err.Error() + " " + out)
			} else {
				g.setStatus("✓ User " + u.Login + " deleted")
				g.refresh()
			}
		}, g.win)
}

func (g *ugApp) showUserPropsDialog() {
	if g.selectedUser < 0 || g.selectedUser >= len(g.users) {
		g.setStatus("No user selected")
		return
	}
	u := g.users[g.selectedUser]
	nameEntry := widget.NewEntry()
	nameEntry.SetText(u.Name)
	shellEntry := widget.NewEntry()
	shellEntry.SetText(u.Shell)

	form := dialog.NewForm("User Properties: "+u.Login, "Apply", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("Full Name", nameEntry),
			widget.NewFormItem("Shell", shellEntry),
		},
		func(ok bool) {
			if !ok {
				return
			}
			out, err := SetUserProps(u.Login, nameEntry.Text, shellEntry.Text)
			if err != nil {
				g.setStatus("✗ usermod: " + err.Error() + " " + out)
			} else {
				g.setStatus("✓ User " + u.Login + " updated")
				g.refresh()
			}
		}, g.win)
	form.Show()
}

func (g *ugApp) showChangePasswordDialog() {
	if g.selectedUser < 0 || g.selectedUser >= len(g.users) {
		g.setStatus("No user selected")
		return
	}
	u := g.users[g.selectedUser]
	pwEntry := widget.NewPasswordEntry()
	pw2Entry := widget.NewPasswordEntry()

	form := dialog.NewForm("Change Password: "+u.Login, "Apply", "Cancel",
		[]*widget.FormItem{
			widget.NewFormItem("New password", pwEntry),
			widget.NewFormItem("Confirm", pw2Entry),
		},
		func(ok bool) {
			if !ok {
				return
			}
			if pwEntry.Text != pw2Entry.Text {
				g.setStatus("✗ Passwords do not match")
				return
			}
			out, err := SetPassword(u.Login, pwEntry.Text)
			if err != nil {
				g.setStatus("✗ chpasswd: " + err.Error() + " " + out)
			} else {
				g.setStatus("✓ Password for " + u.Login + " changed")
			}
		}, g.win)
	form.Show()
}

func (g *ugApp) showAddGroupDialog() {
	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("group name")
	form := dialog.NewForm("Add Group", "Add", "Cancel",
		[]*widget.FormItem{widget.NewFormItem("Name", nameEntry)},
		func(ok bool) {
			if !ok || nameEntry.Text == "" {
				return
			}
			out, err := AddGroup(nameEntry.Text)
			if err != nil {
				g.setStatus("✗ groupadd: " + err.Error() + " " + out)
			} else {
				g.setStatus("✓ Group " + nameEntry.Text + " created")
				g.refresh()
			}
		}, g.win)
	form.Show()
}

func (g *ugApp) showEditGroupMembersDialog() {
	if g.selectedGroup < 0 || g.selectedGroup >= len(g.groups) {
		g.setStatus("No group selected")
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
		"Edit Members: "+gr.Name,
		"Apply", "Cancel",
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
				g.setStatus("✗ groupmod: " + err.Error() + " " + out)
			} else {
				g.setStatus("✓ Group " + gr.Name + " members updated")
				g.refresh()
			}
		}, g.win)
}

func (g *ugApp) showDeleteGroupDialog() {
	if g.selectedGroup < 0 || g.selectedGroup >= len(g.groups) {
		g.setStatus("No group selected")
		return
	}
	gr := g.groups[g.selectedGroup]
	dialog.ShowConfirm("Delete Group",
		"Delete group: "+gr.Name+" (GID "+fmt.Sprintf("%d", gr.GID)+")?",
		func(ok bool) {
			if !ok {
				return
			}
			out, err := DeleteGroup(gr.Name)
			if err != nil {
				g.setStatus("✗ groupdel: " + err.Error() + " " + out)
			} else {
				g.setStatus("✓ Group " + gr.Name + " deleted")
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
	g.statusBar = widget.NewLabel("")
	g.statusBar.TextStyle = fyne.TextStyle{Italic: true, Monospace: true}

	// ── Users tab ─────────────────────────────────────────────────────
	userTable := g.buildUserTable()

	showSystemChk := widget.NewCheck("Show system users", func(v bool) {
		g.showSystem = v
		g.users = LoadUsers(v)
		g.selectedUser = -1
		userTable.Refresh()
	})

	// Toolbar buttons — Users context
	btnAddUser := widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), func() {
		g.showAddUserDialog()
	})
	btnDelUser := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		g.showDeleteUserDialog()
	})
	btnPropsUser := widget.NewButtonWithIcon("Properties", theme.DocumentCreateIcon(), func() {
		g.showUserPropsDialog()
	})
	btnPasswd := widget.NewButtonWithIcon("Change Password", theme.VisibilityIcon(), func() {
		g.showChangePasswordDialog()
	})
	btnRefreshUsers := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		g.refresh()
		g.setStatus("Refreshed")
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

	btnAddGroup := widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), func() {
		g.showAddGroupDialog()
	})
	btnDelGroup := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		g.showDeleteGroupDialog()
	})
	btnEditMembers := widget.NewButtonWithIcon("Members", theme.AccountIcon(), func() {
		g.showEditGroupMembersDialog()
	})
	btnRefreshGroups := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() {
		g.refresh()
		g.setStatus("Refreshed")
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
		container.NewTabItem("Users", usersTab),
		container.NewTabItem("Groups", groupsTab),
	)

	statusBar := container.NewVBox(
		widget.NewSeparator(),
		container.NewPadded(g.statusBar),
	)

	return container.NewBorder(nil, statusBar, nil, nil, tabs)
}
