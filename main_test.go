package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestIsValidPackageName tests the package name validation function
func TestIsValidPackageName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid simple", "vim", true},
		{"valid with hyphen", "my-package", true},
		{"valid with underscore", "my_package", true},
		{"valid with plus", "gcc11", true}, // doesn't contain plus but still valid
		{"valid with dot", "my.package", true},
		{"valid with at", "host@example", true},
		{"valid mixed", "my-package_123.test", true},
		{"empty string", "", false},
		{"space", "my package", false},
		{"special char", "my$package", false},
		{"bracket", "my[package]", false},
		{"backtick", "my`package`", false},
		{"newline", "my\npackage", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidPackageName(tt.input)
			if result != tt.expected {
				t.Errorf("isValidPackageName(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestSanitizePackageNames tests the package name sanitization
func TestSanitizePackageNames(t *testing.T) {
	tests := []struct {
		name         string
		input        []string
		wantValid    []string
		wantAllValid bool
	}{
		{
			name:         "all valid",
			input:        []string{"vim", "neovim", "htop"},
			wantValid:    []string{"vim", "neovim", "htop"},
			wantAllValid: true,
		},
		{
			name:         "some invalid",
			input:        []string{"vim", "bad name", "neovim"},
			wantValid:    []string{"vim", "neovim"},
			wantAllValid: false,
		},
		{
			name:         "all invalid",
			input:        []string{"", "bad name", "test test"},
			wantValid:    nil, // no valid packages returns nil slice
			wantAllValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, allValid := sanitizePackageNames(tt.input)
			if !reflect.DeepEqual(valid, tt.wantValid) {
				t.Errorf("sanitizePackageNames(%v) returned %v, want %v", tt.input, valid, tt.wantValid)
			}
			if allValid != tt.wantAllValid {
				t.Errorf("sanitizePackageNames(%v) allValid = %v, want %v", tt.input, allValid, tt.wantAllValid)
			}
		})
	}
}

// TestParseRepoFilter tests the repository filter parsing
func TestParseRepoFilter(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantFilters map[string]bool
		wantQuery   string
	}{
		{
			name:        "no filter",
			input:       "vim",
			wantFilters: nil,
			wantQuery:   "vim",
		},
		{
			name:        "single repo filter",
			input:       "a:vim",
			wantFilters: map[string]bool{"aur": true},
			wantQuery:   "vim",
		},
		{
			name:        "multiple repo filters",
			input:       "ae:firefox",
			wantFilters: map[string]bool{"aur": true, "extra": true},
			wantQuery:   "firefox",
		},
		{
			name:        "all repos filter",
			input:       "acem:package",
			wantFilters: map[string]bool{"aur": true, "core": true, "extra": true, "multilib": true},
			wantQuery:   "package",
		},
		{
			name:        "filter with spaces in query",
			input:       "a:  my package  ",
			wantFilters: map[string]bool{"aur": true},
			wantQuery:   "my package",
		},
		{
			name:        "filter with no query",
			input:       "a:",
			wantFilters: map[string]bool{"aur": true},
			wantQuery:   "",
		},
		{
			name:        "invalid filter char ignored",
			input:       "z:vim",
			wantFilters: nil,
			wantQuery:   "z:vim",
		},
		{
			name:        "mixed invalid and valid",
			input:       "za:vim",
			wantFilters: map[string]bool{"aur": true},
			wantQuery:   "vim",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters, query := parseRepoFilter(tt.input)
			if !mapsEqual(filters, tt.wantFilters) {
				t.Errorf("parseRepoFilter(%q) returned filters %v, want %v", tt.input, filters, tt.wantFilters)
			}
			if query != tt.wantQuery {
				t.Errorf("parseRepoFilter(%q) returned query %q, want %q", tt.input, query, tt.wantQuery)
			}
		})
	}
}

