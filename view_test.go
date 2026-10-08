package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestViewNoCrash(t *testing.T) {
	m := testModel(t, modeInstall, DefaultConfig())
	m.width = 100
	m.height = 40
	m.loading = false

	// Test all modes
	modes := []viewMode{modeDashboard, modeInstall, modeUpdate, modeRemove}
	for _, mode := range modes {
		m.mode = mode
		t.Run("view mode "+string(rune(mode)), func(t *testing.T) {
			view := m.View()
			if view == "" {
				t.Errorf("View() returned empty string for mode %v", mode)
			}
		})
	}

	// Test with confirmation dialog
	m.showConfirmation = true
	m.confirmType = confirmInstall
	m.confirmPackages = []string{"pkg1", "pkg2"}
	t.Run("confirmation dialog", func(t *testing.T) {
		view := m.View()
		if !strings.Contains(view, "Confirm Installation") {
			t.Errorf("View() didn't contain 'Confirm Installation' in confirmation mode")
		}
	})

	// Test with error overlay
	m.showConfirmation = false
	m.showErrorOverlay = true
	m.errorTitle = "Error Title"
	m.errorMessage = "Error message"
	t.Run("error overlay", func(t *testing.T) {
		view := m.View()
		if !strings.Contains(view, "Error Title") {
			t.Errorf("View() didn't contain 'Error Title' in error mode")
		}
	})
}

func TestRepoSummary(t *testing.T) {
	m := testModel(t, modeInstall, DefaultConfig())
	pkgList := []Package{
		{Source: "core", Name: "linux"},
		{Source: "extra", Name: "vim"},
		{Source: "extra", Name: "libpng"},
		{Source: "aur", Name: "google-chrome"},
	}

	summary := m.renderRepoSummary(pkgList)

	// Should contain the counts and repo names
	// renderRepoSummary uses sourceStyle(r).Render(r) which might add ANSI codes
	if !strings.Contains(summary, "1") || !strings.Contains(summary, "core") {
		t.Errorf("Expected '1 core' in summary, got %q", summary)
	}
	if !strings.Contains(summary, "2") || !strings.Contains(summary, "extra") {
		t.Errorf("Expected '2 extra' in summary, got %q", summary)
	}
	if !strings.Contains(summary, "1") || !strings.Contains(summary, "aur") {
		t.Errorf("Expected '1 aur' in summary, got %q", summary)
	}

	// Test empty list
	if m.renderRepoSummary([]Package{}) != "" {
		t.Error("Empty package list should return empty summary")
	}
}

func TestRenderCenteredWrappedText(t *testing.T) {
	// Empty text or zero/negative width should return empty string
	if got := renderCenteredWrappedText("", 50); got != "" {
		t.Errorf("Expected empty string for empty input, got %q", got)
	}
	if got := renderCenteredWrappedText("hello", 0); got != "" {
		t.Errorf("Expected empty string for zero width, got %q", got)
	}
	if got := renderCenteredWrappedText("hello", -10); got != "" {
		t.Errorf("Expected empty string for negative width, got %q", got)
	}

	// Normal text
	text := "First line\nSecond line"
	width := 40
	rendered := renderCenteredWrappedText(text, width)
	lines := strings.Split(rendered, "\n")
	if len(lines) != 2 {
		t.Fatalf("Expected 2 lines, got %d", len(lines))
	}
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if trimmed == "" {
			t.Errorf("Unexpected empty line in rendered output")
		}
		if w := lipgloss.Width(l); w != width {
			t.Errorf("Expected line width %d, got %d", width, w)
		}
	}
}

func TestRenderHelpText(t *testing.T) {
	m := testModel(t, modeInstall, DefaultConfig())
	activeColor := lipgloss.Color("#a6e3a1")

	// Verify all modes generate non-empty help text containing essential actions
	modes := []viewMode{modeDashboard, modeInstall, modeUpdate, modeUpdateSelective, modeRemove, modeSettings}
	for _, mode := range modes {
		m.mode = mode
		help := stripAnsi(m.renderHelpText(activeColor))
		if help == "" {
			t.Errorf("Help text is empty for mode %v", mode)
		}
		for _, required := range []string{"search", "mark", "[d]ash", "[i]nstall", "[u]pdate", "[r]emove", "settings", "[q]uit"} {
			if !strings.Contains(help, required) {
				t.Errorf("Help text in mode %v missing required item %q in %q", mode, required, help)
			}
		}
	}
}


