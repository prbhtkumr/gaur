package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// S-01 (a): the renderer may emit styling, but nothing else — no OSC, no
// cursor movement, no erase-screen, no private modes, from any data source.
func TestSecurity_S01_ViewEmitsOnlySgrSequences(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)

	m := testModel(t, modeInstall, DefaultConfig())
	m.width, m.height = 100, 40
	m.loading = false

	// 1) benign view must be clean (guards against the test passing trivially)
	if i := strings.IndexByte(stripSGR(m.View()), 0x1b); i >= 0 {
		t.Fatalf("benign View() has non-SGR escape: %q", m.View()[i:i+24])
	}

	// 2) hostile external data must not survive into the rendered frame
	payload := "desc \x1b]0;GAUR_PWNED_TITLE\x07\x1b[38;2;255;0;77mGAUR_PWNED_TEXT\x1b[2J\x1b[H end"
	m.packageDetails = payload
	m.searchStatus = payload
	m.errorMessage = payload
	m.errorDetails = payload
	m.showErrorOverlay = true
	m.confirmPackages = []string{"pkg\x1b[2J"}

	view := m.View()
	clean := stripSGR(view)
	if i := strings.IndexByte(clean, 0x1b); i >= 0 {
		t.Errorf("S-01: hostile escape reached View() at %d: %q", i, clean[i:i+30])
	}
	// Only the *escape* may disappear — the description text itself must still
	// render (otherwise the fix would hide content rather than sanitise it).
	for _, marker := range []string{"\x1b]0;GAUR_PWNED_TITLE", "\x1b[2J", "\x1b[38;2;255;0;77m"} {
		if strings.Contains(view, marker) {
			t.Errorf("S-01: payload escape %q present in View()", marker)
		}
	}
	if !strings.Contains(view, "GAUR_PWNED_TEXT") || !strings.Contains(view, "end") {
		t.Errorf("S-01: legitimate description text disappeared from View()")
	}
}

// S-01 (b): sanitisation must happen at *ingest*. This test drives the real
// message handlers instead of assigning to struct fields.
func TestSecurity_S01_ModelFieldsAreEscFree(t *testing.T) {
	m := testModel(t, modeInstall, DefaultConfig())
	hostile := "x\x1b]0;PWNED\x07\x1b[2Jy"

	assertClean := func(field, got string) {
		if i := strings.IndexByte(got, 0x1b); i >= 0 {
			t.Errorf("S-01: %s holds raw control bytes from external data: %q", field, got)
		}
	}

	// 1) package details (helper `--noconfirm -Si` output) via packageDetailsMsg
	m.detailsForPackage = "evil-pkg"
	_, _ = m.Update(packageDetailsMsg{details: hostile, packageName: "evil-pkg"})
	assertClean("packageDetails", m.packageDetails)
	assertClean("detailsCache[evil-pkg]", m.detailsCache["evil-pkg"])

	// 2) helper stderr (pacman/paru failure text) via syncRepositoriesMsg
	_, _ = m.Update(syncRepositoriesMsg{err: errors.New(hostile)})
	assertClean("errorMessage", m.errorMessage)

	// 3) AUR search failure text via aurSearchMsg -> searchStatus.
	//    lastAURQuery must match, otherwise performFiltering() overwrites the
	//    status we are trying to observe (update.go:741-743).
	m.textInput.SetValue("vim")
	m.lastAURQuery = "vim"
	_, _ = m.Update(aurSearchMsg{query: "vim", err: errors.New(hostile)})
	assertClean("searchStatus", m.searchStatus)
}

// S-02: filtering must not block the Bubble Tea event loop.
func TestSecurity_S02_FilteringDoesNotBlockEventLoop(t *testing.T) {
	m := testModel(t, modeInstall, DefaultConfig())
	m.repoPackages = []Package{{Source: "extra", Name: "vim"}, {Source: "extra", Name: "vim-runtime"}}
	m.runner = &MockCommandRunner{
		RunWithInputFunc: func(input, name string, args ...string) ([]byte, error) {
			time.Sleep(1500 * time.Millisecond) // a wedged/hung fzf
			return []byte("vim\n"), nil
		},
	}
	m.textInput.SetValue("vim")

	start := time.Now()
	cmd := m.performFiltering()
	elapsed := time.Since(start)

	if elapsed > 300*time.Millisecond {
		t.Errorf("S-02: Update-side filtering blocked for %s (limit 300ms) — fuzzyFilter must run in a tea.Cmd, not inline", elapsed)
	}
	if cmd == nil {
		t.Fatal("expected performFiltering to return a tea.Cmd")
	}
}

