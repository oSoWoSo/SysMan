package serman

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"codeberg.org/oSoWoSo/SysMan/src/api"
	"codeberg.org/oSoWoSo/SysMan/src/common"
)

// ── App metadata (re-exported from common for backward compatibility) ──

// Version is the serman version.
var Version = common.Version

const (
	// AppAuthor is the author of serman.
	AppAuthor = common.AppAuthor
	// AppLicense is the license of serman.
	AppLicense = common.AppLicense
	// AppURL is the URL of serman.
	AppURL = common.AppURL
	// Usage is the --help text for svman.
	Usage = "svman [-g|-t]\n\nOptions:\n  -g, --gui   GUI (default)\n  -t, --tui   TUI\n  -h, --help  show this help\n\nEnvironment:\n  SERVICEDIR          service dir (default: /etc/sv)\n  SERVICEDESTDIR      enabled services dir (default: /var/service)\n  USER_SERVICEDIR     user service dir (default: ~/.config/service)\n  USER_SERVICEDESTDIR user enabled services dir (default: ~/service)\n  SYSMAN_LANG         language override (e.g. cs)"
)

// ── Defaults ─────────────────────────────────────────────────────────

// DefaultServiceDir is the default service definition directory.
const DefaultServiceDir = "/etc/sv"

// DefaultServiceDestDir is the default enabled services directory.
const DefaultServiceDestDir = "/var/service"

// ── Types ────────────────────────────────────────────────────────────

// Scope identifies whether a service belongs to the system or to a user.
type Scope string

const (
	// ScopeSystem is the system-wide runit scope (/etc/sv, /var/service).
	ScopeSystem Scope = "system"
	// ScopeUser is a per-user runit scope (~/.config/service, ~/service).
	ScopeUser Scope = "user"
)

// Service represents a single runit service with its name and enabled state.
type Service struct {
	Name    string // service name (directory name)
	Enabled bool   // true if symlink exists in destination directory
	Scope   Scope  // system or user scope
}

// Key returns a unique identifier for the service across scopes.
func (s Service) Key() string { return string(s.Scope) + "/" + s.Name }

// ScopeDir holds the directories and elevation policy for a single scope.
type ScopeDir struct {
	Scope      Scope
	ServiceDir string // e.g. /etc/sv or ~/.config/service
	DestDir    string // e.g. /var/service or ~/service
	Elevated   bool   // true if operations need privilege escalation
}

// FilterMode represents the filter state for service/package lists.
type FilterMode int

const (
	// FilterAll selects all services.
	FilterAll FilterMode = iota
	// FilterEnabled selects only enabled services.
	FilterEnabled
	// FilterDisabled selects only disabled services.
	FilterDisabled
)

// ScopeFilter selects which scopes are visible. The System and User toggles
// are independent and combine with the state filter (Filter). With no active
// scope the full list is shown.
type ScopeFilter struct {
	System bool
	User   bool
}

// Active reports whether any scope filter is selected.
func (f ScopeFilter) Active() bool { return f.System || f.User }

// matches reports whether the given service scope passes the filter.
// An empty filter (no scope selected) matches every scope.
func (f ScopeFilter) matches(s Scope) bool {
	if !f.Active() {
		return true
	}
	switch s {
	case ScopeSystem:
		return f.System
	case ScopeUser:
		return f.User
	default:
		return true
	}
}

// Toggle flips the filter state for the given scope.
func (f ScopeFilter) Toggle(s Scope) ScopeFilter {
	switch s {
	case ScopeSystem:
		f.System = !f.System
	case ScopeUser:
		f.User = !f.User
	}
	return f
}

// DefaultScopeFilter resolves the scope shown on first display from the
// serman config (default_scope: "system" | "user", defaulting to system).
// Without a user scope present the filter stays empty (matches everything).
func DefaultScopeFilter(scopes []ScopeDir) ScopeFilter {
	hasUser := false
	for _, sd := range scopes {
		if sd.Scope == ScopeUser {
			hasUser = true
			break
		}
	}
	if !hasUser {
		return ScopeFilter{}
	}
	cfg := common.LoadSysManConfig()
	if cfg.Serman.DefaultScope == string(ScopeUser) {
		return ScopeFilter{User: true}
	}
	return ScopeFilter{System: true}
}

// cfgDefaultScope returns the configured default scope for the settings dialog,
// falling back to the system scope.
func cfgDefaultScope(scopes []ScopeDir) string {
	cfg := common.LoadSysManConfig()
	if cfg.Serman.DefaultScope == string(ScopeUser) {
		return string(ScopeUser)
	}
	return string(ScopeSystem)
}

