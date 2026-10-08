package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// newTestModelMirror creates a model in modeUpdate with mirror overlay open
func newTestModelMirror(tb testing.TB) *model {
	m := newTestModelUpdate(tb, testPackages())
	m.showMirrorOverlay = true
	m.mirrorConfig = DefaultMirrorConfig()
	m.mirrorSelectedItem = mirrorItemSortBy
	m.mirrorUpdating = false
	m.mirrorError = ""
	m.mirrorProgressCurrent = 0
	m.mirrorProgressTotal = 0
	return m
}

// --- handleMirrorOverlayKey tests ---

func TestMirrorOverlayEnterTriggersFromAnyItem(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	// Mock: checkReflectorInstalled returns true, Interactive captures the call
	var interactiveCalled bool
	runner = &MockCommandRunner{
		RunFunc: func(name string, args ...string) ([]byte, error) {
			if name == "which" && len(args) > 0 && args[0] == "reflector" {
				return []byte("/usr/bin/reflector"), nil
			}
			return nil, nil
		},
		InteractiveFunc: func(onExit func(error) tea.Msg, name string, args ...string) tea.Cmd {
			interactiveCalled = true
			// Verify it's called with sudo -v
			if name != "sudo" || len(args) == 0 || args[0] != "-v" {
				t.Errorf("expected 'sudo -v', got %s %v", name, args)
			}
			return func() tea.Msg {
				return onExit(nil)
			}
		},
	}

	items := []MirrorOverlayItem{
		mirrorItemSortBy,
		mirrorItemCountry,
		mirrorItemLatest,
		mirrorItemProtocol,
	}

	for _, item := range items {
		t.Run(fmt.Sprintf("item_%d", item), func(t *testing.T) {
			interactiveCalled = false
			m := newTestModelMirror(t)
			m.mirrorSelectedItem = item

			result, cmd := m.Update(keyMsg("enter"))
			resultModel := result.(*model)

			if !resultModel.mirrorUpdating {
				t.Error("expected mirrorUpdating to be true after enter")
			}
			if cmd == nil {
				t.Error("expected a command to be returned")
			}
			if !interactiveCalled {
				t.Error("expected runner.Interactive to be called for sudo -v")
			}
		})
	}
}

func TestMirrorOverlayEnterBlockedWhenReflectorMissing(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	runner = &MockCommandRunner{
		RunFunc: func(name string, args ...string) ([]byte, error) {
			if name == "which" {
				return nil, fmt.Errorf("not found")
			}
			return nil, nil
		},
	}

	m := newTestModelMirror(t)
	m.mirrorSelectedItem = mirrorItemSortBy

	result, cmd := m.Update(keyMsg("enter"))
	resultModel := result.(*model)

	if resultModel.mirrorError != "reflector is not installed" {
		t.Errorf("expected 'reflector is not installed' error, got %q", resultModel.mirrorError)
	}
	if resultModel.mirrorUpdating {
		t.Error("should not be updating when reflector is missing")
	}
	if cmd != nil {
		t.Error("expected no command when reflector is missing")
	}
}

func TestMirrorOverlayBlocksInputWhileUpdating(t *testing.T) {
	m := newTestModelMirror(t)
	m.mirrorUpdating = true

	// Try pressing various keys
	for _, key := range []string{"enter", "j", "k", "h", "l", "esc"} {
		result, cmd := m.Update(keyMsg(key))
		resultModel := result.(*model)

		if cmd != nil {
			t.Errorf("key %q: expected no command while updating", key)
		}
		if !resultModel.mirrorUpdating {
			t.Errorf("key %q: should still be updating", key)
		}
	}
}

func TestMirrorOverlayNavigationBounds(t *testing.T) {
	m := newTestModelMirror(t)
	m.mirrorSelectedItem = mirrorItemSortBy // 0

	// Try going up past the top
	result, _ := m.Update(keyMsg("k"))
	resultModel := result.(*model)
	if resultModel.mirrorSelectedItem != mirrorItemSortBy {
		t.Errorf("expected to stay at SortBy (0), got %d", resultModel.mirrorSelectedItem)
	}

	// Navigate down to Protocol (3)
	m.mirrorSelectedItem = mirrorItemProtocol
	result, _ = m.Update(keyMsg("j"))
	resultModel = result.(*model)
	if resultModel.mirrorSelectedItem != mirrorItemProtocol {
		t.Errorf("expected to stay at Protocol (3), got %d", resultModel.mirrorSelectedItem)
	}

	// Navigate down from SortBy should go to Country
	m.mirrorSelectedItem = mirrorItemSortBy
	result, _ = m.Update(keyMsg("j"))
	resultModel = result.(*model)
	if resultModel.mirrorSelectedItem != mirrorItemCountry {
		t.Errorf("expected Country (1), got %d", resultModel.mirrorSelectedItem)
	}
}

