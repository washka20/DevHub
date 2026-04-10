package testutil

import (
	"context"
	"fmt"
)

// MockCall describes an expected call and its response.
type MockCall struct {
	Output string
	Err    error
}

// MockRunner implements runner.CommandRunner for testing.
type MockRunner struct {
	Calls   []MockCall
	Current int
}

func (m *MockRunner) Run(dir, name string, args ...string) (string, error) {
	if m.Current >= len(m.Calls) {
		return "", fmt.Errorf("unexpected call #%d: %s %v", m.Current, name, args)
	}
	call := m.Calls[m.Current]
	m.Current++
	return call.Output, call.Err
}

func (m *MockRunner) RunContext(ctx context.Context, dir, name string, args ...string) (string, error) {
	return m.Run(dir, name, args...)
}