// TestParseRemoveFilter tests the remove filter parsing
func TestParseRemoveFilter(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantFilters map[string]bool
		wantQuery   string
	}{
		{
			name:        "no filter",
			input:       "vim",
			wantFilters: nil,
			wantQuery:   "vim",
		},
		{
			name:        "total filter",
			input:       "t:",
			wantFilters: map[string]bool{"total": true},
			wantQuery:   "",
		},
		{
			name:        "explicit filter",
			input:       "e:firefox",
			wantFilters: map[string]bool{"explicit": true},
			wantQuery:   "firefox",
		},
		{
			name:        "multiple filters",
			input:       "et:package",
			wantFilters: map[string]bool{"total": true, "explicit": true},
			wantQuery:   "package",
		},
		{
			name:        "orphan filter",
			input:       "o:",
			wantFilters: map[string]bool{"orphan": true},
			wantQuery:   "",
		},
		{
			name:        "invalid filter char",
			input:       "z:vim",
			wantFilters: nil,
			wantQuery:   "z:vim",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filters, query := parseRemoveFilter(tt.input)
			if !mapsEqual(filters, tt.wantFilters) {
				t.Errorf("parseRemoveFilter(%q) returned filters %v, want %v", tt.input, filters, tt.wantFilters)
			}
			if query != tt.wantQuery {
				t.Errorf("parseRemoveFilter(%q) returned query %q, want %q", tt.input, query, tt.wantQuery)
			}
		})
	}
}

