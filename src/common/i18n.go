package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type translations map[string]string

type langFile struct {
	Meta struct {
		Code string `yaml:"code"`
		Name string `yaml:"name"`
	} `yaml:"meta"`
	Strings translations `yaml:"strings"`
}

var langs = map[string]translations{}

// T is the translation map.
var T translations
var i18nOnce sync.Once

// GetLangDirs returns the search paths for language files.
// Priority: SYSMAN_LANGDIR env > config lang_dir > default paths.
func GetLangDirs(module string) []string {
	var dirs []string

	// 1. SYSMAN_LANGDIR environment variable (highest priority)
	if langDir := os.Getenv("SYSMAN_LANGDIR"); langDir != "" {
		dirs = append(dirs, filepath.Join(langDir, module))
	}

	// 2. Config file lang_dir (only if not already set via env)
	if langDir := os.Getenv("SYSMAN_LANGDIR"); langDir == "" {
		if cfg := LoadSysManConfig(); cfg.LangDir != "" {
			dirs = append(dirs, filepath.Join(cfg.LangDir, module))
		}
	}

	// 3. Default system paths
	defaultDirs := []string{
		"/usr/local/share/SysMan/lang/" + module,
		"/usr/share/SysMan/lang/" + module,
	}
	if exe, err := os.Executable(); err == nil {
		defaultDirs = append([]string{filepath.Join(filepath.Dir(exe), "lang", module)}, defaultDirs...)
	}

	// 4. Development paths (relative to CWD - various depths)
	// These cover running from project root, src/, src/<module>/, or during tests
	dirs = append(dirs,
		"./lang/"+module,         // from project root
		"./src/lang/"+module,     // from project root
		"../lang/"+module,        // from src/<module>/
		"../../lang/"+module,     // from src/<module>/ (alternate)
		"../src/lang/"+module,    // from src/<module>/
		"../../src/lang/"+module, // from src/<module>/
	)

	dirs = append(dirs, defaultDirs...)
	return dirs
}

func loadLangDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		loadLangFile(filepath.Join(dir, e.Name()))
	}
}

func loadLangFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var lf langFile
	if err := yaml.Unmarshal(data, &lf); err != nil {
		fmt.Fprintf(os.Stderr, "config: error loading %s: %v\n", path, err)
		return
	}
	if lf.Meta.Code == "" {
		return
	}
	langs[strings.ToLower(lf.Meta.Code)] = lf.Strings
}

func detectLang() string {
	if l := os.Getenv("SYSMAN_LANG"); l != "" {
		return strings.ToLower(strings.TrimSpace(l))
	}
	for _, env := range []string{"LANGUAGE", "LANG", "LC_ALL", "LC_MESSAGES"} {
		if l := os.Getenv(env); l != "" {
			l = strings.ToLower(l)
			l = strings.SplitN(l, "_", 2)[0]
			l = strings.SplitN(l, ".", 2)[0]
			if _, ok := langs[l]; ok {
				return l
			}
		}
	}
	return "en"
}

// InitI18n initializes the i18n system.
func InitI18n() {
	i18nOnce.Do(func() {
		for _, dir := range GetLangDirs(".") {
			loadLangDir(dir)
		}
		lang := detectLang()
		if tr, ok := langs[lang]; ok {
			T = tr
			return
		}
		if tr, ok := langs["en"]; ok {
			T = tr
			return
		}
		T = translations{}
	})
}

// InitModuleI18n initializes i18n for a module.
func InitModuleI18n(module string) {
	i18nOnce.Do(func() {
		for _, dir := range GetLangDirs(module) {
			loadLangDir(dir)
		}
		lang := detectLang()
		if tr, ok := langs[lang]; ok {
			T = tr
			return
		}
		if tr, ok := langs["en"]; ok {
			T = tr
			return
		}
		T = translations{}
	})
}

func t(key string) string {
	if T == nil {
		InitI18n()
	}
	if v, ok := T[key]; ok {
		return v
	}
	return key
}

// TFunc returns the translation for a key.
func TFunc(key string) string {
	return t(key)
}
