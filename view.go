package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

func sourceStyle(source string) lipgloss.Style {
	if color, ok := sourceColors[source]; ok {
		return styleWithForeground(color)
	}
	return styleWithForeground(colorWhite)
}

// renderHelpText creates the help menu with the active mode highlighted
func (m *model) renderHelpText(activeColor lipgloss.Color) string {
	dimStyle := helpStyle
	activeStyle := styleBoldWithForeground(activeColor)

	entries := []struct {
		label  string
		key    key.Binding
		active bool
	}{
		{"search", m.keys.Search, false},
		{"mark", m.keys.Mark, false},
		{"dash", m.keys.DashboardMode, m.mode == modeDashboard},
		{"install", m.keys.InstallMode, m.mode == modeInstall},
		{"update", m.keys.UpdateMode, m.mode == modeUpdate || m.mode == modeUpdateSelective},
		{"remove", m.keys.RemoveMode, m.mode == modeRemove},
		{"settings", m.keys.Settings, m.mode == modeSettings},
		{"quit", m.keys.Quit, false},
	}

	separator := dimStyle.Render("  ")
	parts := make([]string, len(entries))
	for i, e := range entries {
		st := dimStyle
		if e.active {
			st = activeStyle
		}
		parts[i] = renderKeyHint(e.label, e.key, st)
	}

	return strings.Join(parts, separator)
}

func (m *model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	innerWidth := m.width
	innerHeight := m.height // Use full terminal height for base calculations

	// Determine effective mode for rendering background elements
	effectiveMode := m.mode
	if m.mode == modeSettings {
		effectiveMode = m.previousMode
	}

	// Apply configured border style
	baseBorderStyle = baseBorderStyle.Border(m.getBorderStyle())

	activeColor := modeColors[effectiveMode]
	if activeColor == "" {
		activeColor = defaultBorderColor
	}

	helpText := m.renderHelpText(activeColor)

	var content string

	if m.showConfirmation {
		content = m.renderConfirmationDialog(innerWidth, innerHeight, activeColor)
	} else if m.showErrorOverlay {
		content = m.renderErrorOverlay(innerWidth, innerHeight)
	} else if effectiveMode == modeCacheMenu {
		content = m.renderCacheMenu(helpText, innerWidth, innerHeight)
	} else if effectiveMode == modeCacheSelective {
		content = m.renderSelectiveCacheView(helpText, innerWidth, innerHeight, activeColor)
	} else if effectiveMode == modeDashboard {
		content = m.renderDashboard(helpText, innerWidth, innerHeight)
	} else if effectiveMode == modeUpdate {
		content = m.renderSimpleUpdateView(helpText, innerWidth, innerHeight, activeColor)
	} else if effectiveMode == modeUpdateSelective {
		content = m.renderUpdateSelectiveView(helpText, innerWidth, innerHeight, activeColor)
	} else {
		// Handle modeInstall and modeRemove
		footer := renderCenteredFooter(helpText, innerWidth)
		content = m.renderPackageListLayout(innerWidth, innerHeight, activeColor, "", footer)
	}

	// If settings are active, overlay them on top of the rendered content
	if m.mode == modeSettings {
		settingsOverlay := m.renderSettings(innerWidth, innerHeight)
		content = m.overlaySettings(content, settingsOverlay, innerWidth, innerHeight)
	}

	// If mirror overlay is active, overlay it on top
	if m.showMirrorOverlay {
		mirrorOverlay := m.renderMirrorOverlay(innerWidth, innerHeight)
		content = overlayOnBase(content, mirrorOverlay, innerWidth, innerHeight)
	}

	return stripNonSGREscapes(content)
}

// overlaySettings manually layers the settings menu on top of base content
func (m *model) overlaySettings(base, overlay string, width, height int) string {
	result := overlayOnBase(base, overlay, width, height)
	return SafeJoinVertical(width, height, "", []string{result}, "")
}

// renderUpdateSelectiveView renders the selective update overlay on top of the simple update view
func (m *model) renderUpdateSelectiveView(helpText string, innerWidth, innerHeight int, activeColor lipgloss.Color) string {
	overlayWidth := int(float64(innerWidth) * 0.75)
	overlayHeight := int(float64(innerHeight) * 0.75)

	if overlayWidth < 60 {
		overlayWidth = 60
	}
	if overlayHeight < 20 {
		overlayHeight = 20
	}

	// Ensure we don't exceed terminal dimensions
	if overlayWidth > innerWidth {
		overlayWidth = innerWidth
	}

	// In all terminals, we must leave at least one line for the footer
	maxOverlayHeight := innerHeight - 1
	if overlayHeight > maxOverlayHeight {
		overlayHeight = maxOverlayHeight
	}
	// Sanity check for extremely small terminals
	if overlayHeight < 5 && innerHeight >= 6 {
		overlayHeight = 5
	}

	warningSymbol := styleWithForeground(colorRed).Render("⚠")
	warningText := styleWithForeground(colorRed).Render(" Selective updates can break system dependencies")
	warningBox := lipgloss.NewStyle().
		Border(m.getBorderStyle()).
		BorderForeground(colorRed).
		Padding(0, 1).
		Render(warningSymbol + warningText)

	warningOverlay := lipgloss.PlaceHorizontal(overlayWidth, lipgloss.Center, warningBox)
	warningHeight := lipgloss.Height(warningOverlay)

	// Subtract space for the actual warning overlay height AND the JoinVertical separator
	overlayInnerHeight := overlayHeight - warningHeight - 1
	if overlayInnerHeight < 5 {
		overlayInnerHeight = 5
	}

	paneContent := m.renderVerticalSplitLayout(overlayWidth, overlayInnerHeight, activeColor)

	// Composite panels and warning
	// Ensure warning overlay has a fixed height that we accounted for
	warningBoxWrapped := lipgloss.NewStyle().
		Height(warningHeight).
		Render(strings.TrimSuffix(warningOverlay, "\n"))

	paneContent = lipgloss.JoinVertical(lipgloss.Left, strings.TrimSuffix(paneContent, "\n"), strings.TrimSuffix(warningBoxWrapped, "\n"))

	// Enforce strict rectangle bounds - this ensures we exactly match overlayHeight
	paneContent = lipgloss.Place(overlayWidth, overlayHeight, lipgloss.Center, lipgloss.Center, paneContent)

	bg := strings.TrimSuffix(m.renderSimpleUpdateView(helpText, innerWidth, innerHeight, activeColor), "\n")
	output := overlayOnBase(bg, paneContent, innerWidth, innerHeight)
	return SafeJoinVertical(innerWidth, innerHeight, "", []string{output}, "")
}

// renderPackageListItem formats a single package row for package list views.
func (m *model) renderPackageListItem(pkg Package, index int, maxWidth int, showVersion bool, showInstalledBadge bool) string {
	marker := " "
	if m.markedPackages[pkg.Name] {
		marker = "*"
	}
	prefix := " " + marker
	if index == m.selectedIndex {
		prefix = ">" + marker
	}

	sourceStyle := lipgloss.NewStyle()
	if color, ok := sourceColors[pkg.Source]; ok {
		sourceStyle = sourceStyle.Foreground(color)
	}

	var displayPkgStr string
	if indices, ok := m.matchIndices[index]; ok {
		displayPkgStr = highlightMatchesWithSourceColor(pkg, indices)
	} else {
		displayPkgStr = sourceStyle.Render(pkg.Source) + "/" + pkg.Name
	}

	var line string
	if showVersion {
		line = fmt.Sprintf("%s%s %s",
			prefix,
			displayPkgStr,
			styleWithForeground(colorMediumGray).Render(pkg.Version),
		)
		if showInstalledBadge && pkg.Installed {
			line += " " + installedBadge.Render("[installed]")
		}
	} else {
		line = fmt.Sprintf("%s%s", prefix, displayPkgStr)
	}

	if lipgloss.Width(line) > maxWidth {
		line = truncateWithAnsi(line, maxWidth-3) + "..."
	}

	if index == m.selectedIndex {
		line = selectedStyle.Render(line)
	}
	return line
}

