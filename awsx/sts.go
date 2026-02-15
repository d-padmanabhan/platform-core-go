package awsx

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/aws-sdk-go-v2/service/sts/types"
)

// AccountID returns the caller account ID for the configured credentials.
func (f *Factory) AccountID(ctx context.Context) (string, error) {
	client := sts.NewFromConfig(f.cfg)
	output, err := client.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return "", fmt.Errorf("get caller identity: %w", err)
	}
	if output.Account == nil || strings.TrimSpace(*output.Account) == "" {
		return "", errors.New("get caller identity returned empty account ID")
	}

	return strings.TrimSpace(*output.Account), nil
}

// AssumeRole assumes an IAM role and returns temporary credentials.
func (f *Factory) AssumeRole(
	ctx context.Context,
	roleARN string,
	sessionName string,
	duration time.Duration,
) (*types.Credentials, error) {
	if strings.TrimSpace(roleARN) == "" {
		return nil, errors.New("role ARN must not be empty")
	}
	if strings.TrimSpace(sessionName) == "" {
		return nil, errors.New("role session name must not be empty")
	}

	input := &sts.AssumeRoleInput{
		RoleArn:         &roleARN,
		RoleSessionName: &sessionName,
	}
	if duration > 0 {
		seconds := int32(duration.Seconds())
		input.DurationSeconds = &seconds
	}

	client := sts.NewFromConfig(f.cfg)
	output, err := client.AssumeRole(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("assume role: %w", err)
	}
	if output.Credentials == nil {
		return nil, errors.New("assume role returned empty credentials")
	}

	return output.Credentials, nil
}
