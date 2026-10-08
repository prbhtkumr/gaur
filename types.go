package main

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Config represents the TOML configuration structure
type Config struct {
	Startup  StartupConfig  `toml:"startup"`
	UI       UIConfig       `toml:"ui"`
	Commands CommandConfig  `toml:"commands"`
	Advanced AdvancedConfig `toml:"advanced"`
	Keys     KeyConfig      `toml:"keys"`
	Logging  LogConfig      `toml:"logging"`
}

type StartupConfig struct {
	DefaultMode string `toml:"default_mode"`
}

type UIConfig struct {
	Theme      string `toml:"theme"`
	BorderType string `toml:"border_type"`
}

type CommandConfig struct {
	AurHelper    string `toml:"aur_helper"`
	InstallFlags string `toml:"install_flags"`
	RemoveFlags  string `toml:"remove_flags"`
	CacheTool    string `toml:"cache_tool"`
}

type AdvancedConfig struct {
	DebounceMs int    `toml:"debounce_ms"`
	CacheDir   string `toml:"cache_dir"`
}

type LogConfig struct {
	Level string `toml:"level"` // off, error, warn, info, debug, verbose
}

type KeyConfig struct {
	Quit          []string `toml:"quit"`
	InstallMode   []string `toml:"install_mode"`
	RemoveMode    []string `toml:"remove_mode"`
	UpdateMode    []string `toml:"update_mode"`
	DashboardMode []string `toml:"dashboard_mode"`
	Search        string   `toml:"search"`
	Mark          string   `toml:"mark"`
	Selective     string   `toml:"selective"`
	Settings      string   `toml:"settings"`
	Confirm       string   `toml:"confirm"`
	Cancel        string   `toml:"cancel"`
}

// KeyMap defines the application's keybindings using charmbracelet/bubbles/key
type KeyMap struct {
	Quit          key.Binding
	InstallMode   key.Binding
	RemoveMode    key.Binding
	UpdateMode    key.Binding
	DashboardMode key.Binding
	Search        key.Binding
	Mark          key.Binding
	Selective     key.Binding
	Settings      key.Binding
	Confirm       key.Binding
	Cancel        key.Binding
}

// CommandRunner defines an interface for executing shell commands.
type CommandRunner interface {
	Run(name string, args ...string) ([]byte, error)
	RunContext(ctx context.Context, name string, args ...string) ([]byte, error)
	RunWithInput(input string, name string, args ...string) ([]byte, error)
	RunWithInputContext(ctx context.Context, input string, name string, args ...string) ([]byte, error)
	Interactive(onExit func(error) tea.Msg, name string, args ...string) tea.Cmd
	RunWithStderrScan(name string, onLine func(string), args ...string) error
}

// RealCommandRunner implements CommandRunner using os/exec.
type RealCommandRunner struct{}

// Run executes a command and returns the combined output.
// Security note: Commands are validated by ValidateConfig and only trusted binaries
// (paru, yay, pacman, paccache) are used. Package names are sanitized before use.
func (r RealCommandRunner) Run(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return r.RunContext(ctx, name, args...)
}

// RunContext executes a command with context cancellation and returns the combined output.
// Security note: Binaries are validated in config (aur_helper, cache_tool) or fixed system utilities (pacman, fzf, sudo, rm).
func (r RealCommandRunner) RunContext(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput() // #nosec G204 - executable and arguments are validated
}

// RunWithInput executes a command with stdin input and returns the combined output.
// Security note: Binaries are validated in config (aur_helper, cache_tool) or fixed system utilities (pacman, fzf, sudo, rm).
func (r RealCommandRunner) RunWithInput(input string, name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return r.RunWithInputContext(ctx, input, name, args...)
}

// RunWithInputContext executes a command with stdin input and context cancellation.
func (r RealCommandRunner) RunWithInputContext(ctx context.Context, input string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...) // #nosec G204 - executable and arguments are validated
	cmd.Stdin = strings.NewReader(input)
	return cmd.CombinedOutput()
}

// RunWithStderrScan executes a command and streams each line of stderr to onLine.
func (r RealCommandRunner) RunWithStderrScan(name string, onLine func(string), args ...string) error {
	cmd := exec.Command(name, args...) // #nosec G204 - executable and arguments are validated
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		if onLine != nil {
			onLine(scanner.Text())
		}
	}
	return cmd.Wait()
}

// filterResultMsg is delivered asynchronously after fuzzy filtering finishes.
type filterResultMsg struct {
	query        string
	mode         viewMode
	packages     []Package
	matchIndices map[int][]int
}

