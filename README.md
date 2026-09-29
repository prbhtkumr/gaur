<div align="center">

<img src="gaur.png" alt="gaur" width="800" />

# gaur

**A beautiful, interactive TUI for Arch Linux package management**

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) • Powered by [paru](https://github.com/Morganamilo/paru) or [yay](https://github.com/Jguer/yay)

**[Documentation: gaur.prbhtkumr.xyz](https://gaur.prbhtkumr.xyz)**

> ⚠️ **Disclaimer:** This project is mostly vibecoded and continues to be developed through vibecoding.  
> Do report rough edges, and expect an occasional "it works on my machine" moment (trying my best to eliminate those).

</div>

---

## Table of Contents

- [Features](#-features)
- [Requirements](#-requirements)
- [Interface](#-interface)
- [Installation](#-installation)
- [Configuration](#-configuration)
- [Usage](#-usage)
- [How It Works](#-how-it-works)
- [License](#-license)

## ✨ Features

### 📦 Package Management

- **Fuzzy Search** — Lightning-fast fuzzy matching powered by `fzf` with match highlighting
- **Dynamic Helper Support** — Seamlessly switch between `paru` and `yay` via configuration
- **Repository Filtering** — Filter by source with prefixes: `c:` (core), `e:` (extra), `m:` (multilib), `a:` (aur)
- **Batch Operations** — Mark multiple packages with `Tab` and install/remove them all at once
- **Selective Updates** — Carefully manage your system by selecting specific packages to update rather than full updates
- **Interactive Hand-off** — Correctly handles terminal prompts for password entry and conflict resolution
- **Real-time Package Details** — View detailed package information with debounced loading

### 📊 System Dashboard

- **Package Statistics** — Total, explicit, foreign (AUR), and orphan package counts
- **Disk Space Analysis** — Capacity, usage breakdown (packages, cache, other, free) with visual proportional bars
- **Repository Distribution** — Breakdown of packages by repository (core, extra, multilib, AUR)
- **XDG Compliance** — Automatically respects `$XDG_CACHE_HOME` for all cache operations
- **Top Packages** — See your largest installed packages at a glance
- **Top Cache Hogs** — Identifies the largest packages taking up space across both pacman and helper caches
- **Cache Management** — Clean package caches via an interactive menu, choose keep policies, or selectively delete packages from cache
- **Orphan Removal** — Identify and remove orphaned packages

### 🎨 Interface

- **11 Built-in Themes** — Catppuccin, Dracula, Gruvbox, One Dark, Monokai Pro, Rose Pine, Solarized, Tokyonight, and more
- **Custom TOML Themes** — Create your own themes in `$XDG_CONFIG_HOME/gaur/themes/`
- **Theme Export** — Use `--export-themes` to customize default themes
- **Mouse Support** — Full mouse wheel scrolling throughout the interface
- **Mode-specific Theming** — Each mode (Install, Dash, Remove, Update) has its own color scheme
- **In-App Settings Menu** — Press `,` to instantly change themes, borders, and helpers without restarting
- **Live Theme Preview** — See theme changes instantly as you scroll through options
- **Selection Panel** — Dedicated panel for managing marked packages
- **Centered Dialogs** — All confirmation and error boxes are perfectly centered line-by-line
- **Automatic Refresh** — The entire UI refreshes automatically after any system change to ensure data integrity

## 📋 Requirements

- Arch Linux (or Arch-based distribution)
- [paru](https://github.com/Morganamilo/paru) or [yay](https://github.com/Jguer/yay) — AUR helper
- [fzf](https://github.com/junegunn/fzf) — Fuzzy finder (for search)
- [paccache](https://man.archlinux.org/man/paccache.8) — Cache management (from `pacman-contrib`)
- Go 1.21+ (for building from source)

## 🖼️ Interface

```
╭──────────────────────────────────────────────────────────────────────────╭──────────────────╮
│ Repository   : extra                                                     | Selected (2) [*] │
│ Name         : firefox                                                   |  firefox         │
│ Version      : 133.0-1                                                   |  firefoxpwa      |
│ Description  : Fast, Private & Safe Web Browser                          ╰──────────────────│
│ Architecture : x86_64                                                                       │
│ URL          : https://www.mozilla.org/firefox                                              │
│                                                                                             │
├─────────────────────────────────────────────────────────────────────────────────────────────┤
│  extra/firefox-i18n-an 147.0.2-1                                                            │
│  extra/firefox-i18n-af 147.0.2-1                                                            │
│ *extra/firefoxpwa 2.18.0.1                                                                  │
│>*extra/firefox 147.0.2-1 [installed]                                                        │
│                                                                                             │
│Found 610 packages (492 from AUR)                                                            │
│> firefox                                                                                    │
╰─────────────────────────────────────────────────────────────────────────────────────────────╯
      [/] search  [tab] mark  [i]nstall  [d]ash  [r]emove  [u]pdate  [,] settings  [q]uit
```

## 🚀 Installation

### From AUR (Recommended)

```bash
paru -S gaur-bin
# or
yay -S gaur-bin
```

### From Source

```bash
git clone https://github.com/prbhtkumr/gaur.git
cd gaur
go build -o gaur .
sudo mv gaur /usr/local/bin/
```

### Using go install

```bash
go install github.com/prbhtkumr/gaur@latest
```

## ⚙️ Configuration

gaur creates a configuration file at `~/.config/gaur/config.toml` on first run. You can also configure most settings directly in the app using the **Settings Menu** (`,`).

```toml
[startup]
# Start in: "dashboard", "dash", "install", "remove", "update"
default_mode = "install"

[ui]
# Color theme (use --list-themes to see options)
theme = "catppuccin-mocha"
# Border style: "rounded", "normal", "thick", "double"
border_type = "rounded"

[commands]
# AUR helper: "paru" (default) or "yay"
aur_helper = "paru"
# Cache management tool
cache_tool = "paccache"
# Custom flags for install/remove
install_flags = ""
remove_flags = "-Rns"

[advanced]
# Debounce delay for package details (ms)
debounce_ms = 150
# Custom cache directory (optional, must be absolute path)
cache_dir = ""
```

## 📖 Usage

```bash
gaur              # Start with default mode
gaur -i           # Start in install mode
gaur -d           # Start in dashboard mode
gaur --theme dracula  # Use Dracula theme
```

### Keybindings

#### Global

| Key | Alt Key | Action |
|-----|---------|--------|
| `i` | `Alt+2` | Switch to **Install** mode |
| `d` | `Alt+1` | Switch to **Dashboard** mode |
| `r` | `Alt+4` | Switch to **Remove** mode |
| `u` | `Alt+3` | Switch to **Update** mode |
| `,` | | Open **Settings** menu |
| `q` | | Quit |
| `Ctrl+C` | | Force quit |

#### Navigation

| Key | Action |
|-----|--------|
| `/` | Focus search input |
| `↑` / `k` | Move selection up |
| `↓` / `j` | Move selection down |
| `PgUp` | Jump 10 items up |
| `PgDown` | Jump 10 items down |
| `Esc` | Defocus input / Clear selections |

#### Package Operations

| Key | Action |
|-----|--------|
| `Tab` | Mark/unmark package for batch operation |
| `Enter` | Install/remove selected or marked packages |
| `*` | Toggle selection panel focus |

#### Update Mode

| Key | Action |
|-----|--------|
| `Enter` / `y` / `a` | Proceed with a full system update |
| `s` | Switch to **Selective Update** mode |

#### Dashboard

| Key | Action |
|-----|--------|
| `t` | Jump to Remove mode → All packages |
| `e` | Jump to Remove mode → Explicit packages |
| `f` | Jump to Remove mode → Foreign (AUR) packages |
| `o` | Jump to Remove mode → Orphan packages |
| `c` | Open Cache Cleaning menu |
| `R` | Remove all orphan packages |
| `Ctrl+R` | Refresh dashboard data |

#### Confirmation Dialogs

| Key | Action |
|-----|--------|
| `y` / `Enter` | Confirm operation |
| `n` / `Esc` | Cancel operation |
| `↑` / `↓` | Scroll package list |

### Mouse Support

gaur has full mouse wheel support:
- Scroll to navigate package lists
- In split views, scroll left side for list, right side for details
- Works in confirmation dialogs too

### Search Filters

#### Install Mode

Prefix your search with repository filters:

| Prefix | Repository |
|--------|------------|
| `c:` | Core |
| `e:` | Extra |
| `m:` | Multilib |
| `a:` | AUR |

Combine filters: `ae:firefox` searches AUR and Extra for "firefox"

#### Remove Mode

Filter installed packages by type:

| Prefix | Filter |
|--------|--------|
| `t:` | Total (all packages) |
| `e:` / `l:` | Explicitly installed |
| `f:` / `a:` | Foreign (AUR) packages |
| `o:` | Orphan packages |

Combined: `of:google` searches for orphaned AUR packages matching "google".

### Themes

gaur ships with 11 color themes. Use the `--theme` flag or press `,` for in-app settings:

```bash
gaur --theme dracula
gaur --list-themes    # See all options
```

#### Custom Themes

gaur supports custom themes via TOML files. Theme files are stored in `$XDG_CONFIG_HOME/gaur/themes/` (typically `~/.config/gaur/themes/`).

**Export default themes for customization:**

```bash
gaur --export-themes
```

This copies all embedded default themes to your themes directory, allowing you to customize them.

**Create a custom theme:**

1. Create a new TOML file: `~/.config/gaur/themes/my_theme.toml`
2. Add your color definitions (see format below)
3. Select it in settings or via `--theme my-theme`

**Theme file format:**

```toml
# Base colors
border = "#6c7086"
selected = "#cba6f7"
text = "#cdd6f4"
subtle = "#6c7086"
title = "#f9e2af"

# UI elements
scrollbar_track = "#181825"
scrollbar_thumb = "#6c7086"
selection_bg = "#313244"
dim_text = "#6c7086"

# Mode colors
install = "#89b4fa"
dashboard = "#f5c2e7"
remove = "#f38ba8"
update = "#a6e3a1"
cache = "#cba6f7"

# Source colors
core = "#a6e3a1"
extra = "#89b4fa"
multilib = "#fab387"
aur = "#cba6f7"

# Status colors
success = "#a6e3a1"
warning = "#f9e2af"
error = "#f38ba8"
highlight = "#f9e2af"

# Dashboard colors
dashboard_label = "#cdd6f4"
dashboard_value = "#89dceb"
dashboard_warning = "#f38ba8"
dashboard_desc = "#a6adc8"

# Dialog colors
dialog_border = "#cba6f7"
confirm_install = "#89b4fa"
confirm_remove = "#fab387"
confirm_clean = "#a6e3a1"
confirm_nuke = "#f38ba8"
confirm_selective = "#cba6f7"
```

**Theme naming:**

- Filename `my_theme.toml` becomes "My Theme" in the UI
- Underscores and hyphens are converted to spaces and title-cased

## 🔧 How It Works

1. **Package Database** — Loads repository packages from local pacman cache on startup
2. **AUR Search** — Queries AUR via your configured helper (gated: minimum 2 characters, no repeated queries, one request in flight at a time)
3. **Fuzzy Matching** — Uses `fzf --filter` for fast, relevance-ranked fuzzy matching
4. **Interactive Operations** — Hands off to the AUR helper in the terminal for install/remove/update with full interactivity (password prompts, conflict resolution, etc.)
5. **Unified Refresh** — Re-scans the system after any change to update dashboard stats and lists instantly.

## 📄 License

GPLv3 License — See [LICENSE](LICENSE) for details.

---

<div align="center">

**[Report Bug](https://github.com/prbhtkumr/gaur/issues)** · **[Request Feature](https://github.com/prbhtkumr/gaur/issues)**

</div>
