package pkgman

import (
	"fmt"
	"io"
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

	pTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(pHighlight)
	pSelectedStyle = lipgloss.NewStyle().Bold(true).Foreground(pHighlight).
			Background(compat.AdaptiveColor{Light: lipgloss.Color("#DDF5FF"), Dark: lipgloss.Color("#003344")}).Padding(0, 1)
	pNormalStyle    = lipgloss.NewStyle().Padding(0, 1)
	pInstalledStyle = lipgloss.NewStyle().Foreground(pSuccess).Padding(0, 1)
	pSubtleStyle    = lipgloss.NewStyle().Foreground(pSubtle)
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
	id           string
	backend      PkgBackend
	packages     []Package
	cursor       int
	search       textinput.Model
	searchMode   bool
	filter       pkgFilter
	appImageOn   bool
	marked       map[string]bool
	queue        []QueueEntry
	detail       PackageDetail
	output       string
	status       string
	statusErr    bool
	loading      bool
	running      bool
	showAbout    bool
	outputLines int
	width        int
	height       int
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
	return pkgModel{
		id:      zone.NewPrefix(),
		backend: b,
		search:  ti,
		marked:  make(map[string]bool),
		loading: true,
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

// toggleQueue adds a package to the queue if not present, or removes it if already queued.
func (m *pkgModel) toggleQueue(pkg Package) {
	idx, _ := m.inQueue(pkg.Name)
	if idx >= 0 {
		m.queue = append(m.queue[:idx], m.queue[idx+1:]...)
		m.status = fmt.Sprintf("Removed %s from queue", pkg.Name)
	} else {
		action := "install"
		if pkg.Installed {
			action = "remove"
		}
		m.queue = append(m.queue, QueueEntry{Name: pkg.Name, Action: action})
		m.status = fmt.Sprintf("Added %s to queue (%s)", pkg.Name, action)
	}
	m.statusErr = false
}

// ── tea.Model ─────────────────────────────────────────────────────────

func (m pkgModel) Init() tea.Cmd {
	return m.loadPackagesCmd()
}

func (m pkgModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		m.queue = nil
		if msg.err != nil {
			m.status = fmt.Sprintf("%s failed: %s", msg.action, msg.err.Error())
			m.statusErr = true
		} else {
			m.status = fmt.Sprintf("%s OK", msg.action)
			m.statusErr = false
		}
		m.loading = true
		return m, m.loadPackagesCmd()

	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			break
		}
		if zone.Get(m.id + "f_all").InBounds(msg) {
			m.filter = FilterAll
			m.cursor = 0
			m.marked = make(map[string]bool)
			if pkg, ok := m.selectedPkg(); ok {
				return m, m.queryDetailCmd(pkg.Name)
			}
			return m, nil
		}
		if zone.Get(m.id + "f_installed").InBounds(msg) {
			m.filter = FilterInstalled
			m.cursor = 0
			m.marked = make(map[string]bool)
			if pkg, ok := m.selectedPkg(); ok {
				return m, m.queryDetailCmd(pkg.Name)
			}
			return m, nil
		}
		if zone.Get(m.id + "f_available").InBounds(msg) {
			m.filter = FilterAvailable
			m.cursor = 0
			m.marked = make(map[string]bool)
			if pkg, ok := m.selectedPkg(); ok {
				return m, m.queryDetailCmd(pkg.Name)
			}
			return m, nil
		}
		if zone.Get(m.id + "f_appimage").InBounds(msg) {
			m.appImageOn = !m.appImageOn
			if m.appImageOn {
				m.backend = NewAppImageBackend()
			} else {
				m.backend = NewXbpsBackend()
			}
			m.loading = true
			m.cursor = 0
			m.packages = nil
			m.marked = make(map[string]bool)
			m.queue = nil
			return m, m.loadPackagesCmd()
		}
		if zone.Get(m.id + "a_toggle").InBounds(msg) {
			if pkg, ok := m.selectedPkg(); ok {
				m.toggleQueue(pkg)
				return m, nil
			}
		}
		if zone.Get(m.id + "a_update").InBounds(msg) {
			m.running = true
			m.status = t("tui.status.updating")
			return m, m.pkgUpdateCmd()
		}
		if zone.Get(m.id + "a_queue_apply").InBounds(msg) {
			if len(m.queue) > 0 {
				return m, m.applyQueueCmd()
			}
		}
		if zone.Get(m.id + "a_queue_clear").InBounds(msg) {
			m.queue = nil
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
			if m.output != "" {
				if err := common.CopyToClipboard(m.output); err != nil {
					m.status = t("action.copy_err")
					m.statusErr = true
				} else {
					m.status = t("action.copied")
					m.statusErr = false
				}
			}
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
		if m.searchMode {
			return m.handleSearchKey(msg)
		}
		if m.showAbout {
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
		if m.detail.Homepage != "" {
			m.backend.OpenURL(m.detail.Homepage)
		}
		return m, nil

	case "tab": // cycle filter: all → installed → available → all
		m.filter = m.filter.next()
		m.cursor = 0
		m.marked = make(map[string]bool)
		if pkg, ok := m.selectedPkg(); ok {
			return m, m.queryDetailCmd(pkg.Name)
		}
		return m, nil

	case "r": // reload
		m.loading = true
		m.status = t("tui.status.reloading")
		return m, m.loadPackagesCmd()

	case "b": // toggle xbps / AppImage
		m.appImageOn = !m.appImageOn
		if m.appImageOn {
			m.backend = NewAppImageBackend()
		} else {
			m.backend = NewXbpsBackend()
		}
		m.loading = true
		m.cursor = 0
		m.packages = nil
		m.marked = make(map[string]bool)
		m.queue = nil
		return m, m.loadPackagesCmd()

	case "y": // copy output to clipboard
		if m.output != "" {
			if err := common.CopyToClipboard(m.output); err != nil {
				m.status = t("action.copy_err")
				m.statusErr = true
			} else {
				m.status = t("action.copied")
				m.statusErr = false
			}
			return m, nil
		}
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

	case "a": // apply queue
		if len(m.queue) == 0 {
			return m, nil
		}
		return m, m.applyQueueCmd()

	case "u": // update all packages
		m.running = true
		m.status = t("tui.status.updating")
		return m, m.pkgUpdateCmd()
	}

	return m, nil
}

// ── Commands ──────────────────────────────────────────────────────────

func (m pkgModel) loadPackagesCmd() tea.Cmd {
	b := m.backend
	return func() tea.Msg {
		b.Reload()
		return pkgLoadedMsg{pkgs: b.List()}
	}
}

func (m pkgModel) queryDetailCmd(name string) tea.Cmd {
	b := m.backend
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

// applyQueueCmd executes all queued install/remove operations.
func (m pkgModel) applyQueueCmd() tea.Cmd {
	b := m.backend
	q := make([]QueueEntry, len(m.queue))
	copy(q, m.queue)
	return func() tea.Msg {
		var installs, removes []string
		for _, e := range q {
			switch e.Action {
			case "install":
				installs = append(installs, e.Name)
			case "remove":
				removes = append(removes, e.Name)
			}
		}
		var allOutput strings.Builder
		var firstErr error
		if len(installs) > 0 {
			out, err := b.Install(installs, io.Discard)
			if out != "" {
				allOutput.WriteString(out + "\n")
			}
			if err != nil && firstErr == nil {
				firstErr = err
			}
		}
		if len(removes) > 0 {
			out, err := b.Remove(removes, io.Discard)
			if out != "" {
				allOutput.WriteString(out + "\n")
			}
			if err != nil && firstErr == nil {
				firstErr = err
			}
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
		common.TBtn(m.id, "f_appimage", t("filter.appimage"), "b", m.appImageOn),
	)
}

// detailButtons renders action buttons inside the right panel detail area.
func (m pkgModel) detailButtons() string {
	var buttons []string
	if pkg, ok := m.selectedPkg(); ok {
		idx, _ := m.inQueue(pkg.Name)
		if idx >= 0 {
			buttons = append(buttons, common.TBtn(m.id, "a_toggle", t("action.dequeue"), "space", false))
		} else if !pkg.Installed {
			buttons = append(buttons, common.TBtn(m.id, "a_toggle", t("action.enqueue_install"), "space", false))
		} else {
			buttons = append(buttons, common.TBtn(m.id, "a_toggle", t("action.enqueue_remove"), "space", false))
		}
	}
	if len(m.queue) > 0 {
		buttons = append(buttons,
			common.TBtn(m.id, "a_queue_apply", t("btn.queue_apply"), "a", false),
			common.TBtn(m.id, "a_queue_clear", t("btn.queue_clear"), "esc", false),
		)
	}
	buttons = append(buttons, common.TBtn(m.id, "a_update", t("action.update"), "u", false))
	return common.TBtnBar(buttons...)
}

// bottomBar renders the status bar at the bottom of the TUI.
func (m pkgModel) bottomBar() string {
	if m.running {
		return pWarnStyle.Render("  "+t("tui.running")) + "  " + common.TBtn(m.id, "a_about", t("btn.about"), "?", false) + " " + common.TBtn(m.id, "a_quit", t("action.quit"), "q", false)
	}
	var status string
	if m.status != "" {
		if m.statusErr {
			status = pDangerStyle.Render("  ✗ " + m.status)
		} else {
			status = pSuccessStyle.Render("  ✓ " + m.status)
		}
	}
	var copyBtn string
	if m.output != "" {
		copyBtn = " " + common.TBtn(m.id, "a_copy", t("btn.copy"), "y", false)
	}
	return status + "  " + copyBtn + " " + common.TBtn(m.id, "a_about", t("btn.about"), "?", false) + " " + common.TBtn(m.id, "a_quit", t("action.quit"), "q", false)
}

// ── View ──────────────────────────────────────────────────────────────

func (m pkgModel) View() tea.View {
	v := tea.NewView(zone.Scan(m.render()))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m pkgModel) render() string {
	if m.showAbout {
		return m.renderAbout()
	}
	w := m.width
	if w <= 0 {
		w = 80
	}
	narrow := w < 80
	divider := pSubtleStyle.Render(strings.Repeat("─", w))

	// ── Title ─────────────────────────────────────────────────────────
	source := "xbps"
	if m.appImageOn {
		source = "appimage"
	}
	title := pTitleStyle.Render(t("app.window"))
	title += "  " + pSubtleStyle.Render("["+source+"]")
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

	var sb strings.Builder
	sb.WriteString(m.topBar() + "\n")
	sb.WriteString(title + "\n")
	sb.WriteString(divider + "\n")

	// ── Package list ──────────────────────────────────────────────────
	overhead := m.listOverhead()
	if m.output != "" {
		overhead += 6
	}
	list := m.filtered()
	listHeight := m.height - overhead
	if listHeight < 4 {
		listHeight = 4
	}

	start := 0
	if m.cursor >= listHeight {
		start = m.cursor - listHeight + 1
	}
	if start > 0 {
		sb.WriteString(pSubtleStyle.Render(fmt.Sprintf("  ↑ %d", start)) + "\n")
	}

	shown := 0
	for i := start; i < len(list) && shown < listHeight; i++ {
		pkg := list[i]
		marker := " "
		if m.marked[pkg.Name] {
			marker = pWarnStyle.Render("M")
		}
		name := pkg.Name
		desc := ""
		if !narrow && pkg.ShortDesc != "" {
			maxDesc := w - len(name) - 8
			if maxDesc > 10 {
				if len(pkg.ShortDesc) > maxDesc {
					desc = "  " + pSubtleStyle.Render(pkg.ShortDesc[:maxDesc]+"…")
				} else {
					desc = "  " + pSubtleStyle.Render(pkg.ShortDesc)
				}
			}
		}

		switch {
		case i == m.cursor:
			sb.WriteString(zone.Mark(m.id+name, pSelectedStyle.Render(marker+" "+name)) + desc + "\n")
		case pkg.Installed:
			sb.WriteString(zone.Mark(m.id+name, pInstalledStyle.Render(marker+" "+name)) + desc + "\n")
		default:
			sb.WriteString(zone.Mark(m.id+name, pNormalStyle.Render(marker+" "+name)) + desc + "\n")
		}
		shown++
	}
	if remaining := len(list) - (start + shown); remaining > 0 {
		sb.WriteString(pSubtleStyle.Render(fmt.Sprintf("  ↓ %d", remaining)) + "\n")
	}
	if len(list) == 0 && !m.loading {
		sb.WriteString(pSubtleStyle.Render("  "+t("tui.none")) + "\n")
	}
	sb.WriteString(divider + "\n")

	// ── Detail panel ─────────────────────────────────────────────────
	if m.detail.Name != "" {
		installed := ""
		if pkg, ok := m.selectedPkg(); ok && pkg.Installed {
			installed = "  " + pSuccessStyle.Render(t("tui.installed"))
		}
		detail := pTitleStyle.Render(m.detail.Name)
		if m.detail.Version != "" {
			detail += "  " + pSubtleStyle.Render("v"+m.detail.Version)
		}
		detail += installed
		sb.WriteString("  " + detail + "\n")
		if m.detail.ShortDesc != "" {
			sb.WriteString("  " + pSubtleStyle.Render(m.detail.ShortDesc) + "\n")
		}
		if m.detail.Architecture != "" {
			sb.WriteString("  " + pSubtleStyle.Render(t("tui.repo")+": "+m.detail.Architecture) + "\n")
		}
		if m.detail.Repository != "" {
			sb.WriteString("  " + pSubtleStyle.Render(t("tui.repo")+": "+m.detail.Repository) + "\n")
		}
		if m.detail.InstalledSize != "" {
			sb.WriteString("  " + pSubtleStyle.Render(t("tui.size")+": "+m.detail.InstalledSize) + "\n")
		}
		if m.detail.Homepage != "" {
			sb.WriteString("  " + pSubtleStyle.Render("🌐 "+m.detail.Homepage) + "\n")
		}
		if len(m.detail.RunDeps) > 0 {
			deps := strings.Join(m.detail.RunDeps, ", ")
			if len(deps) > 100 {
				deps = deps[:100] + "…"
			}
			sb.WriteString("  " + pSubtleStyle.Render(t("tui.deps")+": "+deps) + "\n")
		}
		sb.WriteString("\n  " + m.detailButtons() + "\n")
		if len(m.queue) > 0 {
			sb.WriteString(pWarnStyle.Render(fmt.Sprintf("  %d in queue", len(m.queue))) + "\n")
			for _, e := range m.queue {
				symbol := "[+]"
				if e.Action == "remove" {
					symbol = "[-]"
				}
				sb.WriteString(pSubtleStyle.Render(fmt.Sprintf("    %s %s", symbol, e.Name)) + "\n")
			}
		}
	}
	sb.WriteString(divider + "\n")

	// ── Output (last command) ─────────────────────────────────────────
	if m.output != "" {
		sb.WriteString(divider + "\n")
		lines := strings.Split(m.output, "\n")
		maxLines := m.outputLines
		if maxLines <= 0 {
			maxLines = 5
		}
		if len(lines) > maxLines {
			lines = lines[len(lines)-maxLines:]
		}
		for _, l := range lines {
			sb.WriteString(pSubtleStyle.Render("  "+l) + "\n")
		}
	}

	// ── Status bar ────────────────────────────────────────────────────
	sb.WriteString(divider + "\n")
	sb.WriteString(m.bottomBar() + "\n")

	return sb.String()
}

func (m pkgModel) renderAbout() string {
	title := pTitleStyle.Render(t("app.window"))
	version := pNormalStyle.Render(t("about.version") + ": " + serman.Version)
	author := pNormalStyle.Render(t("about.author") + ": " + serman.AppAuthor)
	license := pNormalStyle.Render(t("about.license") + ": " + serman.AppLicense)
	url := pNormalStyle.Render(serman.AppURL)
	hint := pSubtleStyle.Render(t("about.hint"))
	return "\n" + title + "\n\n" + version + "\n" + author + "\n" + license + "\n" + url + "\n\n" + hint + "\n"
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
