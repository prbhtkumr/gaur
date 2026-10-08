package main

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func getInstalledPackages(r ...CommandRunner) tea.Cmd {
	return getInstalledPackagesWithContext(context.Background(), r...)
}

func getInstalledPackagesWithContext(ctx context.Context, r ...CommandRunner) tea.Cmd {
	return func() tea.Msg {
		activeRunner := getActiveRunner(r...)
		out, err := activeRunner.RunContext(ctx, "pacman", "-Qi")
		if err != nil {
			return installedPackagesMsg{err: err}
		}

		packages, err := parseInstalledPackagesWithContext(ctx, string(out), r...)
		if err != nil {
			return installedPackagesMsg{err: fmt.Errorf("failed to parse installed packages: %w", err)}
		}
		return installedPackagesMsg{packages: packages}
	}
}

func parseInstalledPackages(output string, r ...CommandRunner) ([]Package, error) {
	return parseInstalledPackagesWithContext(context.Background(), output, r...)
}

func parseInstalledPackagesWithContext(ctx context.Context, output string, r ...CommandRunner) ([]Package, error) {
	var packages []Package
	blocks := strings.Split(output, "\n\n")

	for _, block := range blocks {
		if strings.TrimSpace(block) == "" {
			continue
		}

		var pkg Package
		pkg.Installed = true
		pkg.Source = "local"

		lines := strings.Split(block, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					pkg.Name = strings.TrimSpace(parts[1])
				}
			} else if strings.HasPrefix(line, "Version") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					pkg.Version = strings.TrimSpace(parts[1])
				}
			} else if strings.HasPrefix(line, "Description") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					pkg.Description = strings.TrimSpace(parts[1])
				}
			}
		}

		if pkg.Name != "" {
			packages = append(packages, pkg)
		}
	}

	activeRunner := getActiveRunner(r...)

	repoMap, err := queryPackageRepoMap(ctx, activeRunner)
	if err != nil {
		return nil, fmt.Errorf("failed to query package repositories: %w", err)
	}
	for i := range packages {
		if repo, ok := repoMap[packages[i].Name]; ok {
			packages[i].Source = repo
		}
	}

	if foreignPkgs, err := queryPackageSet(ctx, activeRunner, "-Qm"); err == nil {
		for i := range packages {
			if foreignPkgs[packages[i].Name] {
				packages[i].Source = "aur"
			}
		}
	}

	if explicitPkgs, err := queryPackageSet(ctx, activeRunner, "-Qe"); err == nil {
		for i := range packages {
			packages[i].Explicit = explicitPkgs[packages[i].Name]
		}
	}

	if orphanPkgs, err := queryPackageSet(ctx, activeRunner, "-Qdt"); err == nil {
		for i := range packages {
			packages[i].Orphan = orphanPkgs[packages[i].Name]
		}
	}

	return packages, nil
}

// computeFilterInstalledPackages processes remove filters and packages for a query.
func (m *model) computeFilterInstalledPackages(query string) ([]Package, map[int][]int) {
	if query == "" {
		return m.installed, nil
	}

	filters, searchQuery := parseRemoveFilter(query)

	candidates := m.installed
	if len(filters) > 0 {
		var filtered []Package
		for _, pkg := range m.installed {
			match := false
			if filters["total"] {
				match = true
			}
			if filters["explicit"] && pkg.Explicit {
				match = true
			}
			if filters["foreign"] && pkg.Source == "aur" {
				match = true
			}
			if filters["orphan"] && pkg.Orphan {
				match = true
			}

			if match {
				filtered = append(filtered, pkg)
			}
		}
		candidates = filtered
	}

	if searchQuery == "" {
		return candidates, nil
	}

	runner := m.getRunner()
	filtered := fuzzyFilter(candidates, searchQuery, runner)
	matchIndices := computeAllMatchIndices(filtered, searchQuery)
	return filtered, matchIndices
}

func (m *model) filterInstalledPackages(query string) {
	pkgs, indices := m.computeFilterInstalledPackages(query)
	m.filteredInstalled = pkgs
	m.matchIndices = indices
}

// filterInstalledPackagesCmd executes fuzzy filtering asynchronously in a tea.Cmd to keep UI responsive.
func (m *model) filterInstalledPackagesCmd(query string) tea.Cmd {
	filters, searchQuery := parseRemoveFilter(query)

	candidates := make([]Package, len(m.installed))
	copy(candidates, m.installed)

	if len(filters) > 0 {
		var filtered []Package
		for _, pkg := range candidates {
			match := false
			if filters["total"] {
				match = true
			}
			if filters["explicit"] && pkg.Explicit {
				match = true
			}
			if filters["foreign"] && pkg.Source == "aur" {
				match = true
			}
			if filters["orphan"] && pkg.Orphan {
				match = true
			}

			if match {
				filtered = append(filtered, pkg)
			}
		}
		candidates = filtered
	}

	runner := m.getRunner()

	return func() tea.Msg {
		if searchQuery == "" {
			return filterResultMsg{
				query:        query,
				mode:         modeRemove,
				packages:     candidates,
				matchIndices: nil,
			}
		}

		filtered := fuzzyFilter(candidates, searchQuery, runner)
		matchIndices := computeAllMatchIndices(filtered, searchQuery)
		return filterResultMsg{
			query:        query,
			mode:         modeRemove,
			packages:     filtered,
			matchIndices: matchIndices,
		}
	}
}
