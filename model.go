package main

import (
	"context"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model
type model struct {
	config                Config
	runner                CommandRunner
	ctx                   context.Context
	cancelFunc            context.CancelFunc
	aurCancelFunc         context.CancelFunc
	keys                  KeyMap
	theme                 Theme
	themeLoader           *ThemeLoader
	textInput             textinput.Model
	repoPackages          []Package       // All repo packages from local cache
	aurPackages           []Package       // AUR packages from last search
	installedSet          map[string]bool // Quick lookup for installed packages
	packages              []Package
	filtered              []Package
	installed             []Package
	filteredInstalled     []Package
	matchIndices          map[int][]int // Maps package index to matched character indices
	selectedIndex         int
	markedPackages        map[string]bool // Packages marked for batch operation
	selectionPanelFocused bool            // Whether selection panel is focused
	selectionPanelIndex   int             // Selected index within selection panel
	selectionScrollOffset int             // Scroll offset for the selection panel
	packageDetails        string
	detailsCache          map[string]string // Cache for fetched package details
	detailsCacheOrder     []string          // FIFO order for detailsCache eviction
	detailsForPackage     string
	pendingDetailsPackage string // Package waiting for debounce to complete
	detailsScrollOffset   int    // Scroll offset for the dash/details pane
	maxDetailsScroll      int    // Maximum allowed scroll for dash pane
	loadingDetails        bool
	mode                  viewMode
	width                 int
	height                int
	loading               bool
	statusMessage         string
	lastQuery             string
	lastAURQuery          string // Last query sent to AUR search
	searchingAUR          bool   // Whether AUR search is in progress
	searchTerm            string // Current search term for status line
	searchStatus          string // "Searching..." or "Search complete..."
	searchError           bool   // Whether the last search failed
	spinner               spinner.Model
	dashboard             DashboardData
	// Confirmation dialog state
	showConfirmation    bool
	confirmType         confirmationType
	confirmPackages     []string  // Package names to operate on
	pendingUpdates      []Package // Updates available (for update confirmation)
	confirmScrollOffset int       // Scroll offset for confirmation package list
	maxConfirmScroll    int       // Max scroll for confirmation list
	// Update selection state
	updatableAll       []Package // All packages available for update (before selection)
	updateScrollOffset int       // Scroll offset for the simple update view
	maxUpdateScroll    int       // Max scroll for update view
	// Error overlay state
	showErrorOverlay bool
	errorTitle       string
	errorMessage     string
	errorDetails     string
	// Cache cleaning state
	cacheMenuIndex int
	cacheToFree    int64
	// Settings state
	settingsItems  []SettingItem
	settingsIndex  int
	previousMode   viewMode
	originalHelper string // Track AUR helper change for refresh
	// Mirror overlay state
	showMirrorOverlay     bool
	reflectorInstalled    bool
	reflectorCheckDone    bool
	mirrorConfig          MirrorConfig
	mirrorSelectedItem    MirrorOverlayItem
	mirrorUpdating        bool
	mirrorError           string
	mirrorProgressCurrent int // Current number of mirrors processed
	mirrorProgressTotal   int // Total number of mirrors to process
}

func initialModel(initialMode viewMode, cfg Config, tl *ThemeLoader, r ...CommandRunner) *model {
	if tl == nil {
		tl = GetThemeLoader()
	}
	ti := textinput.New()
	ti.CharLimit = textInputCharLimit
	ti.Width = textInputDefaultWidth

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(currentTheme.SpinnerColor)

	ctx, cancel := context.WithCancel(context.Background())

	m := &model{
		config:         cfg,
		runner:         getActiveRunner(r...),
		ctx:            ctx,
		cancelFunc:     cancel,
		keys:           NewKeyMap(cfg.Keys),
		theme:          currentTheme,
		themeLoader:    tl,
		textInput:      ti,
		repoPackages:   []Package{},
		installedSet:   make(map[string]bool),
		packages:       []Package{},
		filtered:       []Package{},
		installed:      []Package{},
		markedPackages: make(map[string]bool),
		detailsCache:   make(map[string]string),
		selectedIndex:  0,
		mode:           initialMode,
		loading:        true,
		spinner:        s,
		mirrorConfig:   DefaultMirrorConfig(),
	}

	m.updatePlaceholder()
	m.statusMessage = "Loading package database..."
	switch initialMode {
	case modeRemove:
		m.statusMessage = "Loading installed packages..."
	case modeUpdate:
		m.statusMessage = "Checking for updates..."
	case modeDashboard:
		m.statusMessage = "Loading system statistics..."
	}

	if initialMode == modeRemove {
		m.loadingDetails = true
		m.detailsForPackage = "..."
	}

	m.initSettings()
	m.recalculateTextInputWidth()
	return m
}

func (m *model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.spinner.Tick,
		loadRepoPackagesWithContext(m.getContext(), m.getRunner()),
		getInstalledPackagesWithContext(m.getContext(), m.getRunner()),
		func() tea.Msg {
			switch m.mode {
			case modeDashboard:
				return getDashboardDataWithContext(m.getContext(), &m.config, m.getRunner())()
			case modeUpdate:
				return checkUpdatesWithContext(m.getContext(), &m.config, m.getRunner())()
			}
			return nil
		},
	)
}

