package assistant

import "testing"

func TestContextPackageAttentionCoreBudget(t *testing.T) {
	pkg := ContextPackage{
		AttentionCore: []ContextItem{
			{EntityKey: "project", FieldKey: "status", Value: "active", Weight: 10, EstimatedTokens: 20},
		},
		SupportingContext: []ContextItem{
			{EntityKey: "project", FieldKey: "description", Value: "long", Weight: 3, EstimatedTokens: 80},
		},
		TokenBudget: 100,
	}

	if pkg.AttentionCoreTokens() != 20 {
		t.Fatalf("attention core tokens = %d, want 20", pkg.AttentionCoreTokens())
	}
	if pkg.TotalEstimatedTokens() != 100 {
		t.Fatalf("total tokens = %d, want 100", pkg.TotalEstimatedTokens())
	}
}

func TestRuleEvaluatorMovesFinanceConflictToSignal(t *testing.T) {
	evaluator := NewContextRuleEvaluator(ContextRuleEvaluatorConfig{AttentionCoreRatio: 0.4})
	items := []ContextItem{
		{EntityKey: "cost_ledger_entry", FieldKey: "amount", Value: 100, Weight: 9, EstimatedTokens: 10, ValidationState: "finance_conflict"},
		{EntityKey: "project", FieldKey: "status", Value: "active", Weight: 8, EstimatedTokens: 10, ValidationState: "verified"},
	}

	pkg := evaluator.BuildPackage(ContextRuleEvaluationInput{Items: items, TokenBudget: 100})

	if len(pkg.AttentionCore) != 1 || pkg.AttentionCore[0].FieldKey != "status" {
		t.Fatalf("attention core = %#v, want only status", pkg.AttentionCore)
	}
	if len(pkg.RiskAndSignals) != 1 || pkg.RiskAndSignals[0].FieldKey != "amount" {
		t.Fatalf("signals = %#v, want amount", pkg.RiskAndSignals)
	}
}

func TestRuleEvaluatorEnforcesAttentionCoreBudget(t *testing.T) {
	evaluator := NewContextRuleEvaluator(ContextRuleEvaluatorConfig{AttentionCoreRatio: 0.3})
	items := []ContextItem{
		{EntityKey: "project", FieldKey: "a", Weight: 10, EstimatedTokens: 20, ValidationState: "verified"},
		{EntityKey: "project", FieldKey: "b", Weight: 9, EstimatedTokens: 20, ValidationState: "verified"},
		{EntityKey: "project", FieldKey: "c", Weight: 8, EstimatedTokens: 20, ValidationState: "verified"},
	}

	pkg := evaluator.BuildPackage(ContextRuleEvaluationInput{Items: items, TokenBudget: 100})

	if len(pkg.AttentionCore) != 1 || pkg.AttentionCore[0].FieldKey != "a" {
		t.Fatalf("attention core = %#v, want only highest-weight field a", pkg.AttentionCore)
	}
	if len(pkg.SupportingContext) != 2 {
		t.Fatalf("supporting context len = %d, want 2", len(pkg.SupportingContext))
	}
}
