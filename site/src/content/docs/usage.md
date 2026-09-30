---
title: "usage"
description: "Master the keyboard shortcuts and workflows"
---

gaur is built for speed. Everything happens through the keyboard, with optional mouse support. Learn these shortcuts and you'll fly through package management.

## command line options

Launch gaur with flags to jump straight into action:

```bash
gaur                     # Start with default mode (from config)
gaur -i                  # Start in install mode
gaur -r                  # Start in remove mode
gaur -u                  # Start in update mode
gaur -d                  # Start in dashboard mode
gaur --theme dracula     # Use a specific theme
gaur --list-themes       # List all available themes
```

| Flag | Long Form | Description |
|------|-----------|-------------|
| `-i` | `--install` | Start in install mode |
| `-r` | `--remove` | Start in remove mode |
| `-u` | `--update` | Start in update mode |
| `-d` | `--dash` | Start in dashboard mode |
| | `--theme <name>` | Set color theme |
| | `--list-themes` | List themes and exit |

## mode switching

Jump between views instantly:

| Key | Alt Key | Action |
|-----|---------|--------|
| `i` | `Alt+2` | Install mode |
| `r` | `Alt+4` | Remove mode |
| `u` | `Alt+3` | Update mode |
| `d` | `Alt+1` | Dashboard mode |
| `,` | | Open settings |
| `q` | | Quit |
| `Ctrl+C` | | Force quit |

## getting around

Navigate lists and search:

| Key | Action |
|-----|--------|
| `/` | Focus search box |
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `PgUp` | Jump 10 items up |
| `PgDown` | Jump 10 items down |
| `Esc` | Clear focus / deselect all |

## mouse support

gaur has full mouse support enabled by default:

- **Scroll wheel** navigates package lists
- **In Install/Remove modes:** Scroll on the left for the list, scroll on the right for package details
- **In Update mode:** Scroll anywhere to navigate the update list
- **In confirmation dialogs:** Scroll to see more packages

No configuration needed—just scroll.

## working with packages

Install, remove, or mark packages for batch operations:

| Key | Action |
|-----|--------|
| `Tab` | Mark/unmark for batch processing |
| `Enter` | Execute on selected/marked |
| `*` | Toggle selection panel |

### batch operations

1. Press `Tab` on packages you want to operate on
2. Watch them appear in the selection panel (top-right)
3. Press `*` to focus the panel and manage selections
4. Press `Enter` to confirm the batch operation

## updating

When updates are available, you have two options:

| Key | Action |
|-----|--------|
| `Enter` / `y` / `a` | Full system upgrade |
| `s` | Switch to selective mode |
| `m` | Open mirror configuration |

**Heads up:** Selective updates can break dependencies. Arch recommends full system upgrades. Proceed with caution.

## mirror management

Press `m` from the update view to configure package mirrors using [reflector](https://wiki.archlinux.org/title/Reflector).

### mirror configuration

| Option | Description |
|--------|-------------|
| **Sort by** | Rate (speed), Age, Score, Country, or Delay |
| **Country** | Filter by country or use Worldwide |
| **Latest** | Number of recently synced mirrors (5-100) |
| **Protocol** | HTTPS, HTTP, or Both |

### mirror overlay controls

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate options |
| `h` / `l` | Change selected value |
| `Enter` | Execute mirror update |
| `Esc` | Close without changes |

**Requires:** The `reflector` package must be installed (`pacman -S reflector`).

Mirror updates write to `/etc/pacman.d/mirrorlist` and require sudo.

## dashboard shortcuts

The dashboard gives you quick access to different package categories:

| Key | Action |
|-----|--------|
| `t` | Remove mode → all packages |
| `e` | Remove mode → explicit only |
| `f` | Remove mode → AUR/foreign |
| `o` | Remove mode → orphans |
| `c` | Cache cleaning menu |
| `R` | Remove all orphan packages |
| `Ctrl+R` | Refresh dashboard data |

## cache management

Press `c` from the dashboard to open the cache cleaning menu:

| Option | Description |
|--------|-------------|
| **Safe Clean (Keep 3)** | Keep installed version + 2 fallbacks |
| **Aggressive (Keep 1)** | Keep only the installed version |
| **Orphaned Cache** | Remove caches for uninstalled packages |
| **Nuke Everything** | Empty the entire cache |
| **Selective Clean** | Choose specific packages to remove |

gaur shows size estimates for each option before you commit.

## confirmation prompts

When gaur asks "are you sure?":

| Key | Action |
|-----|--------|
| `y` / `Enter` | Confirm |
| `n` / `Esc` | Cancel |
| `↑` / `↓` / `j` / `k` | Scroll the list |
| `PgUp` / `PgDown` | Jump to start/end |

## search filters

Narrow down results with prefixes.

### install mode

Filter by repository:

| Prefix | Repo |
|--------|------|
| `c:` | Core |
| `e:` | Extra |
| `m:` | Multilib |
| `a:` | AUR |

**Example:** `ae:firefox` searches both AUR and Extra for "firefox".

### remove mode

Filter installed packages by category:

| Prefix | Shows |
|--------|-------|
| `t:` | All installed packages |
| `e:` / `l:` | Explicitly installed |
| `f:` / `a:` | AUR/foreign packages |
| `o:` | Orphans |

**Example:** `of:google` finds orphaned AUR packages matching "google".

Combine filters: `ef:` shows explicitly installed foreign packages.

## color legend

Packages are color-coded by repository:

| Color | Source |
|-------|--------|
| 🟢 Green | core |
| 🔵 Blue | extra |
| 🟠 Orange | multilib |
| 🟣 Magenta | AUR |
| ⚪ Grey | Other |
