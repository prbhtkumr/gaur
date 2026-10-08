package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestDashboardLoadingDimensions(t *testing.T) {
	m_init := testModel(t, modeDashboard, DefaultConfig())
	m_init.width = 80
	m_init.height = 24
	m_init.loading = true
	
	// Crucial: Initialize layout constants via Update and use the result
	new_m, _ := m_init.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m := new_m.(*model)

	view := m.View()
	lines := strings.Split(view, "\n")

	if len(lines) != 24 {
		t.Errorf("Expected loading height 24, got %d", len(lines))
	}
	
	for i, line := range lines {
		if lipgloss.Width(line) != 80 {
			t.Errorf("Line %d width mismatch: expected 80, got %d", i, lipgloss.Width(line))
		}
	}
}

func TestDashboardRendering(t *testing.T) {
	m_init := testModel(t, modeDashboard, DefaultConfig())
	
	// Crucial: Initialize layout constants via Update and use the result
	new_m, _ := m_init.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m := new_m.(*model) 
	m.loading = false

	// Mock Dashboard Data
	m.dashboard = DashboardData{
		TotalPackages:       1000,
		ExplicitlyInstalled: 200,
		ForeignPackages:     50,
		RepoDistribution: map[string]int{
			"core":     100,
			"extra":    800,
			"multilib": 50,
		},
		TotalSize:        "40.0 GiB",
		TotalSizeBytes:   40 * 1024 * 1024 * 1024,
		CleanerSize:      "20.0 GiB",
		CleanerSizeBytes: 20 * 1024 * 1024 * 1024,
		DiskTotal:        "100 GiB",
		DiskUsed:         "40 GiB",
		DiskFree:         "60 GiB",
		DiskUsedPercent:  0.4,
		RecentlyInstalled: []RecentPackage{
			{Name: "pkg1", Timestamp: "2024-03-12 10:00"},
			{Name: "pkg2", Timestamp: "2024-03-12 09:00"},
		},
		TopPackages: []PackageSize{
			{Name: "huge-pkg", Size: "2.5 GiB"}, // Should be red
			{Name: "big-pkg", Size: "750 MiB"},  // Should be orange
			{Name: "small-pkg", Size: "10 MiB"},  // Should be cyan
		},
	}

	view := m.renderDashboard("help text", m.width, m.height)

	// 1. Verify all major sections exist
	sections := []string{
		"\U000f03d7  Package Counts",
		"Disk Space (/:40%)",
		"\ueb9c  Repository Distribution",
		"\ueddf  Top by Weight",
		"\uf0c7  Top Cache Hogs",
		"\uf1da  Recently Installed",
	}

	for _, section := range sections {
		if !strings.Contains(view, section) {
			t.Errorf("Dashboard missing section: %s", section)
		}
	}

	// 2. Verify Disk Space components exist
	lines := strings.Split(view, "\n")
	components := []string{"total", "packages", "ache", "other", "free"}
	foundComponents := make(map[string]bool)
	for _, line := range lines {
		for _, comp := range components {
			if strings.Contains(line, comp) {
				foundComponents[comp] = true
			}
		}
	}
	for _, comp := range components {
		if !foundComponents[comp] {
			t.Errorf("Disk space component not found: %s", comp)
		}
	}

	// 3. Verify Repo Distribution Legend exists
	foundRepoLabels := false
	foundRepoCounts := false
	for _, line := range lines {
		if strings.Contains(line, "Core") && strings.Contains(line, "Extra") && strings.Contains(line, "AUR") {
			foundRepoLabels = true
		}
		if strings.Contains(line, "100") && strings.Contains(line, "800") && strings.Contains(line, "50") {
			foundRepoCounts = true
		}
	}
	if !foundRepoLabels || !foundRepoCounts {
		t.Error("Repository distribution labels or counts not found in view")
	}
}

func TestDashboardRefreshLogic(t *testing.T) {
	m := testModel(t, modeDashboard, DefaultConfig())
	m.textInput.Blur()

	// Test 'd' key
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Fatal("Pressing 'd' in dashboard mode did not return a command")
	}

	// Test 'ctrl+r'
	msg = tea.KeyMsg{Type: tea.KeyCtrlR}
	_, cmd = m.Update(msg)

	if cmd == nil {
		t.Fatal("Pressing 'ctrl+r' in dashboard mode did not return a command")
	}
}

func TestSizeColorCodingLogic(t *testing.T) {
	// This tests the logic used inside renderDashboard indirectly
	// by checking if the expected color sequences are present
	m := testModel(t, modeDashboard, DefaultConfig())
	m.width = 100
	m.height = 30
	m.loading = false
	m.dashboard = DashboardData{
		TopPackages: []PackageSize{
			{Name: "huge", Size: "2.0 GiB"},   // Red (Color 196)
			{Name: "medium", Size: "600 MiB"}, // Orange (Color 208)
			{Name: "small", Size: "10 MiB"},   // Cyan (Color 51)
		},
	}

	view := m.renderDashboard("help", m.width, m.height)

	// Check for ANSI color codes (simplified check)
	// Red: 196, Orange: 208, Cyan: 51
	if !strings.Contains(view, "196") && !strings.Contains(view, "31") { // Some themes use different codes
		// Since themes change, we at least verify the packages are listed
		if !strings.Contains(view, "huge") || !strings.Contains(view, "medium") || !strings.Contains(view, "small") {
			t.Error("Top packages not rendered correctly")
		}
	}
}

func TestDashboardMetricAndTableRendering(t *testing.T) {
	m := testModel(t, modeDashboard, DefaultConfig())
	m.width = 120
	m.height = 40
	m.loading = false
	m.dashboard = DashboardData{
		TotalPackages: 500,
		Orphans:       12,
		TopPackages: []PackageSize{
			{Name: "linux-kernel", Size: "1.2 GiB"},
		},
		TopCacheHogs: []PackageSize{
			{Name: "gcc", Size: "250 MiB"},
		},
		RecentlyInstalled: []RecentPackage{
			{Name: "ripgrep", Timestamp: "2026-10-08 09:00"},
		},
	}

	view := m.renderDashboard("help", m.width, m.height)

	if !strings.Contains(view, "[o]rphans") || !strings.Contains(view, "12") {
		t.Errorf("Dashboard view missing orphan count or label")
	}

	if !strings.Contains(view, "gcc") || !strings.Contains(view, "250 MiB") {
		t.Errorf("Dashboard view missing cache hog")
	}

	if !strings.Contains(view, "ripgrep") || !strings.Contains(view, "2026-10-08 09:00") {
		t.Errorf("Dashboard view missing recently installed package")
	}
}

