package costing

import (
	"time"

	"github.com/google/uuid"
)

const (
	BaseCurrency = "CNY"

	CurrencyFiat    = "fiat"
	CurrencyVirtual = "virtual"

	RateSourceManual   = "manual"
	RateSourceExternal = "external"

	LedgerActual     = "actual"
	LedgerEstimate   = "estimate"
	LedgerBudget     = "budget"
	LedgerAdjustment = "adjustment"
)

type Currency struct {
	Code            string         `json:"code"`
	Name            string         `json:"name"`
	CurrencyType    string         `json:"currency_type"`
	Symbol          string         `json:"symbol"`
	PrecisionDigits int            `json:"precision_digits"`
	ChainID         string         `json:"chain_id,omitempty"`
	ContractAddress string         `json:"contract_address,omitempty"`
	ExternalSource  string         `json:"external_source,omitempty"`
	IsActive        bool           `json:"is_active"`
	Metadata        map[string]any `json:"metadata"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type ExchangeRateVersion struct {
	ID             uuid.UUID      `json:"id"`
	FromCurrency   string         `json:"from_currency"`
	ToCurrency     string         `json:"to_currency"`
	Rate           float64        `json:"rate"`
	Source         string         `json:"source"`
	Provider       string         `json:"provider,omitempty"`
	ExternalRateID string         `json:"external_rate_id,omitempty"`
	EffectiveFrom  time.Time      `json:"effective_from"`
	EffectiveTo    *time.Time     `json:"effective_to,omitempty"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      time.Time      `json:"created_at"`
}

type CostRateCard struct {
	ID                    uuid.UUID      `json:"id"`
	SubjectType           string         `json:"subject_type"`
	SubjectID             *uuid.UUID     `json:"subject_id,omitempty"`
	ScopeType             string         `json:"scope_type,omitempty"`
	ScopeID               *uuid.UUID     `json:"scope_id,omitempty"`
	RateType              string         `json:"rate_type"`
	Amount                float64        `json:"amount"`
	Currency              string         `json:"currency"`
	BaseAmount            float64        `json:"base_amount"`
	BaseCurrency          string         `json:"base_currency"`
	ExchangeRateVersionID *uuid.UUID     `json:"exchange_rate_version_id,omitempty"`
	EffectiveFrom         time.Time      `json:"effective_from"`
	EffectiveTo           *time.Time     `json:"effective_to,omitempty"`
	Status                string         `json:"status"`
	Metadata              map[string]any `json:"metadata"`
	CreatedAt             time.Time      `json:"created_at"`
}

