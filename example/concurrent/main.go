package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/josuedeavila/taskflow"
)

type slogLogger struct {
	*slog.Logger
}

func (l *slogLogger) Log(v ...any) {
	slog.Info("TaskFlow Log", "message", fmt.Sprint(v...))
}

func newSlogLogger() *slogLogger {
	l := slog.Default()
	return &slogLogger{Logger: l}
}

func main() {
	logger := newSlogLogger()

	// 1. Creating some simple task functions
	taskFn1 := func(ctx context.Context, _ any) (string, error) {
		logger.Info("Executing Task 1: Searching for user data...")
		time.Sleep(1 * time.Second) // Simulates work
		select {
		case <-ctx.Done():
			logger.Info("Task 1 cancelled!")
			return "", ctx.Err()
		default:
			logger.Info("Task 1 completed.")
			return "user_data", nil
		}
	}

	taskFn2 := func(ctx context.Context, input any) (string, error) {
		logger.Info("Executing Task 2: Processing product data...")
		time.Sleep(500 * time.Millisecond) // Simulates work
		select {
		case <-ctx.Done():
			logger.Info("Task 2 cancelled!")
			return "", ctx.Err()
		default:
			logger.Info("Task 2 completed.")
			return "product_data", nil
		}
	}

	taskFn3 := func(ctx context.Context, input any) (string, error) {
		logger.Info("Executing Task 3: Generating report (depends on Task 1 and Task 2)...")
		time.Sleep(1500 * time.Millisecond) // Simulates work
		select {
		case <-ctx.Done():
			logger.Info("Task 3 cancelled!")
			return "", ctx.Err()
		default:
			logger.Info(fmt.Sprintf("Task 3 completed with input: %v", input))
			return "final_report", nil
		}
	}

	taskFnError := func(ctx context.Context, input any) (any, error) {
		logger.Info("Executing Error Task: Simulating a failure...")
		time.Sleep(200 * time.Millisecond)
		select {
		case <-ctx.Done():
			logger.Info("Error Task cancelled!")
			return nil, ctx.Err()
		default:
			return nil, fmt.Errorf("intentional error in Error Task")
		}
	}

	// 2. Creating the tasks
	task1 := taskflow.NewTask("FetchUsers", taskFn1).WithLogger(logger) // Adds a default logger
	task2 := taskflow.NewTask("ProcessProducts", taskFn2).WithLogger(logger)
	task3 := taskflow.NewTask("GenerateReport", taskFn3).WithLogger(logger).After(task1, task2) // Task 3 depends on Task 1 and Task 2
	taskError := taskflow.NewTask("SimulateError", taskFnError).WithLogger(logger)

	// 3. Creating a FanOutTask
	fanOutGenerateFunc := func(ctx context.Context, _ []any) ([]taskflow.TaskFunc[any, float64], error) {
		logger.Info("FanOutTask: Generating fan-out functions...")
		fns := []taskflow.TaskFunc[any, float64]{
			func(ctx context.Context, input any) (float64, error) {
				logger.Info("FanOut Sub-Task A: Calculating metric X...")
				time.Sleep(300 * time.Millisecond)
				return 10.5, nil
			},
			func(ctx context.Context, input any) (float64, error) {
				logger.Info("FanOut Sub-Task B: Calculating metric Y...")
				time.Sleep(700 * time.Millisecond)
				return 20.0, nil
			},
			func(ctx context.Context, input any) (float64, error) {
				logger.Info("FanOut Sub-Task C: Calculating metric Z...")
				time.Sleep(400 * time.Millisecond)
				return 5.2, nil
			},
		}
		return fns, nil
	}

	fanInFunc := func(ctx context.Context, results []float64) (float64, error) {
		logger.Info("FanOutTask: Consolidating results...")
		sum := 0.0
		for _, r := range results {
			sum += r
		}
		logger.Info(fmt.Sprintf("FanOutTask: Sum of results: %.2f", sum))
		return sum, nil
	}

	fanOutTask := &taskflow.FanOutTask[any, float64]{
		Name:     "CalculateMetrics",
		Generate: fanOutGenerateFunc,
		FanIn:    fanInFunc,
	}

	// Converts the FanOutTask to a normal Task
	fanOutConvertedTask := fanOutTask.ToTask()

	// 4. Creating the Runner and adding the tasks
	runner := taskflow.NewRunner()
	runner.Add(task1, task2, task3, taskError, fanOutConvertedTask)

	// 5. Running the tasks with a context that can be cancelled
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Sets a timeout for the runner
	defer cancel()

	logger.Info("Running the Runner...")
	err := runner.Run(ctx)

	if err != nil {
		logger.Info(fmt.Sprintf("Runner completed with error: %v", err))
	} else {
		logger.Info("Runner completed successfully!")
	}

	// 6. Checking the results and states of the tasks
	logger.Info("Checking task results:")
	logger.Info(fmt.Sprintf("Task 'FetchUsers' - Result: %v, Error: %v", task1.Result, task1.Err))
	logger.Info(fmt.Sprintf("Task 'ProcessProducts' - Result: %v, Error: %v", task2.Result, task2.Err))
	logger.Info(fmt.Sprintf("Task 'GenerateReport' - Result: %v, Error: %v", task3.Result, task3.Err))
	logger.Info(fmt.Sprintf("Task 'SimulateError' - Result: %v, Error: %v", taskError.Result, taskError.Err))
	logger.Info(fmt.Sprintf("Task 'CalculateMetrics' - Result: %v, Error: %v", fanOutConvertedTask.Result, fanOutConvertedTask.Err))

	// Example of how you can see the state of a task (after execution)
	// This would require exposing the 'state' field or a method to get it in the Task struct.
	// For now, the `logger.Log` inside the states already shows the transition.
}
