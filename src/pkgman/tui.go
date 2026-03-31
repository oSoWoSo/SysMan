package pkgman

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Styles ────────────────────────────────────────────────────────────

var (
	pSubtle    = lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#585858"}
	pHighlight = lipgloss.AdaptiveColor{Light: "#006688", Dark: "#00DDFF"}
	pDanger    = lipgloss.AdaptiveColor{Light: "#CC3333", Dark: "#FF5555"}
	pSuccess   = lipgloss.AdaptiveColor{Light: "#22AA55", Dark: "#44DD77"}
	pWarn      = lipgloss.AdaptiveColor{Light: "#BB8800", Dark: "#FFCC00"}

	pTitleStyle    = lipgloss.NewStyle().Bold(true).Foreground(pHighlight)
	pSelectedStyle = lipgloss.NewStyle().Bold(true).Foreground(pHighlight).
			Background(lipgloss.AdaptiveColor{Light: "#DDF5FF", Dark: "#003344"}).Padding(0, 1)
	pNormalStyle    = lipgloss.NewStyle().Padding(0, 1)
	pInstalledStyle = lipgloss.NewStyle().Foreground(pSuccess).Padding(0, 1)
	pSubtleStyle    = lipgloss.NewStyle().Foreground(pSubtle)
	pDangerStyle    = lipgloss.NewStyle().Foreground(pDanger).Bold(true)
	pSuccessStyle   = lipgloss.NewStyle().Foreground(pSuccess)
	pWarnStyle      = lipgloss.NewStyle().Foreground(pWarn)
	pHelpStyle      = lipgloss.NewStyle().Foreground(pSubtle)
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
	backend    PkgBackend
	packages   []Package
	cursor     int
	search     textinput.Model
	searchMode bool
	filter     pkgFilter
	marked     map[string]bool // packages toggled for bulk operations
	detail     PackageDetail
	output     string
	status     string
	statusErr  bool
	loading    bool
	running    bool
	width      int
	height     int
}

// NewTuiModel creates an initialized package manager TUI model using the default xbps backend.
func NewTuiModel() tea.Model {
	return NewTuiModelWithBackend(NewXbpsBackend())
}

