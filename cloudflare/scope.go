package cloudflare

import (
	"fmt"
	"net/url"
	"strings"
)

// ScopeKind identifies the Cloudflare resource scope segment used in URLs.
type ScopeKind string

const (
	// ScopeAccounts maps to the /accounts/{account_id}/... API paths.
	ScopeAccounts ScopeKind = "accounts"
	// ScopeZones maps to the /zones/{zone_id}/... API paths.
	ScopeZones ScopeKind = "zones"
)

// Scope identifies whether an operation is account- or zone-scoped.
type Scope struct {
	Kind ScopeKind
	ID   string
}

// AccountScope creates an account-scoped descriptor.
func AccountScope(accountID string) Scope {
	return Scope{
		Kind: ScopeAccounts,
		ID:   strings.TrimSpace(accountID),
	}
}

// ZoneScope creates a zone-scoped descriptor.
func ZoneScope(zoneID string) Scope {
	return Scope{
		Kind: ScopeZones,
		ID:   strings.TrimSpace(zoneID),
	}
}

// PathPrefix returns the resource path prefix for this scope.
func (s Scope) PathPrefix() (string, error) {
	switch s.Kind {
	case ScopeAccounts, ScopeZones:
		// valid kind
	default:
		return "", fmt.Errorf("unsupported scope kind: %q", s.Kind)
	}

	if s.ID == "" {
		return "", fmt.Errorf("scope ID must not be empty for %q", s.Kind)
	}

	return fmt.Sprintf("%s/%s", s.Kind, url.PathEscape(s.ID)), nil
}
