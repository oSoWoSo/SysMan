package serman

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SysManConfig is the SysMan configuration.
type SysManConfig struct {
	Svman      SvmanConfig      `yaml:"svman,omitempty"`
	Pkgman     PkgmanConfig     `yaml:"pkgman,omitempty"`
	Srcman     Config     `yaml:"srcman,omitempty"`
	Vmman      VmmanConfig      `yaml:"vmman,omitempty"`
	Usergroups UsergroupsConfig `yaml:"usergroups,omitempty"`
	Sysinfo    SysinfoConfig    `yaml:"sysinfo,omitempty"`
}

// SvmanConfig is the services configuration.
type SvmanConfig struct {
	ServiceDir     string `yaml:"service_dir,omitempty"`
	ServiceDestDir string `yaml:"service_dest_dir,omitempty"`
}

// PkgmanConfig is the packages configuration.
type PkgmanConfig struct {
}

// Config is the templates configuration.
type Config struct {
	DistDir      string `yaml:"dist_dir,omitempty"`
	SearchEngine string `yaml:"search_engine,omitempty"`
}

// VmmanConfig is the VM configuration.
type VmmanConfig struct {
	VMDir string `yaml:"vm_dir,omitempty"`
}

// UsergroupsConfig is the users & groups configuration.
type UsergroupsConfig struct {
}

// SysinfoConfig is the system info configuration.
type SysinfoConfig struct {
}

func sysmanConfigPath() string {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(cfg, "sysman", "sysman.conf")
}

// LoadSysManConfig loads the configuration.
func LoadSysManConfig() SysManConfig {
	var c SysManConfig
	path := sysmanConfigPath()
	if path == "" {
		return c
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return c
	}

	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return c
	}

	if _, hasSvman := raw["svman"]; !hasSvman {
		if serviceDir, ok := raw["service_dir"].(string); ok {
			c.Svman.ServiceDir = serviceDir
		}
		if serviceDestDir, ok := raw["service_dest_dir"].(string); ok {
			c.Svman.ServiceDestDir = serviceDestDir
		}
		if srcmanDistDir, ok := raw["srcman_dist_dir"].(string); ok {
			c.Srcman.DistDir = srcmanDistDir
		}
		if srcmanSearchEngine, ok := raw["srcman_search_engine"].(string); ok {
			c.Srcman.SearchEngine = srcmanSearchEngine
		}
		if vmmanVMDir, ok := raw["vmman_vm_dir"].(string); ok {
			c.Vmman.VMDir = vmmanVMDir
		}
		return c
	}

	_ = yaml.Unmarshal(data, &c)
	return c
}

// SaveSysManConfig saves the configuration.
func SaveSysManConfig(cfg SysManConfig) error {
	path := sysmanConfigPath()
	if path == "" {
		return os.ErrNotExist
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	existing := LoadSysManConfig()

	if cfg.Svman.ServiceDir == "" {
		cfg.Svman.ServiceDir = existing.Svman.ServiceDir
	}
	if cfg.Svman.ServiceDestDir == "" {
		cfg.Svman.ServiceDestDir = existing.Svman.ServiceDestDir
	}
	if cfg.Srcman.DistDir == "" {
		cfg.Srcman.DistDir = existing.Srcman.DistDir
	}
	if cfg.Srcman.SearchEngine == "" {
		cfg.Srcman.SearchEngine = existing.Srcman.SearchEngine
	}
	if cfg.Vmman.VMDir == "" {
		cfg.Vmman.VMDir = existing.Vmman.VMDir
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}
