package main

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

//go:embed themes/*.toml
var embeddedThemes embed.FS

type Theme struct {
	Name string

	BorderColor   lipgloss.Color
	SelectedColor lipgloss.Color
	TextColor     lipgloss.Color
	SubtleColor   lipgloss.Color
	TitleColor    lipgloss.Color

	ScrollbarTrack lipgloss.Color
	ScrollbarThumb lipgloss.Color
	SelectionBG    lipgloss.Color
	DimText        lipgloss.Color

	InstallColor   lipgloss.Color
	DashboardColor lipgloss.Color
	RemoveColor    lipgloss.Color
	UpdateColor    lipgloss.Color
	CacheColor     lipgloss.Color

	CoreColor     lipgloss.Color
	ExtraColor    lipgloss.Color
	MultilibColor lipgloss.Color
	AurColor      lipgloss.Color

	SuccessColor   lipgloss.Color
	WarningColor   lipgloss.Color
	ErrorColor     lipgloss.Color
	HighlightColor lipgloss.Color

	DashboardLabel   lipgloss.Color
	DashboardValue   lipgloss.Color
	DashboardWarning lipgloss.Color
	DashboardDesc    lipgloss.Color

	DialogBorder     lipgloss.Color
	ConfirmInstall   lipgloss.Color
	ConfirmRemove    lipgloss.Color
	ConfirmClean     lipgloss.Color
	ConfirmNuke      lipgloss.Color
	ConfirmSelective lipgloss.Color

	ButtonBg       lipgloss.Color
	ButtonFg       lipgloss.Color
	ButtonDangerBg lipgloss.Color
	ProgressTrack  lipgloss.Color
	AccentColor    lipgloss.Color
	SpinnerColor   lipgloss.Color
}

type tomlTheme struct {
	Border   string `toml:"border"`
	Selected string `toml:"selected"`
	Text     string `toml:"text"`
	Subtle   string `toml:"subtle"`
	Title    string `toml:"title"`

	ScrollbarTrack string `toml:"scrollbar_track"`
	ScrollbarThumb string `toml:"scrollbar_thumb"`
	SelectionBG    string `toml:"selection_bg"`
	DimText        string `toml:"dim_text"`

	Install   string `toml:"install"`
	Dashboard string `toml:"dashboard"`
	Remove    string `toml:"remove"`
	Update    string `toml:"update"`
	Cache     string `toml:"cache"`

	Core     string `toml:"core"`
	Extra    string `toml:"extra"`
	Multilib string `toml:"multilib"`
	Aur      string `toml:"aur"`

	Success   string `toml:"success"`
	Warning   string `toml:"warning"`
	Error     string `toml:"error"`
	Highlight string `toml:"highlight"`

	DashboardLabel   string `toml:"dashboard_label"`
	DashboardValue   string `toml:"dashboard_value"`
	DashboardWarning string `toml:"dashboard_warning"`
	DashboardDesc    string `toml:"dashboard_desc"`

	DialogBorder     string `toml:"dialog_border"`
	ConfirmInstall   string `toml:"confirm_install"`
	ConfirmRemove    string `toml:"confirm_remove"`
	ConfirmClean     string `toml:"confirm_clean"`
	ConfirmNuke      string `toml:"confirm_nuke"`
	ConfirmSelective string `toml:"confirm_selective"`

	ButtonBg       string `toml:"button_bg"`
	ButtonFg       string `toml:"button_fg"`
	ButtonDangerBg string `toml:"button_danger_bg"`
	ProgressTrack  string `toml:"progress_track"`
	AccentColor    string `toml:"accent"`
	SpinnerColor   string `toml:"spinner"`
}

type ThemeLoader struct {
	themes        map[string]Theme
	userThemesDir string
}

var globalThemeLoader *ThemeLoader

