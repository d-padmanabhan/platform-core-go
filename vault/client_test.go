package vault

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteAndReadKVv2(t *testing.T) {
	t.Parallel()

	secrets := map[string]map[string]any{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Vault-Token") != "token-123" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		switch r.Method {
		case http.MethodPost:
			var payload map[string]map[string]any
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			secrets[r.URL.Path] = payload["data"]
			w.WriteHeader(http.StatusNoContent)
		case http.MethodGet:
			data, ok := secrets[r.URL.Path]
			if !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			response := map[string]any{
				"data": map[string]any{
					"data": data,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	client, err := New(server.URL, "token-123")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	want := map[string]any{
		"username": "svc-user",
		"password": "svc-pass",
	}
	if err := client.WriteKVv2(context.Background(), "secret", "team/app/credentials", want); err != nil {
		t.Fatalf("write kvv2: %v", err)
	}

	got, err := client.ReadKVv2(context.Background(), "secret", "team/app/credentials")
	if err != nil {
		t.Fatalf("read kvv2: %v", err)
	}

	if got["username"] != "svc-user" || got["password"] != "svc-pass" {
		t.Fatalf("unexpected secret data: %#v", got)
	}
}

func TestReadKVv2NotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	client, err := New(server.URL, "token-123")
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.ReadKVv2(context.Background(), "secret", "missing/path")
	if !errors.Is(err, ErrSecretNotFound) {
		t.Fatalf("expected ErrSecretNotFound, got: %v", err)
	}
}
