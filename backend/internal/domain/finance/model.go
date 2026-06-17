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
	ID             uuid.UUID      `json:"id"`
	Name           string         `json:"name"`
	EndpointURL    string         `json:"endpoint_url"`
	AuthType       string         `json:"auth_type"`
	AdapterType    string         `json:"adapter_type"`
	Direction      string         `json:"direction"`
	MaskedSecret   string         `json:"masked_secret"`
	Status         string         `json:"status"`
	TimeoutMS      int            `json:"timeout_ms"`
	RetryCount     int            `json:"retry_count"`
	FieldMapping   map[string]any `json:"field_mapping"`
	PullConfig     map[string]any `json:"pull_config"`
	LastSyncAt     *time.Time     `json:"last_sync_at,omitempty"`
	LastSyncStatus string         `json:"last_sync_status"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type AdapterSecret struct {
	ID           uuid.UUID
	Name         string
	EndpointURL  string
	AuthType     string
	AdapterType  string
	Direction    string
	Secret       string
	Status       string
	TimeoutMS    int
	RetryCount   int
	FieldMapping map[string]any
	PullConfig   map[string]any
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
	CostLedgerEntryID  *uuid.UUID     `json:"cost_ledger_entry_id,omitempty"`
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
	Name         string         `json:"name"`
	EndpointURL  string         `json:"endpoint_url"`
	AuthType     string         `json:"auth_type,omitempty"`
	AdapterType  string         `json:"adapter_type,omitempty"`
	Direction    string         `json:"direction,omitempty"`
	Secret       string         `json:"secret"`
	Status       string         `json:"status,omitempty"`
	TimeoutMS    int            `json:"timeout_ms,omitempty"`
	RetryCount   int            `json:"retry_count,omitempty"`
	FieldMapping map[string]any `json:"field_mapping,omitempty"`
	PullConfig   map[string]any `json:"pull_config,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type UpdateAdapterInput struct {
	Name         *string        `json:"name,omitempty"`
	EndpointURL  *string        `json:"endpoint_url,omitempty"`
	AuthType     *string        `json:"auth_type,omitempty"`
	AdapterType  *string        `json:"adapter_type,omitempty"`
	Direction    *string        `json:"direction,omitempty"`
	Secret       *string        `json:"secret,omitempty"`
	Status       *string        `json:"status,omitempty"`
	TimeoutMS    *int           `json:"timeout_ms,omitempty"`
	RetryCount   *int           `json:"retry_count,omitempty"`
	FieldMapping map[string]any `json:"field_mapping,omitempty"`
	PullConfig   map[string]any `json:"pull_config,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
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

type ImportBatch struct {
	ID               uuid.UUID      `json:"id"`
	AdapterID        *uuid.UUID     `json:"adapter_id,omitempty"`
	SourceType       string         `json:"source_type"`
	FileName         string         `json:"file_name"`
	Status           string         `json:"status"`
	TotalRecords     int            `json:"total_records"`
	ProcessedRecords int            `json:"processed_records"`
	FailedRecords    int            `json:"failed_records"`
	Metadata         map[string]any `json:"metadata"`
	CreatedAt        time.Time      `json:"created_at"`
	CompletedAt      *time.Time     `json:"completed_at,omitempty"`
}

type ImportRecord struct {
	ID                uuid.UUID      `json:"id"`
	BatchID           uuid.UUID      `json:"batch_id"`
	AdapterID         *uuid.UUID     `json:"adapter_id,omitempty"`
	ExternalRecordID  string         `json:"external_record_id"`
	ExpenseType       string         `json:"expense_type"`
	RawPayload        map[string]any `json:"raw_payload"`
	NormalizedPayload map[string]any `json:"normalized_payload"`
	CostLedgerEntryID *uuid.UUID     `json:"cost_ledger_entry_id,omitempty"`
	PayableID         *uuid.UUID     `json:"payable_id,omitempty"`
	Status            string         `json:"status"`
	ErrorMessage      string         `json:"error_message"`
	Metadata          map[string]any `json:"metadata"`
	CreatedAt         time.Time      `json:"created_at"`
}

type FinanceExpenseInput struct {
	ExternalRecordID string         `json:"external_record_id"`
	ExpenseType      string         `json:"expense_type,omitempty"`
	CostCategory     string         `json:"cost_category,omitempty"`
	Amount           float64        `json:"amount"`
	Currency         string         `json:"currency,omitempty"`
	OccurredAt       string         `json:"occurred_at,omitempty"`
	Description      string         `json:"description,omitempty"`
	AccountCode      string         `json:"account_code,omitempty"`
	AccountName      string         `json:"account_name,omitempty"`
	CostCenterCode   string         `json:"cost_center_code,omitempty"`
	CostCenterName   string         `json:"cost_center_name,omitempty"`
	VendorID         string         `json:"vendor_id,omitempty"`
	VendorName       string         `json:"vendor_name,omitempty"`
	EmployeeID       string         `json:"employee_id,omitempty"`
	EmployeeName     string         `json:"employee_name,omitempty"`
	AgentID          *uuid.UUID     `json:"agent_id,omitempty"`
	AgentName        string         `json:"agent_name,omitempty"`
	OrganizationID   *uuid.UUID     `json:"organization_id,omitempty"`
	DepartmentID     *uuid.UUID     `json:"department_id,omitempty"`
	RequirementID    *uuid.UUID     `json:"requirement_id,omitempty"`
	ProjectID        *uuid.UUID     `json:"project_id,omitempty"`
	WorkflowID       *uuid.UUID     `json:"workflow_id,omitempty"`
	TaskID           *uuid.UUID     `json:"task_id,omitempty"`
	CapabilityID     *uuid.UUID     `json:"capability_id,omitempty"`
	TaxAmount        float64        `json:"tax_amount,omitempty"`
	TaxRate          float64        `json:"tax_rate,omitempty"`
	InvoiceNumber    string         `json:"invoice_number,omitempty"`
	InvoiceDate      string         `json:"invoice_date,omitempty"`
	PaymentStatus    string         `json:"payment_status,omitempty"`
	PaymentDueDate   string         `json:"payment_due_date,omitempty"`
	PaidAt           string         `json:"paid_at,omitempty"`
	PeriodStart      string         `json:"period_start,omitempty"`
	PeriodEnd        string         `json:"period_end,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type ImportExpensesInput struct {
	AdapterID  uuid.UUID        `json:"adapter_id"`
	SourceType string           `json:"source_type,omitempty"`
	FileName   string           `json:"file_name,omitempty"`
	Records    []map[string]any `json:"records"`
	Metadata   map[string]any   `json:"metadata,omitempty"`
}

