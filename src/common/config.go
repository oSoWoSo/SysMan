package common

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type SysManConfig struct {
	Serman SermanConfig `yaml:"serman,omitempty"`
	Pkgman PkgmanConfig `yaml:"pkgman,omitempty"`
	Srcman SrcmanConfig `yaml:"srcman,omitempty"`
	Vmsman VmsmanConfig `yaml:"vmsman,omitempty"`
	Ugsman UgsmanConfig `yaml:"ugsman,omitempty"`
	Infman InfmanConfig `yaml:"infman,omitempty"`
}

type SermanConfig struct {
	ServiceDir     string `yaml:"service_dir,omitempty"`
	ServiceDestDir string `yaml:"service_dest_dir,omitempty"`
}

type PkgmanConfig struct {
}

type SrcmanConfig struct {
	DistDir      string `yaml:"dist_dir,omitempty"`
	SearchEngine string `yaml:"search_engine,omitempty"`
}

type VmsmanConfig struct {
	VmDir string `yaml:"vm_dir,omitempty"`
}

type UgsmanConfig struct {
}

type InfmanConfig struct {
}

func sysmanConfigPath() string {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(cfg, "sysman", "sysman.conf")
}

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

	if _, hasSerman := raw["serman"]; !hasSerman {
		if serviceDir, ok := raw["service_dir"].(string); ok {
			c.Serman.ServiceDir = serviceDir
		}
		if serviceDestDir, ok := raw["service_dest_dir"].(string); ok {
			c.Serman.ServiceDestDir = serviceDestDir
		}
		if srcmanDistDir, ok := raw["srcman_dist_dir"].(string); ok {
			c.Srcman.DistDir = srcmanDistDir
		}
		if srcmanSearchEngine, ok := raw["srcman_search_engine"].(string); ok {
			c.Srcman.SearchEngine = srcmanSearchEngine
		}
		if vmsmanVmDir, ok := raw["vmsman_vm_dir"].(string); ok {
			c.Vmsman.VmDir = vmsmanVmDir
		}
		return c
	}

	_ = yaml.Unmarshal(data, &c)
	return c
}

func SaveSysManConfig(cfg SysManConfig) error {
	path := sysmanConfigPath()
	if path == "" {
		return os.ErrNotExist
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	existing := LoadSysManConfig()

	if cfg.Serman.ServiceDir == "" {
		cfg.Serman.ServiceDir = existing.Serman.ServiceDir
	}
	if cfg.Serman.ServiceDestDir == "" {
		cfg.Serman.ServiceDestDir = existing.Serman.ServiceDestDir
	}
	if cfg.Srcman.DistDir == "" {
		cfg.Srcman.DistDir = existing.Srcman.DistDir
	}
	if cfg.Srcman.SearchEngine == "" {
		cfg.Srcman.SearchEngine = existing.Srcman.SearchEngine
	}
	if cfg.Vmsman.VmDir == "" {
		cfg.Vmsman.VmDir = existing.Vmsman.VmDir
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}
