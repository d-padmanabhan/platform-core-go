package awsx

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

var (
	// ErrInvalidRegion indicates the requested AWS region is not allowed.
	ErrInvalidRegion = errors.New("invalid aws region")
	allowedRegions   = map[string]struct{}{
		"us-east-1":      {},
		"us-east-2":      {},
		"us-west-2":      {},
		"ca-central-1":   {},
		"eu-west-1":      {},
		"eu-west-2":      {},
		"eu-central-1":   {},
		"eu-north-1":     {},
		"ap-southeast-1": {},
		"ap-southeast-2": {},
	}
)

// Factory stores shared AWS configuration for helper operations.
type Factory struct {
	cfg aws.Config
}

// ValidateRegion verifies a region against the platform allowlist.
func ValidateRegion(region string) error {
	if _, ok := allowedRegions[region]; !ok {
		return fmt.Errorf("%w: %s", ErrInvalidRegion, region)
	}
	return nil
}

// NewFactory builds an AWS helper factory with standard retry settings.
func NewFactory(
	ctx context.Context,
	region string,
	loadOptions ...func(*config.LoadOptions) error,
) (*Factory, error) {
	if err := ValidateRegion(region); err != nil {
		return nil, err
	}

	baseOptions := []func(*config.LoadOptions) error{
		config.WithRegion(region),
		config.WithRetryMode(aws.RetryModeStandard),
		config.WithRetryMaxAttempts(5),
	}
	baseOptions = append(baseOptions, loadOptions...)

	cfg, err := config.LoadDefaultConfig(ctx, baseOptions...)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	return &Factory{cfg: cfg}, nil
}

// Region returns the configured AWS region.
func (f *Factory) Region() string {
	return f.cfg.Region
}
