package toolruntime

import "testing"

func TestEffectivePolicyGovernanceOverridesAuto(t *testing.T) {
	policy := EffectivePolicy("auto", GovernanceResult{Decision: "approve", Allowed: false})
	if policy != "approve" {
		t.Fatalf("policy = %q, want approve", policy)
	}
}

func TestEffectivePolicyDenyWins(t *testing.T) {
	policy := EffectivePolicy("auto", GovernanceResult{Decision: "deny", Allowed: false})
	if policy != "deny" {
		t.Fatalf("policy = %q, want deny", policy)
	}
}

func TestEffectivePolicyNotifyOverridesAuto(t *testing.T) {
	policy := EffectivePolicy("auto", GovernanceResult{Decision: "notify", Allowed: true})
	if policy != "notify" {
		t.Fatalf("policy = %q, want notify", policy)
	}
}
