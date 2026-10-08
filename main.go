package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Version is the application version, optionally injected at build time via -ldflags "-X main.Version=..."
var Version = "dev"

func main() {
	versionFlag := flag.Bool("version", false, "Print version and exit")
	versionFlagShort := flag.Bool("v", false, "Short flag for version")
	themeFlag := flag.String("theme", "", "Color theme (use --list-themes to see options)")
	listThemesFlag := flag.Bool("list-themes", false, "List available themes and exit")
	exportThemesFlag := flag.Bool("export-themes", false, "Export default themes to config directory")
	installFlag := flag.Bool("install", false, "Start in install mode (search and install packages)")
	installFlagShort := flag.Bool("i", false, "Short flag for install mode")
	removeFlag := flag.Bool("remove", false, "Start in remove mode (remove packages)")
	removeFlagShort := flag.Bool("r", false, "Short flag for remove mode")
	updateFlag := flag.Bool("update", false, "Start in update mode (system updates)")
	updateFlagShort := flag.Bool("u", false, "Short flag for update mode")
	dashFlag := flag.Bool("dash", false, "Start in dashboard mode (view system stats)")
	dashFlagShort := flag.Bool("d", false, "Short flag for dashboard mode")
	flag.Parse()

	if *versionFlag || *versionFlagShort {
		fmt.Printf("gaur %s\n", Version)
		return
	}

	themeLoader, err := InitThemeLoader()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing theme loader: %v\n", err)
		os.Exit(1)
	}

	if *exportThemesFlag {
		destDir := themeLoader.GetUserThemesDir()
		if err := themeLoader.ExportDefaults(destDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting themes: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Default themes exported to: %s\n", destDir)
		return
	}

	if *listThemesFlag {
		fmt.Println("Available themes:")
		for _, name := range themeLoader.ListThemes() {
			fmt.Printf("  - %s\n", name)
		}
		return
	}

	cfg, err := LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load config: %v\n", err)
	}

	logLevel := LogLevelFromString(cfg.Logging.Level)
	if err := InitLogger(logLevel, ""); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not initialize logger: %v\n", err)
	}
	defer CloseLogger()

	if err := checkDependencies(); err != nil {
		LogError("STARTUP", "Dependency check failed: %v", err)
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	LogDebug("STARTUP", "All dependencies satisfied")

	var initialMode viewMode
	modeFromConfig := modeInstall
	switch cfg.Startup.DefaultMode {
	case "dashboard", "dash":
		modeFromConfig = modeDashboard
	case "remove":
		modeFromConfig = modeRemove
	case "update":
		modeFromConfig = modeUpdate
	case "install":
		modeFromConfig = modeInstall
	}

	initialMode = modeFromConfig
	switch {
	case *installFlag || *installFlagShort:
		initialMode = modeInstall
		LogInfo("STARTUP", "Starting in install mode (flag override)")
	case *removeFlag || *removeFlagShort:
		initialMode = modeRemove
		LogInfo("STARTUP", "Starting in remove mode (flag override)")
	case *updateFlag || *updateFlagShort:
		initialMode = modeUpdate
		LogInfo("STARTUP", "Starting in update mode (flag override)")
	case *dashFlag || *dashFlagShort:
		initialMode = modeDashboard
		LogInfo("STARTUP", "Starting in dashboard mode (flag override)")
	default:
		LogInfo("STARTUP", "Starting in %s mode (from config)", modeString(initialMode))
	}

	activeTheme := cfg.UI.Theme
	if *themeFlag != "" {
		activeTheme = *themeFlag
		LogInfo("STARTUP", "Theme override from flag: %s", activeTheme)
	}

	if activeTheme != "" {
		if theme, ok := themeLoader.GetThemeByConfigName(activeTheme); ok {
			setTheme(theme)
			LogDebug("STARTUP", "Theme set to: %s", theme.Name)
		} else if *themeFlag != "" {
			fmt.Printf("Unknown theme: %s\nAvailable themes:\n", *themeFlag)
			for _, name := range themeLoader.ListThemes() {
				fmt.Printf("  - %s\n", name)
			}
			os.Exit(1)
		} else {
			if theme, ok := themeLoader.GetThemeByConfigName("catppuccin-mocha"); ok {
				setTheme(theme)
				LogDebug("STARTUP", "Using fallback theme: catppuccin-mocha")
			}
		}
	} else {
		if theme, ok := themeLoader.GetThemeByConfigName("catppuccin-mocha"); ok {
			setTheme(theme)
			LogDebug("STARTUP", "Using default theme: catppuccin-mocha")
		}
	}

	m := initialModel(initialMode, cfg, themeLoader)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion())
	LogInfo("STARTUP", "TUI program starting")
	if _, err := p.Run(); err != nil {
		LogError("STARTUP", "Program error: %v", err)
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
	LogInfo("STARTUP", "Program exited normally")
}
