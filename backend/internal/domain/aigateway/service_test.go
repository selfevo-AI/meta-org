package aigateway

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type fakeGatewayRepo struct {
	target       ResolvedModel
	recorded     bool
	completed    bool
	failed       bool
	ledgerCount  int
	lastLedger   CreateUsageLedgerInput
	lastComplete CompleteInvocationInput
}

func newFakeGatewayRepo() *fakeGatewayRepo {
	return &fakeGatewayRepo{target: ResolvedModel{
		ProviderID:   uuid.New(),
		ModelID:      uuid.New(),
		ProviderType: ProviderOpenAI,
		Model:        "gpt-test",
		Price:        Price{InputPer1K: 0.01, OutputPer1K: 0.03},
		Currency:     "CNY",
	}}
}

func TestServiceInvokeRecordsUsage(t *testing.T) {
	repo := newFakeGatewayRepo()
	adapter := fakeAdapter{resp: ProviderResponse{Content: "ok", Usage: TokenUsage{InputTokens: 10, OutputTokens: 20}}}
	svc := NewService(repo, AdapterRegistry{ProviderOpenAI: adapter})

	resp, err := svc.Invoke(context.Background(), InvokeInput{
		ProviderType: ProviderOpenAI,
		Model:        "gpt-test",
		Messages:     []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatalf("Invoke returned error: %v", err)
	}
	if resp.Content != "ok" {
		t.Fatalf("content = %q, want ok", resp.Content)
	}
	if !repo.recorded {
		t.Fatalf("invocation was not recorded")
	}
	if !repo.completed {
		t.Fatalf("invocation was not completed")
	}
	if repo.ledgerCount != 1 {
		t.Fatalf("ledger count = %d, want 1", repo.ledgerCount)
	}
	if repo.lastLedger.Amount != 0.0007 {
		t.Fatalf("ledger amount = %.8f, want 0.0007", repo.lastLedger.Amount)
	}
}

func TestServiceInvokeRecordsFailedUsage(t *testing.T) {
	repo := newFakeGatewayRepo()
	adapter := fakeAdapter{err: errors.New("provider down")}
	svc := NewService(repo, AdapterRegistry{ProviderOpenAI: adapter})

	_, err := svc.Invoke(context.Background(), InvokeInput{
		ProviderType: ProviderOpenAI,
		Model:        "gpt-test",
		Messages:     []Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatalf("Invoke returned nil error")
	}
	if !repo.recorded {
		t.Fatalf("invocation was not recorded")
	}
	if !repo.failed {
		t.Fatalf("invocation was not marked failed")
	}
	if repo.ledgerCount != 1 {
		t.Fatalf("ledger count = %d, want 1", repo.ledgerCount)
	}
	if repo.lastLedger.Amount != 0 {
		t.Fatalf("failed ledger amount = %.8f, want 0", repo.lastLedger.Amount)
	}
}

func (f *fakeGatewayRepo) ResolveInvocationTarget(context.Context, InvokeInput) (ResolvedModel, error) {
	return f.target, nil
}

func (f *fakeGatewayRepo) CreateInvocation(context.Context, CreateInvocationInput) (*Invocation, error) {
	f.recorded = true
	return &Invocation{ID: uuid.New(), ProviderID: f.target.ProviderID, ModelID: f.target.ModelID, Status: StatusStarted}, nil
}

func (f *fakeGatewayRepo) CompleteInvocation(_ context.Context, id uuid.UUID, input CompleteInvocationInput) error {
	f.completed = true
	f.lastComplete = input
	return nil
}

func (f *fakeGatewayRepo) FailInvocation(context.Context, uuid.UUID, FailInvocationInput) error {
	f.failed = true
	return nil
}

func (f *fakeGatewayRepo) CreateUsageLedger(_ context.Context, input CreateUsageLedgerInput) error {
	f.ledgerCount++
	f.lastLedger = input
	return nil
}

type fakeAdapter struct {
	resp ProviderResponse
	err  error
}

func (a fakeAdapter) Invoke(context.Context, ProviderRequest) (*ProviderResponse, error) {
	if a.err != nil {
		return nil, a.err
	}
	return &a.resp, nil
}

func (a fakeAdapter) Stream(context.Context, ProviderRequest) (<-chan StreamEvent, error) {
	ch := make(chan StreamEvent)
	close(ch)
	return ch, a.err
}
