package pkgman

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
	"codeberg.org/oSoWoSo/SysMan/src/common"
	serman "codeberg.org/oSoWoSo/SysMan/src/serman"
	zone "github.com/lrstanley/bubblezone/v2"
)

// ── Styles ────────────────────────────────────────────────────────────

var (
	pSubtle    = compat.AdaptiveColor{Light: lipgloss.Color("#9B9B9B"), Dark: lipgloss.Color("#585858")}
	pHighlight = compat.AdaptiveColor{Light: lipgloss.Color("#006688"), Dark: lipgloss.Color("#00DDFF")}
	pDanger    = compat.AdaptiveColor{Light: lipgloss.Color("#CC3333"), Dark: lipgloss.Color("#FF5555")}
	pSuccess   = compat.AdaptiveColor{Light: lipgloss.Color("#22AA55"), Dark: lipgloss.Color("#44DD77")}
	pWarn      = compat.AdaptiveColor{Light: lipgloss.Color("#BB8800"), Dark: lipgloss.Color("#FFCC00")}

	pTitleStyle     = lipgloss.NewStyle().Bold(true).Foreground(pHighlight)
	pSelectedStyle  = common.TSelectedStyle.Padding(0, 1)
	pNormalStyle    = lipgloss.NewStyle().Padding(0, 1)
	pInstalledStyle = lipgloss.NewStyle().Foreground(pSuccess).Padding(0, 1)
	pSubtleStyle    = lipgloss.NewStyle().Foreground(pSubtle)
	pStrikeStyle    = lipgloss.NewStyle().Foreground(pSubtle).Strikethrough(true)
	pDangerStyle    = lipgloss.NewStyle().Foreground(pDanger).Bold(true)
	pSuccessStyle   = lipgloss.NewStyle().Foreground(pSuccess)
	pWarnStyle      = lipgloss.NewStyle().Foreground(pWarn)
)

// ── Messages ──────────────────────────────────────────────────────────

type pkgLoadedMsg struct{ pkgs []Package }
type pkgDoneMsg struct {
	action string
	output string
	err    error
}

// ── Filter ────────────────────────────────────────────────────────────

// pkgFilter is an alias for FilterMode for the TUI.
type pkgFilter = FilterMode

func (f pkgFilter) String() string {
	switch f {
	case FilterInstalled:
		return "installed"
	case FilterAvailable:
		return "available"
	default:
		return "all"
	}
}

func (f pkgFilter) next() pkgFilter {
	return (f + 1) % 3
}

// ── Model ─────────────────────────────────────────────────────────────

// pkgModel is the Bubbletea model for the xbpspkg TUI.
type pkgModel struct {
	id             string
	backend        PkgBackend
	backends       []PkgBackend
	packages       []Package
	cursor         int
	search         textinput.Model
	searchMode     bool
	filter         pkgFilter
	xbpsOn         bool
	appImageOn     bool
	flatpakOn      bool
	nixOn          bool
	guixOn         bool
	marked         map[string]bool
	queue          []QueueEntry
	queueBackends  map[string]PkgBackend
	detail         PackageDetail
	output         string
	status         string
	statusErr      bool
	loading        bool
	running        bool
	showAbout      bool
	showSettings   bool
	settingsFocus  int
	appImageDir    string
	reposFile      string
	repoInput      textinput.Model
	repos          []string
	usedRepos      map[string]bool
	pendingRepo    string
	confirmRepo    string
	repoCursor     int
	outputLines    int
	width          int
	height         int
	appImageInput  textinput.Model
	reposFileInput textinput.Model
}

// NewTuiModel creates an initialized package manager TUI model using the default xbps backend.
func NewTuiModel() tea.Model {
	return NewTuiModelWithBackend(NewXbpsBackend())
}

// NewTuiModelWithBackend creates an initialized TUI model using the provided backend.
func NewTuiModelWithBackend(b PkgBackend) tea.Model {
	zone.NewGlobal()
	ti := textinput.New()
	ti.Placeholder = t("tui.search.placeholder")
	repoInput := textinput.New()
	repoInput.Placeholder = "https://repo.example.com/void-current"
	appImageInput := textinput.New()
	appImageInput.Placeholder = DefaultAppImageDir()
	reposFileInput := textinput.New()
	reposFileInput.Placeholder = defaultReposFile
	cfg := common.LoadSysManConfig()
	reposFile := strings.TrimSpace(cfg.Pkgman.ReposFile)
	if reposFile == "" {
		reposFile = defaultReposFile
	}
	appImageInput.SetValue(cfg.Pkgman.AppImageDir)
	reposFileInput.SetValue(cfg.Pkgman.ReposFile)
	return pkgModel{
		id: "pkgman:" + zone.NewPrefix(), backend: b, backends: []PkgBackend{b}, search: ti,
		repoInput: repoInput, appImageInput: appImageInput, reposFileInput: reposFileInput,
		marked: make(map[string]bool), queueBackends: make(map[string]PkgBackend),
		usedRepos: usedRepoSet(reposFile),
		loading:   true, xbpsOn: true, appImageDir: cfg.Pkgman.AppImageDir,
		reposFile: cfg.Pkgman.ReposFile, repos: append([]string(nil), cfg.Pkgman.Repos...),
	}
}

func (m pkgModel) filtered() []Package {
	return Filter(m.packages, m.filter, m.search.Value(),
		func(p Package) bool { return p.Installed },
		func(p Package, q string) bool {
			return strings.Contains(strings.ToLower(p.Name), q) ||
				strings.Contains(strings.ToLower(p.ShortDesc), q)
		},
	)
}