// filterScoped drops services outside the active scope, then applies the
// state filter and search. The scope axis is applied independently of the
// state filter so every mode (All/Enabled/Disabled) stays within the
// selected scope and never mixes scopes.
func filterScoped(services []Service, mode FilterMode, search string, sf ScopeFilter) []Service {
	scoped := make([]Service, 0, len(services))
	for _, svc := range services {
		if sf.matches(svc.Scope) {
			scoped = append(scoped, svc)
		}
	}
	return common.Filter(scoped, int(mode), search,
		func(svc Service) bool { return svc.Enabled },
		func(svc Service, q string) bool { return strings.Contains(strings.ToLower(svc.Name), q) },
	)
}

// ── Utilities ────────────────────────────────────────────────────────

// isSymlink checks whether the given path is a symbolic link.
// Returns false if the path does not exist or cannot be accessed.
func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

// runCmdRaw runs a command (already including any elevator prefix) and
// returns an error that includes the exit code when available.
func runCmdRaw(args ...string) error {
	cmd := exec.Command(args[0], args[1:]...) //nolint:gosec
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("exit %d: %s", exitErr.ExitCode(), strings.TrimSpace(string(out)))
		}
		return fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// runElevated runs a command with privilege escalation and returns an error
// that includes the exit code when available.
func runElevated(args ...string) error {
	return runCmdRaw(args...)
}

// runCapture runs a command, prefixing it with the elevator when elevate is
// true, and returns the captured output.
func runCapture(elevate bool, args ...string) string {
	if elevate {
		args = api.Elevate(args...)
	}
	out, _ := exec.Command(args[0], args[1:]...).CombinedOutput() //nolint:gosec
	return string(out)
}

// ── Loading ──────────────────────────────────────────────────────────

// LoadServices scans the service directory and returns a sorted list of services.
// Each service's enabled state is determined by checking for a symlink
// in the destination directory.
// Entries that only exist in the destination (live) directory — the per-user
// runsvdir/turnstile layout, where run directories or symlinks live directly
// there — are listed as enabled services too.
// Returns nil if neither directory can be read.
func LoadServices(serviceDir, destDir string) []Service {
	var svcs []Service
	seen := make(map[string]bool)
	if entries, err := os.ReadDir(serviceDir); err == nil {
		for _, e := range entries {
			info, err := os.Stat(filepath.Join(serviceDir, e.Name()))
			if err != nil || !info.IsDir() {
				continue
			}
			svcs = append(svcs, Service{
				Name:    e.Name(),
				Enabled: isSymlink(filepath.Join(destDir, e.Name())),
			})
			seen[e.Name()] = true
		}
	}
	if entries, err := os.ReadDir(destDir); err == nil {
		for _, e := range entries {
			if seen[e.Name()] {
				continue
			}
			info, err := os.Lstat(filepath.Join(destDir, e.Name()))
			if err != nil || (!info.IsDir() && info.Mode()&os.ModeSymlink == 0) {
				continue
			}
			svcs = append(svcs, Service{Name: e.Name(), Enabled: true})
		}
	}
	if len(svcs) == 0 {
		return nil
	}
	// sort services alphabetically by name
	sort.Slice(svcs, func(i, j int) bool { return svcs[i].Name < svcs[j].Name })
	return svcs
}

// LoadServicesScoped loads services for a single scope and tags each service
// with that scope. Returns nil if the service directory cannot be read.
func LoadServicesScoped(sd ScopeDir) []Service {
	svcs := LoadServices(sd.ServiceDir, sd.DestDir)
	for i := range svcs {
		svcs[i].Scope = sd.Scope
	}
	return svcs
}

// ── Service Control ──────────────────────────────────────────────────

// enableService creates a symlink from the service source to the destination,
// enabling the service. Uses privilege escalation when elevate is true.
func enableService(serviceDir, destDir, name string, elevate bool) error {
	src := filepath.Join(serviceDir, name)
	if src == filepath.Dir(src) || serviceDir == "" {
		return fmt.Errorf("service dir not set: cannot enable %q", name)
	}
	dst := filepath.Join(destDir, name)
	args := []string{"ln", "-s", src, dst}
	if elevate {
		args = api.Elevate(args...)
	}
	return runCmdRaw(args...)
}

// EnableService creates a symlink from the service source to the destination,
// enabling the service. Uses privilege escalation to handle permission requirements.
// Returns an error if the symlink creation fails.
func EnableService(serviceDir, destDir, name string) error {
	return enableService(serviceDir, destDir, name, true)
}

