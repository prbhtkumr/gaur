package main

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDetailsCache_HitSkipsDebounce(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)
	m.filtered = []Package{
		{Name: "pkg-a"},
		{Name: "pkg-b"},
	}
	m.selectedIndex = 0
	m.detailsCache["pkg-b"] = "cached details for pkg-b"

	// Visual up in modeInstall (bottom-up menu) increments selectedIndex to 1 (pkg-b)
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(*model)

	if m.selectedIndex != 1 {
		t.Fatalf("expected selectedIndex 1, got %d", m.selectedIndex)
	}
	if cmd != nil {
		t.Errorf("expected cmd to be nil on cache hit, got %v", cmd)
	}
	if m.loadingDetails {
		t.Errorf("expected loadingDetails to be false, got true")
	}
	if m.packageDetails != "cached details for pkg-b" {
		t.Errorf("expected packageDetails to be %q, got %q", "cached details for pkg-b", m.packageDetails)
	}
}

func TestDetailsCache_HitMovesGuards(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)
	m.filtered = []Package{
		{Name: "pkg-a"},
		{Name: "pkg-b"},
	}
	m.selectedIndex = 0
	m.detailsForPackage = "pkg-old"
	m.pendingDetailsPackage = "pkg-old"
	m.detailsCache["pkg-b"] = "cached details for pkg-b"

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(*model)

	if m.detailsForPackage != "pkg-b" {
		t.Errorf("expected detailsForPackage to be %q, got %q", "pkg-b", m.detailsForPackage)
	}
	if m.pendingDetailsPackage != "" {
		t.Errorf("expected pendingDetailsPackage to be empty, got %q", m.pendingDetailsPackage)
	}
}

func TestDetailsCache_StaleResponseDropped(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)
	m.detailsForPackage = "pkg-b"
	m.packageDetails = "details for pkg-b"

	// A late response arrives for pkg-a
	newModel, _ := m.Update(packageDetailsMsg{
		packageName: "pkg-a",
		details:     "late details for pkg-a",
		err:         nil,
	})
	m = newModel.(*model)

	if m.packageDetails != "details for pkg-b" {
		t.Errorf("expected packageDetails to remain %q, got %q", "details for pkg-b", m.packageDetails)
	}
}

func TestDetailsCache_StaleTickIgnored(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)
	m.pendingDetailsPackage = ""
	m.detailsForPackage = "pkg-b"
	m.repoPackages = []Package{{Name: "pkg-a"}}

	// A stale debounce tick for pkg-a arrives when pendingDetailsPackage is ""
	newModel, cmd := m.Update(debounceTickMsg{
		packageName: "pkg-a",
	})
	m = newModel.(*model)

	if cmd != nil {
		t.Errorf("expected cmd to be nil for stale tick, got %v", cmd)
	}
	if m.detailsForPackage != "pkg-b" {
		t.Errorf("expected detailsForPackage to remain %q, got %q", "pkg-b", m.detailsForPackage)
	}
}

func TestDetailsCache_ErrorNotCached(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)
	m.detailsForPackage = "pkg-a"

	newModel, _ := m.Update(packageDetailsMsg{
		packageName: "pkg-a",
		details:     "Failed to get package details",
		err:         errors.New("subprocess failed"),
	})
	m = newModel.(*model)

	if _, exists := m.detailsCache["pkg-a"]; exists {
		t.Errorf("expected pkg-a not to be in detailsCache on error")
	}
	if m.packageDetails != "Failed to get package details" {
		t.Errorf("expected packageDetails to display error text, got %q", m.packageDetails)
	}
	if m.loadingDetails {
		t.Errorf("expected loadingDetails to be false, got true")
	}
}

func TestDetailsCache_ReflectedNotDisplayed(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)
	m.detailsForPackage = "pkg-b"
	m.packageDetails = "details for pkg-b"

	// Response for pkg-a arrives while user is on pkg-b
	newModel, _ := m.Update(packageDetailsMsg{
		packageName: "pkg-a",
		details:     "details for pkg-a",
		err:         nil,
	})
	m = newModel.(*model)

	// Cache was populated
	if cached, ok := m.detailsCache["pkg-a"]; !ok || cached != "details for pkg-a" {
		t.Errorf("expected detailsCache to store pkg-a, got %q (ok=%v)", cached, ok)
	}
	// But screen was not overwritten
	if m.packageDetails != "details for pkg-b" {
		t.Errorf("expected packageDetails to remain %q, got %q", "details for pkg-b", m.packageDetails)
	}
}

func TestDetailsCache_InvalidatedOnRefresh(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)
	m.detailsCache["pkg-a"] = "details a"
	m.detailsCache["pkg-b"] = "details b"

	m.refreshAll()

	if len(m.detailsCache) != 0 {
		t.Errorf("expected detailsCache to be empty after refreshAll(), got %d entries", len(m.detailsCache))
	}
}

func TestDetailsCache_SurvivesResetState(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)
	m.detailsCache["pkg-a"] = "details a"

	m.resetState()

	if cached, ok := m.detailsCache["pkg-a"]; !ok || cached != "details a" {
		t.Errorf("expected detailsCache to survive resetState(), got %q (ok=%v)", cached, ok)
	}
}

func TestDetailsCache_FilterHitSkipsDebounce(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeRemove, cfg)
	m.installed = []Package{
		{Name: "git", Source: "extra"},
	}
	m.filteredInstalled = m.installed
	m.detailsCache["git"] = "git package details"

	cmd := m.performFiltering()
	if cmd != nil {
		t.Errorf("expected cmd to be nil on filter cache hit in remove mode, got %v", cmd)
	}
	if m.packageDetails != "git package details" {
		t.Errorf("expected packageDetails to be %q, got %q", "git package details", m.packageDetails)
	}
	if m.loadingDetails {
		t.Errorf("expected loadingDetails to be false")
	}
	if m.detailsForPackage != "git" {
		t.Errorf("expected detailsForPackage to be git, got %q", m.detailsForPackage)
	}
	if m.pendingDetailsPackage != "" {
		t.Errorf("expected pendingDetailsPackage to be empty, got %q", m.pendingDetailsPackage)
	}
}
