package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyMsg:
		// 1. Global Intercepts (Highest Priority)
		if key.Matches(msg, m.keys.Quit) {
			// ctrl+c should always quit, but 'q' should only quit if input is not focused
			if msg.Type == tea.KeyCtrlC || !m.textInput.Focused() {
				m.saveSettingsToDisk()
				return m, tea.Quit
			}
		}
		if msg.String() == "ctrl+r" && m.mode == modeDashboard {
			m.loading = true
			m.statusMessage = "Refreshing dashboard..."
			return m, getDashboardData(&m.config)
		}

		// 2. Overlays & Panel Intercepts
		if m.showMirrorOverlay {
			return m.handleMirrorOverlayKey(msg)
		}
		if m.mode == modeSettings {
			switch {
			case key.Matches(msg, m.keys.Cancel) || key.Matches(msg, m.keys.Settings):
				m.saveSettingsToDisk()
				m.mode = m.previousMode
				if m.config.Commands.AurHelper != m.originalHelper {
					return m, m.refreshAll()
				}
				return m, nil
			case msg.String() == "up" || msg.String() == "k":
				if m.settingsIndex > 0 {
					m.settingsIndex--
				}
			case msg.String() == "down" || msg.String() == "j":
				if m.settingsIndex < len(m.settingsItems)-1 {
					m.settingsIndex++
				}
			case msg.String() == "left" || msg.String() == "h":
				item := &m.settingsItems[m.settingsIndex]
				item.ActiveIndex--
				if item.ActiveIndex < 0 {
					item.ActiveIndex = len(item.Options) - 1
				}
				m.updateConfigFromSettings()
			case msg.String() == "right" || msg.String() == "l":
				item := &m.settingsItems[m.settingsIndex]
				item.ActiveIndex++
				if item.ActiveIndex >= len(item.Options) {
					item.ActiveIndex = 0
				}
				m.updateConfigFromSettings()
			}
			return m, nil
		}

		if m.showErrorOverlay {
			if key.Matches(msg, m.keys.Cancel) || key.Matches(msg, m.keys.Confirm) || key.Matches(msg, m.keys.Quit) {
				m.showErrorOverlay = false
			}
			return m, nil
		}
		if m.showConfirmation {
			return m.handleConfirmationKey(msg)
		}
		if m.selectionPanelFocused {
			return m.handleSelectionPanelKey(msg)
		}

		// 3. Navigation (Arrows, Page Keys, and JK when not focused)
		isNav := false
		switch msg.Type {
		case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown:
			isNav = true
		}
		if !m.textInput.Focused() && (msg.String() == "j" || msg.String() == "k") {
			isNav = true
		}

		if isNav {
			return m.handleNavigation(msg)
		}

		// 4. Alt+N Mode Switching (work regardless of text input focus)
		// Only Alt+1/2/3/4 are intercepted here; regular keys (d/i/u/r) are handled later
		isAltKey := msg.Alt && len(msg.Runes) == 1
		if isAltKey {
			switch msg.Runes[0] {
			case '1':
				if key.Matches(msg, m.keys.DashboardMode) {
					m.mode = modeDashboard
					m.loading = true
					m.resetState()
					return m, getDashboardData(&m.config)
				}
			case '2':
				if key.Matches(msg, m.keys.InstallMode) {
					if m.mode != modeInstall {
						m.mode = modeInstall
						m.resetState()
						m.textInput.Focus()
					}
					return m, nil
				}
			case '3':
				if key.Matches(msg, m.keys.UpdateMode) {
					m.mode = modeUpdate
					m.resetState()
					m.loading = true
					m.pendingUpdates = nil
					return m, syncRepositoriesInTerminal(m)
				}
			case '4':
				if key.Matches(msg, m.keys.RemoveMode) {
					if m.mode != modeRemove {
						m.mode = modeRemove
						m.resetState()
						m.loading = true
						m.statusMessage = "Refreshing installed packages..."
						return m, getInstalledPackages()
					}
					return m, nil
				}
			}
		}

		// 5. Input Focus Mode
		if m.textInput.Focused() {
			if key.Matches(msg, m.keys.Cancel) {
				if m.mode == modeUpdateSelective {
					m.mode = modeUpdate
					m.resetState()
					m.textInput.SetValue("")
					m.lastQuery = ""
					m.packageDetails = ""
					m.detailsForPackage = ""
					m.detailsScrollOffset = 0
				}
				m.textInput.Blur()
				return m, nil
			}
			if key.Matches(msg, m.keys.Confirm) {
				return m.handleActionTrigger()
			}
			if key.Matches(msg, m.keys.Mark) {
				return m.handleMarking()
			}

			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			filterCmd := m.performFiltering()
			return m, tea.Batch(cmd, filterCmd)
		}

		// 5. General Mode Keys (Unfocused)
		switch {
		case key.Matches(msg, m.keys.Cancel):
			if m.mode == modeCacheSelective {
				m.mode = modeCacheMenu
				m.resetState()
				m.statusMessage = "Selective cache cleaning cancelled"
				return m, nil
			}
			if m.mode == modeCacheMenu {
				m.mode = modeDashboard
				m.statusMessage = "Cache menu cancelled"
				m.resetState()
				return m, nil
			}
			if len(m.markedPackages) > 0 {
				m.resetState()
				m.statusMessage = "Selections cleared"
				return m, nil
			}
			return m, nil
		case key.Matches(msg, m.keys.Settings):
			m.previousMode = m.mode
			m.mode = modeSettings
			m.originalHelper = m.config.Commands.AurHelper
			return m, nil
		case key.Matches(msg, m.keys.Search):
			m.textInput.Focus()
			return m, nil
		case msg.String() == "c":
			if m.mode == modeDashboard && !m.loading {
				m.mode = modeCacheMenu
				m.cacheMenuIndex = 0
				m.resetState()
			}
		case msg.String() == "R":
			if m.mode == modeDashboard && !m.loading && m.dashboard.Orphans > 0 {
				orphanList, _ := runner.Run(m.config.Commands.AurHelper, "-Qdtq")
				m.confirmPackages = strings.Fields(string(orphanList))
				m.showConfirmation = true
				m.confirmType = confirmRemoveOrphans
			}
		case msg.String() == "t", msg.String() == "e", msg.String() == "f", msg.String() == "o":
			if m.mode == modeDashboard && !m.loading {
				m.mode = modeRemove
				m.resetState()
				m.textInput.SetValue(msg.String() + ":")

				if len(m.installed) > 0 {
					m.loading = false
					return m, m.performFiltering()
				}
				m.loading = true
				return m, getInstalledPackages()
			}
		case key.Matches(msg, m.keys.DashboardMode):
			m.mode = modeDashboard
			m.loading = true
			m.resetState()
			return m, getDashboardData(&m.config)
		case key.Matches(msg, m.keys.InstallMode):
			if m.mode != modeInstall {
				m.mode = modeInstall
				m.resetState()
				m.textInput.Focus()
			}
			return m, nil
		case key.Matches(msg, m.keys.UpdateMode):
			m.mode = modeUpdate
			m.resetState()
			m.loading = true
			m.pendingUpdates = nil
			return m, syncRepositoriesInTerminal(m)
		case key.Matches(msg, m.keys.RemoveMode):
			if m.mode != modeRemove {
				m.mode = modeRemove
				m.resetState()
				m.loading = true
				m.statusMessage = "Refreshing installed packages..."
				return m, getInstalledPackages()
			}
			return m, nil
		case key.Matches(msg, m.keys.Selective):
			if m.mode == modeUpdate {
				m.mode = modeUpdateSelective
				m.resetState()
				m.textInput.Focus()
				if len(m.pendingUpdates) > 0 {
					m.filtered = m.pendingUpdates
					m.loadingDetails = true
					m.detailsForPackage = m.filtered[0].Name
					return m, getPackageDetails(m, m.filtered[0])
				}
			}
		case msg.String() == "m" || msg.String() == "M":
			if m.mode == modeUpdate && !m.loading {
				m.showMirrorOverlay = true
				m.mirrorSelectedItem = mirrorItemSortBy
				m.mirrorError = ""
				LogDebug("MIRROR", "Mirror overlay opened")
				return m, nil
			}
		case key.Matches(msg, m.keys.Confirm):
			return m.handleActionTrigger()
		case msg.String() == "y" || msg.String() == "Y" || msg.String() == "a":
			if m.mode == modeUpdate && !m.loading && len(m.pendingUpdates) > 0 {
				m.statusMessage = "Running system update..."
				return m, executeUpdateInTerminal(m)
			}
		case key.Matches(msg, m.keys.Mark):
			return m.handleMarking()
		case msg.String() == "*":
			if len(m.markedPackages) > 0 {
				m.selectionPanelFocused = true
				m.selectionPanelIndex = 0
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalculateTextInputWidth()

	case repoPackagesMsg:
		m.loading = false
		if msg.err == nil {
			m.repoPackages = msg.packages
			m.statusMessage = fmt.Sprintf("Loaded %d packages", len(msg.packages))
			return m, m.performFiltering()
		} else {
			m.statusMessage = "Failed to load packages"
		}

	case syncRepositoriesMsg:
		if msg.err == nil {
			m.statusMessage = "Sync completed successfully"
			return m, m.refreshAll()
		}
		m.loading = false
		m.statusMessage = "Sync failed"
		m.showErrorOverlay = true
		m.errorTitle = "Repository Sync Failed"
		m.errorMessage = msg.err.Error()

	case updateCheckMsg:
		m.loading = false
		if msg.err == nil {
			m.pendingUpdates = msg.packages
			m.updatableAll = msg.packages
			if len(msg.packages) == 0 {
				m.statusMessage = "System is up to date"
			} else {
				m.statusMessage = fmt.Sprintf("%d updates available", len(msg.packages))
			}
		} else {
			m.statusMessage = "Failed to check for updates"
			m.showErrorOverlay = true
			m.errorTitle = "Update Check Error"
			m.errorMessage = msg.err.Error()
		}

	case aurSearchMsg:
		m.searchingAUR = false
		query := m.textInput.Value()
		repoFilters, searchQuery := parseRepoFilter(query)
		shouldSearchAUR := len(repoFilters) == 0 || repoFilters["aur"]

		if msg.err == nil {
			m.searchError = false
			if shouldSearchAUR && searchQuery == msg.query {
				m.aurPackages = msg.packages
				m.searchTerm = msg.query // Mark this query as complete/current
				if len(msg.packages) == 0 {
					m.searchStatus = fmt.Sprintf("No AUR packages found. Took %.2f seconds.", msg.timeTaken.Seconds())
				} else {
					m.searchStatus = fmt.Sprintf("AUR search complete. Took %.2f seconds.", msg.timeTaken.Seconds())
				}
			}
			return m, m.performFiltering()
		} else {
			if shouldSearchAUR && searchQuery == msg.query {
				m.searchError = true
				m.searchTerm = msg.query // Mark this query as complete (even if failed)
				errMsg := msg.err.Error()
				if strings.Contains(errMsg, "Too many package results") {
					m.searchStatus = "Search term too broad (too many results)."
				} else {
					m.searchStatus = fmt.Sprintf("AUR search failed: %s", simplifyErrorMessage(errMsg))
				}
			}
			return m, m.performFiltering()
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case packageDetailsMsg:
		if msg.err == nil {
			m.detailsCache[msg.packageName] = msg.details
		}
		if msg.packageName == m.detailsForPackage {
			m.loadingDetails = false
			m.packageDetails = msg.details
		}

	case debounceTickMsg:
		if msg.packageName == m.pendingDetailsPackage {
			m.detailsForPackage = msg.packageName
			pkg := m.getPackageByName(msg.packageName)
			if pkg != nil {
				m.detailsScrollOffset = 0
				return m, getPackageDetails(m, *pkg)
			}
		}

	case installedPackagesMsg:
		m.loading = false
		if msg.err == nil {
			m.installed = msg.packages
			// Always initialize the filtered list so it's ready even if we aren't in remove mode yet
			m.filteredInstalled = m.installed
			m.statusMessage = fmt.Sprintf("Loaded %d installed packages", len(msg.packages))
			return m, m.performFiltering()
		} else {
			m.statusMessage = "Failed to load installed packages"
		}

	case dashboardMsg:
		m.loading = false
		if msg.err == nil {
			m.dashboard = msg.data
			// If we are in selective cache mode, we need to update our list from the fresh dashboard data
			if m.mode == modeCacheSelective {
				m.filtered = make([]Package, len(m.dashboard.AllCacheHogs))
				for i, h := range m.dashboard.AllCacheHogs {
					m.filtered[i] = Package{Name: h.Name, Size: h.Size, SizeBytes: h.SizeBytes}
				}
				// Re-apply search filter if there was one
				if m.textInput.Value() != "" {
					m.filtered = fuzzyFilter(m.filtered, m.textInput.Value())
					m.matchIndices = computeAllMatchIndices(m.filtered, m.textInput.Value())
				}
			}
		}

	case actionCompleteMsg:
		m.loading = false
		m.statusMessage = msg.message
		if msg.err != nil {
			m.showErrorOverlay = true
			m.errorTitle = "Action Failed"
			m.errorMessage = msg.err.Error()
			return m, nil
		}
		return m, m.refreshAll()

	case updateOutputMsg:
		if msg.done {
			m.loading = false
			if msg.err != nil {
				m.showErrorOverlay = true
				m.errorTitle = "Update Failed"
				m.errorMessage = msg.err.Error()
			} else {
				m.statusMessage = "Update completed successfully"
			}
			return m, checkUpdates(&m.config)
		}

	case execCompleteMsg:
		return m.handleExecComplete(msg)

	case mirrorSudoReadyMsg:
		if msg.err != nil {
			m.mirrorUpdating = false
			m.mirrorError = "sudo authentication failed"
			LogError("MIRROR", "Sudo authentication failed: %v", msg.err)
			return m, nil
		}
		LogInfo("MIRROR", "Sudo credentials acquired, executing mirror update")
		return m, executeMirrorUpdate(m.mirrorConfig)

	case mirrorProgressMsg:
		m.mirrorProgressCurrent = msg.current
		m.mirrorProgressTotal = msg.total
		return m, waitForMirrorProgress(msg.ch)

	case mirrorUpdateMsg:
		m.mirrorUpdating = false
		if msg.err != nil {
			m.mirrorError = msg.err.Error()
			LogError("MIRROR", "Mirror update failed: %v", msg.err)
		} else {
			m.showMirrorOverlay = false
			m.mirrorError = ""
			m.statusMessage = "Mirrors updated successfully"
			LogInfo("MIRROR", "Mirror update completed successfully")
			// Refresh updates after mirror change
			m.loading = true
			return m, syncRepositoriesInTerminal(m)
		}
	}

	return m, tea.Batch(cmds...)
}

// --- Internal Helper Methods ---

func (m *model) handleNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := strings.ToLower(msg.String())

	// 1. Cache Menu Navigation
	if m.mode == modeCacheMenu {
		switch key {
		case "up", "k":
			if m.cacheMenuIndex > 0 {
				m.cacheMenuIndex--
			}
		case "down", "j":
			if m.cacheMenuIndex < 4 {
				m.cacheMenuIndex++
			}
		case "pgup":
			m.cacheMenuIndex = 0
		case "pgdown":
			m.cacheMenuIndex = 4
		}
		return m, nil
	}

	// 2. Mode Update Scroll
	if m.mode == modeUpdate {
		switch key {
		case "up", "k":
			if m.updateScrollOffset > 0 {
				m.updateScrollOffset--
			}
		case "down", "j":
			if m.updateScrollOffset < len(m.pendingUpdates)-1 {
				m.updateScrollOffset++
			}
		case "pgup":
			m.updateScrollOffset = 0
		case "pgdown":
			m.updateScrollOffset = len(m.pendingUpdates) - 1
		}
		return m, nil
	}

	// 3. Selection List Navigation
	maxIndex := 0
	if m.mode == modeRemove {
		maxIndex = len(m.filteredInstalled) - 1
	} else {
		maxIndex = len(m.filtered) - 1
	}

	oldIdx := m.selectedIndex
	jump := 1
	if key == "pgup" || key == "pgdown" {
		jump = 10
	}

	// For bottom-up menus (modeInstall, modeRemove, modeUpdateSelective), navigation is inverted:
	// - Visual "up" (pressing up arrow) should increase index (move toward higher indices shown at top)
	// - Visual "down" (pressing down arrow) should decrease index (move toward lower indices shown at bottom)
	isBottomUpMenu := m.mode == modeInstall || m.mode == modeRemove || m.mode == modeUpdateSelective
	if isBottomUpMenu {
		// Invert navigation for bottom-up rendering
		if key == "up" || key == "k" || key == "pgup" {
			m.selectedIndex += jump
		} else {
			m.selectedIndex -= jump
		}
	} else {
		// Standard top-down navigation
		if key == "up" || key == "k" || key == "pgup" {
			m.selectedIndex -= jump
		} else {
			m.selectedIndex += jump
		}
	}

	// Clamp
	if m.selectedIndex > maxIndex {
		m.selectedIndex = maxIndex
	}
	if m.selectedIndex < 0 {
		m.selectedIndex = 0
	}

	if m.selectedIndex != oldIdx {
		m.detailsScrollOffset = 0
		cmd := m.ensureDetailsLoaded(m.selectedPackage())
		return m, cmd
	}
	return m, nil
}