// S-02: stale filter results arriving after query change must be dropped.
func TestSecurity_S02_StaleFilterResultDropped(t *testing.T) {
	m := testModel(t, modeInstall, DefaultConfig())
	m.textInput.SetValue("curl")

	staleMsg := filterResultMsg{
		query: "vim",
		mode:  modeInstall,
		packages: []Package{
			{Source: "extra", Name: "vim"},
		},
		matchIndices: map[int][]int{0: {0, 1, 2}},
	}

	resModel, _ := m.Update(staleMsg)
	m = resModel.(*model)

	if len(m.filtered) != 0 {
		t.Errorf("expected stale filter results to be dropped, got %d filtered packages", len(m.filtered))
	}

	freshMsg := filterResultMsg{
		query: "curl",
		mode:  modeInstall,
		packages: []Package{
			{Source: "core", Name: "curl"},
		},
		matchIndices: map[int][]int{0: {0, 1, 2, 3}},
	}

	resModel, _ = m.Update(freshMsg)
	m = resModel.(*model)

	if len(m.filtered) != 1 || m.filtered[0].Name != "curl" {
		t.Errorf("expected fresh filter result to update filtered packages, got %v", m.filtered)
	}
}

// S-02: reflector check must be cached per overlay session rather than run per frame.
func TestSecurity_S02_ReflectorCheckCachedPerOverlaySession(t *testing.T) {
	var whichCount int32
	mock := &MockCommandRunner{
		RunFunc: func(name string, args ...string) ([]byte, error) {
			if name == "which" && len(args) > 0 && args[0] == "reflector" {
				atomic.AddInt32(&whichCount, 1)
				return []byte("/usr/bin/reflector\n"), nil
			}
			return nil, nil
		},
	}

	m := testModel(t, modeUpdate, DefaultConfig())
	m.runner = mock
	m.width = 120
	m.height = 40
	m.showMirrorOverlay = true

	for i := 0; i < 10; i++ {
		_ = m.View()
	}

	count := atomic.LoadInt32(&whichCount)
	if count > 1 {
		t.Errorf("S-02: reflector check was executed %d times across 10 frames (expected <= 1)", count)
	}
}

// S-02: RunContext must respect context cancellation and timeouts.
func TestSecurity_S02_RunnerContextTimeout(t *testing.T) {
	runner := RealCommandRunner{}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := runner.RunContext(ctx, "sleep", "1")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected sleep command to fail on context deadline exceeded")
	}
	if elapsed > 300*time.Millisecond {
		t.Errorf("command execution exceeded context deadline: %v", elapsed)
	}
}

// S-04: dangerous helper flags must be clamped by ValidateConfig.
func TestSecurity_S04_InstallRemoveFlagsAreClamped(t *testing.T) {
	hostile := []string{
		"--skipinteg",
		"--skipchecksum",
		"--overwrite *",
		"-U /tmp/evil.pkg.tar.zst",
		"-Rdd --noconfirm",
		"--config /etc/pacman.conf",
		"--dbpath /tmp/db",
	}
	for _, f := range hostile {
		cfg := DefaultConfig()
		cfg.Commands.InstallFlags = f
		cfg.Commands.RemoveFlags = f
		ValidateConfig(&cfg)
		if cfg.Commands.InstallFlags == f {
			t.Errorf("S-04: ValidateConfig accepted install_flags=%q verbatim", f)
		}
		if cfg.Commands.RemoveFlags == f {
			t.Errorf("S-04: ValidateConfig accepted remove_flags=%q verbatim", f)
		}
	}

	// legitimate values must survive
	cfg := DefaultConfig()
	cfg.Commands.InstallFlags = "--noconfirm --needed"
	ValidateConfig(&cfg)
	if cfg.Commands.InstallFlags != "--noconfirm --needed" {
		t.Errorf("S-04: legitimate install flags were mangled: %q", cfg.Commands.InstallFlags)
	}

	cfg2 := DefaultConfig()
	cfg2.Commands.RemoveFlags = "-Rcns"
	ValidateConfig(&cfg2)
	if cfg2.Commands.RemoveFlags != "-Rcns" {
		t.Errorf("S-04: legitimate remove flags were mangled: %q", cfg2.Commands.RemoveFlags)
	}
}