// renderVerticalSplitLayout renders a side-by-side view (list on left, dash on right)
func (m *model) renderVerticalSplitLayout(innerWidth, innerHeight int, activeColor lipgloss.Color) string {
	borderStyle := baseBorderStyle.BorderForeground(activeColor)

	listWidth := int(float64(innerWidth) * 0.4)
	detailsWidth := innerWidth - listWidth

	if listWidth < 25 {
		listWidth = 25
		detailsWidth = innerWidth - listWidth
	}
	if detailsWidth < 20 {
		detailsWidth = 20
		listWidth = innerWidth - detailsWidth
	}

	// 1. Render List Side (Left)
	var pkgList []Package
	if m.mode == modeUpdateSelective {
		pkgList = m.filtered
	}

	// Calculate results height for the list
	// InnerHeight - 2 for borders - 2 for search input and separator - 1 for bottom padding
	resultsHeight := innerHeight - 5
	if resultsHeight < 1 {
		resultsHeight = 1
	}

	var resultsStr string
	if m.loading {
		resultsStr = "  Loading..."
	} else if len(pkgList) == 0 {
		resultsStr = "  No matches"
	} else {
		resultsStr = RenderPaginatedList(PaginatedListConfig{
			TotalCount:     len(pkgList),
			SelectedIndex:  m.selectedIndex,
			ViewportHeight: resultsHeight,
			ContentWidth:   listWidth - 4,
			ActiveColor:    activeColor,
			Reversed:       true,
			RenderItem: func(i int, itemWidth int) string {
				return m.renderPackageListItem(pkgList[i], i, itemWidth, false, false)
			},
		})
	}

	resultsContainer := lipgloss.NewStyle().
		Height(resultsHeight).
		Width(listWidth-4).
		Align(lipgloss.Left, lipgloss.Bottom).
		Render(strings.TrimSuffix(resultsStr, "\n"))

	listPanel := borderStyle.
		Width(listWidth-2).
		Height(innerHeight-2).
		Align(lipgloss.Left, lipgloss.Bottom).
		Render(strings.TrimSuffix(truncateHeight(lipgloss.JoinVertical(lipgloss.Left,
			resultsContainer,
			"", // spacing separator
			"", // bottom truncation gap
			strings.TrimSuffix(m.textInput.View(), "\n"),
		), innerHeight-2), "\n"))

	// 2. Render Details Side (Right)
	detailsContent := ""
	if m.loadingDetails {
		detailsContent = fmt.Sprintf("Loading details for %s...", m.detailsForPackage)
	} else if m.packageDetails != "" {
		detailsContent = sanitizeUntrusted(m.packageDetails)
	} else {
		detailsContent = "Select an update to see details"
	}

	detailsInnerHeight := innerHeight - 2
	detailsInnerWidth := detailsWidth - 6 // 2 chars padding on each side

	wrappedText := lipgloss.NewStyle().Width(detailsInnerWidth).Render(detailsContent)
	detailsLines := strings.Split(wrappedText, "\n")

	totalLines := len(detailsLines)
	if totalLines > detailsInnerHeight {
		maxScroll := totalLines - detailsInnerHeight
		m.maxDetailsScroll = maxScroll
		if m.detailsScrollOffset > maxScroll {
			m.detailsScrollOffset = maxScroll
		}
		detailsLines = detailsLines[m.detailsScrollOffset : m.detailsScrollOffset+detailsInnerHeight]
	} else {
		m.maxDetailsScroll = 0
		m.detailsScrollOffset = 0
	}
	detailsContent = strings.Join(detailsLines, "\n")

	detailsBox := lipgloss.NewStyle().
		Width(detailsWidth-2).
		Height(detailsInnerHeight).
		Padding(0, 2).
		Render(detailsContent)

	if len(m.markedPackages) > 0 {
		selectionPanel := m.renderSelectionBox(detailsWidth - 6)
		panelLines := strings.Split(selectionPanel, "\n")
		panelHeight := len(panelLines)
		panelWidth := lipgloss.Width(panelLines[0])

		// Overlay selectionPanel on bottom right of detailsBox
		startRow := detailsInnerHeight - panelHeight
		startCol := detailsInnerWidth + 2 - panelWidth
		if startRow < 0 {
			startRow = 0
		}
		if startCol < 0 {
			startCol = 0
		}

		detailsBox = overlayAt(detailsBox, selectionPanel, detailsWidth-2, detailsInnerHeight, startRow, startCol)
	}

	detailsPanel := borderStyle.
		Width(detailsWidth - 2).
		Height(innerHeight - 2).
		Render(strings.TrimSuffix(truncateHeight(detailsBox, innerHeight-2), "\n"))

	return lipgloss.JoinHorizontal(lipgloss.Top, strings.TrimSuffix(listPanel, "\n"), strings.TrimSuffix(detailsPanel, "\n"))
}

// renderSelectionBox renders a box containing currently marked packages
func (m *model) renderSelectionBox(maxWidth int) string {
	if len(m.markedPackages) == 0 {
		return ""
	}

	var pkgNames []string
	for name := range m.markedPackages {
		pkgNames = append(pkgNames, name)
	}
	sort.Strings(pkgNames)

	maxVisible := 8
	startIdx := m.selectionScrollOffset
	if m.selectionPanelFocused {
		if m.selectionPanelIndex >= startIdx+maxVisible {
			startIdx = m.selectionPanelIndex - maxVisible + 1
		} else if m.selectionPanelIndex < startIdx {
			startIdx = m.selectionPanelIndex
		}
	}
	m.selectionScrollOffset = startIdx

	endIdx := startIdx + maxVisible
	if endIdx > len(pkgNames) {
		endIdx = len(pkgNames)
	}

	// Determine dynamic width
	titleStr := fmt.Sprintf(" Selected (%d) [*] ", len(pkgNames))
	maxContentWidth := lipgloss.Width(titleStr)

	for i := startIdx; i < endIdx; i++ {
		nameWidth := lipgloss.Width(pkgNames[i]) + 4 // +2 for prefix/marker, +2 for inner padding
		if nameWidth > maxContentWidth {
			maxContentWidth = nameWidth
		}
	}

	panelWidth := 15
	if maxContentWidth+2 > 20 { // +2 for borders
		panelWidth = 25
	} else if maxContentWidth+2 > 15 {
		panelWidth = 20
	}

	if panelWidth > maxWidth {
		panelWidth = maxWidth
	}

	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMagenta).
		Padding(0, 1)

	if m.selectionPanelFocused {
		panelStyle = panelStyle.BorderForeground(colorYellow)
	}

	titleText := styleBoldWithForeground(colorMagenta).Render(titleStr)

	itemStyle := styleWithForeground(colorWhite)
	selectedItemStyle := styleBoldWithForeground(colorYellow)

	var listBuilder strings.Builder
	nameMaxWidth := panelWidth - 6 // 2 for border, 2 for padding, 2 for prefix

	for i := startIdx; i < endIdx; i++ {
		name := pkgNames[i]
		displayName := name
		if lipgloss.Width(displayName) > nameMaxWidth {
			displayName = truncateWithAnsi(displayName, nameMaxWidth-3) + "..."
		}

		if i > startIdx {
			listBuilder.WriteString("\n")
		}
		if m.selectionPanelFocused && i == m.selectionPanelIndex {
			listBuilder.WriteString(selectedItemStyle.Render("> " + displayName))
		} else {
			listBuilder.WriteString(itemStyle.Render("  " + displayName))
		}
	}

	listStr := listBuilder.String()
	if len(pkgNames) > (endIdx - startIdx) {
		// Add scrollbar for selection box (Top-down)
		scrollbar := renderScrollbar(len(pkgNames), startIdx, (endIdx - startIdx), colorMagenta, false)
		listStr = lipgloss.JoinHorizontal(lipgloss.Top,
			styleWithWidth(panelWidth-4).Render(listStr),
			lipgloss.NewStyle().MarginLeft(1).Render(scrollbar))
	}

	return panelStyle.Width(panelWidth).Render(titleText + "\n" + listStr)
}

