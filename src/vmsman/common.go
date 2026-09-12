// Package vmman provides a VM manager plugin.
package vmman

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"codeberg.org/oSoWoSo/SysMan/src/common"
	"github.com/creack/pty"
	"gopkg.in/yaml.v3"
)

// Version is the application version.
var Version = common.Version

const (
	// AppAuthor is the application author.
	AppAuthor = common.AppAuthor
	// AppLicense is the application license.
	AppLicense = common.AppLicense
	// AppURL is the application URL.
	AppURL = "https://codeberg.org/oSoWoSo/VMman"
	// Usage is the command line usage information.
	Usage = "vmman [-g|-t] [--vm NAME] [--port PORT]\n\nOptions:\n  -g, --gui     GUI (default)\n  -t, --tui     TUI\n  --vm NAME     VM name (from config)\n  --port PORT   SPICE port (auto-detected if not provided)\n  -h, --help    show this help\n\nEnvironment:\n  VMDIR          VM directory (default: ~/vm)\n  SYSMAN_LANG    language override (e.g. cs)"
)

// DefaultVMDir is the default VM directory.
const DefaultVMDir = "vm"

// ResolveVMDir expands ~ in the given path and falls back to ~/vm.
func ResolveVMDir(vmDir string) string {
	if vmDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return DefaultVMDir
		}
		return filepath.Join(home, DefaultVMDir)
	}
	if strings.HasPrefix(vmDir, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, vmDir[2:])
		}
	}
	return vmDir
}

// VM represents a virtual machine.
type VM struct {
	Name      string
	Config    string
	Disk      string
	ISO       string
	PID       int
	SPICEPort int
	SSHPort   int
	SSHUser   string
	Running   bool
}

// VMStatus represents the status of a virtual machine.
type VMStatus struct {
	Running   bool
	PID       int
	SPICEPort int
	SSHPort   int
	Display   string
	Uptime    string
	Raw       string
}

// FilterMode represents the filter mode for VMs.
type FilterMode int

const (
	// FilterAll represents filtering all VMs.
	FilterAll FilterMode = iota
	// FilterRunning represents filtering running VMs.
	FilterRunning
	// FilterStopped represents filtering stopped VMs.
	FilterStopped
)

// Filter filters VMs by mode and search term.
func Filter[T any](
	items []T,
	mode FilterMode,
	search string,
	isRunning func(T) bool,
	matchesSearch func(T, string) bool,
) []T {
	return common.Filter(items, int(mode), search, isRunning, matchesSearch)
}

// CheckVMDir checks if the VM directory exists and is accessible.
// Returns an error message if the directory doesn't exist or can't be read.
func CheckVMDir(vmDir string) string {
	if vmDir == "" {
		return t("error.vm_dir_empty")
	}
	info, err := os.Stat(vmDir)
	if os.IsNotExist(err) {
		return fmt.Sprintf(t("error.vm_dir_not_found"), vmDir)
	}
	if err != nil {
		return fmt.Sprintf(t("error.vm_dir_access"), vmDir)
	}
	if !info.IsDir() {
		return fmt.Sprintf(t("error.vm_dir_not_dir"), vmDir)
	}
	return ""
}

// diskImgDir reads the disk_img="..." line from a VM config file and returns
// its directory component ("" when the disk sits directly in the VM dir).
// quickemu/dh stores all VM state files in the disk image's directory, so this
// mirrors their layout (`VMDIR=$(dirname "${disk_img}")`).
func diskImgDir(configPath string) string {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`(?m)^\s*disk_img\s*=\s*"([^"]+)"`)
	m := re.FindStringSubmatch(string(data))
	if len(m) != 2 {
		return ""
	}
	d := filepath.Dir(m[1])
	if d == "" || d == "." || d == "/" {
		return ""
	}
	return d
}

// confValue reads a single quoted key="value" setting from a VM config file.
func confValue(configPath, key string) string {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(key) + `\s*=\s*"([^"]*)"`)
	m := re.FindStringSubmatch(string(data))
	if len(m) != 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

