package cloudflare

import "testing"

func TestScopePathPrefix(t *testing.T) {
	t.Parallel()

	accountPrefix, err := AccountScope("acc-1").PathPrefix()
	if err != nil {
		t.Fatalf("account scope path prefix: %v", err)
	}
	if accountPrefix != "accounts/acc-1" {
		t.Fatalf("unexpected account scope prefix: %q", accountPrefix)
	}

	zonePrefix, err := ZoneScope("zone-1").PathPrefix()
	if err != nil {
		t.Fatalf("zone scope path prefix: %v", err)
	}
	if zonePrefix != "zones/zone-1" {
		t.Fatalf("unexpected zone scope prefix: %q", zonePrefix)
	}
}

func TestScopePathPrefixValidation(t *testing.T) {
	t.Parallel()

	if _, err := (Scope{Kind: ScopeAccounts, ID: ""}).PathPrefix(); err == nil {
		t.Fatalf("expected empty scope id validation error")
	}

	if _, err := (Scope{Kind: "unsupported", ID: "x"}).PathPrefix(); err == nil {
		t.Fatalf("expected unsupported scope kind validation error")
	}
}
