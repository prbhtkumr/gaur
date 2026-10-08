package main

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

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

// S-08: CacheDir must be confined to known cache roots.
func TestSecurity_S08_CacheDirRestrictedToKnownRoots(t *testing.T) {
	for _, dir := range []string{"/etc", "/", "/home/otheruser", "/var/cache-evil"} {
		cfg := DefaultConfig()
		cfg.Advanced.CacheDir = dir
		ValidateConfig(&cfg)
		if cfg.Advanced.CacheDir != "" {
			t.Errorf("S-08: ValidateConfig kept cache_dir=%q (only $XDG cache or /var/cache/pacman/pkg allowed)", dir)
		}
	}

	// an accepted value must survive cleaning
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

	// Create a dummy package file in aurCache
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

	// Verify command was rm (not sudo, since it's in tmpDir not /var/cache/pacman/pkg)
	if capturedName != "rm" {
		t.Errorf("expected command 'rm', got %q", capturedName)
	}

	// Verify args contain "-f" and "--" before file paths
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
