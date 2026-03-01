package xbpssrc

import (
	svman "codeberg.org/oSoWoSo/SysMan/plugin"
)

const defaultSearchEngine = "https://duckduckgo.com/?q="

// SrcmanConfig holds srcman-specific settings extracted from the shared sysman.conf.
type SrcmanConfig struct {
	SearchEngine string
	DistDir      string
}

// LoadConfig reads srcman settings from the shared sysman.conf.
func LoadConfig() SrcmanConfig {
	c := svman.LoadSysManConfig()
	se := c.SrcmanSearchEngine
	if se == "" {
		se = defaultSearchEngine
	}
	return SrcmanConfig{
		SearchEngine: se,
		DistDir:      c.SrcmanDistDir,
	}
}

// SaveConfig writes srcman settings back into the shared sysman.conf,
// preserving any keys set by other SysMan components.
func SaveConfig(cfg SrcmanConfig) error {
	c := svman.LoadSysManConfig()
	c.SrcmanDistDir = cfg.DistDir
	c.SrcmanSearchEngine = cfg.SearchEngine
	return svman.SaveSysManConfig(c)
}
