package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestFilenameToDisplayName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"catppuccin_mocha.toml", "Catppuccin Mocha"},
		{"catppuccin-mocha.toml", "Catppuccin Mocha"},
		{"dracula.toml", "Dracula"},
		{"tokyonight_night.toml", "Tokyonight Night"},
		{"solarized_dark.toml", "Solarized Dark"},
		{"a_b_c.toml", "A B C"},
		{"single.toml", "Single"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := filenameToDisplayName(tt.input)
			if result != tt.expected {
				t.Errorf("filenameToDisplayName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeThemeName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Catppuccin Mocha", "catppuccin-mocha"},
		{"catppuccin mocha", "catppuccin-mocha"},
		{"catppuccin_mocha", "catppuccin-mocha"},
		{"CATPPUCCIN MOCHA", "catppuccin-mocha"},
		{"  Catppuccin Mocha  ", "catppuccin-mocha"},
		{"Dracula", "dracula"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeThemeName(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeThemeName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestThemeLoaderGetTheme(t *testing.T) {
	tl := newTestThemeLoader()

	theme, ok := tl.GetTheme("Catppuccin Mocha")
	if !ok {
		t.Fatal("Expected to find Catppuccin Mocha theme")
	}

	if theme.Name != "Catppuccin Mocha" {
		t.Errorf("Theme name = %q, want %q", theme.Name, "Catppuccin Mocha")
	}

	if string(theme.BorderColor) == "" {
		t.Error("BorderColor should not be empty")
	}
}

func TestThemeLoaderGetThemeByConfigName(t *testing.T) {
	tl := newTestThemeLoader()

	tests := []struct {
		name   string
		input  string
		wantOK bool
	}{
		{"exact match", "Catppuccin Mocha", true},
		{"lowercase", "catppuccin mocha", true},
		{"hyphens", "catppuccin-mocha", true},
		{"underscore", "catppuccin_mocha", true},
		{"unknown", "Nonexistent Theme", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := tl.GetThemeByConfigName(tt.input)
			if ok != tt.wantOK {
				t.Errorf("GetThemeByConfigName(%q) = %v, want %v", tt.input, ok, tt.wantOK)
			}
		})
	}
}

func TestThemeLoaderListThemes(t *testing.T) {
	tl := newTestThemeLoader()
	themes := tl.ListThemes()

	if len(themes) == 0 {
		t.Fatal("ListThemes returned empty list")
	}

	for i := 1; i < len(themes); i++ {
		if themes[i] < themes[i-1] {
			t.Errorf("Themes not sorted: %q comes before %q", themes[i-1], themes[i])
		}
	}
}

func TestThemeLoaderExportDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	exportDir := filepath.Join(tmpDir, "themes")

	tl := newTestThemeLoader()
	err := tl.ExportDefaults(exportDir)
	if err != nil {
		t.Fatalf("ExportDefaults failed: %v", err)
	}

	entries, err := os.ReadDir(exportDir)
	if err != nil {
		t.Fatalf("Failed to read export directory: %v", err)
	}

	if len(entries) == 0 {
		t.Error("Export directory is empty")
	}

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".toml" {
			t.Errorf("Unexpected file in export directory: %s", entry.Name())
		}
	}
}

func TestParseThemeValid(t *testing.T) {
	validToml := `
border = "#6c7086"
selected = "#cba6f7"
text = "#cdd6f4"
subtle = "#6c7086"
title = "#f9e2af"
scrollbar_track = "#181825"
scrollbar_thumb = "#6c7086"
selection_bg = "#313244"
dim_text = "#6c7086"
install = "#89b4fa"
dashboard = "#f5c2e7"
remove = "#f38ba8"
update = "#a6e3a1"
cache = "#cba6f7"
core = "#a6e3a1"
extra = "#89b4fa"
multilib = "#fab387"
aur = "#cba6f7"
success = "#a6e3a1"
warning = "#f9e2af"
error = "#f38ba8"
highlight = "#f9e2af"
dashboard_label = "#cdd6f4"
dashboard_value = "#89dceb"
dashboard_warning = "#f38ba8"
dashboard_desc = "#a6adc8"
dialog_border = "#cba6f7"
confirm_install = "#89b4fa"
confirm_remove = "#fab387"
confirm_clean = "#a6e3a1"
confirm_nuke = "#f38ba8"
confirm_selective = "#cba6f7"
button_bg = "#45475a"
button_fg = "#cdd6f4"
button_danger_bg = "#f38ba8"
progress_track = "#313244"
accent = "#89b4fa"
spinner = "#f5c2e7"
`
	tl := &ThemeLoader{themes: make(map[string]Theme)}
	theme, err := tl.parseTheme([]byte(validToml), "test.toml")
	if err != nil {
		t.Fatalf("parseTheme failed: %v", err)
	}

	if string(theme.BorderColor) != "#6c7086" {
		t.Errorf("BorderColor = %q, want %q", theme.BorderColor, "#6c7086")
	}
}

func TestParseThemeEmptyColors(t *testing.T) {
	emptyToml := `
border = ""
selected = ""
text = ""
`
	tl := &ThemeLoader{themes: make(map[string]Theme)}
	theme, err := tl.parseTheme([]byte(emptyToml), "test.toml")
	if err != nil {
		t.Fatalf("parseTheme failed: %v", err)
	}

	if string(theme.BorderColor) == "" {
		t.Error("Empty color should get default value from applyDefaults")
	}
}

func TestApplyDefaults(t *testing.T) {
	theme := Theme{
		BorderColor: lipgloss.Color(""),
	}

	result := applyDefaults(theme, "test.toml")

	if string(result.BorderColor) == "" {
		t.Error("applyDefaults should set BorderColor to default")
	}
}

func TestSanitizeColor(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"#6c7086", "#6c7086"},
		{"  #6c7086  ", "#6c7086"},
		{"", "#ffffff"},
		{"   ", "#ffffff"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeColor(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeColor(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSetTheme(t *testing.T) {
	tl := newTestThemeLoader()
	theme, _ := tl.GetTheme("Catppuccin Mocha")

	setTheme(theme)

	if currentTheme.Name != "Catppuccin Mocha" {
		t.Errorf("currentTheme.Name = %q, want %q", currentTheme.Name, "Catppuccin Mocha")
	}

	if string(colorRed) == "" {
		t.Error("colorRed should be set after setTheme")
	}
}

func TestUpdateColorVariables(t *testing.T) {
	theme := Theme{
		DimText:     lipgloss.Color("#6c7086"),
		SubtleColor: lipgloss.Color("#737994"),
		TextColor:   lipgloss.Color("#cdd6f4"),
		ErrorColor:  lipgloss.Color("#f38ba8"),
	}

	currentTheme = theme
	updateColorVariables()

	if string(colorDimGray) != "#6c7086" {
		t.Errorf("colorDimGray = %q, want %q", colorDimGray, "#6c7086")
	}
	if string(colorRed) != "#f38ba8" {
		t.Errorf("colorRed = %q, want %q", colorRed, "#f38ba8")
	}
}

func TestGetFallbackDefaults(t *testing.T) {
	defaults := getFallbackDefaults("test.toml")

	if string(defaults.BorderColor) == "" {
		t.Error("getFallbackDefaults should return non-empty BorderColor")
	}
	if string(defaults.SelectedColor) == "" {
		t.Error("getFallbackDefaults should return non-empty SelectedColor")
	}
	if string(defaults.ButtonBg) == "" {
		t.Error("getFallbackDefaults should return non-empty ButtonBg")
	}
	if string(defaults.ButtonFg) == "" {
		t.Error("getFallbackDefaults should return non-empty ButtonFg")
	}
	if string(defaults.ButtonDangerBg) == "" {
		t.Error("getFallbackDefaults should return non-empty ButtonDangerBg")
	}
	if string(defaults.ProgressTrack) == "" {
		t.Error("getFallbackDefaults should return non-empty ProgressTrack")
	}
	if string(defaults.AccentColor) == "" {
		t.Error("getFallbackDefaults should return non-empty AccentColor")
	}
	if string(defaults.SpinnerColor) == "" {
		t.Error("getFallbackDefaults should return non-empty SpinnerColor")
	}
}

func TestNewThemeColors(t *testing.T) {
	tl := newTestThemeLoader()
	theme, ok := tl.GetTheme("Catppuccin Mocha")
	if !ok {
		t.Fatal("Expected to find Catppuccin Mocha theme")
	}

	// Verify new color fields are populated
	if string(theme.ButtonBg) == "" || string(theme.ButtonBg) == "#ffffff" {
		t.Error("ButtonBg should be populated from theme TOML")
	}
	if string(theme.ButtonFg) == "" || string(theme.ButtonFg) == "#ffffff" {
		t.Error("ButtonFg should be populated from theme TOML")
	}
	if string(theme.ButtonDangerBg) == "" || string(theme.ButtonDangerBg) == "#ffffff" {
		t.Error("ButtonDangerBg should be populated from theme TOML")
	}
	if string(theme.ProgressTrack) == "" || string(theme.ProgressTrack) == "#ffffff" {
		t.Error("ProgressTrack should be populated from theme TOML")
	}
	if string(theme.AccentColor) == "" || string(theme.AccentColor) == "#ffffff" {
		t.Error("AccentColor should be populated from theme TOML")
	}
	if string(theme.SpinnerColor) == "" || string(theme.SpinnerColor) == "#ffffff" {
		t.Error("SpinnerColor should be populated from theme TOML")
	}
}

func TestApplyDefaultsPreservesExplicitWhite(t *testing.T) {
	// CS-06: Explicit #ffffff must not be overwritten by fallback defaults
	theme := Theme{
		BorderColor:   lipgloss.Color("#ffffff"),
		SelectedColor: lipgloss.Color("#ffffff"),
	}

	result := applyDefaults(theme, "test.toml")

	if string(result.BorderColor) != "#ffffff" {
		t.Errorf("BorderColor #ffffff was incorrectly overwritten with %q", result.BorderColor)
	}
	if string(result.SelectedColor) != "#ffffff" {
		t.Errorf("SelectedColor #ffffff was incorrectly overwritten with %q", result.SelectedColor)
	}
}

func TestModelThemeLifecycle(t *testing.T) {
	var nilModel *model
	if nilModel.getTheme().BorderColor == "" {
		t.Error("nilModel.getTheme() should return currentTheme with non-empty border color")
	}

	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)
	if m.getTheme().Name == "" {
		t.Error("m.getTheme() should return initialized theme")
	}

	customTheme := Theme{
		Name:        "Custom White",
		BorderColor: lipgloss.Color("#ffffff"),
	}
	m.theme = customTheme
	if m.getTheme().Name != "Custom White" || string(m.getTheme().BorderColor) != "#ffffff" {
		t.Errorf("m.getTheme() failed to return model-level custom theme")
	}
}
