package xbpssrc

import (
	svman "codeberg.org/oSoWoSo/SysMan/plugin"
)

const defaultSearchEngine = "https://duckduckgo.com/?q="

type SrcmanConfig struct {
	SearchEngine string
	DistDir      string
}

func LoadConfig() SrcmanConfig {
	c := svman.LoadSysManConfig()
	se := c.Srcman.SearchEngine
	if se == "" {
		se = defaultSearchEngine
	}
	return SrcmanConfig{
		SearchEngine: se,
		DistDir:      c.Srcman.DistDir,
	}
}

func SaveConfig(cfg SrcmanConfig) error {
	c := svman.LoadSysManConfig()
	c.Srcman.DistDir = cfg.DistDir
	c.Srcman.SearchEngine = cfg.SearchEngine
	return svman.SaveSysManConfig(c)
}