// disableService removes the symlink from the destination directory,
// disabling the service. Uses privilege escalation when elevate is true.
func disableService(destDir, name string, elevate bool) error {
	dst := filepath.Join(destDir, name)
	args := []string{"rm", dst}
	if elevate {
		args = api.Elevate(args...)
	}
	return runCmdRaw(args...)
}

// DisableService removes the symlink from the destination directory,
// disabling the service. Uses privilege escalation to handle permission requirements.
// Returns an error if the symlink removal fails.
func DisableService(destDir, name string) error {
	return disableService(destDir, name, true)
}

// ── Runtime status ───────────────────────────────────────────────────

// ServiceStatus holds the live runtime status from `sv status`.
type ServiceStatus struct {
	Running bool
	PID     int
	Uptime  string // e.g., "42s", "5m"
	Raw     string // full sv output line
}

// parseStatusLine parses a single `sv status` output line into ServiceStatus.
func parseStatusLine(line string) ServiceStatus {
	s := ServiceStatus{Raw: line}
	s.Running = strings.HasPrefix(line, "run:")
	if i := strings.Index(line, "(pid "); i >= 0 {
		rest := line[i+5:]
		if j := strings.Index(rest, ")"); j >= 0 {
			fmt.Sscanf(rest[:j], "%d", &s.PID) //nolint:errcheck
		}
	}
	if i := strings.LastIndex(line, ") "); i >= 0 {
		rest := strings.TrimSpace(line[i+2:])
		if j := strings.Index(rest, ";"); j >= 0 {
			rest = rest[:j]
		}
		s.Uptime = strings.TrimSpace(rest)
	}
	return s
}

// getServiceStatus runs `sv status <path>` for a single service.
// Privilege escalation is used when elevate is true (system scope).
func getServiceStatus(destDir, name string, elevate bool) ServiceStatus {
	path := filepath.Join(destDir, name)
	out := runCapture(elevate, "sv", "status", path)
	return parseStatusLine(strings.TrimSpace(out))
}

// GetServiceStatus runs `sv status <path>` for a single service.
// Privilege escalation is used because supervise sockets require root access.
func GetServiceStatus(destDir, name string) ServiceStatus {
	return getServiceStatus(destDir, name, true)
}

// getAllServiceStatuses fetches the status of the given services in a single
// `sv status` invocation for one scope. This causes only one password prompt
// per scope regardless of how many services are enabled.
// Returns a map of service Key → ServiceStatus.
func getAllServiceStatuses(sd ScopeDir, svcs []Service) map[string]ServiceStatus {
	result := make(map[string]ServiceStatus, len(svcs))
	if len(svcs) == 0 {
		return result
	}
	paths := make([]string, len(svcs))
	for i, svc := range svcs {
		paths[i] = filepath.Join(sd.DestDir, svc.Name)
	}
	args := append([]string{"sv", "status"}, paths...)
	out := runCapture(sd.Elevated, args...)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	// sv outputs one line per path in the same order as arguments.
	for i, line := range lines {
		if i < len(svcs) && line != "" {
			result[svcs[i].Key()] = parseStatusLine(line)
		}
	}
	return result
}

// GetAllServiceStatuses fetches the status of all given service names in a
// single elevated `sv status` invocation.  This causes only one password
// prompt regardless of how many services are enabled.
// Returns a map of service name → ServiceStatus.
func GetAllServiceStatuses(destDir string, names []string) map[string]ServiceStatus {
	svcs := make([]Service, len(names))
	for i, n := range names {
		svcs[i] = Service{Name: n, Scope: ScopeSystem}
	}
	sd := ScopeDir{Scope: ScopeSystem, DestDir: destDir, Elevated: true}
	byKey := getAllServiceStatuses(sd, svcs)
	result := make(map[string]ServiceStatus, len(byKey))
	for _, n := range names {
		result[n] = byKey["system/"+n]
	}
	return result
}

// ── Backend interface ─────────────────────────────────────────────────
//
// Backend abstracts service manager operations so the UI is independent
// of the underlying init system. Implement this interface to add support
// for openrc, s6, systemd, or any other service manager.
//
// The runit implementation (RunitBackend) is the default.