func (m *model) handleMarking() (tea.Model, tea.Cmd) {
	pkg := m.selectedPackage()
	if pkg == nil {
		return m, nil
	}
	if m.mode == modeCacheSelective {
		if m.markedPackages[pkg.Name] {
			m.cacheToFree -= pkg.SizeBytes
			delete(m.markedPackages, pkg.Name)
		} else {
			m.cacheToFree += pkg.SizeBytes
			m.markedPackages[pkg.Name] = true
		}
	} else {
		if m.markedPackages[pkg.Name] {
			delete(m.markedPackages, pkg.Name)
		} else {
			m.markedPackages[pkg.Name] = true
		}
	}
	return m, nil
}

func (m *model) handleActionTrigger() (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeInstall, modeRemove, modeUpdateSelective:
		var pkgs []string
		for n := range m.markedPackages {
			pkgs = append(pkgs, n)
		}
		if len(pkgs) == 0 {
			p := m.selectedPackage()
			if p != nil {
				pkgs = []string{p.Name}
			}
		}
		if len(pkgs) > 0 {
			sort.Strings(pkgs)
			m.showConfirmation = true
			m.confirmPackages = pkgs
			if m.mode == modeInstall {
				m.confirmType = confirmInstall
			}
			if m.mode == modeRemove {
				m.confirmType = confirmRemove
			}
			if m.mode == modeUpdateSelective {
				m.confirmType = confirmSelectiveUpdate
			}
		}
	case modeCacheMenu:
		switch m.cacheMenuIndex {
		case 4:
			m.mode = modeCacheSelective
			m.selectedIndex = 0
			m.textInput.SetValue("")
			m.lastQuery = ""
			m.packageDetails = ""
			m.detailsForPackage = ""
			m.detailsScrollOffset = 0
			m.resetState()
			m.filtered = make([]Package, len(m.dashboard.AllCacheHogs))
			for i, h := range m.dashboard.AllCacheHogs {
				m.filtered[i] = Package{Name: h.Name, Size: h.Size, SizeBytes: h.SizeBytes}
			}
		default:
			m.showConfirmation = true
			types := []confirmationType{confirmCleanKeep3, confirmCleanKeep1, confirmCleanRemoved, confirmCleanNuke}
			m.confirmType = types[m.cacheMenuIndex]
		}
	case modeCacheSelective:
		if len(m.markedPackages) > 0 {
			var pkgs []string
			for n := range m.markedPackages {
				pkgs = append(pkgs, n)
			}
			sort.Strings(pkgs)
			m.showConfirmation = true
			m.confirmType = confirmCleanSelective
			m.confirmPackages = pkgs
		}
	case modeUpdate:
		if len(m.pendingUpdates) > 0 {
			var pkgs []string
			for _, p := range m.pendingUpdates {
				pkgs = append(pkgs, p.Name)
			}
			m.showConfirmation = true
			m.confirmType = confirmUpdate
			m.confirmPackages = pkgs
			m.statusMessage = fmt.Sprintf("Confirm update for %d packages", len(m.pendingUpdates))
		}
	}
	return m, nil
}

