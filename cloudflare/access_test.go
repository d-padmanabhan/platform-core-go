package cloudflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAccessCreateIdentityProvider(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/accounts/acc-1/access/identity_providers" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"result": map[string]any{
				"id": "idp-1",
			},
		})
	}))
	defer server.Close()

	client, err := New("token", WithBaseURL(server.URL), WithTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	var out map[string]any
	err = client.Access().CreateIdentityProvider(
		context.Background(),
		"acc-1",
		map[string]any{"name": "okta", "type": "okta"},
		&out,
		WithRetryUnsafeMethods(),
	)
	if err != nil {
		t.Fatalf("create identity provider: %v", err)
	}

	if out["id"] != "idp-1" {
		t.Fatalf("unexpected response payload: %#v", out)
	}
}

func TestAccessCreateApplicationZoneScope(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/zones/zone-1/access/apps" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"result": map[string]any{
				"id": "app-1",
			},
		})
	}))
	defer server.Close()

	client, err := New("token", WithBaseURL(server.URL), WithTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	var out map[string]any
	err = client.Access().CreateApplication(
		context.Background(),
		ZoneScope("zone-1"),
		map[string]any{"name": "Admin Site", "type": "self_hosted"},
		&out,
		WithRetryUnsafeMethods(),
	)
	if err != nil {
		t.Fatalf("create application: %v", err)
	}

	if out["id"] != "app-1" {
		t.Fatalf("unexpected response payload: %#v", out)
	}
}

func TestAccessCreateReusablePolicy(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/accounts/acc-1/access/policies" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"result": map[string]any{
				"id": "policy-1",
			},
		})
	}))
	defer server.Close()

	client, err := New("token", WithBaseURL(server.URL), WithTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	var out map[string]any
	err = client.Access().CreateReusablePolicy(
		context.Background(),
		"acc-1",
		map[string]any{"name": "allow-engineering", "decision": "allow"},
		&out,
		WithRetryUnsafeMethods(),
	)
	if err != nil {
		t.Fatalf("create reusable policy: %v", err)
	}

	if out["id"] != "policy-1" {
		t.Fatalf("unexpected response payload: %#v", out)
	}
}

func TestAccessCreateApplicationPolicy(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/accounts/acc-1/access/apps/app-1/policies" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"result": map[string]any{
				"id": "policy-1",
			},
		})
	}))
	defer server.Close()

	client, err := New("token", WithBaseURL(server.URL), WithTimeout(5*time.Second))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	var out map[string]any
	err = client.Access().CreateApplicationPolicy(
		context.Background(),
		AccountScope("acc-1"),
		"app-1",
		map[string]any{"name": "allow-admins", "decision": "allow"},
		&out,
		WithRetryUnsafeMethods(),
	)
	if err != nil {
		t.Fatalf("create application policy: %v", err)
	}

	if out["id"] != "policy-1" {
		t.Fatalf("unexpected response payload: %#v", out)
	}
}

func TestAccessDoRejectsInvalidScope(t *testing.T) {
	t.Parallel()

	client, err := New("token", WithBaseURL("https://api.cloudflare.com/client/v4"))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	err = client.Access().Do(
		context.Background(),
		Scope{Kind: "unsupported", ID: "x"},
		http.MethodGet,
		"/access/apps",
		nil,
		nil,
		nil,
	)
	if err == nil {
		t.Fatalf("expected invalid scope error")
	}
}
