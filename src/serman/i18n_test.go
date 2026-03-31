package serman

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadLangFile(t *testing.T) {
	langDir := findLangDir("serman")
	if langDir == "" {
		t.Skip("lang directory not found")
	}

	testLangDir(t, langDir)
}

func TestLangFileConsistency(t *testing.T) {
	langDir := findLangDir("serman")
	if langDir == "" {
		t.Skip("lang directory not found")
	}

	testLangConsistency(t, langDir)
}

func findLangDir(modName string) string {
	dirs := []string{
		"../../../src/lang/" + modName,
		"../../lang/" + modName,
		"../lang/" + modName,
	}
	for _, d := range dirs {
		if _, err := os.Stat(d); err == nil {
			return d
		}
	}
	return ""
}

func testLangDir(t *testing.T, langDir string) {
	entries, err := os.ReadDir(langDir)
	if err != nil {
		t.Fatalf("failed to read lang dir: %v", err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		path := filepath.Join(langDir, e.Name())
		t.Run(e.Name(), func(t *testing.T) {
			testLangFile(t, path)
		})
	}
}

func testLangFile(t *testing.T, path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var lf struct {
		Meta struct {
			Code string `yaml:"code"`
			Name string `yaml:"name"`
		} `yaml:"meta"`
		Strings map[string]string `yaml:"strings"`
	}
	if err := yaml.Unmarshal(data, &lf); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}

	if lf.Meta.Code == "" {
		t.Error("language code is missing")
	}

	if lf.Strings == nil {
		t.Fatal("strings map is nil")
	}

	for key, value := range lf.Strings {
		if value == "" {
			t.Errorf("empty translation for key: %s", key)
		}
		if value == key {
			t.Errorf("translation equals key (not translated): %s", key)
		}
	}
}

func testLangConsistency(t *testing.T, langDir string) {
	entries, err := os.ReadDir(langDir)
	if err != nil {
		t.Fatalf("failed to read lang dir: %v", err)
	}

	var enStrings map[string]string
	var files []string

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		files = append(files, e.Name())
	}

	for _, fname := range files {
		if fname == "en.yaml" {
			data, _ := os.ReadFile(filepath.Join(langDir, fname))
			var lf struct {
				Strings map[string]string `yaml:"strings"`
			}
			yaml.Unmarshal(data, &lf)
			enStrings = lf.Strings
			break
		}
	}

	if enStrings == nil {
		t.Skip("en.yaml not found for consistency check")
	}

	for _, fname := range files {
		if fname == "en.yaml" {
			continue
		}
		path := filepath.Join(langDir, fname)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var lf struct {
			Meta struct {
				Code string `yaml:"code"`
			} `yaml:"meta"`
			Strings map[string]string `yaml:"strings"`
		}
		if err := yaml.Unmarshal(data, &lf); err != nil {
			continue
		}

		for key := range enStrings {
			if _, ok := lf.Strings[key]; !ok {
				t.Errorf("[%s] missing key from en.yaml: %s", lf.Meta.Code, key)
			}
		}
	}
}

func TestTooltipKeysMatch(t *testing.T) {
	langDir := findLangDir("serman")
	if langDir == "" {
		t.Skip("lang directory not found")
	}

	testTooltipKeysMatch(t, langDir, "serman")
}

func testTooltipKeysMatch(t *testing.T, langDir, modName string) {
	// Load en.yaml strings
	path := filepath.Join(langDir, "en.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read en.yaml: %v", err)
	}

	var lf struct {
		Strings map[string]string `yaml:"strings"`
	}
	if err := yaml.Unmarshal(data, &lf); err != nil {
		t.Fatalf("failed to parse YAML: %v", err)
	}

	// Get all tooltip keys from Go code
	goKeys := make(map[string]bool)
	goFiles := []string{"gui.go", "tui.go"}
	for _, f := range goFiles {
		fp := filepath.Join("..", modName, f)
		if data, err := os.ReadFile(fp); err == nil {
			// Find all t("tooltip.xxx") calls
			content := string(data)
			prefix := "tooltip." + modName + "."
			for _, line := range strings.Split(content, "\n") {
				if idx := strings.Index(line, `t("`+prefix); idx >= 0 {
					start := idx + len(`t("`) + len(prefix)
					end := strings.Index(line[start:], `")`)
					if end > 0 {
						goKeys[prefix+line[start:start+end]] = true
					}
				}
			}
		}
	}

	// Check all Go keys exist in YAML
	for key := range goKeys {
		if _, ok := lf.Strings[key]; !ok {
			t.Errorf("Go code uses tooltip key '%s' but it's not defined in en.yaml", key)
		}
	}

	// Warn about YAML keys not used in Go
	prefix := "tooltip." + modName + "."
	for key := range lf.Strings {
		if strings.HasPrefix(key, prefix) && !goKeys[key] {
			t.Logf("Warning: en.yaml has tooltip key '%s' but it's not used in Go code", key)
		}
	}
}