func (m *model) handleConfirmationKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Confirm) || msg.String() == "y" || msg.String() == "Y" {
		m.showConfirmation = false
		switch m.confirmType {
		case confirmInstall:
			return m, executeInstallInTerminal(m, m.confirmPackages)
		case confirmRemove:
			return m, executeRemoveInTerminal(m, m.confirmPackages)
		case confirmUpdate:
			return m, executeUpdateInTerminal(m)
		case confirmSelectiveUpdate:
			return m, executeSelectiveUpdateInTerminal(m, m.confirmPackages)
		case confirmCleanKeep3:
			return m, executeCleanCache(m, confirmCleanKeep3, 3, false)
		case confirmCleanKeep1:
			return m, executeCleanCache(m, confirmCleanKeep1, 1, false)
		case confirmCleanRemoved:
			return m, executeCleanCache(m, confirmCleanRemoved, 0, true)
		case confirmCleanNuke:
			return m, executeCleanCache(m, confirmCleanNuke, 0, false)
		case confirmCleanSelective:
			return m, executeSelectiveClean(m, m.confirmPackages, m.dashboard.PacmanCachePath, m.dashboard.AurCachePath)
		case confirmRemoveOrphans:
			return m, executeRemoveOrphansInTerminal(m, m.confirmPackages)
		}
	} else if key.Matches(msg, m.keys.Cancel) || msg.String() == "n" || msg.String() == "N" {
		m.showConfirmation = false
	}

	// Scrolling in confirmation
	key := strings.ToLower(msg.String())
	if key == "up" || key == "k" {
		if m.confirmScrollOffset > 0 {
			m.confirmScrollOffset--
		}
	}
	if key == "down" || key == "j" {
		if m.confirmScrollOffset < m.maxConfirmScroll {
			m.confirmScrollOffset++
		}
	}
	if key == "pgup" {
		m.confirmScrollOffset = 0
	}
	if key == "pgdown" {
		m.confirmScrollOffset = m.maxConfirmScroll
	}

	return m, nil
}

