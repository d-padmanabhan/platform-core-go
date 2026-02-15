package httpx

import (
	"context"
	"math"
	"time"
)

const (
	defaultMaxRetries = 3
	defaultBaseDelay  = 1 * time.Second
	defaultMaxDelay   = 30 * time.Second
	maxJitterFraction = 0.1
)

// RetryConfig configures retry behavior for transient operation failures.
type RetryConfig struct {
	// MaxRetries is the number of retry attempts after the initial call.
	MaxRetries int
	// BaseDelay is the first retry delay before exponential growth.
	BaseDelay time.Duration
	// MaxDelay caps the exponential backoff delay.
	MaxDelay time.Duration
	// EnableJitter adds randomized jitter to reduce retry synchronization.
	EnableJitter bool

	// RandomFloat returns a value in [0,1) used for jitter.
	RandomFloat func() float64
	// Sleep can be overridden in tests.
	Sleep func(context.Context, time.Duration) error
}

func (c RetryConfig) withDefaults() RetryConfig {
	cfg := c

	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = defaultMaxRetries
	}
	if cfg.BaseDelay <= 0 {
		cfg.BaseDelay = defaultBaseDelay
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = defaultMaxDelay
	}
	if cfg.RandomFloat == nil {
		cfg.RandomFloat = func() float64 { return 0.0 }
	}
	if cfg.Sleep == nil {
		cfg.Sleep = SleepContext
	}

	return cfg
}

// ExponentialBackoffDelay computes delay for a retry attempt.
func ExponentialBackoffDelay(
	attempt int,
	baseDelay time.Duration,
	maxDelay time.Duration,
	enableJitter bool,
	jitterValue float64,
) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	if baseDelay <= 0 {
		baseDelay = defaultBaseDelay
	}
	if maxDelay <= 0 {
		maxDelay = defaultMaxDelay
	}

	backoff := float64(baseDelay) * math.Pow(2, float64(attempt))
	delay := time.Duration(backoff)
	if delay > maxDelay {
		delay = maxDelay
	}

	if !enableJitter {
		return delay
	}

	if jitterValue < 0 {
		jitterValue = 0
	}
	if jitterValue > 0.999999 {
		jitterValue = 0.999999
	}

	jitterRange := float64(delay) * maxJitterFraction
	jitter := time.Duration(jitterRange * jitterValue)
	return delay + jitter
}

// SleepContext sleeps for the provided delay or returns early when context is canceled.
func SleepContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// Retry runs operation with exponential backoff while shouldRetry returns true.
func Retry(
	ctx context.Context,
	cfg RetryConfig,
	shouldRetry func(error) bool,
	operation func(context.Context) error,
) error {
	config := cfg.withDefaults()

	for attempt := 0; ; attempt++ {
		err := operation(ctx)
		if err == nil {
			return nil
		}

		if !shouldRetry(err) || attempt >= config.MaxRetries {
			return err
		}

		delay := ExponentialBackoffDelay(
			attempt,
			config.BaseDelay,
			config.MaxDelay,
			config.EnableJitter,
			config.RandomFloat(),
		)

		if sleepErr := config.Sleep(ctx, delay); sleepErr != nil {
			return sleepErr
		}
	}
}
