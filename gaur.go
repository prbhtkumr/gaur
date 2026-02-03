package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// View modes for the TUI application
type viewMode int

const (
	modeInstall viewMode = iota
	modeInstalled
	modeUninstall
	modeUpdate
)

// Confirmation operation types
type confirmationType int

const (
	confirmInstall confirmationType = iota
	confirmUninstall
	confirmUpdate
	confirmCleanCache
	confirmRemoveOrphans
)

// Theme type for TUI theming
type themeType int

const (
	themeBasic themeType = iota
	themeCatppuccinMocha
)

// Theme holds all color definitions for the UI
type Theme struct {
	Name string

	// Base colors
	BorderColor     lipgloss.Color
	SelectedColor   lipgloss.Color
	TextColor       lipgloss.Color
	SubtleColor     lipgloss.Color
	TitleColor      lipgloss.Color

	// Mode colors
	InstallColor   lipgloss.Color
	InstalledColor lipgloss.Color
	UninstallColor lipgloss.Color
	UpdateColor    lipgloss.Color

	// Source colors
	CoreColor     lipgloss.Color
	ExtraColor    lipgloss.Color
	MultilibColor lipgloss.Color
	AurColor      lipgloss.Color

	// Status colors
	SuccessColor   lipgloss.Color
	WarningColor   lipgloss.Color
	ErrorColor     lipgloss.Color
	HighlightColor lipgloss.Color

	// Dashboard colors
	DashboardLabel   lipgloss.Color
	DashboardValue   lipgloss.Color
	DashboardWarning lipgloss.Color
	DashboardDesc    lipgloss.Color
}

// Available themes
var themes = map[themeType]Theme{
	themeBasic: {
		Name:            "Basic",
		BorderColor:     lipgloss.Color("62"),
		SelectedColor:   lipgloss.Color("170"),
		TextColor:       lipgloss.Color("252"),
		SubtleColor:     lipgloss.Color("241"),
		TitleColor:      lipgloss.Color("229"),
		InstallColor:    lipgloss.Color("39"),
		InstalledColor:  lipgloss.Color("213"),
		UninstallColor:  lipgloss.Color("196"),
		UpdateColor:     lipgloss.Color("46"),
		CoreColor:       lipgloss.Color("46"),
		ExtraColor:      lipgloss.Color("39"),
		MultilibColor:   lipgloss.Color("214"),
		AurColor:        lipgloss.Color("201"),
		SuccessColor:    lipgloss.Color("46"),
		WarningColor:    lipgloss.Color("226"),
		ErrorColor:      lipgloss.Color("196"),
		HighlightColor:  lipgloss.Color("226"),
		DashboardLabel:  lipgloss.Color("252"),
		DashboardValue:  lipgloss.Color("39"),
		DashboardWarning: lipgloss.Color("196"),
		DashboardDesc:   lipgloss.Color("241"),
	},
	themeCatppuccinMocha: {
		Name:            "Catppuccin Mocha",
		BorderColor:     lipgloss.Color("#6c7086"), // Overlay0
		SelectedColor:   lipgloss.Color("#cba6f7"), // Mauve
		TextColor:       lipgloss.Color("#cdd6f4"), // Text
		SubtleColor:     lipgloss.Color("#6c7086"), // Overlay0
		TitleColor:      lipgloss.Color("#f9e2af"), // Yellow
		InstallColor:    lipgloss.Color("#89b4fa"), // Blue
		InstalledColor:  lipgloss.Color("#f5c2e7"), // Pink
		UninstallColor:  lipgloss.Color("#f38ba8"), // Red
		UpdateColor:     lipgloss.Color("#a6e3a1"), // Green
		CoreColor:       lipgloss.Color("#a6e3a1"), // Green
		ExtraColor:      lipgloss.Color("#89b4fa"), // Blue
		MultilibColor:   lipgloss.Color("#fab387"), // Peach
		AurColor:        lipgloss.Color("#cba6f7"), // Mauve
		SuccessColor:    lipgloss.Color("#a6e3a1"), // Green
		WarningColor:    lipgloss.Color("#f9e2af"), // Yellow
		ErrorColor:      lipgloss.Color("#f38ba8"), // Red
		HighlightColor:  lipgloss.Color("#f9e2af"), // Yellow
		DashboardLabel:  lipgloss.Color("#cdd6f4"), // Text
		DashboardValue:  lipgloss.Color("#89dceb"), // Sky
		DashboardWarning: lipgloss.Color("#f38ba8"), // Red
		DashboardDesc:   lipgloss.Color("#a6adc8"), // Subtext0
	},
}

// Current active theme
var currentTheme = themes[themeCatppuccinMocha]

// setTheme changes the active theme and updates all styles
func setTheme(t themeType) {
	if theme, ok := themes[t]; ok {
		currentTheme = theme
		// Update all style variables
		defaultBorderColor = currentTheme.BorderColor
		selectedColor = currentTheme.SelectedColor
		modeColors = getModeColors()
		sourceColors = getSourceColors()

		baseTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(currentTheme.TitleColor).
			Padding(0, 1)

		selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(currentTheme.SelectedColor)

		normalStyle = lipgloss.NewStyle().
			Foreground(currentTheme.TextColor)

		infoStyle = lipgloss.NewStyle().
			Foreground(currentTheme.TextColor).
			Padding(1)

		statusStyle = lipgloss.NewStyle().
			Foreground(currentTheme.SubtleColor)

		helpStyle = lipgloss.NewStyle().
			Foreground(currentTheme.SubtleColor).
			Bold(true)

		installedBadge = lipgloss.NewStyle().
			Foreground(currentTheme.SuccessColor).
			Bold(true)

		matchHighlightStyle = lipgloss.NewStyle().
			Foreground(currentTheme.HighlightColor).
			Bold(true)

		dashboardLabelStyle = lipgloss.NewStyle().
			Foreground(currentTheme.DashboardLabel).
			Bold(true)

		dashboardValueStyle = lipgloss.NewStyle().
			Foreground(currentTheme.DashboardValue).
			Bold(true)

		dashboardWarningStyle = lipgloss.NewStyle().
			Foreground(currentTheme.DashboardWarning).
			Bold(true)

		dashboardDescStyle = lipgloss.NewStyle().
			Foreground(currentTheme.DashboardDesc)
	}
}

// getThemeByName returns a theme type by its name (case-insensitive)
func getThemeByName(name string) (themeType, bool) {
	nameLower := strings.ToLower(name)
	for t, theme := range themes {
		if strings.ToLower(theme.Name) == nameLower ||
			strings.ToLower(strings.ReplaceAll(theme.Name, " ", "-")) == nameLower ||
			strings.ToLower(strings.ReplaceAll(theme.Name, " ", "")) == nameLower {
			return t, true
		}
	}
	return themeBasic, false
}

// listThemes returns a list of available theme names
func listThemes() []string {
	var names []string
	for _, theme := range themes {
		names = append(names, theme.Name)
	}
	sort.Strings(names)
	return names
}

// UI configuration constants
const (
	minSearchQueryLen       = 2
	textInputCharLimit      = 100
	textInputDefaultWidth   = 50
	packageInfoDebounceTime = 150 * time.Millisecond
)

// Package represents a package with its source and name
type Package struct {
	Source      string // core, extra, multilib, aur
	Name        string
	Version     string
	Description string
	Installed   bool
	Explicit    bool // Explicitly installed (not a dependency)
	Orphan      bool // Orphan package (no longer required)
}

