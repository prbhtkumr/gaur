package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// isValidPackageName checks if a package name contains only safe characters.
// Valid package names contain only alphanumeric, @, ., _, +, and - characters.
// This prevents command injection through malicious package names.
func isValidPackageName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '@' || r == '.' ||
			r == '_' || r == '+' || r == '-') {
			return false
		}
	}
	return true
}

// sanitizePackageNames filters a list of package names to only include valid ones.
// Returns the filtered list and a boolean indicating if all names were valid.
func sanitizePackageNames(names []string) ([]string, bool) {
	var valid []string
	allValid := true
	for _, name := range names {
		if isValidPackageName(name) {
			valid = append(valid, name)
		} else {
			allValid = false
		}
	}
	return valid, allValid
}

// fuzzyFilter filters packages using fzf for fuzzy matching.
// Returns filtered packages sorted by fzf's relevance ranking.
func fuzzyFilter(packages []Package, query string) []Package {
	if query == "" || len(packages) == 0 {
		return packages
	}

	// Build input for fzf: one package name per line with index
	var input strings.Builder
	for i, pkg := range packages {
		input.WriteString(fmt.Sprintf("%d\t%s\n", i, pkg.Name))
	}

	stdout, _ := runner.RunWithInput(input.String(), "fzf", "--filter", query, "-d", "\t", "-n2", "--tiebreak=begin,length")

	// Parse output and rebuild package list
	var result []Package
	lines := strings.Split(strings.TrimSpace(string(stdout)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) >= 1 {
			var idx int
			if _, err := fmt.Sscanf(parts[0], "%d", &idx); err == nil && idx >= 0 && idx < len(packages) {
				result = append(result, packages[idx])
			}
		}
	}

	if len(result) == 0 {
		queryLower := strings.ToLower(query)
		for _, pkg := range packages {
			if strings.Contains(strings.ToLower(pkg.Name), queryLower) {
				result = append(result, pkg)
			}
		}
	}

	return result
}

// computeMatchIndices finds the character indices in the package string (source/name)
// that match the query using case-insensitive fuzzy matching.
// Returns indices relative to the full "source/name" string.
func computeMatchIndices(pkg Package, query string) []int {
	if query == "" {
		return nil
	}

	// Determine the string being displayed
	var pkgStr string
	if pkg.Source != "" {
		pkgStr = pkg.Source + "/" + pkg.Name
	} else {
		pkgStr = pkg.Name
	}

	pkgLower := strings.ToLower(pkgStr)
	queryLower := strings.ToLower(query)

	var indices []int

	// Priority 1: Exact substring match
	if idx := strings.Index(pkgLower, queryLower); idx != -1 {
		for i := 0; i < len(queryLower); i++ {
			indices = append(indices, idx+i)
		}
		return indices
	}

	// Priority 2: Fuzzy match
	pkgRunes := []rune(pkgLower)

	// Simple greedy forward match
	currentPkgIdx := 0
	for _, qr := range queryLower {
		found := false
		for currentPkgIdx < len(pkgRunes) {
			if pkgRunes[currentPkgIdx] == qr {
				indices = append(indices, currentPkgIdx)
				currentPkgIdx++
				found = true
				break
			}
			currentPkgIdx++
		}
		if !found {
			return nil
		}
	}

	return indices
}

// computeAllMatchIndices computes match indices for all packages in the filtered list.
// Returns a map from package index to matched character indices.
func computeAllMatchIndices(packages []Package, query string) map[int][]int {
	if query == "" || len(packages) == 0 {
		return nil
	}

	result := make(map[int][]int, len(packages))
	for i, pkg := range packages {
		indices := computeMatchIndices(pkg, query)
		if len(indices) > 0 {
			result[i] = indices
		}
	}
	return result
}

