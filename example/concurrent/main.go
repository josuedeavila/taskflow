package main

import (
	"context"
	"fmt"
	"time"

	"github.com/josuedeavila/taskflow" // Assumindo que o pacote taskflow está no seu GOPATH/module
)

func main() {
	fmt.Println("Iniciando o exemplo TaskFlow com Máquina de Estados...")

	// 1. Criando algumas funções de tarefa simples
	taskFn1 := func(ctx context.Context, _ any) (string, error) {
		fmt.Println("  Executando Task 1: Busca de dados de usuários...")
		time.Sleep(1 * time.Second) // Simula trabalho
		select {
		case <-ctx.Done():
			fmt.Println("  Task 1 cancelada!")
			return "", ctx.Err()
		default:
			fmt.Println("  Task 1 concluída.")
			return "dados_usuarios", nil
		}
	}

	taskFn2 := func(ctx context.Context, input any) (string, error) {
		fmt.Println("  Executando Task 2: Processamento de dados de produtos...")
		time.Sleep(500 * time.Millisecond) // Simula trabalho
		select {
		case <-ctx.Done():
			fmt.Println("  Task 2 cancelada!")
			return "", ctx.Err()
		default:
			fmt.Println("  Task 2 concluída.")
			return "dados_produtos", nil
		}
	}

	taskFn3 := func(ctx context.Context, input any) (string, error) {
		fmt.Println("  Executando Task 3: Gerando relatório (depende de Task 1 e Task 2)...")
		time.Sleep(1500 * time.Millisecond) // Simula trabalho
		select {
		case <-ctx.Done():
			fmt.Println("  Task 3 cancelada!")
			return "", ctx.Err()
		default:
			fmt.Printf("  Task 3 concluída com input: %v\n", input)
			return "relatorio_final", nil
		}
	}

	taskFnErro := func(ctx context.Context, input any) (any, error) {
		fmt.Println("  Executando Task Erro: Simulando uma falha...")
		time.Sleep(200 * time.Millisecond)
		select {
		case <-ctx.Done():
			fmt.Println("  Task Erro cancelada!")
			return nil, ctx.Err()
		default:
			return nil, fmt.Errorf("erro proposital na Task Erro")
		}
	}

	// 2. Criando as tarefas
	task1 := taskflow.NewTask("BuscarUsuarios", taskFn1)
	task2 := taskflow.NewTask("ProcessarProdutos", taskFn2)
	task3 := taskflow.NewTask("GerarRelatorio", taskFn3).After(task1, task2) // Task 3 depende de Task 1 e Task 2
	taskError := taskflow.NewTask("SimularErro", taskFnErro)

	// 3. Criando um FanOutTask
	fanOutGenerateFunc := func(ctx context.Context) ([]taskflow.TaskFunc[any, float64], error) {
		fmt.Println("  FanOutTask: Gerando funções de fan-out...")
		fns := []taskflow.TaskFunc[any, float64]{
			func(ctx context.Context, input any) (float64, error) {
				fmt.Println("    FanOut Sub-Task A: Calculando métrica X...")
				time.Sleep(300 * time.Millisecond)
				return 10.5, nil
			},
			func(ctx context.Context, input any) (float64, error) {
				fmt.Println("    FanOut Sub-Task B: Calculando métrica Y...")
				time.Sleep(700 * time.Millisecond)
				return 20.0, nil
			},
			func(ctx context.Context, input any) (float64, error) {
				fmt.Println("    FanOut Sub-Task C: Calculando métrica Z...")
				time.Sleep(400 * time.Millisecond)
				return 5.2, nil
			},
		}
		return fns, nil
	}

	fanInFunc := func(ctx context.Context, results any) (float64, error) {
		fmt.Println("  FanOutTask: Consolidando resultados...")
		if resSlice, ok := results.([]any); ok {
			sum := 0.0
			for _, r := range resSlice {
				if val, ok := r.(float64); ok {
					sum += val
				}
			}
			fmt.Printf("  FanOutTask: Soma dos resultados: %.2f\n", sum)
			return sum, nil
		}
		return 0, fmt.Errorf("erro ao consolidar resultados do FanOut")
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

	fmt.Println("\nExecutando o Runner...")
	err := runner.Run(ctx)

	if err != nil {
		fmt.Printf("\nRunner concluído com erro: %v\n", err)
	} else {
		fmt.Println("\nRunner concluído com sucesso!")
	}

	// 6. Verificando os resultados e estados das tarefas
	fmt.Println("\nVerificando resultados das tarefas:")
	fmt.Printf("Task 'BuscarUsuarios' - Resultado: %v, Erro: %v\n", task1.Result, task1.Err)
	fmt.Printf("Task 'ProcessarProdutos' - Resultado: %v, Erro: %v\n", task2.Result, task2.Err)
	fmt.Printf("Task 'GerarRelatorio' - Resultado: %v, Erro: %v\n", task3.Result, task3.Err)
	fmt.Printf("Task 'SimularErro' - Resultado: %v, Erro: %v\n", taskError.Result, taskError.Err)
	fmt.Printf("Task 'CalcularMetricas' - Resultado: %v, Erro: %v\n", fanOutConvertedTask.Result, fanOutConvertedTask.Err)

	// Exemplo de como você pode ver o estado de uma tarefa (após a execução)
	// Isso exigiria expor o campo 'state' ou um método para obtê-lo na struct Task.
	// Por enquanto, o `fmt.Println` dentro dos estados já mostra a transição.
}
