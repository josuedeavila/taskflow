package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
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
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	apis := []string{
		"https://api.github.com",
		"https://jsonplaceholder.typicode.com/posts/1",
		"https://httpbin.org/get",
	}

	fan := &taskflow.FanOutTask[any, map[string]any]{
		Name: "check_public_apis",
		Generate: func(ctx context.Context, _ []any) ([]taskflow.TaskFunc[any, map[string]any], error) {
			var fns []taskflow.TaskFunc[any, map[string]any]
			for _, url := range apis {
				url := url
				fns = append(fns, func(ctx context.Context, _ any) (map[string]any, error) {
					req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
					if err != nil {
						return map[string]any{"url": url, "status": "request-error", "err": err.Error()}, nil
					}
					resp, err := http.DefaultClient.Do(req)
					if err != nil {
						return map[string]any{"url": url, "status": "timeout", "err": err.Error()}, nil
					}
					defer resp.Body.Close()
					logger.Info(fmt.Sprintf("✔️ Fetched from: %s,  Status: %v", url, resp.StatusCode))
					return map[string]any{
						"url":    url,
						"status": resp.StatusCode,
					}, nil
				})
			}
			return fns, nil
		},

		FanIn: func(ctx context.Context, input []map[string]any) (map[string]any, error) {
			results := input
			statusCounts := make(map[string]int)
			for _, r := range results {
				status := fmt.Sprintf("%v", r["status"])
				statusCounts[status]++
			}

			logger.Info("📊 Resumo de status (incluindo falhas):")
			for k, v := range statusCounts {
				logger.Info(fmt.Sprintf("  %s: %d respostas\n", k, v))
			}
			return nil, nil
		},
	}

	aggregate := fan.ToTask()

	logTask := taskflow.NewTask("log_summary", func(ctx context.Context, input any) (any, error) {
		logger.Info("✅ Finalizado.")
		return nil, nil
	}).After(aggregate)

	runner := taskflow.NewRunner()
	runner.Add(logTask)

	// Executa
	if err := runner.Run(ctx); err != nil {
		logger.Error(fmt.Sprintf("❌ Erro: %s", err))
	}
}
