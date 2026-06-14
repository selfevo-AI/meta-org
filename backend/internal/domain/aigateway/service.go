package aigateway

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

var (
	ErrValidation  = errors.New("validation error")
	ErrNotFound    = errors.New("not found")
	ErrUnavailable = errors.New("unavailable")
)

type AdapterRegistry map[string]ProviderAdapter

type InvocationRepository interface {
	ResolveInvocationTarget(ctx context.Context, input InvokeInput) (ResolvedModel, error)
	CreateInvocation(ctx context.Context, input CreateInvocationInput) (*Invocation, error)
	CompleteInvocation(ctx context.Context, id uuid.UUID, input CompleteInvocationInput) error
	FailInvocation(ctx context.Context, id uuid.UUID, input FailInvocationInput) error
	CreateUsageLedger(ctx context.Context, input CreateUsageLedgerInput) error
}

type CatalogRepository interface {
	CreateProvider(ctx context.Context, input CreateProviderInput) (*ModelProvider, error)
	ListProviders(ctx context.Context, limit int) ([]ModelProvider, error)
	UpdateProvider(ctx context.Context, id uuid.UUID, input UpdateProviderInput) (*ModelProvider, error)
	RotateProviderKey(ctx context.Context, id uuid.UUID, apiKey string) (*ModelProvider, error)
	UpdateProviderTestResult(ctx context.Context, id uuid.UUID, status string, message string) error
	GetProviderSecret(ctx context.Context, id uuid.UUID) (ProviderSecret, error)
	CreateModel(ctx context.Context, input CreateModelInput) (*Model, error)
	ListModels(ctx context.Context, providerID *uuid.UUID, limit int) ([]Model, error)
	UpdateModel(ctx context.Context, id uuid.UUID, input UpdateModelInput) (*Model, error)
	ListInvocations(ctx context.Context, limit int) ([]Invocation, error)
	GetInvocation(ctx context.Context, id uuid.UUID) (*Invocation, error)
	CostSummary(ctx context.Context) (*GatewayCostSummary, error)
}

type Service struct {
	repo     InvocationRepository
	catalog  CatalogRepository
	adapters AdapterRegistry
	client   *http.Client
}

func NewService(repo InvocationRepository, adapters AdapterRegistry) *Service {
	catalog, _ := repo.(CatalogRepository)
	if adapters == nil {
		adapters = AdapterRegistry{}
	}
	return &Service{
		repo:     repo,
		catalog:  catalog,
		adapters: adapters,
		client:   &http.Client{Timeout: 60 * time.Second},
	}
}

type ResolvedModel struct {
	ProviderID          uuid.UUID
	ModelID             uuid.UUID
	ProviderType        string
	BaseURL             string
	APIKey              string
	Model               string
	PriceVersionID      *uuid.UUID
	Price               Price
	Currency            string
	MaxOutputTokens     int
	ProviderRequestHint string
}

type InvokeInput struct {
	ProviderID   *uuid.UUID       `json:"provider_id,omitempty"`
	ProviderType string           `json:"provider_type,omitempty"`
	ModelID      *uuid.UUID       `json:"model_id,omitempty"`
	Model        string           `json:"model"`
	Messages     []Message        `json:"messages"`
	Temperature  *float64         `json:"temperature,omitempty"`
	MaxTokens    int              `json:"max_tokens,omitempty"`
	Tools        []ToolDefinition `json:"tools,omitempty"`
	Attribution  Attribution      `json:"attribution"`
	Metadata     map[string]any   `json:"metadata,omitempty"`
}

type InvokeOutput struct {
	InvocationID      uuid.UUID  `json:"invocation_id"`
	ProviderRequestID string     `json:"provider_request_id,omitempty"`
	Content           string     `json:"content"`
	Usage             TokenUsage `json:"usage"`
	CostAmount        float64    `json:"cost_amount"`
	Currency          string     `json:"currency"`
	ToolCalls         []ToolCall `json:"tool_calls,omitempty"`
	CompletedAt       time.Time  `json:"completed_at"`
	ProviderType      string     `json:"provider_type"`
	Model             string     `json:"model"`
}

