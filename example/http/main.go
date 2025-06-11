package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/josuedeavila/taskflow"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	apis := []string{
		"https://api.github.com",
		"https://jsonplaceholder.typicode.com/posts/1",
		"https://httpbin.org/get",
	}

	fan := &taskflow.FanOutTask{
		Name: "check_public_apis",
		Generate: func(ctx context.Context) ([]taskflow.TaskFunc, error) {
			var fns []taskflow.TaskFunc
			for _, url := range apis {
				url := url
				fns = append(fns, func(ctx context.Context, _ any) (any, error) {
					req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
					if err != nil {
						return map[string]any{"url": url, "status": "request-error", "err": err.Error()}, nil
					}
					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						return map[string]any{"url": url, "status": "timeout", "err": err.Error()}, nil
					}
					defer resp.Body.Close()
					fmt.Println("✔️ Fetched from:", url, "Status:", resp.StatusCode)
					return map[string]any{
						"url":    url,
						"status": resp.StatusCode,
					}, nil
				})
			}
			return fns, nil
		},

		FanIn: func(ctx context.Context, input any) (any, error) {
			results := input.([]any)
			statusCounts := make(map[string]int)
			for _, r := range results {
				res := r.(map[string]any)
				status := fmt.Sprintf("%v", res["status"])
				statusCounts[status]++
			}

			fmt.Println("📊 Resumo de status (incluindo falhas):")
			for k, v := range statusCounts {
				fmt.Printf("  %s: %d respostas\n", k, v)
			}
			return statusCounts, nil
		},
	}

	aggregate := fan.ToTask()

	logTask := taskflow.NewTask("log_summary", func(ctx context.Context, input any) (any, error) {
		fmt.Println("✅ Finalizado.")
		return nil, nil
	}).After(aggregate)

	runner := taskflow.NewRunner()
	runner.Add(logTask)

	// Executa
	if err := runner.Run(ctx); err != nil {
		fmt.Println("❌ Erro:", err)
	}
}