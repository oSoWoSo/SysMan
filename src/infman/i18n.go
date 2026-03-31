package infman

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"codeberg.org/oSoWoSo/SysMan/src/common"
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

var (
	langs    = map[string]translations{}
	activeT  translations
	i18nOnce sync.Once
)

func langDirs() []string {
	return common.GetLangDirs("infman")
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
		fmt.Fprintf(os.Stderr, "infoman: error loading %s: %v\n", path, err)
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

// InitI18n initializes translations (idempotent).
func InitI18n() {
	i18nOnce.Do(func() {
		for _, dir := range langDirs() {
			loadLangDir(dir)
		}
		lang := detectLang()
		if tr, ok := langs[lang]; ok {
			activeT = tr
			return
		}
		if tr, ok := langs["en"]; ok {
			activeT = tr
			return
		}
		activeT = translations{}
	})
}

func t(key string) string {
	if activeT == nil {
		InitI18n()
	}
	if v, ok := activeT[key]; ok {
		return v
	}
	return key
}
