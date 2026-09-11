package infman

import (
	"fmt"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"codeberg.org/oSoWoSo/SysMan/src/common"
	zone "github.com/lrstanley/bubblezone/v2"
)

// NewTuiModel returns a Bubbletea tea.Model showing system information.
func NewTuiModel() tea.Model {
	zone.NewGlobal()
	return sysInfoModel{id: zone.NewPrefix()}
}

type fetchDoneMsg struct{ entries []infoEntry }
type fetchOutputMsg struct {
	output string
	source string
}

type sysInfoModel struct {
	id          string
	entries     []infoEntry
	loaded      bool
	fetchOutput string
	fetchSource string
	showAbout   bool
	width       int
	height      int
}

func (m sysInfoModel) Init() tea.Cmd {
	return tea.Batch(
		func() tea.Msg { return fetchDoneMsg{collectNative()} },
		func() tea.Msg {
			out, ok := RunFetch()
			if !ok {
				return fetchOutputMsg{}
			}
			src := "fastfetch"
			if _, err := exec.LookPath("fastfetch"); err != nil {
				src = "neofetch"
			}
			return fetchOutputMsg{output: out, source: src}
		},
	)
}

func (m sysInfoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case fetchDoneMsg:
		m.entries = msg.entries
		m.loaded = true
	case fetchOutputMsg:
		m.fetchOutput = msg.output
		m.fetchSource = msg.source
	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			break
		}
		if zone.Get(m.id + "a_quit").InBounds(msg) {
			return m, tea.Quit
		}
		if zone.Get(m.id + "a_about").InBounds(msg) {
			m.showAbout = true
			return m, nil
		}
	case tea.KeyPressMsg:
		if m.showAbout {
			m.showAbout = false
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "?":
			m.showAbout = true
			return m, nil
		}
	}
	return m, nil
}

// View implements tea.Model; renders the info listing in the alt screen.
func (m sysInfoModel) View() tea.View {
	v := tea.NewView(zone.Scan(m.render()))
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

func (m sysInfoModel) render() string {
	if m.showAbout {
		return m.renderAbout()
	}
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

	if m.fetchOutput != "" {
		sb.WriteString("\n")
		sb.WriteString(xSubtleStyle.Render("  ── "+m.fetchSource+" ──") + "\n")
		stripped := AnsiRe.ReplaceAllString(m.fetchOutput, "")
		for _, line := range strings.Split(stripped, "\n") {
			if line != "" {
				sb.WriteString("  " + line + "\n")
			}
		}
	}

	sb.WriteString("\n")
	quitBtn := common.TBtn(m.id, "a_quit", t("action.quit"), "q", false)
	aboutBtn := common.TBtn(m.id, "a_about", t("btn.about"), "?", false)
	status := ""
	if m.loaded {
		status = lipgloss.NewStyle().Foreground(lipgloss.Color("#55FF55")).Render(fmt.Sprintf("  %d %s", len(m.entries), t("info.entries")))
	}
	sep := strings.Repeat("─", max(0, m.width-2))
	sb.WriteString(sep + "\n")
	sb.WriteString(status + "  " + aboutBtn + " " + quitBtn + "\n")
	return sb.String()
}

// renderAbout renders the about screen.
func (m sysInfoModel) renderAbout() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00DDFF")).Render(t("app.title") + " - " + t("app.subtitle"))
	version := lipgloss.NewStyle().Render(t("about.version") + ": " + common.Version)
	author := lipgloss.NewStyle().Render(t("about.author") + ": " + common.AppAuthor)
	license := lipgloss.NewStyle().Render(t("about.license") + ": " + common.AppLicense)
	url := lipgloss.NewStyle().Render(common.AppURL)
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("#9B9B9B")).Render(t("about.hint"))
	return "\n" + title + "\n\n" + version + "\n" + author + "\n" + license + "\n" + url + "\n\n" + hint + "\n"
}