// TestParseSizeToBytes tests size string parsing
func TestParseSizeToBytes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int64
	}{
		{"bytes", "100 B", 100},
		{"kilobytes", "1.5 KB", 1536}, // 1.5 * 1024
		{"megabytes", "2 MiB", 2 * 1024 * 1024},
		{"gigabytes", "1.5 GiB", int64(1.5 * 1024 * 1024 * 1024)},
		{"terabytes", "1 TiB", 1024 * 1024 * 1024 * 1024},
		{"lowercase kb", "10 kb", 10 * 1024},
		{"lowercase mb", "5 mb", 5 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseSizeToBytes(tt.input)
			if result != tt.expected {
				t.Errorf("parseSizeToBytes(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// TestCountLines tests line counting
func TestCountLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty", "", 0},
		{"single line", "hello", 1},
		{"multiple lines", "line1\nline2\nline3", 3},
		{"trailing newline", "line1\nline2\n", 2},
		{"blank lines", "\n\n\n", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countLines(tt.input)
			if result != tt.expected {
				t.Errorf("countLines(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

// TestFormatBytes tests byte formatting
func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"0 bytes", 0, "0 B"},
		{"500 bytes", 500, "500 B"},
		{"1 KB", 1024, "1.00 KiB"},
		{"1.5 KB", 1536, "1.50 KiB"},
		{"1 MB", 1024 * 1024, "1.00 MiB"},
		{"1 GB", 1024 * 1024 * 1024, "1.00 GiB"},
		{"1 TB", 1024 * 1024 * 1024 * 1024, "1.00 TiB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

// TestComputeMatchIndices tests fuzzy match index calculation
func TestComputeMatchIndices(t *testing.T) {
	tests := []struct {
		name        string
		pkg         Package
		query       string
		wantIndices []int
	}{
		{
			name:        "exact match at start",
			pkg:         Package{Source: "aur", Name: "vim"},
			query:       "vim",
			wantIndices: []int{4, 5, 6}, // "aur/vim" - indices 4,5,6 match
		},
		{
			name:        "consecutive substring",
			pkg:         Package{Source: "extra", Name: "firefox"},
			query:       "fire",
			wantIndices: []int{6, 7, 8, 9}, // "extra/firefox" - "fire" at positions
		},
		{
			name:        "fuzzy match",
			pkg:         Package{Source: "core", Name: "bash"},
			query:       "bsh",
			wantIndices: []int{5, 7, 8}, // "core/bash": b(5), s(7), h(8)
		},
		{
			name:        "no match",
			pkg:         Package{Source: "aur", Name: "vim"},
			query:       "xyz",
			wantIndices: []int{},
		},
		{
			name:        "case insensitive",
			pkg:         Package{Source: "aur", Name: "VIM"},
			query:       "vim",
			wantIndices: []int{4, 5, 6}, // should match case-insensitively
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computeMatchIndices(tt.pkg, tt.query)
			if !intsEqual(result, tt.wantIndices) {
				t.Errorf("computeMatchIndices(%v, %q) = %v, want %v", tt.pkg, tt.query, result, tt.wantIndices)
			}
		})
	}
}

// TestComputeAllMatchIndices tests the batch match index computation
func TestComputeAllMatchIndices(t *testing.T) {
	packages := []Package{
		{Source: "aur", Name: "vim"},
		{Source: "extra", Name: "firefox"},
		{Source: "core", Name: "bash"},
	}
	query := "v"

	result := computeAllMatchIndices(packages, query)

	// Should have entries for packages that match
	if len(result) == 0 {
		t.Error("Expected some matches for query 'v'")
	}

	// Check that indices are within bounds
	for idx, indices := range result {
		if idx < 0 || idx >= len(packages) {
			t.Errorf("Invalid package index: %d", idx)
		}
		for _, charIdx := range indices {
			pkgStr := packages[idx].Source + "/" + packages[idx].Name
			if charIdx < 0 || charIdx >= len(pkgStr) {
				t.Errorf("Index %d out of range for package string %q", charIdx, pkgStr)
			}
		}
	}
}

// TestParseAUROutput tests AUR output parsing
func TestParseAUROutput(t *testing.T) {
	sampleOutput := `aur/yay 12.5.7-1
 AUR helper written in Go
aur/paru 1.9.3-1
 Another AUR helper
`

	result := parseAUROutput(sampleOutput)

	if len(result) != 2 {
		t.Errorf("Expected 2 AUR packages, got %d", len(result))
	}

	if len(result) > 0 {
		if result[0].Source != "aur" || result[0].Name != "yay" {
			t.Errorf("First package incorrect: %+v", result[0])
		}
		if result[0].Description != "AUR helper written in Go" {
			t.Errorf("Description mismatch: %q", result[0].Description)
		}
	}
}

// TestParseSearchOutput tests search output parsing
func TestParseSearchOutput(t *testing.T) {
	sampleOutput := `1 aur/yay 12.5.7-1 [+2480 ~39.12]
 AUR helper written in Go
2 extra/firefox 120.0-1
 Popular web browser
core/linux 6.5.3-1 [+12] [-]
 Linux kernel
`

	result := parsePackageOutput(sampleOutput)

	// Should parse all three packages
	if len(result) != 3 {
		t.Errorf("Expected 3 packages, got %d", len(result))
	}

	// Check first package
	if len(result) > 0 {
		if result[0].Source != "aur" || result[0].Name != "yay" {
			t.Errorf("First package incorrect: %+v", result[0])
		}
		if result[0].Version != "12.5.7-1" {
			t.Errorf("Version mismatch: %q", result[0].Version)
		}
	}
}

// TestGetThemeByName tests theme lookup
func TestGetThemeByName(t *testing.T) {
	tl := newTestThemeLoader()

	tests := []struct {
		name   string
		input  string
		wantOK bool
	}{
		{"exact match", "Catppuccin Mocha", true},
		{"lowercase", "catppuccin mocha", true},
		{"hyphens", "catppuccin-mocha", true},
		{"unknown", "Nonexistent Theme", false},
		{"partial", "cat", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := tl.GetThemeByConfigName(tt.input)
			if ok != tt.wantOK {
				t.Errorf("GetThemeByConfigName(%q) returned ok=%v, want %v", tt.input, ok, tt.wantOK)
			}
		})
	}
}

// TestListThemes tests theme listing
func TestListThemes(t *testing.T) {
	tl := newTestThemeLoader()
	themes := tl.ListThemes()
	if len(themes) == 0 {
		t.Fatal("ListThemes returned empty list")
	}

	if len(themes) < 1 {
		t.Logf("Expected at least 1 theme, got %d. Themes: %v", len(themes), themes)
	}
}

// Helper function to compare maps
func mapsEqual(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

// Helper function to compare int slices
func intsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if b[i] != v {
			return false
		}
	}
	return true
}

// Benchmark Fuzzy Filter (optional performance test)
func BenchmarkFuzzyFilter(b *testing.B) {
	packages := []Package{
		{Source: "aur", Name: "vim"},
		{Source: "extra", Name: "firefox"},
		{Source: "core", Name: "bash"},
		// ... could add more
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = fuzzyFilter(packages, "vim")
	}
}

// ============================================================================
// Test helpers for bubbletea model tests
// ============================================================================

// keyMsg creates a tea.KeyMsg for a single rune key (e.g., 'y', 's', 'a')
func keyMsg(key string) tea.KeyMsg {
	switch key {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	default:
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	}
}

// newTestModelUpdate creates a model in modeUpdate state with the given packages loaded
func newTestModelUpdate(tb testing.TB, packages []Package) *model {
	m := testModel(tb, modeUpdate, DefaultConfig())
	m.loading = false
	m.width = 120
	m.height = 40
	m.pendingUpdates = packages
	m.updatableAll = packages
	m.confirmScrollOffset = 0
	m.statusMessage = fmt.Sprintf("%d updates available", len(packages))
	return m
}

// testPackages returns a set of test packages for use in tests
func testPackages() []Package {
	return []Package{
		{Source: "core", Name: "linux", Version: "6.5.3-1"},
		{Source: "core", Name: "glibc", Version: "2.38-1"},
		{Source: "extra", Name: "firefox", Version: "120.0-1"},
		{Source: "extra", Name: "vlc", Version: "3.0.18-1"},
		{Source: "aur", Name: "yay", Version: "12.5.7-1"},
	}
}

// ============================================================================
// modeUpdate key handling tests
// ============================================================================

// TestUpdateModeYKeyExecutesUpdate tests that pressing 'y' in modeUpdate
// with pending updates triggers the update execution.
func TestUpdateModeYKeyExecutesUpdate(t *testing.T) {
	m := newTestModelUpdate(t, testPackages())

	result, cmd := m.Update(keyMsg("y"))
	resultModel := result.(*model)

	if resultModel.statusMessage != "Running system update..." {
		t.Errorf("Expected status 'Running system update...', got %q", resultModel.statusMessage)
	}
	if cmd == nil {
		t.Error("Expected a command to be returned for update execution")
	}
}

// TestUpdateModeUpperYKeyExecutesUpdate tests that pressing 'Y' in modeUpdate
// also triggers the update execution.
func TestUpdateModeUpperYKeyExecutesUpdate(t *testing.T) {
	m := newTestModelUpdate(t, testPackages())

	result, cmd := m.Update(keyMsg("Y"))
	resultModel := result.(*model)

	if resultModel.statusMessage != "Running system update..." {
		t.Errorf("Expected status 'Running system update...', got %q", resultModel.statusMessage)
	}
	if cmd == nil {
		t.Error("Expected a command to be returned for update execution")
	}
}

// TestUpdateModeEnterKeyShowsConfirmation tests that pressing enter in modeUpdate
// with pending updates triggers a confirmation dialog.
func TestUpdateModeEnterKeyShowsConfirmation(t *testing.T) {
	m := newTestModelUpdate(t, testPackages())

	result, cmd := m.Update(keyMsg("enter"))
	resultModel := result.(*model)

	if !strings.HasPrefix(resultModel.statusMessage, "Confirm update for") {
		t.Errorf("Expected status to start with 'Confirm update for', got %q", resultModel.statusMessage)
	}
	if cmd != nil {
		t.Error("Expected no command to be returned; should just show confirmation")
	}
	// Should show confirmation dialog
	if !resultModel.showConfirmation {
		t.Error("Should show confirmation dialog in modeUpdate after pressing enter")
	}
	if resultModel.confirmType != confirmUpdate {
		t.Errorf("Expected confirmType confirmUpdate, got %d", resultModel.confirmType)
	}
	if len(resultModel.confirmPackages) != len(testPackages()) {
		t.Errorf("Expected confirmPackages to have %d items, got %d", len(testPackages()), len(resultModel.confirmPackages))
	}
}

// TestUpdateModeAKeyExecutesUpdate tests that pressing 'a' in modeUpdate
// triggers the update directly.
func TestUpdateModeAKeyExecutesUpdate(t *testing.T) {
	m := newTestModelUpdate(t, testPackages())

	result, cmd := m.Update(keyMsg("a"))
	resultModel := result.(*model)

	if resultModel.statusMessage != "Running system update..." {
		t.Errorf("Expected status 'Running system update...', got %q", resultModel.statusMessage)
	}
	if cmd == nil {
		t.Error("Expected a command to be returned for update execution")
	}
}

func TestUpdateModeYKeyNoEffectWhenLoading(t *testing.T) {
	m := newTestModelUpdate(t, testPackages())
	m.loading = true

	result, cmd := m.Update(keyMsg("y"))
	resultModel := result.(*model)

	if resultModel.statusMessage == "Running system update..." {
		t.Error("Expected 'y' to have no effect while loading updates")
	}
	if cmd != nil {
		t.Error("Expected nil command when pressing 'y' while loading updates")
	}
}

func TestVersionVariable(t *testing.T) {
	if Version == "" {
		t.Error("Expected Version to have default non-empty value")
	}
}
