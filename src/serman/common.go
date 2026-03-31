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

var Version = common.Version

const (
	AppAuthor  = common.AppAuthor
	AppLicense = common.AppLicense
	AppURL     = common.AppURL
	Usage      = "svman [-g|-t]\n\nOptions:\n  -g, --gui   GUI (default)\n  -t, --tui   TUI\n  -h, --help  show this help\n\nEnvironment:\n  SERVICEDIR      service dir (default: /etc/sv)\n  SERVICEDESTDIR  enabled services dir (default: /var/service)\n  SYSMAN_LANG  language override (e.g. cs)"
)

// ── Defaults ─────────────────────────────────────────────────────────

// Default directories for service definitions and enabled services.
const (
	DefaultServiceDir     = "/etc/sv"      // service definition directory
	DefaultServiceDestDir = "/var/service" // enabled services symlink directory
)

// ── Types ────────────────────────────────────────────────────────────

// Service represents a single runit service with its name and enabled state.
type Service struct {
	Name    string // service name (directory name)
	Enabled bool   // true if symlink exists in destination directory
}

// FilterMode represents the filter state for service/package lists.
type FilterMode int

const (
	FilterAll FilterMode = iota
	FilterEnabled
	FilterDisabled
)

// Filter filters items by state and search query.
// Use FilterAll to show all items, FilterEnabled to show only enabled/installed,
// FilterDisabled to show only disabled/available.
// The isEnabled function should return true for enabled/installed items.
// The matchesSearch function should return true if the item matches the search query.
func Filter[T any](
	items []T,
	mode FilterMode,
	search string,
	isEnabled func(T) bool,
	matchesSearch func(T, string) bool,
) []T {
	return common.Filter(items, int(mode), search,
		func(item T) bool { return isEnabled(item) },
		matchesSearch,
	)
}

// ── Utilities ────────────────────────────────────────────────────────

// isSymlink checks whether the given path is a symbolic link.
// Returns false if the path does not exist or cannot be accessed.
func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