// TestAllTooltipsTranslated verifies that ALL tooltip keys in the YAML files
// return actual translated text (not the key itself). This catches the bug where
// tooltips show keys instead of translations due to lang files not being loaded.
func TestAllTooltipsTranslated(tt *testing.T) {
	// Reset i18n to ensure fresh load - this is what should happen in main()
	T = nil
	i18nOnce = sync.Once{}
	InitI18n()

	// Check that translations loaded
	if T == nil || len(langs) == 0 {
		tt.Fatal("Translations not loaded - lang files not found")
	}

	langDir := findLangDir("serman")
	if langDir == "" {
		tt.Skip("lang directory not found")
	}

	// Load en.yaml strings
	path := filepath.Join(langDir, "en.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		tt.Fatalf("failed to read en.yaml: %v", err)
	}

	var lf struct {
		Strings map[string]string `yaml:"strings"`
	}
	if err := yaml.Unmarshal(data, &lf); err != nil {
		tt.Fatalf("failed to parse YAML: %v", err)
	}

	// Get all tooltip keys
	var tooltipKeys []string
	prefix := "tooltip.serman."
	for key := range lf.Strings {
		if strings.HasPrefix(key, prefix) {
			tooltipKeys = append(tooltipKeys, key)
		}
	}

	if len(tooltipKeys) == 0 {
		tt.Fatal("no tooltip keys found in en.yaml")
	}

	// Test the actual t() function - this is the REAL test
	// If lang files aren't loaded, t() returns the key itself
	translate := func(key string) string {
		if T == nil {
			return key
		}
		if v, ok := T[key]; ok {
			return v
		}
		return key
	}

	untranslated := 0
	for _, key := range tooltipKeys {
		result := translate(key)
		if result == key {
			untranslated++
			tt.Logf("UNTRANSLATED: %q returns key itself", key)
		}
	}

	if untranslated > 0 {
		tt.Errorf("%d/%d tooltips are untranslated (returning key instead of text). "+
			"This indicates lang files are not being loaded.", untranslated, len(tooltipKeys))
	} else {
		tt.Logf("All %d tooltips are properly translated", len(tooltipKeys))
	}
}

// TestTooltipCount verifies the expected number of tooltips exist.
func TestTooltipCount(tt *testing.T) {
	langDir := findLangDir("serman")
	if langDir == "" {
		tt.Skip("lang directory not found")
	}

	path := filepath.Join(langDir, "en.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		tt.Fatalf("failed to read en.yaml: %v", err)
	}

	var lf struct {
		Strings map[string]string `yaml:"strings"`
	}
	if err := yaml.Unmarshal(data, &lf); err != nil {
		tt.Fatalf("failed to parse YAML: %v", err)
	}

	// Count tooltips
	prefix := "tooltip.serman."
	count := 0
	for key := range lf.Strings {
		if strings.HasPrefix(key, prefix) {
			count++
		}
	}

	// serman should have at least these tooltips based on gui.go buttons:
	// filter_all, filter_enabled, filter_disabled, enable, disable, reload,
	// start, stop, restart, hup, pause, continue, kill, about
	minExpected := 10
	if count < minExpected {
		tt.Errorf("Expected at least %d tooltips, found %d", minExpected, count)
	} else {
		tt.Logf("Found %d tooltip translations in en.yaml", count)
	}
}
