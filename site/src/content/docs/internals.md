---
title: "internals"
description: "Architecture, search pipeline, and security model, checked against the source"
---

The technical breakdown: what runs, in what order, and what the code actually does. Every claim here is checked against the source, not against intent.

## architecture

### the elm loop

gaur is a [Bubble Tea](https://github.com/charmbracelet/bubbletea) application following the [Elm Architecture](https://guide.elm-lang.org/architecture/).

- **`model.go`** holds every piece of state: config, package lists, selections, dialogs, scroll offsets
- **`update.go`** is the only place state changes, and only in response to a message
- **`view.go`** is a function of state: the current mode picks a renderer, overlays composite on top

`model.Init()` returns a batch of commands (spinner, package load, installed load, plus one mode-specific load). Nothing blocking runs on the UI thread.

### the command seam

Everything that touches the system goes through one interface in `types.go`:

| Method | Purpose |
|--------|---------|
| `Run(name, args...)` | `exec.Command` with combined output |
| `RunWithInput(stdin, name, args...)` | Same, with stdin. Used to pipe packages into `fzf --filter` |
| `Interactive(onExit, name, args...)` | `tea.ExecProcess`, hands the terminal to the child |

`RealCommandRunner` is the production implementation. The package-level `runner` variable is the seam tests swap for a mock, so they can assert on command construction without running pacman.

### source map

| File | Purpose |
|------|---------|
| `main.go` | Flags, config, logging, dependency check, program startup |
| `model.go` | State struct, initial model, mode-aware list helpers, refresh |
| `update.go` | Message dispatch, key handling, filtering entry points |
| `view.go` | Mode selection, layout, overlays |
| `styles.go` | Lip Gloss style construction |
| `types.go` | `Package`, `Config`, `CommandRunner`, message types, constants |
| `commands.go` | Command builders (`BuildAURCommand`), interactive handoffs, dependency check |
| `config.go` | Defaults, load/save, `ValidateConfig`, keymap construction |
| `settings.go` | In-app settings overlay and persistence |
| `theme.go` | Embedded + user theme loading |
| `logger.go` | Leveled logging with date-suffixed files |
| `feature_install.go` | Repo/AUR loading, combined filtering, AUR search |
| `feature_remove.go` | Installed package parsing and source filters |
| `feature_update.go` | Update check and source attribution |
| `feature_dashboard.go` | Concurrent dashboard data collection and parsing |
| `cache_views.go` | Cache menu and selective clean UI |
| `mirror.go` | Reflector command building, sudo handshake, progress |
| `utils.go` | Validation, sanitizing, fuzzy matching, ANSI-safe truncation |

## startup

`main.go` runs in a fixed order:

1. **Flags first.** `--theme`, `--list-themes`, `--export-themes`, and the mode flags (`-i`, `-r`, `-u`, `-d`) are parsed before anything is loaded, so themes can be listed or exported without touching config.
2. **`InitThemeLoader()`** reads embedded themes from `//go:embed themes/*.toml`, then user themes from the config directory. A user theme with the same name overrides the embedded one.
3. **`LoadConfig()`** reads `~/.config/gaur/config.toml`. On first run the directory is created `0750` and defaults written `0600`. A parse failure logs the error and falls back to defaults instead of exiting.
4. **`ValidateConfig()`** applies the allowlists described below to whatever was parsed.
5. **`InitLogger()`** opens `~/.config/gaur/gaur-YYYY-MM-DD.log` at `0600` inside a `0700` directory.
6. **`checkDependencies()`** requires `pacman`, `fzf`, and `paccache` via `exec.LookPath`. A missing one is fatal. The configured AUR helper is *not* checked here: a bad helper value surfaces later as a failed command, and `ValidateConfig` has already constrained it to `paru` or `yay`.
7. **Mode** comes from `startup.default_mode`, then any mode flag overrides it.
8. **Theme** comes from `--theme`, else `ui.theme`, else `catppuccin-mocha`.
9. **`tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseAllMotion())`** starts the loop.

## loading data

Initial loads are issued together from `Init()`:

| Data | Commands | Source |
|------|----------|--------|
| Repo packages | `pacman -Sl` + `pacman -Qq` | `feature_install.go` |
| Installed packages | `pacman -Qi` | `feature_remove.go` |
| Updates | `pacman -Qm` + `pacman -Sl` + `<helper> -Qu` | `feature_update.go` |
| Dashboard | Concurrent fan-out | `feature_dashboard.go` |

`pacman -Sl` is parsed into `[]Package` while an `installedSet` map is built from `-Qq`, so installed status is one map lookup rather than a call per package.

The dashboard is where concurrency lives in normal use: `getDashboardData` fans out one goroutine per widget under a `sync.WaitGroup`, writes results behind a shared mutex, collects per-widget errors with `addErr`, then waits and returns whatever succeeded alongside the aggregated errors. A widget that fails does not blank the others. The only other goroutine in the codebase is the mirror update runner.

## search pipeline

Every keystroke in the search box calls `performFiltering()` (`update.go`).

### local ranking

The repo list stays in memory for the life of the process, so typing never re-runs `pacman`. `filterAllPackages` (`feature_install.go`) instead:

1. parses any repo filter prefix off the query
2. concatenates `repoPackages`, then `aurPackages`
3. applies the repo filter if one was given
4. ranks the combined list with `fuzzyFilter`

`fuzzyFilter` (`utils.go`) writes `index<TAB>name` lines to `fzf --filter <query> -d '\t' -n2 --tiebreak=begin,length` on stdin and maps the returned indices back to packages. If fzf returns nothing, it falls back to a case-insensitive substring match. That is one `fzf` process per filter pass: cheap enough to run per keystroke, and the reason ranking is fzf's algorithm rather than a reimplementation.

### the aur query

The AUR query is **gated, not debounced**. In install mode a search is dispatched only when all four conditions hold:

- no repo filter is active, or `a:` was explicitly requested
- the query is at least `minSearchQueryLen` (**2**) characters
- the query differs from `lastAURQuery`, the string already sent
- `searchingAUR` is false, so only one request is ever in flight

The command is `<helper> -Ss -a <query>` (`BuildAURCommand`, `search`). The query is sanitized first: everything outside `A-Z a-z 0-9 . _ -` is dropped and spaces become hyphens, so an empty result after sanitizing short-circuits to "no results". An empty output on a non-zero exit is treated as no results rather than a failure.

Responses are matched against the query currently in the box (`searchQuery == msg.query`). A response for a query the user has already replaced is dropped instead of being shown. When a response lands, `performFiltering()` runs again, which is what lets a query typed during the flight fire on the next pass.

Keystrokes that arrive while a search is in flight still filter the local list immediately; they only change the status line until the response comes back.

### filter prefixes

| Mode | Prefix | Meaning |
|------|--------|---------|
| install | `c:` | core repository |
| install | `e:` | extra repository |
| install | `m:` | multilib repository |
| install | `a:` | AUR only |
| remove | `t:` | all packages |
| remove | `e:` or `l:` | explicitly installed |
| remove | `f:` or `a:` | foreign (AUR) packages |
| remove | `o:` | orphaned |

Prefixes combine: `ae:firefox` searches AUR and Extra, `cem:` searches core, extra, and multilib. Without a prefix the query searches everything.

### the details pane

Selecting a row schedules a detail fetch rather than firing one immediately:

1. the cache is checked first: a hit paints immediately and stops, with no timer and no process
2. `pendingDetailsPackage` is set and `debouncePackageDetails` (`commands.go`) returns a `debounceTickMsg` after `advanced.debounce_ms` (default **150**)
3. the tick is discarded unless it still matches `pendingDetailsPackage`, so arrow-key spam never launches fetches
4. `getPackageDetails` validates the name, then runs `<helper> --noconfirm -Si <name>`. It falls back to `-Qi` only for an installed package whose `Source` is still `unknown`, where there is no remote record to read
5. the result is discarded unless it matches `detailsForPackage`, the package currently on screen

The debounce timer is the only one in the codebase. Search does not have one.

## interactive handoff

Operations that need real terminal input go through `runner.Interactive`, which is `tea.ExecProcess`. gaur drops out of the alt screen, the child gets the tty, and an `execCompleteMsg` arrives on exit.

| Operation | Command |
|-----------|---------|
| Install | `<helper> -S [install_flags] <names>` |
| Remove | `<helper> <remove_flags> <names>` (default `-Rns`) |
| Full update | `<helper> -Syu` |
| Update check | `<helper> -Qu` |
| Cache clean | `paccache -r`, `-rk1`, `-ruk0`, or `-rk0` |
| Mirror update | `sudo reflector --latest ... --save /etc/pacman.d/mirrorlist` |

`install_flags` defaults to empty, so installs keep the helper's own prompts: sudo password, conflict resolution, license acceptance, build confirmations. Adding `--noconfirm` is a config change, not the default.

Mirror updates are the one privileged path. `acquireSudoForMirror` runs `sudo -v` interactively through the same handoff to cache credentials, the full reflector command is shown to you in the overlay before it runs, and then `executeMirrorUpdate` starts `sudo reflector ...` in a goroutine with its output scanned for progress. gaur's own process never holds uid 0.

## state refresh

After an operation succeeds, `refreshAll()` (`model.go`) batches four commands: dashboard data, repo packages, installed packages, and an update check. It fires from `handleExecComplete` when an interactive operation exits cleanly, from `actionCompleteMsg` when a non-interactive action succeeds, after a repository sync, and when you change the AUR helper in settings. Failures take the error overlay instead, so nothing is refreshed against a half-finished system.

## security model

### no shell, ever

Every external process is `exec.Command(name, args...)`. Arguments are an array, never a string handed to `sh -c`. There is no concatenation step for an attacker to break out of, so package names, search queries, and config values are data rather than syntax.

### package name validation

`isValidPackageName` (`utils.go`) accepts only `a-z`, `A-Z`, `0-9`, and `@ . _ + -`, and rejects empty strings. `sanitizePackageNames` filters a list through it and reports whether every name survived. The handoff functions (`executeInstallInTerminal`, `executeRemoveInTerminal`, `executeRemoveOrphansInTerminal`, `executeSelectiveUpdateInTerminal`) all check that: if nothing valid remains they return an `execCompleteMsg` carrying the error, so an empty or invalid list never reaches a command line.

Search queries get a separate, looser sanitizer in `searchAUR`: a superset allowed set with spaces mapped to hyphens, since queries are not package names.

### config allowlists

`ValidateConfig` (`config.go`) runs on every load and clamps rather than rejects:

| Field | Rule |
|-------|------|
| `commands.aur_helper` | Lowercased and trimmed; must be `paru` or `yay`, else reset to `paru` |
| `commands.cache_tool` | Trimmed; empty or `paccache`, else reset to `paccache` |
| `advanced.cache_dir` | `filepath.Clean` applied, must be absolute, else cleared |
| `logging.level` | Must be one of `off error warn info debug verbose`, else `info` |

The config path itself is cleaned and checked for absoluteness before it is read or written.

### themes

Themes are compiled into the binary with `//go:embed`, so a shipped build needs no data files. User themes are read from the user themes directory with path traversal checks on both load and export. `sanitizeColor` trims a color and substitutes `#ffffff` when it is empty. `theme_security_test.go` covers traversal on load, traversal on export, malformed and hostile TOML, embedded theme integrity, and user-overrides-embedded ordering.

### file permissions

| Path | Mode |
|------|------|
| `~/.config/gaur/` | `0750` |
| `config.toml` (written by first run or settings) | `0600` |
| `~/.config/gaur/` (log dir) | `0700` |
| `gaur-YYYY-MM-DD.log` | `0600` |

Both the config path and the log directory are rejected if they are not absolute.

### continuous integration

`.github/workflows/go-security.yml` runs these as separate jobs:

| Job | What it does |
|-----|--------------|
| Build & Test | `go test -race -covermode=atomic` with coverage upload |
| Security Tests | `TestCommandInjection`, `TestPrivilegeEscalation`, `TestTUISpoofing`, `TestConfigurationHijacking`, `TestSecurityEdgeCases`, `TestSecurityFixes` |
| Gosec | `gosec` with SARIF upload plus a console pass |
| Static Analysis | `staticcheck ./...` |
| Vulnerability Check | `govulncheck ./...` |
| Dependency Review | Fails on new dependencies licensed `GPL-3.0` or `AGPL-3.0` |
| CodeQL | GitHub's Go analysis |
| License Compliance | `go-licenses check ./...` |

## file locations

| Path | Purpose |
|------|---------|
| `~/.config/gaur/config.toml` | User configuration |
| `~/.config/gaur/gaur-YYYY-MM-DD.log` | Log for the current day |
| `~/.config/gaur/themes/` | User themes (override embedded) |
| `/var/cache/pacman/pkg` | Pacman package cache |
| `~/.cache/paru/clone` | Paru build directory |
| `~/.cache/yay` | Yay build directory |
| `/etc/pacman.d/mirrorlist` | Written by the mirror update |

`$XDG_CONFIG_HOME` and `$XDG_CACHE_HOME` are respected through `os.UserConfigDir` and `os.UserCacheDir`.

## dependencies

| Tool | Package | Required | Purpose |
|------|---------|----------|---------|
| `pacman` | `pacman` | Yes | Package database queries |
| `fzf` | `fzf` | Yes | Fuzzy ranking |
| `paccache` | `pacman-contrib` | Yes | Cache management |
| `paru` or `yay` | AUR helper | Configured | Install, remove, update, AUR search, details |
| `reflector` | `reflector` | Optional | Mirror list updates |

`pacman`, `fzf`, and `paccache` are verified at startup with `exec.LookPath`. The AUR helper is constrained by config rather than probed. `reflector` is checked with `which` while the mirror overlay renders and again when you confirm, so a missing install surfaces as a message instead of a failed background process.

## performance notes

- **Local list is loaded once.** `pacman -Sl` runs at startup and after a refresh, never per keystroke.
- **Ranking is a subprocess per pass.** `fzf --filter` is spawned for each filter, in exchange for not reimplementing its scoring.
- **AUR search is gated, not debounced.** Minimum two characters, no repeat of the same query, one request in flight, stale responses dropped.
- **Detail fetches are debounced** at 150 ms by default and gated twice: once on the way out (`pendingDetailsPackage`) and once on the way back (`detailsForPackage`).
- **Details are cached** by package name for the session and invalidated on every refresh, so revisiting a package paints instantly without spawning a process.
- **Dashboard widgets run concurrently** under a wait group, with errors collected per widget so partial data still renders.
- **Batch operations are one command.** All marked packages are passed to a single `helper -S ...` invocation instead of one process each.
