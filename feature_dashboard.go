package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func getDashboardData(c *Config, r ...CommandRunner) tea.Cmd {
	return getDashboardDataWithContext(context.Background(), c, r...)
}

func getDashboardDataWithContext(ctx context.Context, c *Config, r ...CommandRunner) tea.Cmd {
	return func() tea.Msg {
		activeRunner := getActiveRunner(r...)
		var data DashboardData
		var errs []error
		var dataMu sync.Mutex

		addErr := func(msg string, err error) {
			dataMu.Lock()
			defer dataMu.Unlock()
			errs = append(errs, fmt.Errorf("%s: %w", msg, err))
		}

		var wg sync.WaitGroup

		var installedNamesMu sync.Mutex
		installedNames := make(map[string]bool)

		// Total Packages
		wg.Add(1)
		go func() {
			defer wg.Done()
			args := BuildAURCommand(c, "query-all", "-Qq") // Using a generic query
			out, err := activeRunner.RunContext(ctx, args[0], args[1:]...)
			if err == nil {
				lines := strings.Split(strings.TrimSpace(string(out)), "\n")
				dataMu.Lock()
				data.TotalPackages = len(lines)
				dataMu.Unlock()

				installedNamesMu.Lock()
				for _, line := range lines {
					if name := strings.TrimSpace(line); name != "" {
						installedNames[name] = true
					}
				}
				installedNamesMu.Unlock()
			} else {
				addErr("total packages", err)
			}
		}()

		// Explicitly Installed
		wg.Add(1)
		go func() {
			defer wg.Done()
			args := BuildAURCommand(c, "query-explicit", "-Qe")
			out, err := activeRunner.RunContext(ctx, args[0], args[1:]...)
			if err == nil {
				val := countLines(string(out))
				dataMu.Lock()
				data.ExplicitlyInstalled = val
				dataMu.Unlock()
			} else {
				addErr("explicitly installed", err)
			}
		}()

		// Foreign Packages
		wg.Add(1)
		go func() {
			defer wg.Done()
			args := BuildAURCommand(c, "query-foreign", "-Qm")
			out, err := activeRunner.RunContext(ctx, args[0], args[1:]...)
			if err == nil {
				val := countLines(string(out))
				dataMu.Lock()
				data.ForeignPackages = val
				dataMu.Unlock()
			} else {
				addErr("foreign packages", err)
			}
		}()

		// Orphans (with ExitCode 1 fix)
		wg.Add(1)
		go func() {
			defer wg.Done()
			args := BuildAURCommand(c, "query-orphans", "-Qdt")
			out, err := activeRunner.RunContext(ctx, args[0], args[1:]...)
			if err == nil {
				val := countLines(string(out))
				dataMu.Lock()
				data.Orphans = val
				dataMu.Unlock()
			} else {
				if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 && len(out) == 0 {
					dataMu.Lock()
					data.Orphans = 0
					dataMu.Unlock()
				} else {
					addErr("orphans", err)
				}
			}
		}()

		// Repository Distribution
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := activeRunner.RunContext(ctx, "pacman", "-Sl")
			if err == nil {
				dist := make(map[string]int)
				lines := strings.Split(strings.TrimSpace(string(out)), "\n")
				for _, line := range lines {
					if strings.Contains(line, "[installed") {
						parts := strings.Fields(line)
						if len(parts) > 0 {
							dist[parts[0]]++
						}
					}
				}
				dataMu.Lock()
				data.RepoDistribution = dist
				dataMu.Unlock()
			}
		}()

		// Package Statistics
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Many helpers support -Ps for stats
			out, err := activeRunner.RunContext(ctx, c.Commands.AurHelper, "-Ps")
			if err == nil {
				ts, tsb, miss, top := parseParuStats(string(out))
				dataMu.Lock()
				data.TotalSize = ts
				data.TotalSizeBytes = tsb
				data.MissingFromAUR = miss
				data.TopPackages = top
				dataMu.Unlock()
			} else {
				addErr("package statistics", err)
			}
		}()

		// Recently Installed
		wg.Add(1)
		go func() {
			defer wg.Done()
			out, err := activeRunner.RunContext(ctx, "grep", " installed ", "/var/log/pacman.log")
			if err == nil {
				var recent []RecentPackage
				lines := strings.Split(strings.TrimSpace(string(out)), "\n")
				seen := make(map[string]bool)
				for i := len(lines) - 1; i >= 0 && len(recent) < 5; i-- {
					parts := strings.Fields(lines[i])
					if len(parts) >= 4 {
						pkg := parts[3]
						if !seen[pkg] {
							ts := parts[0]
							if len(ts) > 17 {
								ts = ts[1:11] + " " + ts[12:17]
							}
							recent = append(recent, RecentPackage{
								Name:      pkg,
								Timestamp: ts,
							})
							seen[pkg] = true
						}
					}
				}
				dataMu.Lock()
				data.RecentlyInstalled = recent
				dataMu.Unlock()
			}
		}()

		// Caches
		wg.Add(1)
		go func() {
			defer wg.Done()
			pacmanCachePath := "/var/cache/pacman/pkg"
			pacmanSize := calculateDirSize(pacmanCachePath)

			aurClonePath, err := GetAURCacheDir(c)
			if err != nil {
				addErr("AUR cache path", err)
				return
			}
			// Base is parent of clone for paru, or the dir itself for yay?
			// User logic: paru -> ~/.cache/paru/clone, yay -> ~/.cache/yay
			aurBase := aurClonePath
			if c.Commands.AurHelper == "paru" {
				aurBase = filepath.Dir(aurClonePath)
			}

			// Fetch installed list locally for this goroutine to avoid complex sync
			installed := make(map[string]bool)
			args := BuildAURCommand(c, "query-all", "-Qq")
			if out, err := activeRunner.RunContext(ctx, args[0], args[1:]...); err == nil {
				for _, name := range strings.Split(string(out), "\n") {
					if n := strings.TrimSpace(name); n != "" {
						installed[n] = true
					}
				}
			}

			type aurCacheFile struct {
				path    string
				size    int64
				modTime time.Time
			}
			aurFiles := make(map[string][]aurCacheFile)

			pacmanHogs := make(map[string]int64)
			if entries, err := os.ReadDir(pacmanCachePath); err == nil {
				for _, entry := range entries {
					if entry.IsDir() {
						continue
					}
					name := entry.Name()
					if !strings.HasSuffix(name, ".pkg.tar.zst") && !strings.HasSuffix(name, ".pkg.tar.xz") {
						continue
					}
					parts := strings.Split(name, "-")
					if len(parts) > 3 {
						baseName := strings.Join(parts[:len(parts)-3], "-")
						if info, err := entry.Info(); err == nil {
							pacmanHogs[baseName] += info.Size()
						}
					}
				}
			}

			_ = filepath.WalkDir(aurClonePath, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				name := d.Name()
				if !strings.HasSuffix(name, ".pkg.tar.zst") && !strings.HasSuffix(name, ".pkg.tar.xz") {
					return nil
				}
				parts := strings.Split(name, "-")
				if len(parts) > 3 {
					baseName := strings.Join(parts[:len(parts)-3], "-")
					if info, err := d.Info(); err == nil {
						aurFiles[baseName] = append(aurFiles[baseName], aurCacheFile{
							path:    path,
							size:    info.Size(),
							modTime: info.ModTime(),
						})
					}
				}
				return nil
			})

			aurHogs := make(map[string]int64)
			var aurOrphanSaved, aurKeep1Saved, aurKeep3Saved int64
			var aurOrphanCount, aurKeep1Count, aurKeep3Count int
			var aurPackagesSize int64

			for name, files := range aurFiles {
				// Sum for total hogs
				var totalSize int64
				for _, f := range files {
					totalSize += f.size
					aurPackagesSize += f.size
				}
				aurHogs[name] = totalSize

				// Calculate savings
				sort.Slice(files, func(i, j int) bool {
					return files[i].modTime.After(files[j].modTime)
				})

				// Orphans
				if !installed[name] {
					aurOrphanSaved += totalSize
					aurOrphanCount += len(files)
				}

				// Keep N logic
				if len(files) > 1 {
					for i := 1; i < len(files); i++ {
						aurKeep1Saved += files[i].size
						aurKeep1Count++
					}
				}
				if len(files) > 3 {
					for i := 3; i < len(files); i++ {
						aurKeep3Saved += files[i].size
						aurKeep3Count++
					}
				}
			}

			// Combine for AllCacheHogs
			combinedHogs := make(map[string]int64)
			for k, v := range pacmanHogs {
				combinedHogs[k] += v
			}
			for k, v := range aurHogs {
				combinedHogs[k] += v
			}

			type cacheEntry struct {
				name string
				size int64
			}
			sortEntries := func(m map[string]int64) []cacheEntry {
				var entries []cacheEntry
				for k, v := range m {
					entries = append(entries, cacheEntry{k, v})
				}
				sort.Slice(entries, func(i, j int) bool {
					return entries[i].size > entries[j].size
				})
				return entries
			}

			toPkgSize := func(entries []cacheEntry) []PackageSize {
				var res []PackageSize
				for _, e := range entries {
					res = append(res, PackageSize{
						Name:      e.name,
						Size:      formatBytes(e.size),
						SizeBytes: e.size,
					})
				}
				return res
			}

			allSorted := sortEntries(combinedHogs)

			// Filter removed packages from cache
			removedPacman := make(map[string]int64)
			for k, v := range pacmanHogs {
				if !installed[k] {
					removedPacman[k] = v
				}
			}
			removedAur := make(map[string]int64)
			for k, v := range aurHogs {
				if !installed[k] {
					removedAur[k] = v
				}
			}

			var topHogs []PackageSize
			for i := 0; i < 5 && i < len(allSorted); i++ {
				topHogs = append(topHogs, PackageSize{
					Name:      allSorted[i].name,
					Size:      formatBytes(allSorted[i].size),
					SizeBytes: allSorted[i].size,
				})
			}

			aurBaseSize := calculateDirSize(aurBase)
			totalCache := pacmanSize + aurBaseSize

			// Estimates for paccache (summing system and manual paru calculation)
			estimatesPacman := make(map[confirmationType]string)
			estimatesAur := make(map[confirmationType]string)
			estimatesTotal := make(map[confirmationType]string)

			fetchDetailedEstimate := func(ct confirmationType, aurCount int, aurSaved int64, args ...string) {
				out, _ := activeRunner.RunContext(ctx, "paccache", append([]string{"-d", "-c", pacmanCachePath}, args...)...)
				pacmanCount, pacmanSavedStr := parsePaccacheDryRunDetailed(string(out))
				pacmanSavedBytes := parseSizeToBytes(pacmanSavedStr)

				if pacmanCount > 0 {
					estimatesPacman[ct] = fmt.Sprintf("%d pkgs (%s)", pacmanCount, pacmanSavedStr)
				} else {
					estimatesPacman[ct] = "0 B"
				}

				if aurCount > 0 {
					estimatesAur[ct] = fmt.Sprintf("%d pkgs (%s)", aurCount, formatBytes(aurSaved))
				} else {
					estimatesAur[ct] = "0 B"
				}

				totalBytes := pacmanSavedBytes + aurSaved
				if totalBytes > 0 {
					estimatesTotal[ct] = formatBytes(totalBytes)
				} else {
					estimatesTotal[ct] = "0 B"
				}
			}

			fetchDetailedEstimate(confirmCleanKeep3, aurKeep3Count, aurKeep3Saved, "-k3")
			fetchDetailedEstimate(confirmCleanKeep1, aurKeep1Count, aurKeep1Saved, "-k1")
			fetchDetailedEstimate(confirmCleanRemoved, aurOrphanCount, aurOrphanSaved, "-uk0")

			estimatesPacman[confirmCleanNuke] = formatBytes(pacmanSize)
			estimatesAur[confirmCleanNuke] = formatBytes(aurBaseSize)
			estimatesTotal[confirmCleanNuke] = formatBytes(pacmanSize + aurBaseSize)

			dataMu.Lock()
			data.TopCacheHogs = topHogs
			data.AllCacheHogs = toPkgSize(allSorted)
			data.RemovedPacmanCache = toPkgSize(sortEntries(removedPacman))
			data.RemovedAurCache = toPkgSize(sortEntries(removedAur))
			data.CacheFreedPacman = estimatesPacman
			data.CacheFreedAur = estimatesAur
			data.CacheFreedEstimates = estimatesTotal
			data.PacmanCachePath = pacmanCachePath
			data.PacmanCacheSizeBytes = pacmanSize
			data.PacmanCacheSize = formatBytes(pacmanSize)
			data.AurCachePath = aurBase
			data.AurCacheSizeBytes = aurBaseSize
			data.AurCacheSize = formatBytes(aurBaseSize)
			data.CleanerSizeBytes = totalCache
			data.CleanerSize = formatBytes(totalCache)
			dataMu.Unlock()
		}()
		// Disk Usage
		wg.Add(1)
		go func() {
			defer wg.Done()
			var stat syscall.Statfs_t
			if err := syscall.Statfs("/", &stat); err == nil {
				// Safe conversion: Bsize is int64, we need to handle it safely for uint64 multiplication
				// On Linux, Bsize is always positive, so this is safe
				var bsize uint64
				if stat.Bsize > 0 {
					bsize = uint64(stat.Bsize) // #nosec G115 - Bsize is always positive on Linux
				}
				total := stat.Blocks * bsize
				free := stat.Bfree * bsize
				used := total - free

				// Safe conversion for formatBytes: cap at max int64 to prevent overflow
				const maxInt64 = uint64(1<<63 - 1)
				safeTotal := total
				if safeTotal > maxInt64 {
					safeTotal = maxInt64
				}
				safeFree := free
				if safeFree > maxInt64 {
					safeFree = maxInt64
				}
				safeUsed := used
				if safeUsed > maxInt64 {
					safeUsed = maxInt64
				}

				dataMu.Lock()
				data.DiskTotal = formatBytes(int64(safeTotal)) // #nosec G115 - capped at maxInt64
				data.DiskFree = formatBytes(int64(safeFree))   // #nosec G115 - capped at maxInt64
				data.DiskUsed = formatBytes(int64(safeUsed))   // #nosec G115 - capped at maxInt64
				if total > 0 {
					data.DiskUsedPercent = float64(used) / float64(total)
				}
				dataMu.Unlock()
			}
		}()

		wg.Wait()

		dataMu.Lock()
		defer dataMu.Unlock()
		if len(errs) > 0 {
			return dashboardMsg{data: data, err: fmt.Errorf("errors loading dashboard: %v", errs)}
		}
		return dashboardMsg{data: data}
	}
}

