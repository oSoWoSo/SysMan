// Package infman is a system manager plugin that displays system information
// using native Go (reading /proc, /etc/os-release, syscall.Statfs, etc.).
// The fastfetch/neofetch runner and ANSI colour parser live in ansi.go and
// are exported for reuse by other packages.
// It implements api.PluginIF and can be used:
//   - Standalone: via cmd/sysinfo (GUI or TUI)
//   - Embedded: via cmd/sysmanager (as a static built-in plugin)
//   - Dynamic: via pluginentry/sysinfo compiled as a .so
package infman

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"codeberg.org/oSoWoSo/SysMan/src/common"
	serman "codeberg.org/oSoWoSo/SysMan/src/serman"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/sys/unix"
)

// Usage is the --help text for infoman.
const Usage = "infoman [-g|-t]\n\nOptions:\n  -g, --gui   GUI (default)\n  -t, --tui   TUI\n  -h, --help  show this help\n\nEnvironment:\n  SYSMAN_LANG  language override (e.g. cs)"

// RunTUI runs the sysinfo as a standalone Bubbletea application.
func RunTUI() {
	prog := tea.NewProgram(New().Model(), tea.WithAltScreen())
	if _, err := prog.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// Plugin displays basic system information (hostname, OS, arch, CPUs, Go version).
type Plugin struct {
	statusBar *common.StatusBar
}

// New returns a new sysinfo Plugin.
func New() *Plugin { return &Plugin{} }

// Name returns the plugin display name.
// Implements api.PluginIF.
func (p *Plugin) Name() string { return t("tab.name") }

// SetStatusBar sets a shared status bar for tooltips and messages.
// Implements api.PluginIF.
func (p *Plugin) SetStatusBar(statusBar *common.StatusBar) {
	p.statusBar = statusBar
}

// ── Native info collection ─────────────────────────────────────────────

// infoEntry is a single key/value pair for display.
type infoEntry struct {
	Key   string
	Value string
}

// collectNative gathers system information from /proc and other native sources.
func collectNative() []infoEntry {
	var entries []infoEntry

	add := func(k, v string) {
		entries = append(entries, infoEntry{Key: k, Value: v})
	}

	// Hostname
	if h, err := os.Hostname(); err == nil {
		add(t("info.hostname"), h)
	}

	// OS / distro from /etc/os-release
	if name, ver := readOSRelease(); name != "" {
		add(t("info.os"), name+" "+ver)
	} else {
		add(t("info.os"), runtime.GOOS)
	}

	// Kernel from /proc/version
	if k := readKernel(); k != "" {
		add(t("info.kernel"), k)
	}

	// Architecture
	add(t("info.arch"), runtime.GOARCH)

	// CPU model and core count from /proc/cpuinfo
	model, cores := readCPUInfo()
	if model != "" {
		add(t("info.cpu"), model)
	}
	if cores > 0 {
		add(t("info.cores"), strconv.Itoa(cores))
	}

	// Memory from /proc/meminfo
	if total, avail, ok := readMemInfo(); ok {
		used := total - avail
		add(t("info.memory"), fmt.Sprintf(t("info.memory.fmt"), formatMiB(used), formatMiB(total)))
	}

	// Uptime from /proc/uptime
	if up := readUptime(); up != "" {
		add(t("info.uptime"), up)
	}

	// Shell
	if sh := os.Getenv("SHELL"); sh != "" {
		add(t("info.shell"), filepath.Base(sh))
	}

	// Desktop environment
	for _, env := range []string{"XDG_CURRENT_DESKTOP", "DESKTOP_SESSION", "XDG_SESSION_DESKTOP"} {
		if de := os.Getenv(env); de != "" {
			add(t("info.desktop"), de)
			break
		}
	}

	// Disk usage for /
	if used, total, ok := readDisk("/"); ok {
		add(t("info.disk"), fmt.Sprintf(t("info.disk.fmt"), formatGB(used), formatGB(total)))
	}

	return entries
}

// readOSRelease parses /etc/os-release and returns (NAME, VERSION_ID).
func readOSRelease() (string, string) {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return "", ""
	}
	defer func() { _ = f.Close() }()

	kv := make(map[string]string)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if idx := strings.IndexByte(line, '='); idx >= 0 {
			k := line[:idx]
			v := strings.Trim(line[idx+1:], `"`)
			kv[k] = v
		}
	}
	name := kv["NAME"]
	if name == "" {
		name = kv["ID"]
	}
	return name, kv["VERSION_ID"]
}

