package main

import (
	"context"
	"testing"
	"time"
)

func TestRealCommandRunnerContextCancellation(t *testing.T) {
	r := RealCommandRunner{}

	// 1. Immediately cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err := r.RunContext(ctx, "sleep", "10")
	duration := time.Since(start)

	if err == nil {
		t.Fatal("expected error with pre-cancelled context, got nil")
	}
	if duration > 2*time.Second {
		t.Errorf("expected immediate cancellation, but took %v", duration)
	}
}

func TestRealCommandRunnerRunWithInputContextCancellation(t *testing.T) {
	r := RealCommandRunner{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err := r.RunWithInputContext(ctx, "test input", "cat")
	duration := time.Since(start)

	if err == nil {
		t.Fatal("expected error with pre-cancelled context, got nil")
	}
	if duration > 2*time.Second {
		t.Errorf("expected immediate cancellation, but took %v", duration)
	}
}

func TestMockCommandRunnerContextSupport(t *testing.T) {
	// Test RunContext fallback to Run
	mockRunCalled := false
	mock := &MockCommandRunner{
		RunFunc: func(name string, args ...string) ([]byte, error) {
			mockRunCalled = true
			return []byte("run output"), nil
		},
	}

	out, err := mock.RunContext(context.Background(), "test")
	if err != nil || string(out) != "run output" || !mockRunCalled {
		t.Errorf("Mock RunContext should fallback to RunFunc")
	}

	// Test custom RunContextFunc
	mockCustomCalled := false
	mock.RunContextFunc = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		mockCustomCalled = true
		return []byte("custom context"), nil
	}

	out, err = mock.RunContext(context.Background(), "test")
	if err != nil || string(out) != "custom context" || !mockCustomCalled {
		t.Errorf("Mock RunContextFunc was not called")
	}

	// Test RunWithInputContext fallback
	mockInputCalled := false
	mock.RunWithInputFunc = func(input string, name string, args ...string) ([]byte, error) {
		mockInputCalled = true
		return []byte("input output"), nil
	}

	out, err = mock.RunWithInputContext(context.Background(), "in", "test")
	if err != nil || string(out) != "input output" || !mockInputCalled {
		t.Errorf("Mock RunWithInputContext should fallback to RunWithInputFunc")
	}
}

func TestModelContextLifecycle(t *testing.T) {
	m := &model{}
	if m.getContext() == nil {
		t.Fatal("m.getContext() should never return nil even on empty model")
	}

	var nilModel *model
	if nilModel.getContext() == nil {
		t.Fatal("nilModel.getContext() should never return nil")
	}

	// In initialModel, context should be non-nil and active
	cfg := DefaultConfig()
	im := initialModel(modeInstall, cfg, nil)
	if im.ctx == nil || im.cancelFunc == nil {
		t.Fatal("initialModel should set non-nil ctx and cancelFunc")
	}
	if im.getContext().Err() != nil {
		t.Fatalf("model context should not be cancelled initially, got %v", im.getContext().Err())
	}

	// Cancel model context
	im.cancelFunc()
	if im.getContext().Err() != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", im.getContext().Err())
	}
}

func TestSearchAURContextCancellation(t *testing.T) {
	cfg := DefaultConfig()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancelled

	mock := &MockCommandRunner{
		RunContextFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, ctx.Err()
		},
	}

	cmd := searchAURWithContext(ctx, &cfg, "ripgrep", mock)
	msg := cmd()

	searchMsg, ok := msg.(aurSearchMsg)
	if !ok {
		t.Fatalf("expected aurSearchMsg, got %T", msg)
	}
	if searchMsg.err != nil {
		t.Errorf("cancelled search should not produce visible user error, got %v", searchMsg.err)
	}
}

func TestGetInstalledPackagesDefault(t *testing.T) {
	mock := &MockCommandRunner{
		RunFunc: func(name string, args ...string) ([]byte, error) {
			if len(args) > 0 && args[0] == "-Qi" {
				return []byte("Name: test-pkg\nVersion: 1.0\nDescription: Test package\n\n"), nil
			}
			return []byte(""), nil
		},
	}
	cmd := getInstalledPackages(mock)
	msg := cmd()
	instMsg, ok := msg.(installedPackagesMsg)
	if !ok {
		t.Fatalf("expected installedPackagesMsg, got %T", msg)
	}
	if len(instMsg.packages) != 1 || instMsg.packages[0].Name != "test-pkg" {
		t.Errorf("expected 1 package named test-pkg, got %+v", instMsg.packages)
	}
}

func TestParseInstalledPackages(t *testing.T) {
	mock := &MockCommandRunner{
		RunFunc: func(name string, args ...string) ([]byte, error) {
			return []byte(""), nil
		},
	}
	output := "Name: ripgrep\nVersion: 14.1.0\nDescription: Fast search tool\n\n"
	pkgs, err := parseInstalledPackages(output, mock)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pkgs) != 1 || pkgs[0].Name != "ripgrep" {
		t.Errorf("expected ripgrep, got %+v", pkgs)
	}
}

func TestRealCommandRunnerRunWithStderrScan(t *testing.T) {
	r := RealCommandRunner{}
	var lines []string
	err := r.RunWithStderrScan("sh", func(line string) {
		lines = append(lines, line)
	}, "-c", "echo line1 >&2; echo line2 >&2")
	if err != nil {
		t.Fatalf("RunWithStderrScan failed: %v", err)
	}
	if len(lines) != 2 || lines[0] != "line1" || lines[1] != "line2" {
		t.Errorf("Unexpected lines: %v", lines)
	}
}

func TestMockCommandRunnerRunWithStderrScan(t *testing.T) {
	var capturedName string
	mock := &MockCommandRunner{
		RunWithStderrScanFunc: func(name string, onLine func(string), args ...string) error {
			capturedName = name
			if onLine != nil {
				onLine("progress 1")
				onLine("progress 2")
			}
			return nil
		},
	}

	var scanned []string
	err := mock.RunWithStderrScan("sudo", func(line string) {
		scanned = append(scanned, line)
	}, "reflector")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedName != "sudo" {
		t.Errorf("expected sudo, got %s", capturedName)
	}
	if len(scanned) != 2 || scanned[0] != "progress 1" || scanned[1] != "progress 2" {
		t.Errorf("expected scanned progress lines, got %v", scanned)
	}
}