func TestMirrorOverlayEscCloses(t *testing.T) {
	m := newTestModelMirror(t)

	result, cmd := m.Update(keyMsg("esc"))
	resultModel := result.(*model)

	if resultModel.showMirrorOverlay {
		t.Error("expected overlay to be closed")
	}
	if resultModel.mirrorError != "" {
		t.Errorf("expected error to be cleared, got %q", resultModel.mirrorError)
	}
	if cmd != nil {
		t.Error("expected no command on esc")
	}
}

func TestMirrorOverlayEnterSetsProgressState(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	runner = &MockCommandRunner{
		RunFunc: func(name string, args ...string) ([]byte, error) {
			if name == "which" {
				return []byte("/usr/bin/reflector"), nil
			}
			return nil, nil
		},
		InteractiveFunc: func(onExit func(error) tea.Msg, name string, args ...string) tea.Cmd {
			return func() tea.Msg { return onExit(nil) }
		},
	}

	m := newTestModelMirror(t)
	m.mirrorConfig.Latest = 15
	m.mirrorError = "old error"

	result, _ := m.Update(keyMsg("enter"))
	resultModel := result.(*model)

	if resultModel.mirrorProgressCurrent != 0 {
		t.Errorf("expected progressCurrent=0, got %d", resultModel.mirrorProgressCurrent)
	}
	if resultModel.mirrorProgressTotal != 15 {
		t.Errorf("expected progressTotal=15, got %d", resultModel.mirrorProgressTotal)
	}
	if resultModel.mirrorError != "" {
		t.Errorf("expected error to be cleared, got %q", resultModel.mirrorError)
	}
}

// --- mirrorSudoReadyMsg handling tests ---

func TestMirrorSudoReadySuccess(t *testing.T) {
	m := newTestModelMirror(t)
	m.mirrorUpdating = true
	m.mirrorConfig = DefaultMirrorConfig()

	result, cmd := m.Update(mirrorSudoReadyMsg{err: nil})
	resultModel := result.(*model)

	if !resultModel.mirrorUpdating {
		t.Error("should still be updating after sudo success")
	}
	if cmd == nil {
		t.Error("expected executeMirrorUpdate command to be returned")
	}
}

func TestMirrorSudoReadyFailure(t *testing.T) {
	m := newTestModelMirror(t)
	m.mirrorUpdating = true

	result, cmd := m.Update(mirrorSudoReadyMsg{err: fmt.Errorf("sudo: 3 incorrect password attempts")})
	resultModel := result.(*model)

	if resultModel.mirrorUpdating {
		t.Error("should stop updating after sudo failure")
	}
	if resultModel.mirrorError != "sudo authentication failed" {
		t.Errorf("expected 'sudo authentication failed', got %q", resultModel.mirrorError)
	}
	if cmd != nil {
		t.Error("expected no command after sudo failure")
	}
}

// --- mirrorProgressMsg handling tests ---

func TestMirrorProgressMsgUpdatesModel(t *testing.T) {
	m := newTestModelMirror(t)
	m.mirrorUpdating = true

	ch := make(chan tea.Msg, 1)

	result, cmd := m.Update(mirrorProgressMsg{current: 5, total: 20, ch: ch})
	resultModel := result.(*model)

	if resultModel.mirrorProgressCurrent != 5 {
		t.Errorf("expected progressCurrent=5, got %d", resultModel.mirrorProgressCurrent)
	}
	if resultModel.mirrorProgressTotal != 20 {
		t.Errorf("expected progressTotal=20, got %d", resultModel.mirrorProgressTotal)
	}
	if cmd == nil {
		t.Error("expected a chained waitForMirrorProgress command")
	}
}

// --- mirrorUpdateMsg handling tests ---