// readKernel returns the kernel version string from /proc/version.
func readKernel() string {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return ""
	}
	// "Linux version 6.x.y-... ..."  — keep just the version token
	fields := strings.Fields(string(data))
	if len(fields) >= 3 {
		return fields[2]
	}
	return strings.TrimSpace(string(data))
}

// readCPUInfo returns (model name, number of logical processors) from /proc/cpuinfo.
func readCPUInfo() (string, int) {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "", runtime.NumCPU()
	}
	defer func() { _ = f.Close() }()

	var model string
	cores := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "model name") && model == "" {
			if idx := strings.IndexByte(line, ':'); idx >= 0 {
				model = strings.TrimSpace(line[idx+1:])
			}
		}
		if strings.HasPrefix(line, "processor") {
			cores++
		}
	}
	if cores == 0 {
		cores = runtime.NumCPU()
	}
	return model, cores
}

// readMemInfo returns (totalKiB, availableKiB, ok) from /proc/meminfo.
func readMemInfo() (int64, int64, bool) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, false
	}
	defer func() { _ = f.Close() }()

	kv := make(map[string]int64)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		parts := strings.Fields(sc.Text())
		if len(parts) >= 2 {
			if v, err := strconv.ParseInt(parts[1], 10, 64); err == nil {
				kv[strings.TrimSuffix(parts[0], ":")] = v
			}
		}
	}
	total, ok1 := kv["MemTotal"]
	avail, ok2 := kv["MemAvailable"]
	if !ok1 || !ok2 {
		return 0, 0, false
	}
	return total, avail, true
}

// readUptime returns a human-readable uptime string from /proc/uptime.
func readUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return ""
	}
	secs, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return ""
	}
	total := int(secs)
	days := total / 86400
	hours := (total % 86400) / 3600
	mins := (total % 3600) / 60
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, mins)
	default:
		return fmt.Sprintf("%dm", mins)
	}
}

// readDisk returns (usedBytes, totalBytes, ok) for the given mount point.
// Only available on Linux for now - returns false on other OSes.
func readDisk(path string) (uint64, uint64, bool) {
	// TODO: add proper cross-platform support for OpenBSD/FreeBSD
	// For now, only Linux is fully supported
	if runtime.GOOS != "linux" {
		return 0, 0, false
	}
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, 0, false
	}
	total := stat.Blocks * uint64(stat.Bsize)
	avail := uint64(stat.Bavail) * uint64(stat.Bsize)
	return total - avail, total, true
}

func formatMiB(kib int64) string {
	mib := kib / 1024
	if mib >= 1024 {
		return fmt.Sprintf("%.1f GiB", float64(mib)/1024)
	}
	return fmt.Sprintf("%d MiB", mib)
}

func formatGB(bytes uint64) string {
	gb := float64(bytes) / 1e9
	if gb >= 1 {
		return fmt.Sprintf("%.1f GB", gb)
	}
	return fmt.Sprintf("%d MB", bytes/1e6)
}

// ── Logo ───────────────────────────────────────────────────────────────

// logoImage tries to load a distro logo PNG from well-known paths.
// Returns nil when no logo is found.
func logoImage() *canvas.Image {
	return common.LogoImage()
}

// ── GUI ────────────────────────────────────────────────────────────────

