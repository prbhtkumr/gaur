package main

import (
	"github.com/charmbracelet/lipgloss"
	"strings"
	"testing"
)

func TestAllConfirmationMenus(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name        string
		confirmType confirmationType
		setup       func(m *model)
		expected    []string
	}{
		{
			name:        "Install Confirmation",
			confirmType: confirmInstall,
			setup: func(m *model) {
				m.repoPackages = []Package{{Name: "pkg1", Version: "1.0"}}
				m.confirmPackages = []string{"pkg1"}
			},
			expected: []string{"Confirm Installation", "1 packages will be installed", "pkg1"},
		},
		{
			name:        "Remove Confirmation",
			confirmType: confirmRemove,
			setup: func(m *model) {
				m.installed = []Package{{Name: "pkg-to-remove", Version: "2.0"}}
				m.confirmPackages = []string{"pkg-to-remove"}
			},
			expected: []string{"Confirm Removal", "1 packages will be removed", "pkg-to-remove"},
		},
		{
			name:        "Update Confirmation",
			confirmType: confirmUpdate,
			setup: func(m *model) {
				m.pendingUpdates = []Package{{Name: "update1", Version: "3.0", Source: "core"}}
				m.confirmPackages = []string{"update1"}
			},
			expected: []string{"Confirm System Update", "1 updates are available", "update1"},
		},
		{
			name:        "Selective Update Confirmation",
			confirmType: confirmSelectiveUpdate,
			setup: func(m *model) {
				m.pendingUpdates = []Package{{Name: "sel-update", Version: "4.0", Source: "extra"}}
				m.confirmPackages = []string{"sel-update"}
			},
			expected: []string{"Confirm Selective Update", "1 packages will be updated", "sel-update"},
		},
		{
			name:        "Orphan Removal Confirmation",
			confirmType: confirmRemoveOrphans,
			setup: func(m *model) {
				m.confirmPackages = []string{"orphan1", "orphan2"}
			},
			expected: []string{"Confirm Orphan Removal", "2 packages will be removed", "orphan1", "orphan2"},
		},
		{
			name:        "Cache Selective Confirmation",
			confirmType: confirmCleanSelective,
			setup: func(m *model) {
				m.confirmPackages = []string{"cache-pkg"}
			},
			expected: []string{"Confirm Selective Clean", "1 packages will be removed from cache", "cache-pkg"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := testModel(t, modeInstall, cfg)
			m.confirmType = tt.confirmType
			m.showConfirmation = true
			if tt.setup != nil {
				tt.setup(m)
			}

			view := m.renderConfirmationDialog(80, 24, lipgloss.Color("7"))

			for _, exp := range tt.expected {
				if !strings.Contains(stripAnsi(view), exp) {
					t.Errorf("Expected view to contain %q, but it didn't.\nView:\n%s", exp, view)
				}
			}
		})
	}
}