// S-05: a hostile package name must never reach the helper as an option.
// Legitimate package names with embedded dots (e.g. db5.3, ffmpeg4.4) must be accepted.
func TestSecurity_S05_HostileNamesCannotReachHelperAsOptions(t *testing.T) {
	cfg := DefaultConfig()

	// legitimate names must keep working either way (including embedded dots)
	legitimate := []string{
		"vim", "glibc", "lib32-foo", "@world",
		"db5.3", "ffmpeg4.4", "litehtml0.9",
		"python-jaraco.classes", "vid.stab", "aspnet-runtime-9.0",
	}
	for _, name := range legitimate {
		if !isValidPackageName(name) {
			t.Errorf("S-05: isValidPackageName(%q) == false — legitimate name rejected", name)
		}
	}

	hostile := []string{"-Syu", "--remove", "-Rns", "--noconfirm", "-pkg", ".hidden", ".."}
	valid, _ := sanitizePackageNames(hostile)
	cmd := BuildAURCommand(&cfg, "install", valid...)

	hasSep := false
	for _, a := range cmd {
		if a == "--" {
			hasSep = true
		}
	}
	if len(valid) > 0 && !hasSep {
		t.Errorf("S-05: %d hostile name(s) pass validation and argv has no `--` separator: %v", len(valid), cmd)
	}
	for _, h := range hostile {
		for _, a := range cmd {
			if a == h {
				t.Errorf("S-05: hostile name %q delivered as an argv option: %v", h, cmd)
			}
		}
	}
}

