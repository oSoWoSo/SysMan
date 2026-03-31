package ugsman

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestLoadLangFile(t *testing.T) {
	langDir := findLangDir("ugsman")
	if langDir == "" {
		t.Skip("lang directory not found")
	}

	testLangDir(t, langDir)
}

func TestLangFileConsistency(t *testing.T) {
	langDir := findLangDir("ugsman")
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
