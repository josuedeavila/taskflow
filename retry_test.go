package taskflow_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/josuedeavila/taskflow"
)

func TestRetry_Success_FirstAttempt(t *testing.T) {
	callCount := 0
	fn := func(ctx context.Context) error {
		callCount++
		return nil
	}

	ctx := context.Background()
	err := taskflow.Retry(ctx, fn, 3, 10*time.Millisecond)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected function to be called once, was called %d times", callCount)
	}
}

func TestRetry_Success_SecondAttempt(t *testing.T) {
	callCount := 0
	fn := func(ctx context.Context) error {
		callCount++
		if callCount == 1 {
			return errors.New("first attempt failed")
		}
		return nil
	}

	ctx := context.Background()
	err := taskflow.Retry(ctx, fn, 3, 10*time.Millisecond)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected function to be called twice, was called %d times", callCount)
	}
}

func TestRetry_FailureAllAttempts(t *testing.T) {
	expectedErr := errors.New("persistent error")
	callCount := 0
	fn := func(ctx context.Context) error {
		callCount++
		return expectedErr
	}

	ctx := context.Background()
	err := taskflow.Retry(ctx, fn, 2, 10*time.Millisecond)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}

	// Should be called: initial + 2 retries = 3 times
	if callCount != 3 {
		t.Errorf("Expected function to be called 3 times, was called %d times", callCount)
	}
}

func TestRetry_ZeroRetries(t *testing.T) {
	expectedErr := errors.New("immediate failure")
	callCount := 0
	fn := func(ctx context.Context) error {
		callCount++
		return expectedErr
	}

	ctx := context.Background()
	err := taskflow.Retry(ctx, fn, 0, 10*time.Millisecond)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}

	// Should be called only once (initial attempt, no retries)
	if callCount != 1 {
		t.Errorf("Expected function to be called once, was called %d times", callCount)
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	callCount := 0
	fn := func(ctx context.Context) error {
		callCount++
		return errors.New("this should fail")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	// Use longer backoff to ensure context timeout happens
	err := taskflow.Retry(ctx, fn, 5, 20*time.Millisecond)

	if err == nil {
		t.Fatal("Expected context cancellation error, got nil")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}

	// Function should be called at least once, but not all retry attempts
	if callCount == 0 {
		t.Error("Expected function to be called at least once")
	}

	if callCount > 3 {
		t.Errorf("Expected context cancellation to prevent excessive calls, got %d calls", callCount)
	}
}

func TestRetry_ContextCancellationDuringFunction(t *testing.T) {
	callCount := 0
	fn := func(ctx context.Context) error {
		callCount++
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
			return errors.New("function completed")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	err := taskflow.Retry(ctx, fn, 3, 10*time.Millisecond)

	if err == nil {
		t.Fatal("Expected context cancellation error, got nil")
	}

	if err != context.DeadlineExceeded {
		t.Errorf("Expected context.DeadlineExceeded, got %v", err)
	}

	// Should be called at least once
	if callCount != 1 {
		t.Errorf("Expected function to be called once, was called %d times", callCount)
	}
}

func TestRetry_ExponentialBackoff(t *testing.T) {
	callCount := 0
	callTimes := []time.Time{}
	fn := func(ctx context.Context) error {
		callCount++
		callTimes = append(callTimes, time.Now())
		return errors.New("fail every time")
	}

	ctx := context.Background()
	initialBackoff := 50 * time.Millisecond

	start := time.Now()
	err := taskflow.Retry(ctx, fn, 2, initialBackoff)
	totalDuration := time.Since(start)

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}

	if len(callTimes) < 3 {
		t.Fatal("Not enough call times recorded")
	}

	// Check first backoff (should be ~50ms)
	firstBackoff := callTimes[1].Sub(callTimes[0])
	if firstBackoff < 40*time.Millisecond || firstBackoff > 70*time.Millisecond {
		t.Errorf("Expected first backoff ~50ms, got %v", firstBackoff)
	}

	// Check second backoff (should be ~100ms due to exponential backoff)
	secondBackoff := callTimes[2].Sub(callTimes[1])
	if secondBackoff < 80*time.Millisecond || secondBackoff > 130*time.Millisecond {
		t.Errorf("Expected second backoff ~100ms, got %v", secondBackoff)
	}

	// Total time should be at least the sum of backoffs
	expectedMinDuration := initialBackoff + (initialBackoff * 2)
	if totalDuration < expectedMinDuration {
		t.Errorf("Expected total duration >= %v, got %v", expectedMinDuration, totalDuration)
	}
}

func TestRetry_SuccessAfterRetries_CheckBackoff(t *testing.T) {
	callCount := 0
	callTimes := []time.Time{}
	fn := func(ctx context.Context) error {
		callCount++
		callTimes = append(callTimes, time.Now())
		if callCount < 3 {
			return errors.New("fail first two times")
		}
		return nil
	}

	ctx := context.Background()
	initialBackoff := 30 * time.Millisecond

	start := time.Now()
	err := taskflow.Retry(ctx, fn, 5, initialBackoff)
	totalDuration := time.Since(start)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}

	// Should have stopped retrying after success
	expectedMinDuration := initialBackoff + (initialBackoff * 2) // 30ms + 60ms
	if totalDuration < expectedMinDuration-10*time.Millisecond {
		t.Errorf("Expected total duration >= %v, got %v", expectedMinDuration, totalDuration)
	}
}

func TestRetry_ImmediateContextCancellation(t *testing.T) {
	callCount := 0
	fn := func(ctx context.Context) error {
		callCount++
		return errors.New("should not matter")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := taskflow.Retry(ctx, fn, 3, 10*time.Millisecond)

	if err == nil {
		t.Fatal("Expected context cancellation error, got nil")
	}

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled, got %v", err)
	}

	// Function should still be called once before checking context
	if callCount != 1 {
		t.Errorf("Expected function to be called once, was called %d times", callCount)
	}
}