func (m *model) handleSelectionPanelKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var names []string
	for n := range m.markedPackages {
		names = append(names, n)
	}
	sort.Strings(names)

	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.selectionPanelFocused = false
	case msg.String() == "up", msg.String() == "k":
		if m.selectionPanelIndex > 0 {
			m.selectionPanelIndex--
		}
	case msg.String() == "down", msg.String() == "j":
		if m.selectionPanelIndex < len(names)-1 {
			m.selectionPanelIndex++
		}
	case key.Matches(msg, m.keys.Mark):
		if m.selectionPanelIndex < len(names) {
			delete(m.markedPackages, names[m.selectionPanelIndex])
			if len(m.markedPackages) == 0 {
				m.selectionPanelFocused = false
			}
		}
	case key.Matches(msg, m.keys.Confirm):
		m.selectionPanelFocused = false
		return m.handleActionTrigger()
	}
	return m, nil
}

func (m *model) performFiltering() tea.Cmd {
	query := m.textInput.Value()
	m.lastQuery = query
	m.selectedIndex = 0

	var cmds []tea.Cmd

	if m.mode == modeInstall {
		// Trigger AUR search if query is long enough and different from last AUR search
		repoFilters, searchQuery := parseRepoFilter(query)

		// Check if we should search AUR:
		// 1. No repo filters are applied (global search)
		// 2. OR the 'aur' filter is explicitly requested
		shouldSearchAUR := len(repoFilters) == 0 || repoFilters["aur"]

		// Always update the status display if the user is typing a new query
		if shouldSearchAUR && len(searchQuery) >= minSearchQueryLen {
			// Only show "Searching..." if:
			// 1. We ARE searching and the in-flight query is NOT this one
			// 2. OR we NEED to search (query changed)
			if (m.searchingAUR && searchQuery != m.searchTerm) || searchQuery != m.lastAURQuery {
				m.searchError = false // Reset error state
				m.searchTerm = searchQuery
				m.searchStatus = fmt.Sprintf("Searching AUR for \"%s\"...", searchQuery)
			}
		} else {
			m.searchStatus = ""
			if searchQuery == "" || !shouldSearchAUR || len(searchQuery) < minSearchQueryLen {
				m.aurPackages = nil // Clear results if query is empty, filtered out, or too short
			}
		}

		if shouldSearchAUR && len(searchQuery) >= minSearchQueryLen && searchQuery != m.lastAURQuery && !m.searchingAUR {
			m.searchingAUR = true
			m.lastAURQuery = searchQuery
			cmds = append(cmds, m.spinner.Tick, searchAUR(&m.config, searchQuery))
		}

		m.filterAllPackages(query)
	}
	if m.mode == modeRemove {
		m.filterInstalledPackages(query)
	}
	if m.mode == modeUpdateSelective {
		if query == "" {
			m.filtered = m.updatableAll
			m.matchIndices = nil
		} else {
			m.filtered = fuzzyFilter(m.updatableAll, query)
			m.matchIndices = computeAllMatchIndices(m.filtered, query)
		}
	}
	if m.mode == modeCacheSelective {
		// Convert AllCacheHogs to Package slice for fuzzyFilter
		var allPkgs []Package
		for _, h := range m.dashboard.AllCacheHogs {
			allPkgs = append(allPkgs, Package{Name: h.Name, Size: h.Size, SizeBytes: h.SizeBytes})
		}
		if query == "" {
			m.filtered = allPkgs
			m.matchIndices = nil
		} else {
			m.filtered = fuzzyFilter(allPkgs, query)
			m.matchIndices = computeAllMatchIndices(m.filtered, query)
		}
	}

	// Fetch details for the first item automatically if list is not empty
	if cmd := m.ensureDetailsLoaded(m.selectedPackage()); cmd != nil {
		cmds = append(cmds, cmd)
	}
	return tea.Batch(cmds...)
}

