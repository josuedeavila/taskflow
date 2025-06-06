package taskflow

import (
	"context"
	"sync"
)

// Runner is a simple task runner that executes tasks concurrently.
type Runner struct {
	Tasks []*Task
}

// NewRunner creates a new Runner instance.
func NewRunner() *Runner {
	return &Runner{}
}

// Add adds one or more tasks to the runner.
func (r *Runner) Add(tasks ...*Task) {
	r.Tasks = append(r.Tasks, tasks...)
}

// Run executes all tasks concurrently, respecting their dependencies.
// It returns the first error encountered during execution, or nil if all tasks succeed.
// If a task has dependencies, it will wait for all dependencies to complete before executing.
// If any task returns an error, it stops execution and returns that error.
func (r *Runner) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	errors := make(chan error, len(r.Tasks))

	runTask := func(t *Task) {
		defer wg.Done()

		for _, dep := range t.Depends {
			_, err := dep.Run(ctx, nil)
			if err != nil {
				errors <- err
				return
			}
		}

		_, err := t.Run(ctx, nil)
		if err != nil {
			errors <- err
		}
	}

	for _, t := range r.Tasks {
		wg.Add(1)
		go runTask(t)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		return err // retorna o primeiro erro
	}

	return nil
}