type StreamResult struct {
	InvocationID uuid.UUID
	Events       <-chan StreamEvent
}

type CreateInvocationInput struct {
	ProviderID            uuid.UUID
	ModelID               uuid.UUID
	Mode                  string
	Status                string
	Attribution           Attribution
	RequestHash           string
	EstimatedInputTokens  int
	EstimatedOutputTokens int
	Metadata              map[string]any
}

type CompleteInvocationInput struct {
	ProviderRequestID string
	Usage             TokenUsage
	CostAmount        float64
	Currency          string
	DurationMS        int
}

type FailInvocationInput struct {
	ErrorType  string
	Message    string
	DurationMS int
	Cancelled  bool
}

type CreateUsageLedgerInput struct {
	InvocationID        uuid.UUID
	ModelPriceVersionID *uuid.UUID
	LedgerType          string
	Amount              float64
	Currency            string
	Usage               TokenUsage
	Reason              string
}

type CreateProviderInput struct {
	Name         string         `json:"name"`
	ProviderType string         `json:"provider_type"`
	BaseURL      string         `json:"base_url,omitempty"`
	APIKey       string         `json:"api_key"`
	Status       string         `json:"status,omitempty"`
	TimeoutMS    int            `json:"timeout_ms,omitempty"`
	RetryCount   int            `json:"retry_count,omitempty"`
	RiskLevel    string         `json:"risk_level,omitempty"`
	Tags         []string       `json:"tags,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type UpdateProviderInput struct {
	Name       *string        `json:"name,omitempty"`
	BaseURL    *string        `json:"base_url,omitempty"`
	Status     *string        `json:"status,omitempty"`
	TimeoutMS  *int           `json:"timeout_ms,omitempty"`
	RetryCount *int           `json:"retry_count,omitempty"`
	RiskLevel  *string        `json:"risk_level,omitempty"`
	Tags       []string       `json:"tags,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type ProviderSecret struct {
	ID           uuid.UUID
	Name         string
	ProviderType string
	BaseURL      string
	APIKey       string
	Status       string
}

type RotateProviderKeyInput struct {
	APIKey string `json:"api_key"`
}

type TestProviderInput struct {
	Model string `json:"model,omitempty"`
}

type CreateModelInput struct {
	ProviderID       uuid.UUID      `json:"provider_id"`
	ModelKey         string         `json:"model_key"`
	DisplayName      string         `json:"display_name,omitempty"`
	ContextWindow    int            `json:"context_window,omitempty"`
	MaxOutputTokens  int            `json:"max_output_tokens,omitempty"`
	Capabilities     []string       `json:"capabilities,omitempty"`
	Status           string         `json:"status,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	InputPricePer1K  float64        `json:"input_price_per_1k,omitempty"`
	OutputPricePer1K float64        `json:"output_price_per_1k,omitempty"`
	Currency         string         `json:"currency,omitempty"`
}

type UpdateModelInput struct {
	DisplayName      *string        `json:"display_name,omitempty"`
	ContextWindow    *int           `json:"context_window,omitempty"`
	MaxOutputTokens  *int           `json:"max_output_tokens,omitempty"`
	Capabilities     []string       `json:"capabilities,omitempty"`
	Status           *string        `json:"status,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
	InputPricePer1K  *float64       `json:"input_price_per_1k,omitempty"`
	OutputPricePer1K *float64       `json:"output_price_per_1k,omitempty"`
	Currency         *string        `json:"currency,omitempty"`
}

type GatewayCostSummary struct {
	Total      float64            `json:"total"`
	Unexported float64            `json:"unexported"`
	Currency   string             `json:"currency"`
	ByProvider map[string]float64 `json:"by_provider"`
}

