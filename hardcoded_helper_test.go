package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestNoHardcodedAURHelpers(t *testing.T) {
	// Files to skip
	skipFiles := map[string]bool{
		"config.go":           true, // Default values are allowed
		"config_test.go":      true,
		"commands_test.go":    true,
		"model_test.go":       true,
		"integration_test.go": true,
		"mock_runner_test.go": true,
		"quit_test.go":        true,
		"dashboard_test.go":   true,
		"cache_logic_test.go": true,
		"types.go":            true, // Struct definitions
	}

	// Patterns that look like hardcoded helper usage in commands
	// Searching for "paru" or "yay" inside function calls like runner.Run("paru", ...)
	// or labels like "paru:"
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`runner\.Run\s*\(\s*"paru"`),
		regexp.MustCompile(`runner\.Run\s*\(\s*"yay"`),
		regexp.MustCompile(`runner\.Interactive\s*\(\s*[^,]+,\s*"paru"`),
		regexp.MustCompile(`runner\.Interactive\s*\(\s*[^,]+,\s*"yay"`),
		regexp.MustCompile(`"paru:"`),
		regexp.MustCompile(`"yay:"`),
	}

	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && path != "." {
			return filepath.SkipDir
		}
		if !strings.HasSuffix(path, ".go") || skipFiles[path] || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			for _, pattern := range patterns {
				if pattern.MatchString(line) {
					t.Errorf("%s:%d: found hardcoded AUR helper: %s", path, i+1, strings.TrimSpace(line))
				}
			}
		}

		return nil
	})

	if err != nil {
		t.Fatal(err)
	}
}