// parseParuStats extracts total installed size, missing AUR package count,
// and top 10 biggest packages from paru -Ps output.
func parseParuStats(output string) (totalSize string, totalSizeBytes int64, missingAUR int, topPackages []PackageSize) {
	lines := strings.Split(output, "\n")
	inTopPackages := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, "biggest packages") {
			inTopPackages = true
			continue
		}

		if inTopPackages && line == "" {
			inTopPackages = false
			continue
		}

		if inTopPackages && strings.HasPrefix(line, "===") {
			continue
		}

		if inTopPackages {
			if len(topPackages) >= 10 {
				inTopPackages = false
				continue
			}
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				sizeStr := strings.TrimSpace(parts[1])
				if name != "" {
					sb := parseSizeToBytes(sizeStr)
					topPackages = append(topPackages, PackageSize{
						Name:      name,
						Size:      formatBytes(sb),
						SizeBytes: sb,
					})
				}
			}
			continue
		}

		if strings.Contains(line, "Total Size occupied") || strings.Contains(line, "Total Installed Size") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				ts := strings.TrimSpace(parts[1])
				totalSizeBytes = parseSizeToBytes(ts)
				totalSize = formatBytes(totalSizeBytes)
			}
		}
		if strings.Contains(line, "Missing") && strings.Contains(line, "AUR") {

			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				_, _ = fmt.Sscanf(strings.TrimSpace(parts[1]), "%d", &missingAUR)
			}
		}
	}
	return
}