type ImportExpensesResult struct {
	Batch   *ImportBatch   `json:"batch"`
	Records []ImportRecord `json:"records"`
}

type Payable struct {
	ID                uuid.UUID      `json:"id"`
	PayableType       string         `json:"payable_type"`
	SourceType        string         `json:"source_type"`
	SourceID          *uuid.UUID     `json:"source_id,omitempty"`
	ExternalPayableID string         `json:"external_payable_id"`
	InvoiceNumber     string         `json:"invoice_number"`
	VendorID          string         `json:"vendor_id"`
	VendorName        string         `json:"vendor_name"`
	EmployeeID        string         `json:"employee_id"`
	EmployeeName      string         `json:"employee_name"`
	AgentID           *uuid.UUID     `json:"agent_id,omitempty"`
	ProjectID         *uuid.UUID     `json:"project_id,omitempty"`
	OrganizationID    *uuid.UUID     `json:"organization_id,omitempty"`
	DepartmentID      *uuid.UUID     `json:"department_id,omitempty"`
	AccountCode       string         `json:"account_code"`
	AccountName       string         `json:"account_name"`
	CostCenterCode    string         `json:"cost_center_code"`
	CostCenterName    string         `json:"cost_center_name"`
	Amount            float64        `json:"amount"`
	TaxAmount         float64        `json:"tax_amount"`
	Currency          string         `json:"currency"`
	PeriodStart       *time.Time     `json:"period_start,omitempty"`
	PeriodEnd         *time.Time     `json:"period_end,omitempty"`
	InvoiceDate       *time.Time     `json:"invoice_date,omitempty"`
	DueDate           *time.Time     `json:"due_date,omitempty"`
	Status            string         `json:"status"`
	PaidAmount        float64        `json:"paid_amount"`
	Metadata          map[string]any `json:"metadata"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type CreatePayableInput struct {
	PayableType       string         `json:"payable_type,omitempty"`
	SourceType        string         `json:"source_type,omitempty"`
	SourceID          *uuid.UUID     `json:"source_id,omitempty"`
	ExternalPayableID string         `json:"external_payable_id,omitempty"`
	InvoiceNumber     string         `json:"invoice_number,omitempty"`
	VendorID          string         `json:"vendor_id,omitempty"`
	VendorName        string         `json:"vendor_name,omitempty"`
	EmployeeID        string         `json:"employee_id,omitempty"`
	EmployeeName      string         `json:"employee_name,omitempty"`
	AgentID           *uuid.UUID     `json:"agent_id,omitempty"`
	ProjectID         *uuid.UUID     `json:"project_id,omitempty"`
	OrganizationID    *uuid.UUID     `json:"organization_id,omitempty"`
	DepartmentID      *uuid.UUID     `json:"department_id,omitempty"`
	AccountCode       string         `json:"account_code,omitempty"`
	AccountName       string         `json:"account_name,omitempty"`
	CostCenterCode    string         `json:"cost_center_code,omitempty"`
	CostCenterName    string         `json:"cost_center_name,omitempty"`
	Amount            float64        `json:"amount"`
	TaxAmount         float64        `json:"tax_amount,omitempty"`
	Currency          string         `json:"currency,omitempty"`
	PeriodStart       string         `json:"period_start,omitempty"`
	PeriodEnd         string         `json:"period_end,omitempty"`
	InvoiceDate       string         `json:"invoice_date,omitempty"`
	DueDate           string         `json:"due_date,omitempty"`
	Status            string         `json:"status,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
}

