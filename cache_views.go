package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *model) renderCacheMenu(helpText string, innerWidth, innerHeight int) string {
	menuWidth := 80
	menuHeight := 22

	if menuWidth > innerWidth-4 {
		menuWidth = innerWidth - 4
	}
	if menuHeight > innerHeight-4 {
		menuHeight = innerHeight - 4
	}

	selectedColor := colorGreen // default green
	switch m.cacheMenuIndex {
	case 0:
		selectedColor = colorBlue
	case 1:
		selectedColor = colorOrange
	case 2:
		selectedColor = colorGreen
	case 3:
		selectedColor = colorRed
	case 4:
		selectedColor = colorPurple
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(currentTheme.TitleColor).
		MarginBottom(1).
		Align(lipgloss.Center)

	descStyle := styleItalicDim()

	normalItemStyle := lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(currentTheme.TextColor)

	selectedItemStyle := lipgloss.NewStyle().
		PaddingLeft(2).
		Foreground(selectedColor).
		Bold(true)

	options := []struct {
		title string
		desc  string
	}{
		{"Safe Clean (Keep 3)", "Leaves the installed version and 2 fallbacks (paccache -r)"},
		{"Aggressive (Keep 1)", "Keeps only the currently installed version (paccache -rk1)"},
		{"Orphaned Cache", "Removes cached tarballs for removed packages (paccache -ruk0)"},
		{"Nuke Everything", "Empties the entire cache directory (paccache -rk0)"},
		{"Selective Clean", "Manually select specific packages to remove"},
	}

	var menuContent strings.Builder
	menuContent.WriteString(titleStyle.Width(menuWidth - 4).Render(" \U000f00e2 Cache Management Menu"))
	menuContent.WriteString("\n\n")

	for i, opt := range options {
		var itemStr string
		if i == m.cacheMenuIndex {
			itemStr = selectedItemStyle.Render("> " + opt.title)
		} else {
			itemStr = normalItemStyle.Render("  " + opt.title)
		}
		menuContent.WriteString(itemStr + "\n")

		descPadding := 4
		if i == m.cacheMenuIndex {
			menuContent.WriteString(descStyle.PaddingLeft(descPadding).Foreground(selectedColor).Render(opt.desc) + "\n\n")
		} else {
			menuContent.WriteString(descStyle.PaddingLeft(descPadding).Render(opt.desc) + "\n\n")
		}
	}

	menuBox := lipgloss.NewStyle().
		Width(max(0, menuWidth-2)).
		Height(max(0, menuHeight-4)).
		Border(m.getBorderStyle()).
		BorderForeground(selectedColor).
		Padding(1, 2).
		Render(truncateHeight(menuContent.String(), max(0, menuHeight-4)))

	// Overlay menu onto dashboard
	footerHeight := 0
	var footerLine string
	if helpText != "" {
		footerHeight = 1
		footerLine = renderCenteredFooter(helpText, innerWidth)
	}

	mainContent := lipgloss.Place(innerWidth, innerHeight-footerHeight, lipgloss.Center, lipgloss.Center, menuBox, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(currentTheme.ScrollbarTrack))

	return SafeJoinVertical(innerWidth, innerHeight, "", []string{mainContent}, footerLine)
}

func (m *model) renderSelectiveCacheView(helpText string, innerWidth, innerHeight int, activeColor lipgloss.Color) string {
	borderStyle := baseBorderStyle.BorderForeground(activeColor)

	headerHeight := 1
	footerHeight := 1

	availableHeight := innerHeight - headerHeight - footerHeight
	if availableHeight < 6 {
		availableHeight = 6
	}

	targetBottomPanelHeight := availableHeight
	bottomInnerHeight := targetBottomPanelHeight - 2

	// Calculate resultsHeight precisely to fill available space
	// Overhead: separator(1) + statusLine(1) + inputLine(1)
	resultsHeight := bottomInnerHeight - 3

	if resultsHeight < 1 {
		resultsHeight = 1
	}

	contentWidth := innerWidth - 4
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Build results list
	var resultsStr string
	pkgList := m.filtered

	if len(pkgList) == 0 {
		if m.textInput.Value() != "" {
			resultsStr = "  No matches for '" + m.textInput.Value() + "'"
		} else {
			resultsStr = "  No packages found in cache"
		}
	} else {
		resultsStr = RenderPaginatedList(PaginatedListConfig{
			TotalCount:     len(pkgList),
			SelectedIndex:  m.selectedIndex,
			ViewportHeight: resultsHeight,
			ContentWidth:   contentWidth,
			ActiveColor:    activeColor,
			Reversed:       true,
			RenderItem: func(i int, itemWidth int) string {
				pkg := pkgList[i]

				marker := "[ ]"
				if m.markedPackages[pkg.Name] {
					marker = "[x]"
				}
				prefix := "  " + marker
				if i == m.selectedIndex {
					prefix = "> " + marker
				}

				// nameWidth calculation
				// Left gutter: prefix (5) + space (1) = 6 chars
				// Right gutter: we want 2 chars for scrollbar space
				// Total layout: prefix (5) + space (1) + name (nameWidth) + space (1) + size (10) + scrollbar(2) = contentWidth
				nameWidth := contentWidth - 20
				if nameWidth < 10 {
					nameWidth = 10
				}

				nameStr := pkg.Name
				if indices, ok := m.matchIndices[i]; ok && m.textInput.Value() != "" {
					nameStr = highlightMatches(pkg.Name, indices)
				}
				nameStr = truncateWithAnsi(nameStr, nameWidth)

				visualNameWidth := lipgloss.Width(nameStr)
				paddedName := nameStr
				if visualNameWidth < nameWidth {
					paddedName += strings.Repeat(" ", nameWidth-visualNameWidth)
				}

				// Use non-breaking space to prevent wrapping within the size string
				displaySize := strings.ReplaceAll(pkg.Size, " ", "\u00a0")
				sizeStr := fmt.Sprintf("%10s", displaySize)

				if i == m.selectedIndex || m.markedPackages[pkg.Name] {
					bgColor := currentTheme.SelectionBG
					if i == m.selectedIndex {
						bgColor = currentTheme.SelectedColor
					}
					fgColor := currentTheme.TextColor

					maintainedName := maintainBackground(paddedName, bgColor)
					lineContent := fmt.Sprintf("%s %s %s", prefix, maintainedName, sizeStr)

					return lipgloss.NewStyle().
						Background(bgColor).
						Foreground(fgColor).
						Bold(i == m.selectedIndex).
						Width(itemWidth).
						Render(lineContent)
				}

				namePart := lipgloss.NewStyle().Foreground(currentTheme.TextColor).Render(paddedName)
				sizePart := lipgloss.NewStyle().Foreground(currentTheme.DimText).Render(sizeStr)
				line := fmt.Sprintf("%s %s %s", prefix, namePart, sizePart)
				// Ensure unselected lines also fit the width
				if lipgloss.Width(line) > itemWidth {
					line = truncateWithAnsi(line, itemWidth)
				}
				return line
			},
		})
	}

	resultsBox := lipgloss.NewStyle().
		Width(contentWidth).
		Height(resultsHeight).
		Align(lipgloss.Left, lipgloss.Bottom).
		Render(resultsStr)

	// Footer status text
	var statusMsg string
	if len(m.markedPackages) > 0 {
		statusMsg = fmt.Sprintf("Space to be freed: %s / %s total cache", formatBytes(m.cacheToFree), m.dashboard.CleanerSize)
	} else {
		statusMsg = fmt.Sprintf("Select packages to remove... (Total Cache: %s)", m.dashboard.CleanerSize)
	}

	statusLine := statusStyle.Render(statusMsg)

	inputLine := m.textInput.View()

	bottomContent := lipgloss.JoinVertical(
		lipgloss.Left,
		truncateHeight(resultsBox, resultsHeight),
		"",
		statusLine,
		inputLine,
	)

	bottomPanel := borderStyle.
		Width(innerWidth-2).
		Height(targetBottomPanelHeight-2).
		Align(lipgloss.Left, lipgloss.Bottom).
		Render(truncateHeight(bottomContent, bottomInnerHeight))

	header := styleBoldWithForeground(activeColor).Render(" \uf0c7 Selective Cache Clean")

	footer := renderCenteredFooter(helpText, innerWidth)

	return SafeJoinVertical(innerWidth, innerHeight, header, []string{bottomPanel}, footer)
}
