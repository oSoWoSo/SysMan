package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// ── Types ────────────────────────────────────────────────────────────

// translations maps translation keys to their localized strings.
type translations map[string]string

// langFile represents the structure of a single language YAML file.
// It contains metadata (language code, display name) and string mappings.
type langFile struct {
	Meta struct {
		Code string `yaml:"code"`   // language code (e.g., "en", "cs")
		Name string `yaml:"name"`   // display name (e.g., "English", "Čeština")
	} `yaml:"meta"`
	Strings translations `yaml:"strings"` // key-value translation pairs
}

// ── Register ─────────────────────────────────────────────────────────

// langs stores all loaded language translations, indexed by language code.
var langs = map[string]translations{}

// T is the active translation map selected at initialization.
var T translations

// langDirs are directories where to look for *.yaml translation files.
// They are searched in order — the first file found wins.
var langDirs = []string{
	"./lang",                       // next to binary / in CWD
	"/usr/share/svman/lang",        // system installation
	"/usr/local/share/svman/lang",  // local installation
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
		path := filepath.Join(dir, e.Name())
		loadLangFile(path)
	}
}

// loadLangFile parses a single YAML translation file and registers it in langs.
// Skips files with missing or empty language codes.
// Logs parsing errors to stderr but continues execution.
func loadLangFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var lf langFile
	if err := yaml.Unmarshal(data, &lf); err != nil {
		fmt.Fprintf(os.Stderr, "svman: chyba pri nacitani %s: %v\n", path, err)
		return
	}
	// only register languages with a valid code
	if lf.Meta.Code == "" {
		return
	}
	langs[strings.ToLower(lf.Meta.Code)] = lf.Strings
}

// ── Init ─────────────────────────────────────────────────────────────

// initI18n initializes the translation system:
// 1. Loads all language files from registered directories.
// 2. Detects and selects the appropriate language.
// 3. Falls back to English if the detected language is unavailable.
// 4. Uses an empty map if no languages are loaded.
func initI18n() {
	// Load all language files ─────────────────────────────────────────
	for _, dir := range langDirs {
		loadLangDir(dir)
	}

	// Select language ─────────────────────────────────────────────────
	lang := detectLang()
	if tr, ok := langs[lang]; ok {
		T = tr
		return
	}
	if tr, ok := langs["en"]; ok {
		T = tr // fallback ─────────────────────────────────────────────
		return
	}
	T = translations{}
}

// detectLang determines the active language by checking:
// 1. SVMAN_LANG environment variable (highest priority).
// 2. Standard locale variables (LANGUAGE, LANG, LC_ALL, LC_MESSAGES).
// 3. Returns "en" (English) as the default if no match is found.
// Extracts the language code from locale strings (e.g., "cs_CZ.UTF-8" → "cs").
func detectLang() string {
	// check explicit SVMAN_LANG override
	if l := os.Getenv("SVMAN_LANG"); l != "" {
		return strings.ToLower(strings.TrimSpace(l))
	}
	// check standard locale environment variables
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
		initI18n()
	}
	if v, ok := T[key]; ok {
		return v
	}
	return key // fallback: return untranslated key
}
