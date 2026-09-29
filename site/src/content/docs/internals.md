---
title: "internals"
description: "A look under the hood"
---

# internals

Curious how gaur works? Here's the technical breakdown.

## architecture

gaur follows the [Elm Architecture](https://guide.elm-lang.org/architecture/) via [Bubble Tea](https://github.com/charmbracelet/bubbletea). The codebase is modular and readable.

### core components

| File | Purpose |
|------|---------|
| `model.go` | Central state: config, packages, selections, UI state |
| `update.go` | Event handling: keyboard, mouse, messages |
| `view.go` | Main rendering logic and layout |
| `styles.go` | Themes, colors, and Lip Gloss styles |
| `commands.go` | System interaction: pacman, AUR helpers, fzf |
| `config.go` | Configuration loading and validation |
| `settings.go` | In-app settings overlay |
| `feature_dashboard.go` | Dashboard stats and widgets |
| `feature_install.go` | Install mode logic |
| `feature_remove.go` | Remove mode logic |
| `feature_update.go` | Update mode logic |
| `cache_views.go` | Cache management UI |
| `utils.go` | Helpers: fuzzy matching, highlighting, validation |
| `types.go` | Type definitions and constants |

## aur helper integration

gaur doesn't replace your package manager. It wraps it.

### search pipeline

1. **Local query:** Searches the pacman sync database for official packages
2. **AUR query:** Queries AUR through your configured helper, gated rather than debounced: minimum 2 characters, never re-queries the same string, and only one request in flight at a time (keystrokes arriving mid-flight just update the status line, and the query sitting in the box fires once the response lands)
3. **Fuzzy filter:** Results pipe through `fzf --filter` for relevance-ranked matching
4. **Merge:** Official and AUR results are combined and deduplicated

### interactive handoff

Operations requiring user input use `tea.ExecProcess` to hand control to the terminal:

- Password prompts (sudo)
- Conflict resolution
- License acceptance
- Build confirmations (AUR)

This keeps the terminal fully interactive. gaur resumes after the operation completes.

## real-time state sync

After every install, remove, or update:

1. gaur runs a full system scan
2. Dashboard statistics update
3. Package lists refresh
4. Cache sizes recalculate

The UI always reflects reality. No stale data.

## security measures

gaur is careful about what it executes:

### package name validation

Every package name is validated before being passed to commands:

```go
// Only allows: a-z, A-Z, 0-9, @, ., _, +, -
func isValidPackageName(name string) bool
```

This prevents shell injection through malicious package names.

### config validation

Configuration values are validated and sanitized:

- **AUR helper:** Only `paru` or `yay` allowed
- **Cache tool:** Only `paccache` allowed  
- **Cache directory:** Must be an absolute path, no `..` traversal
- **File permissions:** Config files created with `0600` (owner-only)

### command building

Commands are built using structured argument arrays, never string concatenation:

```go
cmd := exec.Command(helper, "-S", "--noconfirm", pkg1, pkg2)
```

No shell interpretation, no injection vectors.

## file locations

| Path | Purpose |
|------|---------|
| `~/.config/gaur/config.toml` | User configuration |
| `~/.config/gaur/gaur.log` | Error logging |
| `/var/cache/pacman/pkg` | Pacman package cache |
| `~/.cache/paru/clone` | Paru build directory |
| `~/.cache/yay` | Yay build directory |

gaur respects `$XDG_CONFIG_HOME` and `$XDG_CACHE_HOME` if set.

## dependencies

gaur shells out to these tools:

| Tool | Package | Purpose |
|------|---------|---------|
| `pacman` | `pacman` | Package database queries |
| `paru` or `yay` | AUR helper | Install, remove, update, AUR search |
| `fzf` | `fzf` | Fuzzy filtering |
| `paccache` | `pacman-contrib` | Cache management |

## performance notes

- **Debouncing:** Package detail fetches are debounced (default 150ms) to reduce system calls
- **Caching:** Package details are cached in memory to avoid repeat lookups
- **Non-blocking AUR:** The AUR query is issued in the same update tick as local filtering and runs asynchronously. Local results render immediately and AUR results merge in when they arrive, so the list never waits on the network
- **Batch operations:** Multiple packages are passed to a single command, not one-by-one
