package main

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// 1. Core Cache Reads during Navigation

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

func TestDetailsCache_MissSchedulesDebounce(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)
	m.filtered = []Package{
		{Name: "pkg-a"},
		{Name: "pkg-b"},
	}
	m.selectedIndex = 0
	// pkg-b is NOT in detailsCache

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(*model)

	if m.selectedIndex != 1 {
		t.Fatalf("expected selectedIndex 1, got %d", m.selectedIndex)
	}
	if cmd == nil {
		t.Errorf("expected debounce command on cache miss, got nil")
	}
	if !m.loadingDetails {
		t.Errorf("expected loadingDetails to be true on cache miss")
	}
	if m.pendingDetailsPackage != "pkg-b" {
		t.Errorf("expected pendingDetailsPackage to be %q, got %q", "pkg-b", m.pendingDetailsPackage)
	}
}

func TestDetailsCache_NoNavigationNoop(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)
	m.filtered = []Package{
		{Name: "pkg-a"},
	}
	m.selectedIndex = 0
	m.packageDetails = "existing details"
	m.detailsForPackage = "pkg-a"
	m.pendingDetailsPackage = ""

	// Attempting to move beyond boundary in single-item list
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(*model)

	if m.selectedIndex != 0 {
		t.Fatalf("expected selectedIndex to stay 0, got %d", m.selectedIndex)
	}
	if cmd != nil {
		t.Errorf("expected nil cmd when index did not change, got %v", cmd)
	}
	if m.packageDetails != "existing details" {
		t.Errorf("expected packageDetails to remain unchanged, got %q", m.packageDetails)
	}
}

func TestDetailsCache_ModeCacheSelectiveBypassesCache(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeCacheSelective, cfg)
	m.dashboard.AllCacheHogs = []PackageSize{
		{Name: "pkg-a", Size: "10 MB", SizeBytes: 10485760},
		{Name: "pkg-b", Size: "20 MB", SizeBytes: 20971520},
	}
	m.filtered = []Package{
		{Name: "pkg-a"},
		{Name: "pkg-b"},
	}
	m.selectedIndex = 0
	m.detailsCache["pkg-b"] = "details b"

	// Standard navigation in cache selective mode (down moves to index 1)
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(*model)

	if m.selectedIndex != 1 {
		t.Fatalf("expected selectedIndex 1, got %d", m.selectedIndex)
	}
	if cmd != nil {
		t.Errorf("expected nil cmd in modeCacheSelective, got %v", cmd)
	}
	if m.loadingDetails {
		t.Errorf("expected loadingDetails to be false in modeCacheSelective")
	}
	if m.packageDetails != "" {
		t.Errorf("expected packageDetails to remain empty in modeCacheSelective, got %q", m.packageDetails)
	}
}

func TestDetailsCache_NavigationInRemoveMode(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeRemove, cfg)
	m.installed = []Package{
		{Name: "installed-a"},
		{Name: "installed-b"},
	}
	m.filteredInstalled = m.installed
	m.selectedIndex = 0
	m.detailsCache["installed-b"] = "cached details for installed-b"

	// Visual up in modeRemove (bottom-up menu) increments selectedIndex to 1
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(*model)

	if m.selectedIndex != 1 {
		t.Fatalf("expected selectedIndex 1, got %d", m.selectedIndex)
	}
	if cmd != nil {
		t.Errorf("expected cmd to be nil on cache hit in remove mode, got %v", cmd)
	}
	if m.packageDetails != "cached details for installed-b" {
		t.Errorf("expected packageDetails to be %q, got %q", "cached details for installed-b", m.packageDetails)
	}
	if m.detailsForPackage != "installed-b" {
		t.Errorf("expected detailsForPackage to be installed-b, got %q", m.detailsForPackage)
	}
}

func TestDetailsCache_NavigationInUpdateSelectiveMode(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeUpdateSelective, cfg)
	m.updatableAll = []Package{
		{Name: "updatable-a"},
		{Name: "updatable-b"},
	}
	m.filtered = m.updatableAll
	m.selectedIndex = 0
	m.detailsCache["updatable-b"] = "cached details for updatable-b"

	// Visual up in modeUpdateSelective (bottom-up menu) increments selectedIndex to 1
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(*model)

	if m.selectedIndex != 1 {
		t.Fatalf("expected selectedIndex 1, got %d", m.selectedIndex)
	}
	if cmd != nil {
		t.Errorf("expected cmd to be nil on cache hit in selective update mode, got %v", cmd)
	}
	if m.packageDetails != "cached details for updatable-b" {
		t.Errorf("expected packageDetails to be %q, got %q", "cached details for updatable-b", m.packageDetails)
	}
}