func (p Package) String() string {
	return fmt.Sprintf("%s/%s", p.Source, p.Name)
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

	// Use fzf --filter for non-interactive fuzzy filtering
	// -d '\t' -n2: only match on second field (package name), not the index
	// --tiebreak=begin,length: prefer matches at start and shorter names
	cmd := exec.Command("fzf", "--filter", query, "-d", "\t", "-n2", "--tiebreak=begin,length")
	cmd.Stdin = strings.NewReader(input.String())
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	_ = cmd.Run() // fzf returns error if no matches, that's ok

	// Parse output and rebuild package list
	var result []Package
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
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

	// If fzf found nothing, fall back to substring match
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

	pkgStr := pkg.Source + "/" + pkg.Name
	pkgLower := strings.ToLower(pkgStr)
	queryLower := strings.ToLower(query)

	var indices []int

	// Try to find consecutive substring match first (more visually coherent)
	if idx := strings.Index(pkgLower, queryLower); idx != -1 {
		for i := 0; i < len(queryLower); i++ {
			indices = append(indices, idx+i)
		}
		return indices
	}

	// Fall back to fuzzy matching: find each query character in order
	pkgRunes := []rune(pkgLower)
	queryRunes := []rune(queryLower)
	pkgIdx := 0
	for _, qr := range queryRunes {
		found := false
		for pkgIdx < len(pkgRunes) {
			if pkgRunes[pkgIdx] == qr {
				indices = append(indices, pkgIdx)
				pkgIdx++
				found = true
				break
			}
			pkgIdx++
		}
		if !found {
			// Query char not found, return partial matches
			break
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

// Messages
type repoPackagesMsg struct {
	packages []Package
	err      error
}

type aurSearchMsg struct {
	packages []Package
	query    string
	err      error
}

type packageInfoMsg struct {
	info        string
	packageName string
	err         error
}

type installedPackagesMsg struct {
	packages []Package
	err      error
}

type actionCompleteMsg struct {
	message string
	err     error
}

type updateOutputMsg struct {
	output string
	done   bool
	err    error
}

type updateCheckMsg struct {
	packages []Package
	err      error
}

type execCompleteMsg struct {
	operation confirmationType
	packages  []string
	err       error
}

type dashboardMsg struct {
	data DashboardData
	err  error
}

// debounceTickMsg is sent after debounce timer expires to trigger package info fetch
type debounceTickMsg struct {
	packageName string
}

// DashboardData holds system package statistics
type DashboardData struct {
	TotalPackages       int
	ExplicitlyInstalled int
	ForeignPackages     int
	TotalSize           string
	TotalSizeBytes      int64 // For comparison
	CleanerSize         string
	CleanerSizeBytes    int64 // For comparison and coloring
	PacmanCacheSize     string
	PacmanCacheSizeBytes int64
	PacmanCachePath     string
	ParuCacheSize       string
	ParuCacheSizeBytes  int64
	ParuCachePath       string
	Orphans             int
	MissingFromAUR      int
	TopPackages         []PackageSize // Top 10 packages by size
}

// PackageSize holds package name and its installed size
type PackageSize struct {
	Name string
	Size string
}

// Dashboard action messages
type cleanCacheMsg struct {
	output string
	err    error
}

type removeOrphansMsg struct {
	output string
	err    error
}

// Model
type model struct {
	textInput             textinput.Model
	repoPackages          []Package       // All repo packages from local cache
	aurPackages           []Package       // AUR packages from last search
	installedSet          map[string]bool // Quick lookup for installed packages
	packages              []Package
	filtered              []Package
	installed             []Package
	filteredInstalled     []Package
	matchIndices          map[int][]int // Maps package index to matched character indices
	installedMatchIndices map[int][]int
	selectedIndex         int
	markedPackages        map[string]bool // Packages marked for batch operation
	selectionPanelFocused bool            // Whether selection panel is focused
	selectionPanelIndex   int             // Selected index within selection panel
	packageInfo           string
	infoForPackage        string
	pendingInfoPackage    string // Package waiting for debounce to complete
	loadingInfo           bool
	mode                  viewMode
	width                 int
	height                int
	loading               bool
	statusMessage         string
	updateOutput          string
	lastQuery             string
	lastAURQuery          string // Last query sent to AUR search
	searchingAUR          bool   // Whether AUR search is in progress
	dashboard             DashboardData
	dashboardSelected     int // Selected item in dashboard (0=foreign, 1=cache, 2=orphans)
	// Confirmation dialog state
	showConfirmation      bool
	confirmType           confirmationType
	confirmPackages       []string  // Package names to operate on
	pendingUpdates        []Package // Updates available (for update confirmation)
	confirmScrollOffset   int       // Scroll offset for confirmation package list
	lastCompletedOp       string    // Description of last completed operation
	// Error overlay state
	showErrorOverlay      bool
	errorTitle            string
	errorMessage          string
	errorDetails          string
}

// getModeColors returns the mode colors based on current theme
func getModeColors() map[viewMode]lipgloss.Color {
	return map[viewMode]lipgloss.Color{
		modeInstall:   currentTheme.InstallColor,
		modeInstalled: currentTheme.InstalledColor,
		modeUninstall: currentTheme.UninstallColor,
		modeUpdate:    currentTheme.UpdateColor,
	}
}

// getSourceColors returns the source colors based on current theme
func getSourceColors() map[string]lipgloss.Color {
	return map[string]lipgloss.Color{
		"core":     currentTheme.CoreColor,
		"extra":    currentTheme.ExtraColor,
		"multilib": currentTheme.MultilibColor,
		"aur":      currentTheme.AurColor,
	}
}

// Styles - initialized with theme colors
var (
	defaultBorderColor = currentTheme.BorderColor
	selectedColor      = currentTheme.SelectedColor

	// Mode-specific colors for active view highlighting
	modeColors = getModeColors()

	sourceColors = getSourceColors()

	// Base styles (will be customized per mode in View)
	baseTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(currentTheme.TitleColor).
			Padding(0, 1)

	baseBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder())

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(currentTheme.SelectedColor)

	normalStyle = lipgloss.NewStyle().
			Foreground(currentTheme.TextColor)

	infoStyle = lipgloss.NewStyle().
			Foreground(currentTheme.TextColor).
			Padding(1)

	statusStyle = lipgloss.NewStyle().
			Foreground(currentTheme.SubtleColor)

	helpStyle = lipgloss.NewStyle().
			Foreground(currentTheme.SubtleColor).
			Bold(true)

	installedBadge = lipgloss.NewStyle().
			Foreground(currentTheme.SuccessColor).
			Bold(true)

	matchHighlightStyle = lipgloss.NewStyle().
				Foreground(currentTheme.HighlightColor).
				Bold(true)

	dashboardLabelStyle = lipgloss.NewStyle().
				Foreground(currentTheme.DashboardLabel).
				Bold(true)

	dashboardValueStyle = lipgloss.NewStyle().
				Foreground(currentTheme.DashboardValue).
				Bold(true)

	dashboardWarningStyle = lipgloss.NewStyle().
				Foreground(currentTheme.DashboardWarning).
				Bold(true)

	dashboardDescStyle = lipgloss.NewStyle().
				Foreground(currentTheme.DashboardDesc)
)

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Search packages..."
	ti.CharLimit = textInputCharLimit
	ti.Width = textInputDefaultWidth

	return model{
		textInput:      ti,
		repoPackages:   []Package{},
		installedSet:   make(map[string]bool),
		packages:       []Package{},
		filtered:       []Package{},
		installed:      []Package{},
		markedPackages: make(map[string]bool),
		selectedIndex:  0,
		mode:           modeInstall,
		loading:        true,
		statusMessage:  "Loading package database...",
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, loadRepoPackages())
}

// currentPackageList returns the appropriate package list based on current mode.
func (m model) currentPackageList() []Package {
	switch m.mode {
	case modeInstall:
		return m.filtered
	case modeUninstall:
		return m.filteredInstalled
	default:
		return nil
	}
}

// maxSelectableIndex returns the maximum valid index for the current package list.
func (m model) maxSelectableIndex() int {
	pkgList := m.currentPackageList()
	if len(pkgList) == 0 {
		return 0
	}
	return len(pkgList) - 1
}

// selectedPackage returns the currently selected package, or nil if none.
func (m model) selectedPackage() *Package {
	pkgList := m.currentPackageList()
	if m.selectedIndex >= 0 && m.selectedIndex < len(pkgList) {
		return &pkgList[m.selectedIndex]
	}
	return nil
}

// highlightMatches renders a string with matched character indices highlighted.
// It uses a map for O(1) lookup of matched positions and handles Unicode correctly.
func highlightMatches(s string, matchedIndices []int) string {
	if len(matchedIndices) == 0 {
		return s
	}

	// Create a set of matched indices for O(1) lookup
	matchSet := make(map[int]struct{}, len(matchedIndices))
	for _, idx := range matchedIndices {
		matchSet[idx] = struct{}{}
	}

	var result strings.Builder
	result.Grow(len(s) * 2) // Pre-allocate for efficiency
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
	
	// Get source color
	sourceColor, hasSourceColor := sourceColors[pkg.Source]
	
	// If no matches, just apply source coloring
	if len(matchedIndices) == 0 {
		if hasSourceColor {
			return lipgloss.NewStyle().Foreground(sourceColor).Render(pkg.Source) + "/" + pkg.Name
		}
		return pkgStr
	}

	// Create a set of matched indices for O(1) lookup
	matchSet := make(map[int]struct{}, len(matchedIndices))
	for _, idx := range matchedIndices {
		matchSet[idx] = struct{}{}
	}

	// Find where the slash is (end of source)
	slashIdx := len(pkg.Source)

	var result strings.Builder
	result.Grow(len(pkgStr) * 2)
	runes := []rune(pkgStr)
	
	for i, r := range runes {
		if _, matched := matchSet[i]; matched {
			// Matched character - use highlight color
			result.WriteString(matchHighlightStyle.Render(string(r)))
		} else if i < slashIdx && hasSourceColor {
			// Source portion (before slash) - use source color
			result.WriteString(lipgloss.NewStyle().Foreground(sourceColor).Render(string(r)))
		} else {
			// Name portion or no source color - use normal text
			result.WriteRune(r)
		}
	}
	return result.String()
}

// Commands
// loadRepoPackages loads all packages from local pacman database
func loadRepoPackages() tea.Cmd {
	return func() tea.Msg {
		// Get all repo packages: "repo name version"
		cmd := exec.Command("pacman", "-Sl")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		if err := cmd.Run(); err != nil {
			return repoPackagesMsg{err: err}
		}

		// Get installed packages for quick lookup
		installedCmd := exec.Command("pacman", "-Qq")
		var installedOut bytes.Buffer
		installedCmd.Stdout = &installedOut
		_ = installedCmd.Run()
		
		installedSet := make(map[string]bool)
		for _, name := range strings.Split(installedOut.String(), "\n") {
			name = strings.TrimSpace(name)
			if name != "" {
				installedSet[name] = true
			}
		}

		// Parse "repo name version [installed]" format
		var packages []Package
		for _, line := range strings.Split(stdout.String(), "\n") {
			parts := strings.Fields(line)
			if len(parts) < 3 {
				continue
			}
			pkg := Package{
				Source:    parts[0],
				Name:      parts[1],
				Version:   parts[2],
				Installed: installedSet[parts[1]] || (len(parts) > 3 && parts[3] == "[installed]"),
			}
			packages = append(packages, pkg)
		}

		return repoPackagesMsg{packages: packages}
	}
}

// Repo filter character mappings
var repoFilterChars = map[rune]string{
	'c': "core",
	'e': "extra",
	'm': "multilib",
	'a': "aur",
}

// uninstallFilterChars maps single characters to package filter types for uninstall mode
var uninstallFilterChars = map[rune]string{
	't': "total",    // All packages
	'e': "explicit", // Explicitly installed packages
	'f': "foreign",  // Foreign/AUR packages
	'o': "orphan",   // Orphan packages
}

// parseRepoFilter extracts repo filters and search query from input
// Supports combined filters like "ae:", "cem:", "aem:" in any order
// Returns (repoFilters, searchQuery) where repoFilters is empty if no filter specified
func parseRepoFilter(input string) (map[string]bool, string) {
	input = strings.TrimSpace(input)
	
	// Look for colon to identify filter prefix
	colonIdx := strings.Index(input, ":")
	if colonIdx == -1 {
		return nil, input
	}
	
	// Extract prefix before colon
	prefix := strings.ToLower(input[:colonIdx])
	searchQuery := strings.TrimSpace(input[colonIdx+1:])
	
	// Parse each character in prefix as a repo filter
	repoFilters := make(map[string]bool)
	for _, ch := range prefix {
		if repo, ok := repoFilterChars[ch]; ok {
			repoFilters[repo] = true
		}
	}
	
	// If no valid repo chars found, treat as regular search
	if len(repoFilters) == 0 {
		return nil, input
	}
	
	return repoFilters, searchQuery
}

// formatRepoFilters returns a human-readable string of active repo filters
func formatRepoFilters(filters map[string]bool) string {
	if len(filters) == 0 {
		return ""
	}
	var repos []string
	// Order consistently
	for _, repo := range []string{"core", "extra", "multilib", "aur"} {
		if filters[repo] {
			repos = append(repos, repo)
		}
	}
	return strings.Join(repos, "+")
}

// parseUninstallFilter extracts source filters and search query from input for uninstall mode
// Supports 'a:' for AUR/foreign packages and 'l:' for local/official packages
func parseUninstallFilter(input string) (map[string]bool, string) {
	input = strings.TrimSpace(input)
	
	// Look for colon to identify filter prefix
	colonIdx := strings.Index(input, ":")
	if colonIdx == -1 {
		return nil, input
	}
	
	// Extract prefix before colon
	prefix := strings.ToLower(input[:colonIdx])
	searchQuery := strings.TrimSpace(input[colonIdx+1:])
	
	// Parse each character in prefix as a source filter
	sourceFilters := make(map[string]bool)
	for _, ch := range prefix {
		if source, ok := uninstallFilterChars[ch]; ok {
			sourceFilters[source] = true
		}
	}
	
	// If no valid filter chars found, treat as regular search
	if len(sourceFilters) == 0 {
		return nil, input
	}
	
	return sourceFilters, searchQuery
}

// formatUninstallFilters returns a human-readable string of active uninstall filters
func formatUninstallFilters(filters map[string]bool) string {
	if len(filters) == 0 {
		return ""
	}
	var names []string
	if filters["total"] {
		names = append(names, "total")
	}
	if filters["explicit"] {
		names = append(names, "explicit")
	}
	if filters["foreign"] {
		names = append(names, "foreign")
	}
	if filters["orphan"] {
		names = append(names, "orphan")
	}
	return strings.Join(names, "+")
}

// filterAllPackages combines repo and AUR packages, then fuzzy filters together
// This ensures fzf ranks all packages by relevance to the query
// Supports repo filtering with prefixes: c (core), e (extra), m (multilib), a (aur)
// Filters can be combined: ae:, cem:, aem: etc.
func (m *model) filterAllPackages(query string) {
	if query == "" {
		m.filtered = []Package{}
		m.matchIndices = nil
		return
	}

	// Parse repo filter from query
	repoFilters, searchQuery := parseRepoFilter(query)
	
	// Combine repo and AUR packages
	allPackages := make([]Package, 0, len(m.repoPackages)+len(m.aurPackages))
	allPackages = append(allPackages, m.repoPackages...)
	allPackages = append(allPackages, m.aurPackages...)
	
	// Apply repo filters if specified
	if len(repoFilters) > 0 {
		var filtered []Package
		for _, pkg := range allPackages {
			if repoFilters[pkg.Source] {
				filtered = append(filtered, pkg)
			}
		}
		allPackages = filtered
	}
	
	if len(allPackages) == 0 {
		m.filtered = []Package{}
		m.matchIndices = nil
		return
	}

	// If only repo filter with no search query, show all from those repos
	if searchQuery == "" {
		m.filtered = allPackages
		m.matchIndices = nil
		return
	}
	
	// Fuzzy filter all packages together - fzf will rank by relevance
	m.filtered = fuzzyFilter(allPackages, searchQuery)
	
	// Compute match indices for highlighting (use searchQuery, not full query with prefix)
	m.matchIndices = computeAllMatchIndices(m.filtered, searchQuery)
}

// searchAUR searches the AUR via paru (network call)
func searchAUR(query string) tea.Cmd {
	return func() tea.Msg {
		if query == "" {
			return aurSearchMsg{packages: []Package{}, query: query}
		}

		// Search AUR only with paru -Ss --aur
		searchQuery := strings.ReplaceAll(query, " ", "-")
		cmd := exec.Command("paru", "-Ss", "-a", searchQuery)
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		_ = cmd.Run()

		if stdout.Len() == 0 {
			return aurSearchMsg{packages: []Package{}, query: query}
		}

		packages := parseAUROutput(stdout.String())
		return aurSearchMsg{packages: packages, query: query}
	}
}

// parseAUROutput parses paru -Ss output for AUR packages
func parseAUROutput(output string) []Package {
	var packages []Package
	lines := strings.Split(output, "\n")

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		// Skip empty lines and description lines (indented)
		if line == "" || strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}

		// Format: "aur/package version [+votes ~popularity] [Installed]"
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		repoPkg := strings.SplitN(parts[0], "/", 2)
		if len(repoPkg) != 2 {
			continue
		}

		pkg := Package{
			Source:    repoPkg[0],
			Name:      repoPkg[1],
			Version:   parts[1],
			Installed: strings.Contains(line, "[Installed"),
		}

		// Get description from next line
		if i+1 < len(lines) && (strings.HasPrefix(lines[i+1], " ") || strings.HasPrefix(lines[i+1], "\t")) {
			pkg.Description = strings.TrimSpace(lines[i+1])
		}

		packages = append(packages, pkg)
	}

	return packages
}

func parseSearchOutput(output string) []Package {
	var packages []Package
	lines := strings.Split(output, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// Package lines look like: " 1 aur/yay 12.5.7-1 [+2480 ~39.12]"
		// or without number: "aur/yay 12.5.7-1 [+2480 ~39.12]"
		if strings.Contains(line, "/") && !strings.HasPrefix(line, " ") || (len(line) > 0 && line[0] >= '0' && line[0] <= '9') {
			// Strip leading number if present (e.g., "1 aur/yay" -> "aur/yay")
			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}
			
			// Find the field containing "/" (source/name)
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
			if len(parts) < 2 {
				continue
			}

			source := parts[0]
			name := parts[1]

			// Version is the next field after source/name
			version := ""
			if pkgFieldIdx+1 < len(fields) {
				version = fields[pkgFieldIdx+1]
			}

			// Check if installed
			installed := strings.Contains(line, "[Installed")

			// Get description from next line
			description := ""
			if i+1 < len(lines) {
				description = strings.TrimSpace(lines[i+1])
				i++ // Skip description line
			}

			packages = append(packages, Package{
				Source:      source,
				Name:        name,
				Version:     version,
				Description: description,
				Installed:   installed,
			})
		}
	}

	return packages
}

// debouncePackageInfo returns a command that waits for the debounce duration
// then sends a tick message to trigger the actual fetch
func debouncePackageInfo(pkgName string) tea.Cmd {
	return tea.Tick(packageInfoDebounceTime, func(t time.Time) tea.Msg {
		return debounceTickMsg{packageName: pkgName}
	})
}

func getPackageInfo(pkg Package) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("paru", "-Si", pkg.Name)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		err := cmd.Run()
		if err != nil {
			return packageInfoMsg{info: "Failed to get package info", packageName: pkg.Name, err: err}
		}

		return packageInfoMsg{info: out.String(), packageName: pkg.Name}
	}
}

func getInstalledPackages() tea.Cmd {
	return func() tea.Msg {
		// Use pacman -Qi to get all installed package info including repository
		cmd := exec.Command("pacman", "-Qi")
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		err := cmd.Run()
		if err != nil {
			return installedPackagesMsg{err: err}
		}

		packages := parseInstalledPackages(out.String())
		return installedPackagesMsg{packages: packages}
	}
}

func parseInstalledPackages(output string) []Package {
	var packages []Package
	blocks := strings.Split(output, "\n\n")

	for _, block := range blocks {
		if strings.TrimSpace(block) == "" {
			continue
		}

		var pkg Package
		pkg.Installed = true
		pkg.Source = "local" // default

		lines := strings.Split(block, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					pkg.Name = strings.TrimSpace(parts[1])
				}
			} else if strings.HasPrefix(line, "Version") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					pkg.Version = strings.TrimSpace(parts[1])
				}
			} else if strings.HasPrefix(line, "Description") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					pkg.Description = strings.TrimSpace(parts[1])
				}
			}
		}

		if pkg.Name != "" {
			packages = append(packages, pkg)
		}
	}

	// Build a map of package name -> repository from pacman -Sl
	// This gives us the actual repo (core, extra, multilib) for installed packages
	repoMap := make(map[string]string)
	cmd := exec.Command("pacman", "-Sl")
	var repoOut bytes.Buffer
	cmd.Stdout = &repoOut
	if cmd.Run() == nil {
		for _, line := range strings.Split(repoOut.String(), "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				// Format: "repo name version [installed]"
				repoMap[parts[1]] = parts[0]
			}
		}
	}

	// Apply actual repository to installed packages
	for i := range packages {
		if repo, ok := repoMap[packages[i].Name]; ok {
			packages[i].Source = repo
		}
	}

	// Get foreign packages (AUR) to mark them
	cmd = exec.Command("pacman", "-Qm")
	var foreignOut bytes.Buffer
	cmd.Stdout = &foreignOut
	if cmd.Run() == nil {
		foreignPkgs := make(map[string]bool)
		for _, line := range strings.Split(foreignOut.String(), "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				foreignPkgs[parts[0]] = true
			}
		}
		for i := range packages {
			if foreignPkgs[packages[i].Name] {
				packages[i].Source = "aur"
			}
		}
	}

	// Get explicitly installed packages
	cmd = exec.Command("pacman", "-Qe")
	var explicitOut bytes.Buffer
	cmd.Stdout = &explicitOut
	if cmd.Run() == nil {
		explicitPkgs := make(map[string]bool)
		for _, line := range strings.Split(explicitOut.String(), "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				explicitPkgs[parts[0]] = true
			}
		}
		for i := range packages {
			packages[i].Explicit = explicitPkgs[packages[i].Name]
		}
	}

	// Get orphan packages
	cmd = exec.Command("pacman", "-Qdt")
	var orphanOut bytes.Buffer
	cmd.Stdout = &orphanOut
	if cmd.Run() == nil {
		orphanPkgs := make(map[string]bool)
		for _, line := range strings.Split(orphanOut.String(), "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				orphanPkgs[parts[0]] = true
			}
		}
		for i := range packages {
			packages[i].Orphan = orphanPkgs[packages[i].Name]
		}
	}

	return packages
}

