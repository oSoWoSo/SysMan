// Package usergroups provides a user and group management plugin.
// It reads /etc/passwd and /etc/group and delegates mutations to
// standard shadow-utils commands (useradd, usermod, userdel, passwd,
// groupadd, groupmod, groupdel) via sudo.
package ugsman

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"codeberg.org/oSoWoSo/SysMan/src/api"
)

// Usage is the --help text for ugman.
const Usage = "ugman [-g|-t]\n\nOptions:\n  -g, --gui   GUI (default)\n  -t, --tui   TUI\n  -h, --help  show this help\n\nEnvironment:\n  SYSMAN_LANG  language override (e.g. cs)"

// ── Data types ────────────────────────────────────────────────────────

// User holds a single /etc/passwd entry.
type User struct {
	Login   string
	UID     int
	GID     int
	Name    string // GECOS full name
	Home    string
	Shell   string
	Primary string // primary group name (resolved from GID)
}

// Group holds a single /etc/group entry.
type Group struct {
	Name    string
	GID     int
	Members []string
}

// ── Reading system databases ──────────────────────────────────────────

// LoadUsers parses /etc/passwd and returns all users sorted by UID.
// If showSystem is false, users with UID < 1000 (and not root) are filtered.
func LoadUsers(showSystem bool) []User {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	groups := gidToName()

	var users []User
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) < 7 {
			continue
		}
		uid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		gid, err := strconv.Atoi(fields[3])
		if err != nil {
			continue
		}
		if !showSystem && uid < 1000 && uid != 0 {
			continue
		}
		gecos := strings.Split(fields[4], ",")
		fullName := ""
		if len(gecos) > 0 {
			fullName = gecos[0]
		}
		users = append(users, User{
			Login:   fields[0],
			UID:     uid,
			GID:     gid,
			Name:    fullName,
			Home:    fields[5],
			Shell:   fields[6],
			Primary: groups[gid],
		})
	}
	sort.Slice(users, func(i, j int) bool { return users[i].UID < users[j].UID })
	return users
}

// LoadGroups parses /etc/group and returns all groups sorted by GID.
func LoadGroups() []Group {
	f, err := os.Open("/etc/group")
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	var groups []Group
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, ":")
		if len(fields) < 4 {
			continue
		}
		gid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		members := []string{}
		if fields[3] != "" {
			members = strings.Split(fields[3], ",")
		}
		groups = append(groups, Group{
			Name:    fields[0],
			GID:     gid,
			Members: members,
		})
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i].GID < groups[j].GID })
	return groups
}

// gidToName builds a map from GID → group name from /etc/group.
func gidToName() map[int]string {
	m := make(map[int]string)
	f, err := os.Open("/etc/group")
	if err != nil {
		return m
	}
	defer func() { _ = f.Close() }()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ":")
		if len(fields) < 3 {
			continue
		}
		gid, err := strconv.Atoi(fields[2])
		if err != nil {
			continue
		}
		m[gid] = fields[0]
	}
	return m
}

// ── Mutations via privilege escalation ───────────────────────────────

// runPriv executes a privileged command and returns combined output.
func runPriv(args ...string) (string, error) {
	elevated := api.Elevate(args...)
	out, err := exec.Command(elevated[0], elevated[1:]...).CombinedOutput() //nolint:gosec
	return strings.TrimSpace(string(out)), err
}

// AddUser creates a new system user.
func AddUser(login, fullName, shell string) (string, error) {
	args := []string{"useradd", "-m"}
	if fullName != "" {
		args = append(args, "-c", fullName)
	}
	if shell != "" {
		args = append(args, "-s", shell)
	}
	args = append(args, login)
	return runPriv(args...)
}

// DeleteUser removes a user (and optionally their home directory).
func DeleteUser(login string, removeHome bool) (string, error) {
	args := []string{"userdel"}
	if removeHome {
		args = append(args, "-r")
	}
	args = append(args, login)
	return runPriv(args...)
}

// SetPassword sets a user's password interactively via passwd.
// Because passwd requires a TTY, we use chpasswd with a provided password.
func SetPassword(login, password string) (string, error) {
	args := api.Elevate("chpasswd")
	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec
	cmd.Stdin = strings.NewReader(fmt.Sprintf("%s:%s\n", login, password))
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// SetUserProps updates full name and shell for a user.
func SetUserProps(login, fullName, shell string) (string, error) {
	args := []string{"usermod"}
	if fullName != "" {
		args = append(args, "-c", fullName)
	}
	if shell != "" {
		args = append(args, "-s", shell)
	}
	args = append(args, login)
	return runPriv(args...)
}

// AddGroup creates a new group.
func AddGroup(name string) (string, error) {
	return runPriv("groupadd", name)
}

// DeleteGroup removes a group.
func DeleteGroup(name string) (string, error) {
	return runPriv("groupdel", name)
}

// SetGroupMembers replaces the member list of a group.
// It diffs the current membership against the desired list and calls
// gpasswd -a / gpasswd -d for each change, because groupmod -M is not
// available on Void Linux (busybox shadow does not implement that flag).
func SetGroupMembers(group string, members []string) (string, error) {
	// Current members from /etc/group.
	current := map[string]bool{}
	for _, g := range LoadGroups() {
		if g.Name == group {
			for _, m := range g.Members {
				current[m] = true
			}
			break
		}
	}

	desired := map[string]bool{}
	for _, m := range members {
		if m != "" {
			desired[m] = true
		}
	}

	var out strings.Builder
	var firstErr error

	// Add new members.
	for m := range desired {
		if !current[m] {
			o, err := runPriv("gpasswd", "-a", m, group)
			if o != "" {
				out.WriteString(o + "\n")
			}
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}

	// Remove members no longer in the list.
	for m := range current {
		if !desired[m] {
			o, err := runPriv("gpasswd", "-d", m, group)
			if o != "" {
				out.WriteString(o + "\n")
			}
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}
	}

	return strings.TrimSpace(out.String()), firstErr
}