// highlightMatches renders a string with matched character indices highlighted.
// It uses a map for O(1) lookup of matched positions and handles Unicode correctly.
func highlightMatches(s string, matchedIndices []int) string {
	if len(matchedIndices) == 0 {
		return s
	}

	matchSet := make(map[int]struct{}, len(matchedIndices))
	for _, idx := range matchedIndices {
		matchSet[idx] = struct{}{}
	}

	var result strings.Builder
	result.Grow(len(s) * 2)
	runes := []rune(s)
	for i, r := range runes {
		if _, matched := matchSet[i]; matched {
			result.WriteString(matchHighlightStyle.Render(string(r)))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// highlightMatchesWithSourceColor renders a package string (source/name) with:
// - Source portion colored by repository
// - Matched characters highlighted with matchHighlightStyle
// - Non-matched characters in normal text color
func highlightMatchesWithSourceColor(pkg Package, matchedIndices []int) string {
	pkgStr := pkg.Source + "/" + pkg.Name

	sourceColor, hasSourceColor := sourceColors[pkg.Source]

	if len(matchedIndices) == 0 {
		if hasSourceColor {
			return lipgloss.NewStyle().Foreground(sourceColor).Render(pkg.Source) + "/" + pkg.Name
		}
		return pkgStr
	}

	matchSet := make(map[int]struct{}, len(matchedIndices))
	for _, idx := range matchedIndices {
		matchSet[idx] = struct{}{}
	}

	slashIdx := len(pkg.Source)

	var result strings.Builder
	result.Grow(len(pkgStr) * 2)
	runes := []rune(pkgStr)

	for i, r := range runes {
		if _, matched := matchSet[i]; matched {

			result.WriteString(matchHighlightStyle.Render(string(r)))
		} else if i < slashIdx && hasSourceColor {

			result.WriteString(lipgloss.NewStyle().Foreground(sourceColor).Render(string(r)))
		} else {

			result.WriteRune(r)
		}
	}
	return result.String()
}

// Repo filter character mappings
var repoFilterChars = map[rune]string{
	'c': "core",
	'e': "extra",
	'm': "multilib",
	'a': "aur",
}

// removeFilterChars maps single characters to package filter types for remove mode
var removeFilterChars = map[rune]string{
	't': "total",
	'e': "explicit",
	'l': "explicit", // Alias for local/explicit
	'f': "foreign",
	'a': "foreign", // Alias for AUR/foreign
	'o': "orphan",
}

// parseRepoFilter extracts repo filters and search query from input
// Supports combined filters like "ae:", "cem:", "aem:" in any order
// Returns (repoFilters, searchQuery) where repoFilters is empty if no filter specified
func parseRepoFilter(input string) (map[string]bool, string) {
	input = strings.TrimSpace(input)

	colonIdx := strings.Index(input, ":")
	if colonIdx == -1 {
		return nil, input
	}

	prefix := strings.ToLower(input[:colonIdx])
	searchQuery := strings.TrimSpace(input[colonIdx+1:])

	repoFilters := make(map[string]bool)
	for _, ch := range prefix {
		if repo, ok := repoFilterChars[ch]; ok {
			repoFilters[repo] = true
		}
	}

	if len(repoFilters) == 0 {
		return nil, input
	}

	return repoFilters, searchQuery
}

// parseRemoveFilter extracts source filters and search query from input for remove mode
// Supports 'a:' for AUR/foreign packages and 'l:' for local/official packages
func parseRemoveFilter(input string) (map[string]bool, string) {
	input = strings.TrimSpace(input)

	colonIdx := strings.Index(input, ":")
	if colonIdx == -1 {
		return nil, input
	}

	prefix := strings.ToLower(input[:colonIdx])
	searchQuery := strings.TrimSpace(input[colonIdx+1:])

	sourceFilters := make(map[string]bool)
	for _, ch := range prefix {
		if source, ok := removeFilterChars[ch]; ok {
			sourceFilters[source] = true
		}
	}

	if len(sourceFilters) == 0 {
		return nil, input
	}

	return sourceFilters, searchQuery
}

// parsePackageOutput parses package search output from pacman/paru/yay.
// It handles both AUR helper output (paru -Ss, yay -Ss) and pacman -Ss output.
// The format is: repo/name version [installed] followed by description on next line.
func parsePackageOutput(output string) []Package {
	var packages []Package
	lines := strings.Split(output, "\n")

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Skip empty lines and description lines (indented)
		if line == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}

		// Handle numbered output (some helpers prefix with numbers)
		trimmedLine := strings.TrimSpace(line)
		fields := strings.Fields(trimmedLine)
		if len(fields) == 0 {
			continue
		}

		// Find the repo/name field (contains "/")
		pkgField := ""
		pkgFieldIdx := 0
		for idx, f := range fields {
			if strings.Contains(f, "/") {
				pkgField = f
				pkgFieldIdx = idx
				break
			}
		}

		if pkgField == "" {
			continue
		}

		parts := strings.SplitN(pkgField, "/", 2)
		if len(parts) != 2 {
			continue
		}

		source := parts[0]
		name := parts[1]

		// Get version (next field after repo/name)
		version := ""
		if pkgFieldIdx+1 < len(fields) {
			version = fields[pkgFieldIdx+1]
		}

		// Check for installed status
		installed := strings.Contains(line, "[Installed")

		// Get description from next line if it's indented
		description := ""
		if i+1 < len(lines) {
			nextLine := lines[i+1]
			if strings.HasPrefix(nextLine, " ") || strings.HasPrefix(nextLine, "\t") {
				description = strings.TrimSpace(nextLine)
			}
		}

		packages = append(packages, Package{
			Source:      source,
			Name:        name,
			Version:     version,
			Description: description,
			Installed:   installed,
		})
	}

	return packages
}

// parseAUROutput parses paru/yay -Ss output for AUR packages.
// Deprecated: Use parsePackageOutput instead.
func parseAUROutput(output string) []Package {
	return parsePackageOutput(output)
}

func countLines(output string) int {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0
	}
	return len(lines)
}

