package common

import (
	"fmt"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	zone "github.com/lrstanley/bubblezone/v2"
)

type sysmanSettingsTUIModel struct {
	id        string
	fields    []textinput.Model
	focus     int
	width     int
	status    string
	statusErr bool
}

type sysmanSettingField struct {
	label string
	group string
}

var sysmanSettingLabels = []sysmanSettingField{
	{label: "Service Dir", group: "Serman"},
	{label: "Service Dest Dir", group: "Serman"},
	{label: "User Service Dir", group: "Serman"},
	{label: "User Service Dest Dir", group: "Serman"},
	{label: "void-packages dir", group: "Srcman"},
	{label: "Search engine URL", group: "Srcman"},
	{label: "Fork URL (git push)", group: "Srcman"},
	{label: "VM dir", group: "Vmsman"},
	{label: "AppImage Dir", group: "Pkgman"},
	{label: "Language dir", group: "Common"},
	{label: "Show tab names (true/false)", group: "Common"},
	{label: "Show tab icons (true/false)", group: "Common"},
}

// NewSysmanSettingsTUI returns a terminal settings model for the global
// settings page exposed by the SysMan GUI.
func NewSysmanSettingsTUI() tea.Model {
	zone.NewGlobal()
	cfg := LoadSysManConfig()
	values := []string{
		cfg.Serman.ServiceDir,
		cfg.Serman.ServiceDestDir,
		cfg.Serman.UserServiceDir,
		cfg.Serman.UserServiceDestDir,
		cfg.Srcman.DistDir,
		cfg.Srcman.SearchEngine,
		cfg.Srcman.ForkURL,
		cfg.Vmsman.VMDir,
		cfg.Pkgman.AppImageDir,
		cfg.LangDir,
		strconv.FormatBool(cfg.ShowTabNames),
		strconv.FormatBool(cfg.ShowTabIcons),
	}
	m := sysmanSettingsTUIModel{id: "sysman-settings:" + zone.NewPrefix(), width: 80}
	m.fields = make([]textinput.Model, len(values))
	for i, value := range values {
		input := textinput.New()
		input.SetValue(value)
		input.SetWidth(44)
		if i == 0 {
			input.Focus()
		} else {
			input.Blur()
		}
		m.fields[i] = input
	}
	return m
}

func (m sysmanSettingsTUIModel) Init() tea.Cmd { return nil }

func (m sysmanSettingsTUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		fieldWidth := min(max(m.width-34, 12), 64)
		for i := range m.fields {
			m.fields[i].SetWidth(fieldWidth)
		}
	case tea.MouseClickMsg:
		if msg.Button != tea.MouseLeft {
			return m, nil
		}
		if zone.Get(m.id + "save").InBounds(msg) {
			m.save()
			return m, nil
		}
		for i := range m.fields {
			if zone.Get(fmt.Sprintf("%sfield_%d", m.id, i)).InBounds(msg) {
				m.focusField(i)
				return m, nil
			}
		}
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+s":
			m.save()
			return m, nil
		case "tab", "down", "enter":
			if m.focus == len(m.fields)-1 {
				m.save()
			} else {
				m.focusField(m.focus + 1)
			}
			return m, nil
		case "shift+tab", "up":
			m.focusField(max(m.focus-1, 0))
			return m, nil
		}
		var cmd tea.Cmd
		m.fields[m.focus], cmd = m.fields[m.focus].Update(msg)
		m.status = ""
		return m, cmd
	}
	return m, nil
}

func (m *sysmanSettingsTUIModel) focusField(idx int) {
	if idx < 0 || idx >= len(m.fields) {
		return
	}
	m.fields[m.focus].Blur()
	m.focus = idx
	m.fields[m.focus].Focus()
}

func (m *sysmanSettingsTUIModel) save() {
	cfg := LoadSysManConfig()
	cfg.Serman.ServiceDir = strings.TrimSpace(m.fields[0].Value())
	cfg.Serman.ServiceDestDir = strings.TrimSpace(m.fields[1].Value())
	cfg.Serman.UserServiceDir = strings.TrimSpace(m.fields[2].Value())
	cfg.Serman.UserServiceDestDir = strings.TrimSpace(m.fields[3].Value())
	cfg.Srcman.DistDir = strings.TrimSpace(m.fields[4].Value())
	cfg.Srcman.SearchEngine = strings.TrimSpace(m.fields[5].Value())
	cfg.Srcman.ForkURL = strings.TrimSpace(m.fields[6].Value())
	cfg.Vmsman.VMDir = strings.TrimSpace(m.fields[7].Value())
	cfg.Pkgman.AppImageDir = strings.TrimSpace(m.fields[8].Value())
	cfg.LangDir = strings.TrimSpace(m.fields[9].Value())
	if value, err := strconv.ParseBool(strings.TrimSpace(m.fields[10].Value())); err == nil {
		cfg.ShowTabNames = value
	}
	if value, err := strconv.ParseBool(strings.TrimSpace(m.fields[11].Value())); err == nil {
		cfg.ShowTabIcons = value
	}
	if err := SaveSysManConfig(cfg); err != nil {
		m.status = fmt.Sprintf("Save failed: %v", err)
		m.statusErr = true
		return
	}
	m.status = "Settings saved"
	m.statusErr = false
}

func (m sysmanSettingsTUIModel) View() tea.View { return m.view(true) }

// EmbeddedView returns the unscanned settings view for composition in SysMan.
func (m sysmanSettingsTUIModel) EmbeddedView() tea.View { return m.view(false) }

func (m sysmanSettingsTUIModel) view(scan bool) tea.View {
	var sb strings.Builder
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00DDFF")).Render("System Manager Settings")
	sb.WriteString(title + "\n\n")
	lastGroup := ""
	labelWidth := 26
	if m.width < 60 {
		labelWidth = 18
	}
	for i, field := range sysmanSettingLabels {
		if field.group != lastGroup {
			sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00B8D4")).Render(field.group) + "\n")
			lastGroup = field.group
		}
		label := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("%-*s", labelWidth, field.label))
		row := label + " " + m.fields[i].View()
		sb.WriteString(zone.Mark(fmt.Sprintf("%sfield_%d", m.id, i), row) + "\n")
	}
	sb.WriteString("\n" + TBtn(m.id, "save", "Save", "ctrl+s", false) + "\n")
	if m.status != "" {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("#55FF55"))
		if m.statusErr {
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555"))
		}
		sb.WriteString(style.Render(m.status) + "\n")
	}
	sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render("Tab/↑/↓: move between fields  Enter/last field: save  Esc: return to modules"))
	content := sb.String()
	if scan {
		content = zone.Scan(content)
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