func getDashboardData() tea.Cmd {
	return func() tea.Msg {
		var data DashboardData

		// Total Packages: paru -Q
		cmd := exec.Command("paru", "-Q")
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			data.TotalPackages = countLines(out.String())
		}

		// Explicitly Installed: paru -Qe
		out.Reset()
		cmd = exec.Command("paru", "-Qe")
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			data.ExplicitlyInstalled = countLines(out.String())
		}

		// Foreign Packages: paru -Qm
		out.Reset()
		cmd = exec.Command("paru", "-Qm")
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			data.ForeignPackages = countLines(out.String())
		}

		// Orphans: paru -Qdt
		out.Reset()
		cmd = exec.Command("paru", "-Qdt")
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			data.Orphans = countLines(out.String())
		}

		// Stats from paru -Ps (Total Size, Missing from AUR, Top 10 packages)
		out.Reset()
		cmd = exec.Command("paru", "-Ps")
		cmd.Stdout = &out
		if err := cmd.Run(); err == nil {
			data.TotalSize, data.TotalSizeBytes, data.MissingFromAUR, data.TopPackages = parseParuStats(out.String())
		}

		// Calculate Pacman Cache (System)
		pacmanCachePath := "/var/cache/pacman/pkg"
		pacmanCacheSize := calculateDirSize(pacmanCachePath)

		// Calculate Paru Cache (User)
		homeDir, _ := os.UserHomeDir()
		paruCachePath := filepath.Join(homeDir, ".cache", "paru")
		paruCacheSize := calculateDirSize(paruCachePath)

		// Store individual cache info
		data.PacmanCachePath = pacmanCachePath
		data.PacmanCacheSizeBytes = pacmanCacheSize
		data.PacmanCacheSize = formatBytes(pacmanCacheSize)
		data.ParuCachePath = paruCachePath
		data.ParuCacheSizeBytes = paruCacheSize
		data.ParuCacheSize = formatBytes(paruCacheSize)

		// Combine them for total
		totalCacheBytes := pacmanCacheSize + paruCacheSize

		data.CleanerSizeBytes = totalCacheBytes
		data.CleanerSize = formatBytes(totalCacheBytes)

		return dashboardMsg{data: data}
	}
}

func countLines(output string) int {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return 0
	}
	return len(lines)
}

// parseParuStats extracts total installed size, missing AUR package count,
// and top 10 biggest packages from paru -Ps output.
func parseParuStats(output string) (totalSize string, totalSizeBytes int64, missingAUR int, topPackages []PackageSize) {
	lines := strings.Split(output, "\n")
	inTopPackages := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for start of top packages section
		if strings.Contains(line, "biggest packages") {
			inTopPackages = true
			continue
		}

		// End of top packages section (separator line or empty)
		if inTopPackages && (strings.HasPrefix(line, "===") || line == "") {
			inTopPackages = false
			continue
		}

		// Parse top package lines (format: "package-name: 123.45 MiB")
		if inTopPackages {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				topPackages = append(topPackages, PackageSize{
					Name: strings.TrimSpace(parts[0]),
					Size: strings.TrimSpace(parts[1]),
				})
			}
			continue
		}

		if strings.Contains(line, "Total Size occupied") || strings.Contains(line, "Total Installed Size") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				totalSize = strings.TrimSpace(parts[1])
				totalSizeBytes = parseSizeToBytes(totalSize)
			}
		}
		if strings.Contains(line, "Missing") && strings.Contains(line, "AUR") {
			// Extract number from line like "Missing AUR Packages: 3"
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				_, _ = fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &missingAUR)
			}
		}
	}
	return
}

// parseSizeToBytes converts a human-readable size (e.g., "10.5 GiB") to bytes
func parseSizeToBytes(size string) int64 {
	size = strings.TrimSpace(size)
	var value float64
	var unit string
	_, _ = fmt.Sscanf(size, "%f %s", &value, &unit)
	
	unit = strings.ToLower(unit)
	switch {
	case strings.HasPrefix(unit, "kib") || strings.HasPrefix(unit, "kb"):
		return int64(value * 1024)
	case strings.HasPrefix(unit, "mib") || strings.HasPrefix(unit, "mb"):
		return int64(value * 1024 * 1024)
	case strings.HasPrefix(unit, "gib") || strings.HasPrefix(unit, "gb"):
		return int64(value * 1024 * 1024 * 1024)
	case strings.HasPrefix(unit, "tib") || strings.HasPrefix(unit, "tb"):
		return int64(value * 1024 * 1024 * 1024 * 1024)
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
			// If we can't read a specific file/dir, just skip it and continue
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
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// cleanCache runs paru -Sc to clean package cache
func cleanCache() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("paru", "-Sc", "--noconfirm")
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		err := cmd.Run()
		return cleanCacheMsg{output: out.String(), err: err}
	}
}

// removeOrphans runs paru -Rns to remove orphan packages
func removeOrphans() tea.Cmd {
	return func() tea.Msg {
		// First get the list of orphans
		cmd := exec.Command("paru", "-Qdtq")
		var orphanList bytes.Buffer
		cmd.Stdout = &orphanList
		if err := cmd.Run(); err != nil || orphanList.Len() == 0 {
			return removeOrphansMsg{output: "No orphans to remove", err: nil}
		}
		
		// Remove them
		orphans := strings.Fields(orphanList.String())
		args := append([]string{"-Rns", "--noconfirm"}, orphans...)
		cmd = exec.Command("paru", args...)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		err := cmd.Run()
		return removeOrphansMsg{output: out.String(), err: err}
	}
}

func installPackage(pkg Package) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("paru", "-S", "--noconfirm", pkg.Name)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		err := cmd.Run()
		if err != nil {
			return actionCompleteMsg{
				message: fmt.Sprintf("Failed to install %s: %s", pkg.Name, out.String()),
				err:     err,
			}
		}

		return actionCompleteMsg{
			message: fmt.Sprintf("Successfully installed %s", pkg.Name),
		}
	}
}

func installMultiplePackages(pkgNames []string) tea.Cmd {
	return func() tea.Msg {
		args := append([]string{"-S", "--noconfirm"}, pkgNames...)
		cmd := exec.Command("paru", args...)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		err := cmd.Run()
		if err != nil {
			return actionCompleteMsg{
				message: fmt.Sprintf("Failed to install packages: %s", out.String()),
				err:     err,
			}
		}

		return actionCompleteMsg{
			message: fmt.Sprintf("Successfully installed %d packages", len(pkgNames)),
		}
	}
}

func uninstallPackage(pkg Package) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("paru", "-Rns", "--noconfirm", pkg.Name)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		err := cmd.Run()
		if err != nil {
			return actionCompleteMsg{
				message: fmt.Sprintf("Failed to uninstall %s: %s", pkg.Name, out.String()),
				err:     err,
			}
		}

		return actionCompleteMsg{
			message: fmt.Sprintf("Successfully uninstalled %s", pkg.Name),
		}
	}
}

func uninstallMultiplePackages(pkgNames []string) tea.Cmd {
	return func() tea.Msg {
		args := append([]string{"-Rns", "--noconfirm"}, pkgNames...)
		cmd := exec.Command("paru", args...)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		err := cmd.Run()
		if err != nil {
			return actionCompleteMsg{
				message: fmt.Sprintf("Failed to uninstall packages: %s", out.String()),
				err:     err,
			}
		}

		return actionCompleteMsg{
			message: fmt.Sprintf("Successfully uninstalled %d packages", len(pkgNames)),
		}
	}
}

func updateSystem() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("paru", "-Syu", "--noconfirm")
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out

		err := cmd.Run()
		output := out.String()

		if err != nil {
			return updateOutputMsg{
				output: output,
				done:   true,
				err:    err,
			}
		}

		return updateOutputMsg{
			output: output,
			done:   true,
		}
	}
}

// checkUpdates fetches available updates using paru -Qu
func checkUpdates() tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("paru", "-Qu")
		var stdout bytes.Buffer
		cmd.Stdout = &stdout
		_ = cmd.Run() // Returns error if no updates, that's ok

		var packages []Package
		for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
			if line == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				pkg := Package{
					Name:    parts[0],
					Version: strings.Join(parts[1:], " "), // "oldver -> newver" format
				}
				// Determine source (foreign = aur)
				checkCmd := exec.Command("pacman", "-Qq", parts[0])
				if checkCmd.Run() == nil {
					// Check if foreign
					foreignCmd := exec.Command("pacman", "-Qm", parts[0])
					if foreignCmd.Run() == nil {
						pkg.Source = "aur"
					} else {
						pkg.Source = "repo"
					}
				}
				packages = append(packages, pkg)
			}
		}
		return updateCheckMsg{packages: packages}
	}
}

// executeInstallInTerminal runs paru -S interactively using tea.ExecProcess
func executeInstallInTerminal(packages []string) tea.Cmd {
	args := append([]string{"-S"}, packages...)
	c := exec.Command("paru", args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmInstall, packages: packages, err: err}
	})
}

// executeUninstallInTerminal runs paru -Rns interactively using tea.ExecProcess
func executeUninstallInTerminal(packages []string) tea.Cmd {
	args := append([]string{"-Rns"}, packages...)
	c := exec.Command("paru", args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmUninstall, packages: packages, err: err}
	})
}

// executeUpdateInTerminal runs paru -Syu interactively using tea.ExecProcess
func executeUpdateInTerminal() tea.Cmd {
	c := exec.Command("paru", "-Syu")
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmUpdate, err: err}
	})
}

// executeCleanCacheInTerminal runs paru -Sc interactively using tea.ExecProcess
func executeCleanCacheInTerminal() tea.Cmd {
	c := exec.Command("paru", "-Sc")
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmCleanCache, err: err}
	})
}

