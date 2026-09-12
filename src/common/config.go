package common

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// SysManConfig is the main SysMan configuration.
type SysManConfig struct {
	Serman  SermanConfig `yaml:"serman,omitempty"`
	Pkgman  PkgmanConfig `yaml:"pkgman,omitempty"`
	Srcman  Config       `yaml:"srcman,omitempty"`
	Vmsman  VmsmanConfig `yaml:"vmsman,omitempty"`
	Ugsman  UgsmanConfig `yaml:"ugsman,omitempty"`
	Infman  InfmanConfig `yaml:"infman,omitempty"`
	LangDir string       `yaml:"lang_dir,omitempty"`
}

// SermanConfig is the services configuration.
type SermanConfig struct {
	ServiceDir         string `yaml:"service_dir,omitempty"`
	ServiceDestDir     string `yaml:"service_dest_dir,omitempty"`
	UserServiceDir     string `yaml:"user_service_dir,omitempty"`
	UserServiceDestDir string `yaml:"user_service_dest_dir,omitempty"`
	DefaultScope       string `yaml:"default_scope,omitempty"`
}

// PkgmanConfig is the packages configuration.
type PkgmanConfig struct {
	AppImageDir string   `yaml:"appimage_dir,omitempty"` // "" = auto (from appman/am config)
	Repos       []string `yaml:"repos,omitempty"`        // user repo pool (config store only)
	ReposFile   string   `yaml:"repos_file,omitempty"`   // managed file in /etc/xbps.d (default "sysman-repos.conf")
}

// Config is the templates configuration.
type Config struct {
	DistDir      string `yaml:"dist_dir,omitempty"`
	SearchEngine string `yaml:"search_engine,omitempty"`
	ForkURL      string `yaml:"fork_url,omitempty"`
}

// VmsmanConfig is the VM configuration.
type VmsmanConfig struct {
	VMDir string `yaml:"vm_dir,omitempty"`
}

// UgsmanConfig is the users & groups configuration.
type UgsmanConfig struct {
}

// InfmanConfig is the system info configuration.
type InfmanConfig struct {
}

func sysmanConfigPath() string {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(cfg, "sysman", "sysman.conf")
}

// LoadSysManConfig loads the SysMan configuration.
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

	if _, legacy := raw["service_dir"]; legacy {
		if serviceDir, ok := raw["service_dir"].(string); ok {
			c.Serman.ServiceDir = serviceDir
		}
		if serviceDestDir, ok := raw["service_dest_dir"].(string); ok {
			c.Serman.ServiceDestDir = serviceDestDir
		}
		if userServiceDir, ok := raw["user_service_dir"].(string); ok {
			c.Serman.UserServiceDir = userServiceDir
		}
		if userServiceDestDir, ok := raw["user_service_dest_dir"].(string); ok {
			c.Serman.UserServiceDestDir = userServiceDestDir
		}
		if srcmanDistDir, ok := raw["srcman_dist_dir"].(string); ok {
			c.Srcman.DistDir = srcmanDistDir
		}
		if srcmanSearchEngine, ok := raw["srcman_search_engine"].(string); ok {
			c.Srcman.SearchEngine = srcmanSearchEngine
		}
		if vmsmanVMDir, ok := raw["vmsman_vm_dir"].(string); ok {
			c.Vmsman.VMDir = vmsmanVMDir
		}
		return c
	}

	_ = yaml.Unmarshal(data, &c)
	return c
}

// SaveSysManConfig saves the SysMan configuration.
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
	if cfg.Serman.UserServiceDir == "" {
		cfg.Serman.UserServiceDir = existing.Serman.UserServiceDir
	}
	if cfg.Serman.UserServiceDestDir == "" {
		cfg.Serman.UserServiceDestDir = existing.Serman.UserServiceDestDir
	}
	if cfg.Srcman.DistDir == "" {
		cfg.Srcman.DistDir = existing.Srcman.DistDir
	}
	if cfg.Srcman.SearchEngine == "" {
		cfg.Srcman.SearchEngine = existing.Srcman.SearchEngine
	}
	if cfg.Srcman.ForkURL == "" {
		cfg.Srcman.ForkURL = existing.Srcman.ForkURL
	}
	if cfg.Vmsman.VMDir == "" {
		cfg.Vmsman.VMDir = existing.Vmsman.VMDir
	}
	if cfg.LangDir == "" {
		cfg.LangDir = existing.LangDir
	}
	if cfg.Pkgman.AppImageDir == "" {
		cfg.Pkgman.AppImageDir = existing.Pkgman.AppImageDir
	}
	if cfg.Pkgman.ReposFile == "" {
		cfg.Pkgman.ReposFile = existing.Pkgman.ReposFile
	}
	if len(cfg.Pkgman.Repos) == 0 {
		cfg.Pkgman.Repos = existing.Pkgman.Repos
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}
