package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/josuedeavila/taskflow" // Assumindo que o pacote taskflow está no seu GOPATH/module
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

	// 1. Criando algumas funções de tarefa simples
	taskFn1 := func(ctx context.Context, _ any) (string, error) {
		logger.Info("Executando Task 1: Busca de dados de usuários...")
		time.Sleep(1 * time.Second) // Simula trabalho
		select {
		case <-ctx.Done():
			logger.Info("Task 1 cancelada!")
			return "", ctx.Err()
		default:
			logger.Info("Task 1 concluída.")
			return "dados_usuarios", nil
		}
	}

	taskFn2 := func(ctx context.Context, input any) (string, error) {
		logger.Info("Executando Task 2: Processamento de dados de produtos...")
		time.Sleep(500 * time.Millisecond) // Simula trabalho
		select {
		case <-ctx.Done():
			logger.Info("Task 2 cancelada!")
			return "", ctx.Err()
		default:
			logger.Info("Task 2 concluída.")
			return "dados_produtos", nil
		}
	}

	taskFn3 := func(ctx context.Context, input any) (string, error) {
		logger.Info("Executando Task 3: Gerando relatório (depende de Task 1 e Task 2)...")
		time.Sleep(1500 * time.Millisecond) // Simula trabalho
		select {
		case <-ctx.Done():
			logger.Info("Task 3 cancelada!")
			return "", ctx.Err()
		default:
			logger.Info(fmt.Sprintf("Task 3 concluída com input: %v", input))
			return "relatorio_final", nil
		}
	}

	taskFnErro := func(ctx context.Context, input any) (any, error) {
		logger.Info("Executando Task Erro: Simulando uma falha...")
		time.Sleep(200 * time.Millisecond)
		select {
		case <-ctx.Done():
			logger.Info("Task Erro cancelada!")
			return nil, ctx.Err()
		default:
			return nil, fmt.Errorf("erro proposital na Task Erro")
		}
	}

	// 2. Criando as tarefas
	task1 := taskflow.NewTask("BuscarUsuarios", taskFn1).WithLogger(logger) // Adiciona um logger padrão
	task2 := taskflow.NewTask("ProcessarProdutos", taskFn2).WithLogger(logger)
	task3 := taskflow.NewTask("GerarRelatorio", taskFn3).WithLogger(logger).After(task1, task2) // Task 3 depende de Task 1 e Task 2
	taskError := taskflow.NewTask("SimularErro", taskFnErro).WithLogger(logger)

	// 3. Criando um FanOutTask
	fanOutGenerateFunc := func(ctx context.Context, _ []any) ([]taskflow.TaskFunc[any, float64], error) {
		logger.Info("FanOutTask: Gerando funções de fan-out...")
		fns := []taskflow.TaskFunc[any, float64]{
			func(ctx context.Context, input any) (float64, error) {
				logger.Info("FanOut Sub-Task A: Calculando métrica X...")
				time.Sleep(300 * time.Millisecond)
				return 10.5, nil
			},
			func(ctx context.Context, input any) (float64, error) {
				logger.Info("FanOut Sub-Task B: Calculando métrica Y...")
				time.Sleep(700 * time.Millisecond)
				return 20.0, nil
			},
			func(ctx context.Context, input any) (float64, error) {
				logger.Info("FanOut Sub-Task C: Calculando métrica Z...")
				time.Sleep(400 * time.Millisecond)
				return 5.2, nil
			},
		}
		return fns, nil
	}

	fanInFunc := func(ctx context.Context, results []float64) (float64, error) {
		logger.Info("FanOutTask: Consolidando resultados...")
		sum := 0.0
		for _, r := range results {
			sum += r
		}
		logger.Info(fmt.Sprintf("FanOutTask: Soma dos resultados: %.2f", sum))
		return sum, nil
	}

	fanOutTask := &taskflow.FanOutTask[any, float64]{
		Name:     "CalcularMetricas",
		Generate: fanOutGenerateFunc,
		FanIn:    fanInFunc,
	}

	// Converte o FanOutTask em um Task normal
	fanOutConvertedTask := fanOutTask.ToTask()

	// 4. Criando o Runner e adicionando as tarefas
	runner := taskflow.NewRunner()
	runner.Add(task1, task2, task3, taskError, fanOutConvertedTask)

	// 5. Executando as tarefas com um contexto que pode ser cancelado
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) // Define um timeout para o runner
	defer cancel()

	logger.Info("Executando o Runner...")
	err := runner.Run(ctx)

	if err != nil {
		logger.Info(fmt.Sprintf("Runner concluído com erro: %v", err))
	} else {
		logger.Info("Runner concluído com sucesso!")
	}

	// 6. Verificando os resultados e estados das tarefas
	logger.Info("Verificando resultados das tarefas:")
	logger.Info(fmt.Sprintf("Task 'BuscarUsuarios' - Resultado: %v, Erro: %v", task1.Result, task1.Err))
	logger.Info(fmt.Sprintf("Task 'ProcessarProdutos' - Resultado: %v, Erro: %v", task2.Result, task2.Err))
	logger.Info(fmt.Sprintf("Task 'GerarRelatorio' - Resultado: %v, Erro: %v", task3.Result, task3.Err))
	logger.Info(fmt.Sprintf("Task 'SimularErro' - Resultado: %v, Erro: %v", taskError.Result, taskError.Err))
	logger.Info(fmt.Sprintf("Task 'CalcularMetricas' - Resultado: %v, Erro: %v", fanOutConvertedTask.Result, fanOutConvertedTask.Err))

	// Exemplo de como você pode ver o estado de uma tarefa (após a execução)
	// Isso exigiria expor o campo 'state' ou um método para obtê-lo na struct Task.
	// Por enquanto, o `logger.Log` dentro dos estados já mostra a transição.
}