func (s *Service) Invoke(ctx context.Context, input InvokeInput) (*InvokeOutput, error) {
	if err := validateInvokeInput(input); err != nil {
		return nil, err
	}
	target, err := s.repo.ResolveInvocationTarget(ctx, input)
	if err != nil {
		return nil, err
	}
	adapter, err := s.adapterFor(target)
	if err != nil {
		return nil, err
	}

	started := time.Now()
	invocation, err := s.repo.CreateInvocation(ctx, CreateInvocationInput{
		ProviderID:  target.ProviderID,
		ModelID:     target.ModelID,
		Mode:        ModeSync,
		Status:      StatusStarted,
		Attribution: input.Attribution,
		Metadata:    input.Metadata,
	})
	if err != nil {
		return nil, err
	}

	resp, err := adapter.Invoke(ctx, ProviderRequest{
		Model:       target.Model,
		Messages:    input.Messages,
		Temperature: input.Temperature,
		MaxTokens:   maxTokens(input.MaxTokens, target.MaxOutputTokens),
		Tools:       input.Tools,
	})
	if err != nil {
		s.recordFailedInvocation(ctx, invocation.ID, target, started, err)
		return nil, err
	}

	cost := CalculateCost(resp.Usage, target.Price)
	if err := s.repo.CreateUsageLedger(ctx, CreateUsageLedgerInput{
		InvocationID:        invocation.ID,
		ModelPriceVersionID: target.PriceVersionID,
		LedgerType:          "usage",
		Amount:              cost,
		Currency:            currencyOrDefault(target.Currency),
		Usage:               resp.Usage,
	}); err != nil {
		return nil, err
	}
	completedAt := time.Now()
	if err := s.repo.CompleteInvocation(ctx, invocation.ID, CompleteInvocationInput{
		ProviderRequestID: resp.ProviderRequestID,
		Usage:             resp.Usage,
		CostAmount:        cost,
		Currency:          currencyOrDefault(target.Currency),
		DurationMS:        int(completedAt.Sub(started).Milliseconds()),
	}); err != nil {
		return nil, err
	}
	return &InvokeOutput{
		InvocationID:      invocation.ID,
		ProviderRequestID: resp.ProviderRequestID,
		Content:           resp.Content,
		Usage:             resp.Usage,
		CostAmount:        cost,
		Currency:          currencyOrDefault(target.Currency),
		ToolCalls:         resp.ToolCalls,
		CompletedAt:       completedAt,
		ProviderType:      target.ProviderType,
		Model:             target.Model,
	}, nil
}

func (s *Service) Stream(ctx context.Context, input InvokeInput) (*StreamResult, error) {
	if err := validateInvokeInput(input); err != nil {
		return nil, err
	}
	target, err := s.repo.ResolveInvocationTarget(ctx, input)
	if err != nil {
		return nil, err
	}
	adapter, err := s.adapterFor(target)
	if err != nil {
		return nil, err
	}
	started := time.Now()
	invocation, err := s.repo.CreateInvocation(ctx, CreateInvocationInput{
		ProviderID:  target.ProviderID,
		ModelID:     target.ModelID,
		Mode:        ModeStream,
		Status:      StatusStreaming,
		Attribution: input.Attribution,
		Metadata:    input.Metadata,
	})
	if err != nil {
		return nil, err
	}
	events, err := adapter.Stream(ctx, ProviderRequest{
		Model:       target.Model,
		Messages:    input.Messages,
		Temperature: input.Temperature,
		MaxTokens:   maxTokens(input.MaxTokens, target.MaxOutputTokens),
		Tools:       input.Tools,
	})
	if err != nil {
		s.recordFailedInvocation(ctx, invocation.ID, target, started, err)
		return nil, err
	}
	return &StreamResult{InvocationID: invocation.ID, Events: s.recordingStream(ctx, invocation.ID, target, started, events)}, nil
}

func (s *Service) CreateProvider(ctx context.Context, input CreateProviderInput) (*ModelProvider, error) {
	if err := validateProviderInput(input); err != nil {
		return nil, err
	}
	return s.catalogRepo().CreateProvider(ctx, input)
}

