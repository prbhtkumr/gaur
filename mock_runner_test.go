package main

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
)

// MockCommandRunner implements CommandRunner for testing.
type MockCommandRunner struct {
	capturedArgs            []string
	capturedCalls           [][]string
	RunFunc                 func(name string, args ...string) ([]byte, error)
	RunContextFunc          func(ctx context.Context, name string, args ...string) ([]byte, error)
	RunWithInputFunc        func(input string, name string, args ...string) ([]byte, error)
	RunWithInputContextFunc func(ctx context.Context, input string, name string, args ...string) ([]byte, error)
	InteractiveFunc         func(onExit func(error) tea.Msg, name string, args ...string) tea.Cmd
	RunWithStderrScanFunc   func(name string, onLine func(string), args ...string) error
}

func (m *MockCommandRunner) recordCall(name string, args ...string) {
	call := append([]string{name}, args...)
	m.capturedArgs = call
	m.capturedCalls = append(m.capturedCalls, call)
}

func (m *MockCommandRunner) CapturedArgs() []string {
	return m.capturedArgs
}

func (m *MockCommandRunner) CapturedCalls() [][]string {
	return m.capturedCalls
}

func (m *MockCommandRunner) Run(name string, args ...string) ([]byte, error) {
	m.recordCall(name, args...)
	if m.RunFunc != nil {
		return m.RunFunc(name, args...)
	}
	return nil, nil
}

func (m *MockCommandRunner) RunContext(ctx context.Context, name string, args ...string) ([]byte, error) {
	if m.RunContextFunc != nil {
		m.recordCall(name, args...)
		return m.RunContextFunc(ctx, name, args...)
	}
	return m.Run(name, args...)
}

func (m *MockCommandRunner) RunWithInput(input string, name string, args ...string) ([]byte, error) {
	m.recordCall(name, args...)
	if m.RunWithInputFunc != nil {
		return m.RunWithInputFunc(input, name, args...)
	}
	return nil, nil
}

func (m *MockCommandRunner) RunWithInputContext(ctx context.Context, input string, name string, args ...string) ([]byte, error) {
	if m.RunWithInputContextFunc != nil {
		m.recordCall(name, args...)
		return m.RunWithInputContextFunc(ctx, input, name, args...)
	}
	return m.RunWithInput(input, name, args...)
}

func (m *MockCommandRunner) Interactive(onExit func(error) tea.Msg, name string, args ...string) tea.Cmd {
	m.recordCall(name, args...)
	if m.InteractiveFunc != nil {
		return m.InteractiveFunc(onExit, name, args...)
	}
	return func() tea.Msg { return nil }
}

func (m *MockCommandRunner) RunWithStderrScan(name string, onLine func(string), args ...string) error {
	m.recordCall(name, args...)
	if m.RunWithStderrScanFunc != nil {
		return m.RunWithStderrScanFunc(name, onLine, args...)
	}
	_, err := m.Run(name, args...)
	return err
}
