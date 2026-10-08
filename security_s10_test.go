package main

import (
	"fmt"
	"strings"
	"testing"
)

// S-10: the details cache must be bounded to prevent monotonic memory growth.
func TestSecurity_S10_DetailsCacheIsBounded(t *testing.T) {
	m := testModel(t, modeInstall, DefaultConfig())

	// Feed 5000 package details messages through the real ingest handler
	for i := 0; i < 5000; i++ {
		resModel, _ := m.Update(packageDetailsMsg{
			packageName: fmt.Sprintf("pkg-%05d", i),
			details:     strings.Repeat("x", 4096),
		})
		m = resModel.(*model)
	}

	if len(m.detailsCache) > maxDetailsCacheEntries {
		t.Errorf("S-10: detailsCache grew to %d entries (limit %d)", len(m.detailsCache), maxDetailsCacheEntries)
	}

	// Verify that the newest package is in the cache and an old evicted package is not
	if _, ok := m.detailsCache["pkg-04999"]; !ok {
		t.Errorf("expected newest package pkg-04999 to be present in detailsCache")
	}
	if _, ok := m.detailsCache["pkg-00000"]; ok {
		t.Errorf("expected oldest package pkg-00000 to have been evicted from detailsCache")
	}
}

// S-10: direct cachePackageDetails method bounds cache size and respects FIFO eviction.
func TestSecurity_S10_CachePackageDetailsEviction(t *testing.T) {
	m := testModel(t, modeInstall, DefaultConfig())

	for i := 0; i < 600; i++ {
		m.cachePackageDetails(fmt.Sprintf("pkg-%04d", i), fmt.Sprintf("details-%d", i))
	}

	if len(m.detailsCache) != maxDetailsCacheEntries {
		t.Errorf("expected detailsCache to cap at %d, got %d", maxDetailsCacheEntries, len(m.detailsCache))
	}

	// First 100 packages (0 to 99) should be evicted
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("pkg-%04d", i)
		if _, exists := m.detailsCache[key]; exists {
			t.Errorf("expected key %s to be evicted", key)
		}
	}

	// Packages 100 to 599 should be present
	for i := 100; i < 600; i++ {
		key := fmt.Sprintf("pkg-%04d", i)
		if _, exists := m.detailsCache[key]; !exists {
			t.Errorf("expected key %s to be present", key)
		}
	}
}
