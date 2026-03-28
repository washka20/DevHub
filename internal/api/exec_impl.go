package api

import (
	"context"

	"devhub/internal/executor"
)

// execMakeImpl calls the executor to run a make target.
func execMakeImpl(ctx context.Context, dir, target string) (chan string, chan error) {
	return executor.Execute(ctx, dir, "make", "-C", dir, target)
}