func TestDetailsCache_MultipleLookupsConsistent(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)
	m.filtered = []Package{
		{Name: "pkg-a"},
		{Name: "pkg-b"},
	}
	m.selectedIndex = 0
	m.detailsCache["pkg-a"] = "details a"
	m.detailsCache["pkg-b"] = "details b"

	// Move to pkg-b (index 1)
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(*model)
	if cmd != nil || m.packageDetails != "details b" || m.detailsForPackage != "pkg-b" {
		t.Fatalf("first lookup failed: details=%q, cmd=%v", m.packageDetails, cmd)
	}

	// Move back to pkg-a (index 0)
	newModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(*model)
	if cmd != nil || m.packageDetails != "details a" || m.detailsForPackage != "pkg-a" {
		t.Fatalf("second lookup failed: details=%q, cmd=%v", m.packageDetails, cmd)
	}

	// Move to pkg-b again
	newModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(*model)
	if cmd != nil || m.packageDetails != "details b" || m.detailsForPackage != "pkg-b" {
		t.Fatalf("third lookup failed: details=%q, cmd=%v", m.packageDetails, cmd)
	}
}

// 2. Filter Pipeline Cache Reads

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

func TestDetailsCache_FilterMissSchedulesDebounce(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeRemove, cfg)
	m.installed = []Package{
		{Name: "git", Source: "extra"},
	}
	m.filteredInstalled = m.installed
	// "git" is not in detailsCache

	cmd := m.performFiltering()
	if cmd == nil {
		t.Errorf("expected debounce command on filter cache miss, got nil")
	}
	if !m.loadingDetails {
		t.Errorf("expected loadingDetails to be true on filter cache miss")
	}
	if m.pendingDetailsPackage != "git" {
		t.Errorf("expected pendingDetailsPackage to be git, got %q", m.pendingDetailsPackage)
	}
}

func TestDetailsCache_FilterEmptyListClearsDetails(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeRemove, cfg)
	m.installed = []Package{}
	m.filteredInstalled = m.installed
	m.packageDetails = "stale details"
	m.detailsForPackage = "stale-pkg"
	m.loadingDetails = true

	cmd := m.performFiltering()
	if cmd != nil {
		t.Errorf("expected cmd to be nil when filtered list is empty, got %v", cmd)
	}
	if m.loadingDetails {
		t.Errorf("expected loadingDetails to be false when list is empty")
	}
	if m.packageDetails != "" {
		t.Errorf("expected packageDetails to be cleared, got %q", m.packageDetails)
	}
	if m.detailsForPackage != "" {
		t.Errorf("expected detailsForPackage to be cleared, got %q", m.detailsForPackage)
	}
}

// 3. Stale Response & Concurrency Guard Invariants

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

func TestDetailsCache_StaleStompPreventionSequence(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)
	m.filtered = []Package{
		{Name: "pkg-a"},
		{Name: "pkg-b"},
	}
	m.selectedIndex = 0
	m.detailsCache["pkg-b"] = "cached details for pkg-b"

	// Step 1: Nav onto pkg-a (miss) schedules debounce
	m.loadingDetails = true
	m.pendingDetailsPackage = "pkg-a"
	m.detailsForPackage = ""

	// Step 2: User quickly navigates to pkg-b (hit) before debounce tick fires
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = newModel.(*model)

	if cmd != nil {
		t.Fatalf("expected nil cmd on cache hit, got %v", cmd)
	}
	if m.packageDetails != "cached details for pkg-b" {
		t.Fatalf("expected packageDetails %q, got %q", "cached details for pkg-b", m.packageDetails)
	}
	if m.detailsForPackage != "pkg-b" {
		t.Fatalf("expected detailsForPackage 'pkg-b', got %q", m.detailsForPackage)
	}
	if m.pendingDetailsPackage != "" {
		t.Fatalf("expected pendingDetailsPackage cleared, got %q", m.pendingDetailsPackage)
	}

	// Step 3: Delayed debounce tick for pkg-a arrives
	newModel, tickCmd := m.Update(debounceTickMsg{packageName: "pkg-a"})
	m = newModel.(*model)
	if tickCmd != nil {
		t.Errorf("delayed tick should not spawn command, got %v", tickCmd)
	}
	if m.detailsForPackage != "pkg-b" {
		t.Errorf("detailsForPackage changed to %q after stale tick", m.detailsForPackage)
	}

	// Step 4: Delayed fetch response for pkg-a arrives
	newModel, _ = m.Update(packageDetailsMsg{
		packageName: "pkg-a",
		details:     "fetched details for pkg-a",
		err:         nil,
	})
	m = newModel.(*model)

	// Display on screen remains pkg-b
	if m.packageDetails != "cached details for pkg-b" {
		t.Errorf("active details stomped by late response: got %q", m.packageDetails)
	}
	// pkg-a is now stored in cache for future visits
	if m.detailsCache["pkg-a"] != "fetched details for pkg-a" {
		t.Errorf("pkg-a details were not cached in background: got %q", m.detailsCache["pkg-a"])
	}

	// Step 5: User navigates back to pkg-a (now hits cache!)
	newModel, backCmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = newModel.(*model)
	if backCmd != nil {
		t.Errorf("expected nil cmd for newly cached pkg-a, got %v", backCmd)
	}
	if m.packageDetails != "fetched details for pkg-a" {
		t.Errorf("expected packageDetails %q, got %q", "fetched details for pkg-a", m.packageDetails)
	}
}