// currentPackageList returns the appropriate package list based on current mode.
func (m *model) currentPackageList() []Package {
	switch m.mode {
	case modeInstall, modeUpdateSelective, modeCacheSelective:
		return m.filtered
	case modeRemove:
		return m.filteredInstalled
	default:
		return nil
	}
}

// maxSelectableIndex returns the maximum valid index for the current package list.
func (m *model) maxSelectableIndex() int {
	pkgList := m.currentPackageList()
	if len(pkgList) == 0 {
		return 0
	}
	return len(pkgList) - 1
}

// selectedPackage returns the currently selected package, or nil if none.
func (m *model) selectedPackage() *Package {
	pkgList := m.currentPackageList()
	if m.selectedIndex >= 0 && m.selectedIndex < len(pkgList) {
		return &pkgList[m.selectedIndex]
	}
	return nil
}

// getRunner returns the model's injected CommandRunner, or the global runner if unset.
func (m *model) getRunner() CommandRunner {
	if m != nil && m.runner != nil {
		return m.runner
	}
	return runner
}

// getContext returns the model's lifecycle context, or context.Background() if unset.
func (m *model) getContext() context.Context {
	if m != nil && m.ctx != nil {
		return m.ctx
	}
	return context.Background()
}

// getTheme returns the model's active theme, or currentTheme if unset.
func (m *model) getTheme() Theme {
	if m != nil && m.theme.Name != "" {
		return m.theme
	}
	return currentTheme
}

// isReflectorInstalled returns whether reflector is installed, caching the result per overlay session.
func (m *model) isReflectorInstalled() bool {
	if m == nil {
		return false
	}
	if !m.reflectorCheckDone {
		m.reflectorInstalled = checkReflectorInstalled(m.getRunner())
		m.reflectorCheckDone = true
	}
	return m.reflectorInstalled
}

// Maximum entries allowed in detailsCache to prevent unbounded memory growth (S-10).
const maxDetailsCacheEntries = 500

// cachePackageDetails inserts or updates an entry in detailsCache with bounded FIFO eviction.
func (m *model) cachePackageDetails(pkgName, details string) {
	if m == nil {
		return
	}
	if m.detailsCache == nil {
		m.detailsCache = make(map[string]string)
	}
	if _, exists := m.detailsCache[pkgName]; !exists {
		for len(m.detailsCache) >= maxDetailsCacheEntries {
			if len(m.detailsCacheOrder) > 0 {
				oldest := m.detailsCacheOrder[0]
				m.detailsCacheOrder = m.detailsCacheOrder[1:]
				delete(m.detailsCache, oldest)
			} else {
				for k := range m.detailsCache {
					delete(m.detailsCache, k)
					break
				}
			}
		}
		m.detailsCacheOrder = append(m.detailsCacheOrder, pkgName)
	}
	m.detailsCache[pkgName] = details
}