func (m pkgModel) clampCursor() pkgModel {
	list := m.filtered()
	if len(list) == 0 {
		m.cursor = 0
		return m
	}
	if m.cursor >= len(list) {
		m.cursor = len(list) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	return m
}

func (m pkgModel) selectedPkg() (Package, bool) {
	list := m.filtered()
	if m.cursor < 0 || m.cursor >= len(list) {
		return Package{}, false
	}
	return list[m.cursor], true
}

// inQueue returns the index and action of a package in the queue, or -1 if not queued.
func (m pkgModel) inQueue(name string) (int, string) {
	for i, e := range m.queue {
		if e.Name == name {
			return i, e.Action
		}
	}
	return -1, ""
}

func (m *pkgModel) activeBackends() []PkgBackend {
	var out []PkgBackend
	if m.xbpsOn {
		out = append(out, NewXbpsBackend())
	}
	if m.appImageOn {
		cfg := common.LoadSysManConfig()
		out = append(out, NewAppImageBackendWithDir(effectiveAppImageDir(cfg.Pkgman.AppImageDir)))
	}
	if m.flatpakOn {
		out = append(out, NewFlatpakBackend())
	}
	if m.nixOn {
		out = append(out, NewNixBackend())
	}
	if m.guixOn {
		out = append(out, NewGuixBackend())
	}
	if len(out) == 0 {
		m.xbpsOn = true
		out = append(out, NewXbpsBackend())
	}
	m.backends = out
	return out
}

func (m *pkgModel) toggleBackend(name string) tea.Cmd {
	switch name {
	case "xbps":
		m.xbpsOn = !m.xbpsOn
	case "appimage":
		m.appImageOn = !m.appImageOn
	case "flatpak":
		m.flatpakOn = !m.flatpakOn
	case "nix":
		m.nixOn = !m.nixOn
	case "guix":
		m.guixOn = !m.guixOn
	}
	m.activeBackends()
	m.loading = true
	m.cursor = 0
	m.packages = nil
	m.marked = make(map[string]bool)
	m.queue = nil
	m.queueBackends = make(map[string]PkgBackend)
	return m.loadPackagesCmd()
}

func (m *pkgModel) setSelectedBackend() {
	if pkg, ok := m.selectedPkg(); ok && pkg.Backend != nil {
		m.backend = pkg.Backend
	}
}

func (m pkgModel) selectedPkgExists() bool {
	_, ok := m.selectedPkg()
	return ok
}

func (m *pkgModel) copyOutput() {
	if m.output == "" {
		return
	}
	if err := common.CopyToClipboard(m.output); err != nil {
		m.status = t("action.copy_err")
		m.statusErr = true
	} else {
		m.status = t("action.copied")
		m.statusErr = false
	}
}

func (m *pkgModel) openSettings() {
	cfg := common.LoadSysManConfig()
	m.appImageDir = cfg.Pkgman.AppImageDir
	m.reposFile = cfg.Pkgman.ReposFile
	m.repos = append([]string(nil), cfg.Pkgman.Repos...)
	m.usedRepos = usedRepoSet(m.reposFilename())
	m.appImageInput.SetValue(m.appImageDir)
	m.reposFileInput.SetValue(m.reposFile)
	m.repoInput.SetValue("")
	m.showSettings = true
	m.settingsFocus = 0
	m.appImageInput.Focus()
}

func (m pkgModel) reposFilename() string {
	if strings.TrimSpace(m.reposFile) == "" {
		return defaultReposFile
	}
	return strings.TrimSpace(m.reposFile)
}

type reposSyncedMsg struct {
	used map[string]bool
	err  error
}

type settingsSavedMsg struct{ err error }

func (m pkgModel) syncReposCmd(used map[string]bool) tea.Cmd {
	repos := make([]string, 0, len(used))
	for repo := range used {
		repos = append(repos, repo)
	}
	file := m.reposFilename()
	return func() tea.Msg { return reposSyncedMsg{used: used, err: syncSystemRepos(repos, file)} }
}

func (m pkgModel) saveSettingsCmd() tea.Cmd {
	cfg := common.LoadSysManConfig()
	cfg.Pkgman.AppImageDir = strings.TrimSpace(m.appImageInput.Value())
	cfg.Pkgman.ReposFile = strings.TrimSpace(m.reposFileInput.Value())
	cfg.Pkgman.Repos = append([]string(nil), m.repos...)
	return func() tea.Msg { return settingsSavedMsg{err: common.SaveSysManConfig(cfg)} }
}

func (m *pkgModel) saveRepos() error {
	cfg := common.LoadSysManConfig()
	cfg.Pkgman.Repos = append([]string(nil), m.repos...)
	return common.SaveSysManConfig(cfg)
}

func (m *pkgModel) addRepo() {
	repo := strings.TrimSpace(m.repoInput.Value())
	if repo == "" {
		m.status = t("repos.add_empty")
		return
	}
	for _, item := range m.repos {
		if item == repo {
			m.status = t("repos.add_exists")
			return
		}
	}
	m.repos = append(m.repos, repo)
	if err := m.saveRepos(); err != nil {
		m.repos = m.repos[:len(m.repos)-1]
		m.status = err.Error()
		m.statusErr = true
		return
	}
	m.repoInput.SetValue("")
	m.status = t("repos.added")
	m.statusErr = false
}

func (m *pkgModel) removeRepo(repo string) {
	original := append([]string(nil), m.repos...)
	out := m.repos[:0]
	for _, item := range m.repos {
		if item != repo {
			out = append(out, item)
		}
	}
	m.repos = out
	if err := m.saveRepos(); err != nil {
		m.repos = original
		m.status = err.Error()
		m.statusErr = true
		return
	}
	m.statusErr = false
}

func (m *pkgModel) activateRepo(repo string) tea.Cmd {
	m.settingsFocus = 3
	if !m.usedRepos[repo] {
		m.pendingRepo = repo
		m.confirmRepo = repo
		return nil
	}
	used := make(map[string]bool, len(m.usedRepos))
	for r, on := range m.usedRepos {
		if on && r != repo {
			used[r] = true
		}
	}
	return m.syncReposCmd(used)
}
func (m pkgModel) handleSettingsMouse(msg tea.MouseClickMsg) (pkgModel, tea.Cmd) {
	if msg.Button != tea.MouseLeft {
		return m, nil
	}
	switch {
	case zone.Get(m.id + "s_close").InBounds(msg):
		m.showSettings = false
	case zone.Get(m.id + "s_save").InBounds(msg):
		return m, m.saveSettingsCmd()
	case zone.Get(m.id + "s_appimage").InBounds(msg):
		m.settingsFocus = 0
		m.appImageInput.Focus()
		m.reposFileInput.Blur()
		m.repoInput.Blur()
	case zone.Get(m.id + "s_reposfile").InBounds(msg):
		m.settingsFocus = 1
		m.reposFileInput.Focus()
		m.appImageInput.Blur()
		m.repoInput.Blur()
	case zone.Get(m.id + "s_repo_input").InBounds(msg):
		m.settingsFocus = 2
		m.repoInput.Focus()
		m.appImageInput.Blur()
		m.reposFileInput.Blur()
	case zone.Get(m.id + "s_add").InBounds(msg):
		m.addRepo()
	case zone.Get(m.id + "s_confirm").InBounds(msg):
		m.confirmRepo = ""
		used := make(map[string]bool, len(m.usedRepos)+1)
		for r, on := range m.usedRepos {
			if on {
				used[r] = true
			}
		}
		used[m.pendingRepo] = true
		return m, m.syncReposCmd(used)
	case zone.Get(m.id + "s_cancel").InBounds(msg):
		m.confirmRepo = ""
	default:
		for i, repo := range m.repos {
			if zone.Get(fmt.Sprintf("%srepo_use_%d", m.id, i)).InBounds(msg) {
				m.repoCursor = i
				return m, m.activateRepo(repo)
			}
			if zone.Get(fmt.Sprintf("%srepo_remove_%d", m.id, i)).InBounds(msg) {
				m.repoCursor = i
				m.settingsFocus = 3
				m.removeRepo(repo)
				if m.repoCursor >= len(m.repos) {
					m.repoCursor = len(m.repos) - 1
				}
				if m.repoCursor < 0 {
					m.repoCursor = 0
				}
				return m, nil
			}
		}
	}
	return m, nil
}

func (m pkgModel) handleSettingsKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.confirmRepo != "" {
		if key.String() == "enter" || key.String() == "y" {
			m.confirmRepo = ""
			used := make(map[string]bool, len(m.usedRepos)+1)
			for r, on := range m.usedRepos {
				if on {
					used[r] = true
				}
			}
			used[m.pendingRepo] = true
			return m, m.syncReposCmd(used)
		}
		m.confirmRepo = ""
		return m, nil
	}
	if key.String() == "esc" {
		m.showSettings = false
		return m, nil
	}
	if key.String() == "tab" {
		m.settingsFocus = (m.settingsFocus + 1) % 4
		m.appImageInput.Blur()
		m.reposFileInput.Blur()
		m.repoInput.Blur()
		switch m.settingsFocus {
		case 0:
			m.appImageInput.Focus()
		case 1:
			m.reposFileInput.Focus()
		case 2:
			m.repoInput.Focus()
		}
		return m, nil
	}
	if m.settingsFocus == 3 {
		switch key.String() {
		case "up", "k":
			if m.repoCursor > 0 {
				m.repoCursor--
			}
			return m, nil
		case "down", "j":
			if m.repoCursor < len(m.repos)-1 {
				m.repoCursor++
			}
			return m, nil
		case "d":
			if len(m.repos) > 0 {
				m.removeRepo(m.repos[m.repoCursor])
				if m.repoCursor >= len(m.repos) {
					m.repoCursor = len(m.repos) - 1
				}
				if m.repoCursor < 0 {
					m.repoCursor = 0
				}
			}
			return m, nil
		case "enter":
			if len(m.repos) > 0 {
				return m, m.activateRepo(m.repos[m.repoCursor])
			}
			return m, nil
		}
	}
	if key.Code == tea.KeyEnter {
		if m.settingsFocus == 2 {
			m.addRepo()
			return m, nil
		}
		return m, m.saveSettingsCmd()
	}
	var cmd tea.Cmd
	switch m.settingsFocus {
	case 0:
		m.appImageInput, cmd = m.appImageInput.Update(key)
		m.appImageDir = m.appImageInput.Value()
	case 1:
		m.reposFileInput, cmd = m.reposFileInput.Update(key)
		m.reposFile = m.reposFileInput.Value()
	case 2:
		m.repoInput, cmd = m.repoInput.Update(key)
	}
	return m, cmd
}

