package vmman

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"codeberg.org/oSoWoSo/SysMan/src/common"
	"gopkg.in/yaml.v3"
)

var Version = "0.001 Alpha"

const (
	AppAuthor  = "zenobit @ oSoWoSo.org"
	AppLicense = "MIT"
	AppURL     = "https://codeberg.org/oSoWoSo/VMman"
	Usage      = "vmman [-g|-t] [--vm NAME] [--port PORT]\n\nOptions:\n  -g, --gui     GUI (default)\n  -t, --tui     TUI\n  --vm NAME     VM name (from config)\n  --port PORT   SPICE port (auto-detected if not provided)\n  -h, --help    show this help\n\nEnvironment:\n  VMDIR          VM directory (default: ~/vm)\n  SYSMAN_LANG    language override (e.g. cs)"
)

const DefaultVmDir = "vm"

type VM struct {
	Name      string
	Config    string
	Disk      string
	ISO       string
	PID       int
	SPICEPort int
	Running   bool
}

type VMStatus struct {
	Running   bool
	PID       int
	SPICEPort int
	SSHPort   int
	Display   string
	Uptime    string
	Raw       string
}

type FilterMode int

const (
	FilterAll FilterMode = iota
	FilterRunning
	FilterStopped
)

func Filter[T any](
	items []T,
	mode FilterMode,
	search string,
	isRunning func(T) bool,
	matchesSearch func(T, string) bool,
) []T {
	return common.Filter(items, int(mode), search, isRunning, matchesSearch)
}

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

		pidPath := filepath.Join(vmDir, name+".pid")
		if data, err := os.ReadFile(pidPath); err == nil {
			fmt.Sscanf(strings.TrimSpace(string(data)), "%d", &vm.PID)
			if vm.PID > 0 {
				if _, err := os.Stat(fmt.Sprintf("/proc/%d", vm.PID)); err == nil {
					vm.Running = true
				}
			}
		}

		portsPath := filepath.Join(vmDir, name+".ports")
		if data, err := os.ReadFile(portsPath); err == nil {
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "SPICE=") {
					fmt.Sscanf(strings.TrimPrefix(line, "SPICE="), "%d", &vm.SPICEPort)
				}
			}
		}

		vms = append(vms, vm)
	}
	sort.Slice(vms, func(i, j int) bool { return vms[i].Name < vms[j].Name })
	return vms
}

type Backend interface {
	List() []VM
	Boot(vm *VM) error
	Kill(vm *VM) error
	Status(vm *VM) VMStatus
}

type QEMUBackend struct {
	VMDir string
}

func NewQEMUBackend(vmDir string) *QEMUBackend {
	return &QEMUBackend{VMDir: vmDir}
}

func (b *QEMUBackend) List() []VM {
	return LoadVMs(b.VMDir)
}

func (b *QEMUBackend) Boot(vm *VM) error {
	args := []string{"quickemu", "--vm", vm.Config}
	out, err := exec.Command(args[0], args[1:]...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (b *QEMUBackend) Kill(vm *VM) error {
	if vm.PID <= 0 {
		return fmt.Errorf("VM not running")
	}
	args := []string{"kill", "-9", fmt.Sprintf("%d", vm.PID)}
	out, err := exec.Command("sh", "-c", strings.Join(args, " ")).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}

func (b *QEMUBackend) Status(vm *VM) VMStatus {
	st := VMStatus{}
	if vm.PID <= 0 {
		return st
	}
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", vm.PID)); err == nil {
		st.Running = true
		st.PID = vm.PID
		st.SPICEPort = vm.SPICEPort
	}
	return st
}

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

var langs = map[string]map[string]string{}
var T map[string]string
var i18nOnce sync.Once

func langDirs() []string {
	dirs := []string{
		"/usr/local/share/SysMan/lang/vmsman",
		"/usr/share/SysMan/lang/vmsman",
	}
	if exe, err := os.Executable(); err == nil {
		dirs = append([]string{filepath.Join(filepath.Dir(exe), "lang", "vmsman")}, dirs...)
	}
	dirs = append([]string{"./lang/vmsman"}, dirs...)
	return dirs
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
