package main

import (
	"strings"
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

// setTestRunner temporarily overrides the global runner and restores it via t.Cleanup.
func setTestRunner(tb testing.TB, r CommandRunner) {
	tb.Helper()
	oldRunner := runner
	runner = r
	tb.Cleanup(func() {
		runner = oldRunner
	})
}

// stripSGR removes only legitimate style sequences (ESC[...m)
func stripSGR(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
				j++
			}
			if j < len(s) && s[j] == 'm' {
				i = j
				continue
			}
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
