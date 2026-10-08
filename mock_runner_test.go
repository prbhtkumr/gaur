package main

import (
	"context"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
)

// MockCommandRunner implements CommandRunner for testing.
type MockCommandRunner struct {
	mu                      sync.RWMutex
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
	m.mu.Lock()
	defer m.mu.Unlock()
	call := append([]string{name}, args...)
	m.capturedArgs = call
	m.capturedCalls = append(m.capturedCalls, call)
}

func (m *MockCommandRunner) CapturedArgs() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.capturedArgs == nil {
		return nil
	}
	cp := make([]string, len(m.capturedArgs))
	copy(cp, m.capturedArgs)
	return cp
}

func (m *MockCommandRunner) CapturedCalls() [][]string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.capturedCalls == nil {
		return nil
	}
	cp := make([][]string, len(m.capturedCalls))
	for i, c := range m.capturedCalls {
		callCopy := make([]string, len(c))
		copy(callCopy, c)
		cp[i] = callCopy
	}
	return cp
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
