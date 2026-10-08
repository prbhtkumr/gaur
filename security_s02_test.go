package main

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TestSecurity_S02_FilteringDoesNotBlockEventLoop tests that fuzzy filtering does not
// block the Bubble Tea event loop or UI thread (S-02).
func TestSecurity_S02_FilteringDoesNotBlockEventLoop(t *testing.T) {
	m := testModel(t, modeInstall, DefaultConfig())
	m.repoPackages = []Package{
		{Source: "extra", Name: "vim"},
		{Source: "extra", Name: "vim-runtime"},
	}

	m.runner = &MockCommandRunner{
		RunWithInputFunc: func(input, name string, args ...string) ([]byte, error) {
			time.Sleep(1500 * time.Millisecond) // simulate a wedged / slow fzf
			return []byte("0\tvim\n"), nil
		},
	}
	m.textInput.Focus()

	start := time.Now()
	resModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	elapsed := time.Since(start)

	if elapsed > 300*time.Millisecond {
		t.Fatalf("S-02: Update-side filtering blocked for %s (limit 300ms) — fuzzyFilter must run in a tea.Cmd, not inline", elapsed)
	}

	_ = resModel
	if cmd == nil {
		t.Fatal("expected Update to return a tea.Cmd for async filtering")
	}
}

// TestSecurity_S02_StaleFilterResultDropped tests that stale filter results arriving
// after the query or mode changed are discarded.
func TestSecurity_S02_StaleFilterResultDropped(t *testing.T) {
	m := testModel(t, modeInstall, DefaultConfig())
	m.textInput.SetValue("curl")

	// Message arrived for older query "vim"
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

	// Now send message for current query "curl"
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

// TestSecurity_S02_ReflectorCheckCachedPerOverlaySession tests that view rendering does not
// fork `which reflector` on every frame (S-02).
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

	// Render view 10 consecutive frames
	for i := 0; i < 10; i++ {
		_ = m.View()
	}

	count := atomic.LoadInt32(&whichCount)
	if count > 1 {
		t.Errorf("S-02: reflector check was executed %d times across 10 frames (expected <= 1, cached per overlay session)", count)
	}
}

// TestSecurity_S02_RunnerContextTimeout tests that RunContext respects context deadlines.
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
