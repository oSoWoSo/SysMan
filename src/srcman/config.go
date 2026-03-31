package srcman

import (
	"codeberg.org/oSoWoSo/SysMan/src/common"
)

const defaultSearchEngine = "https://duckduckgo.com/?q="

type SrcmanConfig struct {
	SearchEngine string
	DistDir      string
}

func LoadConfig() SrcmanConfig {
	c := common.LoadSysManConfig()
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
	c := common.LoadSysManConfig()
	c.Srcman.DistDir = cfg.DistDir
	c.Srcman.SearchEngine = cfg.SearchEngine
	return common.SaveSysManConfig(c)
}
