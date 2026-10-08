package main

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func newTestThemeLoader() *ThemeLoader {
	tl := &ThemeLoader{
		themes: make(map[string]Theme),
	}

	defaultTheme := Theme{
		Name:             "Catppuccin Mocha",
		BorderColor:      lipgloss.Color("#6c7086"),
		SelectedColor:    lipgloss.Color("#cba6f7"),
		TextColor:        lipgloss.Color("#cdd6f4"),
		SubtleColor:      lipgloss.Color("#6c7086"),
		TitleColor:       lipgloss.Color("#f9e2af"),
		ScrollbarTrack:   lipgloss.Color("#181825"),
		ScrollbarThumb:   lipgloss.Color("#6c7086"),
		SelectionBG:      lipgloss.Color("#313244"),
		DimText:          lipgloss.Color("#6c7086"),
		InstallColor:     lipgloss.Color("#89b4fa"),
		DashboardColor:   lipgloss.Color("#f5c2e7"),
		RemoveColor:      lipgloss.Color("#f38ba8"),
		UpdateColor:      lipgloss.Color("#a6e3a1"),
		CacheColor:       lipgloss.Color("#cba6f7"),
		CoreColor:        lipgloss.Color("#a6e3a1"),
		ExtraColor:       lipgloss.Color("#89b4fa"),
		MultilibColor:    lipgloss.Color("#fab387"),
		AurColor:         lipgloss.Color("#cba6f7"),
		SuccessColor:     lipgloss.Color("#a6e3a1"),
		WarningColor:     lipgloss.Color("#f9e2af"),
		ErrorColor:       lipgloss.Color("#f38ba8"),
		HighlightColor:   lipgloss.Color("#f9e2af"),
		DashboardLabel:   lipgloss.Color("#cdd6f4"),
		DashboardValue:   lipgloss.Color("#89dceb"),
		DashboardWarning: lipgloss.Color("#f38ba8"),
		DashboardDesc:    lipgloss.Color("#a6adc8"),
		DialogBorder:     lipgloss.Color("#cba6f7"),
		ConfirmInstall:   lipgloss.Color("#89b4fa"),
		ConfirmRemove:    lipgloss.Color("#fab387"),
		ConfirmClean:     lipgloss.Color("#a6e3a1"),
		ConfirmNuke:      lipgloss.Color("#f38ba8"),
		ConfirmSelective: lipgloss.Color("#cba6f7"),
		ButtonBg:         lipgloss.Color("#45475a"),
		ButtonFg:         lipgloss.Color("#cdd6f4"),
		ButtonDangerBg:   lipgloss.Color("#f38ba8"),
		ProgressTrack:    lipgloss.Color("#313244"),
		AccentColor:      lipgloss.Color("#89b4fa"),
		SpinnerColor:     lipgloss.Color("#f5c2e7"),
	}

	tl.themes["Catppuccin Mocha"] = defaultTheme
	setTheme(defaultTheme)

	return tl
}

func testModel(tb testing.TB, mode viewMode, cfg Config) *model {
	tb.Helper()
	prevTheme := currentTheme
	tb.Cleanup(func() {
		setTheme(prevTheme)
	})
	tl := newTestThemeLoader()
	return initialModel(mode, cfg, tl)
}
