package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestHighlightMatches(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	tl := newTestThemeLoader()
	theme, _ := tl.GetTheme("Catppuccin Mocha")
	setTheme(theme)
	s := "aur/vim"
	matchedIndices := []int{4, 5, 6}

	result := highlightMatches(s, matchedIndices)

	if !strings.Contains(result, "v") || !strings.Contains(result, "i") || !strings.Contains(result, "m") {
		t.Errorf("highlightMatches failed to include matched characters: %q", result)
	}

	// Check if the number of highlighted characters matches our expectation
	// This is tricky because of ANSI codes, but we can check for their presence
	if !strings.Contains(result, "\x1b[") {
		t.Error("highlightMatches didn't include ANSI escape codes for highlighting")
	}
}

func TestHighlightMatchesWithSourceColor(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	tl := newTestThemeLoader()
	theme, _ := tl.GetTheme("Catppuccin Mocha")
	setTheme(theme)
	pkg := Package{Source: "aur", Name: "vim"}
	matchedIndices := []int{4, 5, 6}

	result := highlightMatchesWithSourceColor(pkg, matchedIndices)

	if lipgloss.Width(result) != len("aur/vim") {
		t.Errorf("highlightMatchesWithSourceColor width = %d, want %d", lipgloss.Width(result), len("aur/vim"))
	}

	// Check without matches
	resultNoMatch := highlightMatchesWithSourceColor(pkg, nil)
	if lipgloss.Width(resultNoMatch) != len("aur/vim") {
		t.Errorf("highlightMatchesWithSourceColor (no match) width = %d, want %d", lipgloss.Width(resultNoMatch), len("aur/vim"))
	}
}

func TestTruncateWithAnsi(t *testing.T) {
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("Hello World")

	tests := []struct {
		name     string
		input    string
		width    int
		expected int
	}{
		{"no truncate", "hello", 10, 5},
		{"simple truncate", "hello world", 5, 5},
		{"ansi truncate", red, 5, 5},
		{"zero width", "abc", 0, 0},
		{"multibyte bullet", "• bullet", 1, 1},
		{"multibyte with spaces", "  • bullet", 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncateWithAnsi(tt.input, tt.width)
			actualWidth := lipgloss.Width(result)
			if actualWidth != tt.expected {
				t.Errorf("truncateWithAnsi(%q, %d) width = %d, want %d", tt.input, tt.width, actualWidth, tt.expected)
			}
		})
	}
}

func TestSubstringAnsi(t *testing.T) {
	red := lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("Hello World")

	tests := []struct {
		name     string
		input    string
		skip     int
		expected int
	}{
		{"no skip", "hello", 0, 5},
		{"simple skip", "hello", 2, 3},
		{"ansi skip", red, 6, 5},
		{"bullet skip", "• bullet", 1, 7},
		{"skip all", "abc", 5, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := substringAnsi(tt.input, tt.skip)
			actualWidth := lipgloss.Width(result)
			if actualWidth != tt.expected {
				t.Errorf("substringAnsi(%q, %d) width = %d, want %d", tt.input, tt.skip, actualWidth, tt.expected)
			}
		})
	}
}

func TestSafeJoinVertical(t *testing.T) {
	width := 20
	height := 5
	header := "Header"
	panels := []string{
		"Panel1\nLine2",
		"Panel2",
	}
	footer := "Footer"

	got := SafeJoinVertical(width, height, header, panels, footer)
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")

	if len(lines) != height {
		t.Errorf("SafeJoinVertical height = %d, expected %d", len(lines), height)
	}

	for i, line := range lines {
		if lipgloss.Width(line) != width {
			t.Errorf("Line %d width = %d, expected %d. Line: %q", i, lipgloss.Width(line), width, line)
		}
	}
}

