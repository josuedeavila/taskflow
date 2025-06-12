# TaskFlow

Uma biblioteca Go para execução de tarefas com dependências e processamento paralelo.

[Documentação de interfaces, tipos e métodos](https://pkg.go.dev/github.com/josuedeavila/taskflow)

## Instalação

```bash
go get github.com/josuedeavila/taskflow
```

## Uso Básico

### Tarefas com Dependências

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

### Processamento Paralelo (Fan-Out/Fan-In)

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

### Retry com Backoff

```go
err := taskflow.Retry(ctx, func(ctx context.Context) error {
    // operação que pode falhar
    return doSomething()
}, 3, time.Second)
```

## Componentes

- **Task**: Unidade de trabalho com suporte a tipos genéricos
- **Runner**: Executa tarefas respeitando dependências
- **FanOutTask**: Execução paralela com consolidação de resultados
- **Retry**: Retry com backoff exponencial

## Exemplos

O projeto inclui três exemplos na pasta `example/`:

- `simple/`: Orquestrador básico com retry
- `concurrent/`: Tarefas com dependências e processamento paralelo  
- `http/`: Verificação paralela de APIs

Execute com:
```bash
go run example/simple/main.go
```

## Importante

O número de tarefas adicionadas ao Runner define diretamente o número de goroutines criadas durante a execução. Cada tarefa é executada de forma independente (respeitando as dependências), o que pode levar à criação de muitas goroutines simultâneas.
→ Atenção: use com cuidado em ambientes com recursos limitados ou quando estiver lidando com muitas tarefas.

As tarefas são executadas concorretemente, mas não necessariamente em paralelo — a paralelização depende da disponibilidade de threads do Go runtime e do sistema operacional.

## Licença

MIT License - veja [LICENSE](LICENSE)