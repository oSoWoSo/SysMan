package xbpssrc

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const defaultSearchEngine = "https://duckduckgo.com/?q="

// SrcmanConfig holds user-configurable settings for srcman.
type SrcmanConfig struct {
	SearchEngine string `yaml:"search_engine"`
}

// configPath returns the path to ~/.config/SysMan/srcman.conf.
func configPath() string {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(cfg, "SysMan", "srcman.conf")
}

// LoadConfig reads srcman.conf, writing defaults on first run.
func LoadConfig() SrcmanConfig {
	c := SrcmanConfig{SearchEngine: defaultSearchEngine}
	path := configPath()
	if path == "" {
		return c
	}
	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		// First run — write defaults.
		_ = os.MkdirAll(filepath.Dir(path), 0o755)
		if out, merr := yaml.Marshal(c); merr == nil {
			_ = os.WriteFile(path, out, 0o644)
		}
		return c
	}
	_ = yaml.Unmarshal(data, &c)
	if c.SearchEngine == "" {
		c.SearchEngine = defaultSearchEngine
	}
	return c
}