// Backend is the interface every service manager backend must implement.
type Backend interface {
	// ScopeDirs returns the configured scope directories. Used for display.
	ScopeDirs() []ScopeDir
	// Dirs returns the service definition directory and the enabled-services
	// directory for the given service's scope. Used for display purposes.
	Dirs(svc Service) (serviceDir, destDir string)
	// List returns all available services with their enabled state and scope.
	List() []Service
	// Enable activates a service (e.g. create symlink for runit).
	Enable(svc Service) error
	// Disable deactivates a service.
	Disable(svc Service) error
	// Status returns the live runtime status of an enabled service.
	Status(svc Service) ServiceStatus
	// StatusAll fetches the status of all given services in one call per scope.
	// Keyed by Service.Key().
	StatusAll(svcs []Service) map[string]ServiceStatus
	// Start starts an enabled service.
	Start(svc Service) error
	// Stop stops a running service.
	Stop(svc Service) error
	// Restart restarts a service.
	Restart(svc Service) error
	// Reload sends SIGHUP (or equivalent) to a service.
	Reload(svc Service) error
	// Pause suspends a service (SIGSTOP / sv pause).
	Pause(svc Service) error
	// Continue resumes a paused service (SIGCONT / sv cont).
	Continue(svc Service) error
	// Kill sends SIGKILL to a service (sv kill).
	Kill(svc Service) error
}

// ── Runit backend ─────────────────────────────────────────────────────

// RunitBackend implements Backend for the runit init system using the `sv` tool.
type RunitBackend struct {
	Scopes []ScopeDir
}

// NewRunitBackend creates a RunitBackend with the given system directories.
func NewRunitBackend(serviceDir, destDir string) *RunitBackend {
	return &RunitBackend{Scopes: []ScopeDir{{
		Scope:      ScopeSystem,
		ServiceDir: serviceDir,
		DestDir:    destDir,
		Elevated:   true,
	}}}
}

// NewScopedRunitBackend creates a RunitBackend covering both the system and
// user scopes. A scope whose directories are both empty is omitted.
func NewScopedRunitBackend(system, user ScopeDir) *RunitBackend {
	scopes := []ScopeDir{system}
	if user.ServiceDir != "" || user.DestDir != "" {
		scopes = append(scopes, user)
	}
	return &RunitBackend{Scopes: scopes}
}

// scopeFor resolves the ScopeDir owning the given service. Services without a
// scope (zero-value Scope) fall back to the first (system) scope.
func (b *RunitBackend) scopeFor(svc Service) ScopeDir {
	for _, sd := range b.Scopes {
		if sd.Scope == svc.Scope {
			return sd
		}
	}
	if len(b.Scopes) > 0 {
		return b.Scopes[0]
	}
	return ScopeDir{Scope: ScopeSystem, ServiceDir: DefaultServiceDir, DestDir: DefaultServiceDestDir, Elevated: true}
}

// ScopeDirs returns the configured scope directories.
func (b *RunitBackend) ScopeDirs() []ScopeDir { return b.Scopes }

// Dirs returns the service and destination directories for the service scope.
func (b *RunitBackend) Dirs(svc Service) (string, string) {
	sd := b.scopeFor(svc)
	return sd.ServiceDir, sd.DestDir
}

// List returns all services across scopes, system first then user.
func (b *RunitBackend) List() []Service {
	var all []Service
	for _, sd := range b.Scopes {
		all = append(all, LoadServicesScoped(sd)...)
	}
	return all
}

// Enable enables a service.
func (b *RunitBackend) Enable(svc Service) error {
	sd := b.scopeFor(svc)
	return enableService(sd.ServiceDir, sd.DestDir, svc.Name, sd.Elevated)
}

// Disable disables a service.
func (b *RunitBackend) Disable(svc Service) error {
	sd := b.scopeFor(svc)
	return disableService(sd.DestDir, svc.Name, sd.Elevated)
}

// Status returns the status of a service.
func (b *RunitBackend) Status(svc Service) ServiceStatus {
	sd := b.scopeFor(svc)
	return getServiceStatus(sd.DestDir, svc.Name, sd.Elevated)
}

// StatusAll returns the status of all services across scopes, keyed by Service.Key().
func (b *RunitBackend) StatusAll(svcs []Service) map[string]ServiceStatus {
	result := make(map[string]ServiceStatus, len(svcs))
	for _, sd := range b.Scopes {
		var group []Service
		for _, svc := range svcs {
			if b.scopeFor(svc).Scope == sd.Scope {
				group = append(group, svc)
			}
		}
		for k, v := range getAllServiceStatuses(sd, group) {
			result[k] = v
		}
	}
	return result
}

// Start starts a service.
func (b *RunitBackend) Start(svc Service) error {
	sd := b.scopeFor(svc)
	return svCmdScope(sd, svc.Name, "start")
}

// Stop stops a service.
func (b *RunitBackend) Stop(svc Service) error {
	sd := b.scopeFor(svc)
	return svCmdScope(sd, svc.Name, "stop")
}

// Restart restarts a service.
func (b *RunitBackend) Restart(svc Service) error {
	sd := b.scopeFor(svc)
	return svCmdScope(sd, svc.Name, "restart")
}