// vmStatePaths returns the pid and ports file paths for a VM. The quickemu/dh
// per-VM directory layout (<vmDir>/<name>/<name>.pid) is preferred; the flat
// layout (<vmDir>/<name>.pid) is used for reading legacy and SysMan-created
// VMs.
func vmStatePaths(vmDir string, vm VM) (pidPath, portsPath string) {
	stateDir := filepath.Join(vmDir, vm.Name)
	if d := diskImgDir(vm.Config); d != "" {
		stateDir = filepath.Join(vmDir, d)
	}
	pidPath = filepath.Join(stateDir, vm.Name+".pid")
	if _, err := os.Stat(pidPath); err != nil {
		if _, err := os.Stat(filepath.Join(vmDir, vm.Name+".pid")); err == nil {
			pidPath = filepath.Join(vmDir, vm.Name+".pid")
		}
	}
	portsPath = filepath.Join(stateDir, vm.Name+".ports")
	return pidPath, portsPath
}

// LoadVMs loads VMs from the specified directory.
// Returns nil if the directory doesn't exist or can't be read.
func LoadVMs(vmDir string) []VM {
	entries, err := os.ReadDir(vmDir)
	if err != nil {
		return nil
	}
	var vms []VM
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".conf") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".conf")
		configPath := filepath.Join(vmDir, e.Name())

		vm := VM{
			Name:   name,
			Config: configPath,
		}
		if u := confValue(configPath, "ssh_user"); u != "" {
			vm.SSHUser = u
		}

		pidPath, _ := vmStatePaths(vmDir, vm)
		if data, err := os.ReadFile(pidPath); err == nil {
			_, _ = fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &vm.PID)
			if vm.PID > 0 {
				if pidAlive(vm.PID) {
					vm.Running = true
				}
			}
		}

		_, portsPath := vmStatePaths(vmDir, vm)
		if data, err := os.ReadFile(portsPath); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				parts := strings.SplitN(line, ",", 2)
				if len(parts) != 2 {
					continue
				}
				var port int
				_, _ = fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &port)
				switch strings.ToLower(parts[0]) {
				case "spice":
					vm.SPICEPort = port
				case "ssh":
					vm.SSHPort = port
				case "SPICE=":
					vm.SPICEPort = port
				}
			}
		}

		vms = append(vms, vm)
	}
	sort.Slice(vms, func(i, j int) bool { return vms[i].Name < vms[j].Name })
	return vms
}

// VMCreateConfig holds the parameters used to create a new VM.
type VMCreateConfig struct {
	Name      string
	GuestOS   string
	ISO       string
	MemoryMB  int
	CPUCores  int
	SSHUser   string
}

// vmNameRe rejects unsafe VM names.
var vmNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// ValidVMName reports whether name is safe to use as a .conf file name.
func ValidVMName(name string) bool {
	return vmNameRe.MatchString(name)
}

// Backend defines the interface for VM backends.
type Backend interface {
	List() []VM
	Boot(vm *VM) error
	BootStream(vm *VM, onLog func(string)) error
	Kill(vm *VM) error
	Status(vm *VM) VMStatus
	VMDir() string
	Create(cfg VMCreateConfig) error
	ReadConfig(vm *VM) (string, error)
	WriteConfig(vm *VM, content string) error
}

// QEMUBackend is a backend for QEMU VMs.
type QEMUBackend struct {
	vmDir string
}

// NewQEMUBackend creates a new QEMU backend.
func NewQEMUBackend(vmDir string) *QEMUBackend {
	return &QEMUBackend{vmDir: vmDir}
}

// VMDir returns the VM directory path.
func (b *QEMUBackend) VMDir() string {
	return b.vmDir
}

// List returns the list of VMs.
func (b *QEMUBackend) List() []VM {
	return LoadVMs(b.vmDir)
}

// Boot boots a VM.
func (b *QEMUBackend) Boot(vm *VM) error {
	return b.BootStream(vm, nil)
}