type Payment struct {
	ID                uuid.UUID      `json:"id"`
	PaymentNumber     string         `json:"payment_number"`
	ExternalPaymentID string         `json:"external_payment_id"`
	PaymentMethod     string         `json:"payment_method"`
	PayerAccount      string         `json:"payer_account"`
	PayeeAccount      string         `json:"payee_account"`
	VendorID          string         `json:"vendor_id"`
	VendorName        string         `json:"vendor_name"`
	EmployeeID        string         `json:"employee_id"`
	EmployeeName      string         `json:"employee_name"`
	Amount            float64        `json:"amount"`
	Currency          string         `json:"currency"`
	PaidAt            *time.Time     `json:"paid_at,omitempty"`
	Status            string         `json:"status"`
	Metadata          map[string]any `json:"metadata"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

type CreatePaymentInput struct {
	PaymentNumber     string         `json:"payment_number,omitempty"`
	ExternalPaymentID string         `json:"external_payment_id,omitempty"`
	PaymentMethod     string         `json:"payment_method,omitempty"`
	PayerAccount      string         `json:"payer_account,omitempty"`
	PayeeAccount      string         `json:"payee_account,omitempty"`
	VendorID          string         `json:"vendor_id,omitempty"`
	VendorName        string         `json:"vendor_name,omitempty"`
	EmployeeID        string         `json:"employee_id,omitempty"`
	EmployeeName      string         `json:"employee_name,omitempty"`
	Amount            float64        `json:"amount"`
	Currency          string         `json:"currency,omitempty"`
	PaidAt            string         `json:"paid_at,omitempty"`
	Status            string         `json:"status,omitempty"`
	Metadata          map[string]any `json:"metadata,omitempty"`
}

type PaymentAllocation struct {
	ID        uuid.UUID      `json:"id"`
	PaymentID uuid.UUID      `json:"payment_id"`
	PayableID uuid.UUID      `json:"payable_id"`
	Amount    float64        `json:"amount"`
	Currency  string         `json:"currency"`
	Metadata  map[string]any `json:"metadata"`
	CreatedAt time.Time      `json:"created_at"`
}

type AllocatePaymentInput struct {
	PayableID uuid.UUID      `json:"payable_id"`
	Amount    float64        `json:"amount"`
	Currency  string         `json:"currency,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}