// toggleQueue adds a package to the queue if not present, or removes it if already queued.
func (m *pkgModel) toggleQueue(pkg Package) {
	idx, _ := m.inQueue(pkg.Name)
	if idx >= 0 {
		m.queue = append(m.queue[:idx], m.queue[idx+1:]...)
		delete(m.queueBackends, pkg.Name)
		m.status = fmt.Sprintf("Removed %s from queue", pkg.Name)
	} else {
		action := "install"
		if pkg.Installed {
			action = "remove"
		}
		m.queue = append(m.queue, QueueEntry{Name: pkg.Name, Action: action})
		m.queueBackends[pkg.Name] = pkg.Backend
		m.status = fmt.Sprintf("Added %s to queue (%s)", pkg.Name, action)
	}
	m.statusErr = false
}

// ── tea.Model ─────────────────────────────────────────────────────────

func (m pkgModel) Init() tea.Cmd {
	return m.loadPackagesCmd()
}

func (m pkgModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		return m, nil

	case pkgLoadedMsg:
		m.packages = msg.pkgs
		m.loading = false
		m.status = fmt.Sprintf("%d packages", len(m.packages))
		m = m.clampCursor()
		if pkg, ok := m.selectedPkg(); ok {
			return m, m.queryDetailCmd(pkg.Name)
		}
		return m, nil

	case pkgDoneMsg:
		m.running = false
		m.output = msg.output
		if m.output == "" && msg.err != nil {
			m.output = msg.err.Error()
		}
		if msg.action == "queue" {
			m.queue = nil
			m.queueBackends = make(map[string]PkgBackend)
		}
		if msg.err != nil {
			m.status = fmt.Sprintf("%s failed: %s", msg.action, msg.err.Error())
			m.statusErr = true
		} else if msg.action == "queue" {
			m.status = t("pkg.queue_ok")
			m.statusErr = false
		} else {
			m.status = fmt.Sprintf("%s OK", msg.action)
			m.statusErr = false
		}
		m.loading = true
		return m, m.loadPackagesCmd()

	case settingsSavedMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			m.statusErr = true
			return m, nil
		}
		m.appImageDir = strings.TrimSpace(m.appImageInput.Value())
		m.reposFile = strings.TrimSpace(m.reposFileInput.Value())
		m.usedRepos = usedRepoSet(m.reposFilename())
		m.status = t("settings.saved")
		m.statusErr = false
		m.activeBackends()
		m.loading = true
		return m, m.loadPackagesCmd()

	case reposSyncedMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			m.statusErr = true
			return m, nil
		}
		m.usedRepos = msg.used
		m.status = t("repos.synced")
		m.statusErr = false
		m.loading = true
		return m, m.loadPackagesCmd()

	case tea.MouseClickMsg:
		if m.showSettings {
			m, cmd := m.handleSettingsMouse(msg)
			return m, cmd
		}
		if m.showAbout {
			if msg.Button == tea.MouseLeft && zone.Get(m.id+"a_about_home").InBounds(msg) {
				if u, err := url.Parse(serman.AppURL); err == nil && u.IsAbs() {
					m.backend.OpenURL(u.String())
				}
			} else {
				m.showAbout = false
			}
			return m, nil
		}
		if msg.Button != tea.MouseLeft {
			break
		}
		if zone.Get(m.id + "f_all").InBounds(msg) {
			m.filter = FilterAll
			m.detail = PackageDetail{}
			m.cursor = 0
			m.marked = make(map[string]bool)
			if pkg, ok := m.selectedPkg(); ok {
				return m, m.queryDetailCmd(pkg.Name)
			}
			return m, nil
		}
		if zone.Get(m.id + "f_installed").InBounds(msg) {
			m.filter = FilterInstalled
			m.detail = PackageDetail{}
			m.cursor = 0
			m.marked = make(map[string]bool)
			if pkg, ok := m.selectedPkg(); ok {
				return m, m.queryDetailCmd(pkg.Name)
			}
			return m, nil
		}
		if zone.Get(m.id + "f_available").InBounds(msg) {
			m.filter = FilterAvailable
			m.detail = PackageDetail{}
			m.cursor = 0
			m.marked = make(map[string]bool)
			if pkg, ok := m.selectedPkg(); ok {
				return m, m.queryDetailCmd(pkg.Name)
			}
			return m, nil
		}
		if zone.Get(m.id + "f_xbps").InBounds(msg) {
			return m, m.toggleBackend("xbps")
		}
		if zone.Get(m.id + "f_appimage").InBounds(msg) {
			return m, m.toggleBackend("appimage")
		}
		if zone.Get(m.id + "f_flatpak").InBounds(msg) {
			return m, m.toggleBackend("flatpak")
		}
		if zone.Get(m.id + "f_nix").InBounds(msg) {
			return m, m.toggleBackend("nix")
		}
		if zone.Get(m.id + "f_guix").InBounds(msg) {
			return m, m.toggleBackend("guix")
		}
		if zone.Get(m.id + "a_homepage").InBounds(msg) {
			if u, err := url.Parse(m.detail.Homepage); err == nil && u.IsAbs() {
				m.setSelectedBackend()
				m.backend.OpenURL(u.String())
			}
			return m, nil
		}
		if zone.Get(m.id + "a_toggle").InBounds(msg) {
			if pkg, ok := m.selectedPkg(); ok {
				m.toggleQueue(pkg)
				return m, nil
			}
		}
		if zone.Get(m.id + "a_update").InBounds(msg) {
			m.setSelectedBackend()
			m.running = true
			m.output = ""
			m.status = t("tui.status.updating")
			return m, m.pkgUpdateCmd()
		}
		if zone.Get(m.id + "a_queue_apply").InBounds(msg) {
			m.setSelectedBackend()
			if len(m.queue) > 0 || m.selectedPkgExists() {
				m.running = true
				m.output = ""
				m.status = fmt.Sprintf(t("pkg.queue_running"), len(m.queue))
				return m, m.applyQueueCmd()
			}
		}
		if zone.Get(m.id + "a_queue_clear").InBounds(msg) {
			m.queue = nil
			m.queueBackends = make(map[string]PkgBackend)
			return m, nil
		}
		if zone.Get(m.id + "a_reload").InBounds(msg) {
			m.loading = true
			m.status = t("tui.status.reloading")
			return m, m.loadPackagesCmd()
		}
		if zone.Get(m.id + "a_settings").InBounds(msg) {
			m.openSettings()
			return m, nil
		}
		if zone.Get(m.id + "a_quit").InBounds(msg) {
			return m, tea.Quit
		}
		if zone.Get(m.id + "a_about").InBounds(msg) {
			m.showAbout = true
			return m, nil
		}
		if zone.Get(m.id + "a_copy").InBounds(msg) {
			m.copyOutput()
			return m, nil
		}
		list := m.filtered()
		start := m.scrollStart()
		for i := start; i < len(list); i++ {
			if zone.Get(m.id + list[i].Name).InBounds(msg) {
				m.cursor = i
				if pkg, ok := m.selectedPkg(); ok {
					return m, m.queryDetailCmd(pkg.Name)
				}
				return m, nil
			}
		}
	case PackageDetail:
		m.detail = msg
		return m, nil

	case tea.KeyPressMsg:
		if m.showSettings {
			return m.handleSettingsKey(msg)
		}
		if m.searchMode {
			return m.handleSearchKey(msg)
		}
		if m.showAbout {
			if msg.String() == "h" {
				if u, err := url.Parse(serman.AppURL); err == nil && u.IsAbs() {
					m.backend.OpenURL(u.String())
				}
				return m, nil
			}
			m.showAbout = false
			return m, nil
		}
		return m.handleKey(msg)
	}

	if m.searchMode {
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m pkgModel) handleSearchKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch key.Code {
	case tea.KeyEsc:
		m.searchMode = false
		m.search.SetValue("")
		m.search.Blur()
		m = m.clampCursor()
		return m, nil
	case tea.KeyEnter:
		m.searchMode = false
		m.search.Blur()
		m = m.clampCursor()
		if pkg, ok := m.selectedPkg(); ok {
			return m, m.queryDetailCmd(pkg.Name)
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(key)
	m = m.clampCursor()
	if pkg, ok := m.selectedPkg(); ok {
		return m, m.queryDetailCmd(pkg.Name)
	}
	return m, cmd
}

func (m pkgModel) handleKey(key tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	list := m.filtered()

	switch key.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "?":
		m.showAbout = true
		return m, nil

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			if pkg, ok := m.selectedPkg(); ok {
				return m, m.queryDetailCmd(pkg.Name)
			}
		}
		return m, nil

	case "down", "j":
		if m.cursor < len(list)-1 {
			m.cursor++
			if pkg, ok := m.selectedPkg(); ok {
				return m, m.queryDetailCmd(pkg.Name)
			}
		}
		return m, nil

	case "/":
		m.searchMode = true
		m.search.Focus()
		return m, nil

	case "esc":
		if m.search.Value() != "" {
			m.search.SetValue("")
			m = m.clampCursor()
		}
		m.marked = make(map[string]bool)
		return m, nil

	case "h": // open homepage
		if u, err := url.Parse(m.detail.Homepage); err == nil && u.IsAbs() {
			m.setSelectedBackend()
			m.backend.OpenURL(u.String())
		}
		return m, nil

	case "tab":
		m.filter = m.filter.next()
		m.cursor = 0
		m.marked = make(map[string]bool)
		if pkg, ok := m.selectedPkg(); ok {
			return m, m.queryDetailCmd(pkg.Name)
		}
		return m, nil

	case "r":
		m.loading = true
		m.status = t("tui.status.reloading")
		return m, m.loadPackagesCmd()
	case "b":
		return m, m.toggleBackend("appimage")
	case "x":
		return m, m.toggleBackend("xbps")
	case "f":
		return m, m.toggleBackend("flatpak")
	case "n":
		return m, m.toggleBackend("nix")
	case "g":
		return m, m.toggleBackend("guix")
	case "s":
		m.openSettings()
		return m, nil
	case "y":
		m.copyOutput()
		return m, nil
	}

	if m.running || m.loading {
		return m, nil
	}

	switch key.String() {
	case "space": // toggle queue (install/remove)
		if pkg, ok := m.selectedPkg(); ok {
			m.toggleQueue(pkg)
			if m.cursor < len(list)-1 {
				m.cursor++
			}
		}
		return m, nil

	case "ctrl+up": // increase output lines
		if m.outputLines < 20 {
			m.outputLines++
		}
		return m, nil

	case "ctrl+down": // decrease output lines
		if m.outputLines > 0 {
			m.outputLines--
		}
		return m, nil

	case "a":
		m.setSelectedBackend()
		if len(m.queue) == 0 && !m.selectedPkgExists() {
			return m, nil
		}
		m.running = true
		m.output = ""
		m.status = fmt.Sprintf(t("pkg.queue_running"), len(m.queue))
		return m, m.applyQueueCmd()
	case "c":
		m.queue = nil
		m.queueBackends = make(map[string]PkgBackend)
		return m, nil
	case "u":
		m.setSelectedBackend()
		m.running = true
		m.output = ""
		m.status = t("tui.status.updating")
		return m, m.pkgUpdateCmd()
	}

	return m, nil
}