// BootStream starts a VM without blocking and streams quickemu's terminal
// output (as seen when running quickemu interactively) to onLog, called from
// goroutines. The output is also appended to a persistent <name>.log file in
// the VM directory.
func (b *QEMUBackend) BootStream(vm *VM, onLog func(string)) error {
	// Start each boot with a clean log: the old log is removed so the log
	// shows only the current run.
	logPath := filepath.Join(b.vmDir, vm.Name+".log")
	if f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); err == nil {
		_ = f.Close()
	}

	master, tty, err := pty.Open()
	if err != nil {
		return err
	}
	cmd := exec.Command("quickemu", "--vm", vm.Config) //nolint:gosec
	cmd.Dir = b.vmDir
	cmd.Stdout = tty
	cmd.Stderr = tty
	cmd.Stdin = tty
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if err := cmd.Start(); err != nil {
		_ = tty.Close()
		_ = master.Close()
		return fmt.Errorf("%v", err)
	}
	_ = tty.Close()

	// Track our wrapper process immediately so the VM reports as running and
	// can be stopped even before quickemu writes its own .pid file. The pid
	// file goes into the VM's state directory (same layout quickemu/dh uses).
	pidPath, _ := vmStatePaths(b.vmDir, *vm)
	if err := os.MkdirAll(filepath.Dir(pidPath), 0o755); err == nil {
		_ = os.WriteFile(pidPath, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644) //nolint:gosec
	}

	go streamPtyOutput(master, logPath, onLog)

	go func() {
		_ = cmd.Wait()
		_ = master.Close()
		// The wrapper exited: drop any stale tracking files left behind.
		b.cleanupVMFiles(vm.Name)
	}()
	return nil
}

// pidAlive reports whether the given PID references a live process.
func pidAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	_, err := os.Stat(fmt.Sprintf("/proc/%d", pid))
	return err == nil
}

// streamPtyOutput reads lines from the PTY master, appends them to the VM's
// log file and feeds each line (with trailing newline) to onLog. It stops when
// quickemu exits and the PTY is closed.
func streamPtyOutput(master *os.File, logPath string, onLog func(string)) {
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(master)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text() + "\n"
		_, _ = f.WriteString(line)
		if onLog != nil {
			onLog(line)
		}
	}
}

// Create writes a new quickemu-style .conf file for a VM.
func (b *QEMUBackend) Create(cfg VMCreateConfig) error {
	name := strings.TrimSpace(cfg.Name)
	if !ValidVMName(name) {
		return fmt.Errorf("%s", t("new.err_name"))
	}
	if err := os.MkdirAll(b.vmDir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(b.vmDir, name+".conf")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf(t("new.err_exists"), name)
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "guest_os=%q\n", cfg.GuestOS)
	if iso := strings.TrimSpace(cfg.ISO); iso != "" {
		fmt.Fprintf(&sb, "iso=%q\n", iso)
	}
	fmt.Fprintf(&sb, "disk_img=%q\n", name+".qcow2")
	if cfg.MemoryMB > 0 {
		fmt.Fprintf(&sb, "memory=%d\n", cfg.MemoryMB)
	}
	if cfg.CPUCores > 0 {
		fmt.Fprintf(&sb, "cpu_cores=%d\n", cfg.CPUCores)
	}
	if u := strings.TrimSpace(cfg.SSHUser); u != "" {
		fmt.Fprintf(&sb, "ssh_user=%q\n", u)
	}
	return os.WriteFile(path, []byte(sb.String()), 0o644) //nolint:gosec
}