// Interactive executes a command interactively using tea.ExecProcess.
// Security note: Binaries are validated in config (aur_helper, cache_tool) or fixed system utilities (pacman, fzf, sudo, rm).
func (r RealCommandRunner) Interactive(onExit func(error) tea.Msg, name string, args ...string) tea.Cmd {
	return tea.ExecProcess(exec.Command(name, args...), onExit) // #nosec G204 - executable and arguments are validated
}

var runner CommandRunner = RealCommandRunner{}

// getActiveRunner returns the first provided CommandRunner, or the global runner if none provided.
func getActiveRunner(r ...CommandRunner) CommandRunner {
	if len(r) > 0 && r[0] != nil {
		return r[0]
	}
	return runner
}

// View modes for the TUI application
type viewMode int

const (
	modeDashboard viewMode = iota
	modeInstall
	modeUpdate          // Viewing available updates
	modeUpdateSelective // Selecting specific updates
	modeRemove
	modeCacheMenu      // Menu for selecting cache clearing strategy
	modeCacheSelective // Selecting specific packages to clear from cache
	modeSettings       // In-app settings overlay
)

// SettingItem represents a single configurable setting in the carousel
type SettingItem struct {
	Label       string
	ConfigKey   string // Path to config like "ui.theme"
	Options     []string
	ActiveIndex int
}

// Confirmation operation types
type confirmationType int

const (
	confirmInstall confirmationType = iota
	confirmRemove
	confirmUpdate
	confirmSelectiveUpdate
	confirmCleanKeep3     // paccache -r
	confirmCleanKeep1     // paccache -rk1
	confirmCleanRemoved   // paccache -ruk0
	confirmCleanNuke      // paccache -rk0
	confirmCleanSelective // Custom selective clean
	confirmRemoveOrphans
)

// UI configuration constants
const (
	minSearchQueryLen = 2
)

// Package represents a package with its source and name
type Package struct {
	Source      string // core, extra, multilib, aur
	Name        string
	Version     string
	Description string
	Installed   bool
	Explicit    bool   // Explicitly installed (not a dependency)
	Orphan      bool   // Orphan package (no longer required)
	Size        string // For formatted sizes, like in cache hogs
	SizeBytes   int64  // Raw size
}

func (p Package) String() string {
	return fmt.Sprintf("%s/%s", p.Source, p.Name)
}

// Messages
type repoPackagesMsg struct {
	packages []Package
	err      error
}

type aurSearchMsg struct {
	packages  []Package
	query     string
	timeTaken time.Duration
	err       error
}

type packageDetailsMsg struct {
	details     string
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
	done bool
	err  error
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

// debounceTickMsg is sent after debounce timer expires to trigger package details fetch
type debounceTickMsg struct {
	packageName string
}

// DashboardData holds system package statistics
type DashboardData struct {
	TotalPackages        int
	ExplicitlyInstalled  int
	ForeignPackages      int
	RepoDistribution     map[string]int // core, extra, multilib, etc.
	TotalSize            string
	TotalSizeBytes       int64 // For comparison
	CleanerSize          string
	CleanerSizeBytes     int64 // For comparison and coloring
	PacmanCacheSize      string
	PacmanCacheSizeBytes int64
	PacmanCachePath      string
	AurCacheSize         string
	AurCacheSizeBytes    int64
	AurCachePath         string
	Orphans              int
	MissingFromAUR       int
	TopPackages          []PackageSize               // Top 10 packages by size
	RecentlyInstalled    []RecentPackage             // Details of 5 recently installed packages
	TopCacheHogs         []PackageSize               // Top 5 packages taking up cache space
	AllCacheHogs         []PackageSize               // All packages taking up cache space
	RemovedPacmanCache   []PackageSize               // Removed packages in pacman cache
	RemovedAurCache      []PackageSize               // Removed packages in AUR helper cache
	CacheFreedPacman     map[confirmationType]string // Estimated savings for pacman
	CacheFreedAur        map[confirmationType]string // Estimated savings for AUR helper
	CacheFreedEstimates  map[confirmationType]string // Total estimated savings
	// Disk usage dash
	DiskTotal       string
	DiskUsed        string
	DiskFree        string
	DiskUsedPercent float64
}

// PackageSize holds package name and its installed size
type PackageSize struct {
	Name      string
	Size      string
	SizeBytes int64
}

// RecentPackage holds details about a recently installed package
type RecentPackage struct {
	Name      string
	Timestamp string // e.g. "2024-03-12 10:00"
}

type syncRepositoriesMsg struct {
	err error
}