// runElevated runs a command with privilege escalation and returns an error
// that includes the exit code when available.
func runElevated(args ...string) error {
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

// ── Loading ──────────────────────────────────────────────────────────

// LoadServices scans the service directory and returns a sorted list of services.
// Each service's enabled state is determined by checking for a symlink
// in the destination directory.
// Returns nil if the service directory cannot be read.
func LoadServices(serviceDir, destDir string) []Service {
	entries, err := os.ReadDir(serviceDir)
	if err != nil {
		return nil
	}
	var svcs []Service
	for _, e := range entries {
		// skip non-directory entries
		info, err := os.Stat(filepath.Join(serviceDir, e.Name()))
		if err != nil || !info.IsDir() {
			continue
		}
		svcs = append(svcs, Service{
			Name:    e.Name(),
			Enabled: isSymlink(filepath.Join(destDir, e.Name())),
		})
	}
	// sort services alphabetically by name
	sort.Slice(svcs, func(i, j int) bool { return svcs[i].Name < svcs[j].Name })
	return svcs
}

// ── Service Control ──────────────────────────────────────────────────

// EnableService creates a symlink from the service source to the destination,
// enabling the service. Uses sudo to handle permission requirements.
// Returns an error if the symlink creation fails.
func EnableService(serviceDir, destDir, name string) error {
	src := filepath.Join(serviceDir, name)
	dst := filepath.Join(destDir, name)
	args := api.Elevate("ln", "-s", src, dst)
	return runElevated(args...)
}

// DisableService removes the symlink from the destination directory,
// disabling the service. Uses privilege escalation to handle permission requirements.
// Returns an error if the symlink removal fails.
func DisableService(destDir, name string) error {
	dst := filepath.Join(destDir, name)
	args := api.Elevate("rm", dst)
	return runElevated(args...)
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

// GetServiceStatus runs `sv status <path>` for a single service.
// Privilege escalation is used because supervise sockets require root access.
func GetServiceStatus(destDir, name string) ServiceStatus {
	path := filepath.Join(destDir, name)
	args := api.Elevate("sv", "status", path)
	out, _ := exec.Command(args[0], args[1:]...).CombinedOutput() //nolint:gosec
	return parseStatusLine(strings.TrimSpace(string(out)))
}

// GetAllServiceStatuses fetches the status of all given service names in a
// single elevated `sv status` invocation.  This causes only one password
// prompt regardless of how many services are enabled.
// Returns a map of service name → ServiceStatus.
func GetAllServiceStatuses(destDir string, names []string) map[string]ServiceStatus {
	result := make(map[string]ServiceStatus, len(names))
	if len(names) == 0 {
		return result
	}
	paths := make([]string, len(names))
	for i, n := range names {
		paths[i] = filepath.Join(destDir, n)
	}
	args := api.Elevate(append([]string{"sv", "status"}, paths...)...)
	out, _ := exec.Command(args[0], args[1:]...).CombinedOutput() //nolint:gosec
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	// sv outputs one line per path in the same order as arguments.
	for i, line := range lines {
		if i < len(names) && line != "" {
			result[names[i]] = parseStatusLine(line)
		}
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
	// Dirs returns the service definition directory and the enabled-services
	// directory. Used for display purposes (path labels, status header).
	Dirs() (serviceDir, destDir string)
	// List returns all available services with their enabled state.
	List() []Service
	// Enable activates a service (e.g. create symlink for runit).
	Enable(name string) error
	// Disable deactivates a service.
	Disable(name string) error
	// Status returns the live runtime status of an enabled service.
	Status(name string) ServiceStatus
	// StatusAll fetches the status of all given services in one elevated call.
	// Keyed by service name.
	StatusAll(names []string) map[string]ServiceStatus
	// Start starts an enabled service.
	Start(name string) error
	// Stop stops a running service.
	Stop(name string) error
	// Restart restarts a service.
	Restart(name string) error
	// Reload sends SIGHUP (or equivalent) to a service.
	Reload(name string) error
	// Pause suspends a service (SIGSTOP / sv pause).
	Pause(name string) error
	// Continue resumes a paused service (SIGCONT / sv cont).
	Continue(name string) error
	// Kill sends SIGKILL to a service (sv kill).
	Kill(name string) error
}

// ── Runit backend ─────────────────────────────────────────────────────

// RunitBackend implements Backend for the runit init system using the `sv` tool.
type RunitBackend struct {
	ServiceDir string // e.g. /etc/sv
	DestDir    string // e.g. /var/service
}

// NewRunitBackend creates a RunitBackend with the given directories.
func NewRunitBackend(serviceDir, destDir string) *RunitBackend {
	return &RunitBackend{ServiceDir: serviceDir, DestDir: destDir}
}

func (b *RunitBackend) Dirs() (string, string) { return b.ServiceDir, b.DestDir }
func (b *RunitBackend) List() []Service        { return LoadServices(b.ServiceDir, b.DestDir) }

func (b *RunitBackend) Enable(name string) error {
	return EnableService(b.ServiceDir, b.DestDir, name)
}
func (b *RunitBackend) Disable(name string) error { return DisableService(b.DestDir, name) }
func (b *RunitBackend) Status(name string) ServiceStatus {
	return GetServiceStatus(b.DestDir, name)
}
func (b *RunitBackend) StatusAll(names []string) map[string]ServiceStatus {
	return GetAllServiceStatuses(b.DestDir, names)
}
func (b *RunitBackend) Start(name string) error    { return svCmd(b.DestDir, name, "start") }
func (b *RunitBackend) Stop(name string) error     { return svCmd(b.DestDir, name, "stop") }
func (b *RunitBackend) Restart(name string) error  { return svCmd(b.DestDir, name, "restart") }
func (b *RunitBackend) Reload(name string) error   { return svCmd(b.DestDir, name, "reload") }
func (b *RunitBackend) Pause(name string) error    { return svCmd(b.DestDir, name, "pause") }
func (b *RunitBackend) Continue(name string) error { return svCmd(b.DestDir, name, "cont") }
func (b *RunitBackend) Kill(name string) error     { return svCmd(b.DestDir, name, "kill") }

// ── sv control commands ──────────────────────────────────────────────

func svCmd(destDir, name, action string) error {
	path := filepath.Join(destDir, name)
	args := api.Elevate("sv", action, path)
	return runElevated(args...)
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
