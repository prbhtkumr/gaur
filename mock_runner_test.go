package main

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

// MockCommandRunner implements CommandRunner for testing.
type MockCommandRunner struct {
	RunFunc                 func(name string, args ...string) ([]byte, error)
	RunContextFunc          func(ctx context.Context, name string, args ...string) ([]byte, error)
	RunWithInputFunc        func(input string, name string, args ...string) ([]byte, error)
	RunWithInputContextFunc func(ctx context.Context, input string, name string, args ...string) ([]byte, error)
	InteractiveFunc         func(onExit func(error) tea.Msg, name string, args ...string) tea.Cmd
	RunWithStderrScanFunc   func(name string, onLine func(string), args ...string) error
}

func (m *MockCommandRunner) Run(name string, args ...string) ([]byte, error) {
	if m.RunFunc != nil {
		return m.RunFunc(name, args...)
	}
	return nil, nil
}

func (m *MockCommandRunner) RunContext(ctx context.Context, name string, args ...string) ([]byte, error) {
	if m.RunContextFunc != nil {
		return m.RunContextFunc(ctx, name, args...)
	}
	return m.Run(name, args...)
}

func (m *MockCommandRunner) RunWithInput(input string, name string, args ...string) ([]byte, error) {
	if m.RunWithInputFunc != nil {
		return m.RunWithInputFunc(input, name, args...)
	}
	return nil, nil
}

func (m *MockCommandRunner) RunWithInputContext(ctx context.Context, input string, name string, args ...string) ([]byte, error) {
	if m.RunWithInputContextFunc != nil {
		return m.RunWithInputContextFunc(ctx, input, name, args...)
	}
	return m.RunWithInput(input, name, args...)
}

func (m *MockCommandRunner) Interactive(onExit func(error) tea.Msg, name string, args ...string) tea.Cmd {
	if m.InteractiveFunc != nil {
		return m.InteractiveFunc(onExit, name, args...)
	}
	return nil
}

func (m *MockCommandRunner) RunWithStderrScan(name string, onLine func(string), args ...string) error {
	if m.RunWithStderrScanFunc != nil {
		return m.RunWithStderrScanFunc(name, onLine, args...)
	}
	_, err := m.Run(name, args...)
	return err
}

