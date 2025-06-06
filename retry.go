package taskflow

import (
    "context"
    "time"
)


// Retry executes a function with retries and exponential backoff.
// It will retry the function up to 'retries' times, doubling the backoff duration each time.
// If the context is done before the function succeeds, it returns the context's error.
func Retry(ctx context.Context, fn func(context.Context) error, retries int, backoff time.Duration) error {
    var err error
    for i := 0; i <= retries; i++ {
        err = fn(ctx)
        if err == nil {
            return nil
        }

        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(backoff):
            backoff *= 2
        }
    }
    return err
}