type CostBudget struct {
	ID                    uuid.UUID      `json:"id"`
	ScopeType             string         `json:"scope_type"`
	ScopeID               *uuid.UUID     `json:"scope_id,omitempty"`
	Amount                float64        `json:"amount"`
	Currency              string         `json:"currency"`
	BaseAmount            float64        `json:"base_amount"`
	BaseCurrency          string         `json:"base_currency"`
	ExchangeRateVersionID *uuid.UUID     `json:"exchange_rate_version_id,omitempty"`
	PeriodStart           *time.Time     `json:"period_start,omitempty"`
	PeriodEnd             *time.Time     `json:"period_end,omitempty"`
	Status                string         `json:"status"`
	Metadata              map[string]any `json:"metadata"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
}

type CostLedgerEntry struct {
	ID                    uuid.UUID      `json:"id"`
	LedgerType            string         `json:"ledger_type"`
	CostCategory          string         `json:"cost_category"`
	SourceType            string         `json:"source_type"`
	SourceID              *uuid.UUID     `json:"source_id,omitempty"`
	OrganizationID        *uuid.UUID     `json:"organization_id,omitempty"`
	DepartmentID          *uuid.UUID     `json:"department_id,omitempty"`
	RequirementID         *uuid.UUID     `json:"requirement_id,omitempty"`
	ProjectID             *uuid.UUID     `json:"project_id,omitempty"`
	WorkflowID            *uuid.UUID     `json:"workflow_id,omitempty"`
	TaskID                *uuid.UUID     `json:"task_id,omitempty"`
	CapabilityID          *uuid.UUID     `json:"capability_id,omitempty"`
	ActorID               *uuid.UUID     `json:"actor_id,omitempty"`
	ActorType             string         `json:"actor_type,omitempty"`
	ResourceType          string         `json:"resource_type,omitempty"`
	Amount                float64        `json:"amount"`
	Currency              string         `json:"currency"`
	BaseAmount            float64        `json:"base_amount"`
	BaseCurrency          string         `json:"base_currency"`
	ExchangeRateVersionID *uuid.UUID     `json:"exchange_rate_version_id,omitempty"`
	OccurredAt            time.Time      `json:"occurred_at"`
	Status                string         `json:"status"`
	FinanceExportLineID   *uuid.UUID     `json:"finance_export_line_id,omitempty"`
	Description           string         `json:"description"`
	Metadata              map[string]any `json:"metadata"`
	CreatedAt             time.Time      `json:"created_at"`
}

type CostSummary struct {
	ScopeType      string             `json:"scope_type,omitempty"`
	ScopeID        *uuid.UUID         `json:"scope_id,omitempty"`
	Currency       string             `json:"currency"`
	TotalAmount    float64            `json:"total_amount"`
	BudgetAmount   float64            `json:"budget_amount"`
	BudgetVariance float64            `json:"budget_variance"`
	EntryCount     int                `json:"entry_count"`
	ByCategory     map[string]float64 `json:"by_category"`
	BySource       map[string]float64 `json:"by_source"`
	ByCurrency     map[string]float64 `json:"by_currency"`
	RecentEntries  []CostLedgerEntry  `json:"recent_entries"`
	Metadata       map[string]any     `json:"metadata,omitempty"`
}

type CreateCurrencyInput struct {
	Code            string         `json:"code"`
	Name            string         `json:"name,omitempty"`
	CurrencyType    string         `json:"currency_type,omitempty"`
	Symbol          string         `json:"symbol,omitempty"`
	PrecisionDigits int            `json:"precision_digits,omitempty"`
	ChainID         string         `json:"chain_id,omitempty"`
	ContractAddress string         `json:"contract_address,omitempty"`
	ExternalSource  string         `json:"external_source,omitempty"`
	IsActive        *bool          `json:"is_active,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type CreateExchangeRateInput struct {
	FromCurrency   string         `json:"from_currency"`
	ToCurrency     string         `json:"to_currency"`
	Rate           float64        `json:"rate"`
	Source         string         `json:"source,omitempty"`
	Provider       string         `json:"provider,omitempty"`
	ExternalRateID string         `json:"external_rate_id,omitempty"`
	EffectiveFrom  *time.Time     `json:"effective_from,omitempty"`
	EffectiveTo    *time.Time     `json:"effective_to,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type ConvertInput struct {
	Amount       float64    `json:"amount"`
	FromCurrency string     `json:"from_currency"`
	ToCurrency   string     `json:"to_currency"`
	At           *time.Time `json:"at,omitempty"`
}

type ConversionResult struct {
	Amount                float64    `json:"amount"`
	FromCurrency          string     `json:"from_currency"`
	ToCurrency            string     `json:"to_currency"`
	ConvertedAmount       float64    `json:"converted_amount"`
	Rate                  float64    `json:"rate"`
	ExchangeRateVersionID *uuid.UUID `json:"exchange_rate_version_id,omitempty"`
}

type CreateRateCardInput struct {
	SubjectType   string         `json:"subject_type"`
	SubjectID     *uuid.UUID     `json:"subject_id,omitempty"`
	ScopeType     string         `json:"scope_type,omitempty"`
	ScopeID       *uuid.UUID     `json:"scope_id,omitempty"`
	RateType      string         `json:"rate_type,omitempty"`
	Amount        float64        `json:"amount"`
	Currency      string         `json:"currency,omitempty"`
	EffectiveFrom *time.Time     `json:"effective_from,omitempty"`
	EffectiveTo   *time.Time     `json:"effective_to,omitempty"`
	Status        string         `json:"status,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
}

type CreateBudgetInput struct {
	ScopeType   string         `json:"scope_type"`
	ScopeID     *uuid.UUID     `json:"scope_id,omitempty"`
	Amount      float64        `json:"amount"`
	Currency    string         `json:"currency,omitempty"`
	PeriodStart *time.Time     `json:"period_start,omitempty"`
	PeriodEnd   *time.Time     `json:"period_end,omitempty"`
	Status      string         `json:"status,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type CreateLedgerEntryInput struct {
	LedgerType     string         `json:"ledger_type,omitempty"`
	CostCategory   string         `json:"cost_category,omitempty"`
	SourceType     string         `json:"source_type,omitempty"`
	SourceID       *uuid.UUID     `json:"source_id,omitempty"`
	OrganizationID *uuid.UUID     `json:"organization_id,omitempty"`
	DepartmentID   *uuid.UUID     `json:"department_id,omitempty"`
	RequirementID  *uuid.UUID     `json:"requirement_id,omitempty"`
	ProjectID      *uuid.UUID     `json:"project_id,omitempty"`
	WorkflowID     *uuid.UUID     `json:"workflow_id,omitempty"`
	TaskID         *uuid.UUID     `json:"task_id,omitempty"`
	CapabilityID   *uuid.UUID     `json:"capability_id,omitempty"`
	ActorID        *uuid.UUID     `json:"actor_id,omitempty"`
	ActorType      string         `json:"actor_type,omitempty"`
	ResourceType   string         `json:"resource_type,omitempty"`
	Amount         float64        `json:"amount"`
	Currency       string         `json:"currency,omitempty"`
	OccurredAt     *time.Time     `json:"occurred_at,omitempty"`
	Status         string         `json:"status,omitempty"`
	Description    string         `json:"description,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

type UpdateExchangeRateInput CreateExchangeRateInput
type UpdateRateCardInput CreateRateCardInput
type UpdateBudgetInput CreateBudgetInput
type UpdateLedgerEntryInput CreateLedgerEntryInput

type SummaryFilter struct {
	ScopeType string
	ScopeID   *uuid.UUID
	Currency  string
	Limit     int
}