// parseSizeToBytes converts a human-readable size (e.g., "10.5 GiB", "500 M") to bytes
func parseSizeToBytes(size string) int64 {
	size = strings.TrimSpace(size)
	var value float64
	var unit string
	_, _ = fmt.Sscanf(size, "%f %s", &value, &unit)

	unit = strings.ToLower(unit)
	switch {
	case strings.HasPrefix(unit, "t"):
		return int64(value * 1024 * 1024 * 1024 * 1024)
	case strings.HasPrefix(unit, "g"):
		return int64(value * 1024 * 1024 * 1024)
	case strings.HasPrefix(unit, "m"):
		return int64(value * 1024 * 1024)
	case strings.HasPrefix(unit, "k"):
		return int64(value * 1024)
	default:
		return int64(value)
	}
}

// calculateDirSize walks a directory and returns the total size of all files in bytes.
// It gracefully handles permission errors by skipping inaccessible files.
func calculateDirSize(path string) int64 {
	var size int64
	err := filepath.WalkDir(path, func(_ string, d os.DirEntry, err error) error {
		if err != nil {

			return nil
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				size += info.Size()
			}
		}
		return nil
	})

	if err != nil {
		return 0
	}
	return size
}

// formatBytes converts bytes to human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// fitToBox truncates a string to fit within a specified width and height.
// This is a convenience function that combines truncateHeight and truncateWithAnsi.
func fitToBox(s string, width, height int) string {
	return truncateWithAnsi(truncateHeight(s, height), width)
}

// truncateWithAnsi truncates a string to a visual width, preserving ANSI codes.
// It uses lipgloss.Width to correctly account for multibyte and wide characters.
func truncateWithAnsi(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return "\x1b[0m"
	}

	var result strings.Builder
	currentWidth := 0
	inEscape := false

	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			result.WriteRune(r)
			continue
		}
		if inEscape {
			result.WriteRune(r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}

		w := lipgloss.Width(string(r))
		if currentWidth+w > maxWidth {
			break
		}
		result.WriteRune(r)
		currentWidth += w
	}

	result.WriteString("\x1b[0m")
	return result.String()
}

// truncateHeight truncates a string to a specified number of lines.
func truncateHeight(s string, maxHeight int) string {
	if maxHeight <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= maxHeight {
		return s
	}
	return strings.Join(lines[:maxHeight], "\n")
}

// substringAnsi returns a substring of s that skips skipWidth visual characters,
// while preserving ANSI escape sequences correctly.
func substringAnsi(s string, skipWidth int) string {
	var result strings.Builder
	currentWidth := 0
	inEscape := false

	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			// We only include escape sequences if we are past the skip point,
			// OR if they are required to set state for later.
			// To be safe and simple, we include all escape sequences but
			// we must be careful they don't count towards width.
			result.WriteRune(r)
			continue
		}
		if inEscape {
			result.WriteRune(r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}

		w := lipgloss.Width(string(r))
		if currentWidth >= skipWidth {
			result.WriteRune(r)
		}
		currentWidth += w
	}

	return result.String()
}

