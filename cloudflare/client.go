package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/d-padmanabhan/platform-core-go/internal/httpx"
)

const (
	defaultBaseURL           = "https://api.cloudflare.com/client/v4"
	defaultTokenEnv          = "CLOUDFLARE_API_TOKEN"
	defaultMaxRetriesEnv     = "CLOUDFLARE_HTTP_MAX_RETRIES"
	defaultRetryBaseDelayEnv = "CLOUDFLARE_HTTP_RETRY_BASE_DELAY_SECONDS"
	defaultRetryMaxDelayEnv  = "CLOUDFLARE_HTTP_RETRY_MAX_DELAY_SECONDS"
	defaultMaxRetries        = 3
	defaultRetryBaseDelay    = 1 * time.Second
	defaultRetryMaxDelay     = 30 * time.Second
)

// ErrZoneNotFound indicates no matching zone was returned by Cloudflare.
var ErrZoneNotFound = errors.New("cloudflare zone not found")

// Config controls Cloudflare client behavior.
type Config struct {
	BaseURL        string
	Timeout        time.Duration
	MaxRetries     int
	RetryBaseDelay time.Duration
	RetryMaxDelay  time.Duration
	HTTPClient     *http.Client
}

// Option configures Client construction behavior.
type Option func(*Config)

// WithBaseURL overrides the default Cloudflare API base URL.
func WithBaseURL(baseURL string) Option {
	return func(cfg *Config) {
		cfg.BaseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	}
}

// WithHTTPClient injects a custom HTTP client.
func WithHTTPClient(client *http.Client) Option {
	return func(cfg *Config) {
		cfg.HTTPClient = client
	}
}

// WithTimeout sets request timeout for the Cloudflare client.
func WithTimeout(timeout time.Duration) Option {
	return func(cfg *Config) {
		cfg.Timeout = timeout
	}
}

// WithRetries sets retry count and backoff parameters.
func WithRetries(maxRetries int, baseDelay, maxDelay time.Duration) Option {
	return func(cfg *Config) {
		cfg.MaxRetries = maxRetries
		cfg.RetryBaseDelay = baseDelay
		cfg.RetryMaxDelay = maxDelay
	}
}

func defaultConfig() Config {
	maxRetries := getenvInt(defaultMaxRetriesEnv, defaultMaxRetries)
	baseDelaySeconds := getenvFloat(defaultRetryBaseDelayEnv, defaultRetryBaseDelay.Seconds())
	maxDelaySeconds := getenvFloat(defaultRetryMaxDelayEnv, defaultRetryMaxDelay.Seconds())

	return Config{
		BaseURL:        defaultBaseURL,
		Timeout:        httpx.DefaultTimeout,
		MaxRetries:     maxRetries,
		RetryBaseDelay: time.Duration(baseDelaySeconds * float64(time.Second)),
		RetryMaxDelay:  time.Duration(maxDelaySeconds * float64(time.Second)),
	}
}

// Client is a retry-aware Cloudflare API client.
type Client struct {
	token string
	cfg   Config
}

// NewFromEnv creates a Cloudflare client using CLOUDFLARE_API_TOKEN.
func NewFromEnv(opts ...Option) (*Client, error) {
	token := strings.TrimSpace(os.Getenv(defaultTokenEnv))
	if token == "" {
		return nil, fmt.Errorf("%s is required", defaultTokenEnv)
	}
	return New(token, opts...)
}

// New creates a Cloudflare client from an explicit API token.
func New(token string, opts ...Option) (*Client, error) {
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("cloudflare API token must be provided")
	}

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultBaseURL
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = httpx.DefaultTimeout
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	if cfg.RetryBaseDelay <= 0 {
		cfg.RetryBaseDelay = defaultRetryBaseDelay
	}
	if cfg.RetryMaxDelay <= 0 {
		cfg.RetryMaxDelay = defaultRetryMaxDelay
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = httpx.NewClient(cfg.Timeout)
	} else if cfg.HTTPClient.Timeout <= 0 {
		cfg.HTTPClient.Timeout = cfg.Timeout
	}

	return &Client{
		token: token,
		cfg:   cfg,
	}, nil
}

// HTTPStatusError captures non-2xx responses returned by Cloudflare.
type HTTPStatusError struct {
	StatusCode int
	Body       string
}

// Error implements the error interface.
func (e *HTTPStatusError) Error() string {
	if e.Body == "" {
		return fmt.Sprintf("cloudflare request failed with status %d", e.StatusCode)
	}
	return fmt.Sprintf("cloudflare request failed with status %d: %s", e.StatusCode, e.Body)
}