// executeRemoveOrphansInTerminal runs paru -Rns $(paru -Qdtq) interactively using tea.ExecProcess
func executeRemoveOrphansInTerminal(orphans []string) tea.Cmd {
	args := append([]string{"-Rns"}, orphans...)
	c := exec.Command("paru", args...)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return execCompleteMsg{operation: confirmRemoveOrphans, packages: orphans, err: err}
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// ctrl+c always quits
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

		// Handle error overlay dismissal
		if m.showErrorOverlay {
			if msg.String() == "esc" || msg.String() == "enter" || msg.String() == "q" {
				m.showErrorOverlay = false
				m.errorTitle = ""
				m.errorMessage = ""
				m.errorDetails = ""
				return m, nil
			}
			return m, nil
		}

		// Handle confirmation dialog keys
		if m.showConfirmation {
			switch msg.String() {
			case "y", "Y", "enter":
				m.showConfirmation = false
				m.confirmScrollOffset = 0
				switch m.confirmType {
				case confirmInstall:
					m.statusMessage = fmt.Sprintf("Installing %d package(s)...", len(m.confirmPackages))
					return m, executeInstallInTerminal(m.confirmPackages)
				case confirmUninstall:
					m.statusMessage = fmt.Sprintf("Removing %d package(s)...", len(m.confirmPackages))
					return m, executeUninstallInTerminal(m.confirmPackages)
				case confirmUpdate:
					m.statusMessage = "Running system update..."
					return m, executeUpdateInTerminal()
				case confirmCleanCache:
					m.statusMessage = "Cleaning package cache..."
					return m, executeCleanCacheInTerminal()
				case confirmRemoveOrphans:
					m.statusMessage = fmt.Sprintf("Removing %d orphan package(s)...", len(m.confirmPackages))
					orphans := m.confirmPackages
					m.confirmPackages = nil
					return m, executeRemoveOrphansInTerminal(orphans)
				}
			case "n", "N", "esc":
				m.showConfirmation = false
				m.confirmPackages = nil
				m.pendingUpdates = nil
				m.confirmScrollOffset = 0
				m.statusMessage = "Operation cancelled"
				return m, nil
			case "down", "j":
				// Scroll down in package list
				maxScroll := len(m.confirmPackages) - 10
				if m.confirmType == confirmUpdate {
					maxScroll = len(m.pendingUpdates) - 10
				}
				if maxScroll < 0 {
					maxScroll = 0
				}
				if m.confirmScrollOffset < maxScroll {
					m.confirmScrollOffset++
				}
				return m, nil
			case "up", "k":
				// Scroll up in package list
				if m.confirmScrollOffset > 0 {
					m.confirmScrollOffset--
				}
				return m, nil
			}
			return m, nil
		}

		// Handle * key to toggle selection panel focus
		if msg.String() == "*" {
			if len(m.markedPackages) > 0 {
				m.selectionPanelFocused = !m.selectionPanelFocused
				if m.selectionPanelFocused {
					m.textInput.Blur()
					m.selectionPanelIndex = 0
					m.statusMessage = "Selection panel: [↑↓] navigate  [tab] deselect  [enter] install  [*] close"
				} else {
					m.statusMessage = fmt.Sprintf("%d packages marked", len(m.markedPackages))
				}
			}
			return m, nil
		}

		// When selection panel is focused, handle its navigation
		if m.selectionPanelFocused {
			// Get sorted package names (same order as displayed)
			var pkgNames []string
			for name := range m.markedPackages {
				pkgNames = append(pkgNames, name)
			}
			sort.Strings(pkgNames)
			maxIdx := len(pkgNames) - 1
			if maxIdx > 9 {
				maxIdx = 9 // Match maxDisplay limit
			}

			switch msg.String() {
			case "esc", "*":
				m.selectionPanelFocused = false
				m.statusMessage = fmt.Sprintf("%d packages marked", len(m.markedPackages))
				return m, nil
			case "up", "k":
				if m.selectionPanelIndex > 0 {
					m.selectionPanelIndex--
				}
				return m, nil
			case "down", "j":
				if m.selectionPanelIndex < maxIdx {
					m.selectionPanelIndex++
				}
				return m, nil
			case "tab":
				// Deselect the highlighted package
				if m.selectionPanelIndex < len(pkgNames) {
					nameToRemove := pkgNames[m.selectionPanelIndex]
					delete(m.markedPackages, nameToRemove)
					// Adjust index if needed
					if m.selectionPanelIndex >= len(m.markedPackages) && m.selectionPanelIndex > 0 {
						m.selectionPanelIndex--
					}
					// Close panel if no more selections
					if len(m.markedPackages) == 0 {
						m.selectionPanelFocused = false
						m.statusMessage = "All selections cleared"
					} else {
						m.statusMessage = fmt.Sprintf("%d packages marked - [tab] to deselect", len(m.markedPackages))
					}
				}
				return m, nil
			case "enter":
				// Close panel and show confirmation dialog
				m.selectionPanelFocused = false
				if len(m.markedPackages) > 0 {
					if m.mode == modeInstall {
						var pkgsToInstall []string
						for name := range m.markedPackages {
							if !m.installedSet[name] {
								pkgsToInstall = append(pkgsToInstall, name)
							}
						}
						if len(pkgsToInstall) > 0 {
							sort.Strings(pkgsToInstall)
							m.showConfirmation = true
							m.confirmType = confirmInstall
							m.confirmPackages = pkgsToInstall
							m.confirmScrollOffset = 0
							m.markedPackages = make(map[string]bool)
							m.statusMessage = "Confirm installation"
						} else {
							m.statusMessage = "All marked packages are already installed"
						}
					} else if m.mode == modeUninstall {
						var pkgsToUninstall []string
						for name := range m.markedPackages {
							pkgsToUninstall = append(pkgsToUninstall, name)
						}
						sort.Strings(pkgsToUninstall)
						m.showConfirmation = true
						m.confirmType = confirmUninstall
						m.confirmPackages = pkgsToUninstall
						m.confirmScrollOffset = 0
						m.markedPackages = make(map[string]bool)
						m.statusMessage = "Confirm removal"
					}
				}
				return m, nil
			}
			return m, nil
		}

		// When input is focused, only allow esc, arrow keys, and typing
		if m.textInput.Focused() {
			switch msg.String() {
			case "esc":
				m.textInput.Blur()
				return m, nil
			case "down":
				// Down moves toward more relevant (lower index, visually down)
				if m.selectedIndex > 0 {
					m.selectedIndex--
					if m.mode == modeInstall && len(m.filtered) > 0 {
						m.loadingInfo = true
						m.pendingInfoPackage = m.filtered[m.selectedIndex].Name
						return m, debouncePackageInfo(m.pendingInfoPackage)
					} else if m.mode == modeUninstall && len(m.filteredInstalled) > 0 {
						m.loadingInfo = true
						m.pendingInfoPackage = m.filteredInstalled[m.selectedIndex].Name
						return m, debouncePackageInfo(m.pendingInfoPackage)
					}
				}
				return m, nil
			case "up":
				// Up moves toward less relevant (higher index, visually up)
				maxIndex := 0
				if m.mode == modeInstall {
					maxIndex = len(m.filtered) - 1
				} else if m.mode == modeUninstall {
					maxIndex = len(m.filteredInstalled) - 1
				}
				if m.selectedIndex < maxIndex {
					m.selectedIndex++
					if m.mode == modeInstall && len(m.filtered) > 0 {
						m.loadingInfo = true
						m.pendingInfoPackage = m.filtered[m.selectedIndex].Name
						return m, debouncePackageInfo(m.pendingInfoPackage)
					} else if m.mode == modeUninstall && len(m.filteredInstalled) > 0 {
						m.loadingInfo = true
						m.pendingInfoPackage = m.filteredInstalled[m.selectedIndex].Name
						return m, debouncePackageInfo(m.pendingInfoPackage)
					}
				}
				return m, nil
			case "enter":
				if m.mode == modeInstall && len(m.filtered) > 0 {
					// If packages are marked, show confirmation for all marked packages
					if len(m.markedPackages) > 0 {
						var pkgsToInstall []string
						for name := range m.markedPackages {
							if !m.installedSet[name] {
								pkgsToInstall = append(pkgsToInstall, name)
							}
						}
						if len(pkgsToInstall) > 0 {
							sort.Strings(pkgsToInstall)
							m.showConfirmation = true
							m.confirmType = confirmInstall
							m.confirmPackages = pkgsToInstall
							m.confirmScrollOffset = 0
							m.markedPackages = make(map[string]bool)
							m.statusMessage = "Confirm installation"
						} else {
							m.statusMessage = "All marked packages are already installed"
						}
					} else {
						// Show confirmation dialog for single package
						pkg := m.filtered[m.selectedIndex]
						if !pkg.Installed {
							m.showConfirmation = true
							m.confirmType = confirmInstall
							m.confirmPackages = []string{pkg.Name}
							m.confirmScrollOffset = 0
							m.statusMessage = "Confirm installation"
						} else {
							m.statusMessage = fmt.Sprintf("%s is already installed", pkg.Name)
						}
					}
				} else if m.mode == modeUninstall && len(m.filteredInstalled) > 0 {
					// If packages are marked, show confirmation for all marked packages
					if len(m.markedPackages) > 0 {
						var pkgsToUninstall []string
						for name := range m.markedPackages {
							pkgsToUninstall = append(pkgsToUninstall, name)
						}
						sort.Strings(pkgsToUninstall)
						m.showConfirmation = true
						m.confirmType = confirmUninstall
						m.confirmPackages = pkgsToUninstall
						m.confirmScrollOffset = 0
						m.markedPackages = make(map[string]bool)
						m.statusMessage = "Confirm removal"
					} else {
						// Show confirmation dialog for single package
						pkg := m.filteredInstalled[m.selectedIndex]
						m.showConfirmation = true
						m.confirmType = confirmUninstall
						m.confirmPackages = []string{pkg.Name}
						m.confirmScrollOffset = 0
						m.statusMessage = "Confirm removal"
					}
				}
				return m, nil
			case "tab":
				// Toggle mark on current package (works even while typing)
				if m.mode == modeInstall && len(m.filtered) > 0 {
					pkg := m.filtered[m.selectedIndex]
					if m.markedPackages[pkg.Name] {
						delete(m.markedPackages, pkg.Name)
					} else {
						m.markedPackages[pkg.Name] = true
					}
					markedCount := len(m.markedPackages)
					if markedCount > 0 {
						m.statusMessage = fmt.Sprintf("%d packages marked", markedCount)
					} else {
						m.statusMessage = fmt.Sprintf("Found %d packages", len(m.filtered))
					}
				} else if m.mode == modeUninstall && len(m.filteredInstalled) > 0 {
					pkg := m.filteredInstalled[m.selectedIndex]
					if m.markedPackages[pkg.Name] {
						delete(m.markedPackages, pkg.Name)
					} else {
						m.markedPackages[pkg.Name] = true
					}
					markedCount := len(m.markedPackages)
					if markedCount > 0 {
						m.statusMessage = fmt.Sprintf("%d packages marked", markedCount)
					} else {
						m.statusMessage = fmt.Sprintf("%d installed packages", len(m.installed))
					}
				}
				return m, nil
			}
			// All other keys go to text input
			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)
			cmds = append(cmds, cmd)
			// Handle filtering logic
			if m.mode == modeInstall {
				query := m.textInput.Value()
				if query != m.lastQuery {
					m.lastQuery = query
					
					// Parse repo filter to check query length correctly
					repoFilters, searchQuery := parseRepoFilter(query)
					effectiveQueryLen := len(searchQuery)
					
					// Allow filtering with just repo prefix (e.g., "a:" shows all AUR)
					hasRepoFilter := len(repoFilters) > 0
					
					if effectiveQueryLen >= minSearchQueryLen || hasRepoFilter {
						// Fuzzy filter combined repo + AUR packages (also computes match indices)
						m.filterAllPackages(query)
						m.selectedIndex = 0
						
						// Trigger AUR search only if:
						// 1. No repo filter OR filter includes AUR
						// 2. Have a search query (not just "a:")
						// 3. Haven't searched this query yet
						includesAUR := len(repoFilters) == 0 || repoFilters["aur"]
						shouldSearchAUR := includesAUR && 
							effectiveQueryLen >= minSearchQueryLen &&
							searchQuery != m.lastAURQuery
						
						if shouldSearchAUR {
							m.lastAURQuery = searchQuery
							m.searchingAUR = true
							cmds = append(cmds, searchAUR(searchQuery))
						}
						
						if len(m.filtered) > 0 {
							status := fmt.Sprintf("Found %d packages", len(m.filtered))
							if hasRepoFilter {
								status = fmt.Sprintf("Found %d %s packages", len(m.filtered), formatRepoFilters(repoFilters))
							}
							if m.searchingAUR {
								status += " (searching AUR...)"
							}
							m.statusMessage = status
							m.loadingInfo = true
							m.infoForPackage = m.filtered[0].Name
							cmds = append(cmds, getPackageInfo(m.filtered[0]))
						} else {
							if m.searchingAUR {
								m.statusMessage = "Searching AUR..."
							} else if hasRepoFilter && searchQuery == "" {
								m.statusMessage = fmt.Sprintf("No packages in %s", formatRepoFilters(repoFilters))
							} else {
								m.statusMessage = fmt.Sprintf("No matches for '%s'", query)
							}
							m.packageInfo = ""
							m.infoForPackage = ""
						}
					} else {
						m.filtered = []Package{}
						m.aurPackages = []Package{}
						m.lastAURQuery = ""
						m.packageInfo = ""
						m.infoForPackage = ""
						m.matchIndices = nil
						if len(m.repoPackages) > 0 {
							m.statusMessage = fmt.Sprintf("Type at least %d chars or use  to filter (c: e: m: a:) (%d repo packages)", minSearchQueryLen, len(m.repoPackages))
						} else {
							m.statusMessage = "Loading package database..."
						}
					}
				}
			} else if m.mode == modeUninstall {
				query := m.textInput.Value()
				if len(m.installed) > 0 {
					if query == "" {
						m.filteredInstalled = m.installed
						m.installedMatchIndices = nil
						m.statusMessage = fmt.Sprintf("%d installed packages", len(m.installed))
					} else {
						// Parse source filter from query
						sourceFilters, searchQuery := parseUninstallFilter(query)
						hasSourceFilter := len(sourceFilters) > 0
						
						// Start with all installed packages
						basePackages := m.installed
						
						// Apply source filters if specified
						if hasSourceFilter {
							var filtered []Package
							for _, pkg := range basePackages {
								// 't' (total) - all packages
								if sourceFilters["total"] {
									filtered = append(filtered, pkg)
								} else {
									// 'e' (explicit) - explicitly installed packages
									if sourceFilters["explicit"] && pkg.Explicit {
										filtered = append(filtered, pkg)
									}
									// 'f' (foreign) - foreign/AUR packages
									if sourceFilters["foreign"] && pkg.Source == "aur" {
										filtered = append(filtered, pkg)
									}
									// 'o' (orphan) - orphan packages
									if sourceFilters["orphan"] && pkg.Orphan {
										filtered = append(filtered, pkg)
									}
								}
							}
							basePackages = filtered
						}
						
						// Apply fuzzy filtering if there's a search query
						if searchQuery != "" {
							m.filteredInstalled = fuzzyFilter(basePackages, searchQuery)
							m.installedMatchIndices = computeAllMatchIndices(m.filteredInstalled, searchQuery)
						} else {
							m.filteredInstalled = basePackages
							m.installedMatchIndices = nil
						}
						
						// Update status message
						if hasSourceFilter {
							m.statusMessage = fmt.Sprintf("Found %d %s packages", len(m.filteredInstalled), formatUninstallFilters(sourceFilters))
						} else {
							m.statusMessage = fmt.Sprintf("Showing %d of %d packages", len(m.filteredInstalled), len(m.installed))
						}
					}
					if m.selectedIndex >= len(m.filteredInstalled) {
						m.selectedIndex = 0
					}
					if len(m.filteredInstalled) > 0 && m.filteredInstalled[m.selectedIndex].Name != m.infoForPackage {
						m.loadingInfo = true
						m.infoForPackage = m.filteredInstalled[m.selectedIndex].Name
						cmds = append(cmds, getPackageInfo(m.filteredInstalled[m.selectedIndex]))
					}
				}
			}
			return m, tea.Batch(cmds...)
		}

		// Input not focused - handle normal keybindings
		switch msg.String() {
		case "q":
			return m, tea.Quit

		case "esc":
			if m.textInput.Focused() {
				m.textInput.Blur()
				return m, nil
			}
			// Clear selections and reset state but stay in current mode
			if len(m.markedPackages) > 0 {
				m.markedPackages = make(map[string]bool)
				m.statusMessage = "Selections cleared"
				return m, nil
			}

		case "c":
			// Clean cache - only in dashboard mode
			if m.mode == modeInstalled && !m.loading {
				m.showConfirmation = true
				m.confirmType = confirmCleanCache
				m.confirmScrollOffset = 0
				m.statusMessage = "Confirm cache cleaning"
				return m, nil
			}

		case "R":
			// Remove orphans - only in dashboard mode and when there are orphans
			if m.mode == modeInstalled && !m.loading && m.dashboard.Orphans > 0 {
				// Get orphan list for confirmation
				cmd := exec.Command("paru", "-Qdtq")
				var orphanList bytes.Buffer
				cmd.Stdout = &orphanList
				if err := cmd.Run(); err == nil && orphanList.Len() > 0 {
					orphans := strings.Fields(orphanList.String())
					m.confirmPackages = orphans
					m.showConfirmation = true
					m.confirmType = confirmRemoveOrphans
					m.confirmScrollOffset = 0
					m.statusMessage = "Confirm orphan removal"
				}
				return m, nil
			}

		case "t":
			// Switch to remove mode with total filter - only from dashboard
			if m.mode == modeInstalled && !m.loading {
				m.mode = modeUninstall
				m.loading = true
				m.statusMessage = "Loading all packages..."
				m.selectedIndex = 0
				m.textInput.SetValue("t:")
				m.textInput.Placeholder = "Filter (t: total  e: explicit  f: foreign  o: orphan)..."
				m.markedPackages = make(map[string]bool)
				return m, getInstalledPackages()
			}

		case "e":
			// Switch to remove mode with explicit filter - only from dashboard
			if m.mode == modeInstalled && !m.loading {
				m.mode = modeUninstall
				m.loading = true
				m.statusMessage = "Loading explicit packages..."
				m.selectedIndex = 0
				m.textInput.SetValue("e:")
				m.textInput.Placeholder = "Filter (t: total  e: explicit  f: foreign  o: orphan)..."
				m.markedPackages = make(map[string]bool)
				return m, getInstalledPackages()
			}

		case "f":
			// Switch to remove mode with foreign filter - only from dashboard
			if m.mode == modeInstalled && !m.loading {
				m.mode = modeUninstall
				m.loading = true
				m.statusMessage = "Loading foreign packages..."
				m.selectedIndex = 0
				m.textInput.SetValue("f:")
				m.textInput.Placeholder = "Filter (t: total  e: explicit  f: foreign  o: orphan)..."
				m.markedPackages = make(map[string]bool)
				return m, getInstalledPackages()
			}

		case "o":
			// Switch to remove mode with orphan filter - only from dashboard
			if m.mode == modeInstalled && !m.loading {
				m.mode = modeUninstall
				m.loading = true
				m.statusMessage = "Loading orphan packages..."
				m.selectedIndex = 0
				m.textInput.SetValue("o:")
				m.textInput.Placeholder = "Filter (t: total  e: explicit  f: foreign  o: orphan)..."
				m.markedPackages = make(map[string]bool)
				return m, getInstalledPackages()
			}

		case "n":
			if m.mode != modeInstalled && !m.textInput.Focused() {
				m.mode = modeInstalled
				m.loading = true
				m.statusMessage = "Loading system statistics..."
				m.markedPackages = make(map[string]bool)
				return m, getDashboardData()
			}

		case "r":
			if m.mode != modeUninstall {
				m.mode = modeUninstall
				m.loading = true
				m.statusMessage = "Loading installed packages..."
				m.selectedIndex = 0
				m.textInput.SetValue("")
				m.textInput.Placeholder = "Filter (t: total  e: explicit  f: foreign  o: orphan)..."
				m.markedPackages = make(map[string]bool)
				return m, getInstalledPackages()
			}

		case "u":
			if !m.textInput.Focused() {
				if m.mode != modeUpdate {
					// Switch to update mode
					m.mode = modeUpdate
					m.markedPackages = make(map[string]bool)
				}
				// Check for updates (works both when switching to update mode and when already in it)
				m.loading = true
				m.statusMessage = "Checking for updates..."
				m.updateOutput = ""
				m.pendingUpdates = nil
				return m, checkUpdates()
			}

		case "i":
			if m.mode != modeInstall {
				m.mode = modeInstall
				m.selectedIndex = 0
				m.filtered = []Package{}
				m.packageInfo = ""
				m.statusMessage = "Press [/] to search packages"
				m.textInput.SetValue("")
				m.textInput.Placeholder = "Search packages..."
				m.markedPackages = make(map[string]bool)
				return m, nil
			}

		case "down", "j":
			// Down/j moves toward more relevant (lower index, visually down)
			if m.selectedIndex > 0 {
				m.selectedIndex--
				if m.mode == modeInstall && len(m.filtered) > 0 {
					m.loadingInfo = true
					m.pendingInfoPackage = m.filtered[m.selectedIndex].Name
					return m, debouncePackageInfo(m.pendingInfoPackage)
				} else if m.mode == modeUninstall && len(m.filteredInstalled) > 0 {
					m.loadingInfo = true
					m.pendingInfoPackage = m.filteredInstalled[m.selectedIndex].Name
					return m, debouncePackageInfo(m.pendingInfoPackage)
				}
			}

		case "up", "k":
			// Up/k moves toward less relevant (higher index, visually up)
			maxIndex := 0
			if m.mode == modeInstall {
				maxIndex = len(m.filtered) - 1
			} else if m.mode == modeUninstall {
				maxIndex = len(m.filteredInstalled) - 1
			}
			if m.selectedIndex < maxIndex {
				m.selectedIndex++
				if m.mode == modeInstall && len(m.filtered) > 0 {
					m.loadingInfo = true
					m.pendingInfoPackage = m.filtered[m.selectedIndex].Name
					return m, debouncePackageInfo(m.pendingInfoPackage)
				} else if m.mode == modeUninstall && len(m.filteredInstalled) > 0 {
					m.loadingInfo = true
					m.pendingInfoPackage = m.filteredInstalled[m.selectedIndex].Name
					return m, debouncePackageInfo(m.pendingInfoPackage)
				}
			}

		case "enter":
			if m.mode == modeInstall && len(m.filtered) > 0 {
				// If packages are marked, show confirmation for all marked packages
				if len(m.markedPackages) > 0 {
					var pkgsToInstall []string
					for name := range m.markedPackages {
						// Check if not already installed
						if !m.installedSet[name] {
							pkgsToInstall = append(pkgsToInstall, name)
						}
					}
					if len(pkgsToInstall) > 0 {
						sort.Strings(pkgsToInstall)
						m.showConfirmation = true
						m.confirmType = confirmInstall
						m.confirmPackages = pkgsToInstall
						m.confirmScrollOffset = 0
						m.markedPackages = make(map[string]bool) // Clear marks
						m.statusMessage = "Confirm installation"
					} else {
						m.statusMessage = "All marked packages are already installed"
					}
				} else {
					// Show confirmation for single selected package
					pkg := m.filtered[m.selectedIndex]
					if !pkg.Installed {
						m.showConfirmation = true
						m.confirmType = confirmInstall
						m.confirmPackages = []string{pkg.Name}
						m.confirmScrollOffset = 0
						m.statusMessage = "Confirm installation"
					} else {
						m.statusMessage = fmt.Sprintf("%s is already installed", pkg.Name)
					}
				}
			} else if m.mode == modeUninstall && len(m.filteredInstalled) > 0 {
				// If packages are marked, show confirmation for all marked packages
				if len(m.markedPackages) > 0 {
					var pkgsToUninstall []string
					for name := range m.markedPackages {
						pkgsToUninstall = append(pkgsToUninstall, name)
					}
					sort.Strings(pkgsToUninstall)
					m.showConfirmation = true
					m.confirmType = confirmUninstall
					m.confirmPackages = pkgsToUninstall
					m.confirmScrollOffset = 0
					m.markedPackages = make(map[string]bool) // Clear marks
					m.statusMessage = "Confirm removal"
				} else {
					// Show confirmation for single selected package
					pkg := m.filteredInstalled[m.selectedIndex]
					m.showConfirmation = true
					m.confirmType = confirmUninstall
					m.confirmPackages = []string{pkg.Name}
					m.confirmScrollOffset = 0
					m.statusMessage = "Confirm removal"
				}
			} else if m.mode == modeUpdate && len(m.pendingUpdates) > 0 {
				// Show confirmation dialog for system update
				m.showConfirmation = true
				m.confirmType = confirmUpdate
				m.confirmScrollOffset = 0
				m.statusMessage = "Confirm system update"
			}

		case "tab":
			// Toggle mark on current package
			if m.mode == modeInstall && len(m.filtered) > 0 {
				pkg := m.filtered[m.selectedIndex]
				if m.markedPackages[pkg.Name] {
					delete(m.markedPackages, pkg.Name)
				} else {
					m.markedPackages[pkg.Name] = true
				}
				markedCount := len(m.markedPackages)
				if markedCount > 0 {
					m.statusMessage = fmt.Sprintf("%d packages marked", markedCount)
				} else {
					m.statusMessage = fmt.Sprintf("Found %d packages", len(m.filtered))
				}
			} else if m.mode == modeUninstall && len(m.filteredInstalled) > 0 {
				pkg := m.filteredInstalled[m.selectedIndex]
				if m.markedPackages[pkg.Name] {
					delete(m.markedPackages, pkg.Name)
				} else {
					m.markedPackages[pkg.Name] = true
				}
				markedCount := len(m.markedPackages)
				if markedCount > 0 {
					m.statusMessage = fmt.Sprintf("%d packages marked", markedCount)
				} else {
					m.statusMessage = fmt.Sprintf("%d installed packages", len(m.installed))
				}
			}

		case "/":
			if (m.mode == modeInstall || m.mode == modeUninstall) && !m.textInput.Focused() {
				m.textInput.Focus()
				if m.mode == modeInstall && len(m.repoPackages) > 0 && m.textInput.Value() == "" {
					m.statusMessage = fmt.Sprintf("Type at least %d chars or use prefix (c: e: m: a:) to filter (%d repo packages)", minSearchQueryLen, len(m.repoPackages))
				} else if m.mode == modeUninstall && len(m.installed) > 0 && m.textInput.Value() == "" {
					m.statusMessage = fmt.Sprintf("Filter: t: total  e: explicit  f: foreign  o: orphan (%d installed)", len(m.installed))
				}
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textInput.Width = msg.Width - 6

	case repoPackagesMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to load packages: %v", msg.err)
		} else {
			m.repoPackages = msg.packages
			
			// Update installed set for quick lookup
			m.installedSet = make(map[string]bool)
			for _, pkg := range m.repoPackages {
				if pkg.Installed {
					m.installedSet[pkg.Name] = true
				}
			}
			
			// Re-apply current search filter if there's a query
			query := m.textInput.Value()
			if m.mode == modeInstall && query != "" {
				repoFilters, searchQuery := parseRepoFilter(query)
				hasRepoFilter := len(repoFilters) > 0
				effectiveQueryLen := len(searchQuery)
				
				if effectiveQueryLen >= minSearchQueryLen || hasRepoFilter {
					m.filterAllPackages(query)
					// Reset selection to top
					m.selectedIndex = 0
					
					if len(m.filtered) > 0 {
						status := fmt.Sprintf("Found %d packages", len(m.filtered))
						if hasRepoFilter {
							status = fmt.Sprintf("Found %d %s packages", len(m.filtered), formatRepoFilters(repoFilters))
						}
						if m.lastCompletedOp != "" {
							status = m.lastCompletedOp + " | " + status
						}
						m.statusMessage = status
						// Load info for first result
						m.loadingInfo = true
						m.infoForPackage = m.filtered[0].Name
						return m, getPackageInfo(m.filtered[0])
					} else {
						m.statusMessage = fmt.Sprintf("No matches for '%s'", query)
					}
				} else {
					m.filtered = []Package{}
					m.matchIndices = nil
					if m.lastCompletedOp != "" {
						m.statusMessage = m.lastCompletedOp
					} else {
						m.statusMessage = fmt.Sprintf("Loaded %d repo packages - press [/] to search", len(m.repoPackages))
					}
				}
			} else {
				if m.lastCompletedOp != "" {
					m.statusMessage = m.lastCompletedOp
				} else {
					m.statusMessage = fmt.Sprintf("Loaded %d repo packages - press [/] to search", len(m.repoPackages))
				}
			}
		}

	case aurSearchMsg:
		m.searchingAUR = false
		// Check if these results are still useful
		// Results are useful if:
		// 1. They match the current query exactly, OR
		// 2. The current query starts with this query (e.g., "hello" results useful for "helloa")
		currentQuery := m.textInput.Value()
		isExactMatch := msg.query == m.lastAURQuery
		isUsefulPrefix := strings.HasPrefix(strings.ToLower(currentQuery), strings.ToLower(msg.query))
		
		if !isExactMatch && !isUsefulPrefix {
			// Truly stale results - discard
			return m, nil
		}
		
		if msg.err == nil {
			// If this is a prefix query's results (e.g., "hello" for "helloa"), 
			// only use them if we don't have better results already
			if !isExactMatch && isUsefulPrefix {
				// Only add prefix results if we don't have AUR packages yet
				if len(m.aurPackages) == 0 && len(msg.packages) > 0 {
					m.aurPackages = msg.packages
				}
			} else {
				// Exact match - use new results, or keep existing if new is empty
				if len(msg.packages) > 0 {
					m.aurPackages = msg.packages
				}
				// If empty, keep existing aurPackages (they'll be filtered)
			}
			
			// Re-filter all packages together for unified relevance ranking
			query := m.textInput.Value()
			if len(query) >= minSearchQueryLen {
				// Remember if user was on the first (most relevant) option
				wasOnFirst := m.selectedIndex == 0
				prevSelected := ""
				if !wasOnFirst && m.selectedIndex < len(m.filtered) {
					prevSelected = m.filtered[m.selectedIndex].Name
				}
				
				m.filterAllPackages(query)
				
				// If user was on first option, stay on first (to see new most relevant)
				// Otherwise try to keep the same package selected
				if wasOnFirst {
					m.selectedIndex = 0
				} else if prevSelected != "" {
					for i, pkg := range m.filtered {
						if pkg.Name == prevSelected {
							m.selectedIndex = i
							break
						}
					}
				}
				if m.selectedIndex >= len(m.filtered) {
					m.selectedIndex = 0
				}
				
				if len(m.filtered) > 0 {
					m.statusMessage = fmt.Sprintf("Found %d packages (%d from AUR)", len(m.filtered), len(msg.packages))
					// Load info for selected result
					if m.filtered[m.selectedIndex].Name != m.infoForPackage {
						m.loadingInfo = true
						m.infoForPackage = m.filtered[m.selectedIndex].Name
						return m, getPackageInfo(m.filtered[m.selectedIndex])
					}
				} else {
					m.statusMessage = fmt.Sprintf("No matches for '%s'", query)
				}
			}
		} else if len(m.filtered) == 0 {
			m.statusMessage = fmt.Sprintf("No matches for '%s'", m.textInput.Value())
		}

	case packageInfoMsg:
		// Only update if this info is for the currently selected package
		if msg.packageName == m.infoForPackage {
			m.loadingInfo = false
			if msg.err != nil {
				m.packageInfo = "Failed to load package info"
			} else {
				m.packageInfo = msg.info
			}
		}
		// If it's stale info (user moved selection), just discard it
		// and keep loadingInfo = true so we continue showing the loading screen

	case debounceTickMsg:
		// Only fetch if this is still the package the user wants info for
		// (i.e., they haven't scrolled away since the debounce started)
		if msg.packageName == m.pendingInfoPackage {
			m.infoForPackage = msg.packageName
			// Find the package and fetch its info
			var pkg *Package
			if m.mode == modeInstall {
				for i := range m.filtered {
					if m.filtered[i].Name == msg.packageName {
						pkg = &m.filtered[i]
						break
					}
				}
			} else if m.mode == modeUninstall {
				for i := range m.filteredInstalled {
					if m.filteredInstalled[i].Name == msg.packageName {
						pkg = &m.filteredInstalled[i]
						break
					}
				}
			}
			if pkg != nil {
				return m, getPackageInfo(*pkg)
			}
		}
		// If pendingInfoPackage changed, this tick is stale - ignore it

	case installedPackagesMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error loading packages: %v", msg.err)
		} else {
			m.installed = msg.packages
			
			// Update installedSet for quick lookup (used by install view)
			m.installedSet = make(map[string]bool)
			for _, pkg := range m.installed {
				m.installedSet[pkg.Name] = true
			}
			
			// Also update the Installed flag on repo packages for install view
			for i := range m.repoPackages {
				m.repoPackages[i].Installed = m.installedSet[m.repoPackages[i].Name]
			}
			// Update filtered list as well
			for i := range m.filtered {
				m.filtered[i].Installed = m.installedSet[m.filtered[i].Name]
			}
			
			// Check if there's a pre-set filter (from dashboard shortcuts)
			query := m.textInput.Value()
			if query != "" {
				// Apply the filter
				sourceFilters, searchQuery := parseUninstallFilter(query)
				hasSourceFilter := len(sourceFilters) > 0
				
				basePackages := m.installed
				if hasSourceFilter {
					var filtered []Package
					for _, pkg := range basePackages {
						if sourceFilters["total"] {
							filtered = append(filtered, pkg)
						} else {
							if sourceFilters["explicit"] && pkg.Explicit {
								filtered = append(filtered, pkg)
							}
							if sourceFilters["foreign"] && pkg.Source == "aur" {
								filtered = append(filtered, pkg)
							}
							if sourceFilters["orphan"] && pkg.Orphan {
								filtered = append(filtered, pkg)
							}
						}
					}
					basePackages = filtered
				}
				
				if searchQuery != "" {
					m.filteredInstalled = fuzzyFilter(basePackages, searchQuery)
					m.installedMatchIndices = computeAllMatchIndices(m.filteredInstalled, searchQuery)
				} else {
					m.filteredInstalled = basePackages
					m.installedMatchIndices = nil
				}
				
				// Reset selection to top
				m.selectedIndex = 0
				
				if hasSourceFilter {
					status := fmt.Sprintf("Found %d %s packages", len(m.filteredInstalled), formatUninstallFilters(sourceFilters))
					if m.lastCompletedOp != "" {
						status = m.lastCompletedOp + " | " + status
					}
					m.statusMessage = status
				} else {
					status := fmt.Sprintf("%d packages - Press [/] to filter", len(m.filteredInstalled))
					if m.lastCompletedOp != "" {
						status = m.lastCompletedOp + " | " + status
					}
					m.statusMessage = status
				}
			} else {
				m.filteredInstalled = m.installed
				status := fmt.Sprintf("%d packages - Press [/] to filter", len(m.installed))
				if m.lastCompletedOp != "" {
					status = m.lastCompletedOp + " | " + status
				}
				m.statusMessage = status
			}
			
			if len(m.filteredInstalled) > 0 {
				m.loadingInfo = true
				m.infoForPackage = m.filteredInstalled[0].Name
				return m, getPackageInfo(m.filteredInstalled[0])
			}
		}

	case dashboardMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error loading dashboard: %v", msg.err)
		} else {
			m.dashboard = msg.data
			// Preserve lastCompletedOp message if set, otherwise show default
			if m.lastCompletedOp != "" {
				m.statusMessage = m.lastCompletedOp
			} else {
				m.statusMessage = "Dashboard loaded"
			}
		}

	case actionCompleteMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMessage = msg.message
		} else {
			m.statusMessage = msg.message
			// Refresh the list
			if m.mode == modeInstall {
				// Reload packages to update installed status
				return m, loadRepoPackages()
			} else if m.mode == modeUninstall {
				return m, getInstalledPackages()
			}
		}

	case cleanCacheMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Cache clean failed: %v", msg.err)
		} else {
			m.statusMessage = "Cache cleaned successfully!"
			// Refresh dashboard to show updated cache size
			return m, getDashboardData()
		}

	case removeOrphansMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Orphan removal failed: %v", msg.err)
		} else {
			m.statusMessage = "Orphans removed successfully!"
			// Refresh dashboard to show updated orphan count
			return m, getDashboardData()
		}

	case updateOutputMsg:
		m.loading = false
		m.updateOutput = msg.output
		if msg.err != nil {
			m.statusMessage = "Update failed"
		} else {
			m.statusMessage = "Update complete!"
		}

	case updateCheckMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Error checking updates: %v", msg.err)
		} else if len(msg.packages) == 0 {
			m.statusMessage = "System is up to date!"
			m.updateOutput = "No updates available."
		} else {
			// Show confirmation dialog with available updates
			m.pendingUpdates = msg.packages
			m.showConfirmation = true
			m.confirmType = confirmUpdate
			m.confirmScrollOffset = 0
			m.statusMessage = fmt.Sprintf("%d update(s) available", len(msg.packages))
		}

	case execCompleteMsg:
		m.loading = false
		m.confirmPackages = nil
		m.pendingUpdates = nil
		
		// Check if operation failed and show error overlay
		if msg.err != nil {
			opName := ""
			switch msg.operation {
			case confirmInstall:
				opName = "Installation"
			case confirmUninstall:
				opName = "Removal"
			case confirmUpdate:
				opName = "System Update"
			case confirmCleanCache:
				opName = "Cache Cleaning"
			case confirmRemoveOrphans:
				opName = "Orphan Removal"
			}
			
			m.showErrorOverlay = true
			m.errorTitle = fmt.Sprintf("%s Failed", opName)
			m.errorMessage = "The operation exited with a non-zero exit code."
			
			// Get error details
			if exitErr, ok := msg.err.(*exec.ExitError); ok {
				m.errorDetails = fmt.Sprintf("Exit code: %d\n\nThe error output was displayed in the terminal.\nPlease check the terminal output for details.", exitErr.ExitCode())
			} else {
				m.errorDetails = fmt.Sprintf("Error: %v\n\nThe error output was displayed in the terminal.\nPlease check the terminal output for details.", msg.err)
			}
			
			m.statusMessage = fmt.Sprintf("%s failed", opName)
			m.lastCompletedOp = ""
			
			// Still refresh the appropriate data
			switch msg.operation {
			case confirmInstall:
				return m, loadRepoPackages()
			case confirmUninstall:
				return m, getInstalledPackages()
			case confirmUpdate:
				return m, loadRepoPackages()
			case confirmCleanCache, confirmRemoveOrphans:
				return m, getDashboardData()
			}
			return m, nil
		}
		
		// Operation succeeded
		switch msg.operation {
		case confirmInstall:
			if len(msg.packages) == 1 {
				m.lastCompletedOp = fmt.Sprintf("Installed: %s", msg.packages[0])
			} else {
				m.lastCompletedOp = fmt.Sprintf("Installed %d packages", len(msg.packages))
			}
			m.statusMessage = m.lastCompletedOp
			return m, loadRepoPackages()
		case confirmUninstall:
			if len(msg.packages) == 1 {
				m.lastCompletedOp = fmt.Sprintf("Removed: %s", msg.packages[0])
			} else {
				m.lastCompletedOp = fmt.Sprintf("Removed %d packages", len(msg.packages))
			}
			m.statusMessage = m.lastCompletedOp
			return m, getInstalledPackages()
		case confirmUpdate:
			m.lastCompletedOp = "System update completed"
			m.statusMessage = m.lastCompletedOp
			return m, loadRepoPackages()
		case confirmCleanCache:
			m.lastCompletedOp = "Cache cleaned successfully"
			m.statusMessage = m.lastCompletedOp
			return m, getDashboardData()
		case confirmRemoveOrphans:
			if len(msg.packages) == 1 {
				m.lastCompletedOp = fmt.Sprintf("Removed orphan: %s", msg.packages[0])
			} else {
				m.lastCompletedOp = fmt.Sprintf("Removed %d orphan packages", len(msg.packages))
			}
			m.statusMessage = m.lastCompletedOp
			return m, getDashboardData()
		}
	}

	return m, tea.Batch(cmds...)
}

