package taskflow

import (
	"context"
	"fmt"
	"sync"
)

// Executable defines the interface for a task that can be executed.
type Executable interface {
	Run(ctx context.Context, input any) (any, error)
	GetResult() any
}

// TaskFunc defines the signature for a function that can be executed as a task.
type TaskFunc[In any, Out any] func(ctx context.Context, input In) (Out, error)

// Task represents a unit of work that can be executed.
type Task[In any, Out any] struct {
	Name    string
	Fn      TaskFunc[In, Out]
	Depends []Executable // Dependencies that must be completed before this task can run
	Result  Out
	Err     error
	once    sync.Once
	Logger  Logger // Optional logger for task execution
}

// NewTask creates a new Task with the given name and function.
func NewTask[In any, Out any](name string, fn TaskFunc[In, Out]) *Task[In, Out] {
	return &Task[In, Out]{Name: name, Fn: fn, Logger: newDefaultLogger()}
}

// WithLogger sets the logger for the task.
func (t *Task[In, Out]) WithLogger(logger Logger) *Task[In, Out] {
	t.Logger = logger
	return t
}

// After adds dependencies to the task.
func (t *Task[In, Out]) After(tasks ...Executable) *Task[In, Out] {
	t.Depends = append(t.Depends, tasks...)
	return t
}

// Run executes the task and its dependencies.
func (t *Task[In, Out]) Run(ctx context.Context, input any) (any, error) {
	t.once.Do(func() {
		var currInput any = input

		for _, dep := range t.Depends {
			output, err := dep.Run(ctx, currInput)
			if err != nil {
				t.Logger.Log(fmt.Sprintf("task %s dependency failed: %v", t.Name, err))
				t.Err = err
				return
			}
			currInput = output
		}

		var in In
		if currInput != nil {
			typedInput, ok := currInput.(In)
			if !ok {
				err := fmt.Errorf("task: input type mismatch: expected %T, got %T", in, currInput)
				t.Logger.Log(err.Error())
				t.Err = err
				return
			}
			in = typedInput
		} else {
			var zeroIn In
			in = zeroIn
		}

		t.Result, t.Err = t.Fn(ctx, in)
	})

	return t.Result, t.Err
}

// GetResult returns the result of the task execution.
func (t *Task[In, Out]) GetResult() any {
	return t.Result
}
