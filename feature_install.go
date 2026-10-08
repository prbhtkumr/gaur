package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Commands
// loadRepoPackages loads all packages from local pacman database
func loadRepoPackages(r ...CommandRunner) tea.Cmd {
	return func() tea.Msg {
		activeRunner := getActiveRunner(r...)
		stdout, err := activeRunner.Run("pacman", "-Sl")
		if err != nil {
			return repoPackagesMsg{err: err}
		}

		installedOut, err := activeRunner.Run("pacman", "-Qq")
		if err != nil {
			return repoPackagesMsg{err: fmt.Errorf("failed to get installed packages list: %w", err)}
		}

		installedSet := make(map[string]bool)
		for _, name := range strings.Split(string(installedOut), "\n") {
			name = strings.TrimSpace(name)
			if name != "" {
				installedSet[name] = true
			}
		}

		// Parse "repo name version [installed]" format
		var packages []Package
		for _, line := range strings.Split(string(stdout), "\n") {
			parts := strings.Fields(line)
			if len(parts) < 3 {
				continue
			}
			pkg := Package{
				Source:    parts[0],
				Name:      parts[1],
				Version:   parts[2],
				Installed: installedSet[parts[1]] || (len(parts) > 3 && parts[3] == "[installed]"),
			}
			packages = append(packages, pkg)
		}

		return repoPackagesMsg{packages: packages}
	}
}

// filterAllPackages combines repo and AUR packages, then fuzzy filters together
// This ensures fzf ranks all packages by relevance to the query
// Supports repo filtering with prefixes: c (core), e (extra), m (multilib), a (aur)
// Filters can be combined: ae:, cem:, aem: etc.
func (m *model) filterAllPackages(query string) {
	if query == "" {
		m.filtered = []Package{}
		m.matchIndices = nil
		return
	}

	repoFilters, searchQuery := parseRepoFilter(query)
	shouldIncludeAUR := len(repoFilters) == 0 || repoFilters["aur"]

	allPackages := make([]Package, 0, len(m.repoPackages))
	allPackages = append(allPackages, m.repoPackages...)
	if shouldIncludeAUR {
		allPackages = append(allPackages, m.aurPackages...)
	}

	if len(repoFilters) > 0 {
		var filtered []Package
		for _, pkg := range allPackages {
			if repoFilters[pkg.Source] {
				filtered = append(filtered, pkg)
			}
		}
		allPackages = filtered
	}

	if len(allPackages) == 0 {
		m.filtered = []Package{}
		m.matchIndices = nil
		return
	}

	if searchQuery == "" {
		m.filtered = allPackages
		m.matchIndices = nil
		return
	}

	m.filtered = fuzzyFilter(allPackages, searchQuery, m.getRunner())

	m.matchIndices = computeAllMatchIndices(m.filtered, searchQuery)
}

func searchAUR(c *Config, query string, r ...CommandRunner) tea.Cmd {
	return func() tea.Msg {
		if query == "" {
			return aurSearchMsg{packages: []Package{}, query: query}
		}

		start := time.Now()

		// Sanitize search query
		var sanitized strings.Builder
		for _, r := range query {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
				sanitized.WriteRune(r)
			} else if r == ' ' {
				sanitized.WriteRune('-')
			}
		}
		searchQuery := sanitized.String()
		if searchQuery == "" {
			return aurSearchMsg{packages: []Package{}, query: query}
		}

		args := BuildAURCommand(c, "search", searchQuery)
		stdout, err := getActiveRunner(r...).Run(args[0], args[1:]...)
		duration := time.Since(start)
		if err != nil {
			// Some helpers return exit 1 when no packages are found.
			// If output is empty, treat it as "no results" rather than an error.
			if len(stdout) == 0 {
				return aurSearchMsg{packages: []Package{}, query: query, timeTaken: duration}
			}

			// Include the output (stdout+stderr) in the error message for better diagnostics
			errMsg := strings.TrimSpace(string(stdout))
			if errMsg == "" {
				errMsg = err.Error()
			}

			// Clean up redundant prefixes often found in AUR helper output
			errMsg = strings.TrimPrefix(errMsg, "error: ")
			errMsg = strings.TrimPrefix(errMsg, "aur search failed: ")
			errMsg = strings.TrimSpace(errMsg)

			return aurSearchMsg{packages: nil, query: query, timeTaken: duration, err: fmt.Errorf("%s", simplifyErrorMessage(errMsg))}
		}
		if len(stdout) == 0 {
			return aurSearchMsg{packages: []Package{}, query: query, timeTaken: duration}
		}

		packages := parseAUROutput(string(stdout))
		return aurSearchMsg{packages: packages, query: query, timeTaken: duration}
	}
}
