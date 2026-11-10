package common

import "strings"

// TabIcon returns the compact icon used for a SysMan module tab.
func TabIcon(name string) string {
	switch strings.ToLower(name) {
	case "services", "služby":
		return "⚙"
	case "packages", "balíčky":
		return "📦"
	case "templates", "šablony":
		return "🧰"
	case "system info", "systémové informace":
		return "ⓘ"
	case "users & groups", "uživatelé a skupiny":
		return "👥"
	case "virtual machines", "virtuální počítače", "vms":
		return "▣"
	case "settings", "nastavení":
		return "⚙"
	default:
		return "•"
	}
}

// FormatTabLabel builds a tab label from name according to display options.
func FormatTabLabel(name string, showNames, showIcons bool) string {
	parts := make([]string, 0, 2)
	if showIcons {
		parts = append(parts, TabIcon(name))
	}
	if showNames {
		parts = append(parts, name)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}
