package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// checkUpdates fetches available updates using the AUR helper
func checkUpdates(c *Config, r ...CommandRunner) tea.Cmd {
	return checkUpdatesWithContext(context.Background(), c, r...)
}

// checkUpdatesWithContext fetches available updates with context cancellation support
func checkUpdatesWithContext(ctx context.Context, c *Config, r ...CommandRunner) tea.Cmd {
	return func() tea.Msg {
		activeRunner := getActiveRunner(r...)

		foreignOut, err := activeRunner.RunContext(ctx, "pacman", "-Qm")
		foreignPkgs := make(map[string]bool)
		if err == nil {
			for _, line := range strings.Split(string(foreignOut), "\n") {
				line = strings.TrimSpace(line)
				if line != "" {

					parts := strings.Fields(line)
					if len(parts) >= 1 {
						foreignPkgs[parts[0]] = true
					}
				}
			}
		}

		repoOut, err := activeRunner.RunContext(ctx, "pacman", "-Sl")
		repoMap := make(map[string]string)
		if err == nil {
			for _, line := range strings.Split(string(repoOut), "\n") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {

					repoName := parts[0]
					pkgName := parts[1]
					repoMap[pkgName] = repoName
				}
			}
		}

		args := BuildAURCommand(c, "check-updates")
		stdout, err := activeRunner.RunContext(ctx, args[0], args[1:]...)
		if err != nil {

			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {

				return updateCheckMsg{packages: []Package{}}
			}

			return updateCheckMsg{packages: nil, err: fmt.Errorf("failed to check for updates: %w", err)}
		}

		// Step 4: Parse updates and assign accurate source
		var packages []Package
		for _, line := range strings.Split(strings.TrimSpace(string(stdout)), "\n") {
			if line == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				pkgName := parts[0]

				if !isValidPackageName(pkgName) {
					continue
				}
				pkg := Package{
					Name:      pkgName,
					Version:   strings.Join(parts[1:], " "),
					Installed: true,
				}

				if foreignPkgs[pkgName] {
					pkg.Source = "aur"
				} else if repoName, ok := repoMap[pkgName]; ok {
					pkg.Source = repoName
				} else {
					pkg.Source = "unknown"
				}
				packages = append(packages, pkg)
			}
		}
		return updateCheckMsg{packages: packages}
	}
}