// ── Commands ──────────────────────────────────────────────────────────

func (m pkgModel) loadPackagesCmd() tea.Cmd {
	backends := append([]PkgBackend(nil), m.backends...)
	if len(backends) == 0 {
		if m.xbpsOn || (!m.appImageOn && !m.flatpakOn) {
			backends = append(backends, NewXbpsBackend())
		}
		if m.appImageOn {
			cfg := common.LoadSysManConfig()
			backends = append(backends, NewAppImageBackendWithDir(effectiveAppImageDir(cfg.Pkgman.AppImageDir)))
		}
		if m.flatpakOn {
			backends = append(backends, NewFlatpakBackend())
		}
		if m.nixOn {
			backends = append(backends, NewNixBackend())
		}
		if m.guixOn {
			backends = append(backends, NewGuixBackend())
		}
	}
	return func() tea.Msg {
		var pkgs []Package
		for _, b := range backends {
			b.Reload()
			for _, pkg := range b.List() {
				pkg.Backend = b
				pkgs = append(pkgs, pkg)
			}
		}
		return pkgLoadedMsg{pkgs: pkgs}
	}
}

func (m pkgModel) queryDetailCmd(name string) tea.Cmd {
	b := m.backend
	if pkg, ok := m.selectedPkg(); ok && pkg.Name == name && pkg.Backend != nil {
		b = pkg.Backend
	}
	return func() tea.Msg {
		return b.Detail(name)
	}
}
func (m pkgModel) pkgUpdateCmd() tea.Cmd {
	b := m.backend
	return func() tea.Msg {
		out, err := b.Update(io.Discard)
		return pkgDoneMsg{action: "update", output: out, err: err}
	}
}

