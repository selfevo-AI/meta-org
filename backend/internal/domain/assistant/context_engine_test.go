package assistant

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestVerifiedContextEngineBuildsAttentionCoreFromWorkRecords(t *testing.T) {
	sessionID := uuid.New()
	resolver := &fakeContextResolver{
		result: WorkRecordContext{
			ModuleKey: "project",
			Records: []WorkRecord{
				{ID: uuid.New().String(), Type: "project", Title: "Launch", Status: "active"},
			},
		},
	}
	engine := NewVerifiedContextEngine(VerifiedContextEngineConfig{
		Resolver:  resolver,
		Evaluator: NewContextRuleEvaluator(ContextRuleEvaluatorConfig{AttentionCoreRatio: 0.5}),
	})

	pkg, err := engine.BuildContextPackage(context.Background(), ContextRequest{
		SessionID:   sessionID,
		ActorID:     uuid.New(),
		ActorType:   "internal_human",
		ModuleKey:   "project",
		TargetType:  "project",
		TokenBudget: 200,
	})
	if err != nil {
		t.Fatalf("BuildContextPackage returned error: %v", err)
	}
	if pkg.ID == uuid.Nil {
		t.Fatalf("context package id is nil")
	}
	if len(pkg.AttentionCore) == 0 {
		t.Fatalf("attention core is empty")
	}
	if pkg.AttentionCore[0].EntityKey != "project" {
		t.Fatalf("entity = %s, want project", pkg.AttentionCore[0].EntityKey)
	}
	if pkg.Provenance["source"] != "compatibility_resolver" {
		t.Fatalf("source = %v, want compatibility_resolver", pkg.Provenance["source"])
	}
}