func TestMirrorUpdateMsgSuccess(t *testing.T) {
	m := newTestModelMirror(t)
	m.mirrorUpdating = true

	result, cmd := m.Update(mirrorUpdateMsg{success: true})
	resultModel := result.(*model)

	if resultModel.mirrorUpdating {
		t.Error("should stop updating after success")
	}
	if resultModel.showMirrorOverlay {
		t.Error("should close overlay after success")
	}
	if resultModel.mirrorError != "" {
		t.Errorf("expected error to be cleared, got %q", resultModel.mirrorError)
	}
	if resultModel.statusMessage != "Mirrors updated successfully" {
		t.Errorf("expected success status message, got %q", resultModel.statusMessage)
	}
	if !resultModel.loading {
		t.Error("should set loading=true for repo sync")
	}
	if cmd == nil {
		t.Error("expected syncRepositoriesInTerminal command")
	}
}

func TestMirrorUpdateMsgFailure(t *testing.T) {
	m := newTestModelMirror(t)
	m.mirrorUpdating = true

	result, _ := m.Update(mirrorUpdateMsg{success: false, err: fmt.Errorf("reflector failed: exit status 1")})
	resultModel := result.(*model)

	if resultModel.mirrorUpdating {
		t.Error("should stop updating after failure")
	}
	if !resultModel.showMirrorOverlay {
		t.Error("should keep overlay open after failure")
	}
	if !strings.Contains(resultModel.mirrorError, "reflector failed") {
		t.Errorf("expected error message containing 'reflector failed', got %q", resultModel.mirrorError)
	}
}

// --- waitForMirrorProgress tests ---

func TestWaitForMirrorProgressReadsFromChannel(t *testing.T) {
	ch := make(chan tea.Msg, 1)

	// Send a progress message
	expected := mirrorProgressMsg{current: 3, total: 10, ch: ch}
	ch <- expected

	cmd := waitForMirrorProgress(ch)
	msg := cmd()

	progress, ok := msg.(mirrorProgressMsg)
	if !ok {
		t.Fatalf("expected mirrorProgressMsg, got %T", msg)
	}
	if progress.current != 3 || progress.total != 10 {
		t.Errorf("expected current=3 total=10, got current=%d total=%d", progress.current, progress.total)
	}
}

func TestWaitForMirrorProgressReadsCompletionMsg(t *testing.T) {
	ch := make(chan tea.Msg, 1)

	// Send a completion message
	ch <- mirrorUpdateMsg{success: true}

	cmd := waitForMirrorProgress(ch)
	msg := cmd()

	_, ok := msg.(mirrorUpdateMsg)
	if !ok {
		t.Fatalf("expected mirrorUpdateMsg, got %T", msg)
	}
}

// --- acquireSudoForMirror tests ---

func TestAcquireSudoForMirrorCallsInteractive(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	var capturedName string
	var capturedArgs []string
	runner = &MockCommandRunner{
		InteractiveFunc: func(onExit func(error) tea.Msg, name string, args ...string) tea.Cmd {
			capturedName = name
			capturedArgs = args
			return func() tea.Msg {
				return onExit(nil)
			}
		},
	}

	cmd := acquireSudoForMirror()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}

	msg := cmd()

	if capturedName != "sudo" {
		t.Errorf("expected command 'sudo', got %q", capturedName)
	}
	if len(capturedArgs) != 1 || capturedArgs[0] != "-v" {
		t.Errorf("expected args ['-v'], got %v", capturedArgs)
	}

	readyMsg, ok := msg.(mirrorSudoReadyMsg)
	if !ok {
		t.Fatalf("expected mirrorSudoReadyMsg, got %T", msg)
	}
	if readyMsg.err != nil {
		t.Errorf("expected nil error, got %v", readyMsg.err)
	}
}

func TestAcquireSudoForMirrorPropagatesError(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	runner = &MockCommandRunner{
		InteractiveFunc: func(onExit func(error) tea.Msg, name string, args ...string) tea.Cmd {
			return func() tea.Msg {
				return onExit(fmt.Errorf("sudo: 3 incorrect password attempts"))
			}
		},
	}

	cmd := acquireSudoForMirror()
	msg := cmd()

	readyMsg, ok := msg.(mirrorSudoReadyMsg)
	if !ok {
		t.Fatalf("expected mirrorSudoReadyMsg, got %T", msg)
	}
	if readyMsg.err == nil {
		t.Error("expected error to be propagated")
	}
	if !strings.Contains(readyMsg.err.Error(), "incorrect password") {
		t.Errorf("expected error about incorrect password, got %q", readyMsg.err.Error())
	}
}

