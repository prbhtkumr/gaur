package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSettingsNavigation(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)

	// Open settings
	m.mode = modeSettings

	// Find Theme index
	themeIdx := -1
	for i, item := range m.settingsItems {
		if item.Label == "Theme" {
			themeIdx = i
			break
		}
	}
	if themeIdx == -1 {
		t.Fatal("Theme setting not found")
	}
	m.settingsIndex = themeIdx

	initialTheme := m.config.UI.Theme

	// Test cycle right (l)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	m = newModel.(*model)

	if m.config.UI.Theme == initialTheme {
		t.Errorf("Expected theme to change after cycling right, but stayed %q", initialTheme)
	}

	// Test cycle left (h) - should go back to initial
	newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	m = newModel.(*model)

	normalize := func(s string) string {
		return strings.ToLower(strings.ReplaceAll(s, " ", "-"))
	}

	if normalize(m.config.UI.Theme) != normalize(initialTheme) {
		t.Errorf("Expected theme to return to %q, but got %q", initialTheme, m.config.UI.Theme)
	}
}

func TestSettingsBounds(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)
	m.mode = modeSettings
	m.settingsIndex = 0

	// Move up at top (should stay at 0)
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	m = newModel.(*model)
	if m.settingsIndex != 0 {
		t.Errorf("Expected index 0 after moving up at top, got %d", m.settingsIndex)
	}

	// Move down to bottom
	for i := 0; i < len(m.settingsItems); i++ {
		newModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		m = newModel.(*model)
	}

	expectedLast := len(m.settingsItems) - 1
	if m.settingsIndex != expectedLast {
		t.Errorf("Expected index %d after moving down multiple times, got %d", expectedLast, m.settingsIndex)
	}
}

func TestSettingsCloseEsc(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)
	m.mode = modeSettings
	m.previousMode = modeInstall

	// Press ESC
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(*model)

	if m.mode != modeInstall {
		t.Errorf("Expected to return to modeInstall after ESC, but in mode %v", m.mode)
	}
}