func (s *Service) ListProviders(ctx context.Context, limit int) ([]ModelProvider, error) {
	return s.catalogRepo().ListProviders(ctx, limit)
}

func (s *Service) UpdateProvider(ctx context.Context, id uuid.UUID, input UpdateProviderInput) (*ModelProvider, error) {
	return s.catalogRepo().UpdateProvider(ctx, id, input)
}

func (s *Service) RotateProviderKey(ctx context.Context, id uuid.UUID, input RotateProviderKeyInput) (*ModelProvider, error) {
	if input.APIKey == "" {
		return nil, fmt.Errorf("%w: api_key is required", ErrValidation)
	}
	return s.catalogRepo().RotateProviderKey(ctx, id, input.APIKey)
}

func (s *Service) TestProvider(ctx context.Context, id uuid.UUID, input TestProviderInput) error {
	provider, err := s.catalogRepo().GetProviderSecret(ctx, id)
	if err != nil {
		return err
	}
	target := ResolvedModel{
		ProviderID:   provider.ID,
		ProviderType: provider.ProviderType,
		BaseURL:      provider.BaseURL,
		APIKey:       provider.APIKey,
		Model:        input.Model,
	}
	if target.Model == "" {
		target.Model = defaultTestModel(provider.ProviderType)
	}
	adapter, err := s.adapterFor(target)
	if err != nil {
		return err
	}
	_, err = adapter.Invoke(ctx, ProviderRequest{
		Model:     target.Model,
		Messages:  []Message{{Role: "user", Content: "ping"}},
		MaxTokens: 8,
	})
	if err != nil {
		_ = s.catalogRepo().UpdateProviderTestResult(ctx, id, "failed", err.Error())
		return err
	}
	return s.catalogRepo().UpdateProviderTestResult(ctx, id, "ok", "")
}

func (s *Service) CreateModel(ctx context.Context, input CreateModelInput) (*Model, error) {
	if input.ProviderID == uuid.Nil || input.ModelKey == "" {
		return nil, fmt.Errorf("%w: provider_id and model_key are required", ErrValidation)
	}
	return s.catalogRepo().CreateModel(ctx, input)
}

func (s *Service) ListModels(ctx context.Context, providerID *uuid.UUID, limit int) ([]Model, error) {
	return s.catalogRepo().ListModels(ctx, providerID, limit)
}

func (s *Service) UpdateModel(ctx context.Context, id uuid.UUID, input UpdateModelInput) (*Model, error) {
	return s.catalogRepo().UpdateModel(ctx, id, input)
}

func (s *Service) ListInvocations(ctx context.Context, limit int) ([]Invocation, error) {
	return s.catalogRepo().ListInvocations(ctx, limit)
}

func (s *Service) GetInvocation(ctx context.Context, id uuid.UUID) (*Invocation, error) {
	return s.catalogRepo().GetInvocation(ctx, id)
}

func (s *Service) CostSummary(ctx context.Context) (*GatewayCostSummary, error) {
	return s.catalogRepo().CostSummary(ctx)
}

