package taskflow_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/josuedeavila/taskflow" // Adjust the import path as necessary
)

func TestNewRunner(t *testing.T) {
	runner := taskflow.NewRunner()

	if runner == nil {
		t.Error("Expected runner to be created")
	} else {
		if runner.Tasks != nil || len(runner.Tasks) != 0 {
			t.Errorf("Expected empty tasks slice, got %d tasks", len(runner.Tasks))
		}
	}
}

func TestRunnerAdd(t *testing.T) {
	runner := taskflow.NewRunner()

	task1 := taskflow.NewTask("task1", func(ctx context.Context, input any) (string, error) {
		return "result1", nil
	})
	task2 := taskflow.NewTask("task2", func(ctx context.Context, input any) (string, error) {
		return "result2", nil
	})

	runner.Add(task1)
	if len(runner.Tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(runner.Tasks))
	}

	runner.Add(task2)
	if len(runner.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(runner.Tasks))
	}
}

func TestRunnerAddMultiple(t *testing.T) {
	runner := taskflow.NewRunner()

	task1 := taskflow.NewTask("task1", func(ctx context.Context, input any) (string, error) {
		return "result1", nil
	})
	task2 := taskflow.NewTask("task2", func(ctx context.Context, input any) (string, error) {
		return "result2", nil
	})
	task3 := taskflow.NewTask("task3", func(ctx context.Context, input any) (string, error) {
		return "result3", nil
	})

	runner.Add(task1, task2, task3)
	if len(runner.Tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(runner.Tasks))
	}
}

func TestRunnerRunSuccess(t *testing.T) {
	runner := taskflow.NewRunner()

	var results []string
	var mu sync.Mutex

	task1 := taskflow.NewTask("task1", func(ctx context.Context, input any) (string, error) {
		mu.Lock()
		results = append(results, "task1")
		mu.Unlock()
		return "result1", nil
	})

	task2 := taskflow.NewTask("task2", func(ctx context.Context, input any) (string, error) {
		mu.Lock()
		results = append(results, "task2")
		mu.Unlock()
		return "result2", nil
	})

	runner.Add(task1, task2)

	err := runner.Run(context.Background())

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Check that both tasks were executed
	hasTask1 := false
	hasTask2 := false
	for _, result := range results {
		if result == "task1" {
			hasTask1 = true
		}
		if result == "task2" {
			hasTask2 = true
		}
	}

	if !hasTask1 || !hasTask2 {
		t.Errorf("Expected both tasks to be executed, got results: %v", results)
	}
}

