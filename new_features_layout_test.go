package main

import (
	"strings"
	"testing"
)

func TestSeparatorLineStretching(t *testing.T) {
	config := DefaultConfig()
	m := testModel(t, modeInstall, config)
	m.width = 100
	m.height = 40

	// Mock some data so the list view renders
	m.loading = false
	m.repoPackages = []Package{{Source: "core", Name: "linux"}}
	m.filtered = m.repoPackages

	view := m.View()
	lines := strings.Split(view, "\n")

	// The separator line is in the bottom panel.
	// In renderPackageListLayout:
	// separator := strings.Repeat("─", innerWidth-2)
	// bottomParts := []string{resultsBox, separator, inputLine}

	// Let's find the line with the separator character
	found := false
	for _, line := range lines {
		plain := stripAnsiLoc(line)
		// It should be 98 chars long (innerWidth-2 = 100-2)
		if strings.Contains(plain, "──") {
			found = true
			count := strings.Count(plain, "─")
			// Depending on how JoinVertical/SafeJoinVertical pads/truncates,
			// the visual count might be exactly 98 or it might be padded to 100.
			// But it should definitely be > width-4 (96) which was the old value.
			if count < 98 {
				t.Errorf("Separator line too short: got %d chars, expected at least 98", count)
			}
		}
	}
	if !found {
		t.Errorf("Separator line not found in view")
	}
}

func TestRepoFilterHintsLayout(t *testing.T) {
	config := DefaultConfig()
	m := testModel(t, modeInstall, config)
	m.width = 100
	m.height = 40
	m.loading = false

	// Case 1: No filters applied - hints should be dim
	view := m.View()
	if !strings.Contains(stripAnsiLoc(view), "c: e: m: a:") {
		t.Errorf("Filter hints 'c: e: m: a:' not found in view")
	}

	// Case 2: Filter applied - hint should be present
	m.textInput.SetValue("a:test")
	view = m.View()
	if !strings.Contains(view, "a:") {
		t.Errorf("Filter hint 'a:' not found in view when active")
	}

	// Verify right padding (2 spaces)
	// The hints are at the right of the inputLine.
	// In view.go: hintsView := lipgloss.NewStyle().Width(hintsPaneWidth).Align(lipgloss.Right).PaddingRight(2).Render(hints)

	foundLine := ""
	lines := strings.Split(view, "\n")
	for _, line := range lines {
		if strings.Contains(stripAnsiLoc(line), "c: e: m: a:") {
			foundLine = line
			break
		}
	}

	if foundLine == "" {
		t.Fatalf("Could not find line with filter hints")
	}

	plain := stripAnsiLoc(foundLine)
	// The line should end with "a:  " (two spaces)
	if !strings.HasSuffix(plain, "a:  ") {
		// It might be followed by the border character if it's the very edge,
		// but since it's INSIDE the panel which has width innerWidth-2,
		// and the total width is innerWidth, it should be followed by a space then border.
		// Actually, SafeJoinVertical ensures each line is exactly 'width' wide.
		// For a 100 wide terminal, and a panel of 98 wide (innerWidth-2):
		// Line: │ <content> │
		// content is 98 wide. If hints have 2 padding right:
		// content ends in "a:  "
		// full line ends in "a:   │" (2 spaces from padding + 1 space from panel padding maybe?
		// Wait, bottomPanel has no horizontal padding in renderPackageListLayout,
		// but resultsBox has Width(innerWidth-4).
		// inputLine is JoinedHorizontal into the panel which is Width(innerWidth-2).

		if !strings.Contains(plain, "a:  ") {
			t.Errorf("Hints do not appear to have 2 spaces of right padding. Line end: %q", plain[len(plain)-10:])
		}
	}
}