func InitThemeLoader() (*ThemeLoader, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("could not determine config directory: %w", err)
	}

	tl := &ThemeLoader{
		themes:        make(map[string]Theme),
		userThemesDir: filepath.Join(configDir, "gaur", "themes"),
	}

	if err := tl.loadAll(); err != nil {
		return nil, err
	}

	globalThemeLoader = tl
	return tl, nil
}

func GetThemeLoader() *ThemeLoader {
	return globalThemeLoader
}

func (tl *ThemeLoader) loadAll() error {
	if err := tl.loadEmbedded(); err != nil {
		return fmt.Errorf("failed to load embedded themes: %w", err)
	}

	tl.loadUserThemes()

	return nil
}

func (tl *ThemeLoader) loadEmbedded() error {
	entries, err := embeddedThemes.ReadDir("themes")
	if err != nil {
		return fmt.Errorf("could not read embedded themes: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		data, err := embeddedThemes.ReadFile("themes/" + entry.Name())
		if err != nil {
			LogWarn("THEME", "Could not read embedded theme %s: %v", entry.Name(), err)
			continue
		}

		theme, err := tl.parseTheme(data, entry.Name())
		if err != nil {
			LogWarn("THEME", "Could not parse embedded theme %s: %v", entry.Name(), err)
			continue
		}

		displayName := filenameToDisplayName(entry.Name())
		theme.Name = displayName
		tl.themes[displayName] = theme
		LogDebug("THEME", "Loaded embedded theme: %s", displayName)
	}

	return nil
}

func (tl *ThemeLoader) loadUserThemes() {
	if _, err := os.Stat(tl.userThemesDir); os.IsNotExist(err) {
		LogDebug("THEME", "User themes directory does not exist: %s", tl.userThemesDir)
		return
	}

	entries, err := os.ReadDir(tl.userThemesDir)
	if err != nil {
		LogWarn("THEME", "Could not read user themes directory: %v", err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		root, err := os.OpenRoot(tl.userThemesDir)
		if err != nil {
			LogWarn("THEME", "Could not open user themes directory: %v", err)
			continue
		}
		data, err := func() ([]byte, error) {
			defer root.Close()
			f, err := root.Open(entry.Name())
			if err != nil {
				return nil, err
			}
			defer f.Close()
			return io.ReadAll(f)
		}()
		if err != nil {
			LogWarn("THEME", "Could not read user theme %s: %v", entry.Name(), err)
			continue
		}

		theme, err := tl.parseTheme(data, entry.Name())
		if err != nil {
			LogWarn("THEME", "Could not parse user theme %s: %v", entry.Name(), err)
			continue
		}

		displayName := filenameToDisplayName(entry.Name())
		theme.Name = displayName
		tl.themes[displayName] = theme
		LogDebug("THEME", "Loaded user theme: %s (overrides embedded if exists)", displayName)
	}
}

func (tl *ThemeLoader) parseTheme(data []byte, filename string) (Theme, error) {
	var tt tomlTheme
	if err := toml.Unmarshal(data, &tt); err != nil {
		return Theme{}, fmt.Errorf("TOML parse error: %w", err)
	}

	cleanColor := func(c string) lipgloss.Color {
		return lipgloss.Color(strings.TrimSpace(c))
	}

	theme := Theme{
		BorderColor:      cleanColor(tt.Border),
		SelectedColor:    cleanColor(tt.Selected),
		TextColor:        cleanColor(tt.Text),
		SubtleColor:      cleanColor(tt.Subtle),
		TitleColor:       cleanColor(tt.Title),
		ScrollbarTrack:   cleanColor(tt.ScrollbarTrack),
		ScrollbarThumb:   cleanColor(tt.ScrollbarThumb),
		SelectionBG:      cleanColor(tt.SelectionBG),
		DimText:          cleanColor(tt.DimText),
		InstallColor:     cleanColor(tt.Install),
		DashboardColor:   cleanColor(tt.Dashboard),
		RemoveColor:      cleanColor(tt.Remove),
		UpdateColor:      cleanColor(tt.Update),
		CacheColor:       cleanColor(tt.Cache),
		CoreColor:        cleanColor(tt.Core),
		ExtraColor:       cleanColor(tt.Extra),
		MultilibColor:    cleanColor(tt.Multilib),
		AurColor:         cleanColor(tt.Aur),
		SuccessColor:     cleanColor(tt.Success),
		WarningColor:     cleanColor(tt.Warning),
		ErrorColor:       cleanColor(tt.Error),
		HighlightColor:   cleanColor(tt.Highlight),
		DashboardLabel:   cleanColor(tt.DashboardLabel),
		DashboardValue:   cleanColor(tt.DashboardValue),
		DashboardWarning: cleanColor(tt.DashboardWarning),
		DashboardDesc:    cleanColor(tt.DashboardDesc),
		DialogBorder:     cleanColor(tt.DialogBorder),
		ConfirmInstall:   cleanColor(tt.ConfirmInstall),
		ConfirmRemove:    cleanColor(tt.ConfirmRemove),
		ConfirmClean:     cleanColor(tt.ConfirmClean),
		ConfirmNuke:      cleanColor(tt.ConfirmNuke),
		ConfirmSelective: cleanColor(tt.ConfirmSelective),
		ButtonBg:         cleanColor(tt.ButtonBg),
		ButtonFg:         cleanColor(tt.ButtonFg),
		ButtonDangerBg:   cleanColor(tt.ButtonDangerBg),
		ProgressTrack:    cleanColor(tt.ProgressTrack),
		AccentColor:      cleanColor(tt.AccentColor),
		SpinnerColor:     cleanColor(tt.SpinnerColor),
	}

	theme = applyDefaults(theme, filename)

	return theme, nil
}

func sanitizeColor(color string) string {
	color = strings.TrimSpace(color)
	if color == "" {
		return "#ffffff"
	}
	return color
}

func applyDefaults(theme Theme, filename string) Theme {
	defaults := getFallbackDefaults(filename)

	if string(theme.BorderColor) == "" {
		theme.BorderColor = defaults.BorderColor
	}
	if string(theme.SelectedColor) == "" {
		theme.SelectedColor = defaults.SelectedColor
	}
	if string(theme.TextColor) == "" {
		theme.TextColor = defaults.TextColor
	}
	if string(theme.SubtleColor) == "" {
		theme.SubtleColor = defaults.SubtleColor
	}
	if string(theme.TitleColor) == "" {
		theme.TitleColor = defaults.TitleColor
	}
	if string(theme.ScrollbarTrack) == "" {
		theme.ScrollbarTrack = defaults.ScrollbarTrack
	}
	if string(theme.ScrollbarThumb) == "" {
		theme.ScrollbarThumb = defaults.ScrollbarThumb
	}
	if string(theme.SelectionBG) == "" {
		theme.SelectionBG = defaults.SelectionBG
	}
	if string(theme.DimText) == "" {
		theme.DimText = defaults.DimText
	}
	if string(theme.InstallColor) == "" {
		theme.InstallColor = defaults.InstallColor
	}
	if string(theme.DashboardColor) == "" {
		theme.DashboardColor = defaults.DashboardColor
	}
	if string(theme.RemoveColor) == "" {
		theme.RemoveColor = defaults.RemoveColor
	}
	if string(theme.UpdateColor) == "" {
		theme.UpdateColor = defaults.UpdateColor
	}
	if string(theme.CacheColor) == "" {
		theme.CacheColor = defaults.CacheColor
	}
	if string(theme.CoreColor) == "" {
		theme.CoreColor = defaults.CoreColor
	}
	if string(theme.ExtraColor) == "" {
		theme.ExtraColor = defaults.ExtraColor
	}
	if string(theme.MultilibColor) == "" {
		theme.MultilibColor = defaults.MultilibColor
	}
	if string(theme.AurColor) == "" {
		theme.AurColor = defaults.AurColor
	}
	if string(theme.SuccessColor) == "" {
		theme.SuccessColor = defaults.SuccessColor
	}
	if string(theme.WarningColor) == "" {
		theme.WarningColor = defaults.WarningColor
	}
	if string(theme.ErrorColor) == "" {
		theme.ErrorColor = defaults.ErrorColor
	}
	if string(theme.HighlightColor) == "" {
		theme.HighlightColor = defaults.HighlightColor
	}
	if string(theme.DashboardLabel) == "" {
		theme.DashboardLabel = defaults.DashboardLabel
	}
	if string(theme.DashboardValue) == "" {
		theme.DashboardValue = defaults.DashboardValue
	}
	if string(theme.DashboardWarning) == "" {
		theme.DashboardWarning = defaults.DashboardWarning
	}
	if string(theme.DashboardDesc) == "" {
		theme.DashboardDesc = defaults.DashboardDesc
	}
	if string(theme.DialogBorder) == "" {
		theme.DialogBorder = defaults.DialogBorder
	}
	if string(theme.ConfirmInstall) == "" {
		theme.ConfirmInstall = defaults.ConfirmInstall
	}
	if string(theme.ConfirmRemove) == "" {
		theme.ConfirmRemove = defaults.ConfirmRemove
	}
	if string(theme.ConfirmClean) == "" {
		theme.ConfirmClean = defaults.ConfirmClean
	}
	if string(theme.ConfirmNuke) == "" {
		theme.ConfirmNuke = defaults.ConfirmNuke
	}
	if string(theme.ConfirmSelective) == "" {
		theme.ConfirmSelective = defaults.ConfirmSelective
	}
	if string(theme.ButtonBg) == "" {
		theme.ButtonBg = defaults.ButtonBg
	}
	if string(theme.ButtonFg) == "" {
		theme.ButtonFg = defaults.ButtonFg
	}
	if string(theme.ButtonDangerBg) == "" {
		theme.ButtonDangerBg = defaults.ButtonDangerBg
	}
	if string(theme.ProgressTrack) == "" {
		theme.ProgressTrack = defaults.ProgressTrack
	}
	if string(theme.AccentColor) == "" {
		theme.AccentColor = defaults.AccentColor
	}
	if string(theme.SpinnerColor) == "" {
		theme.SpinnerColor = defaults.SpinnerColor
	}

	return theme

	return theme
}

func getFallbackDefaults(filename string) Theme {
	return Theme{
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
}

func filenameToDisplayName(filename string) string {
	name := strings.TrimSuffix(filename, ".toml")
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	return cases.Title(language.English).String(name)
}

func (tl *ThemeLoader) GetTheme(name string) (Theme, bool) {
	theme, ok := tl.themes[name]
	return theme, ok
}

func (tl *ThemeLoader) GetThemeByConfigName(configName string) (Theme, bool) {
	normalizedName := normalizeThemeName(configName)

	for displayName, theme := range tl.themes {
		if normalizeThemeName(displayName) == normalizedName {
			return theme, true
		}
	}

	return Theme{}, false
}

func normalizeThemeName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	return name
}

func (tl *ThemeLoader) ListThemes() []string {
	var names []string
	for name := range tl.themes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (tl *ThemeLoader) ExportDefaults(destDir string) error {
	if err := os.MkdirAll(destDir, 0750); err != nil {
		return fmt.Errorf("could not create themes directory: %w", err)
	}

	entries, err := embeddedThemes.ReadDir("themes")
	if err != nil {
		return fmt.Errorf("could not read embedded themes: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		data, err := embeddedThemes.ReadFile("themes/" + entry.Name())
		if err != nil {
			return fmt.Errorf("could not read embedded theme %s: %w", entry.Name(), err)
		}

		destPath := filepath.Join(destDir, entry.Name())
		if err := os.WriteFile(destPath, data, 0600); err != nil {
			return fmt.Errorf("could not write theme %s: %w", entry.Name(), err)
		}
	}

	return nil
}

func (tl *ThemeLoader) GetUserThemesDir() string {
	return tl.userThemesDir
}