// renderPackageListLayout renders the standard split-pane list view for install, remove, and selective update
// renderRepoSummary creates a color-coded summary of packages by repository
func (m *model) renderRepoSummary(pkgList []Package) string {
	if len(pkgList) == 0 {
		return ""
	}

	counts := make(map[string]int)
	for _, p := range pkgList {
		counts[p.Source]++
	}

	// Ordered repos for consistent display
	standardRepos := []string{"core", "extra", "multilib", "aur"}
	var parts []string
	for _, r := range standardRepos {
		if c, ok := counts[r]; ok && c > 0 {
			style := sourceStyle(r)
			parts = append(parts, fmt.Sprintf("%d %s", c, style.Render(r)))
		}
	}

	// Add any "other" repos
	var others []string
	for r := range counts {
		isStandard := false
		for _, sr := range standardRepos {
			if r == sr {
				isStandard = true
				break
			}
		}
		if !isStandard {
			others = append(others, r)
		}
	}
	sort.Strings(others)
	for _, r := range others {
		style := sourceStyle(r)
		parts = append(parts, fmt.Sprintf("%d %s", counts[r], style.Render(r)))
	}

	return strings.Join(parts, ", ")
}

func (m *model) renderPackageListLayout(innerWidth, innerHeight int, activeColor lipgloss.Color, header, footer string) string {
	borderStyle := baseBorderStyle.BorderForeground(activeColor)

	// Use manual height calculation to avoid trailing zero-width lines issues
	calcHeight := func(s string) int {
		if s == "" {
			return 0
		}
		lines := strings.Split(s, "\n")
		// Trust the string's internal line count, just trim the trailing newline from Render()
		// if lipgloss.Width(lines[len(lines)-1]) == 0 {
		// 	return len(lines) - 1
		// }
		return len(lines)
	}

	headerHeight := calcHeight(header)
	footerHeight := calcHeight(footer)

	availableHeight := innerHeight - headerHeight - footerHeight
	if availableHeight < 6 {
		availableHeight = 6
	}

	targetDetailsPanelHeight := availableHeight / 2
	targetBottomPanelHeight := availableHeight - targetDetailsPanelHeight

	detailsInnerHeight := targetDetailsPanelHeight - 2
	bottomInnerHeight := targetBottomPanelHeight - 2

	if bottomInnerHeight < 5 {
		bottomInnerHeight = 5
		targetBottomPanelHeight = bottomInnerHeight + 2
		targetDetailsPanelHeight = availableHeight - targetBottomPanelHeight
		detailsInnerHeight = targetDetailsPanelHeight - 2
		if detailsInnerHeight < 1 {
			detailsInnerHeight = 1
		}
	}

	resultsHeight := bottomInnerHeight - 4
	if resultsHeight < 1 {
		resultsHeight = 1
	}

	detailsContent := ""
	if m.mode == modeUpdateSelective {
		if m.loadingDetails {
			detailsContent = fmt.Sprintf("Loading details for %s...", m.detailsForPackage)
		} else if m.packageDetails != "" {
			detailsContent = sanitizeUntrusted(m.packageDetails)
		} else {
			detailsContent = "Select an update to see details"
		}
	} else if m.mode == modeInstall {
		if m.loadingDetails {
			detailsContent = fmt.Sprintf("Loading details for %s...", m.detailsForPackage)
		} else if m.packageDetails != "" {
			detailsContent = sanitizeUntrusted(m.packageDetails)
		} else {
			if m.textInput.Value() == "" {
				detailsContent = "Search for a package to see details"
			} else {
				detailsContent = "Select a package to see details"
			}
		}
	} else {
		if m.loadingDetails {
			detailsContent = fmt.Sprintf("Loading details for %s...", m.detailsForPackage)
		} else if m.packageDetails != "" {
			detailsContent = sanitizeUntrusted(m.packageDetails)
		} else {
			detailsContent = "Select a package to see details"
		}
	}

	// InnerWidth is the total terminal width
	// detailsPanel Total Width = innerWidth
	// detailsPanel Inner Width = innerWidth - 2
	// detailsBox Total Width (including padding) = innerWidth - 4 (1 char margin on each side)
	// detailsBox Content Width = innerWidth - 6
	contentWidth := innerWidth - 6
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Render the text with a width limit first so Lipgloss wraps it
	wrappedText := lipgloss.NewStyle().Width(contentWidth).Render(detailsContent)
	detailsLines := strings.Split(wrappedText, "\n")

	totalLines := len(detailsLines)
	if totalLines > detailsInnerHeight {
		maxScroll := totalLines - detailsInnerHeight
		m.maxDetailsScroll = maxScroll
		// Clamp offset
		if m.detailsScrollOffset > maxScroll {
			m.detailsScrollOffset = maxScroll
		} else if m.detailsScrollOffset < 0 {
			m.detailsScrollOffset = 0
		}
		detailsLines = detailsLines[m.detailsScrollOffset : m.detailsScrollOffset+detailsInnerHeight]
	} else {
		m.maxDetailsScroll = 0
		m.detailsScrollOffset = 0
	}
	detailsContent = strings.Join(detailsLines, "\n")

	detailsBox := lipgloss.NewStyle().
		Width(innerWidth-2).
		Padding(0, 2).
		Render(truncateHeight(detailsContent, detailsInnerHeight))

	detailsPanel := borderStyle.
		Width(innerWidth - 2).
		Height(max(0, targetDetailsPanelHeight-2)).
		Render(truncateHeight(detailsBox, max(0, targetDetailsPanelHeight-2)))

	inputLine := ""
	statusLine := ""

	// Determine what mode's input line to show
	displayMode := m.mode
	if m.mode == modeSettings {
		displayMode = m.previousMode
	}

	// Build results list
	var pkgList []Package
	if displayMode == modeInstall {
		pkgList = m.filtered
	} else if displayMode == modeRemove {
		pkgList = m.filteredInstalled
	} else if displayMode == modeUpdateSelective {
		pkgList = m.filtered
	}

	if displayMode == modeInstall || displayMode == modeRemove || displayMode == modeUpdateSelective {
		inputLine = m.textInput.View()

		// Add repository filter hints to the right side of the search bar in install mode
		if displayMode == modeInstall {
			repoFilters, _ := parseRepoFilter(m.textInput.Value())

			dimStyle := styleWithForeground(colorLightGray)

			var hintParts []string
			filters := []struct {
				char rune
				repo string
			}{
				{'c', "core"},
				{'e', "extra"},
				{'m', "multilib"},
				{'a', "aur"},
			}

			for _, f := range filters {
				text := string(f.char) + ":"
				if repoFilters[f.repo] {
					color, ok := sourceColors[f.repo]
					if !ok {
						color = colorWhite
					}
					hintParts = append(hintParts, styleBoldWithForeground(color).Render(text))
				} else {
					hintParts = append(hintParts, dimStyle.Render(text))
				}
			}

			hints := strings.Join(hintParts, " ")

			// Total width for content inside the panel is innerWidth-2
			availableWidth := innerWidth - 2

			// Hints are 11 chars + 2 padding on right = 13 chars
			// We give it a bit more for safety or flexibility
			hintsPaneWidth := lipgloss.Width(hints) + 2
			inputPaneWidth := availableWidth - hintsPaneWidth

			if inputPaneWidth < 20 {
				inputPaneWidth = 20
			}

			hintsView := lipgloss.NewStyle().
				Width(hintsPaneWidth).
				Align(lipgloss.Right).
				PaddingRight(2).
				Render(hints)

			inputLine = lipgloss.JoinHorizontal(lipgloss.Bottom,
				lipgloss.NewStyle().Width(inputPaneWidth).Render(m.textInput.View()),
				hintsView,
			)
		}

		if m.searchStatus != "" {
			style := styleItalicDim()

			if m.searchError {
				style = style.Foreground(currentTheme.ErrorColor)
			}

			renderedStatus := style.Render(m.searchStatus)

			if m.searchingAUR {
				spinnerStr := m.spinner.View()
				sw := lipgloss.Width(spinnerStr)
				if sw >= 2 {
					statusLine = spinnerStr + renderedStatus
				} else {
					statusLine = spinnerStr + " " + renderedStatus
				}
			} else {
				statusLine = "  " + renderedStatus
			}
		}

		// Add repository summary to the right of the status line
		repoSummary := m.renderRepoSummary(pkgList)
		if repoSummary != "" {
			if statusLine == "" {
				statusLine = "  "
			}
			summaryWidth := lipgloss.Width(repoSummary)
			statusWidth := lipgloss.Width(statusLine)
			// Status line is inside a panel with width innerWidth-2
			// Padding calculation to push repoSummary to the right
			padding := (innerWidth - 2) - statusWidth - summaryWidth - 2
			if padding > 0 {
				statusLine = statusLine + strings.Repeat(" ", padding) + repoSummary
			} else {
				statusLine = statusLine + " " + repoSummary
			}
		}
	} else {
		inputLine = statusStyle.Render(m.statusMessage)
	}

	// Calculate resultsHeight precisely to fill available space
	// Overhead: separator(1) + inputLine(1)
	overhead := 2
	if statusLine != "" {
		overhead++
	}
	resultsHeight = bottomInnerHeight - overhead
	if resultsHeight < 1 {
		resultsHeight = 1
	}

	// Build results list
	var resultsStr string

	if m.loading {
		resultsStr = "  Loading..."
	} else if m.mode == modeUpdateSelective && len(pkgList) == 0 && !m.loading {
		resultsStr = "  " + m.statusMessage
	} else if len(pkgList) == 0 {
		resultsStr = "  No packages to display"
	} else {
		resultsStr = RenderPaginatedList(PaginatedListConfig{
			TotalCount:     len(pkgList),
			SelectedIndex:  m.selectedIndex,
			ViewportHeight: resultsHeight,
			ContentWidth:   innerWidth - 4,
			ActiveColor:    activeColor,
			Reversed:       true,
			RenderItem: func(i int, itemWidth int) string {
				return m.renderPackageListItem(pkgList[i], i, itemWidth, true, m.mode == modeInstall)
			},
		})
	}

	resultsBox := lipgloss.NewStyle().
		Width(innerWidth-4).
		Height(resultsHeight).
		Align(lipgloss.Left, lipgloss.Bottom).
		Render(resultsStr)

	bottomParts := []string{
		resultsBox,
		styleWithForeground(colorDimGray).Render(strings.Repeat("─", innerWidth-2)),
		inputLine,
	}
	if statusLine != "" {
		bottomParts = append(bottomParts, statusLine)
	}
	bottomContent := strings.Join(bottomParts, "\n")

	bottomPanel := borderStyle.
		Width(innerWidth-2).
		Height(max(0, targetBottomPanelHeight-2)).
		Align(lipgloss.Left, lipgloss.Bottom).
		Render(truncateHeight(bottomContent, max(0, targetBottomPanelHeight-2)))

	content := SafeJoinVertical(innerWidth, innerHeight, header, []string{detailsPanel, bottomPanel}, footer)

	if len(m.markedPackages) > 0 {
		content = m.overlaySelectionsPanel(content, innerWidth, headerHeight)
	}

	return content
}

