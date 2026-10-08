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

	repoMap := make(map[string]string)
	repoOut, err := activeRunner.RunContext(ctx, "pacman", "-Sl")
	if err != nil {
		return nil, fmt.Errorf("failed to query package repositories: %w", err)
	}
	for _, line := range strings.Split(string(repoOut), "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 {

			repoMap[parts[1]] = parts[0]
		}
	}

	for i := range packages {
		if repo, ok := repoMap[packages[i].Name]; ok {
			packages[i].Source = repo
		}
	}

	foreignOut, err := activeRunner.RunContext(ctx, "pacman", "-Qm")
	if err == nil {
		foreignPkgs := make(map[string]bool)
		for _, line := range strings.Split(string(foreignOut), "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				foreignPkgs[parts[0]] = true
			}
		}
		for i := range packages {
			if foreignPkgs[packages[i].Name] {
				packages[i].Source = "aur"
			}
		}
	}

	explicitOut, err := activeRunner.RunContext(ctx, "pacman", "-Qe")
	if err == nil {
		explicitPkgs := make(map[string]bool)
		for _, line := range strings.Split(string(explicitOut), "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				explicitPkgs[parts[0]] = true
			}
		}
		for i := range packages {
			packages[i].Explicit = explicitPkgs[packages[i].Name]
		}
	}

	orphanOut, err := activeRunner.RunContext(ctx, "pacman", "-Qdt")
	if err == nil {
		orphanPkgs := make(map[string]bool)
		for _, line := range strings.Split(string(orphanOut), "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				orphanPkgs[parts[0]] = true
			}
		}
		for i := range packages {
			packages[i].Orphan = orphanPkgs[packages[i].Name]
		}
	}

	return packages, nil
}

func (m *model) filterInstalledPackages(query string) {
	if query == "" {
		m.filteredInstalled = m.installed
		m.matchIndices = nil
		return
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
		m.filteredInstalled = candidates
		m.matchIndices = nil
		return
	}

	m.filteredInstalled = fuzzyFilter(candidates, searchQuery, m.getRunner())
	m.matchIndices = computeAllMatchIndices(m.filteredInstalled, searchQuery)
}