// applyQueueCmd executes queued operations and the selected package when it is not queued.
func (m pkgModel) applyQueueCmd() tea.Cmd {
	q := append([]QueueEntry(nil), m.queue...)
	backendByName := make(map[string]PkgBackend, len(m.queueBackends)+1)
	for name, backend := range m.queueBackends {
		backendByName[name] = backend
	}
	var selected *Package
	if pkg, ok := m.selectedPkg(); ok {
		selected = &pkg
	}
	fallback := m.backend
	return func() tea.Msg {
		groups := make(map[string][]QueueEntry)
		for _, entry := range q {
			backend := backendByName[entry.Name]
			if backend == nil {
				backend = fallback
				backendByName[entry.Name] = backend
			}
			groups[backend.Name()] = append(groups[backend.Name()], entry)
		}
		if selected != nil {
			if _, queued := backendByName[selected.Name]; !queued {
				backend := selected.Backend
				if backend == nil {
					backend = fallback
				}
				backendByName[selected.Name] = backend
				action := "install"
				if selected.Installed {
					action = "remove"
				}
				groups[backend.Name()] = append(groups[backend.Name()], QueueEntry{Name: selected.Name, Action: action})
			}
		}
		var allOutput strings.Builder
		var firstErr error
		for name, entries := range groups {
			backend := fallback
			for _, entry := range entries {
				if b := backendByName[entry.Name]; b != nil {
					backend = b
					break
				}
			}
			var installs, removes []string
			for _, entry := range entries {
				if entry.Action == "install" {
					installs = append(installs, entry.Name)
				}
				if entry.Action == "remove" {
					removes = append(removes, entry.Name)
				}
			}
			if len(installs) > 0 {
				out, err := backend.Install(installs, io.Discard)
				if out != "" {
					allOutput.WriteString(out + "\n")
				}
				if err != nil && firstErr == nil {
					firstErr = err
				}
			}
			if len(removes) > 0 {
				out, err := backend.Remove(removes, io.Discard)
				if out != "" {
					allOutput.WriteString(out + "\n")
				}
				if err != nil && firstErr == nil {
					firstErr = err
				}
			}
			_ = name
		}
		return pkgDoneMsg{action: "queue", output: strings.TrimSpace(allOutput.String()), err: firstErr}
	}
}

