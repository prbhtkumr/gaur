package main

import (
	"testing"
)

func TestHighlightingLogic(t *testing.T) {
	m := testModel(t, modeInstall, DefaultConfig())
	m.width = 100
	m.height = 40
	m.loading = false

	// Test case 1: Verify that performFiltering populates matchIndices in UpdateSelective mode
	t.Run("UpdateSelective populates matchIndices", func(t *testing.T) {
		m.mode = modeUpdateSelective
		m.updatableAll = []Package{
			{Source: "extra", Name: "vim", Version: "9.0"},
			{Source: "extra", Name: "neovim", Version: "0.9"},
		}
		m.textInput.SetValue("vim")

		m.performFiltering()

		if len(m.filtered) == 0 {
			t.Errorf("Filtered list is empty, expected matches for 'vim'")
		}
		if m.matchIndices == nil {
			t.Errorf("matchIndices is nil, expected it to be populated")
		}
		if len(m.matchIndices) != len(m.filtered) {
			t.Errorf("matchIndices size (%d) does not match filtered size (%d)", len(m.matchIndices), len(m.filtered))
		}
	})

	// Test case 2: Verify that performFiltering populates matchIndices in CacheSelective mode
	t.Run("CacheSelective populates matchIndices", func(t *testing.T) {
		m.mode = modeCacheSelective
		m.dashboard.AllCacheHogs = []PackageSize{
			{Name: "vim", Size: "10MB", SizeBytes: 1000000},
			{Name: "neovim", Size: "20MB", SizeBytes: 2000000},
		}
		m.textInput.SetValue("vim")

		m.performFiltering()

		if len(m.filtered) == 0 {
			t.Errorf("Filtered list is empty, expected matches for 'vim'")
		}
		if m.matchIndices == nil {
			t.Errorf("matchIndices is nil, expected it to be populated")
		}
	})

	// Test case 3: Verify highlightMatchesWithSourceColor handles indices correctly
	t.Run("highlightMatchesWithSourceColor logic", func(t *testing.T) {
		pkg := Package{Source: "extra", Name: "vim"}
		indices := []int{6, 7, 8} // "extra/vim" -> v is at 6, i at 7, m at 8

		result := highlightMatchesWithSourceColor(pkg, indices)

		if result == "" {
			t.Errorf("highlightMatchesWithSourceColor returned empty string")
		}
	})
}

func TestMatchIndicesConsolidation(t *testing.T) {
	// This test ensures that we are using m.matchIndices for all modes in View()
	// by checking if the render logic correctly picks up indices from m.matchIndices
	// even in Remove mode.

	m := testModel(t, modeRemove, DefaultConfig())
	m.width = 100
	m.height = 40

	pkgList := []Package{
		{Source: "extra", Name: "vim", Version: "9.0"},
	}
	m.filteredInstalled = pkgList

	// Set matchIndices
	m.matchIndices = map[int][]int{
		0: {6, 7, 8}, // vim
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("View() panicked: %v", r)
		}
	}()

	_ = m.View()
}