// overlaySelectionsPanel renders a selection panel on the bottom right of the screen
func (m *model) overlaySelectionsPanel(content string, innerWidth int, headerHeight int) string {
	panel := m.renderSelectionBox(32)
	panelLines := strings.Split(panel, "\n")
	panelWidth := lipgloss.Width(panelLines[0])

	startCol := innerWidth - panelWidth
	if startCol < 0 {
		startCol = 0
	}

	lines := strings.Split(content, "\n")
	return overlayAt(content, panel, innerWidth, len(lines), 0, startCol)
}

// resolvePackages resolves a list of package names into Package structs using getPackageByName with a fallback.
func (m *model) resolvePackages(names []string) []Package {
	packages := make([]Package, 0, len(names))
	for _, name := range names {
		cleanName := sanitizeUntrusted(name)
		if pkg := m.getPackageByName(name); pkg != nil {
			cp := *pkg
			cp.Name = cleanName
			packages = append(packages, cp)
		} else {
			packages = append(packages, Package{Name: cleanName})
		}
	}
	return packages
}

// renderCacheBreakdown formats the pacman and AUR helper cache freed breakdown lines.
func (m *model) renderCacheBreakdown(contentWidth int, pacmanEst, aurEst string) []string {
	if pacmanEst == "" {
		pacmanEst = "calculating..."
	}
	if aurEst == "" {
		aurEst = "calculating..."
	}

	valStyle := lipgloss.NewStyle().Foreground(currentTheme.TextColor)
	pacmanLabel := sourceStyle("core").Render("  pacman:")
	helperLabel := m.config.Commands.AurHelper + ":"
	if len(helperLabel) < 7 {
		helperLabel += strings.Repeat(" ", 7-len(helperLabel))
	}
	aurLabel := sourceStyle("aur").Render("  " + helperLabel)

	return []string{
		lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, fmt.Sprintf("%s %s", pacmanLabel, valStyle.Render(pacmanEst))),
		lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, fmt.Sprintf("%s %s", aurLabel, valStyle.Render(aurEst))),
	}
}