// ensureDetailsLoaded checks cache and triggers debounce loading for the given package.
func (m *model) ensureDetailsLoaded(pkg *Package) tea.Cmd {
	if pkg == nil || m.mode == modeCacheSelective {
		m.loadingDetails = false
		m.packageDetails = ""
		m.detailsForPackage = ""
		return nil
	}
	if cached, ok := m.detailsCache[pkg.Name]; ok {
		m.packageDetails = cached
		m.loadingDetails = false
		m.detailsForPackage = pkg.Name
		m.pendingDetailsPackage = ""
		return nil
	}
	m.loadingDetails = true
	m.pendingDetailsPackage = pkg.Name
	return debouncePackageDetails(m, m.pendingDetailsPackage)
}

func (m *model) getPackageByName(name string) *Package {
	// 1. Check currently filtered list (fastest)
	list := m.filtered
	if m.mode == modeRemove {
		list = m.filteredInstalled
	}
	for i := range list {
		if list[i].Name == name {
			return &list[i]
		}
	}

	// 2. Check full source lists if not found in filtered
	// Check installed packages (for removal/updates)
	for i := range m.installed {
		if m.installed[i].Name == name {
			return &m.installed[i]
		}
	}

	// Check repo packages (for installation)
	for i := range m.repoPackages {
		if m.repoPackages[i].Name == name {
			return &m.repoPackages[i]
		}
	}

	// Check AUR packages (for installation)
	for i := range m.aurPackages {
		if m.aurPackages[i].Name == name {
			return &m.aurPackages[i]
		}
	}

	// Check pending updates
	for i := range m.pendingUpdates {
		if m.pendingUpdates[i].Name == name {
			return &m.pendingUpdates[i]
		}
	}

	return nil
}

