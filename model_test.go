package main

import (
	"testing"
)

func TestInitialModel(t *testing.T) {
	tests := []struct {
		name        string
		mode        viewMode
		placeholder string
		status      string
	}{
		{
			name:        "install mode",
			mode:        modeInstall,
			placeholder: "Search packages...",
			status:      "Loading package database...",
		},
		{
			name:        "remove mode",
			mode:        modeRemove,
			placeholder: "Filter installed packages...",
			status:      "Loading installed packages...",
		},
		{
			name:        "update mode",
			mode:        modeUpdate,
			placeholder: "Checking for updates...",
			status:      "Checking for updates...",
		},
		{
			name:        "installed mode",
			mode:        modeDashboard,
			placeholder: "View system dashboard",
			status:      "Loading system statistics...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := testModel(t, tt.mode, DefaultConfig())
			if m.mode != tt.mode {
				t.Errorf("testModel(t, %v, DefaultConfig()).mode = %v, want %v", tt.mode, m.mode, tt.mode)
			}
			if m.textInput.Placeholder != tt.placeholder {
				t.Errorf("testModel(t, %v, DefaultConfig()).textInput.Placeholder = %q, want %q", tt.mode, m.textInput.Placeholder, tt.placeholder)
			}
			if m.statusMessage != tt.status {
				t.Errorf("testModel(t, %v, DefaultConfig()).statusMessage = %q, want %q", tt.mode, m.statusMessage, tt.status)
			}
			if !m.loading {
				t.Errorf("testModel(t, %v, DefaultConfig()).loading = %v, want true", tt.mode, m.loading)
			}
		})
	}
}

func TestCurrentPackageList(t *testing.T) {
	m := model{
		mode:     modeInstall,
		filtered: []Package{{Name: "pkg1"}, {Name: "pkg2"}},
	}
	list := m.currentPackageList()
	if len(list) != 2 {
		t.Errorf("currentPackageList (install) length = %d, want 2", len(list))
	}

	m.mode = modeRemove
	m.filteredInstalled = []Package{{Name: "pkg3"}}
	list = m.currentPackageList()
	if len(list) != 1 {
		t.Errorf("currentPackageList (remove) length = %d, want 1", len(list))
	}

	m.mode = modeUpdate
	list = m.currentPackageList()
	if list != nil {
		t.Errorf("currentPackageList (update) should be nil, got %v", list)
	}
}

func TestMaxSelectableIndex(t *testing.T) {
	m := model{
		mode:     modeInstall,
		filtered: []Package{{Name: "pkg1"}, {Name: "pkg2"}},
	}
	if m.maxSelectableIndex() != 1 {
		t.Errorf("maxSelectableIndex = %d, want 1", m.maxSelectableIndex())
	}

	m.filtered = []Package{}
	if m.maxSelectableIndex() != 0 {
		t.Errorf("maxSelectableIndex (empty) = %d, want 0", m.maxSelectableIndex())
	}
}

func TestSelectedPackage(t *testing.T) {
	pkg1 := Package{Name: "pkg1"}
	pkg2 := Package{Name: "pkg2"}
	m := model{
		mode:          modeInstall,
		filtered:      []Package{pkg1, pkg2},
		selectedIndex: 1,
	}
	selected := m.selectedPackage()
	if selected == nil || selected.Name != "pkg2" {
		t.Errorf("selectedPackage = %v, want pkg2", selected)
	}

	m.selectedIndex = 5
	selected = m.selectedPackage()
	if selected != nil {
		t.Errorf("selectedPackage (out of bounds) should be nil, got %v", selected)
	}

	// Test modeUpdateSelective
	m.mode = modeUpdateSelective
	m.selectedIndex = 0
	selected = m.selectedPackage()
	if selected == nil || selected.Name != "pkg1" {
		t.Errorf("selectedPackage (updateSelective) = %v, want pkg1", selected)
	}

	// Test modeCacheSelective
	m.mode = modeCacheSelective
	m.selectedIndex = 1
	selected = m.selectedPackage()
	if selected == nil || selected.Name != "pkg2" {
		t.Errorf("selectedPackage (cacheSelective) = %v, want pkg2", selected)
	}
}

func TestEnsureDetailsLoaded(t *testing.T) {
	pkg := &Package{Name: "git"}
	m := model{
		mode:         modeInstall,
		detailsCache: map[string]string{"git": "Fast, scalable, distributed VCS"},
	}

	// 1. Cached hit
	cmd := m.ensureDetailsLoaded(pkg)
	if cmd != nil {
		t.Errorf("Expected nil command for cached details, got %v", cmd)
	}
	if m.packageDetails != "Fast, scalable, distributed VCS" {
		t.Errorf("Expected cached details to be assigned, got %q", m.packageDetails)
	}
	if m.loadingDetails {
		t.Errorf("Expected loadingDetails to be false on cache hit")
	}

	// 2. Cache miss triggers debounce command
	pkgMiss := &Package{Name: "neovim"}
	cmd = m.ensureDetailsLoaded(pkgMiss)
	if cmd == nil {
		t.Errorf("Expected debounce command for uncached package, got nil")
	}
	if !m.loadingDetails {
		t.Errorf("Expected loadingDetails to be true on cache miss")
	}
	if m.pendingDetailsPackage != "neovim" {
		t.Errorf("Expected pendingDetailsPackage to be neovim, got %q", m.pendingDetailsPackage)
	}

	// 3. Nil package clears details
	cmd = m.ensureDetailsLoaded(nil)
	if cmd != nil {
		t.Errorf("Expected nil command for nil package, got %v", cmd)
	}
	if m.loadingDetails || m.packageDetails != "" {
		t.Errorf("Expected details to be cleared for nil package")
	}
}

func TestResolvePackages(t *testing.T) {
	m := model{
		filtered: []Package{
			{Name: "ripgrep", Version: "14.1.0", Source: "extra"},
		},
	}
	resolved := m.resolvePackages([]string{"ripgrep", "missing-pkg"})
	if len(resolved) != 2 {
		t.Fatalf("Expected 2 resolved packages, got %d", len(resolved))
	}
	if resolved[0].Name != "ripgrep" || resolved[0].Version != "14.1.0" {
		t.Errorf("Expected full package info for ripgrep, got %+v", resolved[0])
	}
	if resolved[1].Name != "missing-pkg" || resolved[1].Version != "" {
		t.Errorf("Expected fallback package for missing-pkg, got %+v", resolved[1])
	}
}