func (s *Service) recordingStream(ctx context.Context, invocationID uuid.UUID, target ResolvedModel, started time.Time, events <-chan StreamEvent) <-chan StreamEvent {
	out := make(chan StreamEvent)
	go func() {
		defer close(out)
		usage := TokenUsage{}
		failed := ""
		for event := range events {
			if event.Usage.InputTokens > 0 || event.Usage.OutputTokens > 0 {
				usage = event.Usage
			}
			if event.Error != "" {
				failed = event.Error
			}
			select {
			case <-ctx.Done():
				_ = s.repo.FailInvocation(context.Background(), invocationID, FailInvocationInput{ErrorType: "cancelled", Message: ctx.Err().Error(), DurationMS: int(time.Since(started).Milliseconds()), Cancelled: true})
				return
			case out <- event:
			}
		}
		cost := CalculateCost(usage, target.Price)
		_ = s.repo.CreateUsageLedger(context.Background(), CreateUsageLedgerInput{
			InvocationID:        invocationID,
			ModelPriceVersionID: target.PriceVersionID,
			LedgerType:          "usage",
			Amount:              cost,
			Currency:            currencyOrDefault(target.Currency),
			Usage:               usage,
			Reason:              failed,
		})
		if failed != "" {
			_ = s.repo.FailInvocation(context.Background(), invocationID, FailInvocationInput{ErrorType: "provider_error", Message: failed, DurationMS: int(time.Since(started).Milliseconds())})
			return
		}
		_ = s.repo.CompleteInvocation(context.Background(), invocationID, CompleteInvocationInput{
			Usage:      usage,
			CostAmount: cost,
			Currency:   currencyOrDefault(target.Currency),
			DurationMS: int(time.Since(started).Milliseconds()),
		})
	}()
	return out
}

func (s *Service) recordFailedInvocation(ctx context.Context, id uuid.UUID, target ResolvedModel, started time.Time, cause error) {
	_ = s.repo.CreateUsageLedger(ctx, CreateUsageLedgerInput{
		InvocationID:        id,
		ModelPriceVersionID: target.PriceVersionID,
		LedgerType:          "usage",
		Amount:              0,
		Currency:            currencyOrDefault(target.Currency),
		Reason:              cause.Error(),
	})
	_ = s.repo.FailInvocation(ctx, id, FailInvocationInput{ErrorType: "provider_error", Message: cause.Error(), DurationMS: int(time.Since(started).Milliseconds())})
}

func (s *Service) adapterFor(target ResolvedModel) (ProviderAdapter, error) {
	if adapter, ok := s.adapters[target.ProviderType]; ok {
		return adapter, nil
	}
	switch target.ProviderType {
	case ProviderOpenAI:
		return NewOpenAIAdapter(target.BaseURL, target.APIKey, s.client), nil
	case ProviderAnthropic:
		return NewAnthropicAdapter(target.BaseURL, target.APIKey, s.client), nil
	case ProviderGemini:
		return NewGeminiAdapter(target.BaseURL, target.APIKey, s.client), nil
	default:
		return nil, fmt.Errorf("%w: unsupported provider type %q", ErrValidation, target.ProviderType)
	}
}

func (s *Service) catalogRepo() CatalogRepository {
	if s.catalog == nil {
		panic("aigateway: catalog repository is not configured")
	}
	return s.catalog
}

func validateProviderInput(input CreateProviderInput) error {
	if input.Name == "" || input.ProviderType == "" || input.APIKey == "" {
		return fmt.Errorf("%w: name, provider_type, and api_key are required", ErrValidation)
	}
	switch input.ProviderType {
	case ProviderOpenAI, ProviderAnthropic, ProviderGemini:
		return nil
	default:
		return fmt.Errorf("%w: unsupported provider type %q", ErrValidation, input.ProviderType)
	}
}

func defaultTestModel(providerType string) string {
	switch providerType {
	case ProviderOpenAI:
		return "gpt-4o-mini"
	case ProviderAnthropic:
		return "claude-3-5-haiku-latest"
	case ProviderGemini:
		return "gemini-1.5-flash"
	default:
		return ""
	}
}

func validateInvokeInput(input InvokeInput) error {
	if (input.ProviderID == nil && input.ProviderType == "") || (input.ModelID == nil && input.Model == "") {
		return fmt.Errorf("%w: provider and model are required", ErrValidation)
	}
	if len(input.Messages) == 0 {
		return fmt.Errorf("%w: messages are required", ErrValidation)
	}
	return nil
}

func maxTokens(requested int, modelDefault int) int {
	if requested > 0 {
		return requested
	}
	return modelDefault
}

func currencyOrDefault(currency string) string {
	if currency == "" {
		return "CNY"
	}
	return currency
}