func TestOverlayCompositingLogic(t *testing.T) {
	// Simulate the compositing logic used in renderUpdateSelectiveView
	innerWidth := 40
	overlayWidth := 20
	paddingX := (innerWidth - overlayWidth) / 2 // 10

	bgLine := strings.Repeat("x", innerWidth)
	paneLine := "│" + strings.Repeat("-", overlayWidth-2) + "│"

	// Composite one line logic
	leftStr := truncateWithAnsi(bgLine, paddingX)
	if lipgloss.Width(leftStr) < paddingX {
		leftStr += strings.Repeat(" ", paddingX-lipgloss.Width(leftStr))
	}

	rightStart := paddingX + overlayWidth
	rightPartWidth := innerWidth - rightStart
	rightStr := substringAnsi(bgLine, rightStart)
	rightStr = truncateWithAnsi(rightStr, rightPartWidth)
	if lipgloss.Width(rightStr) < rightPartWidth {
		rightStr += strings.Repeat(" ", rightPartWidth-lipgloss.Width(rightStr))
	}

	composite := leftStr + paneLine + rightStr

	if lipgloss.Width(composite) != innerWidth {
		t.Errorf("Composite width = %d, expected %d", lipgloss.Width(composite), innerWidth)
	}
}
func TestMaintainBackground(t *testing.T) {
	bgColor := lipgloss.Color("235")
	input := "\x1b[31mRed\x1b[0m Text"

	result := maintainBackground(input, bgColor)

	// Result should contain the background color sequence after the reset
	if !strings.Contains(result, "\x1b[0m") {
		t.Error("maintainBackground stripped the reset code")
	}

	// Ensure it still has the original color code too
	if !strings.Contains(result, "\x1b[31m") {
		t.Error("maintainBackground stripped original foreground code")
	}
}

func TestSimplifyErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			"no repeats",
			"network error: connection refused",
			"network error: connection refused",
		},
		{
			"simple repeats",
			"error: error: something failed",
			"error: something failed",
		},
		{
			"complex repeats from screenshot",
			"error sending request for url: error trying to connect: dns error: failed to lookup address: Temporary failure: error trying to connect: dns error: failed to lookup address: Temporary failure",
			"error sending request for url: error trying to connect: dns error: failed to lookup address: Temporary failure",
		},
		{
			"empty string",
			"",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := simplifyErrorMessage(tt.input)
			if result != tt.expected {
				t.Errorf("simplifyErrorMessage(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

// stripAnsi removes ANSI escape codes from a string for easier testing.
func stripAnsi(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func TestRenderPaginatedList(t *testing.T) {
	items := []string{"apple", "banana", "cherry", "date", "elderberry"}

	// 1. Empty list
	empty := RenderPaginatedList(PaginatedListConfig{
		TotalCount:     0,
		SelectedIndex:  0,
		ViewportHeight: 5,
		ContentWidth:   20,
	})
	if empty != "" {
		t.Errorf("Expected empty string for 0 items, got %q", empty)
	}

	// 2. Viewport height larger than items, forward order
	forward := RenderPaginatedList(PaginatedListConfig{
		TotalCount:     len(items),
		SelectedIndex:  0,
		ViewportHeight: 10,
		ContentWidth:   20,
		Reversed:       false,
		RenderItem: func(idx int, width int) string {
			return items[idx]
		},
	})
	expectedForward := "apple\nbanana\ncherry\ndate\nelderberry"
	if forward != expectedForward {
		t.Errorf("Forward list mismatch: got %q, want %q", forward, expectedForward)
	}

	// 3. Reversed order (bottom-up)
	reversed := RenderPaginatedList(PaginatedListConfig{
		TotalCount:     len(items),
		SelectedIndex:  0,
		ViewportHeight: 10,
		ContentWidth:   20,
		Reversed:       true,
		RenderItem: func(idx int, width int) string {
			return items[idx]
		},
	})
	expectedReversed := "elderberry\ndate\ncherry\nbanana\napple"
	if reversed != expectedReversed {
		t.Errorf("Reversed list mismatch: got %q, want %q", reversed, expectedReversed)
	}

	// 4. Windowing with scrollbar when items exceed viewport
	paged := RenderPaginatedList(PaginatedListConfig{
		TotalCount:     len(items),
		SelectedIndex:  3, // "date"
		ViewportHeight: 3,
		ContentWidth:   20,
		ActiveColor:    lipgloss.Color("35"),
		Reversed:       false,
		RenderItem: func(idx int, width int) string {
			return items[idx]
		},
	})
	// With selectedIndex=3 and viewportHeight=3: startIdx = 3 - 3 + 1 = 1 ("banana")
	// window covers idx 1, 2, 3: "banana", "cherry", "date"
	if !strings.Contains(paged, "banana") || !strings.Contains(paged, "cherry") || !strings.Contains(paged, "date") {
		t.Errorf("Expected paged output to contain banana, cherry, date; got %q", paged)
	}
	if strings.Contains(paged, "apple") {
		t.Errorf("Did not expect apple in paged window; got %q", paged)
	}
}

func TestMapSlice(t *testing.T) {
	// Test nil slice
	var nilSlice []int
	mappedNil := mapSlice(nilSlice, func(n int) int { return n * 2 })
	if mappedNil != nil {
		t.Errorf("Expected nil for nil input, got %v", mappedNil)
	}

	// Test empty slice
	emptySlice := []int{}
	mappedEmpty := mapSlice(emptySlice, func(n int) int { return n * 2 })
	if mappedEmpty == nil || len(mappedEmpty) != 0 {
		t.Errorf("Expected empty non-nil slice, got %v", mappedEmpty)
	}

	// Test mapping struct to string
	type testItem struct {
		Name string
	}
	items := []testItem{
		{Name: "alpha"},
		{Name: "beta"},
		{Name: "gamma"},
	}
	names := mapSlice(items, func(item testItem) string { return item.Name })
	expected := []string{"alpha", "beta", "gamma"}
	if len(names) != len(expected) {
		t.Fatalf("Length mismatch: got %d, want %d", len(names), len(expected))
	}
	for i := range names {
		if names[i] != expected[i] {
			t.Errorf("Index %d mismatch: got %q, want %q", i, names[i], expected[i])
		}
	}
}

func TestParsePackageRepoMap(t *testing.T) {
	out := []byte("core linux 6.6.1-arch1-1 [installed]\nextra firefox 120.0-1\nmultilib steam 1.0.0.78-2\ninvalidline\n")
	repoMap := parsePackageRepoMap(out)

	if repoMap["linux"] != "core" {
		t.Errorf("Expected linux -> core, got %q", repoMap["linux"])
	}
	if repoMap["firefox"] != "extra" {
		t.Errorf("Expected firefox -> extra, got %q", repoMap["firefox"])
	}
	if repoMap["steam"] != "multilib" {
		t.Errorf("Expected steam -> multilib, got %q", repoMap["steam"])
	}
	if _, ok := repoMap["invalidline"]; ok {
		t.Errorf("Unexpected entry for invalidline in repoMap")
	}
}

func TestParsePackageNameSet(t *testing.T) {
	out := []byte("linux 6.6.1\nfirefox 120.0\n\nsteam\n")
	set := parsePackageNameSet(out)

	if !set["linux"] {
		t.Errorf("Expected linux in set")
	}
	if !set["firefox"] {
		t.Errorf("Expected firefox in set")
	}
	if !set["steam"] {
		t.Errorf("Expected steam in set")
	}
	if len(set) != 3 {
		t.Errorf("Expected set size 3, got %d", len(set))
	}
}

func TestQueryPackageHelpers(t *testing.T) {
	ctx := context.Background()
	mock := &MockCommandRunner{
		RunContextFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			if len(args) > 0 && args[0] == "-Sl" {
				return []byte("core linux 6.6.1\nextra vim 9.0\n"), nil
			}
			if len(args) > 0 && args[0] == "-Qm" {
				return []byte("google-chrome 120.0\nspotify 1.2\n"), nil
			}
			return nil, fmt.Errorf("unexpected args: %v", args)
		},
	}

	repoMap, err := queryPackageRepoMap(ctx, mock)
	if err != nil {
		t.Fatalf("queryPackageRepoMap failed: %v", err)
	}
	if repoMap["linux"] != "core" || repoMap["vim"] != "extra" {
		t.Errorf("Unexpected repoMap: %v", repoMap)
	}

	foreignSet, err := queryPackageSet(ctx, mock, "-Qm")
	if err != nil {
		t.Fatalf("queryPackageSet failed: %v", err)
	}
	if !foreignSet["google-chrome"] || !foreignSet["spotify"] {
		t.Errorf("Unexpected foreignSet: %v", foreignSet)
	}

	// Error path
	errMock := &MockCommandRunner{
		RunContextFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, fmt.Errorf("command failed")
		},
	}
	if _, err := queryPackageRepoMap(ctx, errMock); err == nil {
		t.Errorf("Expected error from queryPackageRepoMap")
	}
	if _, err := queryPackageSet(ctx, errMock, "-Qm"); err == nil {
		t.Errorf("Expected error from queryPackageSet")
	}
}


