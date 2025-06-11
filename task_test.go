package taskflow_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/josuedeavila/taskflow" // Adjust the import path as necessary
)

func TestNewTask(t *testing.T) {
	fn := func(ctx context.Context, input string) (int, error) {
		return len(input), nil
	}

	task := taskflow.NewTask("test", fn)

	if task.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", task.Name)
	}

	if task.Fn == nil {
		t.Error("Expected function to be set")
	}

	if len(task.Depends) != 0 {
		t.Errorf("Expected no dependencies, got %d", len(task.Depends))
	}
}

func TestTaskRun(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected int
		wantErr  bool
	}{
		{
			name:     "successful execution with string input",
			input:    "hello",
			expected: 5,
			wantErr:  false,
		},
		{
			name:     "successful execution with nil input",
			input:    nil,
			expected: 0,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := taskflow.NewTask("test", func(ctx context.Context, input string) (int, error) {
				return len(input), nil
			})

			result, err := task.Run(context.Background(), tt.input)

			if tt.wantErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.wantErr {
				if task.Result != tt.expected {
					t.Errorf("Expected result %d, got %d", tt.expected, task.Result)
				}
				if result != tt.expected {
					t.Errorf("Expected returned result %d, got %d", tt.expected, result)
				}
			}
		})
	}
}

func TestTaskRunWithError(t *testing.T) {
	expectedErr := errors.New("test error")
	task := taskflow.NewTask("error_task", func(ctx context.Context, input string) (int, error) {
		return 0, expectedErr
	})

	result, err := task.Run(context.Background(), "test")

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
	if task.Err != expectedErr {
		t.Errorf("Expected task.Err %v, got %v", expectedErr, task.Err)
	}
	if result != 0 {
		t.Errorf("Expected result 0, got %v", result)
	}
}

func TestTaskRunWithWrongInputType(t *testing.T) {
	task := taskflow.NewTask("type_task", func(ctx context.Context, input string) (int, error) {
		return len(input), nil
	})

	_, err := task.Run(context.Background(), 123) // passing int instead of string

	if err == nil {
		t.Error("Expected type mismatch error but got none")
	}
	if task.Err == nil {
		t.Error("Expected task.Err to be set but got nil")
	}
}

func TestTaskAfter(t *testing.T) {
	task1 := taskflow.NewTask("task1", func(ctx context.Context, input string) (string, error) {
		return input + "_processed", nil
	})

	task2 := taskflow.NewTask("task2", func(ctx context.Context, input int) (int, error) {
		return input * 2, nil
	})

	task3 := taskflow.NewTask("task3", func(ctx context.Context, input string) (int, error) {
		return len(input), nil
	}).After(task1, task2)

	if len(task3.Depends) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(task3.Depends))
	}
}

func TestTaskRunWithDependencies(t *testing.T) {
	task1 := taskflow.NewTask("dep1", func(ctx context.Context, input any) (string, error) {
		return "hello", nil
	})

	task2 := taskflow.NewTask("dep2", func(ctx context.Context, input string) (string, error) {
		return input + " world", nil
	})

	mainTask := taskflow.NewTask("main", func(ctx context.Context, input string) (int, error) {
		return len(input), nil
	}).After(task1, task2)

	result, err := mainTask.Run(context.Background(), nil)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if result != 11 { // "hello world" has 11 characters
		t.Errorf("Expected result 11, got %v", result)
	}
}

func TestTaskRunWithDependencyError(t *testing.T) {
	expectedErr := errors.New("dependency error")

	failingDep := taskflow.NewTask("failing_dep", func(ctx context.Context, input any) (string, error) {
		return "", expectedErr
	})

	mainTask := taskflow.NewTask("main", func(ctx context.Context, input string) (int, error) {
		return len(input), nil
	}).After(failingDep)

	_, err := mainTask.Run(context.Background(), nil)

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
	if mainTask.Err != expectedErr {
		t.Errorf("Expected mainTask.Err %v, got %v", expectedErr, mainTask.Err)
	}
}

func TestTaskRunOnce(t *testing.T) {
	callCount := 0
	task := taskflow.NewTask("once_task", func(ctx context.Context, input any) (int, error) {
		callCount++
		return callCount, nil
	})

	// Run the task multiple times
	result1, err1 := task.Run(context.Background(), nil)
	result2, err2 := task.Run(context.Background(), nil)

	if err1 != nil || err2 != nil {
		t.Errorf("Unexpected errors: %v, %v", err1, err2)
	}
	if result1 != 1 {
		t.Errorf("Expected first result 1, got %v", result1)
	}
	if result2 != 1 {
		t.Errorf("Expected second result 1 (cached), got %v", result2)
	}
	if callCount != 1 {
		t.Errorf("Expected function to be called once, was called %d times", callCount)
	}
}

func TestTaskRunWithContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	task := taskflow.NewTask("slow_task", func(ctx context.Context, input any) (string, error) {
		select {
		case <-time.After(100 * time.Millisecond):
			return "completed", nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	})

	_, err := task.Run(ctx, nil)

	if err == nil {
		t.Error("Expected context timeout error but got none")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}
}

func TestTaskRunConcurrent(t *testing.T) {
	task := taskflow.NewTask("concurrent_task", func(ctx context.Context, input int) (int, error) {
		time.Sleep(50 * time.Millisecond) // Simulate some work
		return input * 2, nil
	})

	const numGoroutines = 10
	results := make(chan int, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(input int) {
			result, err := task.Run(context.Background(), input)
			if err != nil {
				errors <- err
			} else {
				results <- result.(int)
			}
		}(i)
	}

	// Collect results
	var resultSlice []int
	var errorSlice []error

	for i := 0; i < numGoroutines; i++ {
		select {
		case result := <-results:
			resultSlice = append(resultSlice, result)
		case err := <-errors:
			errorSlice = append(errorSlice, err)
		case <-time.After(5 * time.Second):
			t.Fatal("Test timed out")
		}
	}

	if len(errorSlice) > 0 {
		t.Errorf("Unexpected errors: %v", errorSlice)
	}

	// All goroutines should get the same result (first one to execute)
	if len(resultSlice) > 0 {
		expectedResult := resultSlice[0]
		for _, result := range resultSlice {
			if result != expectedResult {
				t.Errorf("Expected all results to be %d, got %d", expectedResult, result)
			}
		}
	}
}
