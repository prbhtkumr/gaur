package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// MirrorSortOption represents a sorting criterion for mirrors
type MirrorSortOption struct {
	Name        string
	Flag        string // reflector flag
	Description string
}

// MirrorProtocol represents a protocol filter option
type MirrorProtocol struct {
	Name    string
	Flag    string
	Enabled bool
}

// MirrorCountry represents a country for mirror filtering
type MirrorCountry struct {
	Name string
	Code string
}

// MirrorConfig holds the current mirror configuration choices
type MirrorConfig struct {
	SortBy       int  // Index into MirrorSortOptions
	CountryIndex int  // Index into countries list (0 = worldwide)
	Latest       int  // Number of most recently synced mirrors to use
	Protocol     int  // Index into protocols
	Save         bool // Whether to save to /etc/pacman.d/mirrorlist
}

// MirrorSortOptions defines available sorting options for reflector
var MirrorSortOptions = []MirrorSortOption{
	{Name: "Rate", Flag: "rate", Description: "Download speed (slowest to fastest)"},
	{Name: "Age", Flag: "age", Description: "Sync age (oldest to newest)"},
	{Name: "Score", Flag: "score", Description: "MirrorStatus score (worst to best)"},
	{Name: "Country", Flag: "country", Description: "Country name alphabetically"},
	{Name: "Delay", Flag: "delay", Description: "Sync delay (highest to lowest)"},
}

// MirrorProtocols defines available protocol options
var MirrorProtocols = []MirrorProtocol{
	{Name: "HTTPS", Flag: "https", Enabled: true},
	{Name: "HTTP", Flag: "http", Enabled: false},
	{Name: "Both", Flag: "", Enabled: false}, // Empty flag means no filter
}

// MirrorCountries defines available countries (subset of common ones)
var MirrorCountries = []MirrorCountry{
	{Name: "Worldwide", Code: ""},
	{Name: "Australia", Code: "AU"},
	{Name: "Austria", Code: "AT"},
	{Name: "Belgium", Code: "BE"},
	{Name: "Brazil", Code: "BR"},
	{Name: "Canada", Code: "CA"},
	{Name: "China", Code: "CN"},
	{Name: "Denmark", Code: "DK"},
	{Name: "Finland", Code: "FI"},
	{Name: "France", Code: "FR"},
	{Name: "Germany", Code: "DE"},
	{Name: "Hong Kong", Code: "HK"},
	{Name: "India", Code: "IN"},
	{Name: "Ireland", Code: "IE"},
	{Name: "Italy", Code: "IT"},
	{Name: "Japan", Code: "JP"},
	{Name: "Netherlands", Code: "NL"},
	{Name: "New Zealand", Code: "NZ"},
	{Name: "Norway", Code: "NO"},
	{Name: "Poland", Code: "PL"},
	{Name: "Portugal", Code: "PT"},
	{Name: "Russia", Code: "RU"},
	{Name: "Singapore", Code: "SG"},
	{Name: "South Korea", Code: "KR"},
	{Name: "Spain", Code: "ES"},
	{Name: "Sweden", Code: "SE"},
	{Name: "Switzerland", Code: "CH"},
	{Name: "Taiwan", Code: "TW"},
	{Name: "Ukraine", Code: "UA"},
	{Name: "United Kingdom", Code: "GB"},
	{Name: "United States", Code: "US"},
}

// DefaultMirrorConfig returns sensible defaults
func DefaultMirrorConfig() MirrorConfig {
	return MirrorConfig{
		SortBy:       0, // Rate (speed)
		CountryIndex: 0, // Worldwide
		Latest:       20,
		Protocol:     0, // HTTPS only
		Save:         true,
	}
}

// MirrorOverlayItem represents a configurable item in the mirror overlay
type MirrorOverlayItem int

const (
	mirrorItemSortBy MirrorOverlayItem = iota
	mirrorItemCountry
	mirrorItemLatest
	mirrorItemProtocol
)

const mirrorItemCount = 4