// refreshAll triggers a full refresh of all system data
func (m *model) refreshAll() tea.Cmd {
	m.loading = true
	m.pendingUpdates = nil
	m.detailsCache = make(map[string]string)
	m.detailsCacheOrder = nil
	return tea.Batch(
		getDashboardDataWithContext(m.getContext(), &m.config, m.getRunner()),
		loadRepoPackagesWithContext(m.getContext(), m.getRunner()),
		getInstalledPackagesWithContext(m.getContext(), m.getRunner()),
		checkUpdatesWithContext(m.getContext(), &m.config, m.getRunner()),
	)
}

func (m *model) resetSearchState() {
	if m.aurCancelFunc != nil {
		m.aurCancelFunc()
		m.aurCancelFunc = nil
	}
	m.searchingAUR = false
	m.lastAURQuery = ""
	m.searchStatus = ""
	m.searchError = false
	m.searchTerm = ""
	m.textInput.SetValue("")
	m.lastQuery = ""
	m.aurPackages = nil
}

func (m *model) resetSelectionPanel() {
	m.markedPackages = make(map[string]bool)
	m.selectionPanelFocused = false
	m.selectionPanelIndex = 0
	m.selectionScrollOffset = 0
}

func (m *model) resetDetailsPane() {
	m.packageDetails = ""
	m.detailsForPackage = ""
	m.detailsScrollOffset = 0
	m.loadingDetails = false
}

func (m *model) resetConfirmation() {
	m.showConfirmation = false
	m.confirmPackages = nil
	m.confirmScrollOffset = 0
	m.maxConfirmScroll = 0
}

func (m *model) resetErrorOverlay() {
	m.showErrorOverlay = false
	m.errorTitle = ""
	m.errorMessage = ""
	m.errorDetails = ""
}

// resetState clears common state fields like search progress and package selections
func (m *model) resetState() {
	m.resetSearchState()
	m.resetSelectionPanel()
	m.resetDetailsPane()
	m.resetConfirmation()
	m.resetErrorOverlay()
	m.cacheToFree = 0
	m.selectedIndex = 0
	m.filtered = nil
	m.filteredInstalled = nil
	m.updateScrollOffset = 0
	m.maxUpdateScroll = 0
	m.cacheMenuIndex = 0
	m.updatePlaceholder()
	m.recalculateTextInputWidth()
}

// updatePlaceholder sets the text input placeholder based on the current mode
func (m *model) updatePlaceholder() {
	placeholder := "Search packages..."
	switch m.mode {
	case modeRemove:
		placeholder = "Filter installed packages..."
	case modeUpdate:
		placeholder = "Checking for updates..."
	case modeDashboard:
		placeholder = "View system dashboard"
	}
	m.textInput.Placeholder = placeholder
}

// switchToMode transitions the model to the target view mode and returns any initialization command.
func (m *model) switchToMode(target viewMode) tea.Cmd {
	switch target {
	case modeDashboard:
		m.mode = modeDashboard
		m.loading = true
		m.resetState()
		return getDashboardDataWithContext(m.getContext(), &m.config, m.getRunner())
	case modeInstall:
		if m.mode != modeInstall {
			m.mode = modeInstall
			m.resetState()
			m.textInput.Focus()
		}
		return nil
	case modeUpdate:
		m.mode = modeUpdate
		m.resetState()
		m.loading = true
		m.pendingUpdates = nil
		return syncRepositoriesInTerminal(m)
	case modeRemove:
		if m.mode != modeRemove {
			m.mode = modeRemove
			m.resetState()
			m.loading = true
			m.statusMessage = "Refreshing installed packages..."
			return getInstalledPackagesWithContext(m.getContext(), m.getRunner())
		}
		return nil
	}
	return nil
}
