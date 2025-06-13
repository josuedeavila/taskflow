# TaskFlow

A Go library for task execution with dependencies and parallel processing.

[Interface, types, and methods documentation](https://pkg.go.dev/github.com/josuedeavila/taskflow)

## Installation

```bash
go get github.com/josuedeavila/taskflow
```

## Basic Usage

### Tasks with Dependencies

```go
package main

import (
    "context"
    "fmt"
    "github.com/josuedeavila/taskflow"
)

func main() {
    task1 := taskflow.NewTask("fetch", func(ctx context.Context, input any) (string, error) {
        return "data", nil
    })
    
    task2 := taskflow.NewTask("process", func(ctx context.Context, input string) (string, error) {
        return "processed_" + input, nil
    }).After(task1)
    
    runner := taskflow.NewRunner()
    runner.Add(task2)
    
    err := runner.Run(context.Background())
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    }
}
```

### Parallel Processing (Fan-Out/Fan-In)

```go
fanOut := &taskflow.FanOutTask[any, float64]{
    Name: "parallel_calc",
    Generate: func(ctx context.Context) ([]taskflow.TaskFunc[any, float64], error) {
        return []taskflow.TaskFunc[any, float64]{
            func(ctx context.Context, _ any) (float64, error) { return 10.0, nil },
            func(ctx context.Context, _ any) (float64, error) { return 20.0, nil },
            func(ctx context.Context, _ any) (float64, error) { return 30.0, nil },
        }, nil
    },
    FanIn: func(ctx context.Context, results any) (float64, error) {
        sum := 0.0
        for _, r := range results.([]any) {
            sum += r.(float64)
        }
        return sum, nil
    },
}

task := fanOut.ToTask()
runner := taskflow.NewRunner()
runner.Add(task)
runner.Run(context.Background())
```

### Retry with Backoff

```go
err := taskflow.Retry(ctx, func(ctx context.Context) error {
    // operation that might fail
    return doSomething()
}, 3, time.Second)
```

## Components

- **Task**: Work unit with generic type support
- **Runner**: Executes tasks respecting dependencies
- **FanOutTask**: Parallel execution with result consolidation
- **Retry**: Retry with exponential backoff

## Examples

The project includes three examples in the `example/` folder:

- `simple/`: Basic orchestrator with retry
- `concurrent/`: Tasks with dependencies and parallel processing
- `http/`: Parallel API checking

Run with:

```bash
go run example/simple/main.go
```

## Important

The number of tasks added to the Runner directly defines the number of goroutines created during execution. Each task is executed independently (respecting dependencies), which can lead to the creation of many simultaneous goroutines.

→ **Caution**: use carefully in resource-constrained environments or when dealing with many tasks.

Tasks are executed concurrently, but not necessarily in parallel — parallelization depends on the availability of Go runtime threads and the operating system.

## License

MIT License - see [LICENSE](LICENSE)