// mirrorUpdateMsg is sent when mirror update completes
type mirrorUpdateMsg struct {
	success bool
	err     error
}

// mirrorSudoReadyMsg is sent after sudo credentials are acquired interactively
type mirrorSudoReadyMsg struct {
	err error
}

// acquireSudoForMirror suspends the TUI and runs "sudo -v" to cache credentials,
// so the subsequent background reflector process won't need to prompt for a password.
func acquireSudoForMirror(r ...CommandRunner) tea.Cmd {
	return getActiveRunner(r...).Interactive(func(err error) tea.Msg {
		return mirrorSudoReadyMsg{err: err}
	}, "sudo", "-v")
}

// BuildReflectorCommand constructs the reflector command from the config
func BuildReflectorCommand(cfg MirrorConfig) []string {
	args := []string{}

	// Add latest mirrors count
	args = append(args, "--latest", fmt.Sprintf("%d", cfg.Latest))

	// Add sort option
	if cfg.SortBy >= 0 && cfg.SortBy < len(MirrorSortOptions) {
		args = append(args, "--sort", MirrorSortOptions[cfg.SortBy].Flag)
	}

	// Add country filter if not worldwide
	if cfg.CountryIndex > 0 && cfg.CountryIndex < len(MirrorCountries) {
		args = append(args, "--country", MirrorCountries[cfg.CountryIndex].Code)
	}

	// Add protocol filter
	if cfg.Protocol >= 0 && cfg.Protocol < len(MirrorProtocols) {
		proto := MirrorProtocols[cfg.Protocol]
		if proto.Flag != "" {
			args = append(args, "--protocol", proto.Flag)
		}
	}

	// Add save option
	if cfg.Save {
		args = append(args, "--save", "/etc/pacman.d/mirrorlist")
	}

	return args
}

// ValidateMirrorConfig ensures mirror configuration values are within bounds
func ValidateMirrorConfig(cfg *MirrorConfig) {
	if cfg.SortBy < 0 || cfg.SortBy >= len(MirrorSortOptions) {
		cfg.SortBy = 0
	}
	if cfg.CountryIndex < 0 || cfg.CountryIndex >= len(MirrorCountries) {
		cfg.CountryIndex = 0
	}
	if cfg.Latest < 1 {
		cfg.Latest = 1
	}
	if cfg.Latest > 100 {
		cfg.Latest = 100
	}
	if cfg.Protocol < 0 || cfg.Protocol >= len(MirrorProtocols) {
		cfg.Protocol = 0
	}
}

// GetMirrorSortOptions returns the list of sort option names
func GetMirrorSortOptions() []string {
	return mapSlice(MirrorSortOptions, func(opt MirrorSortOption) string { return opt.Name })
}

// GetMirrorCountryNames returns the list of country names
func GetMirrorCountryNames() []string {
	return mapSlice(MirrorCountries, func(c MirrorCountry) string { return c.Name })
}

// GetMirrorProtocolNames returns the list of protocol names
func GetMirrorProtocolNames() []string {
	return mapSlice(MirrorProtocols, func(p MirrorProtocol) string { return p.Name })
}

// mirrorProgressMsg is sent for each line of reflector output during mirror update
type mirrorProgressMsg struct {
	current int
	total   int
	ch      chan tea.Msg
}

