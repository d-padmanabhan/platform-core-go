package vault

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/d-padmanabhan/platform-core-go/internal/httpx"
)

const (
	envVaultAddr    = "VAULT_ADDR"
	envVaultToken   = "VAULT_TOKEN"
	envVaultTimeout = "VAULT_HTTP_TIMEOUT_SECONDS"
)

// ErrSecretNotFound indicates a requested secret path does not exist.
var ErrSecretNotFound = errors.New("vault secret not found")

// Config controls Vault client behavior.
type Config struct {
	Address    string
	Token      string
	Timeout    time.Duration
	HTTPClient *http.Client
}

// Option configures Client construction behavior.
type Option func(*Config)

// WithAddress overrides the Vault server address.
func WithAddress(address string) Option {
	return func(cfg *Config) {
		cfg.Address = strings.TrimRight(strings.TrimSpace(address), "/")
	}
}

// WithToken overrides the Vault token used for authentication.
func WithToken(token string) Option {
	return func(cfg *Config) {
		cfg.Token = strings.TrimSpace(token)
	}
}

// WithTimeout sets request timeout for the Vault client.
func WithTimeout(timeout time.Duration) Option {
	return func(cfg *Config) {
		cfg.Timeout = timeout
	}
}

// WithHTTPClient injects a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(cfg *Config) {
		cfg.HTTPClient = client
	}
}

// Client provides Vault KV v2 read/write operations.
type Client struct {
	address    string
	token      string
	httpClient *http.Client
}

// NewFromEnv creates a Vault client from environment variables.
func NewFromEnv(opts ...Option) (*Client, error) {
	timeoutSeconds := getenvInt(envVaultTimeout, int(httpx.DefaultTimeout.Seconds()))
	cfg := Config{
		Address: strings.TrimRight(strings.TrimSpace(os.Getenv(envVaultAddr)), "/"),
		Token:   strings.TrimSpace(os.Getenv(envVaultToken)),
		Timeout: time.Duration(timeoutSeconds) * time.Second,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return newClient(cfg)
}

// New creates a Vault client from explicit address and token values.
func New(address string, token string, opts ...Option) (*Client, error) {
	cfg := Config{
		Address: strings.TrimRight(strings.TrimSpace(address), "/"),
		Token:   strings.TrimSpace(token),
		Timeout: httpx.DefaultTimeout,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	return newClient(cfg)
}

func newClient(cfg Config) (*Client, error) {
	if cfg.Address == "" {
		return nil, fmt.Errorf("%s is required", envVaultAddr)
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("%s is required", envVaultToken)
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = httpx.DefaultTimeout
	}

	if cfg.HTTPClient == nil {
		cfg.HTTPClient = httpx.NewClient(cfg.Timeout)
	} else if cfg.HTTPClient.Timeout <= 0 {
		cfg.HTTPClient.Timeout = cfg.Timeout
	}

	return &Client{
		address:    cfg.Address,
		token:      cfg.Token,
		httpClient: cfg.HTTPClient,
	}, nil
}

// WriteKVv2 writes secret data to a KV v2 path.
func (c *Client) WriteKVv2(
	ctx context.Context,
	secretsEngine string,
	secretPath string,
	credentials map[string]any,
) error {
	vaultURL, err := c.kvV2URL(secretsEngine, secretPath)
	if err != nil {
		return err
	}

	payload := map[string]any{"data": credentials}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal vault write payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, vaultURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create vault write request: %w", err)
	}
	req.Header.Set("X-Vault-Token", c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("vault write request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	responseBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("vault write failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	return nil
}

// ReadKVv2 reads secret data from a KV v2 path.
func (c *Client) ReadKVv2(ctx context.Context, secretsEngine string, secretPath string) (map[string]any, error) {
	vaultURL, err := c.kvV2URL(secretsEngine, secretPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, vaultURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create vault read request: %w", err)
	}
	req.Header.Set("X-Vault-Token", c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vault read request failed: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	responseBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: %s", ErrSecretNotFound, secretPath)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("vault read failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var decoded struct {
		Data struct {
			Data map[string]any `json:"data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return nil, fmt.Errorf("decode vault read response: %w", err)
	}
	if decoded.Data.Data == nil {
		return nil, fmt.Errorf("vault response missing secret data at path: %s", secretPath)
	}

	return decoded.Data.Data, nil
}

func (c *Client) kvV2URL(secretsEngine string, secretPath string) (string, error) {
	mount := strings.Trim(strings.TrimSpace(secretsEngine), "/")
	path := strings.Trim(strings.TrimSpace(secretPath), "/")
	if mount == "" {
		return "", errors.New("secrets engine must not be empty")
	}
	if path == "" {
		return "", errors.New("secret path must not be empty")
	}

	return fmt.Sprintf("%s/v1/%s/data/%s", c.address, mount, path), nil
}

func getenvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
