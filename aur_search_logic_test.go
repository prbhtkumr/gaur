package main

import (
	"strings"
	"testing"
	"time"
)

func TestAurSearchTriggeringLogic(t *testing.T) {
	config := DefaultConfig()
	m := testModel(t, modeInstall, config)
	m.width = 100
	m.height = 40
	m.repoPackages = []Package{
		{Source: "core", Name: "linux"},
		{Source: "extra", Name: "vim"},
	}

	tests := []struct {
		name               string
		query              string
		shouldTrigger      bool
		shouldClearAur     bool
		expectedStatusPart string
	}{
		{
			name:               "Very short query (no search)",
			query:              "g", // Length 1 < minSearchQueryLen (2)
			shouldTrigger:      false,
			shouldClearAur:     true,
			expectedStatusPart: "",
		},
		{
			name:               "Minimum length query (trigger search)",
			query:              "go", // Length 2 == minSearchQueryLen (2)
			shouldTrigger:      true,
			shouldClearAur:     false,
			expectedStatusPart: "Searching AUR",
		},
		{
			name:               "Long query (trigger search)",
			query:              "google",
			shouldTrigger:      true,
			shouldClearAur:     false,
			expectedStatusPart: "Searching AUR",
		},
		{
			name:               "Repo filter (no AUR search)",
			query:              "c:linux",
			shouldTrigger:      false,
			shouldClearAur:     true,
			expectedStatusPart: "",
		},
		{
			name:               "AUR filter (trigger search)",
			query:              "a:chrome",
			shouldTrigger:      true,
			shouldClearAur:     false,
			expectedStatusPart: "Searching AUR",
		},
		{
			name:               "Combined filter with AUR (trigger search)",
			query:              "ae:chrome",
			shouldTrigger:      true,
			shouldClearAur:     false,
			expectedStatusPart: "Searching AUR",
		},
		{
			name:               "Combined filter without AUR (no search)",
			query:              "cem:linux",
			shouldTrigger:      false,
			shouldClearAur:     true,
			expectedStatusPart: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// RESET STATE for each test case
			m.aurPackages = []Package{{Source: "aur", Name: "cached-pkg"}}
			m.searchingAUR = false
			m.lastAURQuery = "some-other-query" // Ensure it's different so it triggers
			m.searchStatus = ""

			m.textInput.SetValue(tt.query)
			_ = m.performFiltering()

			// In our logic, performFiltering sets m.searchingAUR to true if it's triggering a search
			triggered := m.searchingAUR

			if triggered != tt.shouldTrigger {
				t.Errorf("%s: expected triggered=%v, got %v for query %q", tt.name, tt.shouldTrigger, triggered, tt.query)
			}

			if tt.shouldClearAur && m.aurPackages != nil {
				t.Errorf("%s: expected aurPackages to be cleared for query %q", tt.name, tt.query)
			}

			if tt.expectedStatusPart != "" {
				if !strings.Contains(m.searchStatus, tt.expectedStatusPart) {
					t.Errorf("%s: expected status to contain %q, got %q", tt.name, tt.expectedStatusPart, m.searchStatus)
				}
			} else {
				if m.searchStatus != "" && tt.query != "" {
					t.Errorf("%s: expected status to be empty, got %q", tt.name, m.searchStatus)
				}
			}
		})
	}
}

func TestAurSearchResponseHandling(t *testing.T) {
	m := testModel(t, modeInstall, DefaultConfig())
	m.textInput.SetValue("a:test")
	m.searchingAUR = true

	// Mock successful search response
	msg := aurSearchMsg{
		packages:  []Package{{Name: "test-pkg", Source: "aur"}},
		query:     "test",
		timeTaken: 1 * time.Second,
	}

	// Case 1: Filter still matches query
	m.Update(msg)
	if len(m.aurPackages) != 1 || m.aurPackages[0].Name != "test-pkg" {
		t.Errorf("aurPackages should have been updated")
	}
	if m.searchStatus == "" {
		t.Errorf("searchStatus should have been updated")
	}

	// Case 2: Filter changed to exclude AUR while search was in flight
	m.textInput.SetValue("c:test") // Repo filter without AUR
	m.aurPackages = nil
	m.searchStatus = ""
	msg.query = "test" // same query

	m.Update(msg)
	if m.aurPackages != nil {
		t.Errorf("aurPackages should NOT have been updated because repo filter excludes AUR")
	}
	if m.searchStatus != "" {
		t.Errorf("searchStatus should NOT have been updated")
	}
}

func TestAurSearchRaceCondition(t *testing.T) {
	m := testModel(t, modeInstall, DefaultConfig())

	// User types "hel", search 1 triggers
	m.textInput.SetValue("hel")
	m.performFiltering()
	if !m.searchingAUR || m.lastAURQuery != "hel" {
		t.Fatalf("Search 1 for 'hel' should have triggered")
	}

	// User quickly types "hello", search 2 is NOT triggered yet because searchingAUR is true
	m.textInput.SetValue("hello")
	m.performFiltering()
	if m.lastAURQuery != "hel" {
		t.Errorf("lastAURQuery should still be 'hel' until a new search is actually sent")
	}
	if !strings.Contains(m.searchStatus, "hello") {
		t.Errorf("Status should mention 'hello', got: %q", m.searchStatus)
	}

	// Now Search 1 for "hel" returns
	msg := aurSearchMsg{
		packages:  []Package{{Name: "hel-pkg", Source: "aur"}},
		query:     "hel",
		timeTaken: 100 * time.Millisecond,
	}

	m.Update(msg)

	// THE FIX: m.searchingAUR should be true NOW because it should have triggered a new search for "hello"
	if !m.searchingAUR {
		t.Errorf("searchingAUR should be true after update because a pending query was waiting")
	}
	if m.lastAURQuery != "hello" {
		t.Errorf("lastAURQuery should now be 'hello', got %q", m.lastAURQuery)
	}
	if !strings.Contains(m.searchStatus, "hello") {
		t.Errorf("Status should now mention 'hello', got: %q", m.searchStatus)
	}
}
