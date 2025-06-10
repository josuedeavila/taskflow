package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/josuedeavila/taskflow"
)

type InteractionType string

const (
	OfferUpdate InteractionType = "offer_update"
)

type ProcessingConfig struct {
	InteractionType InteractionType
	ProcessInterval time.Duration
	MaxConcurrency  int
	MaxRetries      int
	RetryDelay      time.Duration
}

type MinimalOrchestrator struct {
	configs    map[InteractionType]*ProcessingConfig
	semaphores map[InteractionType]chan struct{}
	shutdown   chan struct{}
	wg         sync.WaitGroup
}

func NewMinimalOrchestrator() *MinimalOrchestrator {
	return &MinimalOrchestrator{
		configs:    make(map[InteractionType]*ProcessingConfig),
		semaphores: make(map[InteractionType]chan struct{}),
		shutdown:   make(chan struct{}),
	}
}

func (o *MinimalOrchestrator) AddConfig(config *ProcessingConfig) {
	o.configs[config.InteractionType] = config
	o.semaphores[config.InteractionType] = make(chan struct{}, config.MaxConcurrency)
}

func (o *MinimalOrchestrator) Start(ctx context.Context) {
	for t, config := range o.configs {
		o.wg.Add(1)
		go o.runLoop(ctx, t, config)
	}
}

func (o *MinimalOrchestrator) runLoop(ctx context.Context, interactionType InteractionType, config *ProcessingConfig) {
	defer o.wg.Done()
	ticker := time.NewTicker(config.ProcessInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-o.shutdown:
			return
		case <-ticker.C:
			o.semaphores[interactionType] <- struct{}{}
			err := o.execute(ctx, interactionType, config)
			<-o.semaphores[interactionType]

			if err != nil {
				log.Printf("⚠️ Tarefa falhou após todos os retries: %v", err)
			}
		}
	}
}

func (o *MinimalOrchestrator) execute(ctx context.Context, interactionType InteractionType, config *ProcessingConfig) error {
	var err error
	var result interface{}

	for attempt := 1; attempt <= config.MaxRetries+1; attempt++ {
		log.Printf("🚀 Executando tarefa (%d/%d): %s", attempt, config.MaxRetries+1, interactionType)

		jobCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		result, err = o.runPipeline(jobCtx, interactionType)
		cancel()

		if err == nil {
			log.Printf("✅ Tarefa concluída: %+v", result)
			return nil
		}

		log.Printf("❌ Falha na tentativa %d: %v", attempt, err)

		if attempt <= config.MaxRetries {
			log.Printf("⏳ Aguardando %s antes do retry...", config.RetryDelay)
			time.Sleep(config.RetryDelay)
		}
	}

	log.Printf("🛑 Todas as tentativas falharam para %s", interactionType)
	return err
}

func (o *MinimalOrchestrator) runPipeline(ctx context.Context, interactionType InteractionType) (interface{}, error) {
	var finalResult interface{}

	fetch := taskflow.NewTask("fetch", func(ctx context.Context, input interface{}) (interface{}, error) {
		log.Printf("🔍 Buscando eventos para %s", interactionType)
		time.Sleep(500 * time.Millisecond)
		return []string{"evt1", "evt2"}, nil
	})

	process := taskflow.NewTask("process", func(ctx context.Context, input interface{}) (interface{}, error) {
		events := input.([]string)
		log.Printf("⚙️ Processando %d eventos", len(events))

		// Simula erro
		if time.Now().Unix()%2 == 0 {
			return nil, fmt.Errorf("erro simulado")
		}

		result := map[string]int{"processed": len(events)}
		return result, nil
	}).After(fetch)

	capture := taskflow.NewTask("capture", func(ctx context.Context, input interface{}) (interface{}, error) {
		finalResult = input.(map[string]int)
		log.Printf("📦 Capturando resultado: %+v", finalResult)
		return input, nil
	}).After(process)

	runner := taskflow.NewRunner()
	runner.Add(capture)

	err := runner.Run(ctx)
	return finalResult, err
}

func (o *MinimalOrchestrator) Shutdown() {
	log.Println("🛑 Encerrando orquestrador...")
	close(o.shutdown)
	o.wg.Wait()
	log.Println("✅ Encerrado")
}

func main() {
	orchestrator := NewMinimalOrchestrator()

	orchestrator.AddConfig(&ProcessingConfig{
		InteractionType: OfferUpdate,
		ProcessInterval: 3 * time.Second,
		MaxConcurrency:  1,
		MaxRetries:      3,
		RetryDelay:      2 * time.Second,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	orchestrator.Start(ctx)

	time.AfterFunc(20*time.Second, func() {
		cancel()
		orchestrator.Shutdown()
	})

	<-ctx.Done()
}
