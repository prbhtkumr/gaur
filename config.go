package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/pelletier/go-toml/v2"
)

const (
	defaultConfigDir      = "gaur"
	defaultConfigFile     = "config.toml"
	textInputCharLimit    = 100
	textInputDefaultWidth = 50
)

// DefaultConfig returns the fallback configuration
func DefaultConfig() Config {
	return Config{
		Startup: StartupConfig{
			DefaultMode: "install",
		},
		UI: UIConfig{
			Theme:      "catppuccin-mocha",
			BorderType: "rounded",
		},
		Commands: CommandConfig{
			AurHelper:    "paru",
			InstallFlags: "",
			RemoveFlags:  "-Rns",
			CacheTool:    "paccache",
		},
		Advanced: AdvancedConfig{
			DebounceMs: 150,
			CacheDir:   "",
		},
		Keys: KeyConfig{
			Quit:          []string{"q", "ctrl+c"},
			InstallMode:   []string{"i", "alt+2"},
			RemoveMode:    []string{"r", "alt+4"},
			UpdateMode:    []string{"u", "alt+3"},
			DashboardMode: []string{"d", "alt+1"},
			Search:        "/",
			Mark:          "tab",
			Selective:     "s",
			Settings:      ",",
			Confirm:       "enter",
			Cancel:        "esc",
		},
		Logging: LogConfig{
			Level: "info",
		},
	}
}

// LoadConfig resolves the config path, ensures it exists, and parses it
func LoadConfig() (Config, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return DefaultConfig(), err
	}

	fullDir := filepath.Join(configDir, defaultConfigDir)
	configPath := filepath.Join(fullDir, defaultConfigFile)

	// Validate that the path is within the expected config directory (prevent path traversal)
	cleanPath := filepath.Clean(configPath)
	if !filepath.IsAbs(cleanPath) {
		return DefaultConfig(), fmt.Errorf("config path must be absolute")
	}

	// Ensure directory exists with restrictive permissions (0750 or less)
	if _, err := os.Stat(fullDir); os.IsNotExist(err) {
		if err := os.MkdirAll(fullDir, 0750); err != nil {
			return DefaultConfig(), err
		}
	}

	// Ensure config file exists, if not write default
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		cfg := DefaultConfig()
		if err := saveConfig(cleanPath, cfg); err != nil {
			return cfg, nil // Return default even if save fails
		}
		return cfg, nil
	}

	// Read and parse using the cleaned path
	data, err := os.ReadFile(cleanPath) // #nosec G304 - path is constructed from trusted os.UserConfigDir()
	if err != nil {
		return DefaultConfig(), err
	}

	cfg := DefaultConfig()
	if err := toml.Unmarshal(data, &cfg); err != nil {
		// Log to file if possible (logger may not be initialized yet)
		LogError("CONFIG", "Failed to parse config file: %v", err)
		return DefaultConfig(), fmt.Errorf("failed to parse config file: %w", err)
	}

	ValidateConfig(&cfg)
	LogDebug("CONFIG", "Configuration loaded from %s", cleanPath)
	return cfg, nil
}

// ValidateConfig ensures the configuration values are supported and safe.
func ValidateConfig(c *Config) {
	def := DefaultConfig()

	helper := strings.ToLower(strings.TrimSpace(c.Commands.AurHelper))
	if helper == "" || (helper != "paru" && helper != "yay") {
		LogWarn("CONFIG", "Unsupported AUR helper '%s'. Resetting to 'paru'.", c.Commands.AurHelper)
		c.Commands.AurHelper = "paru"
	} else {
		c.Commands.AurHelper = helper
	}

	// Validate CacheTool - only allow known safe tools
	tool := strings.TrimSpace(c.Commands.CacheTool)
	if tool != "paccache" {
		if tool != "" {
			LogWarn("CONFIG", "Unsupported cache tool '%s'. Resetting to 'paccache'.", c.Commands.CacheTool)
		}
		c.Commands.CacheTool = "paccache"
	} else {
		c.Commands.CacheTool = tool
	}

	// Validate install and remove flags against allowlist (S-04)
	c.Commands.InstallFlags = validateInstallFlags(c.Commands.InstallFlags)
	c.Commands.RemoveFlags = validateRemoveFlags(c.Commands.RemoveFlags)

	// Guard DebounceMs
	if c.Advanced.DebounceMs <= 0 {
		c.Advanced.DebounceMs = 150
	}

	// Clean and validate CacheDir if provided (S-08)
	if c.Advanced.CacheDir != "" {
		cleaned := filepath.Clean(c.Advanced.CacheDir)
		isAllowed := false

		// 1. System pacman cache root
		if cleaned == "/var/cache/pacman/pkg" || strings.HasPrefix(cleaned, "/var/cache/pacman/pkg/") {
			isAllowed = true
		}

		// 2. User XDG cache root
		if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
			cleanXDG := filepath.Clean(xdgCache)
			if cleaned == cleanXDG || strings.HasPrefix(cleaned, cleanXDG+string(filepath.Separator)) {
				isAllowed = true
			}
		}

		// 3. User ~/.cache root
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			dotCache := filepath.Join(home, ".cache")
			if cleaned == dotCache || strings.HasPrefix(cleaned, dotCache+string(filepath.Separator)) {
				isAllowed = true
			}
		}

		if !isAllowed || !filepath.IsAbs(cleaned) {
			LogWarn("CONFIG", "Cache directory '%s' is not in allowed cache roots. Resetting to default.", c.Advanced.CacheDir)
			c.Advanced.CacheDir = ""
		} else {
			c.Advanced.CacheDir = cleaned
		}
	}

	// Validate log level
	validLevels := map[string]bool{"off": true, "error": true, "warn": true, "info": true, "debug": true, "verbose": true}
	level := strings.ToLower(strings.TrimSpace(c.Logging.Level))
	if level == "" {
		c.Logging.Level = "info"
	} else if !validLevels[level] {
		LogWarn("CONFIG", "Unknown log level '%s'. Resetting to 'info'.", c.Logging.Level)
		c.Logging.Level = "info"
	} else {
		c.Logging.Level = level
	}

	// Ensure critical keybindings are present
	if len(c.Keys.Quit) == 0 {
		c.Keys.Quit = def.Keys.Quit
	}
	if len(c.Keys.Cancel) == 0 {
		c.Keys.Cancel = def.Keys.Cancel
	}
	if len(c.Keys.Confirm) == 0 {
		c.Keys.Confirm = def.Keys.Confirm
	}
	if len(c.Keys.Search) == 0 {
		c.Keys.Search = def.Keys.Search
	}
}

