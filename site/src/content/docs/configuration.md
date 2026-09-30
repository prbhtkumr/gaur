---
title: "configuration"
description: "Customize gaur to fit your workflow"
---

gaur works great out of the box, but you can tweak everything. Change settings on the fly or edit the config file directly.

## in-app settings

Press `,` anywhere in gaur to open the settings overlay. From there you can adjust:

- **AUR Helper:** Toggle between `paru` and `yay`
- **Theme:** Preview and apply color schemes instantly
- **Default View:** Choose what gaur opens to (`dashboard`, `install`, `update`, `remove`)
- **Border Style:** Pick your aesthetic (`rounded`, `normal`, `thick`, `double`)
- **Log Level:** Control logging verbosity (`off`, `error`, `warn`, `info`, `debug`, `verbose`)

Changes apply immediately and are saved automatically. No restart needed.

## config file

On first run, gaur creates a config at:

```
~/.config/gaur/config.toml
```

The format is TOML. Here's a complete reference:

### startup options

Control what gaur shows when it launches:

```toml
[startup]
# Which mode to open: "dashboard", "dash", "install", "remove", "update"
default_mode = "install"
```

### ui customization

Visual appearance settings:

```toml
[ui]
# Color theme (see --list-themes for options)
theme = "catppuccin-mocha"

# Border style: "rounded", "normal", "thick", "double"
border_type = "rounded"
```

### aur helper & tools

Set your preferred AUR helper and cache cleaner:

```toml
[commands]
# "paru" (default) or "yay"
aur_helper = "paru"

# cache cleaning tool (paccache by default)
cache_tool = "paccache"

# custom flags for install operations
install_flags = ""

# custom flags for remove operations (default: remove with deps and configs)
remove_flags = "-Rns"
```

### advanced options

Fine-tune performance and paths:

```toml
[advanced]
# debounce delay for package details fetch (ms)
debounce_ms = 150

# custom cache directory (must be absolute path, leave empty for default)
cache_dir = ""
```

### logging

Control diagnostic logging:

```toml
[logging]
# Log level: "off", "error", "warn", "info", "debug", "verbose"
level = "info"
```

Log files are stored at `~/.config/gaur/gaur-YYYY-MM-DD.log` with daily rotation.

**Log Levels:**
- `off` - No logging
- `error` - Only errors
- `warn` - Warnings and errors
- `info` - General operations (default)
- `debug` - Detailed diagnostics
- `verbose` - Everything including key presses

## custom keybindings

Remap any key to fit your muscle memory:

```toml
[keys]
quit = ["q", "ctrl+c"]
install_mode = ["i", "alt+2"]
remove_mode = ["r", "alt+4"]
update_mode = ["u", "alt+3"]
dashboard_mode = ["d", "alt+1"]
search = "/"
mark = "tab"
selective = "s"
settings = ","
confirm = "enter"
cancel = "esc"
```

## themes

gaur bundles 11 popular themes. Switch via settings, config file, or command line.

**Available themes:**
- Catppuccin Frappe
- Catppuccin Macchiato
- Catppuccin Mocha (default)
- Dracula
- Gruvbox Dark
- Monokai Pro
- One Dark
- Rose Pine
- Solarized Dark
- Tokyonight Night
- Tokyonight Storm

### command line

Launch with a specific theme:

```bash
gaur --theme dracula
```

### list all themes

See what's available:

```bash
gaur --list-themes
```

Preview them live with the theme picker in this site's header — it repaints the page
from the same palettes.

## file locations

gaur follows XDG conventions:

| Path | Purpose |
|------|---------|
| `~/.config/gaur/config.toml` | Configuration file |
| `~/.config/gaur/gaur-YYYY-MM-DD.log` | Daily log file |
| `/var/cache/pacman/pkg` | Pacman cache (system) |
| `~/.cache/paru/clone` | Paru build cache |
| `~/.cache/yay` | Yay build cache |