// renderCenteredFooter creates a footer line centered within the given width.
// It takes the content to center and truncates if necessary.
func renderCenteredFooter(content string, width int) string {
	contentWidth := lipgloss.Width(content)
	padding := (width - contentWidth) / 2
	if padding < 0 {
		padding = 0
	}
	footer := strings.Repeat(" ", padding) + content
	if lipgloss.Width(footer) > width {
		footer = truncateWithAnsi(footer, width)
	}
	return footer
}

// overlayAt renders an overlay onto a base string at the given row and column.
// If startRow is negative, the overlay is vertically centered.
// If startCol is negative, the overlay is horizontally centered.
// Both base and overlay should be rectangular strings (same width per line).
// Returns the composited result maintaining the base's dimensions.
func overlayAt(base, overlay string, baseWidth, baseHeight, startRow, startCol int) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	overlayHeight := len(overlayLines)
	if overlayHeight == 0 {
		return base
	}

	overlayWidth := lipgloss.Width(overlayLines[0])
	for _, ol := range overlayLines {
		if w := lipgloss.Width(ol); w > overlayWidth {
			overlayWidth = w
		}
	}

	// Ensure base has enough lines
	for len(baseLines) < baseHeight {
		baseLines = append(baseLines, strings.Repeat(" ", baseWidth))
	}

	startY := startRow
	if startY < 0 {
		startY = (baseHeight - overlayHeight) / 2
	}
	if startY < 0 {
		startY = 0
	}

	startX := startCol
	if startX < 0 {
		startX = (baseWidth - overlayWidth) / 2
	}
	if startX < 0 {
		startX = 0
	}

	result := make([]string, len(baseLines))
	copy(result, baseLines)

	for y := 0; y < overlayHeight; y++ {
		targetY := startY + y
		if targetY >= 0 && targetY < len(result) {
			bgLine := result[targetY]

			// Ensure bgLine is at least baseWidth chars wide
			bgWidth := lipgloss.Width(bgLine)
			if bgWidth < baseWidth {
				bgLine += strings.Repeat(" ", baseWidth-bgWidth)
			}

			// Ensure overlay line is exactly overlayWidth chars wide
			ovLine := overlayLines[y]
			ow := lipgloss.Width(ovLine)
			if ow < overlayWidth {
				ovLine += strings.Repeat(" ", overlayWidth-ow)
			} else if ow > overlayWidth {
				ovLine = truncateWithAnsi(ovLine, overlayWidth)
			}

			// Reconstruct the line using precise slicing
			left := ""
			if startX > 0 {
				left = truncateWithAnsi(bgLine, startX)
				leftWidth := lipgloss.Width(left)
				if leftWidth < startX {
					left += strings.Repeat(" ", startX-leftWidth)
				}
			}

			right := ""
			rightStart := startX + overlayWidth
			rightPartWidth := baseWidth - rightStart
			if rightPartWidth > 0 {
				right = substringAnsi(bgLine, rightStart)
				right = truncateWithAnsi(right, rightPartWidth)
				rw := lipgloss.Width(right)
				if rw < rightPartWidth {
					right += strings.Repeat(" ", rightPartWidth-rw)
				}
			}

			result[targetY] = left + ovLine + right
		}
	}

	return strings.Join(result, "\n")
}

// overlayOnBase renders an overlay centered on top of a base string.
// Both base and overlay should be rectangular strings (same width per line).
// Returns the composited result maintaining the base's dimensions.
func overlayOnBase(base, overlay string, baseWidth, baseHeight int) string {
	return overlayAt(base, overlay, baseWidth, baseHeight, -1, -1)
}

// GetAURCacheDir resolves the AUR build/clone directory based on the helper or override.
func GetAURCacheDir(c *Config) (string, error) {
	if c.Advanced.CacheDir != "" {
		return c.Advanced.CacheDir, nil
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user cache directory: %w", err)
	}

	if c.Commands.AurHelper == "yay" {
		return filepath.Join(cacheDir, "yay"), nil
	}

	// Default to paru's clone path
	return filepath.Join(cacheDir, "paru", "clone"), nil
}

