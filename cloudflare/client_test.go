package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestZoneIDByName(t *testing.T) {
	t.Parallel()

	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("name") != "acme.com" {
			t.Fatalf("expected zone filter query parameter")
		}

		response := map[string]any{
			"success": true,
			"result": []map[string]any{
				{
					"id":   "zone-123",
					"name": "acme.com",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := New(
		"token",
		WithBaseURL(server.URL),
		WithTimeout(5*time.Second),
		WithRetries(1, time.Millisecond, 2*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	zoneID, err := client.ZoneIDByName(context.Background(), "acme.com")
	if err != nil {
		t.Fatalf("zone id by name: %v", err)
	}
	if zoneID != "zone-123" {
		t.Fatalf("unexpected zone id: got=%q want=%q", zoneID, "zone-123")
	}
	if calls != 1 {
		t.Fatalf("unexpected call count: got=%d want=1", calls)
	}
}

func TestZoneIDByName_NotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		response := map[string]any{
			"success": true,
			"result":  []map[string]any{},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client, err := New("token", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.ZoneIDByName(context.Background(), "missing.acme.com")
	if !errors.Is(err, ErrZoneNotFound) {
		t.Fatalf("expected ErrZoneNotFound, got: %v", err)
	}
}

func TestDo_RetriesOn429(t *testing.T) {
	t.Parallel()

	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":10000,"message":"rate limited"}]}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"result":{"ok":true}}`))
	}))
	defer server.Close()

	client, err := New(
		"token",
		WithBaseURL(server.URL),
		WithTimeout(5*time.Second),
		WithRetries(2, time.Millisecond, 2*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	var out map[string]any
	if err := client.Do(context.Background(), http.MethodGet, "/zones", nil, nil, &out); err != nil {
		t.Fatalf("do request: %v", err)
	}

	if calls != 2 {
		t.Fatalf("expected 2 calls (one retry), got: %d", calls)
	}
	if ok, _ := out["ok"].(bool); !ok {
		t.Fatalf("unexpected response payload: %#v", out)
	}
}

func TestListZones_Paginates(t *testing.T) {
	t.Parallel()

	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++

		page := r.URL.Query().Get("page")
		w.Header().Set("Content-Type", "application/json")
		switch page {
		case "1":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"result": []map[string]any{
					{"id": "zone-1", "name": "one.acme.com"},
				},
				"result_info": map[string]any{
					"page":        1,
					"per_page":    1,
					"total_pages": 2,
					"count":       1,
					"total_count": 2,
				},
			})
		case "2":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"result": []map[string]any{
					{"id": "zone-2", "name": "two.acme.com"},
				},
				"result_info": map[string]any{
					"page":        2,
					"per_page":    1,
					"total_pages": 2,
					"count":       1,
					"total_count": 2,
				},
			})
		default:
			t.Fatalf("unexpected page query value: %q", page)
		}
	}))
	defer server.Close()

	client, err := New("token", WithBaseURL(server.URL), WithTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	zones, err := client.ListZones(context.Background())
	if err != nil {
		t.Fatalf("list zones: %v", err)
	}

	if calls != 2 {
		t.Fatalf("expected two paginated calls, got: %d", calls)
	}
	if len(zones) != 2 {
		t.Fatalf("expected 2 zones, got: %d", len(zones))
	}
	if zones[0].ID != "zone-1" || zones[1].ID != "zone-2" {
		t.Fatalf("unexpected zones payload: %#v", zones)
	}
}

func TestDo_DoesNotRetryUnsafeMethodByDefault(t *testing.T) {
	t.Parallel()

	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":10013,"message":"temporary failure"}]}`))
	}))
	defer server.Close()

	client, err := New(
		"token",
		WithBaseURL(server.URL),
		WithTimeout(5*time.Second),
		WithRetries(3, time.Millisecond, 2*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	err = client.Do(
		context.Background(),
		http.MethodPost,
		"/accounts/acc-1/access/apps",
		nil,
		map[string]any{"name": "app-1"},
		nil,
	)
	if err == nil {
		t.Fatalf("expected error for POST request")
	}

	var statusErr *HTTPStatusError
	if !errors.As(err, &statusErr) {
		t.Fatalf("expected HTTPStatusError, got: %T", err)
	}

	if calls != 1 {
		t.Fatalf("expected single call for unsafe method default, got: %d", calls)
	}
}

func TestDoWithOptions_RetriesUnsafeMethodWhenEnabled(t *testing.T) {
	t.Parallel()

	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"success":false,"errors":[{"code":10013,"message":"temporary failure"}]}`))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"result":{"id":"app-1"}}`))
	}))
	defer server.Close()

	client, err := New(
		"token",
		WithBaseURL(server.URL),
		WithTimeout(5*time.Second),
		WithRetries(2, time.Millisecond, 2*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	var out map[string]any
	err = client.DoWithOptions(
		context.Background(),
		http.MethodPost,
		"/accounts/acc-1/access/apps",
		nil,
		map[string]any{"name": "app-1"},
		&out,
		WithRetryUnsafeMethods(),
	)
	if err != nil {
		t.Fatalf("expected retry-enabled POST to succeed: %v", err)
	}

	if calls != 2 {
		t.Fatalf("expected 2 calls with unsafe retries enabled, got: %d", calls)
	}
}

func TestDoWithOptions_RetriesUnsafeMethodOnTransportErrorWhenEnabled(t *testing.T) {
	t.Parallel()

	var calls int
	httpClient := &http.Client{
		Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			calls++
			if calls == 1 {
				return nil, errors.New("temporary transport failure")
			}

			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(
					`{"success":true,"result":{"id":"app-1"}}`,
				)),
			}, nil
		}),
	}

	client, err := New(
		"token",
		WithBaseURL("https://api.cloudflare.com/client/v4"),
		WithHTTPClient(httpClient),
		WithRetries(2, time.Millisecond, 2*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	var out map[string]any
	err = client.DoWithOptions(
		context.Background(),
		http.MethodPost,
		"/accounts/acc-1/access/apps",
		nil,
		map[string]any{"name": "app-1"},
		&out,
		WithRetryUnsafeMethods(),
	)
	if err != nil {
		t.Fatalf("expected retry-enabled POST to succeed after transport error: %v", err)
	}

	if calls != 2 {
		t.Fatalf("expected 2 calls with unsafe retries enabled, got: %d", calls)
	}
	if out["id"] != "app-1" {
		t.Fatalf("unexpected response payload: %#v", out)
	}
}