// renderHelpText creates the help menu with the active mode highlighted
func (m model) renderHelpText(activeColor lipgloss.Color) string {
	dimStyle := helpStyle
	activeStyle := lipgloss.NewStyle().
		Foreground(activeColor).
		Bold(true)

	var parts []string

	// Common items (always dim)
	parts = append(parts, dimStyle.Render("[/] search  [tab] mark  "))

	// [i]nstall
	if m.mode == modeInstall {
		parts = append(parts, activeStyle.Render("[i]nstall"))
	} else {
		parts = append(parts, dimStyle.Render("[i]nstall"))
	}
	parts = append(parts, dimStyle.Render("  "))

	// i[n]fo
	if m.mode == modeInstalled {
		parts = append(parts, activeStyle.Render("i[n]fo"))
	} else {
		parts = append(parts, dimStyle.Render("i[n]fo"))
	}
	parts = append(parts, dimStyle.Render("  "))

	// [r]emove
	if m.mode == modeUninstall {
		parts = append(parts, activeStyle.Render("[r]emove"))
	} else {
		parts = append(parts, dimStyle.Render("[r]emove"))
	}
	parts = append(parts, dimStyle.Render("  "))

	// [u]pdate
	if m.mode == modeUpdate {
		parts = append(parts, activeStyle.Render("[u]pdate"))
	} else {
		parts = append(parts, dimStyle.Render("[u]pdate"))
	}
	parts = append(parts, dimStyle.Render("  "))

	// [q]uit (always dim)
	parts = append(parts, dimStyle.Render("[q]uit"))

	return strings.Join(parts, "")
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Calculate dimensions
	contentWidth := m.width - 4
	contentHeight := m.height - 4

	// Get mode-specific color
	activeColor := modeColors[m.mode]
	if activeColor == "" {
		activeColor = defaultBorderColor
	}

	// Create mode-specific styles
	titleStyle := baseTitleStyle.Background(activeColor)
	borderStyle := baseBorderStyle.BorderForeground(activeColor)

	// Build header with mode
	modeText := ""
	switch m.mode {
	case modeInstall:
		modeText = "INSTALL"
	case modeInstalled:
		modeText = "INFO"
	case modeUninstall:
		modeText = "UNINSTALL"
	case modeUpdate:
		modeText = "UPDATE"
	}

	header := titleStyle.Render(" GAUR - " + modeText + " ")

	// Help text for bottom right with active item highlighted
	helpText := m.renderHelpText(activeColor)

	// Render confirmation dialog if active
	if m.showConfirmation {
		return m.renderConfirmationDialog(contentWidth, contentHeight, activeColor)
	}

	// Render error overlay if active
	if m.showErrorOverlay {
		return m.renderErrorOverlay(contentWidth, contentHeight)
	}

	// Dashboard view
	if m.mode == modeInstalled {
		return m.renderDashboard(helpText, contentWidth, contentHeight)
	}

	// Top half: Package info
	infoHeight := contentHeight / 2
	infoContent := ""
	if m.mode == modeUpdate {
		if m.updateOutput != "" {
			infoContent = m.updateOutput
		} else if m.loading {
			infoContent = "Checking for updates..."
		} else if len(m.pendingUpdates) > 0 {
			infoContent = fmt.Sprintf("%d update(s) available. Press [enter] to review and update.", len(m.pendingUpdates))
		} else {
			infoContent = "System is up to date. Press [u] to check again."
		}
	} else if m.loadingInfo {
		infoContent = fmt.Sprintf("Loading details for %s...", m.infoForPackage)
	} else if m.packageInfo != "" {
		infoContent = m.packageInfo
	} else {
		infoContent = "Select a package to see details"
	}

	// Wrap and truncate info
	infoLines := strings.Split(infoContent, "\n")
	if len(infoLines) > infoHeight-2 {
		infoLines = infoLines[:infoHeight-2]
	}
	infoContent = strings.Join(infoLines, "\n")

	infoBox := lipgloss.NewStyle().
		Width(contentWidth-2).
		Height(infoHeight-2).
		Padding(0, 1).
		Render(infoContent)

	infoPanel := borderStyle.
		Width(contentWidth).
		Height(infoHeight).
		Render(infoBox)

	// Bottom half: Results + Input
	bottomHeight := contentHeight - infoHeight - 1
	resultsHeight := bottomHeight - 3

	// Build results list
	var results strings.Builder
	var pkgList []Package
	if m.mode == modeInstall {
		pkgList = m.filtered
	} else if m.mode == modeUninstall {
		pkgList = m.filteredInstalled
	}

	if m.loading {
		results.WriteString("  Loading...")
	} else if m.mode == modeUpdate {
		results.WriteString("  " + m.statusMessage)
	} else if len(pkgList) == 0 {
		results.WriteString("  No packages to display")
	} else {
		// Show packages that fit, reversed so most relevant is at bottom (near input)
		startIdx := 0
		if m.selectedIndex >= resultsHeight {
			startIdx = m.selectedIndex - resultsHeight + 1
		}
		endIdx := startIdx + resultsHeight
		if endIdx > len(pkgList) {
			endIdx = len(pkgList)
		}

		// Get the appropriate match indices map
		var matchIndicesMap map[int][]int
		if m.mode == modeInstall {
			matchIndicesMap = m.matchIndices
		} else if m.mode == modeUninstall {
			matchIndicesMap = m.installedMatchIndices
		}

		// Build lines in reverse order (most relevant at bottom, near input field)
		var lines []string
		for i := startIdx; i < endIdx; i++ {
			pkg := pkgList[i]
			// Show marker for marked packages
			marker := " "
			if m.markedPackages[pkg.Name] {
				marker = "*"
			}
			prefix := " " + marker
			if i == m.selectedIndex {
				prefix = ">" + marker
			}

			// Color by source
			sourceStyle := lipgloss.NewStyle()
			if color, ok := sourceColors[pkg.Source]; ok {
				sourceStyle = sourceStyle.Foreground(color)
			}

			// Apply highlighting with source colors
			var displayPkgStr string
			if matchIndicesMap != nil {
				if indices, ok := matchIndicesMap[i]; ok {
					// Use combined highlighting that preserves source colors
					displayPkgStr = highlightMatchesWithSourceColor(pkg, indices)
				} else {
					displayPkgStr = sourceStyle.Render(pkg.Source) + "/" + pkg.Name
				}
			} else {
				displayPkgStr = sourceStyle.Render(pkg.Source) + "/" + pkg.Name
			}

			line := fmt.Sprintf("%s%s %s",
				prefix,
				displayPkgStr,
				lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(pkg.Version),
			)

			if pkg.Installed && m.mode == modeInstall {
				line += " " + installedBadge.Render("[installed]")
			}

			// Truncate if too long
			if lipgloss.Width(line) > contentWidth-4 {
				line = line[:contentWidth-7] + "..."
			}

			if i == m.selectedIndex {
				line = selectedStyle.Render(line)
			}

			lines = append(lines, line)
		}

		// Reverse the lines so most relevant (index 0) is at bottom
		for i := len(lines) - 1; i >= 0; i-- {
			results.WriteString(lines[i])
			if i > 0 {
				results.WriteString("\n")
			}
		}
	}

	resultsBox := lipgloss.NewStyle().
		Width(contentWidth - 2).
		Height(resultsHeight).
		Render(results.String())

	// Input field
	inputLine := ""
	if m.mode == modeInstall || m.mode == modeUninstall {
		inputLine = m.textInput.View()
	} else {
		inputLine = statusStyle.Render("System update in progress...")
	}

	// Status line
	statusLine := statusStyle.Render(m.statusMessage)

	// Layout: results at top, input at bottom (fzf-style)
	bottomContent := lipgloss.JoinVertical(
		lipgloss.Left,
		resultsBox,
		"",
		statusLine,
		inputLine,
	)

	bottomPanel := borderStyle.
		Width(contentWidth).
		Height(bottomHeight).
		Render(bottomContent)

	// Footer with help text aligned to the right
	helpWidth := lipgloss.Width(helpText)
	padding := contentWidth - helpWidth
	if padding < 0 {
		padding = 0
	}
	footer := strings.Repeat(" ", padding) + helpText

	// Combine all
	content := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		infoPanel,
		bottomPanel,
		footer,
	)

	// Overlay selections panel if there are marked packages
	if len(m.markedPackages) > 0 {
		content = m.overlaySelectionsPanel(content, contentWidth)
	}

	return content
}