// executeMirrorUpdate runs reflector with the given configuration,
// streaming stderr to report per-mirror progress back to the TUI.
func executeMirrorUpdate(cfg MirrorConfig, r ...CommandRunner) tea.Cmd {
	// Validate config indices before launching goroutine to prevent panics
	ValidateMirrorConfig(&cfg)
	activeRunner := getActiveRunner(r...)

	return func() tea.Msg {
		ch := make(chan tea.Msg, 1)

		go func() {
			LogInfo("MIRROR", "Starting mirror update with reflector")
			LogDebug("MIRROR", "Config: sort=%s, country=%s, latest=%d, protocol=%s",
				MirrorSortOptions[cfg.SortBy].Name,
				MirrorCountries[cfg.CountryIndex].Name,
				cfg.Latest,
				MirrorProtocols[cfg.Protocol].Name)

			args := BuildReflectorCommand(cfg)
			LogDebug("MIRROR", "Command: sudo reflector %s", strings.Join(args, " "))

			// reflector needs sudo to write to /etc/pacman.d/mirrorlist
			fullArgs := append([]string{"reflector"}, args...)

			if _, isReal := activeRunner.(RealCommandRunner); !isReal {
				// For mock / injected runners, execute through activeRunner
				_, err := activeRunner.Run("sudo", fullArgs...)
				if err != nil {
					LogError("MIRROR", "Reflector failed: %v", err)
					ch <- mirrorUpdateMsg{success: false, err: fmt.Errorf("reflector failed: %w", err)}
					return
				}
				LogInfo("MIRROR", "Mirror update completed successfully")
				ch <- mirrorUpdateMsg{success: true}
				return
			}

			cmd := exec.Command("sudo", fullArgs...) // #nosec G204 - reflector arguments are strictly constructed from structured MirrorConfig and hardcoded flags
			stderr, err := cmd.StderrPipe()
			if err != nil {
				LogError("MIRROR", "Failed to create stderr pipe: %v", err)
				ch <- mirrorUpdateMsg{success: false, err: fmt.Errorf("reflector failed: %w", err)}
				return
			}

			if err := cmd.Start(); err != nil {
				LogError("MIRROR", "Failed to start reflector: %v", err)
				ch <- mirrorUpdateMsg{success: false, err: fmt.Errorf("reflector failed: %w", err)}
				return
			}

			total := cfg.Latest
			current := 0
			scanner := bufio.NewScanner(stderr)
			for scanner.Scan() {
				current++
				progress := current
				if progress > total {
					progress = total
				}
				ch <- mirrorProgressMsg{current: progress, total: total, ch: ch}
			}

			if scanErr := scanner.Err(); scanErr != nil {
				LogError("MIRROR", "Error reading reflector output: %v", scanErr)
			}

			err = cmd.Wait()
			if err != nil {
				LogError("MIRROR", "Reflector failed: %v", err)
				ch <- mirrorUpdateMsg{success: false, err: fmt.Errorf("reflector failed: %w", err)}
				return
			}

			LogInfo("MIRROR", "Mirror update completed successfully")
			ch <- mirrorUpdateMsg{success: true}
		}()

		// Wait for the first message from the channel
		return <-ch
	}
}

// waitForMirrorProgress returns a tea.Cmd that reads the next message from the channel
func waitForMirrorProgress(ch chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

// checkReflectorInstalled verifies reflector is available
func checkReflectorInstalled(r ...CommandRunner) bool {
	_, err := getActiveRunner(r...).Run("which", "reflector")
	return err == nil
}

// GetReflectorCommandPreview returns a preview of the command that would be run
func GetReflectorCommandPreview(cfg MirrorConfig) string {
	args := BuildReflectorCommand(cfg)
	return fmt.Sprintf("sudo reflector %s", strings.Join(args, " "))
}

// SortCountriesByName returns countries sorted alphabetically (keeping Worldwide first)
func SortCountriesByName() []MirrorCountry {
	sorted := make([]MirrorCountry, len(MirrorCountries))
	copy(sorted, MirrorCountries)

	// Sort all except the first one (Worldwide)
	if len(sorted) > 1 {
		toSort := sorted[1:]
		sort.Slice(toSort, func(i, j int) bool {
			return toSort[i].Name < toSort[j].Name
		})
	}

	return sorted
}

// FindCountryByCode returns the index of a country by its code
func FindCountryByCode(code string) int {
	for i, c := range MirrorCountries {
		if c.Code == code {
			return i
		}
	}
	return 0 // Default to Worldwide
}

// FindCountryByName returns the index of a country by its name
func FindCountryByName(name string) int {
	nameLower := strings.ToLower(name)
	for i, c := range MirrorCountries {
		if strings.ToLower(c.Name) == nameLower {
			return i
		}
	}
	return 0 // Default to Worldwide
}
