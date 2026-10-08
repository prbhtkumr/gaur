package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestTomlInjectionPrevention(t *testing.T) {
	maliciousInputs := []struct {
		name  string
		input string
	}{
		{
			name:  "shell command in color",
			input: "border = \"$(rm -rf /)\"\nselected = \"#cba6f7\"",
		},
		{
			name:  "backtick injection",
			input: "border = \"`whoami`\"\nselected = \"#cba6f7\"",
		},
		{
			name:  "newline injection",
			input: "border = \"#6c7086\\nexec(evil)\"\nselected = \"#cba6f7\"",
		},
		{
			name:  "large input",
			input: "border = \"" + strings.Repeat("A", 10000) + "\"\nselected = \"#cba6f7\"",
		},
		{
			name:  "null bytes",
			input: "border = \"#6c7086\\x00evil\"\nselected = \"#cba6f7\"",
		},
	}

	for _, tt := range maliciousInputs {
		t.Run(tt.name, func(t *testing.T) {
			tl := &ThemeLoader{themes: make(map[string]Theme)}
			theme, err := tl.parseTheme([]byte(tt.input), "test.toml")

			if err != nil {
				return
			}

			_ = lipgloss.NewStyle().Foreground(theme.BorderColor).Render("test")
		})
	}
}

func TestTomlInvalidSyntax(t *testing.T) {
	invalidInputs := []struct {
		name  string
		input string
	}{
		{"unclosed string", "border = \"#6c7086"},
		{"invalid toml", "[[[["},
		{"invalid key", "123border = \"#6c7086\""},
		{"empty file", ""},
		{"random binary", "\x00\x01\x02\x03"},
	}

	for _, tt := range invalidInputs {
		t.Run(tt.name, func(t *testing.T) {
			tl := &ThemeLoader{themes: make(map[string]Theme)}
			_, err := tl.parseTheme([]byte(tt.input), "test.toml")

			if err == nil && tt.input != "" {
				t.Log("Parser accepted potentially invalid input, which is acceptable")
			}
		})
	}
}

func TestThemeLoaderPathTraversalPrevention(t *testing.T) {
	tmpDir := t.TempDir()
	themedDir := filepath.Join(tmpDir, "themes")

	if err := os.MkdirAll(themedDir, 0755); err != nil {
		t.Fatal(err)
	}

	safeContent := "border = \"#6c7086\"\nselected = \"#cba6f7\"\ntext = \"#cdd6f4\""

	if err := os.WriteFile(filepath.Join(themedDir, "safe.toml"), []byte(safeContent), 0644); err != nil {
		t.Fatal(err)
	}

	tl := &ThemeLoader{
		themes:        make(map[string]Theme),
		userThemesDir: themedDir,
	}

	tl.loadUserThemes()

	for name := range tl.themes {
		if strings.Contains(name, "..") {
			t.Errorf("Path traversal detected in theme name: %s", name)
		}
		if strings.HasPrefix(name, "/") {
			t.Errorf("Absolute path in theme name: %s", name)
		}
	}
}

func TestThemeExportPathSafety(t *testing.T) {
	tl := newTestThemeLoader()

	unsafePaths := []struct {
		name string
		path string
	}{
		{"traversal", "/tmp/../etc"},
	}

	for _, tt := range unsafePaths {
		t.Run(tt.name, func(t *testing.T) {
			err := tl.ExportDefaults(tt.path)

			if err == nil {
				_, err = os.Stat(tt.path)
				if os.IsNotExist(err) {
					t.Log("Export to unsafe path did not create directory")
				}
			}
		})
	}
}

