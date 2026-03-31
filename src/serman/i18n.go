package serman

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// ── Types ────────────────────────────────────────────────────────────

// translations maps translation keys to their localized strings.
type translations map[string]string

// langFile represents the structure of a single language YAML file.
type langFile struct {
	Meta struct {
		Code string `yaml:"code"` // language code (e.g., "en", "cs")
		Name string `yaml:"name"` // display name (e.g., "English", "Čeština")
	} `yaml:"meta"`
	Strings translations `yaml:"strings"` // key-value translation pairs
}

// ── Register ─────────────────────────────────────────────────────────

// langs stores all loaded language translations, indexed by language code.
var langs = map[string]translations{}

// T is the active translation map selected at initialization.
// Exported so standalone main.go can read values for --help output.
var T translations

// langDirs returns directories where to look for *.yaml translation files.
// They are searched in order — first match wins.
func langDirs() []string {
	dirs := []string{
		"/usr/local/share/SysMan/lang/serman",
		"/usr/share/SysMan/lang/serman",
	}
	if exe, err := os.Executable(); err == nil {
		dirs = append([]string{filepath.Join(filepath.Dir(exe), "lang", "serman")}, dirs...)
	}
	dirs = append([]string{"./lang/serman"}, dirs...)
	return dirs
}

// ── Loading ──────────────────────────────────────────────────────────

// loadLangDir loads all *.yaml files from the given directory.
// Silently skips directories that don't exist or can't be read.
func loadLangDir(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		// skip subdirectories and non-YAML files
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		loadLangFile(filepath.Join(dir, e.Name()))
	}
}

// loadLangFile parses a single YAML translation file and registers it in langs.
// Skips files with missing or empty language codes.
func loadLangFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var lf langFile
	if err := yaml.Unmarshal(data, &lf); err != nil {
		fmt.Fprintf(os.Stderr, "svman: error loading %s: %v\n", path, err)
		return
	}
	if lf.Meta.Code == "" {
		return
	}
	langs[strings.ToLower(lf.Meta.Code)] = lf.Strings
}

// ── Init ─────────────────────────────────────────────────────────────

var i18nOnce sync.Once

// InitI18n initializes the translation system (idempotent — safe to call multiple times).
// Loads all language files, detects and selects the appropriate language,
// falling back to English if the detected language is unavailable.
func InitI18n() {
	i18nOnce.Do(func() {
		for _, dir := range langDirs() {
			loadLangDir(dir)
		}
		lang := detectLang()
		if tr, ok := langs[lang]; ok {
			T = tr
			return
		}
		if tr, ok := langs["en"]; ok {
			T = tr // fallback to English
			return
		}
		T = translations{}
	})
}

// detectLang determines the active language by checking:
// 1. SYSMAN_LANG environment variable (highest priority).
// 2. Standard locale variables (LANGUAGE, LANG, LC_ALL, LC_MESSAGES).
// 3. Returns "en" (English) as the default.
func detectLang() string {
	if l := os.Getenv("SYSMAN_LANG"); l != "" {
		return strings.ToLower(strings.TrimSpace(l))
	}
	for _, env := range []string{"LANGUAGE", "LANG", "LC_ALL", "LC_MESSAGES"} {
		if l := os.Getenv(env); l != "" {
			l = strings.ToLower(l)
			l = strings.SplitN(l, "_", 2)[0] // remove region (e.g., cs_CZ → cs)
			l = strings.SplitN(l, ".", 2)[0] // remove encoding (e.g., cs.UTF-8 → cs)
			if _, ok := langs[l]; ok {
				return l
			}
		}
	}
	return "en"
}

// t returns the translated string for the given key.
// If the key is missing from the active translation map, returns the key itself.
// Lazily initializes the translation system on first call.
func t(key string) string {
	if T == nil {
		InitI18n()
	}
	if v, ok := T[key]; ok {
		return v
	}
	return key // fallback: return untranslated key
}