// renderConfirmationDialog renders a centered confirmation dialog for install/remove/update
func (m *model) renderConfirmationDialog(innerWidth, innerHeight int, activeColor lipgloss.Color) string {
	var title string
	var packages []Package
	var actionDesc string
	var simpleConfirm bool

	switch m.confirmType {
	case confirmInstall:
		title = "📦 Confirm Installation"
		actionDesc = "installed"
		packages = m.resolvePackages(m.confirmPackages)
	case confirmRemove:
		title = "🗑 Confirm Removal"
		actionDesc = "removed"
		packages = m.resolvePackages(m.confirmPackages)
	case confirmUpdate:
		title = "🔄 Confirm System Update"
		actionDesc = "updated"
		packages = m.pendingUpdates
	case confirmSelectiveUpdate:
		title = "🔄 Confirm Selective Update"
		actionDesc = "updated"
		packages = m.resolvePackages(m.confirmPackages)
	case confirmRemoveOrphans:
		title = "🧹 Confirm Orphan Removal"
		actionDesc = "removed"
		for _, name := range m.confirmPackages {
			packages = append(packages, Package{Name: sanitizeUntrusted(name)})
		}
	case confirmCleanKeep3, confirmCleanKeep1, confirmCleanNuke:
		title = "🧹 Confirm Cache Cleaning"
		actionDesc = "cleaned"
		simpleConfirm = true
	case confirmCleanRemoved:
		title = "🧹 Confirm Orphaned Cache Clean"
		actionDesc = "cleaned"
		if len(m.dashboard.RemovedPacmanCache) > 0 {
			for _, p := range m.dashboard.RemovedPacmanCache {
				packages = append(packages, Package{Name: sanitizeUntrusted(p.Name), Size: p.Size})
			}
		}
		if len(m.dashboard.RemovedAurCache) > 0 {
			for _, p := range m.dashboard.RemovedAurCache {
				packages = append(packages, Package{Name: sanitizeUntrusted(p.Name), Size: p.Size})
			}
		}
	case confirmCleanSelective:
		title = "🧹 Confirm Selective Clean"
		actionDesc = "removed from cache"
		for _, name := range m.confirmPackages {
			packages = append(packages, Package{Name: sanitizeUntrusted(name), Size: ""}) // Size filled later if available
		}
	}

	dialogWidth := innerWidth - 10
	if dialogWidth < 60 {
		dialogWidth = 60
	}
	if dialogWidth > 90 {
		dialogWidth = 90
	}

	activeBorderColor := activeColor
	switch m.confirmType {
	case confirmInstall:
		activeBorderColor = currentTheme.ConfirmInstall
	case confirmRemove:
		activeBorderColor = currentTheme.ConfirmRemove
	case confirmCleanRemoved:
		activeBorderColor = currentTheme.ConfirmClean
	case confirmCleanNuke:
		activeBorderColor = currentTheme.ConfirmNuke
	case confirmCleanSelective:
		activeBorderColor = currentTheme.ConfirmSelective
	}

	dialogBorderStyle := lipgloss.NewStyle().
		Border(m.getBorderStyle()).
		BorderForeground(activeBorderColor).
		Align(lipgloss.Left)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(activeBorderColor)
	packageNameStyle := lipgloss.NewStyle().Foreground(currentTheme.TextColor).Bold(true)
	packageVersionStyle := lipgloss.NewStyle().Foreground(currentTheme.DimText)
	countStyle := lipgloss.NewStyle().Foreground(currentTheme.WarningColor).Bold(true)
	promptStyle := lipgloss.NewStyle().Foreground(currentTheme.TextColor).MarginTop(1)
	keyStyle := lipgloss.NewStyle().Foreground(activeBorderColor).Bold(true)
	scrollHintStyle := lipgloss.NewStyle().Foreground(currentTheme.DimText)

	var dialogContent []string
	contentWidth := dialogWidth - 4

	// Title
	dialogContent = append(dialogContent, lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, titleStyle.Render(title)))
	dialogContent = append(dialogContent, "")

	// Warning/Description
	descStyle := lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center)
	if simpleConfirm {
		if m.confirmType == confirmCleanNuke {
			warningStyle := lipgloss.NewStyle().Foreground(currentTheme.ErrorColor).Bold(true).Width(contentWidth).Align(lipgloss.Center)
			dialogContent = append(dialogContent, warningStyle.Render("WARNING: This will completely empty the package cache."))
			dialogContent = append(dialogContent, "")
		} else {
			if m.confirmType == confirmCleanKeep3 {
				dialogContent = append(dialogContent, descStyle.Render("This will remove all but the 3 most recent cached versions of packages."))
			} else {
				dialogContent = append(dialogContent, descStyle.Render("This will aggressively remove all but the currently installed cached versions."))
			}
			dialogContent = append(dialogContent, "")
		}

		if m.confirmType != confirmCleanNuke {
			labelStyle := lipgloss.NewStyle().Width(8).Foreground(currentTheme.DimText)
			dialogContent = append(dialogContent, packageNameStyle.Render("System Cache:"))
			dialogContent = append(dialogContent, fmt.Sprintf("  %s %s", labelStyle.Render("Path:"), scrollHintStyle.Render(m.dashboard.PacmanCachePath)))
			dialogContent = append(dialogContent, packageNameStyle.Render("User Cache:"))
			dialogContent = append(dialogContent, fmt.Sprintf("  %s %s", labelStyle.Render("Path:"), scrollHintStyle.Render(m.dashboard.AurCachePath)))
			dialogContent = append(dialogContent, "")
		}

		breakdownHeaderStyle := lipgloss.NewStyle().Foreground(currentTheme.DimText).Bold(true).Width(contentWidth).Align(lipgloss.Center)
		dialogContent = append(dialogContent, breakdownHeaderStyle.Render("Breakdown:"))

		pacmanEstimate := m.dashboard.CacheFreedPacman[m.confirmType]
		aurEstimate := m.dashboard.CacheFreedAur[m.confirmType]
		if m.confirmType == confirmCleanNuke {
			pacmanEstimate = m.dashboard.PacmanCacheSize
			aurEstimate = m.dashboard.AurCacheSize
		}
		dialogContent = append(dialogContent, m.renderCacheBreakdown(contentWidth, pacmanEstimate, aurEstimate)...)
		dialogContent = append(dialogContent, "")

		estStyle := lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center)
		estimate := m.dashboard.CacheFreedEstimates[m.confirmType]
		if estimate == "" {
			estimate = "calculating..."
		}
		dialogContent = append(dialogContent, estStyle.Render(fmt.Sprintf("Estimated space to be freed: %s", lipgloss.NewStyle().Bold(true).Foreground(currentTheme.WarningColor).Render(estimate))))
	} else {
		// List-based Confirmations
		listTitleStyle := lipgloss.NewStyle().Width(contentWidth).Align(lipgloss.Center)
		if m.confirmType == confirmUpdate {
			dialogContent = append(dialogContent, listTitleStyle.Render(fmt.Sprintf("The following %s updates are available:", countStyle.Render(fmt.Sprintf("%d", len(packages))))))
		} else if m.confirmType == confirmCleanRemoved {
			dialogContent = append(dialogContent, listTitleStyle.Render("This will remove cached packages that are no longer installed."))
		} else {
			dialogContent = append(dialogContent, listTitleStyle.Render(fmt.Sprintf("The following %s packages will be %s:", countStyle.Render(fmt.Sprintf("%d", len(packages))), actionDesc)))
		}
		dialogContent = append(dialogContent, "")

		maxVisible := 10
		startIdx := m.confirmScrollOffset
		endIdx := startIdx + maxVisible
		if endIdx > len(packages) {
			endIdx = len(packages)
		}
		m.maxConfirmScroll = len(packages) - maxVisible
		if m.maxConfirmScroll < 0 {
			m.maxConfirmScroll = 0
		}

		// Calculate max repo width for update alignment
		maxRepoWidth := 0
		if m.confirmType == confirmUpdate {
			for _, pkg := range packages {
				w := len(pkg.Source) + 2 // [source]
				if w > maxRepoWidth {
					maxRepoWidth = w
				}
			}
		}

		for i := startIdx; i < endIdx; i++ {
			pkg := packages[i]
			var line string
			if m.confirmType == confirmUpdate {
				sourceName := fmt.Sprintf("[%s]", pkg.Source)
				sourceBadge := sourceStyle(pkg.Source).Render(sourceName)
				paddedSourceBadge := sourceBadge + strings.Repeat(" ", max(0, maxRepoWidth-len(sourceName)))
				line = fmt.Sprintf("  • %s %s %s", paddedSourceBadge, packageNameStyle.Render(pkg.Name), packageVersionStyle.Render(pkg.Version))
			} else if pkg.Version == "HEADER" {
				line = lipgloss.NewStyle().Bold(true).Foreground(currentTheme.TitleColor).Render(pkg.Name)
			} else if m.confirmType == confirmCleanRemoved || m.confirmType == confirmCleanSelective {
				namePart := "  • " + packageNameStyle.Render(pkg.Name)
				sizePart := lipgloss.NewStyle().Foreground(currentTheme.DimText).Render(pkg.Size)
				spacing := (dialogWidth - 10) - lipgloss.Width(namePart) - lipgloss.Width(sizePart)
				if spacing < 1 {
					spacing = 1
				}
				line = namePart + strings.Repeat(" ", spacing) + sizePart
			} else {
				line = fmt.Sprintf("  • %s", packageNameStyle.Render(pkg.Name))
			}
			dialogContent = append(dialogContent, line)
		}

		if len(packages) > maxVisible {
			dialogContent = append(dialogContent, "")
			dialogContent = append(dialogContent, lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, scrollHintStyle.Render("  Use [↑/↓] or [j/k] to scroll")))
		}

		if m.confirmType == confirmCleanRemoved {
			dialogContent = append(dialogContent, "")
			breakdownHeaderStyle := lipgloss.NewStyle().Foreground(currentTheme.DimText).Bold(true).Width(contentWidth).Align(lipgloss.Center)
			dialogContent = append(dialogContent, breakdownHeaderStyle.Render("Breakdown:"))
			pacmanEst := m.dashboard.CacheFreedPacman[m.confirmType]
			aurEst := m.dashboard.CacheFreedAur[m.confirmType]
			dialogContent = append(dialogContent, m.renderCacheBreakdown(contentWidth, pacmanEst, aurEst)...)
		}

		if m.confirmType == confirmCleanRemoved || m.confirmType == confirmCleanSelective {
			dialogContent = append(dialogContent, "")
			est := m.dashboard.CacheFreedEstimates[m.confirmType]
			if m.confirmType == confirmCleanSelective {
				est = formatBytes(m.cacheToFree)
			}
			if est == "" {
				est = "calculating..."
			}
			dialogContent = append(dialogContent, lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, fmt.Sprintf("Estimated space to be freed: %s", lipgloss.NewStyle().Bold(true).Foreground(currentTheme.WarningColor).Render(est))))
		}
	}

	dialogContent = append(dialogContent, "")
	dialogContent = append(dialogContent, lipgloss.PlaceHorizontal(contentWidth, lipgloss.Center, promptStyle.Render(fmt.Sprintf("Proceed? %s  %s",
		renderKeyHint("yes", m.keys.Confirm, keyStyle),
		renderKeyHint("no", m.keys.Cancel, keyStyle)))))

	dialog := dialogBorderStyle.Width(dialogWidth).Render(strings.Join(dialogContent, "\n"))

	return SafeJoinVertical(innerWidth, innerHeight, "", []string{lipgloss.Place(innerWidth, innerHeight, lipgloss.Center, lipgloss.Center, dialog)}, "")
}

