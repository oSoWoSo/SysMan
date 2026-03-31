package srcman

// Package srcman provides templates configuration.

import (
	"codeberg.org/oSoWoSo/SysMan/src/common"
)

const defaultSearchEngine = "https://repology.org/projects/?search="

// Config holds the srcman configuration.
type Config struct {
	SearchEngine string
	DistDir      string
	ForkURL      string
}

// LoadConfig loads the srcman configuration.
func LoadConfig() Config {
	c := common.LoadSysManConfig()
	se := c.Srcman.SearchEngine
	if se == "" {
		se = defaultSearchEngine
	}
	return Config{
		SearchEngine: se,
		DistDir:      c.Srcman.DistDir,
		ForkURL:      c.Srcman.ForkURL,
	}
}

// SaveConfig saves the srcman configuration.
func SaveConfig(cfg Config) error {
	c := common.LoadSysManConfig()
	c.Srcman.DistDir = cfg.DistDir
	c.Srcman.SearchEngine = cfg.SearchEngine
	c.Srcman.ForkURL = cfg.ForkURL
	return common.SaveSysManConfig(c)
}
