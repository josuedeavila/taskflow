package taskflow

import (
	"context"
	"sync"
)

// TaskFunc defines the signature for a function that can be executed as a task.
type TaskFunc func(ctx context.Context, input any) (any, error)

// Task represents a unit of work that can be executed.
type Task struct {
	Name    string
	Fn      TaskFunc
	Depends []*Task
	Result  any
	Err     error
	once    sync.Once
}

// NewTask creates a new Task with the given name and function.
func NewTask(name string, fn TaskFunc) *Task {
	return &Task{Name: name, Fn: fn}
}

// After adds dependencies to the task.
// The task will wait for these dependencies to complete before executing.
// It returns the task itself to allow method chaining.
// If the dependencies are already set, it appends to the existing list.
func (t *Task) After(tasks ...*Task) *Task {
	t.Depends = append(t.Depends, tasks...)
	return t
}

// Run executes the task and its dependencies.
func (t *Task) Run(ctx context.Context, input any) (any, error) {
	t.once.Do(func() {
		for _, dep := range t.Depends {
			out, err := dep.Run(ctx, input)
			input = out
			if err != nil {
				t.Err = err
				return
			}
		}

		t.Result, t.Err = t.Fn(ctx, input)
	})

	return t.Result, t.Err
}