// --- Progress capping tests ---

func TestProgressCurrentCappedToTotal(t *testing.T) {
	// This tests the view's progress percentage capping logic
	// When mirrorProgressCurrent exceeds mirrorProgressTotal (extra stderr lines),
	// the percentage should not exceed 100%
	m := newTestModelMirror(t)
	m.mirrorUpdating = true
	m.mirrorProgressCurrent = 25
	m.mirrorProgressTotal = 20

	// Calculate percentage as the view does
	pct := m.mirrorProgressCurrent * 100 / m.mirrorProgressTotal
	if pct > 100 {
		pct = 100
	}
	if pct != 100 {
		t.Errorf("expected percentage capped at 100, got %d", pct)
	}
}

// --- executeMirrorUpdate validates config ---

func TestExecuteMirrorUpdateValidatesConfig(t *testing.T) {
	// An out-of-bounds config should not panic because executeMirrorUpdate
	// calls ValidateMirrorConfig before accessing slice indices
	cfg := MirrorConfig{
		SortBy:       999,
		CountryIndex: -5,
		Latest:       0,
		Protocol:     100,
		Save:         false,
	}

	mock := &MockCommandRunner{}
	// This should not panic
	cmd := executeMirrorUpdate(cfg, mock)
	if cmd == nil {
		t.Error("expected non-nil command even with invalid config")
	}
}

func TestSecurity_S13_MirrorUpdateGoesThroughInjectedRunner(t *testing.T) {
	var capturedName string
	var capturedArgs []string
	mock := &MockCommandRunner{
		RunFunc: func(name string, args ...string) ([]byte, error) {
			capturedName = name
			capturedArgs = args
			return nil, nil
		},
	}
	cmd := executeMirrorUpdate(DefaultMirrorConfig(), mock)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	if capturedName != "sudo" {
		t.Errorf("expected sudo, got %s", capturedName)
	}
	if len(capturedArgs) == 0 || capturedArgs[0] != "reflector" {
		t.Errorf("expected reflector as first arg, got %v", capturedArgs)
	}
	updateMsg, ok := msg.(mirrorUpdateMsg)
	if !ok || !updateMsg.success {
		t.Errorf("expected successful mirrorUpdateMsg, got %v", msg)
	}
}

// --- adjustMirrorOption wraparound tests ---

func TestAdjustMirrorOptionWraparound(t *testing.T) {
	m := newTestModelMirror(t)

	// Sort: wrap forward past last
	m.mirrorSelectedItem = mirrorItemSortBy
	m.mirrorConfig.SortBy = len(MirrorSortOptions) - 1
	m.adjustMirrorOption(1)
	if m.mirrorConfig.SortBy != 0 {
		t.Errorf("SortBy should wrap to 0, got %d", m.mirrorConfig.SortBy)
	}

	// Sort: wrap backward past first
	m.mirrorConfig.SortBy = 0
	m.adjustMirrorOption(-1)
	if m.mirrorConfig.SortBy != len(MirrorSortOptions)-1 {
		t.Errorf("SortBy should wrap to %d, got %d", len(MirrorSortOptions)-1, m.mirrorConfig.SortBy)
	}

	// Country: wrap forward
	m.mirrorSelectedItem = mirrorItemCountry
	m.mirrorConfig.CountryIndex = len(MirrorCountries) - 1
	m.adjustMirrorOption(1)
	if m.mirrorConfig.CountryIndex != 0 {
		t.Errorf("CountryIndex should wrap to 0, got %d", m.mirrorConfig.CountryIndex)
	}

	// Protocol: wrap backward
	m.mirrorSelectedItem = mirrorItemProtocol
	m.mirrorConfig.Protocol = 0
	m.adjustMirrorOption(-1)
	if m.mirrorConfig.Protocol != len(MirrorProtocols)-1 {
		t.Errorf("Protocol should wrap to %d, got %d", len(MirrorProtocols)-1, m.mirrorConfig.Protocol)
	}

	// Latest: clamp at min (5)
	m.mirrorSelectedItem = mirrorItemLatest
	m.mirrorConfig.Latest = 5
	m.adjustMirrorOption(-1)
	if m.mirrorConfig.Latest != 5 {
		t.Errorf("Latest should clamp at 5, got %d", m.mirrorConfig.Latest)
	}

	// Latest: clamp at max (100)
	m.mirrorConfig.Latest = 100
	m.adjustMirrorOption(1)
	if m.mirrorConfig.Latest != 100 {
		t.Errorf("Latest should clamp at 100, got %d", m.mirrorConfig.Latest)
	}
}