func TestRunnerRunWithError(t *testing.T) {
	runner := taskflow.NewRunner()

	expectedErr := errors.New("task error")

	task1 := taskflow.NewTask("task1", func(ctx context.Context, input any) (string, error) {
		return "result1", nil
	})

	task2 := taskflow.NewTask("task2", func(ctx context.Context, input any) (string, error) {
		return "", expectedErr
	})

	runner.Add(task1, task2)

	err := runner.Run(context.Background())

	if err == nil {
		t.Error("Expected error but got none")
	}
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestRunnerRunWithTimeout(t *testing.T) {
	runner := taskflow.NewRunner()

	task1 := taskflow.NewTask("slow_task", func(ctx context.Context, input any) (string, error) {
		select {
		case <-time.After(200 * time.Millisecond):
			return "completed", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	})

	runner.Add(task1)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := runner.Run(ctx)

	if err == nil {
		t.Error("Expected timeout error but got none")
	}
}

func TestRunnerRunConcurrency(t *testing.T) {
	runner := taskflow.NewRunner()

	const numTasks = 10
	start := time.Now()
	executionTimes := make([]time.Time, numTasks)
	var mu sync.Mutex

	for i := 0; i < numTasks; i++ {
		i := i
		task := taskflow.NewTask("task", func(ctx context.Context, input any) (int, error) {
			time.Sleep(100 * time.Millisecond) // Simulate work
			mu.Lock()
			executionTimes[i] = time.Now()
			mu.Unlock()
			return i, nil
		})
		runner.Add(task)
	}

	err := runner.Run(context.Background())
	totalTime := time.Since(start)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// If tasks run concurrently, total time should be much less than numTasks * 100ms
	maxExpectedTime := 300 * time.Millisecond // Allow some overhead
	if totalTime > maxExpectedTime {
		t.Errorf("Tasks appear to run sequentially. Total time: %v, expected less than %v", totalTime, maxExpectedTime)
	}

	// Check that all tasks started around the same time (within 50ms of each other)
	for i, execTime := range executionTimes {
		if execTime.IsZero() {
			continue
		}
		for j, otherTime := range executionTimes {
			if i != j && !otherTime.IsZero() {
				diff := execTime.Sub(otherTime)
				if diff < 0 {
					diff = -diff
				}
				if diff > 50*time.Millisecond {
					t.Errorf("Tasks %d and %d started too far apart: %v", i, j, diff)
				}
			}
		}
	}
}

func TestRunnerRunEmpty(t *testing.T) {
	runner := taskflow.NewRunner()

	err := runner.Run(context.Background())

	if err != nil {
		t.Errorf("Expected no error for empty runner, got %v", err)
	}
}

func TestRunnerRunWithDependencies(t *testing.T) {
	runner := taskflow.NewRunner()

	var executionOrder []string
	var mu sync.Mutex

	task1 := taskflow.NewTask("task1", func(ctx context.Context, input any) (string, error) {
		time.Sleep(50 * time.Millisecond) // Ensure some execution time
		mu.Lock()
		executionOrder = append(executionOrder, "task1")
		mu.Unlock()
		return "result1", nil
	})

	task2 := taskflow.NewTask("task2", func(ctx context.Context, input string) (string, error) {
		mu.Lock()
		executionOrder = append(executionOrder, "task2")
		mu.Unlock()
		return "result2", nil
	}).After(task1)

	task3 := taskflow.NewTask("task3", func(ctx context.Context, input any) (string, error) {
		mu.Lock()
		executionOrder = append(executionOrder, "task3")
		mu.Unlock()
		return "result3", nil
	})

	runner.Add(task2, task3) // Note: task1 is a dependency of task2

	err := runner.Run(context.Background())

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Verify execution order
	if len(executionOrder) != 3 {
		t.Errorf("Expected 3 executions, got %d: %v", len(executionOrder), executionOrder)
	}

	// task1 should execute before task2 due to dependency
	task1Index := -1
	task2Index := -1
	task3Index := -1

	for i, task := range executionOrder {
		switch task {
		case "task1":
			task1Index = i
		case "task2":
			task2Index = i
		case "task3":
			task3Index = i
		}
	}

	if task1Index == -1 || task2Index == -1 || task3Index == -1 {
		t.Errorf("Not all tasks were executed: %v", executionOrder)
	}

	if task1Index >= task2Index {
		t.Errorf("task1 should execute before task2, got order: %v", executionOrder)
	}

	// task3 should execute independently (no specific order relative to others)
}

func TestRunnerRunStopsOnFirstError(t *testing.T) {
	runner := taskflow.NewRunner()

	expectedErr := errors.New("first error")
	var executionCount int32
	var mu sync.Mutex

	// Create multiple tasks, some will fail
	for i := 0; i < 5; i++ {
		i := i
		task := taskflow.NewTask("task", func(ctx context.Context, input any) (int, error) {
			mu.Lock()
			executionCount++
			mu.Unlock()

			if i == 2 { // Make one task fail
				return 0, expectedErr
			}
			time.Sleep(100 * time.Millisecond) // Simulate work
			return i, nil
		})
		runner.Add(task)
	}

	err := runner.Run(context.Background())

	if err == nil {
		t.Error("Expected error but got none")
	}
	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}

	// The runner should return the first error it encounters
	// Some tasks might still be running when the error is returned
}

// Mock task for testing
type mockTask struct {
	name   string
	result any
	err    error
	called bool
	mu     sync.Mutex
}

func (m *mockTask) Run(ctx context.Context, input any) (any, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.called = true
	return m.result, m.err
}

func (m *mockTask) wasCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.called
}

func TestRunnerRunWithMockTasks(t *testing.T) {
	runner := taskflow.NewRunner()

	mock1 := &mockTask{name: "mock1", result: "result1"}
	mock2 := &mockTask{name: "mock2", result: "result2"}
	mock3 := &mockTask{name: "mock3", err: errors.New("mock error")}

	runner.Add(mock1, mock2, mock3)

	err := runner.Run(context.Background())

	if err == nil {
		t.Error("Expected error but got none")
	}

	// All tasks should have been called (they run concurrently)
	if !mock1.wasCalled() {
		t.Error("mock1 was not called")
	}
	if !mock2.wasCalled() {
		t.Error("mock2 was not called")
	}
	if !mock3.wasCalled() {
		t.Error("mock3 was not called")
	}
}
