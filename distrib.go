package taskflow

import (
	"context"
	"sync"
)

// FanOutTask is a task that generates multiple TaskFunc instances,
type FanOutTask[In any, Out any] struct {
	Generate func(ctx context.Context) ([]TaskFunc[In, Out], error)
	FanIn    TaskFunc[any, Out] // Function to combine results from multiple TaskFunc instances
	Name     string
}

// ToTask converts the FanOutTask into a Task.
// It generates multiple TaskFunc instances and executes them concurrently.
// After all functions are executed, it combines their results using the FanIn function.
// If any function returns an error, it stops execution and returns the first error encountered.
func (f *FanOutTask[In, Out]) ToTask() *Task[In, Out] {
	return NewTask(f.Name, func(ctx context.Context, _ In) (Out, error) {
		var zeroOut Out // Create a zero value of type T
		var zeroIn In   // Create a zero value of type In
		fns, err := f.Generate(ctx)
		if err != nil {
			return zeroOut, err
		}

		results := make([]In, len(fns))
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
				r, ok := any(res).(In)
				if !ok {
					return
				}
				results[i] = r
			}()
		}

		wg.Wait()

		if firstErr != nil {
			return zeroOut, firstErr
		}

		return f.FanIn(ctx, results)
	})
}
