package plugin

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SysManConfig holds all user-configurable settings shared across SysMan components.
// Each program reads only the keys it understands; unknown keys are preserved on save.
type SysManConfig struct {
	// srcman (xbps-src template manager)
	SrcmanDistDir      string `yaml:"srcman_dist_dir,omitempty"`
	SrcmanSearchEngine string `yaml:"srcman_search_engine,omitempty"`
}

// configPath returns ~/.config/sysman/sysman.conf.
func sysmanConfigPath() string {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(cfg, "sysman", "sysman.conf")
}

// LoadSysManConfig reads sysman.conf. Returns defaults if the file does not exist.
func LoadSysManConfig() SysManConfig {
	var c SysManConfig
	path := sysmanConfigPath()
	if path == "" {
		return c
	}
	data, err := os.ReadFile(path) //nolint:gosec
	if err != nil {
		return c
	}
	_ = yaml.Unmarshal(data, &c)
	return c
}

// SaveSysManConfig writes cfg to sysman.conf.
func SaveSysManConfig(cfg SysManConfig) error {
	path := sysmanConfigPath()
	if path == "" {
		return os.ErrNotExist
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644) //nolint:gosec
}
