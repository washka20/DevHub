package executor

import (
	"bufio"
	"context"
	"io"
	"os/exec"
)

// Execute runs a shell command and streams output line by line.
// Returns a channel for output lines and a channel for the final error (or nil).
func Execute(ctx context.Context, dir string, cmd string, args ...string) (chan string, chan error) {
	outputChan := make(chan string, 64)
	errChan := make(chan error, 1)

	go func() {
		defer close(outputChan)
		defer close(errChan)

		c := exec.CommandContext(ctx, cmd, args...)
		c.Dir = dir

		// Merge stdout and stderr
		stdout, err := c.StdoutPipe()
		if err != nil {
			errChan <- err
			return
		}
		c.Stderr = c.Stdout

		if err := c.Start(); err != nil {
			errChan <- err
			return
		}

		scanner := bufio.NewScanner(io.Reader(stdout))
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				_ = c.Process.Kill()
				errChan <- ctx.Err()
				return
			case outputChan <- scanner.Text():
			}
		}

		errChan <- c.Wait()
	}()

	return outputChan, errChan
}
