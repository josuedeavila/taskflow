package taskflow

import (
	"context"
	"sync"
)

// FanOutTask is a task that generates multiple TaskFunc instances,
type FanOutTask struct {
	Generate func(ctx context.Context) ([]TaskFunc, error)
	FanIn    TaskFunc
	Name     string
}

// ToTask converts the FanOutTask into a Task.
// It generates multiple TaskFunc instances and executes them concurrently.
// After all functions are executed, it combines their results using the FanIn function.
// If any function returns an error, it stops execution and returns the first error encountered.
func (f *FanOutTask) ToTask() *Task {
	return NewTask(f.Name, func(ctx context.Context, _ any) (any, error) {
		fns, err := f.Generate(ctx)
		if err != nil {
			return nil, err
		}

		results := make([]any, len(fns))
		var wg sync.WaitGroup
		var mu sync.Mutex
		var firstErr error

		for i, fn := range fns {
			i := i
			wg.Add(1)
			go func() {
				defer wg.Done()
				res, err := fn(ctx, nil)
				mu.Lock()
				defer mu.Unlock()
				if err != nil && firstErr == nil {
					firstErr = err
				}
				results[i] = res
			}()
		}

		wg.Wait()

		if firstErr != nil {
			return nil, firstErr
		}

		// Fan-In
		return f.FanIn(ctx, results)
	})
}
