package metaorg

import (
	"context"
	"testing"
	"time"
)

type fakeRepository struct{}

func (fakeRepository) Overview(context.Context) (Overview, error) {
	return Overview{
		GeneratedAt: time.Unix(100, 0).UTC(),
		Health: HealthSummary{
			OpenRequirements: 3,
			ActiveProjects:   2,
			ActiveAgents:     9,
			PendingApprovals: 4,
			UnexportedCost:   12.5,
			Currency:         "CNY",
		},
		Risks: []RiskItem{{ID: "risk-1", Title: "High-risk tool pending approval", Severity: "high", Source: "toolruntime"}},
	}, nil
}

func (fakeRepository) Inbox(context.Context, InboxFilter) ([]InboxItem, error) {
	return []InboxItem{{ID: "approval-1", Type: "tool_approval", Title: "Approve member matching", Status: "pending", Priority: "high"}}, nil
}

func TestServiceOverviewReturnsRepositoryData(t *testing.T) {
	svc := NewService(fakeRepository{})
	overview, err := svc.GetOverview(context.Background())
	if err != nil {
		t.Fatalf("GetOverview returned error: %v", err)
	}
	if overview.Health.ActiveAgents != 9 {
		t.Fatalf("ActiveAgents = %d, want 9", overview.Health.ActiveAgents)
	}
	if len(overview.Risks) != 1 {
		t.Fatalf("Risks length = %d, want 1", len(overview.Risks))
	}
}

func TestServiceInboxNormalizesLimit(t *testing.T) {
	svc := NewService(fakeRepository{})
	items, err := svc.GetInbox(context.Background(), InboxFilter{Limit: 0})
	if err != nil {
		t.Fatalf("GetInbox returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items length = %d, want 1", len(items))
	}
}