// ReadConfig returns the raw contents of a VM's .conf file.
func (b *QEMUBackend) ReadConfig(vm *VM) (string, error) {
	data, err := os.ReadFile(vm.Config)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteConfig replaces the contents of a VM's .conf file.
func (b *QEMUBackend) WriteConfig(vm *VM, content string) error {
	return os.WriteFile(vm.Config, []byte(content), 0o644) //nolint:gosec
}

// SetSSHUser updates the ssh_user setting in a VM's config file, replacing an
// existing ssh_user="..." line or appending one when absent. An empty user
// removes the setting so the current OS user is used as the fallback.
func SetSSHUser(vm *VM, user string) error {
	data, err := os.ReadFile(vm.Config)
	if err != nil {
		return err
	}
	re := regexp.MustCompile(`(?m)^\s*ssh_user\s*=\s*"[^"]*"\s*$`)
	user = strings.TrimSpace(user)
	var out string
	if user == "" {
		out = re.ReplaceAllString(string(data), "")
	} else {
		line := fmt.Sprintf("ssh_user=%q", user)
		if re.Match(data) {
			out = re.ReplaceAllString(string(data), line)
		} else {
			out = strings.TrimRight(string(data), "\n") + "\n" + line + "\n"
		}
	}
	vm.SSHUser = user
	return os.WriteFile(vm.Config, []byte(out), 0o644) //nolint:gosec
}

// Kill stops a VM. It first asks quickemu for a graceful shutdown (which
// cleans up its own files), then escalates to SIGTERM and SIGKILL on the
// tracked PID, and finally removes any leftover pid/ports files.
func (b *QEMUBackend) Kill(vm *VM) error {
	// Graceful stop first: quickemu powers the VM down and removes its own
	// pid/ports files. Any error is ignored and we fall back to signals.
	if vm.Config != "" {
		_ = exec.Command("quickemu", "--vm", vm.Config, "--kill").Run() //nolint:gosec
	}

	if vm.PID > 0 && pidAlive(vm.PID) {
		if err := syscall.Kill(vm.PID, syscall.SIGTERM); err == nil {
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) && pidAlive(vm.PID) {
				time.Sleep(100 * time.Millisecond)
			}
		}
		if pidAlive(vm.PID) {
			_ = syscall.Kill(vm.PID, syscall.SIGKILL)
		}
		return b.cleanupVMFiles(vm.Name)
	}
	if vm.PID <= 0 {
		return fmt.Errorf("VM not running")
	}
	return b.cleanupVMFiles(vm.Name)
}

// cleanupVMFiles removes stale state files (pid, ports, and the quickemu/dh
// spice/sock files) for a VM. The pid file is only removed once the PID it
// records no longer exists, so a still running process (e.g. an orphaned qemu)
// keeps its tracking files.
func (b *QEMUBackend) cleanupVMFiles(name string) error {
	pidPath := filepath.Join(b.vmDir, name+".pid")
	stateDir := filepath.Join(b.vmDir, name)
	if p, _ := vmStatePaths(b.vmDir, VM{Name: name, Config: filepath.Join(b.vmDir, name+".conf")}); p != "" {
		pidPath = p
		stateDir = filepath.Dir(p)
	}
	data, err := os.ReadFile(pidPath)
	if err != nil {
		// No pid file tracked; only clear leftover state files.
		_ = os.Remove(filepath.Join(stateDir, name+".ports"))
		_ = os.Remove(filepath.Join(stateDir, name+".spice"))
		_ = os.Remove(filepath.Join(stateDir, name+".sock"))
		return nil
	}
	var pid int
	_, _ = fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &pid)
	if pid > 0 && pidAlive(pid) {
		return nil
	}
	_ = os.Remove(pidPath)
	_ = os.Remove(filepath.Join(stateDir, name+".ports"))
	_ = os.Remove(filepath.Join(stateDir, name+".spice"))
	_ = os.Remove(filepath.Join(stateDir, name+".sock"))
	return nil
}

// Status returns the status of a VM.
func (b *QEMUBackend) Status(vm *VM) VMStatus {
	st := VMStatus{}
	if vm.PID <= 0 {
		return st
	}
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", vm.PID)); err == nil {
		st.Running = true
		st.PID = vm.PID
		st.SPICEPort = vm.SPICEPort
		st.SSHPort = vm.SSHPort
	}
	return st
}

// ConnectToVM connects to a VM via SPICE.
func ConnectToVM(port int, viewer string) error {
	var args []string
	switch viewer {
	case "remote-viewer", "rv":
		args = []string{"remote-viewer", fmt.Sprintf("spice://localhost:%d", port)}
	case "spicy":
		args = []string{"spicy", "-h", "localhost", "-p", fmt.Sprintf("%d", port)}
	default:
		args = []string{"remote-viewer", fmt.Sprintf("spice://localhost:%d", port)}
	}
	cmd := exec.Command(args[0], args[1:]...)
	return cmd.Start()
}

// resolveTerminal returns a terminal emulator binary that accepts `-e CMD`,
// preferring $SYSMAN_TERMINAL and otherwise probing common emulators.
func resolveTerminal() string {
	if t := os.Getenv("SYSMAN_TERMINAL"); t != "" {
		if bin, err := exec.LookPath(t); err == nil {
			return bin
		}
	}
	for _, t := range []string{"sakura", "x-terminal-emulator", "xterm", "konsole", "gnome-terminal", "xfce4-terminal", "kitty", "alacritty"} {
		if bin, err := exec.LookPath(t); err == nil {
			return bin
		}
	}
	return ""
}