func saveConfig(path string, cfg Config) error {
	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, "config-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if err := tmpFile.Chmod(0600); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// NewKeyMap creates a KeyMap from the configuration
func NewKeyMap(k KeyConfig) KeyMap {
	return KeyMap{
		Quit:          newBinding("quit", k.Quit...),
		InstallMode:   newBinding("install mode", k.InstallMode...),
		RemoveMode:    newBinding("remove mode", k.RemoveMode...),
		UpdateMode:    newBinding("update mode", k.UpdateMode...),
		DashboardMode: newBinding("dashboard mode", k.DashboardMode...),
		Search:        newBinding("search", k.Search),
		Mark:          newBinding("mark", k.Mark),
		Selective:     newBinding("selective", k.Selective),
		Settings:      newBinding("settings", k.Settings),
		Confirm:       newBinding("confirm", k.Confirm),
		Cancel:        newBinding("cancel", k.Cancel),
	}
}

func newBinding(help string, keys ...string) key.Binding {
	return key.NewBinding(
		key.WithKeys(keys...),
		key.WithHelp("", help),
	)
}

// TokenizeFlags safely splits a flag string into a slice of strings
// Simple implementation using strings.Fields, could be improved with shlex if needed
func TokenizeFlags(flags string) []string {
	if strings.TrimSpace(flags) == "" {
		return nil
	}
	return strings.Fields(flags)
}

var allowedInstallFlags = map[string]bool{
	"--noconfirm":     true,
	"--needed":        true,
	"--asdeps":        true,
	"--asexplicit":    true,
	"--noprogressbar": true,
	"--quiet":         true,
	"-q":              true,
	"--debug":         true,
	"--clean":         true,
	"--rebuild":       true,
	"--redownload":    true,
	"--sudoloop":      true,
	"--nodiffmenu":    true,
	"--noeditmenu":    true,
	"--noupgrademenu": true,
	"--removemake":    true,
	"--topdown":       true,
	"--bottomup":      true,
}

var allowedRemoveFlags = map[string]bool{
	"--noconfirm":     true,
	"--nosave":        true,
	"-n":              true,
	"--recursive":     true,
	"-s":              true,
	"--unneeded":      true,
	"-u":              true,
	"--cascade":       true,
	"-c":              true,
	"--noprogressbar": true,
	"--quiet":         true,
	"-q":              true,
}

func isAllowedInstallFlag(token string) bool {
	return allowedInstallFlags[token]
}

func isAllowedRemoveFlag(token string) bool {
	if allowedRemoveFlags[token] {
		return true
	}
	if strings.HasPrefix(token, "-R") {
		rest := token[2:]
		if len(rest) == 0 {
			return true // bare -R
		}
		for _, r := range rest {
			if r != 'n' && r != 's' && r != 'c' && r != 'u' {
				return false
			}
		}
		return true
	}
	return false
}

func validateInstallFlags(raw string) string {
	tokens := TokenizeFlags(raw)
	if len(tokens) == 0 {
		return ""
	}
	var validTokens []string
	for _, tok := range tokens {
		if isAllowedInstallFlag(tok) {
			validTokens = append(validTokens, tok)
		} else {
			LogWarn("CONFIG", "Disallowed install flag '%s'. Stripping.", tok)
		}
	}
	return strings.Join(validTokens, " ")
}

func validateRemoveFlags(raw string) string {
	tokens := TokenizeFlags(raw)
	if len(tokens) == 0 {
		return "-Rns"
	}
	var validTokens []string
	for _, tok := range tokens {
		if isAllowedRemoveFlag(tok) {
			validTokens = append(validTokens, tok)
		} else {
			LogWarn("CONFIG", "Disallowed remove flag '%s'. Stripping.", tok)
		}
	}
	if len(validTokens) == 0 {
		return "-Rns"
	}
	hasAction := false
	for _, tok := range validTokens {
		if strings.HasPrefix(tok, "-R") {
			hasAction = true
			break
		}
	}
	if !hasAction {
		validTokens = append([]string{"-R"}, validTokens...)
	}
	return strings.Join(validTokens, " ")
}