// getKeyDisplay returns a string representing the primary key for a binding
func getKeyDisplay(b key.Binding) string {
	keys := b.Keys()
	if len(keys) == 0 {
		return ""
	}
	k := keys[0]
	// Special cases for display
	if strings.HasPrefix(k, "ctrl+") && len(k) == 6 {
		return "^" + string(k[5])
	}
	switch k {
	case "enter":
		return "enter"
	case "esc":
		return "esc"
	case "tab":
		return "tab"
	case " ":
		return "space"
	}
	return k
}

// renderKeyHint formats a label with a keybinding hint.
// If the key exists in the label, it highlights it in [].
// Otherwise, it prepends the key in [].
func renderKeyHint(label string, b key.Binding, style lipgloss.Style) string {
	k := getKeyDisplay(b)
	if k == "" {
		return style.Render(label)
	}

	// Create a segment style that preserves colors/bolding but strips layout AND padding
	// to prevent each segment from adding its own internal gaps.
	segStyle := style.UnsetWidth().UnsetAlign().Padding(0, 0)

	// Only try to embed if it's a single character
	if len(k) == 1 {
		idx := strings.Index(strings.ToLower(label), strings.ToLower(k))
		if idx != -1 {
			// Found the letter in the word
			before := label[:idx]
			letter := label[idx : idx+1]
			after := label[idx+1:]

			// We only render non-empty parts to avoid any potential zero-width rendering artifacts
			var result string
			if before != "" {
				result += segStyle.Render(before)
			}
			result += segStyle.Render("[" + letter + "]")
			if after != "" {
				result += segStyle.Render(after)
			}
			return result
		}
	}

	// Not found or not single char, prepend with colon.
	// The colon must be styled to match the background of the segments.
	return segStyle.Render("["+k+"]") + segStyle.Render(":") + segStyle.Render(label)
}

// maintainBackground replaces ANSI reset codes with a sequence that resets but then re-applies the given background color.
func maintainBackground(s string, bgColor lipgloss.Color) string {
	// Get the ANSI sequence for the background color
	bgStyle := lipgloss.NewStyle().Background(bgColor)
	bgSeq := bgStyle.Render(" ")
	// Extract just the escape sequence (everything before the space)
	idx := strings.Index(bgSeq, " ")
	if idx == -1 {
		return s
	}
	bgSeq = bgSeq[:idx]

	// Replace \x1b[0m with \x1b[0m + bgSeq
	return strings.ReplaceAll(s, "\x1b[0m", "\x1b[0m"+bgSeq)
}

// renderScrollbar generates a vertical scrollbar indicator
func renderScrollbar(total, offset, visibleHeight int, activeColor lipgloss.Color, reversed bool) string {
	if total <= visibleHeight || visibleHeight <= 0 {
		return ""
	}

	scrollbarHeight := visibleHeight
	trackStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("236"))
	thumbStyle := lipgloss.NewStyle().Foreground(activeColor)

	// Calculate thumb size and position
	thumbHeight := int(float64(visibleHeight) * (float64(visibleHeight) / float64(total)))
	if thumbHeight < 1 {
		thumbHeight = 1
	}

	maxOffset := total - visibleHeight
	if maxOffset <= 0 {
		return ""
	}

	// Clamp offset to ensure it stays within valid bounds [0, maxOffset]
	if offset > maxOffset {
		offset = maxOffset
	}
	if offset < 0 {
		offset = 0
	}

	thumbStart := int(float64(offset) / float64(maxOffset) * float64(visibleHeight-thumbHeight))

	// Invert for bottom-to-top menus
	if reversed {
		thumbStart = visibleHeight - thumbHeight - thumbStart
	}

	var scrollbar strings.Builder
	for i := 0; i < scrollbarHeight; i++ {
		if i >= thumbStart && i < thumbStart+thumbHeight {
			scrollbar.WriteString(thumbStyle.Render("┃"))
		} else {
			scrollbar.WriteString(trackStyle.Render("│"))
		}
		if i < scrollbarHeight-1 {
			scrollbar.WriteString("\n")
		}
	}

	return scrollbar.String()
}

// PaginatedListConfig defines parameters for rendering a scrollable, paginated list.
type PaginatedListConfig struct {
	TotalCount     int
	SelectedIndex  int
	ViewportHeight int
	ContentWidth   int
	ActiveColor    lipgloss.Color
	Reversed       bool
	RenderItem     func(index int, itemWidth int) string
}

