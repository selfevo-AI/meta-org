package aigateway

import (
	"time"

	"github.com/google/uuid"
)

const (
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
	ProviderGemini    = "gemini"
	ModeSync          = "sync"
	ModeStream        = "stream"
	StatusStarted     = "started"
	StatusStreaming   = "streaming"
	StatusCompleted   = "completed"
	StatusFailed      = "failed"
	StatusCancelled   = "cancelled"
)

type ModelProvider struct {
	ID             uuid.UUID      `json:"id"`
	Name           string         `json:"name"`
	ProviderType   string         `json:"provider_type"`
	BaseURL        string         `json:"base_url"`
	MaskedAPIKey   string         `json:"masked_api_key"`
	Status         string         `json:"status"`
	TimeoutMS      int            `json:"timeout_ms"`
	RetryCount     int            `json:"retry_count"`
	RiskLevel      string         `json:"risk_level"`
	Tags           []string       `json:"tags"`
	Metadata       map[string]any `json:"metadata"`
	LastTestStatus string         `json:"last_test_status"`
	LastTestError  string         `json:"last_test_error,omitempty"`
	LastTestedAt   *time.Time     `json:"last_tested_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

type Model struct {
	ID              uuid.UUID      `json:"id"`
	ProviderID      uuid.UUID      `json:"provider_id"`
	ModelKey        string         `json:"model_key"`
	DisplayName     string         `json:"display_name"`
	ContextWindow   int            `json:"context_window"`
	MaxOutputTokens int            `json:"max_output_tokens"`
	Capabilities    []string       `json:"capabilities"`
	Status          string         `json:"status"`
	Metadata        map[string]any `json:"metadata"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type PriceVersion struct {
	ID               uuid.UUID  `json:"id"`
	ModelID          uuid.UUID  `json:"model_id"`
	InputPricePer1K  float64    `json:"input_price_per_1k"`
	OutputPricePer1K float64    `json:"output_price_per_1k"`
	Currency         string     `json:"currency"`
	EffectiveFrom    time.Time  `json:"effective_from"`
	EffectiveTo      *time.Time `json:"effective_to,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type Attribution struct {
	OrganizationID *uuid.UUID `json:"organization_id,omitempty"`
	DepartmentID   *uuid.UUID `json:"department_id,omitempty"`
	ProjectID      *uuid.UUID `json:"project_id,omitempty"`
	RequirementID  *uuid.UUID `json:"requirement_id,omitempty"`
	WorkflowID     *uuid.UUID `json:"workflow_id,omitempty"`
	TaskID         *uuid.UUID `json:"task_id,omitempty"`
	AgentID        *uuid.UUID `json:"agent_id,omitempty"`
	UserID         *uuid.UUID `json:"user_id,omitempty"`
	CapabilityID   *uuid.UUID `json:"capability_id,omitempty"`
	SourceSurface  string     `json:"source_surface,omitempty"`
}

type Invocation struct {
	ID                    uuid.UUID      `json:"id"`
	ProviderID            uuid.UUID      `json:"provider_id"`
	ModelID               uuid.UUID      `json:"model_id"`
	Mode                  string         `json:"mode"`
	Status                string         `json:"status"`
	Attribution           Attribution    `json:"attribution"`
	RequestHash           string         `json:"request_hash,omitempty"`
	ProviderRequestID     string         `json:"provider_request_id,omitempty"`
	InputTokens           int            `json:"input_tokens"`
	OutputTokens          int            `json:"output_tokens"`
	EstimatedInputTokens  int            `json:"estimated_input_tokens"`
	EstimatedOutputTokens int            `json:"estimated_output_tokens"`
	CostAmount            float64        `json:"cost_amount"`
	Currency              string         `json:"currency"`
	FirstTokenMS          int            `json:"first_token_ms"`
	DurationMS            int            `json:"duration_ms"`
	ErrorType             string         `json:"error_type,omitempty"`
	ErrorMessage          string         `json:"error_message,omitempty"`
	Metadata              map[string]any `json:"metadata"`
	CreatedAt             time.Time      `json:"created_at"`
	CompletedAt           *time.Time     `json:"completed_at,omitempty"`
}

type UsageLedgerEntry struct {
	ID                  uuid.UUID  `json:"id"`
	InvocationID        uuid.UUID  `json:"invocation_id"`
	ModelPriceVersionID *uuid.UUID `json:"model_price_version_id,omitempty"`
	LedgerType          string     `json:"ledger_type"`
	Amount              float64    `json:"amount"`
	Currency            string     `json:"currency"`
	InputTokens         int        `json:"input_tokens"`
	OutputTokens        int        `json:"output_tokens"`
	PostedToProjectCost bool       `json:"posted_to_project_cost"`
	ProjectCostEntryID  *uuid.UUID `json:"project_cost_entry_id,omitempty"`
	FinanceExportLineID *uuid.UUID `json:"finance_export_line_id,omitempty"`
	Reason              string     `json:"reason,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

type Message struct {
	Role       string         `json:"role"`
	Content    string         `json:"content,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type ToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Schema      map[string]any `json:"schema"`
}

type InvokeRequest struct {
	ProviderID  *uuid.UUID     `json:"provider_id,omitempty"`
	ModelID     *uuid.UUID     `json:"model_id,omitempty"`
	ModelKey    string         `json:"model_key,omitempty"`
	Messages    []Message      `json:"messages"`
	Temperature *float64       `json:"temperature,omitempty"`
	TopP        *float64       `json:"top_p,omitempty"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
	Stream      bool           `json:"stream"`
	Tools       []ToolSpec     `json:"tools,omitempty"`
	Attribution Attribution    `json:"attribution"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type InvokeResponse struct {
	InvocationID uuid.UUID        `json:"invocation_id"`
	Status       string           `json:"status"`
	Message      Message          `json:"message"`
	Usage        TokenUsage       `json:"usage"`
	CostAmount   float64          `json:"cost_amount"`
	Currency     string           `json:"currency"`
	ToolCalls    []ToolCallResult `json:"tool_calls,omitempty"`
}

type ToolCallResult struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Status    string         `json:"status"`
	Arguments map[string]any `json:"arguments,omitempty"`
	Result    map[string]any `json:"result,omitempty"`
	Error     string         `json:"error,omitempty"`
}
