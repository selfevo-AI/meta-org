package finance

import (
	"time"

	"github.com/google/uuid"
)

const (
	AuthHMAC   = "hmac"
	AuthBearer = "bearer"

	AdapterActive   = "active"
	AdapterDisabled = "disabled"
	AdapterError    = "error"

	BatchDraft        = "draft"
	BatchReady        = "ready"
	BatchExporting    = "exporting"
	BatchExported     = "exported"
	BatchAcknowledged = "acknowledged"
	BatchPosted       = "posted"
	BatchReconciled   = "reconciled"
	BatchFailed       = "failed"
	BatchCancelled    = "cancelled"
)

type FinanceAdapter struct {
	ID           uuid.UUID      `json:"id"`
	Name         string         `json:"name"`
	EndpointURL  string         `json:"endpoint_url"`
	AuthType     string         `json:"auth_type"`
	MaskedSecret string         `json:"masked_secret"`
	Status       string         `json:"status"`
	TimeoutMS    int            `json:"timeout_ms"`
	RetryCount   int            `json:"retry_count"`
	Metadata     map[string]any `json:"metadata"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type AdapterSecret struct {
	ID          uuid.UUID
	Name        string
	EndpointURL string
	AuthType    string
	Secret      string
	Status      string
	TimeoutMS   int
	RetryCount  int
}

type ExportBatch struct {
	ID              uuid.UUID      `json:"id"`
	AdapterID       uuid.UUID      `json:"adapter_id"`
	PeriodStart     time.Time      `json:"period_start"`
	PeriodEnd       time.Time      `json:"period_end"`
	Status          string         `json:"status"`
	Currency        string         `json:"currency"`
	TotalAmount     float64        `json:"total_amount"`
	ExternalBatchID string         `json:"external_batch_id"`
	ErrorMessage    string         `json:"error_message"`
	IdempotencyKey  string         `json:"idempotency_key"`
	Metadata        map[string]any `json:"metadata"`
	Lines           []ExportLine   `json:"lines,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
	SubmittedAt     *time.Time     `json:"submitted_at,omitempty"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type ExportLine struct {
	ID                 uuid.UUID      `json:"id"`
	BatchID            uuid.UUID      `json:"batch_id"`
	UsageLedgerID      *uuid.UUID     `json:"usage_ledger_id,omitempty"`
	ProjectCostEntryID *uuid.UUID     `json:"project_cost_entry_id,omitempty"`
	OrganizationID     *uuid.UUID     `json:"organization_id,omitempty"`
	DepartmentID       *uuid.UUID     `json:"department_id,omitempty"`
	ProjectID          *uuid.UUID     `json:"project_id,omitempty"`
	ProviderID         *uuid.UUID     `json:"provider_id,omitempty"`
	ModelID            *uuid.UUID     `json:"model_id,omitempty"`
	Amount             float64        `json:"amount"`
	Currency           string         `json:"currency"`
	ExternalLineID     string         `json:"external_line_id"`
	Status             string         `json:"status"`
	Metadata           map[string]any `json:"metadata"`
	CreatedAt          time.Time      `json:"created_at"`
}

type WebhookEvent struct {
	ID             uuid.UUID      `json:"id"`
	AdapterID      uuid.UUID      `json:"adapter_id"`
	BatchID        *uuid.UUID     `json:"batch_id,omitempty"`
	EventType      string         `json:"event_type"`
	SignatureValid bool           `json:"signature_valid"`
	Payload        map[string]any `json:"payload"`
	Processed      bool           `json:"processed"`
	ErrorMessage   string         `json:"error_message"`
	CreatedAt      time.Time      `json:"created_at"`
}

type ReconciliationItem struct {
	BatchID          uuid.UUID  `json:"batch_id"`
	AdapterID        uuid.UUID  `json:"adapter_id"`
	Status           string     `json:"status"`
	Currency         string     `json:"currency"`
	TotalAmount      float64    `json:"total_amount"`
	ExternalAmount   float64    `json:"external_amount"`
	DifferenceAmount float64    `json:"difference_amount"`
	ExternalBatchID  string     `json:"external_batch_id"`
	ErrorMessage     string     `json:"error_message"`
	SubmittedAt      *time.Time `json:"submitted_at,omitempty"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type CreateAdapterInput struct {
	Name        string         `json:"name"`
	EndpointURL string         `json:"endpoint_url"`
	AuthType    string         `json:"auth_type,omitempty"`
	Secret      string         `json:"secret"`
	Status      string         `json:"status,omitempty"`
	TimeoutMS   int            `json:"timeout_ms,omitempty"`
	RetryCount  int            `json:"retry_count,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type UpdateAdapterInput struct {
	Name        *string        `json:"name,omitempty"`
	EndpointURL *string        `json:"endpoint_url,omitempty"`
	AuthType    *string        `json:"auth_type,omitempty"`
	Secret      *string        `json:"secret,omitempty"`
	Status      *string        `json:"status,omitempty"`
	TimeoutMS   *int           `json:"timeout_ms,omitempty"`
	RetryCount  *int           `json:"retry_count,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type CreateExportBatchInput struct {
	AdapterID      uuid.UUID      `json:"adapter_id"`
	PeriodStart    string         `json:"period_start"`
	PeriodEnd      string         `json:"period_end"`
	Currency       string         `json:"currency,omitempty"`
	ActorID        *uuid.UUID     `json:"actor_id,omitempty"`
	ActorType      string         `json:"actor_type,omitempty"`
	IdempotencyKey string         `json:"idempotency_key,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`

	periodStartTime time.Time
	periodEndTime   time.Time
}

type UpdateExportBatchStatusInput struct {
	Status          string         `json:"status"`
	ExternalBatchID string         `json:"external_batch_id,omitempty"`
	ErrorMessage    string         `json:"error_message,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	Submitted       bool           `json:"submitted,omitempty"`
}

type RecordWebhookEventInput struct {
	AdapterID      uuid.UUID
	BatchID        *uuid.UUID
	EventType      string
	SignatureValid bool
	Payload        map[string]any
	Processed      bool
	ErrorMessage   string
}

type UpdateExportLineStatusInput struct {
	Status         string
	ExternalLineID string
	Metadata       map[string]any
}