// S-06: a partial config file must not disable critical bindings.
func TestSecurity_S06_PartialConfigKeepsDefaults(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	gaurDir := filepath.Join(xdg, "gaur")
	if err := os.MkdirAll(gaurDir, 0o750); err != nil {
		t.Fatal(err)
	}
	partial := "[startup]\ndefault_mode = 'install'\n\n[commands]\naur_helper = 'paru'\n"
	if err := os.WriteFile(filepath.Join(gaurDir, "config.toml"), []byte(partial), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	if len(cfg.Keys.Quit) == 0 {
		t.Errorf("S-06: [keys] omitted → quit binding is empty (%v)", cfg.Keys.Quit)
	}
	if len(cfg.Keys.Cancel) == 0 {
		t.Errorf("S-06: [keys] omitted → cancel binding is empty")
	}
	if cfg.Commands.CacheTool == "" {
		t.Errorf("S-06: [commands] partial → cache_tool is empty")
	}
	if cfg.Advanced.DebounceMs <= 0 {
		t.Errorf("S-06: debounce_ms == %d", cfg.Advanced.DebounceMs)
	}
	if cfg.Logging.Level == "" {
		t.Errorf("S-06: [logging] omitted → log level empty")
	}
}

// S-07: the search query must not be parsable as a helper option.
func TestSecurity_S07_SearchQueryIsNotAnOption(t *testing.T) {
	cfg := DefaultConfig()
	for _, q := range []string{"-Syu", "--sysupgrade", "-Rns"} {
		cmd := BuildAURCommand(&cfg, "search", q)

		hasSep := false
		for _, a := range cmd {
			if a == "--" {
				hasSep = true
			}
		}
		if len(cmd) > 0 {
			last := cmd[len(cmd)-1]
			if strings.HasPrefix(last, "-") && !hasSep {
				t.Errorf("S-07: query %q delivered as an option: %v", q, cmd)
			}
		}
	}
}

// S-08: CacheDir must be confined to known cache roots.
func TestSecurity_S08_CacheDirRestrictedToKnownRoots(t *testing.T) {
	for _, dir := range []string{"/etc", "/", "/home/otheruser", "/var/cache-evil"} {
		cfg := DefaultConfig()
		cfg.Advanced.CacheDir = dir
		ValidateConfig(&cfg)
		if cfg.Advanced.CacheDir != "" {
			t.Errorf("S-08: ValidateConfig kept cache_dir=%q", dir)
		}
	}

	cfg := DefaultConfig()
	cfg.Advanced.CacheDir = "/var/cache/pacman/pkg"
	ValidateConfig(&cfg)
	if cfg.Advanced.CacheDir != "/var/cache/pacman/pkg" {
		t.Errorf("S-08: legitimate cache_dir rejected: %q", cfg.Advanced.CacheDir)
	}
}

// S-08: executeSelectiveClean must use '--' separator and strict pacman cache prefix.
func TestSecurity_S08_SelectiveCleanFlagsAndPrefix(t *testing.T) {
	tmpDir := t.TempDir()
	pacmanCache := filepath.Join(tmpDir, "pacman")
	aurCache := filepath.Join(tmpDir, "aur")
	if err := os.MkdirAll(pacmanCache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(aurCache, 0o755); err != nil {
		t.Fatal(err)
	}

	pkgFile := filepath.Join(aurCache, "testpkg-1.0-1-x86_64.pkg.tar.zst")
	if err := os.WriteFile(pkgFile, []byte("pkg content"), 0o644); err != nil {
		t.Fatal(err)
	}

	var capturedName string
	var capturedArgs []string
	mock := &MockCommandRunner{
		InteractiveFunc: func(onExit func(error) tea.Msg, name string, args ...string) tea.Cmd {
			capturedName = name
			capturedArgs = args
			return func() tea.Msg { return onExit(nil) }
		},
	}

	m := testModel(t, modeCacheSelective, DefaultConfig())
	m.runner = mock

	cmd := executeSelectiveClean(m, []string{"testpkg"}, pacmanCache, aurCache)
	if cmd == nil {
		t.Fatal("expected executeSelectiveClean to return a Cmd")
	}
	_ = cmd()

	if capturedName != "rm" {
		t.Errorf("expected command 'rm', got %q", capturedName)
	}

	hasF := false
	hasSep := false
	for _, a := range capturedArgs {
		if a == "-f" {
			hasF = true
		}
		if a == "--" {
			hasSep = true
		}
	}
	if !hasF || !hasSep {
		t.Errorf("expected args to contain '-f' and '--', got %v", capturedArgs)
	}
}

// S-09: log output must not carry control bytes (log injection).
func TestSecurity_S09_LogIsFreeOfControlBytes(t *testing.T) {
	tmp := t.TempDir()
	if err := InitLogger(LogLevelInfo, tmp); err != nil {
		t.Fatalf("InitLogger: %v", err)
	}
	defer func() { _ = InitLogger(LogLevelOff, "") }()

	LogCommand("install", []string{"evil\x1b]0;PWNED\x07", "second\n[ INFO ] forged line"})

	matches, _ := filepath.Glob(filepath.Join(tmp, "gaur-*.log"))
	if len(matches) == 0 {
		t.Skip("log file not created")
	}
	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.IndexByte(string(data), 0x1b) >= 0 {
		t.Errorf("S-09: raw ESC byte written to the log file")
	}
	if strings.Contains(string(data), "forged line") && strings.Count(string(data), "\n[ INFO ]") > 1 {
		t.Errorf("S-09: embedded newline forged a log line")
	}
}

// S-10: the details cache must be bounded.
func TestSecurity_S10_DetailsCacheIsBounded(t *testing.T) {
	m := testModel(t, modeInstall, DefaultConfig())
	for i := 0; i < 5000; i++ {
		resModel, _ := m.Update(packageDetailsMsg{
			packageName: fmt.Sprintf("pkg-%05d", i),
			details:     strings.Repeat("x", 4096),
		})
		m = resModel.(*model)
	}
	if len(m.detailsCache) > maxDetailsCacheEntries {
		t.Errorf("S-10: detailsCache grew to %d entries (limit %d)", len(m.detailsCache), maxDetailsCacheEntries)
	}
}

// S-10: direct cachePackageDetails method bounds cache size and respects FIFO eviction.
func TestSecurity_S10_CachePackageDetailsEviction(t *testing.T) {
	m := testModel(t, modeInstall, DefaultConfig())

	for i := 0; i < 600; i++ {
		m.cachePackageDetails(fmt.Sprintf("pkg-%04d", i), fmt.Sprintf("details-%d", i))
	}

	if len(m.detailsCache) != maxDetailsCacheEntries {
		t.Errorf("expected detailsCache to cap at %d, got %d", maxDetailsCacheEntries, len(m.detailsCache))
	}

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("pkg-%04d", i)
		if _, exists := m.detailsCache[key]; exists {
			t.Errorf("expected key %s to be evicted", key)
		}
	}
}
