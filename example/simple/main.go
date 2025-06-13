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
				log.Printf("⚠️ Task failed after all retries: %v", err)
			}
		}
	}
}

func (o *MinimalOrchestrator) execute(ctx context.Context, interactionType InteractionType, config *ProcessingConfig) error {
	var err error
	var result any

	for attempt := 1; attempt <= config.MaxRetries+1; attempt++ {
		log.Printf("🚀 Executing task (%d/%d): %s", attempt, config.MaxRetries+1, interactionType)

		jobCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		result, err = o.runPipeline(jobCtx, interactionType)
		cancel()

		if err == nil {
			log.Printf("✅ Task completed: %+v", result)
			return nil
		}

		log.Printf("❌ Attempt %d failed: %v", attempt, err)

		if attempt <= config.MaxRetries {
			log.Printf("⏳ Waiting %s before retry...", config.RetryDelay)
			time.Sleep(config.RetryDelay)
		}
	}

	log.Printf("🛑 All attempts failed for %s", interactionType)
	return err
}

func (o *MinimalOrchestrator) runPipeline(ctx context.Context, interactionType InteractionType) (any, error) {
	var finalResult any

	fetch := taskflow.NewTask("fetch", func(ctx context.Context, _ any) ([]string, error) {
		log.Printf("🔍 Fetching events for %s", interactionType)
		time.Sleep(500 * time.Millisecond)
		return []string{"evt1", "evt2"}, nil
	}).WithLogger(&taskflow.NoOpLogger{})

	process := taskflow.NewTask("process", func(ctx context.Context, input []string) (map[string]int, error) {
		log.Printf("⚙️ Processing %d events", len(input))

		// Simulates error
		if time.Now().Unix()%2 == 0 {
			return nil, fmt.Errorf("simulated error")
		}

		result := map[string]int{"processed": len(input)}
		return result, nil
	}).WithLogger(&taskflow.NoOpLogger{}).After(fetch)

	capture := taskflow.NewTask("capture", func(ctx context.Context, input map[string]int) (any, error) {
		finalResult = input
		log.Printf("📦 Result captured: %+v", finalResult)
		return nil, nil
	}).WithLogger(&taskflow.NoOpLogger{}).After(process)

	runner := taskflow.NewRunner()
	runner.Add(capture)

	err := runner.Run(ctx)
	return finalResult, err
}

func (o *MinimalOrchestrator) Shutdown() {
	log.Println("🛑 Shutting down orchestrator...")
	close(o.shutdown)
	o.wg.Wait()
	log.Println("✅ Shutdown complete")
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
