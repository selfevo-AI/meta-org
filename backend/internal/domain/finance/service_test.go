package finance

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/selfevo-AI/meta-org/backend/internal/domain/project"
)

func TestSignAndVerifyPayload(t *testing.T) {
	body := []byte(`{"batch_id":"b1"}`)
	secret := "secret"
	signature := SignPayload(body, secret)
	if !VerifyPayload(body, signature, secret) {
		t.Fatalf("VerifyPayload returned false")
	}
	if VerifyPayload(body, signature, "other") {
		t.Fatalf("VerifyPayload accepted wrong secret")
	}
}

func TestBatchIdempotencyKeyStable(t *testing.T) {
	key1 := BatchIdempotencyKey("adapter-1", "2026-06-01", "2026-06-30", "CNY")
	key2 := BatchIdempotencyKey("adapter-1", "2026-06-01", "2026-06-30", "CNY")
	if key1 != key2 {
		t.Fatalf("keys differ: %q %q", key1, key2)
	}
}

func TestCreateExportBatchGeneratesStableIdempotencyKey(t *testing.T) {
	adapterID := uuid.New()
	repo := &fakeRepository{}
	svc := NewService(repo)

	batch, err := svc.CreateExportBatch(context.Background(), CreateExportBatchInput{
		AdapterID:   adapterID,
		PeriodStart: "2026-06-01",
		PeriodEnd:   "2026-06-30",
		Currency:    "cny",
	})
	if err != nil {
		t.Fatalf("CreateExportBatch returned error: %v", err)
	}

	want := BatchIdempotencyKey(adapterID.String(), "2026-06-01", "2026-06-30", "CNY")
	if repo.createdBatch.IdempotencyKey != want {
		t.Fatalf("idempotency key = %q, want %q", repo.createdBatch.IdempotencyKey, want)
	}
	if batch.Currency != "CNY" {
		t.Fatalf("currency = %q, want CNY", batch.Currency)
	}
}

func TestCreateExportBatchRejectsInvalidPeriod(t *testing.T) {
	svc := NewService(&fakeRepository{})
	_, err := svc.CreateExportBatch(context.Background(), CreateExportBatchInput{
		AdapterID:   uuid.New(),
		PeriodStart: "2026-06-30",
		PeriodEnd:   "2026-06-01",
	})
	if err == nil {
		t.Fatalf("CreateExportBatch accepted invalid period")
	}
}

type fakeRepository struct {
	createdBatch CreateExportBatchInput
}

func (f *fakeRepository) CreateAdapter(context.Context, CreateAdapterInput) (*FinanceAdapter, error) {
	return &FinanceAdapter{}, nil
}

func (f *fakeRepository) ListAdapters(context.Context, int) ([]FinanceAdapter, error) {
	return []FinanceAdapter{}, nil
}

func (f *fakeRepository) UpdateAdapter(context.Context, uuid.UUID, UpdateAdapterInput) (*FinanceAdapter, error) {
	return &FinanceAdapter{}, nil
}

func (f *fakeRepository) GetAdapterSecret(context.Context, uuid.UUID) (AdapterSecret, error) {
	return AdapterSecret{}, nil
}

func (f *fakeRepository) CreateExportBatch(_ context.Context, input CreateExportBatchInput) (*ExportBatch, error) {
	f.createdBatch = input
	return &ExportBatch{
		ID:             uuid.New(),
		AdapterID:      input.AdapterID,
		PeriodStart:    input.periodStartTime,
		PeriodEnd:      input.periodEndTime,
		Status:         "ready",
		Currency:       input.Currency,
		IdempotencyKey: input.IdempotencyKey,
		Lines:          []ExportLine{},
	}, nil
}

func (f *fakeRepository) ListExportBatches(context.Context, int) ([]ExportBatch, error) {
	return []ExportBatch{}, nil
}

func (f *fakeRepository) GetExportBatch(context.Context, uuid.UUID) (*ExportBatch, error) {
	return &ExportBatch{}, nil
}

func (f *fakeRepository) UpdateExportBatchStatus(context.Context, uuid.UUID, UpdateExportBatchStatusInput) (*ExportBatch, error) {
	return &ExportBatch{}, nil
}

func (f *fakeRepository) RecordWebhookEvent(context.Context, RecordWebhookEventInput) (*WebhookEvent, error) {
	return &WebhookEvent{}, nil
}

func (f *fakeRepository) UpdateExportLineStatus(context.Context, uuid.UUID, UpdateExportLineStatusInput) (*ExportLine, error) {
	return &ExportLine{}, nil
}

func (f *fakeRepository) LinkProjectCostEntry(context.Context, uuid.UUID, uuid.UUID) error {
	return nil
}

func (f *fakeRepository) ListReconciliation(context.Context, int) ([]ReconciliationItem, error) {
	return []ReconciliationItem{}, nil
}

func (f *fakeRepository) CreateImportBatch(context.Context, *uuid.UUID, string, string, int, map[string]any) (*ImportBatch, error) {
	return &ImportBatch{}, nil
}

func (f *fakeRepository) CompleteImportBatch(context.Context, uuid.UUID, int, int) (*ImportBatch, error) {
	return &ImportBatch{}, nil
}

func (f *fakeRepository) ListImportBatches(context.Context, int) ([]ImportBatch, error) {
	return []ImportBatch{}, nil
}

func (f *fakeRepository) ListImportRecords(context.Context, int) ([]ImportRecord, error) {
	return []ImportRecord{}, nil
}

func (f *fakeRepository) CreateImportedExpense(context.Context, uuid.UUID, uuid.UUID, map[string]any, FinanceExpenseInput, time.Time, financeExpenseDates) (*ImportRecord, error) {
	return &ImportRecord{}, nil
}

func (f *fakeRepository) CreatePayable(context.Context, CreatePayableInput, financeExpenseDates) (*Payable, error) {
	return &Payable{}, nil
}

func (f *fakeRepository) ListPayables(context.Context, int) ([]Payable, error) {
	return []Payable{}, nil
}

func (f *fakeRepository) CreatePayment(context.Context, CreatePaymentInput, *time.Time) (*Payment, error) {
	return &Payment{}, nil
}

func (f *fakeRepository) ListPayments(context.Context, int) ([]Payment, error) {
	return []Payment{}, nil
}

func (f *fakeRepository) AllocatePayment(context.Context, uuid.UUID, AllocatePaymentInput) (*PaymentAllocation, error) {
	return &PaymentAllocation{}, nil
}

type fakeCostPoster struct{}

func (fakeCostPoster) CreateCostEntryFromAIUsage(context.Context, uuid.UUID, project.CreateCostEntryInput) (*project.CostEntry, error) {
	return &project.CostEntry{}, nil
}
