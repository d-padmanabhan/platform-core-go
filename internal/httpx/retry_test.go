package httpx

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExponentialBackoffDelay_NoJitter(t *testing.T) {
	t.Parallel()

	delay0 := ExponentialBackoffDelay(0, time.Second, 30*time.Second, false, 0.0)
	delay1 := ExponentialBackoffDelay(1, time.Second, 30*time.Second, false, 0.0)
	delay2 := ExponentialBackoffDelay(2, time.Second, 30*time.Second, false, 0.0)

	if delay0 != time.Second {
		t.Fatalf("attempt 0 delay mismatch: got=%s want=%s", delay0, time.Second)
	}
	if delay1 != 2*time.Second {
		t.Fatalf("attempt 1 delay mismatch: got=%s want=%s", delay1, 2*time.Second)
	}
	if delay2 != 4*time.Second {
		t.Fatalf("attempt 2 delay mismatch: got=%s want=%s", delay2, 4*time.Second)
	}
}

func TestRetry_SucceedsAfterTransientFailures(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	attempts := 0
	var slept []time.Duration

	err := Retry(
		ctx,
		RetryConfig{
			MaxRetries:   3,
			BaseDelay:    time.Second,
			MaxDelay:     30 * time.Second,
			EnableJitter: false,
			Sleep: func(_ context.Context, d time.Duration) error {
				slept = append(slept, d)
				return nil
			},
		},
		func(err error) bool { return errors.Is(err, errTransient) },
		func(context.Context) error {
			attempts++
			if attempts < 3 {
				return errTransient
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempt mismatch: got=%d want=3", attempts)
	}
	if len(slept) != 2 {
		t.Fatalf("sleep count mismatch: got=%d want=2", len(slept))
	}
	if slept[0] != time.Second || slept[1] != 2*time.Second {
		t.Fatalf("unexpected sleep sequence: %#v", slept)
	}
}

var (
	errTransient = errors.New("transient")
	errPermanent = errors.New("permanent")
)

func TestRetry_StopsOnNonRetryableError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	attempts := 0

	err := Retry(
		ctx,
		RetryConfig{
			MaxRetries: 2,
			BaseDelay:  time.Second,
			MaxDelay:   10 * time.Second,
			Sleep:      func(context.Context, time.Duration) error { return nil },
		},
		func(err error) bool { return errors.Is(err, errTransient) },
		func(context.Context) error {
			attempts++
			return errPermanent
		},
	)

	if !errors.Is(err, errPermanent) {
		t.Fatalf("expected permanent error, got: %v", err)
	}
	if attempts != 1 {
		t.Fatalf("expected single attempt, got: %d", attempts)
	}
}