// overlaySelectionsPanel renders a selection panel on the bottom right of the screen
func (m model) overlaySelectionsPanel(content string, contentWidth int) string {
	// Panel styling - brighter border when focused
	borderColor := lipgloss.Color("205")
	if m.selectionPanelFocused {
		borderColor = lipgloss.Color("213")
	}
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1)

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	itemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	selectedItemStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("213")).
		Bold(true)

	keyHintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)

	// Build the selections list with * hint in title
	var selectionsList strings.Builder
	// Render the title (styled)
	titleText := titleStyle.Render(fmt.Sprintf("Selected (%d) ", len(m.markedPackages))) + keyHintStyle.Render("[*]")
	selectionsList.WriteString(titleText)

	// Collect and sort package names for consistent display
	var pkgNames []string
	for name := range m.markedPackages {
		pkgNames = append(pkgNames, name)
	}
	sort.Strings(pkgNames)

	// Determine panel width dynamically within bounds
	maxDisplay := 20
	minPanelWidth := 12
	maxPanelWidth := 32

	// Compute widest line among title, package names (up to maxDisplay), and the "... +N more" line
	maxContentWidth := lipgloss.Width(titleText)
	visibleCount := maxDisplay
	if len(pkgNames) < visibleCount {
		visibleCount = len(pkgNames)
	}
	for i := 0; i < visibleCount; i++ {
		// account for prefix ("  " or "> ")
		nameWidth := lipgloss.Width(pkgNames[i]) + 2
		if nameWidth > maxContentWidth {
			maxContentWidth = nameWidth
		}
	}
	if len(pkgNames) > maxDisplay {
		moreStr := itemStyle.Render(fmt.Sprintf("... +%d more", len(pkgNames)-maxDisplay))
		if w := lipgloss.Width(moreStr); w > maxContentWidth {
			maxContentWidth = w
		}
	}

	desiredPanelWidth := maxContentWidth + 4 // add room for borders and padding
	if desiredPanelWidth < minPanelWidth {
		desiredPanelWidth = minPanelWidth
	}
	if desiredPanelWidth > maxPanelWidth {
		desiredPanelWidth = maxPanelWidth
	}
	panelWidth := desiredPanelWidth

	// Build the lines, truncating names that exceed available space
	for i, name := range pkgNames {
		if i >= maxDisplay {
			selectionsList.WriteString("\n")
			selectionsList.WriteString(itemStyle.Render(fmt.Sprintf("... +%d more", len(pkgNames)-maxDisplay)))
			break
		}

		// calculate maximum width available for the name itself
		innerWidth := panelWidth - 4 // subtract borders and padding
		nameMaxWidth := innerWidth - 2 // subtract prefix width
		if nameMaxWidth < 1 {
			nameMaxWidth = 1
		}

		displayName := name
		if lipgloss.Width(displayName) > nameMaxWidth {
			// Truncate to fit with ellipsis - preserve runes
			runes := []rune(displayName)
			truncWidth := nameMaxWidth - 3
			if truncWidth < 1 {
				truncWidth = 1
			}
			var truncated string
			for j := 1; j <= len(runes); j++ {
				s := string(runes[:j])
				if lipgloss.Width(s) > truncWidth {
					truncated = string(runes[:j-1]) + "..."
					break
				}
				if j == len(runes) {
					truncated = s
				}
			}
			displayName = truncated
		}

		selectionsList.WriteString("\n")
		// Highlight selected item when panel is focused
		if m.selectionPanelFocused && i == m.selectionPanelIndex {
			selectionsList.WriteString(selectedItemStyle.Render("> " + displayName))
		} else {
			selectionsList.WriteString(itemStyle.Render("  " + displayName))
		}
	}

	panel := panelStyle.Width(panelWidth).Render(selectionsList.String())
	panelHeight := strings.Count(panel, "\n") + 1

	// Split content into lines
	lines := strings.Split(content, "\n")

	// Calculate position (top right corner, flush with borders)
	panelActualWidth := lipgloss.Width(panel)

	// Position panel: start slightly lower to avoid overlapping the top border and flush right (add offset to push further right)
	startRow := 1
	startCol := contentWidth - panelActualWidth + 2
	if startCol < 0 {
		startCol = 0
	}

	// Build new content with overlay
	var result strings.Builder
	panelLines := strings.Split(panel, "\n")

	for i, line := range lines {
		if i >= startRow && i < startRow+panelHeight {
			panelLineIdx := i - startRow
			if panelLineIdx < len(panelLines) {
				// Calculate visible width of line before panel
				lineWidth := lipgloss.Width(line)
				if lineWidth < startCol {
					// Pad line to reach panel position
					line = line + strings.Repeat(" ", startCol-lineWidth)
				} else if lineWidth > startCol {
					// Truncate line to make room for panel
					// We need to be careful with ANSI codes
					line = truncateWithAnsi(line, startCol)
				}
				line = line + panelLines[panelLineIdx]
			}
		}
		result.WriteString(line)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}

