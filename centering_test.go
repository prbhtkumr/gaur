package main

import (
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func TestErrorOverlayCentering(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	m := testModel(t, modeInstall, DefaultConfig())
	m.errorTitle = "Test Error"
	m.errorMessage = "This is a multi-line\nerror message that should\nbe perfectly centered."

	width := 100
	height := 20
	output := m.renderErrorOverlay(width, height)

	lines := strings.Split(output, "\n")

	// Check if at least one line contains the error message and is centered
	found := false
	for _, line := range lines {
		if strings.Contains(line, "multi-line") {
			found = true
			// Calculate padding
			trimmed := strings.TrimSpace(line)
			contentWidth := lipgloss.Width(trimmed)

			// The line should be roughly in the middle of the 'width'
			lineOffset := strings.Index(line, trimmed)
			rightOffset := width - lineOffset - contentWidth

			// Allow for 1 character difference due to rounding
			if diff := lineOffset - rightOffset; diff > 1 || diff < -1 {
				t.Errorf("Line not centered: left padding %d, right padding %d. Line: %q", lineOffset, rightOffset, line)
			}
		}
	}

	if !found {
		t.Error("Error message lines not found in output")
	}
}

func TestConfirmationCentering(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	m := testModel(t, modeInstall, DefaultConfig())
	m.confirmType = confirmCleanNuke
	// We don't hardcode paths here because they are now dynamic
	m.dashboard.PacmanCachePath = "/var/cache/pacman/pkg"
	path, _ := GetAURCacheDir(&m.config)
	m.dashboard.AurCachePath = path

	width := 100
	height := 30
	output := m.renderConfirmationDialog(width, height, lipgloss.Color("6"))

	lines := strings.Split(output, "\n")

	// Check if the WARNING line is centered
	found := false
	for _, line := range lines {
		if strings.Contains(line, "WARNING") {
			found = true
			trimmed := strings.TrimSpace(line)
			contentWidth := lipgloss.Width(trimmed)
			lineOffset := strings.Index(line, trimmed)
			rightOffset := width - lineOffset - contentWidth

			if diff := lineOffset - rightOffset; diff > 1 || diff < -1 {
				t.Errorf("Warning line not centered: left padding %d, right padding %d. Line: %q", lineOffset, rightOffset, line)
			}
		}
	}

	if !found {
		t.Error("Warning message not found in confirmation output")
	}
}

func TestSearchStatusAlignment(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	m := testModel(t, modeInstall, DefaultConfig())
	m.searchStatus = "Searching..."
	m.searchingAUR = true
	m.width = 100
	m.height = 40

	output := m.View()
	lines := strings.Split(output, "\n")

	var inputLine, statusLine string
	for _, line := range lines {
		// Find line starting with prompt "> "
		if strings.Contains(line, "> ") && !strings.Contains(line, "Searching") {
			inputLine = line
		}
		if strings.Contains(line, "Searching") {
			statusLine = line
		}
	}

	if inputLine == "" || statusLine == "" {
		t.Fatalf("Could not find input or status lines in View output")
	}

	// The prompt starts after some padding from JoinVertical/Place
	// Find the visual column where the query would start (after "> ")
	ansi := regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")
	plainInput := ansi.ReplaceAllString(inputLine, "")
	inputRunes := []rune(plainInput)

	queryStartCol := -1
	for i := 0; i < len(inputRunes)-1; i++ {
		if inputRunes[i] == '>' && inputRunes[i+1] == ' ' {
			queryStartCol = i + 2
			break
		}
	}

	// Find where "Searching" starts in plain statusLine
	plainStatus := ansi.ReplaceAllString(statusLine, "")
	statusRunes := []rune(plainStatus)
	searchingCol := -1

	statusStr := string(statusRunes)
	searchingIdx := strings.Index(statusStr, "Searching")
	if searchingIdx != -1 {
		searchingCol = len([]rune(statusStr[:searchingIdx]))
	}

	if queryStartCol != searchingCol {
		t.Errorf("Search status not aligned with query. Query starts at visual col %d, status starts at visual col %d", queryStartCol, searchingCol)
		t.Errorf("Plain input line: %q", plainInput)
		t.Errorf("Plain status line: %q", plainStatus)
	}
}

func TestFooterCentering(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)

	m := testModel(t, modeInstall, DefaultConfig())
	m.width = 100
	m.height = 40
	m.loading = false

	modes := []viewMode{modeInstall, modeDashboard, modeUpdate, modeRemove, modeCacheMenu, modeCacheSelective, modeUpdateSelective}

	for _, mode := range modes {
		t.Run("mode "+string(rune(mode)), func(t *testing.T) {
			m.mode = mode
			m.previousMode = modeInstall // for settings overlay check
			output := m.View()
			lines := strings.Split(output, "\n")

			// The footer is the very last line
			footerLine := lines[len(lines)-1]

			// Help text contains "search" and "quit" normally
			if !strings.Contains(footerLine, "search") && !strings.Contains(footerLine, "quit") && !strings.Contains(footerLine, "dash") {
				t.Fatalf("Footer line not found or doesn't contain expected help text. Mode: %v, Line: %q", mode, footerLine)
			}

			trimmed := strings.TrimSpace(footerLine)
			contentWidth := lipgloss.Width(trimmed)

			// Find actual left padding
			// We need to handle ANSI codes
			ansi := regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")
			plainLine := ansi.ReplaceAllString(footerLine, "")

			leftPadding := 0
			for _, r := range plainLine {
				if r == ' ' {
					leftPadding++
				} else {
					break
				}
			}

			rightPadding := m.width - leftPadding - contentWidth

			// For centering, left and right padding should be roughly equal
			diff := leftPadding - rightPadding
			if diff > 2 || diff < -2 {
				t.Errorf("Footer not centered in mode %v: left padding %d, right padding %d, diff %d. Line length: %d, content width: %d",
					mode, leftPadding, rightPadding, diff, len(plainLine), contentWidth)
			}
		})
	}
}
