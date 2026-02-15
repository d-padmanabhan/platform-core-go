package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

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
