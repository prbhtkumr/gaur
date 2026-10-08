package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestAurDriver_RegistryAndLookup(t *testing.T) {
	// Defaults
	paru := GetAurDriver("paru")
	if paru.Name() != "paru" {
		t.Errorf("expected paru driver, got %s", paru.Name())
	}

	yay := GetAurDriver("yay")
	if yay.Name() != "yay" {
		t.Errorf("expected yay driver, got %s", yay.Name())
	}

	// Normalization
	if GetAurDriver("  PARU  ").Name() != "paru" {
		t.Errorf("expected trimmed uppercase PARU to resolve to paru")
	}
	if GetAurDriver("YAY").Name() != "yay" {
		t.Errorf("expected uppercase YAY to resolve to yay")
	}

	// Unknown falls back to paru
	if GetAurDriver("unknown_helper").Name() != "paru" {
		t.Errorf("expected unknown helper to fall back to paru")
	}

	// IsSupportedAurHelper
	if !IsSupportedAurHelper("paru") || !IsSupportedAurHelper("yay") {
		t.Errorf("expected paru and yay to be supported")
	}
	if IsSupportedAurHelper("pacman") || IsSupportedAurHelper("") {
		t.Errorf("expected pacman and empty string to not be supported")
	}

	// Supported list contains paru and yay
	helpers := GetSupportedAurHelpers()
	if len(helpers) < 2 || helpers[0] != "paru" || helpers[1] != "yay" {
		t.Errorf("unexpected supported helpers slice: %v", helpers)
	}
}

type mockCustomDriver struct{}

func (m mockCustomDriver) Name() string { return "customaur" }
func (m mockCustomDriver) BuildCommand(action, flags string, args ...string) []string {
	return []string{"customaur", action}
}
func (m mockCustomDriver) StatsArgs() []string { return []string{"--stats"} }
func (m mockCustomDriver) ResolveCacheDir(userCacheDir, customDir string) string {
	return filepath.Join(userCacheDir, "custom")
}
func (m mockCustomDriver) ResolveBaseCacheDir(userCacheDir, customDir string) string {
	return filepath.Join(userCacheDir, "custom")
}

func TestAurDriver_CustomRegistration(t *testing.T) {
	RegisterAurDriver(mockCustomDriver{})

	if !IsSupportedAurHelper("customaur") {
		t.Errorf("expected customaur to be registered as supported")
	}

	driver := GetAurDriver("customaur")
	if driver.Name() != "customaur" {
		t.Errorf("expected customaur driver, got %s", driver.Name())
	}

	if !reflect.DeepEqual(driver.StatsArgs(), []string{"--stats"}) {
		t.Errorf("unexpected stats args: %v", driver.StatsArgs())
	}
}

func TestParuDriver_BuildCommand(t *testing.T) {
	driver := ParuDriver{}

	tests := []struct {
		action   string
		flags    string
		args     []string
		expected []string
	}{
		{
			action:   "install",
			flags:    "--noconfirm",
			args:     []string{"htop", "neovim"},
			expected: []string{"paru", "-S", "--noconfirm", "htop", "neovim"},
		},
		{
			action:   "remove",
			flags:    "",
			args:     []string{"htop"},
			expected: []string{"paru", "-Rns", "htop"},
		},
		{
			action:   "remove",
			flags:    "-R",
			args:     []string{"htop"},
			expected: []string{"paru", "-R", "htop"},
		},
		{
			action:   "update",
			flags:    "",
			args:     nil,
			expected: []string{"paru", "-Qu"},
		},
		{
			action:   "search",
			flags:    "",
			args:     []string{"-q", "linux"},
			expected: []string{"paru", "-Ss", "-a", "q", "linux"},
		},
		{
			action:   "dash",
			flags:    "",
			args:     []string{"ripgrep"},
			expected: []string{"paru", "-Si", "ripgrep"},
		},
		{
			action:   "check-updates",
			flags:    "",
			args:     nil,
			expected: []string{"paru", "-Qu"},
		},
		{
			action:   "sync",
			flags:    "",
			args:     nil,
			expected: []string{"paru", "-Sy"},
		},
		{
			action:   "full-update",
			flags:    "",
			args:     nil,
			expected: []string{"paru", "-Syu"},
		},
		{
			action:   "unknown",
			flags:    "",
			args:     []string{"arg1"},
			expected: []string{"paru", "arg1"},
		},
	}

	for _, tt := range tests {
		actual := driver.BuildCommand(tt.action, tt.flags, tt.args...)
		if !reflect.DeepEqual(actual, tt.expected) {
			t.Errorf("action %q: got %v, want %v", tt.action, actual, tt.expected)
		}
	}
}

func TestYayDriver_BuildCommand(t *testing.T) {
	driver := YayDriver{}

	tests := []struct {
		action   string
		flags    string
		args     []string
		expected []string
	}{
		{
			action:   "install",
			flags:    "--needed",
			args:     []string{"curl"},
			expected: []string{"yay", "-S", "--needed", "curl"},
		},
		{
			action:   "remove",
			flags:    "",
			args:     []string{"curl"},
			expected: []string{"yay", "-Rns", "curl"},
		},
		{
			action:   "search",
			flags:    "",
			args:     []string{"git"},
			expected: []string{"yay", "-Ss", "-a", "git"},
		},
	}

	for _, tt := range tests {
		actual := driver.BuildCommand(tt.action, tt.flags, tt.args...)
		if !reflect.DeepEqual(actual, tt.expected) {
			t.Errorf("action %q: got %v, want %v", tt.action, actual, tt.expected)
		}
	}
}

func TestAurDriver_CacheDirResolution(t *testing.T) {
	fakeUserCache := "/home/user/.cache"
	customDir := "/mnt/custom/aur"

	paru := ParuDriver{}
	if paru.ResolveCacheDir(fakeUserCache, "") != filepath.Join(fakeUserCache, "paru", "clone") {
		t.Errorf("unexpected paru cache dir: %s", paru.ResolveCacheDir(fakeUserCache, ""))
	}
	if paru.ResolveCacheDir(fakeUserCache, customDir) != customDir {
		t.Errorf("custom dir should override default paru cache dir")
	}
	if paru.ResolveBaseCacheDir(fakeUserCache, "") != filepath.Join(fakeUserCache, "paru") {
		t.Errorf("unexpected paru base cache dir: %s", paru.ResolveBaseCacheDir(fakeUserCache, ""))
	}

	yay := YayDriver{}
	if yay.ResolveCacheDir(fakeUserCache, "") != filepath.Join(fakeUserCache, "yay") {
		t.Errorf("unexpected yay cache dir: %s", yay.ResolveCacheDir(fakeUserCache, ""))
	}
	if yay.ResolveCacheDir(fakeUserCache, customDir) != customDir {
		t.Errorf("custom dir should override default yay cache dir")
	}
	if yay.ResolveBaseCacheDir(fakeUserCache, "") != filepath.Join(fakeUserCache, "yay") {
		t.Errorf("unexpected yay base cache dir: %s", yay.ResolveBaseCacheDir(fakeUserCache, ""))
	}
}
