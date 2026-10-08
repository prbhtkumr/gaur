package main

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model
type model struct {
	config                Config
	keys                  KeyMap
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
	mirrorConfig          MirrorConfig
	mirrorSelectedItem    MirrorOverlayItem
	mirrorUpdating        bool
	mirrorError           string
	mirrorProgressCurrent int // Current number of mirrors processed
	mirrorProgressTotal   int // Total number of mirrors to process
}

func initialModel(initialMode viewMode, cfg Config, tl *ThemeLoader) *model {
	ti := textinput.New()
	ti.CharLimit = textInputCharLimit
	ti.Width = textInputDefaultWidth

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(currentTheme.SpinnerColor)

	m := &model{
		config:         cfg,
		keys:           NewKeyMap(cfg.Keys),
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
		loadRepoPackages(),
		getInstalledPackages(),
		func() tea.Msg {
			switch m.mode {
			case modeDashboard:
				return getDashboardData(&m.config)()
			case modeUpdate:
				return checkUpdates(&m.config)()
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

// refreshAll triggers a full refresh of all system data
func (m *model) refreshAll() tea.Cmd {
	m.loading = true
	m.pendingUpdates = nil
	m.detailsCache = make(map[string]string)
	return tea.Batch(
		getDashboardData(&m.config),
		loadRepoPackages(),
		getInstalledPackages(),
		checkUpdates(&m.config),
	)
}

// resetState clears common state fields like search progress and package selections
func (m *model) resetState() {
	m.searchingAUR = false
	m.lastAURQuery = ""
	m.searchStatus = ""
	m.searchError = false
	m.searchTerm = ""
	m.markedPackages = make(map[string]bool)
	m.selectionPanelFocused = false
	m.selectionPanelIndex = 0
	m.selectionScrollOffset = 0
	m.cacheToFree = 0
	m.packageDetails = ""
	m.detailsForPackage = ""
	m.detailsScrollOffset = 0
	m.loadingDetails = false
	m.textInput.SetValue("")
	m.lastQuery = ""
	m.selectedIndex = 0
	m.aurPackages = nil
	m.filtered = nil
	m.filteredInstalled = nil
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
