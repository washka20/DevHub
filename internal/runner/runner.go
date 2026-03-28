package runner

import (
	"bytes"
	"context"
	"os/exec"
)

// CommandRunner abstracts shell command execution for testability.
type CommandRunner interface {
	Run(dir, name string, args ...string) (string, error)
	RunContext(ctx context.Context, dir, name string, args ...string) (string, error)
}

// ExecRunner implements CommandRunner using os/exec.
type ExecRunner struct{}

func (ExecRunner) Run(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (ExecRunner) RunContext(ctx context.Context, dir, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return stderr.String() + stdout.String(), err
	}
	return stdout.String(), nil
}