// Do executes a Cloudflare API request and unmarshals result into out.
func (c *Client) Do(
	ctx context.Context,
	method string,
	endpoint string,
	params url.Values,
	requestBody any,
	out any,
) error {
	targetURL, err := c.buildURL(endpoint, params)
	if err != nil {
		return err
	}

	var payload []byte
	if requestBody != nil {
		payload, err = json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
	}

	for attempt := 0; ; attempt++ {
		req, reqErr := c.newRequest(ctx, method, targetURL, payload)
		if reqErr != nil {
			return reqErr
		}

		resp, doErr := c.cfg.HTTPClient.Do(req)
		if doErr != nil {
			if attempt >= c.cfg.MaxRetries {
				return fmt.Errorf("cloudflare request failed after retries: %w", doErr)
			}
			delay := httpx.ExponentialBackoffDelay(
				attempt,
				c.cfg.RetryBaseDelay,
				c.cfg.RetryMaxDelay,
				true,
				rand.Float64(),
			)
			if sleepErr := httpx.SleepContext(ctx, delay); sleepErr != nil {
				return sleepErr
			}
			continue
		}

		bodyBytes, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return fmt.Errorf("read cloudflare response body: %w", readErr)
		}

		if shouldRetryStatus(resp.StatusCode) && attempt < c.cfg.MaxRetries {
			delay := c.retryDelay(attempt, resp.Header.Get("Retry-After"))
			if sleepErr := httpx.SleepContext(ctx, delay); sleepErr != nil {
				return sleepErr
			}
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return &HTTPStatusError{
				StatusCode: resp.StatusCode,
				Body:       string(bodyBytes),
			}
		}

		var env envelope
		if err := json.Unmarshal(bodyBytes, &env); err != nil {
			return fmt.Errorf("decode cloudflare envelope: %w", err)
		}

		if !env.Success {
			return fmt.Errorf("cloudflare API returned unsuccessful response: %s", formatAPIErrors(env.Errors))
		}

		if out == nil || len(env.Result) == 0 || string(env.Result) == "null" {
			return nil
		}

		if err := json.Unmarshal(env.Result, out); err != nil {
			return fmt.Errorf("decode cloudflare result: %w", err)
		}

		return nil
	}
}

// ListZones lists zones visible to the authenticated token.
func (c *Client) ListZones(ctx context.Context) ([]Zone, error) {
	var zones []Zone
	if err := c.Do(ctx, http.MethodGet, "/zones", nil, nil, &zones); err != nil {
		return nil, err
	}
	return zones, nil
}

// ZoneIDByName resolves a zone name to its Cloudflare zone ID.
func (c *Client) ZoneIDByName(ctx context.Context, zoneName string) (string, error) {
	if strings.TrimSpace(zoneName) == "" {
		return "", errors.New("zone name must not be empty")
	}

	var zones []Zone
	params := url.Values{}
	params.Set("name", zoneName)

	if err := c.Do(ctx, http.MethodGet, "/zones", params, nil, &zones); err != nil {
		return "", err
	}
	if len(zones) == 0 {
		return "", fmt.Errorf("%w: %s", ErrZoneNotFound, zoneName)
	}

	return zones[0].ID, nil
}

func (c *Client) buildURL(endpoint string, params url.Values) (string, error) {
	base, err := url.Parse(strings.TrimRight(c.cfg.BaseURL, "/"))
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	cleanEndpoint := endpoint
	if !strings.HasPrefix(cleanEndpoint, "/") {
		cleanEndpoint = "/" + cleanEndpoint
	}

	base.Path = strings.TrimRight(base.Path, "/") + cleanEndpoint
	if params != nil {
		base.RawQuery = params.Encode()
	}

	return base.String(), nil
}

func (c *Client) newRequest(ctx context.Context, method, targetURL string, payload []byte) (*http.Request, error) {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, targetURL, body)
	if err != nil {
		return nil, fmt.Errorf("create cloudflare request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

func (c *Client) retryDelay(attempt int, retryAfterHeader string) time.Duration {
	if delay, ok := parseRetryAfter(retryAfterHeader); ok {
		return delay
	}

	return httpx.ExponentialBackoffDelay(
		attempt,
		c.cfg.RetryBaseDelay,
		c.cfg.RetryMaxDelay,
		true,
		rand.Float64(),
	)
}

func shouldRetryStatus(statusCode int) bool {
	return statusCode == http.StatusRequestTimeout ||
		statusCode == http.StatusTooManyRequests ||
		(statusCode >= 500 && statusCode <= 599)
}

func parseRetryAfter(value string) (time.Duration, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, false
	}

	if seconds, err := strconv.Atoi(trimmed); err == nil {
		if seconds <= 0 {
			return 0, true
		}
		return time.Duration(seconds) * time.Second, true
	}

	parsedTime, err := http.ParseTime(trimmed)
	if err != nil {
		return 0, false
	}

	delay := time.Until(parsedTime)
	if delay < 0 {
		return 0, true
	}
	return delay, true
}

func formatAPIErrors(items []APIErrorItem) string {
	if len(items) == 0 {
		return "unknown API error"
	}

	parts := make([]string, 0, len(items))
	for _, item := range items {
		if item.Code == 0 {
			parts = append(parts, item.Message)
			continue
		}
		parts = append(parts, fmt.Sprintf("%d:%s", item.Code, item.Message))
	}
	return strings.Join(parts, ", ")
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

func getenvFloat(key string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
