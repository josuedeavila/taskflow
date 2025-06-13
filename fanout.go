package taskflow

import (
	"context"
	"sync"
)

// FanOutTask is a task that generates multiple TaskFunc instances,
type FanOutTask[In any, Out any] struct {
	Generate func(ctx context.Context, input []In) ([]TaskFunc[In, Out], error)
	FanIn    TaskFunc[[]Out, Out] // Function to combine results from multiple TaskFunc instances
	Name     string
}

// ToTask converts the FanOutTask into a Task.
// It generates multiple TaskFunc instances and executes them concurrently.
// After all functions are executed, it combines their results using the FanIn function.
// If any function returns an error, it stops execution and returns the first error encountered.
func (f *FanOutTask[In, Out]) ToTask() *Task[[]In, Out] {
	return NewTask(f.Name, func(ctx context.Context, input []In) (Out, error) {
		var zeroOut Out
		var zeroIn In 
		fns, err := f.Generate(ctx, input)
		if err != nil {
			return zeroOut, err
		}

		results := make([]Out, len(fns))
		var wg sync.WaitGroup
		var mu sync.Mutex
		var firstErr error

		for i, fn := range fns {
			i := i
			wg.Add(1)
			go func() {
				defer wg.Done()
				res, err := fn(ctx, zeroIn)
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
			return zeroOut, firstErr
		}

		return f.FanIn(ctx, results)
	})
}