// NewTuiModelWithBackend creates an initialized TUI model using the provided backend.
// Use this to embed the plugin with a custom package manager.
func NewTuiModelWithBackend(b PkgBackend) tea.Model {
	ti := textinput.New()
	ti.Placeholder = t("tui.search.placeholder")
	return pkgModel{
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
		if msg.err != nil {
			m.status = fmt.Sprintf("%s failed: %s", msg.action, msg.err.Error())
			m.statusErr = true
		} else {
			m.status = fmt.Sprintf("%s OK", msg.action)
			m.statusErr = false
		}
		// Reload package list after install/remove
		m.loading = true
		return m, m.loadPackagesCmd()

	case PackageDetail:
		m.detail = msg
		return m, nil

	case tea.KeyMsg:
		if m.searchMode {
			return m.handleSearchKey(msg)
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

func (m pkgModel) handleSearchKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch key.Type {
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

func (m pkgModel) handleKey(key tea.KeyMsg) (tea.Model, tea.Cmd) {
	list := m.filtered()

	switch key.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

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

	case " ": // toggle mark for bulk operations
		if pkg, ok := m.selectedPkg(); ok {
			if m.marked[pkg.Name] {
				delete(m.marked, pkg.Name)
			} else {
				m.marked[pkg.Name] = true
			}
			// Move cursor down after marking
			if m.cursor < len(list)-1 {
				m.cursor++
			}
		}
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
	}

	if m.running || m.loading {
		return m, nil
	}

	switch key.String() {
	case "i": // install selected or marked
		names := m.targetNames(true)
		if len(names) == 0 {
			return m, nil
		}
		m.running = true
		m.status = fmt.Sprintf(t("tui.status.installing"), strings.Join(names, ", "))
		m.marked = make(map[string]bool)
		return m, m.pkgInstallCmd(names)

	case "d": // delete/remove selected or marked
		names := m.targetNames(false)
		if len(names) == 0 {
			return m, nil
		}
		m.running = true
		m.status = fmt.Sprintf(t("tui.status.removing"), strings.Join(names, ", "))
		m.marked = make(map[string]bool)
		return m, m.pkgRemoveCmd(names)

	case "u": // update all packages
		m.running = true
		m.status = t("tui.status.updating")
		return m, m.pkgUpdateCmd()
	}

	return m, nil
}

// targetNames returns the names to act on.
// If there are marked packages, use those; otherwise use the current selection.
// FilterInstalled: true = only non-installed (for install), false = only installed (for remove).
func (m pkgModel) targetNames(forInstall bool) []string {
	if len(m.marked) > 0 {
		var names []string
		for n := range m.marked {
			names = append(names, n)
		}
		return names
	}
	pkg, ok := m.selectedPkg()
	if !ok {
		return nil
	}
	if forInstall && pkg.Installed {
		return nil // already installed
	}
	if !forInstall && !pkg.Installed {
		return nil // not installed
	}
	return []string{pkg.Name}
}

// ── Commands ──────────────────────────────────────────────────────────

func (m pkgModel) loadPackagesCmd() tea.Cmd {
	b := m.backend
	return func() tea.Msg {
		return pkgLoadedMsg{pkgs: b.List()}
	}
}

func (m pkgModel) queryDetailCmd(name string) tea.Cmd {
	b := m.backend
	return func() tea.Msg {
		return b.Detail(name)
	}
}

func (m pkgModel) pkgInstallCmd(names []string) tea.Cmd {
	b := m.backend
	return func() tea.Msg {
		// io.Discard: capture output (no TTY passthrough) but discard streaming —
		// the full output is returned in pkgDoneMsg for display in the TUI.
		out, err := b.Install(names, io.Discard)
		return pkgDoneMsg{action: "install", output: out, err: err}
	}
}

func (m pkgModel) pkgRemoveCmd(names []string) tea.Cmd {
	b := m.backend
	return func() tea.Msg {
		out, err := b.Remove(names, io.Discard)
		return pkgDoneMsg{action: "remove", output: out, err: err}
	}
}

func (m pkgModel) pkgUpdateCmd() tea.Cmd {
	b := m.backend
	return func() tea.Msg {
		out, err := b.Update(io.Discard)
		return pkgDoneMsg{action: "update", output: out, err: err}
	}
}

// ── View ──────────────────────────────────────────────────────────────

func (m pkgModel) View() string {
	w := m.width
	if w <= 0 {
		w = 80
	}
	narrow := w < 80
	divider := pSubtleStyle.Render(strings.Repeat("─", w))

	// ── Title ─────────────────────────────────────────────────────────
	title := pTitleStyle.Render(t("app.window"))
	title += "  " + pSubtleStyle.Render("["+m.filter.String()+"]")
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
	sb.WriteString(title + "\n")
	sb.WriteString(divider + "\n")

	// ── Package list ──────────────────────────────────────────────────
	// overhead: title(1) + 2×divider(2) + detail≤3 + divider(1) + help(1) + divider(1) + status(1)
	// + sysmanager tab bar (2) = 12; add margin for optional output section
	overhead := 14
	if m.output != "" {
		overhead = 20
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
			sb.WriteString(pSelectedStyle.Render(marker+" "+name) + desc + "\n")
		case pkg.Installed:
			sb.WriteString(pInstalledStyle.Render(marker+" "+name) + desc + "\n")
		default:
			sb.WriteString(pNormalStyle.Render(marker+" "+name) + desc + "\n")
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
		if m.detail.Homepage != "" {
			sb.WriteString("  " + pSubtleStyle.Render("🌐 "+m.detail.Homepage) + "\n")
		}
	}
	sb.WriteString(divider + "\n")

	// ── Help ──────────────────────────────────────────────────────────
	switch {
	case m.running:
		sb.WriteString(pWarnStyle.Render("  "+t("tui.running")) + "\n")
	case narrow:
		sb.WriteString(pHelpStyle.Render("  "+t("tui.help.simple")) + "\n")
	default:
		sb.WriteString(pHelpStyle.Render("  "+t("tui.help.full")) + "\n")
	}

	// ── Output (last command) ─────────────────────────────────────────
	if m.output != "" {
		sb.WriteString(divider + "\n")
		lines := strings.Split(m.output, "\n")
		if len(lines) > 5 {
			lines = lines[len(lines)-5:]
		}
		for _, l := range lines {
			sb.WriteString(pSubtleStyle.Render("  "+l) + "\n")
		}
	}

	// ── Status bar ───────────────────────────────────────────────────
	sb.WriteString(divider + "\n")
	if m.status != "" {
		if m.statusErr {
			sb.WriteString(pDangerStyle.Render("  ✗ "+m.status) + "\n")
		} else {
			sb.WriteString(pSuccessStyle.Render("  ✓ "+m.status) + "\n")
		}
	}

	return sb.String()
}

// RunTUI runs the package manager as a standalone Bubbletea application.
func RunTUI() {
	p := tea.NewProgram(NewTuiModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