// truncateWithAnsi truncates a string to a visual width, preserving ANSI codes
func truncateWithAnsi(s string, maxWidth int) string {
	var result strings.Builder
	width := 0
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
		if width >= maxWidth {
			break
		}
		result.WriteRune(r)
		width++
	}

	// Reset any open styles
	result.WriteString("\x1b[0m")
	return result.String()
}

// renderConfirmationDialog renders a centered confirmation dialog for install/uninstall/update
func (m model) renderConfirmationDialog(contentWidth, contentHeight int, activeColor lipgloss.Color) string {
	// Dialog dimensions
	dialogWidth := contentWidth - 20
	if dialogWidth < 50 {
		dialogWidth = 50
	}
	if dialogWidth > 80 {
		dialogWidth = 80
	}
	
	// Determine packages to display and title
	var packages []Package
	var title string
	var actionDesc string
	var simpleConfirm bool // For confirmations without package lists
	
	switch m.confirmType {
	case confirmInstall:
		title = "📦 Confirm Installation"
		actionDesc = "install"
		for _, name := range m.confirmPackages {
			packages = append(packages, Package{Name: name})
		}
	case confirmUninstall:
		title = "🗑️  Confirm Removal"
		actionDesc = "remove"
		for _, name := range m.confirmPackages {
			packages = append(packages, Package{Name: name})
		}
	case confirmUpdate:
		title = "🔄 Confirm System Update"
		actionDesc = "update"
		packages = m.pendingUpdates
	case confirmCleanCache:
		title = "🧹 Confirm Cache Cleaning"
		actionDesc = "clean"
		simpleConfirm = true
	case confirmRemoveOrphans:
		title = "🗑️  Confirm Orphan Removal"
		actionDesc = "remove"
		for _, name := range m.confirmPackages {
			packages = append(packages, Package{Name: name})
		}
	}
	
	// Styles
	dialogBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(activeColor).
		Padding(1, 2)
	
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(activeColor).
		MarginBottom(1)
	
	packageNameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39"))
	
	packageVersionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
	
	sourceStyle := func(source string) lipgloss.Style {
		if color, ok := sourceColors[source]; ok {
			return lipgloss.NewStyle().Foreground(color)
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	}
	
	countStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)
	
	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")).
		MarginTop(1)
	
	keyStyle := lipgloss.NewStyle().
		Foreground(activeColor).
		Bold(true)
	
	scrollHintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
	
	// Build dialog content
	var content strings.Builder
	
	// Title
	content.WriteString(titleStyle.Render(title))
	content.WriteString("\n\n")
	
	// Handle simple confirmations (no package list)
	if simpleConfirm {
		if m.confirmType == confirmCleanCache {
			content.WriteString("This will remove cached packages that are no longer installed.\n\n")
			
			// Pacman cache info
			content.WriteString(packageNameStyle.Render("Pacman Cache (system):\n"))
			content.WriteString(fmt.Sprintf("  Path: %s\n", scrollHintStyle.Render(m.dashboard.PacmanCachePath)))
			content.WriteString(fmt.Sprintf("  Size: %s\n\n", countStyle.Render(m.dashboard.PacmanCacheSize)))
			
			// Paru cache info
			content.WriteString(packageNameStyle.Render("Paru Cache (user):\n"))
			content.WriteString(fmt.Sprintf("  Path: %s\n", scrollHintStyle.Render(m.dashboard.ParuCachePath)))
			content.WriteString(fmt.Sprintf("  Size: %s\n\n", countStyle.Render(m.dashboard.ParuCacheSize)))
			
			// Total
			content.WriteString(fmt.Sprintf("Total cache size: %s\n", 
				lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214")).Render(m.dashboard.CleanerSize)))
		}
	} else {
		// Package count
		if len(packages) == 1 {
			content.WriteString(fmt.Sprintf("The following package will be %sd:\n\n", actionDesc))
		} else {
			content.WriteString(fmt.Sprintf("The following %s packages will be %sd:\n\n", 
				countStyle.Render(fmt.Sprintf("%d", len(packages))), actionDesc))
		}
		
		// Package list with scrolling
		maxVisible := 10
		startIdx := m.confirmScrollOffset
		endIdx := startIdx + maxVisible
		if endIdx > len(packages) {
			endIdx = len(packages)
		}
		
		// Show scroll indicator at top if needed
		if startIdx > 0 {
			content.WriteString(scrollHintStyle.Render(fmt.Sprintf("  ↑ %d more above\n", startIdx)))
		}
		
		// List packages
		for i := startIdx; i < endIdx; i++ {
			pkg := packages[i]
			if m.confirmType == confirmUpdate {
				// Show source and version info for updates
				sourceBadge := sourceStyle(pkg.Source).Render(fmt.Sprintf("[%s]", pkg.Source))
				content.WriteString(fmt.Sprintf("  • %s %s %s\n",
					sourceBadge,
					packageNameStyle.Render(pkg.Name),
					packageVersionStyle.Render(pkg.Version)))
			} else {
				// Just show package name for install/uninstall
				content.WriteString(fmt.Sprintf("  • %s\n", packageNameStyle.Render(pkg.Name)))
			}
		}
		
		// Show scroll indicator at bottom if needed
		remaining := len(packages) - endIdx
		if remaining > 0 {
			content.WriteString(scrollHintStyle.Render(fmt.Sprintf("  ↓ %d more below\n", remaining)))
		}
		
		// Scroll hint if list is scrollable
		if len(packages) > maxVisible {
			content.WriteString("\n")
			content.WriteString(scrollHintStyle.Render("  Use [↑/↓] or [j/k] to scroll"))
		}
	}
	
	// Prompt - build as single line to prevent wrapping issues
	content.WriteString("\n\n")
	promptLine := fmt.Sprintf("Proceed? %ses  %so",
		keyStyle.Render("[y]"),
		keyStyle.Render("[n]"))
	content.WriteString(promptStyle.Render(promptLine))
	
	// Render dialog box
	dialogContent := content.String()
	dialog := dialogBorderStyle.Width(dialogWidth).Render(dialogContent)
	
	// Center the dialog on screen
	dialogHeight := strings.Count(dialog, "\n") + 1
	
	// Calculate vertical and horizontal padding
	vertPadding := (contentHeight - dialogHeight) / 2
	if vertPadding < 0 {
		vertPadding = 0
	}
	horizPadding := (contentWidth - lipgloss.Width(dialog)) / 2
	if horizPadding < 0 {
		horizPadding = 0
	}
	
	// Build final output with centering
	var output strings.Builder
	
	// Add top padding
	for i := 0; i < vertPadding; i++ {
		output.WriteString("\n")
	}
	
	// Add dialog with horizontal padding
	for _, line := range strings.Split(dialog, "\n") {
		output.WriteString(strings.Repeat(" ", horizPadding))
		output.WriteString(line)
		output.WriteString("\n")
	}
	
	return output.String()
}

// renderErrorOverlay renders a centered error overlay dialog
func (m model) renderErrorOverlay(contentWidth, contentHeight int) string {
	// Error color (red)
	errorColor := lipgloss.Color("#FF5555")
	
	// Dialog dimensions
	dialogWidth := contentWidth - 20
	if dialogWidth < 50 {
		dialogWidth = 50
	}
	if dialogWidth > 80 {
		dialogWidth = 80
	}
	
	// Styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(errorColor).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)
	
	messageStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)
	
	detailsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#999999")).
		Width(dialogWidth - 4).
		Padding(1, 0)
	
	hintStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Width(dialogWidth - 4).
		Align(lipgloss.Center)
	
	dialogBorderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(errorColor).
		Padding(1, 2)
	
	// Build content
	var content strings.Builder
	
	// Error icon and title
	content.WriteString(titleStyle.Render("⚠  " + m.errorTitle + "  ⚠"))
	content.WriteString("\n\n")
	
	// Error message
	content.WriteString(messageStyle.Render(m.errorMessage))
	content.WriteString("\n")
	
	// Error details
	if m.errorDetails != "" {
		content.WriteString(detailsStyle.Render(m.errorDetails))
		content.WriteString("\n")
	}
	
	// Dismiss hint
	content.WriteString(hintStyle.Render("Press [esc], [enter], or [q] to dismiss"))
	
	// Render dialog box
	dialogContent := content.String()
	dialog := dialogBorderStyle.Width(dialogWidth).Render(dialogContent)
	
	// Center the dialog on screen
	dialogHeight := strings.Count(dialog, "\n") + 1
	
	// Calculate vertical and horizontal padding
	vertPadding := (contentHeight - dialogHeight) / 2
	if vertPadding < 0 {
		vertPadding = 0
	}
	horizPadding := (contentWidth - lipgloss.Width(dialog)) / 2
	if horizPadding < 0 {
		horizPadding = 0
	}
	
	// Build final output with centering
	var output strings.Builder
	
	// Add top padding
	for i := 0; i < vertPadding; i++ {
		output.WriteString("\n")
	}
	
	// Add dialog with horizontal padding
	for _, line := range strings.Split(dialog, "\n") {
		output.WriteString(strings.Repeat(" ", horizPadding))
		output.WriteString(line)
		output.WriteString("\n")
	}
	
	return output.String()
}

func (m model) renderDashboard(helpText string, contentWidth, contentHeight int) string {
	// Get mode-specific border style
	activeColor := modeColors[m.mode]
	if activeColor == "" {
		activeColor = defaultBorderColor
	}
	borderStyle := baseBorderStyle.BorderForeground(activeColor)

	// Footer with help text aligned to the right
	helpWidth := lipgloss.Width(helpText)
	padding := contentWidth - helpWidth
	if padding < 0 {
		padding = 0
	}
	footerLine := strings.Repeat(" ", padding) + helpText

	if m.loading {
		loadingBox := borderStyle.
			Width(contentWidth).
			Height(contentHeight - 1).
			Render(lipgloss.Place(contentWidth-2, contentHeight-3, lipgloss.Center, lipgloss.Center, "Loading system statistics..."))
		return lipgloss.JoinVertical(lipgloss.Left, loadingBox, footerLine)
	}

	var dashboard strings.Builder

	// Color definitions
	greenColor := lipgloss.Color("42")
	redColor := lipgloss.Color("196")
	yellowColor := lipgloss.Color("214")
	orangeColor := lipgloss.Color("208")
	cyanColor := lipgloss.Color("51")
	dimColor := lipgloss.Color("240")

	// Box styles
	boxTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("229"))

	// Shortcut hint style
	shortcutStyle := lipgloss.NewStyle().Foreground(dimColor)

	// ═══════════════════════════════════════════════════════
	// GROUP 1: Package Counts (with shortcuts to filter in remove mode)
	// ═══════════════════════════════════════════════════════
	
	// Build package counts content as simple lines
	countsLines := []string{
		fmt.Sprintf(" %s Total    │ %s",
			shortcutStyle.Render("[t]"),
			lipgloss.NewStyle().Bold(true).Foreground(cyanColor).Render(fmt.Sprintf("%d", m.dashboard.TotalPackages))),
		fmt.Sprintf(" %s Explicit │ %s",
			shortcutStyle.Render("[e]"),
			lipgloss.NewStyle().Bold(true).Foreground(greenColor).Render(fmt.Sprintf("%d", m.dashboard.ExplicitlyInstalled))),
		fmt.Sprintf(" %s Foreign  │ %s",
			shortcutStyle.Render("[f]"),
			lipgloss.NewStyle().Bold(true).Foreground(yellowColor).Render(fmt.Sprintf("%d", m.dashboard.ForeignPackages))),
	}
	
	// Orphan line with optional remove hint
	orphanStyle := lipgloss.NewStyle().Bold(true).Foreground(greenColor)
	if m.dashboard.Orphans > 0 {
		orphanStyle = lipgloss.NewStyle().Bold(true).Foreground(redColor)
	}
	orphanLine := fmt.Sprintf(" %s Orphans  │ %s",
		shortcutStyle.Render("[o]"),
		orphanStyle.Render(fmt.Sprintf("%d", m.dashboard.Orphans)))
	if m.dashboard.Orphans > 0 {
		orphanLine += shortcutStyle.Render(" [R]rm")
	}
	countsLines = append(countsLines, orphanLine)

	// ═══════════════════════════════════════════════════════
	// GROUP 2: Storage Info
	// ═══════════════════════════════════════════════════════
	
	// Cache coloring: warm colors if > 10 GiB
	cacheStyle := lipgloss.NewStyle().Bold(true).Foreground(greenColor)
	const tenGiB = 10 * 1024 * 1024 * 1024
	if m.dashboard.CleanerSizeBytes > tenGiB {
		cacheStyle = lipgloss.NewStyle().Bold(true).Foreground(orangeColor)
	}
	if m.dashboard.CleanerSizeBytes > tenGiB*2 {
		cacheStyle = lipgloss.NewStyle().Bold(true).Foreground(redColor)
	}
	
	// Missing from AUR style
	missingStyle := lipgloss.NewStyle().Bold(true).Foreground(greenColor)
	if m.dashboard.MissingFromAUR > 0 {
		missingStyle = lipgloss.NewStyle().Bold(true).Foreground(redColor)
	}

	storageLines := []string{
		fmt.Sprintf("  System  │ %s",
			lipgloss.NewStyle().Bold(true).Foreground(cyanColor).Render(m.dashboard.TotalSize)),
		fmt.Sprintf("  Cache   │ %s %s",
			cacheStyle.Render(m.dashboard.CleanerSize),
			shortcutStyle.Render("[c]lean")),
		fmt.Sprintf("  Missing │ %s",
			missingStyle.Render(fmt.Sprintf("%d AUR", m.dashboard.MissingFromAUR))),
		"", // Empty line to match height
	}

	// Render boxes manually with Unicode box drawing
	borderColor := lipgloss.NewStyle().Foreground(activeColor)
	
	// Helper to render a box with title
	renderBox := func(title string, lines []string, width int) string {
		var b strings.Builder
		
		// Ensure minimum content width
		innerWidth := width - 4 // Account for border chars and padding
		if innerWidth < 20 {
			innerWidth = 20
		}
		
		// Top border with title
		titleLen := lipgloss.Width(title)
		topLeft := borderColor.Render("╭─")
		topRight := borderColor.Render("─╮")
		topPadding := innerWidth - titleLen
		if topPadding < 0 {
			topPadding = 0
		}
		b.WriteString(topLeft + title + borderColor.Render(strings.Repeat("─", topPadding)) + topRight + "\n")
		
		// Content lines
		leftBorder := borderColor.Render("│ ")
		rightBorder := borderColor.Render(" │")
		for _, line := range lines {
			// Pad line to fill width
			lineWidth := lipgloss.Width(line)
			padding := innerWidth - lineWidth
			if padding < 0 {
				padding = 0
			}
			b.WriteString(leftBorder + line + strings.Repeat(" ", padding) + rightBorder + "\n")
		}
		
		// Bottom border
		b.WriteString(borderColor.Render("╰" + strings.Repeat("─", innerWidth+2) + "╯"))
		
		return b.String()
	}

	// Calculate box width
	boxWidth := (contentWidth - 6) / 2
	if boxWidth < 30 {
		boxWidth = 30
	}

	countsBox := renderBox(boxTitleStyle.Render(" 📦 Package Counts "), countsLines, boxWidth)
	storageBox := renderBox(boxTitleStyle.Render(" 💾 Storage "), storageLines, boxWidth)

	// Layout boxes side by side
	countsBoxLines := strings.Split(countsBox, "\n")
	storageBoxLines := strings.Split(storageBox, "\n")
	
	// Ensure same number of lines
	maxLines := len(countsBoxLines)
	if len(storageBoxLines) > maxLines {
		maxLines = len(storageBoxLines)
	}
	for len(countsBoxLines) < maxLines {
		countsBoxLines = append(countsBoxLines, strings.Repeat(" ", boxWidth))
	}
	for len(storageBoxLines) < maxLines {
		storageBoxLines = append(storageBoxLines, strings.Repeat(" ", boxWidth))
	}
	
	// Join boxes horizontally
	for i := 0; i < maxLines; i++ {
		dashboard.WriteString(countsBoxLines[i] + "  " + storageBoxLines[i] + "\n")
	}
	dashboard.WriteString("\n")

	// ═══════════════════════════════════════════════════════
	// Bar Layout Constants - ensures all bars align perfectly
	// ═══════════════════════════════════════════════════════
	const barLeftMargin = 2                    // Spaces before label
	const barLabelWidth = 8                    // Fixed width for labels (e.g., "System", "Cache")
	const barSeparator = "│"                   // Separator between label and bar
	const barSuffixReserve = 30                // Reserve space for suffix text (e.g., "1234/5678 (100% explicit)")
	barStartCol := barLeftMargin + barLabelWidth + len(barSeparator)
	availableBarWidth := contentWidth - barStartCol - barSuffixReserve
	if availableBarWidth < 20 {
		availableBarWidth = 20
	}

	// Helper to create aligned bar line
	renderBarLine := func(label string, bar string, suffix string) string {
		paddedLabel := fmt.Sprintf("%*s%-*s%s", barLeftMargin, "", barLabelWidth, label, barSeparator)
		return paddedLabel + bar + " " + suffix
	}

	// ═══════════════════════════════════════════════════════
	// Progress Bar: Explicit vs Dependency Ratio
	// ═══════════════════════════════════════════════════════
	dependencies := m.dashboard.TotalPackages - m.dashboard.ExplicitlyInstalled
	explicitRatio := float64(m.dashboard.ExplicitlyInstalled) / float64(m.dashboard.TotalPackages)
	if m.dashboard.TotalPackages == 0 {
		explicitRatio = 0
	}
	
	filledWidth := int(explicitRatio * float64(availableBarWidth))
	if filledWidth > availableBarWidth {
		filledWidth = availableBarWidth
	}
	
	filledBar := lipgloss.NewStyle().Background(greenColor).Foreground(lipgloss.Color("0")).
		Render(strings.Repeat(" ", filledWidth))
	emptyBar := lipgloss.NewStyle().Background(lipgloss.Color("238")).
		Render(strings.Repeat(" ", availableBarWidth-filledWidth))
	
	ratioTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).
		Render("📊 Explicit vs Dependencies")
	ratioSuffix := fmt.Sprintf("%d/%d (%.0f%% explicit)", m.dashboard.ExplicitlyInstalled, dependencies, explicitRatio*100)
	ratioBar := renderBarLine("", filledBar+emptyBar, ratioSuffix)
	
	dashboard.WriteString(ratioTitle + "\n")
	dashboard.WriteString(ratioBar + "\n\n")

	// ═══════════════════════════════════════════════════════
	// Bar Chart: System Size vs Cache Size
	// ═══════════════════════════════════════════════════════
	chartTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).
		Render("📈 Size Comparison")
	dashboard.WriteString(chartTitle + "\n")
	
	maxSize := m.dashboard.TotalSizeBytes
	if m.dashboard.CleanerSizeBytes > maxSize {
		maxSize = m.dashboard.CleanerSizeBytes
	}
	if maxSize == 0 {
		maxSize = 1
	}
	
	systemBarWidth := int(float64(m.dashboard.TotalSizeBytes) / float64(maxSize) * float64(availableBarWidth))
	cacheBarWidth := int(float64(m.dashboard.CleanerSizeBytes) / float64(maxSize) * float64(availableBarWidth))
	if systemBarWidth < 1 {
		systemBarWidth = 1
	}
	if cacheBarWidth < 1 && m.dashboard.CleanerSizeBytes > 0 {
		cacheBarWidth = 1
	}
	
	systemBar := lipgloss.NewStyle().Background(cyanColor).Render(strings.Repeat(" ", systemBarWidth))
	cacheBar := lipgloss.NewStyle().Background(orangeColor).Render(strings.Repeat(" ", cacheBarWidth))
	
	dashboard.WriteString(renderBarLine("System", systemBar, m.dashboard.TotalSize) + "\n")
	dashboard.WriteString(renderBarLine("Cache", cacheBar, m.dashboard.CleanerSize) + "\n\n")

	// ═══════════════════════════════════════════════════════
	// Top 10 Packages by Size
	// ═══════════════════════════════════════════════════════
	if len(m.dashboard.TopPackages) > 0 {
		topTitle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("229")).
			Render("🏆 Top 10 Packages by Size")
		dashboard.WriteString(topTitle + "\n")
		
		for i, pkg := range m.dashboard.TopPackages {
			rankStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
			nameStyle := lipgloss.NewStyle().Foreground(cyanColor)
			sizeStyle := lipgloss.NewStyle().Foreground(yellowColor)
			
			dashboard.WriteString(fmt.Sprintf("  %s %s %s\n",
				rankStyle.Render(fmt.Sprintf("%2d.", i+1)),
				nameStyle.Render(fmt.Sprintf("%-30s", pkg.Name)),
				sizeStyle.Render(pkg.Size)))
		}
		dashboard.WriteString("\n")
	}

	// ═══════════════════════════════════════════════════════
	// Interactive Actions Footer
	// ═══════════════════════════════════════════════════════
	// actionsStyle := lipgloss.NewStyle().Foreground(dimColor)
	// var actions []string
	// actions = append(actions, "[c] Clean cache")
	// if m.dashboard.Orphans > 0 {
	// 	actions = append(actions, lipgloss.NewStyle().Foreground(redColor).Render("[R] Remove orphans"))
	// }
	// actions = append(actions, "[esc] back")
	// actions = append(actions, "[q] quit")
	
	// actionsText := actionsStyle.Render("  " + strings.Join(actions, " │ "))
	// dashboard.WriteString(actionsText)

	dashContent := lipgloss.NewStyle().
		Width(contentWidth-2).
		Height(contentHeight-3).
		Padding(0, 1).
		Render(dashboard.String())

	dashPanel := borderStyle.
		Width(contentWidth).
		Height(contentHeight - 1).
		Render(dashContent)

	return lipgloss.JoinVertical(lipgloss.Left, dashPanel, footerLine)
}

func main() {
	themeFlag := flag.String("theme", "", "Color theme to use (basic, catppuccin-mocha)")
	listThemesFlag := flag.Bool("list-themes", false, "List available themes and exit")
	flag.Parse()

	// Handle --list-themes
	if *listThemesFlag {
		fmt.Println("Available themes:")
		for _, name := range listThemes() {
			fmt.Printf("  - %s\n", name)
		}
		return
	}

	// Apply theme if specified
	if *themeFlag != "" {
		if t, ok := getThemeByName(*themeFlag); ok {
			setTheme(t)
		} else {
			fmt.Printf("Unknown theme: %s\nAvailable themes:\n", *themeFlag)
			for _, name := range listThemes() {
				fmt.Printf("  - %s\n", name)
			}
			os.Exit(1)
		}
	}

	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