// sshUser returns the SSH login name for a VM: its ssh_user config setting
// when present and valid, falling back to the current OS user name.
func sshUser(vm VM) string {
	if u := strings.TrimSpace(vm.SSHUser); vmNameRe.MatchString(u) {
		return u
	}
	if cur, err := user.Current(); err == nil {
		if u := strings.TrimSpace(cur.Username); u != "" {
			return u
		}
	}
	return ""
}

// ConnectToVMSSH opens a terminal emulator running an SSH session into the
// VM's forwarded port. An empty user falls back to the current OS user name.
// The shell keeps the window open after ssh exits so output/errors stay
// visible instead of closing instantly.
func ConnectToVMSSH(port int, username string) error {
	term := resolveTerminal()
	if term == "" {
		return fmt.Errorf("no terminal emulator found (set $SYSMAN_TERMINAL)")
	}
	u := strings.TrimSpace(username)
	if !vmNameRe.MatchString(u) {
		if cur, err := user.Current(); err == nil {
			u = strings.TrimSpace(cur.Username)
		}
	}
	if u == "" {
		return fmt.Errorf("no ssh user name for VM (set ssh_user= in the VM config)")
	}
	portStr := strconv.Itoa(port)
	target := u + "@localhost"
	shell := fmt.Sprintf(`printf 'Connecting to %s:%s...\n'; ssh -p %s -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null %s; echo; printf 'SSH session ended. Press Enter to close... '; read _`,
		target, portStr, portStr, target)
	cmd := exec.Command(term, "-e", "sh", "-c", shell) //nolint:gosec
	return cmd.Start()
}

// OpenEditor opens a VM's .conf file in $EDITOR (xdg-open as fallback),
// launched detached and non-blocking.
func OpenEditor(vmDir, name string) {
	path := filepath.Join(ResolveVMDir(vmDir), name+".conf")
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "xdg-open"
	}
	cmd := exec.Command(editor, path) //nolint:gosec
	_ = cmd.Start()
}

// readVMLog returns the on-disk <name>.log contents (without a trailing
// newline), or an empty string if none exists.
func readVMLog(vmDir, name string) string {
	data, err := os.ReadFile(filepath.Join(vmDir, name+".log"))
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(string(data), "\n")
}

var langs = map[string]map[string]string{}

// T is the translation map.
var T map[string]string
var i18nOnce sync.Once

func langDirs() []string {
	return common.GetLangDirs("vmsman")
}

func loadLangDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		loadLangFile(filepath.Join(dir, e.Name()))
	}
}

type langFile struct {
	Meta struct {
		Code string `yaml:"code"`
		Name string `yaml:"name"`
	} `yaml:"meta"`
	Strings map[string]string `yaml:"strings"`
}

func loadLangFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var lf langFile
	if err := yaml.Unmarshal(data, &lf); err != nil {
		return
	}
	if lf.Meta.Code == "" {
		return
	}
	langs[strings.ToLower(lf.Meta.Code)] = lf.Strings
}

func detectLang() string {
	if l := os.Getenv("SYSMAN_LANG"); l != "" {
		return strings.ToLower(strings.TrimSpace(l))
	}
	for _, env := range []string{"LANGUAGE", "LANG", "LC_ALL", "LC_MESSAGES"} {
		if l := os.Getenv(env); l != "" {
			l = strings.ToLower(l)
			l = strings.SplitN(l, "_", 2)[0]
			l = strings.SplitN(l, ".", 2)[0]
			if _, ok := langs[l]; ok {
				return l
			}
		}
	}
	return "en"
}

// InitI18n loads the translation files.
func InitI18n() {
	i18nOnce.Do(func() {
		for _, dir := range langDirs() {
			loadLangDir(dir)
		}
		lang := detectLang()
		if tr, ok := langs[lang]; ok {
			T = tr
			return
		}
		if tr, ok := langs["en"]; ok {
			T = tr
			return
		}
		T = map[string]string{}
	})
}

func t(key string) string {
	if T == nil {
		InitI18n()
	}
	if v, ok := T[key]; ok {
		return v
	}
	return key
}
