package awsx

import (
	"context"
	"errors"
	"testing"
)

func TestValidateRegion(t *testing.T) {
	t.Parallel()

	if err := ValidateRegion("us-east-1"); err != nil {
		t.Fatalf("expected valid region, got error: %v", err)
	}

	err := ValidateRegion("moon-1")
	if !errors.Is(err, ErrInvalidRegion) {
		t.Fatalf("expected ErrInvalidRegion, got: %v", err)
	}
}

func TestNewFactory_InvalidRegion(t *testing.T) {
	t.Parallel()

	_, err := NewFactory(context.Background(), "invalid-1")
	if !errors.Is(err, ErrInvalidRegion) {
		t.Fatalf("expected ErrInvalidRegion, got: %v", err)
	}
}