// Reload reloads a service.
func (b *RunitBackend) Reload(svc Service) error {
	sd := b.scopeFor(svc)
	return svCmdScope(sd, svc.Name, "reload")
}

// Pause pauses a service.
func (b *RunitBackend) Pause(svc Service) error {
	sd := b.scopeFor(svc)
	return svCmdScope(sd, svc.Name, "pause")
}

// Continue continues a service.
func (b *RunitBackend) Continue(svc Service) error {
	sd := b.scopeFor(svc)
	return svCmdScope(sd, svc.Name, "cont")
}

// Kill kills a service.
func (b *RunitBackend) Kill(svc Service) error {
	sd := b.scopeFor(svc)
	return svCmdScope(sd, svc.Name, "kill")
}

// ── sv control commands ──────────────────────────────────────────────

// svCmdScope runs `sv <action> <path>` for a service in the given scope,
// escalating privileges only when the scope requires it.
func svCmdScope(sd ScopeDir, name, action string) error {
	path := filepath.Join(sd.DestDir, name)
	args := []string{"sv", action, path}
	if sd.Elevated {
		args = api.Elevate(args...)
	}
	return runCmdRaw(args...)
}

// svCmd runs `sv <action> <path>` with privilege escalation.
func svCmd(destDir, name, action string) error {
	return svCmdScope(ScopeDir{Scope: ScopeSystem, DestDir: destDir, Elevated: true}, name, action)
}

// StartService starts an enabled service via `sv start`.
func StartService(destDir, name string) error { return svCmd(destDir, name, "start") }

// StopService stops an enabled service via `sv stop`.
func StopService(destDir, name string) error { return svCmd(destDir, name, "stop") }

// RestartService restarts an enabled service via `sv restart`.
func RestartService(destDir, name string) error { return svCmd(destDir, name, "restart") }

// ReloadService sends SIGHUP to an enabled service via `sv reload`.
func ReloadService(destDir, name string) error { return svCmd(destDir, name, "reload") }

// PauseService sends SIGSTOP to an enabled service via `sv pause`.
func PauseService(destDir, name string) error { return svCmd(destDir, name, "pause") }

// ContinueService sends SIGCONT to a paused service via `sv cont`.
func ContinueService(destDir, name string) error { return svCmd(destDir, name, "cont") }

// KillService sends SIGKILL to a service via `sv kill`.
func KillService(destDir, name string) error { return svCmd(destDir, name, "kill") }

// ── Scope resolution ─────────────────────────────────────────────────

// DefaultUserServiceDir returns the default user service definition directory.
func DefaultUserServiceDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "service")
}

// DefaultUserServiceDestDir returns the default user enabled services directory.
func DefaultUserServiceDestDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "service")
}

func expandHome(p string) string {
	if p == "" {
		return p
	}
	if p == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		return home
	}
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		return filepath.Join(home, p[2:])
	}
	return p
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// ResolveScopes determines the system and user scope directories using the
// priority env > config > default. User scope is included only when the
// user service directory exists.
func ResolveScopes() (system, user ScopeDir) {
	cfg := common.LoadSysManConfig()

	system = ScopeDir{
		Scope:      ScopeSystem,
		ServiceDir: expandHome(firstNonEmpty(os.Getenv("SERVICEDIR"), cfg.Serman.ServiceDir, DefaultServiceDir)),
		DestDir:    expandHome(firstNonEmpty(os.Getenv("SERVICEDESTDIR"), cfg.Serman.ServiceDestDir, DefaultServiceDestDir)),
		Elevated:   true,
	}

	user = ScopeDir{
		Scope:      ScopeUser,
		ServiceDir: expandHome(firstNonEmpty(os.Getenv("USER_SERVICEDIR"), cfg.Serman.UserServiceDir, DefaultUserServiceDir())),
		DestDir:    expandHome(firstNonEmpty(os.Getenv("USER_SERVICEDESTDIR"), cfg.Serman.UserServiceDestDir, DefaultUserServiceDestDir())),
		Elevated:   false,
	}

	if !userScopeActive(user.ServiceDir, user.DestDir) {
		user.ServiceDir = ""
		user.DestDir = ""
	}
	return system, user
}

// userScopeActive reports whether a user scope is worth activating: either its
// service (definitions) directory or its live services directory exists. The
// live-dir-only case covers the per-user runsvdir/turnstile layout, where run
// directories or symlinks live directly in the enabled-services directory.
func userScopeActive(serviceDir, destDir string) bool {
	return dirExists(serviceDir) || dirExists(destDir)
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}