// listOverhead returns the number of non-list lines consumed by the layout.
func (m pkgModel) listOverhead() int {
	overhead := 16
	if m.width > 0 && m.width < 80 {
		overhead = 17
	}
	return overhead
}

// scrollStart returns the first visible item index for the current cursor and terminal height.
func (m pkgModel) scrollStart() int {
	listHeight := m.height - m.listOverhead()
	if listHeight < 4 {
		listHeight = 4
	}
	start := 0
	if m.cursor >= listHeight {
		start = m.cursor - listHeight + 1
	}
	return start
}

// topBar renders the filter button bar at the top of the TUI.
func (m pkgModel) topBar() string {
	return common.TBtnBar(
		common.TBtn(m.id, "f_all", t("filter.all"), "tab", m.filter == FilterAll),
		common.TBtn(m.id, "f_installed", t("filter.installed"), "tab", m.filter == FilterInstalled),
		common.TBtn(m.id, "f_available", t("filter.available"), "tab", m.filter == FilterAvailable),
		common.TBtn(m.id, "f_xbps", t("filter.xbps"), "x", m.xbpsOn),
		common.TBtn(m.id, "f_appimage", t("filter.appimage"), "b", m.appImageOn),
		common.TBtn(m.id, "f_flatpak", t("filter.flatpak"), "f", m.flatpakOn),
		common.TBtn(m.id, "f_nix", t("filter.nix"), "n", m.nixOn),
		common.TBtn(m.id, "f_guix", t("filter.guix"), "g", m.guixOn),
	)
}

// detailButtons renders action buttons inside the right panel detail area.
func (m pkgModel) detailButtons() string {
	var buttons []string
	if pkg, ok := m.selectedPkg(); ok {
		idx, _ := m.inQueue(pkg.Name)
		if idx >= 0 {
			buttons = append(buttons, common.TBtn(m.id, "a_toggle", t("action.dequeue"), "space", false))
		} else if pkg.Installed {
			buttons = append(buttons, common.TBtn(m.id, "a_toggle", t("action.enqueue_remove"), "space", false))
		} else {
			buttons = append(buttons, common.TBtn(m.id, "a_toggle", t("action.enqueue_install"), "space", false))
		}
		if m.detail.Homepage != "" {
			buttons = append(buttons, common.TBtn(m.id, "a_homepage", t("detail.homepage"), "h", false))
		}
	}
	buttons = append(buttons,
		common.TBtn(m.id, "a_queue_apply", t("btn.queue_apply"), "a", false),
		common.TBtn(m.id, "a_queue_clear", t("btn.queue_clear"), "c", false),
		common.TBtn(m.id, "a_update", t("btn.update_all"), "u", false),
	)
	return common.TBtnBar(buttons...)
}

// bottomBar mirrors the GUI footer: controls left, status right.
func (m pkgModel) bottomBar() string {
	var status string
	if m.status != "" {
		if m.statusErr {
			status = pDangerStyle.Render("  ✗ " + m.status)
		} else {
			status = pSuccessStyle.Render("  ✓ " + m.status)
		}
	}
	buttons := []string{
		common.TBtn(m.id, "a_about", t("btn.about"), "?", false),
		common.TBtn(m.id, "a_reload", t("btn.reload"), "r", false),
		common.TBtn(m.id, "a_settings", t("btn.settings"), "s", false),
	}
	if !common.TUIEmbedded() {
		buttons = append(buttons, common.TBtn(m.id, "a_quit", t("action.quit"), "q", false))
	}
	controls := common.TBtnBar(buttons...)
	if m.running {
		status = pWarnStyle.Render("  " + t("tui.running"))
	}
	return common.TUIFooter(controls, status, m.width)
}

// ── View ──────────────────────────────────────────────────────────────

func (m pkgModel) View() tea.View { return m.view(true) }

// EmbeddedView returns the unscanned view for composition inside sysman-tui.
func (m pkgModel) EmbeddedView() tea.View { return m.view(false) }