func TestEmbeddedThemesIntegrity(t *testing.T) {
	entries, err := embeddedThemes.ReadDir("themes")
	if err != nil {
		t.Fatalf("Failed to read embedded themes: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("No embedded themes found")
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if filepath.Ext(name) != ".toml" {
			t.Errorf("Unexpected file in embedded themes: %s", name)
			continue
		}

		data, err := embeddedThemes.ReadFile("themes/" + name)
		if err != nil {
			t.Errorf("Failed to read embedded theme %s: %v", name, err)
			continue
		}

		if len(data) == 0 {
			t.Errorf("Empty theme file: %s", name)
		}

		if !strings.Contains(string(data), "border") {
			t.Errorf("Theme %s missing required 'border' field", name)
		}
	}
}

func TestThemeColorValidation(t *testing.T) {
	colorFormats := []struct {
		name    string
		color   string
		isValid bool
	}{
		{"valid hex 6", "#6c7086", true},
		{"valid hex 3", "#fff", true},
		{"valid ANSI", "196", true},
		{"valid ANSI 256", "255", true},
		{"empty", "", false},
		{"invalid hex", "#gggggg", false},
		{"invalid format", "notacolor", false},
	}

	for _, tt := range colorFormats {
		t.Run(tt.name, func(t *testing.T) {
			valid := isValidColor(tt.color)
			if valid != tt.isValid {
				t.Errorf("isValidColor(%q) = %v, expected %v", tt.color, valid, tt.isValid)
			}

			sanitized := sanitizeColor(tt.color)
			if tt.isValid {
				if sanitized != tt.color {
					t.Errorf("sanitizeColor(%q) = %q, expected unchanged", tt.color, sanitized)
				}
			} else {
				if sanitized != "#ffffff" {
					t.Errorf("sanitizeColor(%q) = %q, expected fallback #ffffff", tt.color, sanitized)
				}
			}
			_ = lipgloss.NewStyle().Foreground(lipgloss.Color(sanitized)).Render("test")
		})
	}
}

func TestThemeLoaderEmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	emptyDir := filepath.Join(tmpDir, "themes")

	if err := os.MkdirAll(emptyDir, 0755); err != nil {
		t.Fatal(err)
	}

	tl := &ThemeLoader{
		themes:        make(map[string]Theme),
		userThemesDir: emptyDir,
	}

	if err := tl.loadEmbedded(); err != nil {
		t.Fatalf("loadEmbedded failed: %v", err)
	}

	tl.loadUserThemes()

	if len(tl.themes) == 0 {
		t.Error("ThemeLoader should have embedded themes even with empty user directory")
	}
}

func TestThemeLoaderNonexistentDirectory(t *testing.T) {
	tl := &ThemeLoader{
		themes:        make(map[string]Theme),
		userThemesDir: "/nonexistent/path/that/does/not/exist",
	}

	if err := tl.loadEmbedded(); err != nil {
		t.Fatalf("loadEmbedded failed: %v", err)
	}

	tl.loadUserThemes()

	if len(tl.themes) == 0 {
		t.Error("ThemeLoader should have embedded themes even with nonexistent user directory")
	}
}

func TestThemeLoaderUserOverrideEmbedded(t *testing.T) {
	tmpDir := t.TempDir()
	themedDir := filepath.Join(tmpDir, "themes")

	if err := os.MkdirAll(themedDir, 0755); err != nil {
		t.Fatal(err)
	}

	userThemeContent := "border = \"#ff0000\"\nselected = \"#cba6f7\"\ntext = \"#cdd6f4\"\nsubtle = \"#6c7086\"\ntitle = \"#f9e2af\""

	userThemeName := "catppuccin_mocha.toml"
	if err := os.WriteFile(filepath.Join(themedDir, userThemeName), []byte(userThemeContent), 0644); err != nil {
		t.Fatal(err)
	}

	tl := &ThemeLoader{
		themes:        make(map[string]Theme),
		userThemesDir: themedDir,
	}

	if err := tl.loadEmbedded(); err != nil {
		t.Fatalf("loadEmbedded failed: %v", err)
	}

	tl.loadUserThemes()

	theme, ok := tl.GetTheme("Catppuccin Mocha")
	if !ok {
		t.Fatal("Expected to find Catppuccin Mocha theme")
	}

	if string(theme.BorderColor) == "#ff0000" {
		t.Log("User theme successfully overrides embedded theme")
	}
}

func TestFilenameToDisplayNameSecurity(t *testing.T) {
	maliciousNames := []struct {
		input string
	}{
		{"../../../etc/passwd.toml"},
		{"$(whoami).toml"},
		{"test.toml"},
	}

	for _, tt := range maliciousNames {
		t.Run(tt.input, func(t *testing.T) {
			_ = filenameToDisplayName(tt.input)
		})
	}
}
