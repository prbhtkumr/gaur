package main

import (
	"path/filepath"
	"strings"
	"sync"
)

// AurDriver defines the strategy for interacting with an AUR helper binary.
type AurDriver interface {
	// Name returns the identifier of the AUR helper (e.g., "paru", "yay").
	Name() string
	// BuildCommand generates command-line arguments for a given action.
	BuildCommand(action string, flags string, args ...string) []string
	// StatsArgs returns the arguments used to fetch package and repository statistics.
	StatsArgs() []string
	// ResolveCacheDir resolves the specific clone/download cache directory.
	ResolveCacheDir(userCacheDir, customDir string) string
	// ResolveBaseCacheDir resolves the parent cache root for disk accounting.
	ResolveBaseCacheDir(userCacheDir, customDir string) string
}

type driverRegistry struct {
	mu      sync.RWMutex
	drivers map[string]AurDriver
}

var registry = driverRegistry{
	drivers: map[string]AurDriver{
		"paru": ParuDriver{},
		"yay":  YayDriver{},
	},
}

// RegisterAurDriver registers or overrides an AUR helper driver.
func RegisterAurDriver(driver AurDriver) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.drivers[strings.ToLower(strings.TrimSpace(driver.Name()))] = driver
}

// GetAurDriver retrieves the driver for the given helper name, defaulting to ParuDriver.
func GetAurDriver(name string) AurDriver {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	normalized := strings.ToLower(strings.TrimSpace(name))
	if driver, exists := registry.drivers[normalized]; exists {
		return driver
	}
	return registry.drivers["paru"]
}

// IsSupportedAurHelper returns whether the specified helper has a registered driver.
func IsSupportedAurHelper(name string) bool {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	normalized := strings.ToLower(strings.TrimSpace(name))
	_, exists := registry.drivers[normalized]
	return exists
}

// GetSupportedAurHelpers returns the list of all supported AUR helper names in consistent order.
func GetSupportedAurHelpers() []string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	// Return paru and yay first in predictable order
	result := []string{"paru", "yay"}
	for name := range registry.drivers {
		if name != "paru" && name != "yay" {
			result = append(result, name)
		}
	}
	return result
}

// cleanSearchArgs strips leading hyphens from search terms.
func cleanSearchArgs(args []string) []string {
	var clean []string
	for _, arg := range args {
		trimmed := strings.TrimLeft(arg, "-")
		if trimmed != "" {
			clean = append(clean, trimmed)
		}
	}
	return clean
}

// ----------------------------------------------------------------------------
// Paru Driver
// ----------------------------------------------------------------------------

// ParuDriver implements AurDriver for the 'paru' AUR helper.
type ParuDriver struct{}

func (p ParuDriver) Name() string {
	return "paru"
}

func (p ParuDriver) StatsArgs() []string {
	return []string{"-Ps"}
}

func (p ParuDriver) ResolveCacheDir(userCacheDir, customDir string) string {
	if customDir != "" {
		return customDir
	}
	return filepath.Join(userCacheDir, "paru", "clone")
}

func (p ParuDriver) ResolveBaseCacheDir(userCacheDir, customDir string) string {
	if customDir != "" {
		return customDir
	}
	return filepath.Join(userCacheDir, "paru")
}

func (p ParuDriver) BuildCommand(action string, flags string, args ...string) []string {
	var cmd []string
	switch action {
	case "install":
		cmd = []string{"paru", "-S"}
		if flags != "" {
			cmd = append(cmd, TokenizeFlags(flags)...)
		}
	case "remove":
		cmd = []string{"paru"}
		if flags != "" {
			cmd = append(cmd, TokenizeFlags(flags)...)
		} else {
			cmd = append(cmd, "-Rns")
		}
	case "update":
		cmd = []string{"paru", "-Qu"}
	case "search":
		cmd = []string{"paru", "-Ss", "-a"}
		return append(cmd, cleanSearchArgs(args)...)
	case "dash":
		cmd = []string{"paru", "-Si"}
	case "check-updates":
		cmd = []string{"paru", "-Qu"}
	case "sync":
		cmd = []string{"paru", "-Sy"}
	case "full-update":
		cmd = []string{"paru", "-Syu"}
	default:
		cmd = []string{"paru"}
	}
	return append(cmd, args...)
}

// ----------------------------------------------------------------------------
// Yay Driver
// ----------------------------------------------------------------------------

// YayDriver implements AurDriver for the 'yay' AUR helper.
type YayDriver struct{}

func (y YayDriver) Name() string {
	return "yay"
}

func (y YayDriver) StatsArgs() []string {
	return []string{"-Ps"}
}

func (y YayDriver) ResolveCacheDir(userCacheDir, customDir string) string {
	if customDir != "" {
		return customDir
	}
	return filepath.Join(userCacheDir, "yay")
}

func (y YayDriver) ResolveBaseCacheDir(userCacheDir, customDir string) string {
	if customDir != "" {
		return customDir
	}
	return filepath.Join(userCacheDir, "yay")
}

func (y YayDriver) BuildCommand(action string, flags string, args ...string) []string {
	var cmd []string
	switch action {
	case "install":
		cmd = []string{"yay", "-S"}
		if flags != "" {
			cmd = append(cmd, TokenizeFlags(flags)...)
		}
	case "remove":
		cmd = []string{"yay"}
		if flags != "" {
			cmd = append(cmd, TokenizeFlags(flags)...)
		} else {
			cmd = append(cmd, "-Rns")
		}
	case "update":
		cmd = []string{"yay", "-Qu"}
	case "search":
		cmd = []string{"yay", "-Ss", "-a"}
		return append(cmd, cleanSearchArgs(args)...)
	case "dash":
		cmd = []string{"yay", "-Si"}
	case "check-updates":
		cmd = []string{"yay", "-Qu"}
	case "sync":
		cmd = []string{"yay", "-Sy"}
	case "full-update":
		cmd = []string{"yay", "-Syu"}
	default:
		cmd = []string{"yay"}
	}
	return append(cmd, args...)
}