// buildNativeView constructs a Fyne widget showing key/value pairs
// with the theme primary color for keys and foreground for values.
func buildNativeView(entries []infoEntry) fyne.CanvasObject {
	const keyWidth float32 = 110
	const fontSize float32 = 14

	var rows []fyne.CanvasObject
	for _, e := range entries {
		key := canvas.NewText(e.Key, theme.Color(theme.ColorNamePrimary))
		key.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
		key.TextSize = fontSize

		val := canvas.NewText(e.Value, theme.Color(theme.ColorNameForeground))
		val.TextStyle = fyne.TextStyle{Monospace: true}
		val.TextSize = fontSize

		keyBox := container.New(&fixedWidthLayout{width: keyWidth}, key)
		row := container.NewHBox(keyBox, val)
		rows = append(rows, row)
	}
	return container.NewVBox(rows...)
}

// fixedWidthLayout is a minimal layout that forces its single child to a fixed width.
type fixedWidthLayout struct{ width float32 }

func (l *fixedWidthLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(l.width, 0)
}

func (l *fixedWidthLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(fyne.NewSize(l.width, size.Height))
	}
}

// showAbout displays the About dialog for infoman.
func showAbout(win fyne.Window) {
	common.ShowAbout(common.AboutConfig{
		Win:       win,
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

// Content builds the Fyne widget tree showing system information.
// Implements api.PluginIF.
func (p *Plugin) Content(win fyne.Window) fyne.CanvasObject {
	entries := collectNative()
	infoView := buildNativeView(entries)

	var inner fyne.CanvasObject
	if img := logoImage(); img != nil {
		logoCol := container.NewVBox(layout.NewSpacer(), img, layout.NewSpacer())
		inner = container.NewHBox(logoCol, infoView)
	} else {
		inner = infoView
	}

	scroll := container.NewScroll(inner)

	// Status bar for tooltips
	statusBar := p.statusBar
	if statusBar == nil {
		statusBar = common.NewStatusBar()
	}
	statusBar.TextStyle = fyne.TextStyle{Italic: true, Monospace: true}

	btnAbout := common.NewHoverableButton("", theme.InfoIcon(), t("tooltip.infman.about"), statusBar, func() { showAbout(win) })
	btnAbout.Importance = widget.LowImportance
	statusBarRow := container.NewHBox(btnAbout, layout.NewSpacer(), statusBar)
	statusBarPanel := container.NewVBox(
		widget.NewSeparator(),
		container.NewPadded(statusBarRow),
	)

	return container.NewBorder(nil, statusBarPanel, nil, nil, scroll)
}

// Model returns a Bubbletea tea.Model showing system information.
// Implements api.PluginIF.
func (p *Plugin) Model() tea.Model {
	return sysInfoModel{}
}

// ── TUI model ─────────────────────────────────────────────────────────

type fetchDoneMsg struct{ entries []infoEntry }

type sysInfoModel struct {
	entries []infoEntry
	loaded  bool
	width   int
	height  int
}

var (
	tuiKeyStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00DDFF"))
	tuiValStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#DDDDDD"))
)

func (m sysInfoModel) Init() tea.Cmd {
	return func() tea.Msg {
		return fetchDoneMsg{collectNative()}
	}
}

func (m sysInfoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case fetchDoneMsg:
		m.entries = msg.entries
		m.loaded = true
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m sysInfoModel) View() string {
	if !m.loaded {
		return "\n  " + t("status.loading_info") + "\n"
	}

	// Find longest key for alignment
	maxKey := 0
	for _, e := range m.entries {
		if len(e.Key) > maxKey {
			maxKey = len(e.Key)
		}
	}

	var sb strings.Builder
	sb.WriteString("\n")
	for _, e := range m.entries {
		k := tuiKeyStyle.Render(fmt.Sprintf("  %-*s", maxKey, e.Key))
		v := tuiValStyle.Render("  " + e.Value)
		sb.WriteString(k + v + "\n")
	}
	sb.WriteString("\n")
	return sb.String()
}