func (m pkgModel) view(scan bool) tea.View {
	content := m.render()
	if scan {
		content = zone.Scan(content)
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m pkgModel) renderSettings() string {
	var sb strings.Builder
	sb.WriteString(pTitleStyle.Render(t("settings.title")) + "\n\n")
	sb.WriteString(t("settings.appimage_dir") + ": " + zone.Mark(m.id+"s_appimage", m.appImageInput.View()) + "\n")
	sb.WriteString(t("settings.repos_file") + ": " + zone.Mark(m.id+"s_reposfile", m.reposFileInput.View()) + "\n")
	sb.WriteString(common.TBtn(m.id, "s_save", t("btn.save"), "enter", false) + "  " + common.TBtn(m.id, "s_close", t("btn.close"), "esc", false) + "\n")
	sb.WriteString("\n" + pTitleStyle.Render(t("repos.title")) + "\n")
	if len(m.repos) == 0 {
		sb.WriteString("  " + t("repos.empty") + "\n")
	}
	for i, repo := range m.repos {
		state := t("btn.repos_use")
		if m.usedRepos[repo] {
			state = t("btn.repos_stop")
		}
		prefix := "  "
		if m.settingsFocus == 3 && i == m.repoCursor {
			prefix = "> "
		}
		fmt.Fprintf(&sb, "%s%s  %s  %s\n", prefix, repo,
			common.TBtn(m.id, fmt.Sprintf("repo_use_%d", i), state, "enter", false),
			common.TBtn(m.id, fmt.Sprintf("repo_remove_%d", i), t("btn.repos_remove"), "d", false))
	}
	sb.WriteString("\n" + zone.Mark(m.id+"s_repo_input", m.repoInput.View()) + "  " + common.TBtn(m.id, "s_add", t("btn.repos_add"), "enter", false) + "\n")
	sb.WriteString(pSubtleStyle.Render(t("repos.hint")) + "\n")
	if m.confirmRepo != "" {
		sb.WriteString("\n" + pWarnStyle.Render(t("repos.confirm_title")) + "\n")
		sb.WriteString(pWarnStyle.Render(fmt.Sprintf(t("repos.confirm_body"), m.confirmRepo)) + "\n")
		sb.WriteString(common.TBtn(m.id, "s_confirm", t("btn.confirm"), "y", false) + "  " + common.TBtn(m.id, "s_cancel", t("btn.cancel"), "esc", false) + "\n")
	}
	if m.status != "" {
		sb.WriteString("\n" + m.status + "\n")
	}
	return sb.String()
}
func (m pkgModel) render() string {
	if m.showSettings {
		return m.renderSettings()
	}
	if m.showAbout {
		return m.renderAbout()
	}

	w := m.width
	if w <= 0 {
		w = 80
	}
	h := m.height
	if h <= 0 {
		h = 24
	}
	narrow := w < 100
	var sources []string
	if m.xbpsOn {
		sources = append(sources, "xbps")
	}
	if m.appImageOn {
		sources = append(sources, "appimage")
	}
	if m.flatpakOn {
		sources = append(sources, "flatpak")
	}
	if m.nixOn {
		sources = append(sources, "nix")
	}
	if m.guixOn {
		sources = append(sources, "guix")
	}
	source := strings.Join(sources, ",")
	if source == "" {
		source = "xbps"
	}

	title := pTitleStyle.Render(t("app.window")) + "  " + pSubtleStyle.Render("["+source+"]")
	if m.loading {
		title += "  " + pWarnStyle.Render(t("tui.loading"))
	}
	if m.searchMode {
		title += "  " + m.search.View()
	} else if m.search.Value() != "" {
		title += "  " + pSubtleStyle.Render("/"+m.search.Value())
	}
	if len(m.marked) > 0 {
		title += "  " + pWarnStyle.Render(fmt.Sprintf(t("tui.marked"), len(m.marked)))
	}

	// The persistent status controls occupy the final row; allocate the
	// remaining viewport between the package browser and its detail area.
	bodyHeight := h - 2
	if bodyHeight < 4 {
		bodyHeight = 4
	}
	list := m.filtered()
	leftWidth := w
	if !narrow {
		leftWidth = w * 42 / 100
		if leftWidth < 36 {
			leftWidth = 36
		}
		if leftWidth > w-40 {
			leftWidth = w - 40
		}
	}
	listHeight := bodyHeight - 5
	if listHeight < 1 {
		listHeight = 1
	}
	start := 0
	if m.cursor >= listHeight {
		start = m.cursor - listHeight + 1
	}
	var left strings.Builder
	left.WriteString(m.topBar() + "\n")
	left.WriteString(title + "\n")
	left.WriteString(pSubtleStyle.Render(strings.Repeat("─", leftWidth)) + "\n")
	if start > 0 {
		left.WriteString(pSubtleStyle.Render(fmt.Sprintf("↑ %d", start)) + "\n")
	}
	shown := 0
	for i := start; i < len(list) && shown < listHeight; i++ {
		pkg := list[i]
		marker := " "
		if m.marked[pkg.Name] {
			marker = pWarnStyle.Render("M")
		}
		name := pkg.Name
		if pkg.DisplayName != "" {
			name = pkg.DisplayName
		}
		desc := ""
		maxDesc := leftWidth - len(name) - 8
		if !narrow && pkg.ShortDesc != "" && maxDesc > 10 {
			text := pkg.ShortDesc
			if len(text) > maxDesc {
				text = text[:maxDesc-1] + "…"
			}
			desc = "  " + pSubtleStyle.Render(text)
		}
		switch {
		case i == m.cursor:
			left.WriteString(zone.Mark(m.id+pkg.Name, pSelectedStyle.Render(marker+" "+name)) + desc + "\n")
		case pkg.Installed:
			left.WriteString(zone.Mark(m.id+pkg.Name, pInstalledStyle.Render(marker+" "+name)) + desc + "\n")
		default:
			left.WriteString(zone.Mark(m.id+pkg.Name, pNormalStyle.Render(marker+" "+name)) + desc + "\n")
		}
		shown++
	}
	if remaining := len(list) - start - shown; remaining > 0 {
		left.WriteString(pSubtleStyle.Render(fmt.Sprintf("↓ %d", remaining)) + "\n")
	}
	if len(list) == 0 && !m.loading {
		left.WriteString(pSubtleStyle.Render(t("tui.none")) + "\n")
	}

	var right strings.Builder
	if m.detail.Name != "" {
		installed := pSubtleStyle.Render(t("pkg.not_installed"))
		if pkg, ok := m.selectedPkg(); ok && pkg.Installed {
			installed = pSuccessStyle.Render(t("pkg.installed"))
		}
		right.WriteString(pTitleStyle.Render(m.detail.Name) + "  " + installed + "\n")
		if m.detail.Version != "" {
			right.WriteString(pSubtleStyle.Render("v"+m.detail.Version) + "\n")
		}
		if m.detail.ShortDesc != "" {
			right.WriteString(pSubtleStyle.Render(m.detail.ShortDesc) + "\n")
		}
		if m.detail.Architecture != "" {
			right.WriteString(pSubtleStyle.Render(t("tui.architecture")+": "+m.detail.Architecture) + "\n")
		}
		if m.detail.Repository != "" {
			right.WriteString(pSubtleStyle.Render(t("tui.repo")+": "+m.detail.Repository) + "\n")
		}
		size := m.detail.InstalledSize
		if size == "" {
			size = m.detail.FilenameSize
		}
		if size == "" {
			right.WriteString(pStrikeStyle.Render(t("tui.size")+": —") + "\n")
		} else {
			right.WriteString(pSubtleStyle.Render(t("tui.size")+": "+size) + "\n")
		}
		if m.detail.Homepage != "" {
			right.WriteString(pSubtleStyle.Render("🌐 "+m.detail.Homepage) + "\n")
		}
		if len(m.detail.RunDeps) > 0 {
			deps := strings.Join(m.detail.RunDeps, ", ")
			if len(deps) > 100 {
				deps = deps[:100] + "…"
			}
			right.WriteString(pSubtleStyle.Render(t("tui.deps")+": "+deps) + "\n")
		}
	} else {
		right.WriteString(pSubtleStyle.Render(t("tui.none")) + "\n")
	}
	right.WriteString("\n" + m.detailButtons() + "\n")
	if len(m.queue) > 0 {
		right.WriteString(pWarnStyle.Render(fmt.Sprintf(t("pkg.queue_count"), len(m.queue))) + "\n")
		for _, entry := range m.queue {
			symbol := "[+]"
			if entry.Action == "remove" {
				symbol = "[-]"
			}
			right.WriteString(pSubtleStyle.Render(fmt.Sprintf("%s %s", symbol, entry.Name)) + "\n")
		}
	} else {
		right.WriteString(pSubtleStyle.Render(t("pkg.queue_count")) + "\n")
	}
	right.WriteString("\n" + common.TBtn(m.id, "a_copy", t("btn.copy"), "y", false) + "\n")
	if m.output != "" {
		right.WriteString(pSubtleStyle.Render(strings.Repeat("─", max(1, w-leftWidth-3))) + "\n")
		lines := strings.Split(m.output, "\n")
		maxLines := m.outputLines
		if maxLines <= 0 {
			maxLines = 5
		}
		if len(lines) > maxLines {
			lines = lines[len(lines)-maxLines:]
		}
		for _, line := range lines {
			right.WriteString(pSubtleStyle.Render(line) + "\n")
		}
	}

	status := m.bottomBar()
	var body string
	if narrow {
		body = left.String() + "\n" + pSubtleStyle.Render(strings.Repeat("─", w)) + "\n" + right.String()
	} else {
		rightWidth := w - leftWidth - 3
		leftPane := lipgloss.NewStyle().Width(leftWidth).Height(bodyHeight).MaxHeight(bodyHeight).Render(strings.TrimRight(left.String(), "\n"))
		rightPane := lipgloss.NewStyle().Width(rightWidth).Height(bodyHeight).MaxHeight(bodyHeight).PaddingLeft(1).Render(strings.TrimRight(right.String(), "\n"))
		body = lipgloss.JoinHorizontal(lipgloss.Top, leftPane, pSubtleStyle.Render(" │ "), rightPane)
	}
	if narrow {
		// Keep the status/actions visible at the end even when the stacked
		// panels exceed the available terminal height.
		return body + "\n" + pSubtleStyle.Render(strings.Repeat("─", w)) + "\n" + status + "\n"
	}
	return body + "\n" + pSubtleStyle.Render(strings.Repeat("─", w)) + "\n" + status + "\n"
}

func (m pkgModel) renderAbout() string {
	title := pTitleStyle.Render(t("app.window"))
	version := pNormalStyle.Render(t("about.version") + ": " + serman.Version)
	author := pNormalStyle.Render(t("about.author") + ": " + serman.AppAuthor)
	license := pNormalStyle.Render(t("about.license") + ": " + serman.AppLicense)
	description := pSubtleStyle.Render(t("about.description"))
	url := zone.Mark(m.id+"a_about_home", pNormalStyle.Render(serman.AppURL))
	hint := pSubtleStyle.Render(t("about.hint"))
	controls := common.TBtn(m.id, "a_about_home_btn", t("about.open_homepage"), "h", false) + "  " +
		common.TBtn(m.id, "a_about_close", t("btn.close"), "esc", false)
	return "\n" + title + "\n\n" + version + "\n" + author + "\n" + license + "\n\n" + description + "\n\n" + url + "\n" + controls + "\n\n" + hint + "\n"
}

// RunTUI runs the package manager as a standalone Bubbletea application.
func RunTUI() {
	common.EnsureZone()
	p := tea.NewProgram(NewTuiModel())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
