package main

import (
	"testing"
)

func newTestThemeLoader() *ThemeLoader {
	tl := &ThemeLoader{
		themes: make(map[string]Theme),
	}

	defaultTheme := getFallbackDefaults("catppuccin_mocha.toml")
	defaultTheme.Name = "Catppuccin Mocha"

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