// renderCenteredWrappedText wraps text to width, trims whitespace from each non-empty line,
// and centers each line within width, returning the joined vertical string.
func renderCenteredWrappedText(text string, width int) string {
	if text == "" || width <= 0 {
		return ""
	}
	wrapped := lipgloss.NewStyle().Width(width).Render(text)
	var lines []string
	lineStyle := lipgloss.NewStyle().Width(width).Align(lipgloss.Center)
	for _, line := range strings.Split(wrapped, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			lines = append(lines, lineStyle.Render(trimmed))
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return lipgloss.JoinVertical(lipgloss.Center, lines...)
}

// renderErrorOverlay renders a centered error overlay dialog
func (m *model) renderErrorOverlay(innerWidth, innerHeight int) string {
	dialogWidth := innerWidth - 20
	if dialogWidth < 50 {
		dialogWidth = 50
	}
	if dialogWidth > 100 {
		dialogWidth = 100
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)

	hintStyle := lipgloss.NewStyle().
		Width(dialogWidth - 4).
		Align(lipgloss.Center)

	dialogBorderStyle := lipgloss.NewStyle().
		Border(m.getBorderStyle()).
		BorderForeground(currentTheme.ErrorColor).
		Align(lipgloss.Center)

	// Build content pieces
	title := titleStyle.Render("⚠  " + m.errorTitle + "  ⚠")

	// Ensure each line of the error message is individually centered
	msgWidth := dialogWidth - 4
	message := renderCenteredWrappedText(sanitizeUntrusted(m.errorMessage), msgWidth)

	var details string
	if m.errorDetails != "" {
		if renderedDetails := renderCenteredWrappedText(sanitizeUntrusted(m.errorDetails), msgWidth); renderedDetails != "" {
			details = "\n" + renderedDetails
		}
	}

	hint := "\n" + hintStyle.Render(fmt.Sprintf("Press %s, %s, or %s to dismiss",
		renderKeyHint("esc", m.keys.Cancel, hintStyle),
		renderKeyHint("enter", m.keys.Confirm, hintStyle),
		renderKeyHint("quit", m.keys.Quit, hintStyle)))

	dialogContent := lipgloss.JoinVertical(lipgloss.Center, title, "", message, details, hint)
	dialog := dialogBorderStyle.Width(dialogWidth).Render(dialogContent)

	return SafeJoinVertical(innerWidth, innerHeight, "", []string{lipgloss.Place(innerWidth, innerHeight, lipgloss.Center, lipgloss.Center, dialog)}, "")
}

// renderMirrorOverlay renders the mirror configuration overlay
func (m *model) renderMirrorOverlay(innerWidth, innerHeight int) string {
	overlayWidth := 70
	if overlayWidth > innerWidth-4 {
		overlayWidth = innerWidth - 4
	}

	activeColor := currentTheme.DialogBorder
	dimPurple := currentTheme.DimText
	dimStyle := styleWithForeground(colorLightGray)
	activeStyle := styleBoldWithForeground(activeColor)
	labelStyle := lipgloss.NewStyle().Width(12).Foreground(currentTheme.TextColor).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(colorDimGray).Italic(true)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(activeColor).Width(overlayWidth - 4).Align(lipgloss.Center)

	var content []string

	// Title
	content = append(content, titleStyle.Render("Mirror Configuration"))
	content = append(content, "")

	// Reflector check
	if !m.isReflectorInstalled() {
		errorStyle := lipgloss.NewStyle().Foreground(colorRed).Width(overlayWidth - 4).Align(lipgloss.Center)
		content = append(content, errorStyle.Render("reflector is not installed!"))
		content = append(content, "")
		content = append(content, dimStyle.Render("Install with: sudo pacman -S reflector"))
		content = append(content, "")
		content = append(content, dimStyle.Render("Press [esc] to close"))
	} else {
		// Description of what will happen
		infoStyle := lipgloss.NewStyle().Foreground(colorLightGray).Width(overlayWidth - 4).Align(lipgloss.Center)
		content = append(content, infoStyle.Render("Update pacman mirrorlist using reflector"))
		content = append(content, "")

		// Sort By option
		sortIsActive := m.mirrorSelectedItem == mirrorItemSortBy
		sortStyle := dimStyle
		if sortIsActive {
			sortStyle = activeStyle
		}
		sortValue := MirrorSortOptions[m.mirrorConfig.SortBy].Name
		sortDesc := MirrorSortOptions[m.mirrorConfig.SortBy].Description
		sortLine := fmt.Sprintf("%s %s %s %s",
			labelStyle.Render("Sort by:"),
			dimStyle.Render("<"),
			sortStyle.Render(sortValue),
			dimStyle.Render(">"))
		content = append(content, sortLine)
		content = append(content, "             "+descStyle.Render(sortDesc))
		content = append(content, "")

		// Country option
		countryIsActive := m.mirrorSelectedItem == mirrorItemCountry
		countryStyle := dimStyle
		if countryIsActive {
			countryStyle = activeStyle
		}
		countryValue := MirrorCountries[m.mirrorConfig.CountryIndex].Name
		countryDesc := "Filter mirrors by geographic location"
		if m.mirrorConfig.CountryIndex == 0 {
			countryDesc = "Use mirrors from all countries"
		}
		countryLine := fmt.Sprintf("%s %s %s %s",
			labelStyle.Render("Country:"),
			dimStyle.Render("<"),
			countryStyle.Render(countryValue),
			dimStyle.Render(">"))
		content = append(content, countryLine)
		content = append(content, "             "+descStyle.Render(countryDesc))
		content = append(content, "")

		// Latest count option
		latestIsActive := m.mirrorSelectedItem == mirrorItemLatest
		latestStyle := dimStyle
		if latestIsActive {
			latestStyle = activeStyle
		}
		latestDesc := fmt.Sprintf("Use the %d most recently synchronized mirrors", m.mirrorConfig.Latest)
		latestLine := fmt.Sprintf("%s %s %s %s",
			labelStyle.Render("Latest:"),
			dimStyle.Render("<"),
			latestStyle.Render(fmt.Sprintf("%d", m.mirrorConfig.Latest)),
			dimStyle.Render(">"))
		content = append(content, latestLine)
		content = append(content, "             "+descStyle.Render(latestDesc))
		content = append(content, "")

		// Protocol option
		protocolIsActive := m.mirrorSelectedItem == mirrorItemProtocol
		protocolStyle := dimStyle
		if protocolIsActive {
			protocolStyle = activeStyle
		}
		protocolValue := MirrorProtocols[m.mirrorConfig.Protocol].Name
		protocolDesc := "Connection protocol for mirror access"
		if m.mirrorConfig.Protocol == 0 {
			protocolDesc = "Use only secure HTTPS connections"
		} else if m.mirrorConfig.Protocol == 1 {
			protocolDesc = "Use only HTTP connections (not recommended)"
		} else {
			protocolDesc = "Use both HTTP and HTTPS connections"
		}
		protocolLine := fmt.Sprintf("%s %s %s %s",
			labelStyle.Render("Protocol:"),
			dimStyle.Render("<"),
			protocolStyle.Render(protocolValue),
			dimStyle.Render(">"))
		content = append(content, protocolLine)
		content = append(content, "             "+descStyle.Render(protocolDesc))
		content = append(content, "")

		// Command preview with highlighting
		content = append(content, m.renderMirrorCommandPreview(overlayWidth-6, dimPurple))

		if m.mirrorUpdating {
			content = append(content, "")

			// Progress percentage
			pct := 0
			if m.mirrorProgressTotal > 0 {
				pct = m.mirrorProgressCurrent * 100 / m.mirrorProgressTotal
				if pct > 100 {
					pct = 100
				}
			}
			progressLabel := fmt.Sprintf("Updating mirrors... %d%%", pct)
			content = append(content, lipgloss.PlaceHorizontal(overlayWidth-4, lipgloss.Center,
				dimStyle.Render(progressLabel)))

			// Determinate progress bar
			barWidth := overlayWidth - 12
			if barWidth < 10 {
				barWidth = 10
			}
			filled := 0
			if m.mirrorProgressTotal > 0 {
				filled = m.mirrorProgressCurrent * barWidth / m.mirrorProgressTotal
				if filled > barWidth {
					filled = barWidth
				}
			}
			empty := barWidth - filled

			trackColor := currentTheme.ProgressTrack
			filledColor := activeColor
			bar := lipgloss.NewStyle().Background(filledColor).Render(strings.Repeat(" ", filled)) +
				lipgloss.NewStyle().Background(trackColor).Render(strings.Repeat(" ", empty))
			content = append(content, lipgloss.PlaceHorizontal(overlayWidth-4, lipgloss.Center, bar))
		}

		// Error message if any
		if m.mirrorError != "" {
			content = append(content, "")
			errorStyle := lipgloss.NewStyle().Foreground(colorRed).Width(overlayWidth - 4).Align(lipgloss.Center)
			content = append(content, errorStyle.Render(m.mirrorError))
		}

		content = append(content, "")
		hintStyle := lipgloss.NewStyle().Foreground(colorDimGray).Width(overlayWidth - 4).Align(lipgloss.Center)
		content = append(content, hintStyle.Render("[j/k] navigate  [h/l] change  [enter] update  [esc] close"))
	}

	dialogContent := strings.Join(content, "\n")

	dialogStyle := lipgloss.NewStyle().
		Border(m.getBorderStyle()).
		BorderForeground(activeColor).
		Padding(1, 2).
		Width(overlayWidth)

	dialog := dialogStyle.Render(dialogContent)

	return lipgloss.Place(innerWidth, innerHeight, lipgloss.Center, lipgloss.Center, dialog)
}

// renderMirrorCommandPreview renders the command preview with highlighted parts based on selection
func (m *model) renderMirrorCommandPreview(maxWidth int, highlightColor lipgloss.Color) string {
	baseStyle := lipgloss.NewStyle().Foreground(colorDimGray)
	highlightStyle := lipgloss.NewStyle().Foreground(highlightColor)

	// Build command parts
	cmdPrefix := "sudo reflector"
	latestPart := fmt.Sprintf("--latest %d", m.mirrorConfig.Latest)
	sortPart := fmt.Sprintf("--sort %s", MirrorSortOptions[m.mirrorConfig.SortBy].Flag)

	var countryPart string
	if m.mirrorConfig.CountryIndex > 0 {
		countryPart = fmt.Sprintf("--country %s", MirrorCountries[m.mirrorConfig.CountryIndex].Code)
	}

	var protocolPart string
	if m.mirrorConfig.Protocol < len(MirrorProtocols) && MirrorProtocols[m.mirrorConfig.Protocol].Flag != "" {
		protocolPart = fmt.Sprintf("--protocol %s", MirrorProtocols[m.mirrorConfig.Protocol].Flag)
	}

	savePart := "--save /etc/pacman.d/mirrorlist"

	// Build the command with appropriate highlighting
	var parts []string
	parts = append(parts, baseStyle.Render(cmdPrefix))

	// Latest
	if m.mirrorSelectedItem == mirrorItemLatest {
		parts = append(parts, highlightStyle.Render(latestPart))
	} else {
		parts = append(parts, baseStyle.Render(latestPart))
	}

	// Sort
	if m.mirrorSelectedItem == mirrorItemSortBy {
		parts = append(parts, highlightStyle.Render(sortPart))
	} else {
		parts = append(parts, baseStyle.Render(sortPart))
	}

	// Country (only if set)
	if countryPart != "" {
		if m.mirrorSelectedItem == mirrorItemCountry {
			parts = append(parts, highlightStyle.Render(countryPart))
		} else {
			parts = append(parts, baseStyle.Render(countryPart))
		}
	}

	// Protocol (only if set)
	if protocolPart != "" {
		if m.mirrorSelectedItem == mirrorItemProtocol {
			parts = append(parts, highlightStyle.Render(protocolPart))
		} else {
			parts = append(parts, baseStyle.Render(protocolPart))
		}
	}

	// Save
	parts = append(parts, baseStyle.Render(savePart))

	cmd := strings.Join(parts, " ")

	// Wrap if needed
	cmdStyle := lipgloss.NewStyle().Width(maxWidth)
	return cmdStyle.Render(cmd)
}

// renderSimpleUpdateView renders the simple overview page for pending updates
func (m *model) renderSimpleUpdateView(helpText string, innerWidth, innerHeight int, activeColor lipgloss.Color) string {
	borderStyle := baseBorderStyle.BorderForeground(activeColor)

	footerLine := renderCenteredFooter(helpText, innerWidth)

	footerHeight := 0
	if footerLine != "" {
		footerHeight = 1
	}

	// The panel must fit in the remaining height
	panelHeight := innerHeight - footerHeight
	if panelHeight < 5 {
		panelHeight = 5
	}

	// Build mirror button for top right inside the panel
	var mirrorButton string
	if !m.loading {
		buttonStylePurple := lipgloss.NewStyle().
			Background(currentTheme.SelectedColor).
			Foreground(currentTheme.ButtonFg).
			Padding(0, 1).
			Bold(true)
		mirrorButton = buttonStylePurple.Render("[m]irrors")
	}

	var content strings.Builder

	if m.loading || m.pendingUpdates == nil {
		content.WriteString("\n  Checking for updates...")
	} else if len(m.pendingUpdates) == 0 {
		content.WriteString("\n  System is up to date!")
	} else {
		countStyle := lipgloss.NewStyle().Foreground(currentTheme.WarningColor).Bold(true)
		content.WriteString(fmt.Sprintf("  The following %s system updates are available:\n\n", countStyle.Render(fmt.Sprintf("%d", len(m.pendingUpdates)))))

		// Calculate max repo width for alignment
		maxRepoWidth := 0
		for _, pkg := range m.pendingUpdates {
			w := len(pkg.Source) + 2 // [source]
			if w > maxRepoWidth {
				maxRepoWidth = w
			}
		}

		// Calculate available lines for the list
		// panelHeight is TOTAL height of the panel including borders
		// Inner height is panelHeight - 2
		// mirrorLine takes 1 line (always present when updates available)
		// Buttons take 1 line
		// Header takes 2 lines ("The following... \n\n")
		// availableLinesForList = (innerHeightOfPanel) - mirrorLine - buttons - header
		availableLinesForList := (panelHeight - 2) - 1 - 1 - 2
		if availableLinesForList < 1 {
			availableLinesForList = 1
		}

		displayCount := availableLinesForList
		if displayCount > len(m.pendingUpdates) {
			displayCount = len(m.pendingUpdates)
		}

		m.maxUpdateScroll = len(m.pendingUpdates) - displayCount
		if m.maxUpdateScroll < 0 {
			m.maxUpdateScroll = 0
		}

		// Clamp current offset
		if m.updateScrollOffset > m.maxUpdateScroll {
			m.updateScrollOffset = m.maxUpdateScroll
		}

		var listBuilder strings.Builder
		for i := 0; i < displayCount; i++ {
			pkgIndex := i + m.updateScrollOffset
			if pkgIndex >= len(m.pendingUpdates) {
				break
			}
			pkg := m.pendingUpdates[pkgIndex]
			sourceBadge := ""
			sourceName := fmt.Sprintf("[%s]", pkg.Source)
			if color, ok := sourceColors[pkg.Source]; ok {
				sourceBadge = lipgloss.NewStyle().Foreground(color).Render(sourceName)
			} else {
				sourceBadge = sourceName
			}

			// Add padding to align repo and package name in columns with 1 space separation
			paddedSourceBadge := sourceBadge + strings.Repeat(" ", max(0, maxRepoWidth-len(sourceName)))

			line := fmt.Sprintf("    • %s %s %s",
				paddedSourceBadge,
				lipgloss.NewStyle().Foreground(currentTheme.TextColor).Render(pkg.Name),
				lipgloss.NewStyle().Foreground(currentTheme.DimText).Render(pkg.Version),
			)
			// Truncate to fit innerWidth-8 (accounting for scrollbar space)
			if lipgloss.Width(line) > innerWidth-8 {
				line = truncateWithAnsi(line, innerWidth-11) + "..."
			}
			listBuilder.WriteString(line + "\n")
		}

		listStr := strings.TrimSuffix(listBuilder.String(), "\n")
		if len(m.pendingUpdates) > displayCount {
			// Add scrollbar (Top-down)
			scrollbar := renderScrollbar(len(m.pendingUpdates), m.updateScrollOffset, displayCount, activeColor, false)
			// Join list and scrollbar
			listStr = lipgloss.JoinHorizontal(lipgloss.Top,
				lipgloss.NewStyle().Width(innerWidth-8).Render(listStr),
				lipgloss.NewStyle().MarginLeft(1).Render(scrollbar))
		}
		content.WriteString(listStr)
	}

	// Build main list content
	listContent := content.String()

	// Build buttons separately to pin them to the bottom right
	var buttonsContent string
	if !m.loading && len(m.pendingUpdates) > 0 {
		buttonStyle := lipgloss.NewStyle().
			Background(currentTheme.ButtonBg).
			Foreground(currentTheme.ButtonFg).
			Padding(0, 1).
			Bold(true)

		buttonStyleRed := lipgloss.NewStyle().
			Background(currentTheme.ButtonDangerBg).
			Foreground(currentTheme.ButtonFg).
			Padding(0, 1).
			Bold(true)

		btnUpdate := buttonStyle.Render(renderKeyHint("update", m.keys.Confirm, buttonStyle))
		btnSelective := buttonStyleRed.Render(renderKeyHint("select", m.keys.Selective, buttonStyleRed))
		buttons := btnUpdate + "   " + btnSelective
		buttonsContent = lipgloss.PlaceHorizontal(innerWidth-4, lipgloss.Right, buttons)
	}

	// Total available inner height is panelHeight - 2 (borders)
	innerHeightOfPanel := panelHeight - 2

	// Calculate heights for each section:
	// - mirrorLine: 1 line (when present)
	// - listContent: variable, fills remaining space
	// - buttonsContent: 1 line (when present)
	mirrorLineHeight := 0
	if mirrorButton != "" {
		mirrorLineHeight = 1
	}
	buttonsHeight := 0
	if buttonsContent != "" {
		buttonsHeight = 1
	}

	// List gets remaining height after mirror line and buttons
	listHeight := innerHeightOfPanel - mirrorLineHeight - buttonsHeight
	if listHeight < 1 {
		listHeight = 1
	}

	// Build mirror button line for top right
	mirrorLine := ""
	if mirrorButton != "" {
		mirrorLine = lipgloss.PlaceHorizontal(innerWidth-4, lipgloss.Right, mirrorButton)
	}

	// Create fixed-height list container to push buttons to bottom
	listContainer := lipgloss.NewStyle().
		Height(listHeight).
		Render(truncateHeight(listContent, listHeight))

	innerPanelContent := lipgloss.JoinVertical(lipgloss.Left,
		mirrorLine,
		listContainer,
		buttonsContent,
	)

	mainPanel := borderStyle.
		Width(innerWidth-2).
		Height(max(0, panelHeight-2)).
		Padding(0, 1).
		Align(lipgloss.Left, lipgloss.Top).
		Render(truncateHeight(innerPanelContent, max(0, panelHeight-2)))

	return SafeJoinVertical(innerWidth, innerHeight, "", []string{mainPanel}, footerLine)
}
