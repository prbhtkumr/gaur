package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestRenderScrollbar(t *testing.T) {
	color := lipgloss.Color("12")

	tests := []struct {
		name          string
		total         int
		offset        int
		visibleHeight int
		reversed      bool
		shouldBeEmpty bool
		thumbAtTop    bool
		thumbAtBottom bool
	}{
		{
			name:          "No scrollbar needed",
			total:         10,
			offset:        0,
			visibleHeight: 10,
			shouldBeEmpty: true,
		},
		{
			name:          "Top position (Standard)",
			total:         20,
			offset:        0,
			visibleHeight: 10,
			reversed:      false,
			thumbAtTop:    true,
		},
		{
			name:          "Bottom position (Standard)",
			total:         20,
			offset:        10,
			visibleHeight: 10,
			reversed:      false,
			thumbAtBottom: true,
		},
		{
			name:          "Top position (Reversed)",
			total:         20,
			offset:        0,
			visibleHeight: 10,
			reversed:      true,
			thumbAtBottom: true, // Offset 0 in reversed means bottom of visual list
		},
		{
			name:          "Bottom position (Reversed)",
			total:         20,
			offset:        10,
			visibleHeight: 10,
			reversed:      true,
			thumbAtTop:    true, // Max offset in reversed means top of visual list
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := renderScrollbar(tt.total, tt.offset, tt.visibleHeight, color, tt.reversed)

			if tt.shouldBeEmpty {
				if res != "" {
					t.Errorf("Expected empty string, got %q", res)
				}
				return
			}

			if res == "" {
				t.Error("Expected scrollbar string, got empty")
				return
			}

			lines := strings.Split(res, "\n")
			if len(lines) != tt.visibleHeight {
				t.Errorf("Expected %d lines, got %d", tt.visibleHeight, len(lines))
			}

			// Check for thumb presence
			hasThumb := false
			for _, line := range lines {
				if strings.Contains(line, "┃") {
					hasThumb = true
					break
				}
			}
			if !hasThumb {
				t.Error("Scrollbar does not contain thumb character '┃'")
			}

			if tt.thumbAtTop {
				if !strings.Contains(lines[0], "┃") {
					t.Error("Expected thumb at top line")
				}
			}

			if tt.thumbAtBottom {
				if !strings.Contains(lines[len(lines)-1], "┃") {
					t.Error("Expected thumb at bottom line")
				}
			}
		})
	}
}

func TestScrollbarEdgeCases(t *testing.T) {
	color := lipgloss.Color("12")

	// 1. Visible height 0 or negative
	if renderScrollbar(10, 0, 0, color, false) != "" {
		t.Error("Expected empty string for visibleHeight 0")
	}

	// 2. Total less than visible
	if renderScrollbar(5, 0, 10, color, false) != "" {
		t.Error("Expected empty string when total < visibleHeight")
	}

	// 3. One item list
	if renderScrollbar(1, 0, 1, color, false) != "" {
		t.Error("Expected empty string for 1 item list")
	}
}