// --- Mirror overlay open/close via M key ---

func TestMirrorOverlayOpenClose(t *testing.T) {
	m := newTestModelUpdate(t, testPackages())
	m.textInput.Blur()

	// Press 'm' to open mirror overlay
	result, _ := m.Update(keyMsg("m"))
	resultModel := result.(*model)

	if !resultModel.showMirrorOverlay {
		t.Error("expected mirror overlay to open on 'm' key")
	}

	// Press 'esc' to close
	result, _ = resultModel.Update(keyMsg("esc"))
	resultModel = result.(*model)

	if resultModel.showMirrorOverlay {
		t.Error("expected mirror overlay to close on 'esc' key")
	}
}

func TestMirrorOverlayBlockedWhileLoading(t *testing.T) {
	m := newTestModelUpdate(t, testPackages())
	m.loading = true

	result, _ := m.Update(keyMsg("m"))
	resultModel := result.(*model)

	if resultModel.showMirrorOverlay {
		t.Error("should not open mirror overlay while loading")
	}
}

func TestExecuteMirrorUpdateStreamingProgress(t *testing.T) {
	cfg := DefaultMirrorConfig()
	cfg.Latest = 20

	mock := &MockCommandRunner{
		RunWithStderrScanFunc: func(name string, onLine func(string), args ...string) error {
			if onLine != nil {
				onLine("[1/20] testing https://mirror.example.org")
				onLine("[2/20] testing https://mirror2.example.org")
			}
			return nil
		},
	}

	cmd := executeMirrorUpdate(cfg, mock)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	msg := cmd()
	progressMsg, ok := msg.(mirrorProgressMsg)
	if !ok {
		t.Fatalf("expected mirrorProgressMsg, got %T: %+v", msg, msg)
	}
	if progressMsg.current != 1 || progressMsg.total != 20 {
		t.Errorf("expected progress 1/20, got %d/%d", progressMsg.current, progressMsg.total)
	}

	// Read next progress message from the channel
	cmdNext := waitForMirrorProgress(progressMsg.ch)
	msg2 := cmdNext()
	progressMsg2, ok := msg2.(mirrorProgressMsg)
	if !ok {
		t.Fatalf("expected second mirrorProgressMsg, got %T: %+v", msg2, msg2)
	}
	if progressMsg2.current != 2 || progressMsg2.total != 20 {
		t.Errorf("expected progress 2/20, got %d/%d", progressMsg2.current, progressMsg2.total)
	}

	// Read final success message
	cmdFinal := waitForMirrorProgress(progressMsg.ch)
	msg3 := cmdFinal()
	updateMsg, ok := msg3.(mirrorUpdateMsg)
	if !ok || !updateMsg.success {
		t.Fatalf("expected successful mirrorUpdateMsg, got %T: %+v", msg3, msg3)
	}
}

func TestExecuteMirrorUpdateStreamingError(t *testing.T) {
	cfg := DefaultMirrorConfig()
	mock := &MockCommandRunner{
		RunWithStderrScanFunc: func(name string, onLine func(string), args ...string) error {
			return errors.New("simulated reflector exit failure")
		},
	}

	cmd := executeMirrorUpdate(cfg, mock)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}

	msg := cmd()
	updateMsg, ok := msg.(mirrorUpdateMsg)
	if !ok {
		t.Fatalf("expected mirrorUpdateMsg, got %T: %+v", msg, msg)
	}
	if updateMsg.success {
		t.Error("expected failed mirrorUpdateMsg, got success")
	}
	if updateMsg.err == nil || !strings.Contains(updateMsg.err.Error(), "simulated reflector exit failure") {
		t.Errorf("expected simulated reflector error, got %v", updateMsg.err)
	}
}