func (m *model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	isScrollUp := msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonWheelUp
	isScrollDown := msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonWheelDown

	if m.showConfirmation {
		if isScrollUp {
			if m.confirmScrollOffset > 0 {
				m.confirmScrollOffset--
			}
		}
		if isScrollDown {
			if m.confirmScrollOffset < m.maxConfirmScroll {
				m.confirmScrollOffset++
			}
		}
		return m, nil
	}

	if !isScrollUp && !isScrollDown {
		return m, nil
	}

	// Details pane scroll check
	if (m.mode == modeInstall || m.mode == modeRemove) && msg.Y < m.height/2 {
		if isScrollUp {
			if m.detailsScrollOffset > 0 {
				m.detailsScrollOffset--
			}
		} else {
			if m.detailsScrollOffset < m.maxDetailsScroll {
				m.detailsScrollOffset++
			}
		}
		return m, nil
	}
	if m.mode == modeUpdateSelective && msg.X >= m.width/2 {
		if isScrollUp {
			if m.detailsScrollOffset > 0 {
				m.detailsScrollOffset--
			}
		} else {
			if m.detailsScrollOffset < m.maxDetailsScroll {
				m.detailsScrollOffset++
			}
		}
		return m, nil
	}

	fake := tea.KeyMsg{Type: tea.KeyUp}
	if isScrollDown {
		fake.Type = tea.KeyDown
	}
	return m.handleNavigation(fake)
}

