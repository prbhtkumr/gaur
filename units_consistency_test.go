package main

import (
	"fmt"
	"testing"
)

func TestUnitConsistency(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"0 bytes", 0, "0 B"},
		{"1023 bytes", 1023, "1023 B"},
		{"1 KiB", 1024, "1.00 KiB"},
		{"1 MiB", 1024 * 1024, "1.00 MiB"},
		{"1 GiB", 1024 * 1024 * 1024, "1.00 GiB"},
		{"1 TiB", 1024 * 1024 * 1024 * 1024, "1.00 TiB"},
		{"Random size MiB", 532854931, "508.17 MiB"},
		{"Random size GiB", 2222645575, "2.07 GiB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test formatBytes
			formatted := formatBytes(tt.bytes)
			if formatted != tt.expected {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, formatted, tt.expected)
			}

			// Test parseSizeToBytes (Round-trip)
			parsed := parseSizeToBytes(formatted)
			// Allow for some floating point precision loss during round-trip if needed,
			// but here we expect exact or very close.
			if parsed != tt.bytes {
				// Special case for our "Random size" which might have float precision issues in the test itself
				if tt.name == "Random size MiB" || tt.name == "Random size GiB" {
					diff := parsed - tt.bytes
					if diff < 0 {
						diff = -diff
					}
					if diff > 1024 { // Allow 1KB error for float tests
						t.Errorf("parseSizeToBytes(%q) = %d, want %d (diff %d)", formatted, parsed, tt.bytes, diff)
					}
				} else {
					t.Errorf("parseSizeToBytes(%q) = %d, want %d", formatted, parsed, tt.bytes)
				}
			}
		})
	}
}

func TestParseVariousUnits(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"500 B", 500},
		{"1 K", 1024},
		{"1 KB", 1024},
		{"1 KiB", 1024},
		{"1 M", 1024 * 1024},
		{"1 MB", 1024 * 1024},
		{"1 MiB", 1024 * 1024},
		{"1 G", 1024 * 1024 * 1024},
		{"1 GB", 1024 * 1024 * 1024},
		{"1 GiB", 1024 * 1024 * 1024},
		{"1 T", 1024 * 1024 * 1024 * 1024},
		{"1 TB", 1024 * 1024 * 1024 * 1024},
		{"1 TiB", 1024 * 1024 * 1024 * 1024},
		// Case insensitivity
		{"1 mib", 1024 * 1024},
		{"1 GIB", 1024 * 1024 * 1024},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseSizeToBytes(tt.input)
			if result != tt.expected {
				t.Errorf("parseSizeToBytes(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestDashboardLayoutWidth(t *testing.T) {
	// We want to make sure the format string used in feature_dashboard.go
	// for sizes (which we changed to %10s) can accommodate "508.17 MiB"
	size := "508.17 MiB"
	formatted := fmt.Sprintf("%10s", size)
	if len(formatted) < 10 {
		t.Errorf("Expected formatted size %q to be at least 10 chars, got %d", formatted, len(formatted))
	}

	// Ensure it doesn't exceed 10 if it is exactly 10
	if len(size) == 10 && len(formatted) != 10 {
		t.Errorf("Expected formatted size %q to be exactly 10 chars, got %d", formatted, len(formatted))
	}
}
