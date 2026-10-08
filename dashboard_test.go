package main

import (
	"testing"
)

func TestParseParuStats(t *testing.T) {
	sampleOutput := `
Total Installed Size : 15.4 GiB
Missing AUR Packages : 2
Top 10 biggest packages:
========================
linux: 450.0 MiB
firefox: 250.0 MiB

`
	totalSize, totalSizeBytes, missingAUR, topPackages := parseParuStats(sampleOutput)

	if totalSize != "15.40 GiB" {
		t.Errorf("totalSize = %q, want 15.40 GiB", totalSize)
	}

	if totalSizeBytes <= 0 {
		t.Errorf("totalSizeBytes = %d, want > 0", totalSizeBytes)
	}
	// 15.4 GiB is 16,535,614,259 bytes approximately
	if totalSizeBytes < 15*1024*1024*1024 || totalSizeBytes > 16*1024*1024*1024 {
		t.Errorf("totalSizeBytes %d is outside expected range for 15.40 GiB", totalSizeBytes)
	}

	if missingAUR != 2 {
		t.Errorf("missingAUR = %d, want 2", missingAUR)
	}

	if len(topPackages) != 2 {
		t.Errorf("len(topPackages) = %d, want 2", len(topPackages))
	}

	if len(topPackages) > 0 && topPackages[0].Name != "linux" {
		t.Errorf("topPackages[0].Name = %q, want linux", topPackages[0].Name)
	}
	if len(topPackages) > 0 && topPackages[0].Size != "450.00 MiB" {
		t.Errorf("topPackages[0].Size = %q, want 450.00 MiB", topPackages[0].Size)
	}
}

func TestParseParuStats_WithForeignMessage(t *testing.T) {
	sampleOutput := `
Total Installed Size : 15.4 GiB
Missing AUR Packages : 2
Top 10 biggest packages:
========================
linux: 450.0 MiB
firefox: 250.0 MiB
: packages not in the AUR: paru-debug
`
	_, _, _, topPackages := parseParuStats(sampleOutput)

	if len(topPackages) != 2 {
		t.Errorf("len(topPackages) = %d, want 2 (should skip the foreign message line)", len(topPackages))
	}

	for _, pkg := range topPackages {
		if pkg.Name == "" {
			t.Errorf("found package with empty name in topPackages: %+v", pkg)
		}
	}
}