func (m *model) handleExecComplete(msg execCompleteMsg) (tea.Model, tea.Cmd) {
	m.loading = false
	if msg.err != nil {
		m.showErrorOverlay = true
		m.errorTitle = "Operation Failed"
		m.errorMessage = msg.err.Error()
		return m, nil
	}

	// Clear selection state on success
	m.resetState()
	m.confirmPackages = nil

	m.statusMessage = "Operation completed successfully"
	return m, m.refreshAll()
}

func (m *model) recalculateTextInputWidth() {
	if m.mode == modeInstall {
		// Leave room for hints "c: e: m: a:" (11 chars) + padding (3 chars)
		m.textInput.Width = max(20, m.width-20)
	} else {
		m.textInput.Width = max(20, m.width-6)
	}
}

// handleMirrorOverlayKey handles keyboard input when the mirror overlay is active
func (m *model) handleMirrorOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.mirrorUpdating {
		// Don't allow interaction while updating
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.showMirrorOverlay = false
		m.mirrorError = ""
		LogDebug("MIRROR", "Mirror overlay closed")
		return m, nil

	case msg.String() == "up" || msg.String() == "k":
		if m.mirrorSelectedItem > 0 {
			m.mirrorSelectedItem--
		}

	case msg.String() == "down" || msg.String() == "j":
		if m.mirrorSelectedItem < mirrorItemProtocol {
			m.mirrorSelectedItem++
		}

	case msg.String() == "left" || msg.String() == "h":
		m.adjustMirrorOption(-1)

	case msg.String() == "right" || msg.String() == "l":
		m.adjustMirrorOption(1)

	case key.Matches(msg, m.keys.Confirm):
		if !checkReflectorInstalled() {
			m.mirrorError = "reflector is not installed"
			return m, nil
		}
		m.mirrorUpdating = true
		m.mirrorProgressCurrent = 0
		m.mirrorProgressTotal = m.mirrorConfig.Latest
		m.mirrorError = ""
		LogInfo("MIRROR", "Acquiring sudo credentials for mirror update")
		return m, acquireSudoForMirror()
	}

	return m, nil
}