// 4. Invalidation & State Lifecycles

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

func TestDetailsCache_InvalidatedOnActionComplete(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)
	m.detailsCache["pkg-a"] = "details a"

	// Failure does NOT invalidate
	newModel, _ := m.Update(actionCompleteMsg{
		message: "Failed",
		err:     errors.New("action failed"),
	})
	m = newModel.(*model)
	if len(m.detailsCache) == 0 {
		t.Errorf("detailsCache should not be cleared on action failure")
	}

	// Success DOES invalidate
	newModel, cmd := m.Update(actionCompleteMsg{
		message: "Cleaned cache",
		err:     nil,
	})
	m = newModel.(*model)
	if cmd == nil {
		t.Errorf("expected refresh command on action success")
	}
	if len(m.detailsCache) != 0 {
		t.Errorf("expected detailsCache to be cleared on action success, got %d entries", len(m.detailsCache))
	}
}

func TestDetailsCache_InvalidatedOnSyncRepositories(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)
	m.detailsCache["pkg-a"] = "details a"

	// Failure does NOT invalidate
	newModel, _ := m.Update(syncRepositoriesMsg{
		err: errors.New("sync failed"),
	})
	m = newModel.(*model)
	if len(m.detailsCache) == 0 {
		t.Errorf("detailsCache should not be cleared on sync failure")
	}

	// Success DOES invalidate
	newModel, cmd := m.Update(syncRepositoriesMsg{
		err: nil,
	})
	m = newModel.(*model)
	if cmd == nil {
		t.Errorf("expected refresh command on sync success")
	}
	if len(m.detailsCache) != 0 {
		t.Errorf("expected detailsCache to be cleared on sync success, got %d entries", len(m.detailsCache))
	}
}

func TestDetailsCache_InvalidatedOnExecComplete(t *testing.T) {
	cfg := DefaultConfig()
	m := testModel(t, modeInstall, cfg)
	m.detailsCache["pkg-a"] = "details a"

	// Failure does NOT invalidate
	newModel, _ := m.Update(execCompleteMsg{
		operation: confirmInstall,
		packages:  []string{"pkg-a"},
		err:       errors.New("terminal install failed"),
	})
	m = newModel.(*model)
	if len(m.detailsCache) == 0 {
		t.Errorf("detailsCache should not be cleared on exec failure")
	}

	// Success DOES invalidate
	newModel, cmd := m.Update(execCompleteMsg{
		operation: confirmInstall,
		packages:  []string{"pkg-a"},
		err:       nil,
	})
	m = newModel.(*model)
	if cmd == nil {
		t.Errorf("expected refresh command on exec success")
	}
	if len(m.detailsCache) != 0 {
		t.Errorf("expected detailsCache to be cleared on exec success, got %d entries", len(m.detailsCache))
	}
}

func TestDetailsCache_InvalidatedOnHelperSettingChange(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Commands.AurHelper = "paru"
	m := testModel(t, modeSettings, cfg)
	m.previousMode = modeInstall
	m.originalHelper = "paru"
	m.detailsCache["pkg-a"] = "details a"

	// Exiting settings without changing helper preserves cache
	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(*model)
	if cmd != nil {
		t.Errorf("expected nil cmd when helper unchanged, got %v", cmd)
	}
	if len(m.detailsCache) == 0 {
		t.Errorf("detailsCache should not be cleared when helper unchanged")
	}

	// Now change helper and exit settings
	m.mode = modeSettings
	m.config.Commands.AurHelper = "yay"
	newModel, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = newModel.(*model)
	if cmd == nil {
		t.Errorf("expected refresh command when helper changed")
	}
	if len(m.detailsCache) != 0 {
		t.Errorf("expected detailsCache to be cleared when helper changed, got %d entries", len(m.detailsCache))
	}
}