// RenderPaginatedList renders a slice of items into a scrollable, paginated text block with optional scrollbar.
func RenderPaginatedList(cfg PaginatedListConfig) string {
	if cfg.ViewportHeight <= 0 || cfg.TotalCount <= 0 || cfg.RenderItem == nil {
		return ""
	}

	startIdx := 0
	if cfg.SelectedIndex >= cfg.ViewportHeight {
		startIdx = cfg.SelectedIndex - cfg.ViewportHeight + 1
	}
	endIdx := startIdx + cfg.ViewportHeight
	if endIdx > cfg.TotalCount {
		endIdx = cfg.TotalCount
	}

	itemWidth := cfg.ContentWidth
	if cfg.TotalCount > cfg.ViewportHeight {
		itemWidth = cfg.ContentWidth - 2
	}

	var lines []string
	for i := startIdx; i < endIdx; i++ {
		lines = append(lines, cfg.RenderItem(i, itemWidth))
	}

	var results strings.Builder
	if cfg.Reversed {
		for i := len(lines) - 1; i >= 0; i-- {
			results.WriteString(lines[i])
			if i > 0 {
				results.WriteString("\n")
			}
		}
	} else {
		for i := 0; i < len(lines); i++ {
			results.WriteString(lines[i])
			if i < len(lines)-1 {
				results.WriteString("\n")
			}
		}
	}

	resultsStr := results.String()
	if cfg.TotalCount > cfg.ViewportHeight {
		scrollbar := renderScrollbar(cfg.TotalCount, startIdx, cfg.ViewportHeight, cfg.ActiveColor, cfg.Reversed)
		resultsStr = lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(cfg.ContentWidth-2).Render(resultsStr),
			lipgloss.NewStyle().MarginLeft(1).Render(scrollbar))
	}

	return resultsStr
}

// SafeJoinVertical joins multiple sections vertically, ensuring the result is exactly width x height.
// It takes a header, a slice of panels (which will be truncated if they exceed available space),
// and a footer. The header and footer are prioritized over panels.
func SafeJoinVertical(width, height int, header string, panels []string, footer string) string {
	if height <= 0 {
		return ""
	}

	var allLines []string

	// Helper to split lines consistently with how Lipgloss calculates height
	split := func(s string) []string {
		if s == "" {
			return nil
		}
		// Trust the string's internal line count, just trim the trailing newline from Render()
		return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	}

	headerLines := split(header)
	headerH := len(headerLines)

	footerLines := split(footer)
	footerH := len(footerLines)

	// Available space for panels
	availableForPanels := height - headerH - footerH
	if availableForPanels < 0 {
		availableForPanels = 0
	}

	// Collect all panel lines
	var middleLines []string
	for _, p := range panels {
		middleLines = append(middleLines, split(p)...)
	}

	// Truncate panels ONLY if they exceed the available space
	if len(middleLines) > availableForPanels {
		middleLines = middleLines[:availableForPanels]
	}

	// Assemble everything
	allLines = append(allLines, headerLines...)
	allLines = append(allLines, middleLines...)

	// Pad with blanks if still short (BEFORE footer) to ensure footer is pinned to bottom
	for len(allLines) < height-footerH {
		allLines = append(allLines, strings.Repeat(" ", width))
	}

	// Add footer
	allLines = append(allLines, footerLines...)

	// Final safety truncation and width normalization
	if len(allLines) > height {
		allLines = allLines[:height]
	}

	for i := range allLines {
		w := lipgloss.Width(allLines[i])
		if w < width {
			allLines[i] += strings.Repeat(" ", width-w)
		} else if w > width {
			allLines[i] = truncateWithAnsi(allLines[i], width)
		}
	}

	return strings.Join(allLines, "\n")
}

// simplifyErrorMessage deduplicates repeating segments in an error message.
// It splits by common separators like ": " and keeps only unique parts to avoid
// extremely long repetitive messages from nested errors.
func simplifyErrorMessage(msg string) string {
	if msg == "" {
		return ""
	}

	// Split by ": "
	parts := strings.Split(msg, ": ")
	seen := make(map[string]bool)
	var uniqueParts []string

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !seen[part] {
			uniqueParts = append(uniqueParts, part)
			seen[part] = true
		}
	}

	return strings.Join(uniqueParts, ": ")
}