// adjustMirrorOption adjusts the currently selected mirror option by delta
func (m *model) adjustMirrorOption(delta int) {
	switch m.mirrorSelectedItem {
	case mirrorItemSortBy:
		m.mirrorConfig.SortBy += delta
		if m.mirrorConfig.SortBy < 0 {
			m.mirrorConfig.SortBy = len(MirrorSortOptions) - 1
		} else if m.mirrorConfig.SortBy >= len(MirrorSortOptions) {
			m.mirrorConfig.SortBy = 0
		}

	case mirrorItemCountry:
		m.mirrorConfig.CountryIndex += delta
		if m.mirrorConfig.CountryIndex < 0 {
			m.mirrorConfig.CountryIndex = len(MirrorCountries) - 1
		} else if m.mirrorConfig.CountryIndex >= len(MirrorCountries) {
			m.mirrorConfig.CountryIndex = 0
		}

	case mirrorItemLatest:
		m.mirrorConfig.Latest += delta * 5 // Increment by 5
		if m.mirrorConfig.Latest < 5 {
			m.mirrorConfig.Latest = 5
		} else if m.mirrorConfig.Latest > 100 {
			m.mirrorConfig.Latest = 100
		}

	case mirrorItemProtocol:
		m.mirrorConfig.Protocol += delta
		if m.mirrorConfig.Protocol < 0 {
			m.mirrorConfig.Protocol = len(MirrorProtocols) - 1
		} else if m.mirrorConfig.Protocol >= len(MirrorProtocols) {
			m.mirrorConfig.Protocol = 0
		}
	}
}