// parsePaccacheDryRunDetailed extracts both package count and disk space saved from paccache -d output
func parsePaccacheDryRunDetailed(output string) (int, string) {
	if strings.Contains(output, "no candidate packages found") {
		return 0, "0 B"
	}
	// Format: ==> finished dry run: 78 candidates (disk space saved: 782.43 MiB)
	var count int
	if idx := strings.Index(output, "finished dry run: "); idx != -1 {
		_, _ = fmt.Sscanf(output[idx+len("finished dry run: "):], "%d", &count)
	}

	parts := strings.Split(output, "disk space saved: ")
	if len(parts) > 1 {
		resParts := strings.Fields(parts[1])
		if len(resParts) >= 2 {
			sizeStr := resParts[0] + " " + strings.TrimSuffix(resParts[1], ")")
			return count, formatBytes(parseSizeToBytes(sizeStr))
		}
	}
	return count, "0 B"
}

// parsePaccacheDryRun extracts the "disk space saved" string from paccache -d output
func parsePaccacheDryRun(output string) string {
	_, size := parsePaccacheDryRunDetailed(output)
	return size
}

func (m *model) renderDashboard(helpText string, innerWidth, innerHeight int) string {

	activeColor := modeColors[m.mode]
	if activeColor == "" {
		activeColor = defaultBorderColor
	}
	borderStyle := baseBorderStyle.BorderForeground(activeColor)

	footerLine := renderCenteredFooter(helpText, innerWidth)

	if m.loading {
		// Total height is innerHeight. Footer is 1 line.
		// dashboardPanel Total Height = innerHeight - 1.
		panelTotalHeight := innerHeight - 1
		if panelTotalHeight < 3 {
			panelTotalHeight = 3
		}

		loadingBox := borderStyle.
			Width(innerWidth - 2).
			Height(max(0, panelTotalHeight-2)).
			Render(lipgloss.Place(innerWidth-2, max(0, panelTotalHeight-2), lipgloss.Center, lipgloss.Center, "Loading system statistics..."))
		return lipgloss.JoinVertical(lipgloss.Left, strings.TrimSuffix(loadingBox, "\n"), footerLine)
	}

	// ═══════════════════════════════════════════════════════
	// Bar Layout Constants - ensures all bars align perfectly
	// ═══════════════════════════════════════════════════════
	// availableDashboardWidth accounts for dashboardPanel borders (-2) and dashboardContent padding (-4 total)
	safeWidth := innerWidth - 6
	if safeWidth < 60 {
		safeWidth = 60
	}

	barStartCol := 0
	availableBarWidth := safeWidth
	if availableBarWidth < 20 {
		availableBarWidth = 20
	}

	renderBarLine := func(bar string, suffix string) string {
		if suffix == "" {
			return bar
		}
		return bar + " " + suffix
	}

	var dashboard strings.Builder

	// Use centralized color constants
	greenColor := colorGreen
	redColor := colorRed
	yellowColor := colorYellow
	orangeColor := colorOrange
	cyanColor := colorCyan
	dimColor := colorLightGray

	boxTitleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(currentTheme.TitleColor)

	topBoxWidth := (safeWidth - 2) / 2
	if topBoxWidth < 30 {
		topBoxWidth = 30
	}

	renderBarMetricLine := func(char, label, valueStr string, value, total int64, color lipgloss.Color, currentBoxWidth int, colorWhenZero bool) string {
		prefix := ""
		if char != "" {
			prefix = fmt.Sprintf("[%s]", char)
		}
		sStyle := lipgloss.NewStyle().Bold(true)
		vStyle := lipgloss.NewStyle().Bold(true)

		if colorWhenZero || value > 0 {
			sStyle = sStyle.Foreground(color)
			vStyle = vStyle.Foreground(color)
		}

		fullLabel := prefix + label
		// Use Width(12) and Width(10) for symmetry between boxes
		labelPart := lipgloss.NewStyle().Width(12).Render(sStyle.Render(fullLabel))
		valuePart := lipgloss.NewStyle().Width(10).Render(vStyle.Render(valueStr))

		internalBoxWidth := currentBoxWidth - 4
		const textWidth = 26 // 12 + 10 + 4
		dynamicBarWidth := internalBoxWidth - textWidth - 2
		if dynamicBarWidth < 5 {
			dynamicBarWidth = 5
		}

		bar := ""
		if value > 0 && total > 0 {
			filled := int(float64(value) / float64(total) * float64(dynamicBarWidth))
			if filled < 1 && value > 0 {
				filled = 1
			}
			if filled > dynamicBarWidth {
				filled = dynamicBarWidth
			}
			bar = lipgloss.NewStyle().Background(color).Render(strings.Repeat(" ", filled))
		}

		return "  " + labelPart + " │ " + valuePart + " " + bar
	}

	renderCountLine := func(char, label string, count int, color lipgloss.Color, currentBoxWidth int) string {
		return renderBarMetricLine(char, label, fmt.Sprintf("%d", count), int64(count), int64(m.dashboard.TotalPackages), color, currentBoxWidth, false)
	}

	countsLines := []string{
		renderCountLine("t", "otal", m.dashboard.TotalPackages, cyanColor, topBoxWidth),
		renderCountLine("e", "xplicit", m.dashboard.ExplicitlyInstalled, greenColor, topBoxWidth),
		renderCountLine("f", "oreign", m.dashboard.ForeignPackages, yellowColor, topBoxWidth),
		renderCountLine("m", "issing", m.dashboard.MissingFromAUR, redColor, topBoxWidth),
		renderCountLine("o", "rphans", m.dashboard.Orphans, redColor, topBoxWidth),
	}

	// Calculate Disk Usage Breakdown for Disk Space Box
	totalDiskBytes := parseSizeToBytes(m.dashboard.DiskTotal)
	if totalDiskBytes == 0 {
		totalDiskBytes = 1
	}

	usedDiskBytes := parseSizeToBytes(m.dashboard.DiskUsed)
	otherUsedBytes := usedDiskBytes - m.dashboard.TotalSizeBytes - m.dashboard.CleanerSizeBytes
	if otherUsedBytes < 0 {
		otherUsedBytes = 0
	}

	freeBytes := totalDiskBytes - usedDiskBytes
	if freeBytes < 0 {
		freeBytes = 0
	}

	otherColor := currentTheme.SelectedColor

	renderDiskLine := func(char, label, size string, bytes int64, color lipgloss.Color, currentBoxWidth int) string {
		return renderBarMetricLine(char, label, size, bytes, totalDiskBytes, color, currentBoxWidth, true)
	}

	storageLines := []string{
		renderDiskLine("", "total", m.dashboard.DiskTotal, totalDiskBytes, currentTheme.TitleColor, topBoxWidth),
		renderDiskLine("", "packages", m.dashboard.TotalSize, m.dashboard.TotalSizeBytes, cyanColor, topBoxWidth),
		renderDiskLine("c", "ache", m.dashboard.CleanerSize, m.dashboard.CleanerSizeBytes, orangeColor, topBoxWidth),
		renderDiskLine("", "other", formatBytes(otherUsedBytes), otherUsedBytes, otherColor, topBoxWidth),
		renderDiskLine("", "free", formatBytes(freeBytes), freeBytes, currentTheme.DimText, topBoxWidth),
	}

	borderColor := lipgloss.NewStyle().Foreground(activeColor)

	renderBox := func(title string, lines []string, width int) string {
		var b strings.Builder

		innerWidth := width - 4
		if innerWidth < 10 {
			innerWidth = 10
		}

		titleLen := lipgloss.Width(title)
		topLeft := borderColor.Render("╭─")
		topRight := borderColor.Render("─╮")
		topPadding := innerWidth - titleLen
		if topPadding < 0 {
			topPadding = 0
		}
		b.WriteString(topLeft + title + borderColor.Render(strings.Repeat("─", topPadding)) + topRight + "\n")

		leftBorder := borderColor.Render("│ ")
		rightBorder := borderColor.Render(" │")
		for _, line := range lines {
			// Ensure no internal newlines break the box formatting
			cleanLine := strings.ReplaceAll(line, "\n", " ")
			lineWidth := lipgloss.Width(cleanLine)
			padding := innerWidth - lineWidth
			if padding < 0 {
				cleanLine = truncateWithAnsi(cleanLine, innerWidth)
				padding = 0
			}
			b.WriteString(leftBorder + cleanLine + strings.Repeat(" ", padding) + rightBorder + "\n")
		}

		b.WriteString(borderColor.Render("╰" + strings.Repeat("─", innerWidth+2) + "╯"))

		return b.String()
	}

	// ═══════════════════════════════════════════════════════
	// Top Row: Package Counts & Disk Space
	// ═══════════════════════════════════════════════════════
	countsBox := renderBox(boxTitleStyle.Render(" \U000f03d7  Package Counts "), countsLines, topBoxWidth)
	diskSpaceTitle := fmt.Sprintf(" \uf0c7  Disk Space (/:%.0f%%) ", m.dashboard.DiskUsedPercent*100)
	storageBox := renderBox(boxTitleStyle.Render(diskSpaceTitle), storageLines, topBoxWidth)

	topRow := lipgloss.JoinHorizontal(lipgloss.Top, countsBox, "  ", storageBox)
	dashboard.WriteString(topRow + "\n\n")

	// Repo Distribution Bar
	repoTitle := lipgloss.NewStyle().Bold(true).Foreground(currentTheme.TitleColor).
		Render("\ueb9c  Repository Distribution")
	dashboard.WriteString(repoTitle + "\n")

	if m.dashboard.TotalPackages > 0 {
		var repoBar strings.Builder

		// Identify all repositories and group non-core/extra/multilib into "Other"
		type repoStat struct {
			name  string
			count int
			color lipgloss.Color
		}

		coreCount := m.dashboard.RepoDistribution["core"]
		extraCount := m.dashboard.RepoDistribution["extra"]
		multilibCount := m.dashboard.RepoDistribution["multilib"]
		aurCount := m.dashboard.ForeignPackages

		// Calculate "Other" packages (any repo not in core/extra/multilib, or manually installed)
		accounted := coreCount + extraCount + multilibCount + aurCount
		otherCount := m.dashboard.TotalPackages - accounted
		if otherCount < 0 {
			otherCount = 0
		}

		allStats := []repoStat{
			{"Core", coreCount, sourceColors["core"]},
			{"Extra", extraCount, sourceColors["extra"]},
			{"Multilib", multilibCount, sourceColors["multilib"]},
			{"AUR", aurCount, sourceColors["aur"]},
		}
		if otherCount > 0 {
			allStats = append(allStats, repoStat{"Other", otherCount, currentTheme.DimText})
		}

		// Calculate initial widths using floor
		widths := make([]int, len(allStats))
		totalWUsed := 0
		maxWIdx := 0
		for i, stat := range allStats {
			if stat.count > 0 {
				w := int(float64(stat.count) / float64(m.dashboard.TotalPackages) * float64(availableBarWidth))
				if w < 1 {
					w = 1
				}
				widths[i] = w
				totalWUsed += w
				if widths[i] > widths[maxWIdx] {
					maxWIdx = i
				}
			}
		}

		// Distribute remainder to the largest segment to fill bar perfectly
		if totalWUsed < availableBarWidth {
			widths[maxWIdx] += (availableBarWidth - totalWUsed)
		} else if totalWUsed > availableBarWidth {
			// This could happen due to the "at least 1 cell" rule
			widths[maxWIdx] -= (totalWUsed - availableBarWidth)
			if widths[maxWIdx] < 1 {
				widths[maxWIdx] = 1
			}
		}

		// Build the bar
		for i, stat := range allStats {
			if widths[i] > 0 {
				repoBar.WriteString(lipgloss.NewStyle().Background(stat.color).Render(strings.Repeat(" ", widths[i])))
			}
		}

		dashboard.WriteString(renderBarLine(repoBar.String(), "") + "\n\n")

		// Legend blocks below bar
		renderBlock := func(label string, count int, color lipgloss.Color) string {
			cStr := fmt.Sprintf("%d", count)
			if len(label)%2 != len(cStr)%2 {
				cStr = " " + cStr
			}
			w := len(label)
			if len(cStr) > w {
				w = len(cStr)
			}
			w += 4
			style := lipgloss.NewStyle().Background(color).Foreground(currentTheme.ScrollbarTrack).Bold(true).Width(w).Align(lipgloss.Center)
			return lipgloss.JoinVertical(lipgloss.Center, style.Render(label), style.Render(cStr))
		}

		var legendBlocks []string
		for _, stat := range allStats {
			if stat.count > 0 {
				legendBlocks = append(legendBlocks, renderBlock(stat.name, stat.count, stat.color))
			}
		}

		repoLegend := lipgloss.JoinHorizontal(lipgloss.Top, legendBlocks...)
		repoLegendWidth := lipgloss.Width(repoLegend)
		repoPadding := (availableBarWidth - repoLegendWidth) / 2
		if repoPadding < 0 {
			repoPadding = 0
		}

		legendLines := strings.Split(repoLegend, "\n")
		for _, line := range legendLines {
			dashboard.WriteString(fmt.Sprintf("%*s%*s%s\n", barStartCol, "", repoPadding, "", line))
		}
		dashboard.WriteString("\n")
	}

	// ═══════════════════════════════════════════════════════
	// Bottom Row: Top by Weight, Cache Hogs & Recently Installed
	// ═══════════════════════════════════════════════════════
	bottomVizWidth := (safeWidth - 4) / 3 // 2 spaces between boxes
	if bottomVizWidth < 30 {
		bottomVizWidth = 30
	}
	boxInnerW := bottomVizWidth - 4
	if boxInnerW < 10 {
		boxInnerW = 10
	}

	formatTableRow := func(rank int, name string, nameW int, nameColor lipgloss.Color, valStr string, valStyle lipgloss.Style) string {
		rankStyle := lipgloss.NewStyle().Foreground(currentTheme.DimText)
		nameStyle := lipgloss.NewStyle().Foreground(nameColor)
		return fmt.Sprintf("%s %s %s",
			rankStyle.Render(fmt.Sprintf("%2d.", rank)),
			lipgloss.NewStyle().Width(nameW).Render(nameStyle.Render(truncateWithAnsi(name, nameW))),
			valStyle.Render(valStr))
	}

	var topWeightLines []string
	if len(m.dashboard.TopPackages) > 0 {
		count := len(m.dashboard.TopPackages)
		if count > 5 {
			count = 5
		} // Cut down to 5

		nameW := boxInnerW - 15 // rank(3) + space(1) + space(1) + size(~10)
		if nameW < 5 {
			nameW = 5
		}

		for i := 0; i < count; i++ {
			pkg := m.dashboard.TopPackages[i]
			pkgSizeBytes := parseSizeToBytes(pkg.Size)
			sizeColor := cyanColor
			if pkgSizeBytes > 1024*1024*1024 {
				sizeColor = redColor
			} else if pkgSizeBytes > 500*1024*1024 {
				sizeColor = orangeColor
			}

			topWeightLines = append(topWeightLines, formatTableRow(i+1, pkg.Name, nameW, cyanColor, fmt.Sprintf("%10s", pkg.Size), lipgloss.NewStyle().Foreground(sizeColor)))
		}
	}
	topWeightBox := renderBox(boxTitleStyle.Render(" \ueddf  Top by Weight "), topWeightLines, bottomVizWidth)

	var topCacheLines []string
	if len(m.dashboard.TopCacheHogs) > 0 {
		count := len(m.dashboard.TopCacheHogs)
		if count > 5 {
			count = 5
		}

		nameW := boxInnerW - 15 // rank(3) + space(1) + space(1) + size(~10)
		if nameW < 5 {
			nameW = 5
		}

		for i := 0; i < count; i++ {
			pkg := m.dashboard.TopCacheHogs[i]
			topCacheLines = append(topCacheLines, formatTableRow(i+1, pkg.Name, nameW, orangeColor, fmt.Sprintf("%10s", pkg.Size), lipgloss.NewStyle().Foreground(yellowColor)))
		}
	}
	cacheHogsBox := renderBox(boxTitleStyle.Render(" \uf0c7  Top Cache Hogs "), topCacheLines, bottomVizWidth)

	var recentLines []string
	for i, pkg := range m.dashboard.RecentlyInstalled {
		nameW := boxInnerW - 22 // rank(3) + space(1) + time(16) + space(1) + extra
		if nameW < 5 {
			nameW = 5
		}

		recentLines = append(recentLines, formatTableRow(i+1, pkg.Name, nameW, greenColor, pkg.Timestamp, lipgloss.NewStyle().Foreground(dimColor)))
	}
	recentBox := renderBox(boxTitleStyle.Render(" \uf1da  Recently Installed "), recentLines, bottomVizWidth)

	bottomRow := lipgloss.JoinHorizontal(lipgloss.Top, topWeightBox, "  ", cacheHogsBox, "  ", recentBox)
	dashboard.WriteString(bottomRow)

	// Total height = innerHeight. Footer = 1 line.
	// dashboardPanel Total Height = innerHeight - 1.
	panelTotalHeight := innerHeight - 1
	if panelTotalHeight < 5 {
		panelTotalHeight = 5
	}
	panelInnerHeight := panelTotalHeight - 2

	dashboardContent := lipgloss.NewStyle().
		Width(innerWidth-4). // account for Padding(0, 1)
		Padding(0, 1).
		Render(truncateHeight(dashboard.String(), panelInnerHeight))

	dashboardPanel := borderStyle.
		Width(innerWidth - 2).
		Height(max(0, panelTotalHeight-2)).
		Render(dashboardContent)

	return SafeJoinVertical(innerWidth, innerHeight, "", []string{dashboardPanel}, footerLine)
}
