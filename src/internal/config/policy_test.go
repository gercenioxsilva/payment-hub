package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadPolicy(t *testing.T) {
	tmpDir := t.TempDir()
	policyPath := filepath.Join(tmpDir, "policy.yaml")
	policyYAML := `
version: "1"
defaults:
  timeouts:
    request_ms: 1500
  retries:
    max_attempts: 3
  fallback:
    max_provider_hops: 2
providers:
  pixmock:
    capabilities: [PIX]
    priority_weight: 10
routing_rules:
  - name: pix-only
    when:
      method: PIX
    providers: [pixmock]
`
	if err := os.WriteFile(policyPath, []byte(policyYAML), 0o600); err != nil {
		t.Fatalf("write policy: %v", err)
	}

	policy, err := LoadPolicy(policyPath)
	if err != nil {
		t.Fatalf("LoadPolicy error: %v", err)
	}

	if policy.Version != "1" {
		t.Fatalf("expected version 1, got %s", policy.Version)
	}
	if policy.Defaults.Timeouts.RequestMS != 1500 {
		t.Fatalf("expected request_ms 1500, got %d", policy.Defaults.Timeouts.RequestMS)
	}
	if policy.Defaults.Retries.MaxAttempts != 3 {
		t.Fatalf("expected max_attempts 3, got %d", policy.Defaults.Retries.MaxAttempts)
	}
	if policy.Providers["pixmock"].PriorityWeight != 10 {
		t.Fatalf("expected pixmock priority 10, got %d", policy.Providers["pixmock"].PriorityWeight)
	}
	if len(policy.RoutingRules) != 1 || policy.RoutingRules[0].When.Method != "PIX" {
		t.Fatalf("expected PIX routing rule, got %+v", policy.RoutingRules)
	}
}
