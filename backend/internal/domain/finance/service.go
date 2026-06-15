package finance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/selfevo-AI/meta-org/backend/internal/domain/observability"
	"github.com/selfevo-AI/meta-org/backend/internal/domain/project"
)

var (
	ErrValidation = errors.New("validation error")
	ErrForbidden  = errors.New("forbidden")
	ErrNotFound   = errors.New("not found")
)

type Repository interface {
	CreateAdapter(ctx context.Context, input CreateAdapterInput) (*FinanceAdapter, error)
	ListAdapters(ctx context.Context, limit int) ([]FinanceAdapter, error)
	UpdateAdapter(ctx context.Context, id uuid.UUID, input UpdateAdapterInput) (*FinanceAdapter, error)
	GetAdapterSecret(ctx context.Context, id uuid.UUID) (AdapterSecret, error)
	CreateExportBatch(ctx context.Context, input CreateExportBatchInput) (*ExportBatch, error)
	ListExportBatches(ctx context.Context, limit int) ([]ExportBatch, error)
	GetExportBatch(ctx context.Context, id uuid.UUID) (*ExportBatch, error)
	UpdateExportBatchStatus(ctx context.Context, id uuid.UUID, input UpdateExportBatchStatusInput) (*ExportBatch, error)
	RecordWebhookEvent(ctx context.Context, input RecordWebhookEventInput) (*WebhookEvent, error)
	UpdateExportLineStatus(ctx context.Context, id uuid.UUID, input UpdateExportLineStatusInput) (*ExportLine, error)
	LinkProjectCostEntry(ctx context.Context, lineID uuid.UUID, entryID uuid.UUID) error
	ListReconciliation(ctx context.Context, limit int) ([]ReconciliationItem, error)
}

type CostPoster interface {
	CreateCostEntryFromAIUsage(ctx context.Context, projectID uuid.UUID, input project.CreateCostEntryInput) (*project.CostEntry, error)
}

type ObservabilityRecorder interface {
	StartTrace(ctx context.Context, workflowID *uuid.UUID, metadata map[string]any) (*observability.Trace, error)
	RecordSpan(ctx context.Context, input observability.RecordSpanInput) (*observability.Span, error)
	RecordMetric(ctx context.Context, input observability.RecordMetricInput) (*observability.Metric, error)
	CompleteTrace(ctx context.Context, id uuid.UUID, status string) error
}

type Service struct {
	repo          Repository
	httpClient    *http.Client
	costPoster    CostPoster
	observability ObservabilityRecorder
}

type ServiceOption func(*Service)

func WithHTTPClient(client *http.Client) ServiceOption {
	return func(s *Service) {
		if client != nil {
			s.httpClient = client
		}
	}
}

func WithCostPoster(costPoster CostPoster) ServiceOption {
	return func(s *Service) {
		s.costPoster = costPoster
	}
}

func WithObservability(recorder ObservabilityRecorder) ServiceOption {
	return func(s *Service) {
		s.observability = recorder
	}
}

func NewService(repo Repository, opts ...ServiceOption) *Service {
	s := &Service{
		repo:       repo,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Service) CreateAdapter(ctx context.Context, input CreateAdapterInput) (*FinanceAdapter, error) {
	if input.Name == "" || input.EndpointURL == "" || input.Secret == "" {
		return nil, fmt.Errorf("%w: name, endpoint_url, and secret are required", ErrValidation)
	}
	normalizeCreateAdapterInput(&input)
	if err := validateEndpoint(input.EndpointURL); err != nil {
		return nil, err
	}
	if !validAuthType(input.AuthType) {
		return nil, fmt.Errorf("%w: unsupported auth_type %q", ErrValidation, input.AuthType)
	}
	if !validAdapterStatus(input.Status) {
		return nil, fmt.Errorf("%w: unsupported adapter status %q", ErrValidation, input.Status)
	}
	return s.repo.CreateAdapter(ctx, input)
}

func (s *Service) ListAdapters(ctx context.Context, limit int) ([]FinanceAdapter, error) {
	items, err := s.repo.ListAdapters(ctx, normalizeLimit(limit))
	if items == nil {
		items = []FinanceAdapter{}
	}
	return items, err
}

func (s *Service) UpdateAdapter(ctx context.Context, id uuid.UUID, input UpdateAdapterInput) (*FinanceAdapter, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: adapter id is required", ErrValidation)
	}
	if input.EndpointURL != nil {
		if err := validateEndpoint(*input.EndpointURL); err != nil {
			return nil, err
		}
	}
	if input.AuthType != nil && !validAuthType(*input.AuthType) {
		return nil, fmt.Errorf("%w: unsupported auth_type %q", ErrValidation, *input.AuthType)
	}
	if input.Secret != nil && strings.TrimSpace(*input.Secret) == "" {
		return nil, fmt.Errorf("%w: secret cannot be empty", ErrValidation)
	}
	if input.Status != nil && !validAdapterStatus(*input.Status) {
		return nil, fmt.Errorf("%w: unsupported adapter status %q", ErrValidation, *input.Status)
	}
	return s.repo.UpdateAdapter(ctx, id, input)
}

func (s *Service) TestAdapter(ctx context.Context, id uuid.UUID) error {
	adapter, err := s.repo.GetAdapterSecret(ctx, id)
	if err != nil {
		return err
	}
	if adapter.Status == AdapterDisabled {
		return fmt.Errorf("%w: adapter is disabled", ErrValidation)
	}
	payload := map[string]any{
		"format_version": "meta-org.finance.test.v1",
		"adapter_id":     adapter.ID.String(),
		"sent_at":        time.Now().UTC().Format(time.RFC3339),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal adapter test payload: %w", err)
	}
	result, err := s.sendAdapterRequest(ctx, adapter, body, "adapter-test")
	if err != nil {
		status := AdapterError
		_, _ = s.repo.UpdateAdapter(ctx, id, UpdateAdapterInput{Status: &status})
		return fmt.Errorf("test finance adapter: %w", err)
	}
	if result.StatusCode < 200 || result.StatusCode >= 300 {
		status := AdapterError
		_, _ = s.repo.UpdateAdapter(ctx, id, UpdateAdapterInput{Status: &status})
		return fmt.Errorf("%w: finance adapter returned HTTP %d", ErrValidation, result.StatusCode)
	}
	status := AdapterActive
	_, _ = s.repo.UpdateAdapter(ctx, id, UpdateAdapterInput{Status: &status})
	return nil
}

func (s *Service) CreateExportBatch(ctx context.Context, input CreateExportBatchInput) (*ExportBatch, error) {
	if input.AdapterID == uuid.Nil {
		return nil, fmt.Errorf("%w: adapter_id is required", ErrValidation)
	}
	adapter, err := s.repo.GetAdapterSecret(ctx, input.AdapterID)
	if err != nil {
		return nil, err
	}
	if adapter.Status != "" && adapter.Status != AdapterActive {
		return nil, fmt.Errorf("%w: finance adapter is %s", ErrValidation, adapter.Status)
	}
	if err := normalizeExportBatchInput(&input); err != nil {
		return nil, err
	}
	batch, err := s.repo.CreateExportBatch(ctx, input)
	if err != nil {
		return nil, err
	}
	if err := s.postProjectCosts(ctx, batch, input.ActorID, input.ActorType); err != nil {
		_, _ = s.repo.UpdateExportBatchStatus(ctx, batch.ID, UpdateExportBatchStatusInput{
			Status:       BatchFailed,
			ErrorMessage: err.Error(),
			Metadata: map[string]any{
				"failed_stage": "project_cost_posting",
			},
		})
		s.recordFinanceMetric(ctx, "finance_export_failed", &batch.ID, "finance_export_batch", 1, map[string]any{"stage": "project_cost_posting"})
		return nil, err
	}
	return batch, nil
}

func (s *Service) ListExportBatches(ctx context.Context, limit int) ([]ExportBatch, error) {
	items, err := s.repo.ListExportBatches(ctx, normalizeLimit(limit))
	if items == nil {
		items = []ExportBatch{}
	}
	return items, err
}

func (s *Service) GetExportBatch(ctx context.Context, id uuid.UUID) (*ExportBatch, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: batch id is required", ErrValidation)
	}
	return s.repo.GetExportBatch(ctx, id)
}

func (s *Service) SubmitExportBatch(ctx context.Context, id uuid.UUID) (*ExportBatch, error) {
	batch, err := s.repo.GetExportBatch(ctx, id)
	if err != nil {
		return nil, err
	}
	trace := s.startFinanceTrace(ctx, "finance_export", batch.AdapterID, &batch.ID, map[string]any{
		"status":          batch.Status,
		"currency":        batch.Currency,
		"total_amount":    batch.TotalAmount,
		"idempotency_key": batch.IdempotencyKey,
	})
	if batch.Status == BatchCancelled || batch.Status == BatchReconciled {
		message := fmt.Sprintf("batch status %q cannot be submitted", batch.Status)
		s.recordFinanceSpan(ctx, trace, observability.SpanFinanceExport, &batch.ID, "finance_export_batch", map[string]any{"status": batch.Status}, map[string]any{"status": BatchFailed, "error": message}, 0)
		s.completeObservationTrace(ctx, trace, observability.TraceFailed)
		return nil, fmt.Errorf("%w: batch status %q cannot be submitted", ErrValidation, batch.Status)
	}
	if len(batch.Lines) == 0 {
		message := "export batch has no lines"
		s.recordFinanceSpan(ctx, trace, observability.SpanFinanceExport, &batch.ID, "finance_export_batch", map[string]any{"status": batch.Status}, map[string]any{"status": BatchFailed, "error": message}, 0)
		s.completeObservationTrace(ctx, trace, observability.TraceFailed)
		return nil, fmt.Errorf("%w: export batch has no lines", ErrValidation)
	}
	adapter, err := s.repo.GetAdapterSecret(ctx, batch.AdapterID)
	if err != nil {
		s.recordFinanceSpan(ctx, trace, observability.SpanFinanceExport, &batch.ID, "finance_export_batch", map[string]any{"adapter_id": batch.AdapterID.String()}, map[string]any{"status": BatchFailed, "error": err.Error()}, 0)
		s.completeObservationTrace(ctx, trace, observability.TraceFailed)
		return nil, err
	}
	if adapter.Status != AdapterActive {
		message := fmt.Sprintf("finance adapter is %s", adapter.Status)
		s.recordFinanceSpan(ctx, trace, observability.SpanFinanceExport, &batch.ID, "finance_export_batch", map[string]any{"adapter_status": adapter.Status}, map[string]any{"status": BatchFailed, "error": message}, 0)
		s.completeObservationTrace(ctx, trace, observability.TraceFailed)
		return nil, fmt.Errorf("%w: finance adapter is %s", ErrValidation, adapter.Status)
	}
	payload := exportPayload(batch)
	body, err := json.Marshal(payload)
	if err != nil {
		s.recordFinanceSpan(ctx, trace, observability.SpanFinanceExport, &batch.ID, "finance_export_batch", map[string]any{"line_count": len(batch.Lines)}, map[string]any{"status": BatchFailed, "error": err.Error()}, 0)
		s.completeObservationTrace(ctx, trace, observability.TraceFailed)
		return nil, fmt.Errorf("marshal finance export payload: %w", err)
	}
	if _, err := s.repo.UpdateExportBatchStatus(ctx, id, UpdateExportBatchStatusInput{Status: BatchExporting, Submitted: true}); err != nil {
		s.recordFinanceSpan(ctx, trace, observability.SpanFinanceExport, &batch.ID, "finance_export_batch", map[string]any{"line_count": len(batch.Lines)}, map[string]any{"status": BatchFailed, "error": err.Error()}, 0)
		s.completeObservationTrace(ctx, trace, observability.TraceFailed)
		return nil, err
	}
	started := time.Now()
	result, err := s.sendAdapterRequest(ctx, adapter, body, batch.IdempotencyKey)
	durationMS := int(time.Since(started).Milliseconds())
	if err != nil {
		s.recordFinanceSpan(ctx, trace, observability.SpanFinanceExport, &batch.ID, "finance_export_batch", map[string]any{"line_count": len(batch.Lines)}, map[string]any{"status": BatchFailed, "error": err.Error()}, durationMS)
		s.completeObservationTrace(ctx, trace, observability.TraceFailed)
		return s.markExportFailed(ctx, id, fmt.Sprintf("submit finance export: %v", err))
	}
	responseBody := result.Body
	if result.StatusCode < 200 || result.StatusCode >= 300 {
		message := fmt.Sprintf("finance adapter returned HTTP %d", result.StatusCode)
		s.recordFinanceSpan(ctx, trace, observability.SpanFinanceExport, &batch.ID, "finance_export_batch", map[string]any{"line_count": len(batch.Lines)}, map[string]any{"status": BatchFailed, "error": message}, durationMS)
		s.completeObservationTrace(ctx, trace, observability.TraceFailed)
		return s.markExportFailed(ctx, id, fmt.Sprintf("finance adapter returned HTTP %d: %s", result.StatusCode, strings.TrimSpace(string(responseBody))))
	}
	statusInput := UpdateExportBatchStatusInput{
		Status:    BatchExported,
		Submitted: true,
		Metadata: map[string]any{
			"submit_status_code": result.StatusCode,
		},
	}
	var adapterResponse struct {
		ExternalBatchID string         `json:"external_batch_id"`
		Status          string         `json:"status"`
		Metadata        map[string]any `json:"metadata"`
	}
	if len(responseBody) > 0 && json.Unmarshal(responseBody, &adapterResponse) == nil {
		statusInput.ExternalBatchID = adapterResponse.ExternalBatchID
		if mapped := mapExternalBatchStatus(adapterResponse.Status); mapped != "" {
			statusInput.Status = mapped
		}
		for key, value := range adapterResponse.Metadata {
			statusInput.Metadata[key] = value
		}
	}
	updated, err := s.repo.UpdateExportBatchStatus(ctx, id, statusInput)
	if err != nil {
		s.recordFinanceSpan(ctx, trace, observability.SpanFinanceExport, &batch.ID, "finance_export_batch", map[string]any{"line_count": len(batch.Lines)}, map[string]any{"status": BatchFailed, "error": err.Error()}, durationMS)
		s.completeObservationTrace(ctx, trace, observability.TraceFailed)
		return nil, err
	}
	s.recordFinanceSpan(ctx, trace, observability.SpanFinanceExport, &batch.ID, "finance_export_batch", map[string]any{"line_count": len(batch.Lines)}, map[string]any{"status": updated.Status, "status_code": result.StatusCode}, durationMS)
	if updated.Status == BatchFailed {
		s.recordFinanceMetric(ctx, "finance_export_failed", &batch.ID, "finance_export_batch", 1, map[string]any{"stage": "adapter_response"})
		s.completeObservationTrace(ctx, trace, observability.TraceFailed)
	} else {
		s.completeObservationTrace(ctx, trace, observability.TraceComplete)
	}
	return updated, nil
}

func (s *Service) ReceiveWebhook(ctx context.Context, adapterID uuid.UUID, body []byte, signature string, authorization string) (*WebhookEvent, error) {
	if adapterID == uuid.Nil {
		return nil, fmt.Errorf("%w: adapter id is required", ErrValidation)
	}
	adapter, err := s.repo.GetAdapterSecret(ctx, adapterID)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{}
	if len(body) > 0 {
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("%w: invalid webhook payload", ErrValidation)
		}
	}
	trace := s.startFinanceTrace(ctx, "finance_webhook", adapterID, uuidFromPayload(payload, "batch_id"), map[string]any{
		"event_type": stringFromPayload(payload, "event_type", "finance.webhook"),
	})
	started := time.Now()
	valid := verifyAdapterCallback(adapter, body, signature, authorization)
	eventInput := RecordWebhookEventInput{
		AdapterID:      adapterID,
		BatchID:        uuidFromPayload(payload, "batch_id"),
		EventType:      stringFromPayload(payload, "event_type", "finance.webhook"),
		SignatureValid: valid,
		Payload:        payload,
		Processed:      false,
	}
	if !valid {
		eventInput.ErrorMessage = "invalid signature"
		event, _ := s.repo.RecordWebhookEvent(ctx, eventInput)
		s.recordFinanceSpan(ctx, trace, observability.SpanFinanceWebhook, nil, "finance_webhook_event", map[string]any{"adapter_id": adapterID.String()}, map[string]any{"signature_valid": false, "error": eventInput.ErrorMessage}, int(time.Since(started).Milliseconds()))
		s.recordFinanceMetric(ctx, "finance_webhook_invalid_signature", nil, "finance_webhook_event", 1, map[string]any{"adapter_id": adapterID.String()})
		s.completeObservationTrace(ctx, trace, observability.TraceFailed)
		return event, fmt.Errorf("%w: invalid webhook signature", ErrForbidden)
	}
	if eventInput.BatchID != nil {
		if amount, ok := externalAmountFloatFromPayload(payload); ok {
			if batch, err := s.repo.GetExportBatch(ctx, *eventInput.BatchID); err == nil {
				s.recordFinanceMetric(ctx, "finance_reconciliation_difference", eventInput.BatchID, "finance_export_batch", amount-batch.TotalAmount, map[string]any{
					"external_amount": amount,
					"total_amount":    batch.TotalAmount,
					"currency":        batch.Currency,
				})
			}
		}
		status := mapExternalBatchStatus(firstNonEmptyString(payload, "status", "batch_status"))
		if status != "" {
			metadata := map[string]any{
				"last_webhook": payload,
			}
			if amount, ok := externalAmountFromPayload(payload); ok {
				metadata["external_amount"] = amount
			}
			update := UpdateExportBatchStatusInput{
				Status:          status,
				ExternalBatchID: stringFromPayload(payload, "external_batch_id", ""),
				ErrorMessage:    stringFromPayload(payload, "error_message", ""),
				Metadata:        metadata,
			}
			if _, err := s.repo.UpdateExportBatchStatus(ctx, *eventInput.BatchID, update); err != nil {
				eventInput.ErrorMessage = err.Error()
				event, _ := s.repo.RecordWebhookEvent(ctx, eventInput)
				s.recordFinanceSpan(ctx, trace, observability.SpanFinanceWebhook, eventInput.BatchID, "finance_export_batch", map[string]any{"adapter_id": adapterID.String()}, map[string]any{"processed": false, "error": err.Error()}, int(time.Since(started).Milliseconds()))
				s.completeObservationTrace(ctx, trace, observability.TraceFailed)
				return event, err
			}
			if status == BatchFailed {
				s.recordFinanceMetric(ctx, "finance_export_failed", eventInput.BatchID, "finance_export_batch", 1, map[string]any{"stage": "webhook"})
			}
		}
	}
	s.updateWebhookLines(ctx, payload)
	eventInput.Processed = true
	event, err := s.repo.RecordWebhookEvent(ctx, eventInput)
	if err != nil {
		s.recordFinanceSpan(ctx, trace, observability.SpanFinanceWebhook, eventInput.BatchID, "finance_webhook_event", map[string]any{"adapter_id": adapterID.String()}, map[string]any{"processed": false, "error": err.Error()}, int(time.Since(started).Milliseconds()))
		s.completeObservationTrace(ctx, trace, observability.TraceFailed)
		return nil, err
	}
	entityID := &event.ID
	entityType := "finance_webhook_event"
	if eventInput.BatchID != nil {
		entityID = eventInput.BatchID
		entityType = "finance_export_batch"
	}
	s.recordFinanceSpan(ctx, trace, observability.SpanFinanceWebhook, entityID, entityType, map[string]any{"adapter_id": adapterID.String(), "signature_valid": true}, map[string]any{"processed": true, "event_type": event.EventType}, int(time.Since(started).Milliseconds()))
	s.completeObservationTrace(ctx, trace, observability.TraceComplete)
	return event, nil
}

func (s *Service) ListReconciliation(ctx context.Context, limit int) ([]ReconciliationItem, error) {
	items, err := s.repo.ListReconciliation(ctx, normalizeLimit(limit))
	if items == nil {
		items = []ReconciliationItem{}
	}
	return items, err
}

func (s *Service) markExportFailed(ctx context.Context, id uuid.UUID, message string) (*ExportBatch, error) {
	batch, err := s.repo.UpdateExportBatchStatus(ctx, id, UpdateExportBatchStatusInput{
		Status:       BatchFailed,
		ErrorMessage: message,
	})
	if err != nil {
		return nil, err
	}
	s.recordFinanceMetric(ctx, "finance_export_failed", &id, "finance_export_batch", 1, map[string]any{"message": message})
	return batch, fmt.Errorf("%w: %s", ErrValidation, message)
}

func (s *Service) postProjectCosts(ctx context.Context, batch *ExportBatch, actorID *uuid.UUID, actorType string) error {
	if s.costPoster == nil || batch == nil {
		return nil
	}
	for _, line := range batch.Lines {
		if line.ProjectID == nil || line.UsageLedgerID == nil || line.ProjectCostEntryID != nil || line.Amount == 0 {
			continue
		}
		entry, err := s.costPoster.CreateCostEntryFromAIUsage(ctx, *line.ProjectID, project.CreateCostEntryInput{
			ActorInput:  project.ActorInput{ActorID: actorID, ActorType: actorType},
			SourceID:    line.UsageLedgerID,
			Amount:      line.Amount,
			Currency:    line.Currency,
			OccurredAt:  &line.CreatedAt,
			Description: "AI usage cost",
			Metadata: map[string]any{
				"finance_export_batch_id": batch.ID.String(),
				"finance_export_line_id":  line.ID.String(),
			},
		})
		if err != nil {
			return fmt.Errorf("post AI usage to project cost: %w", err)
		}
		if entry != nil && entry.ID != uuid.Nil {
			if err := s.repo.LinkProjectCostEntry(ctx, line.ID, entry.ID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) updateWebhookLines(ctx context.Context, payload map[string]any) {
	raw, ok := payload["lines"].([]any)
	if !ok {
		return
	}
	for _, item := range raw {
		linePayload, ok := item.(map[string]any)
		if !ok {
			continue
		}
		lineID := uuidFromPayload(linePayload, "line_id")
		if lineID == nil {
			continue
		}
		_, _ = s.repo.UpdateExportLineStatus(ctx, *lineID, UpdateExportLineStatusInput{
			Status:         mapLineStatus(stringFromPayload(linePayload, "status", "")),
			ExternalLineID: stringFromPayload(linePayload, "external_line_id", ""),
			Metadata: map[string]any{
				"last_webhook_line": linePayload,
			},
		})
	}
}

type adapterHTTPResult struct {
	StatusCode int
	Body       []byte
}

func (s *Service) sendAdapterRequest(ctx context.Context, adapter AdapterSecret, body []byte, idempotencyKey string) (adapterHTTPResult, error) {
	attempts := adapterAttempts(adapter.RetryCount)
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		reqCtx, cancel := context.WithTimeout(ctx, adapterTimeout(adapter.TimeoutMS))
		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, adapter.EndpointURL, bytes.NewReader(body))
		if err != nil {
			cancel()
			return adapterHTTPResult{}, fmt.Errorf("create finance adapter request: %w", err)
		}
		applyAuthHeaders(req, adapter, body, idempotencyKey)
		resp, err := s.httpClient.Do(req)
		if err != nil {
			cancel()
			lastErr = err
			if ctx.Err() != nil || attempt == attempts-1 {
				break
			}
			sleepBeforeRetry(ctx, attempt)
			continue
		}
		responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		closeErr := resp.Body.Close()
		cancel()
		if readErr != nil {
			return adapterHTTPResult{}, fmt.Errorf("read finance adapter response: %w", readErr)
		}
		if closeErr != nil {
			return adapterHTTPResult{}, fmt.Errorf("close finance adapter response: %w", closeErr)
		}
		result := adapterHTTPResult{StatusCode: resp.StatusCode, Body: responseBody}
		if shouldRetryStatus(resp.StatusCode) && attempt < attempts-1 {
			sleepBeforeRetry(ctx, attempt)
			continue
		}
		return result, nil
	}
	if lastErr == nil {
		lastErr = ctx.Err()
	}
	return adapterHTTPResult{}, lastErr
}

func adapterAttempts(retryCount int) int {
	if retryCount < 0 {
		retryCount = 0
	}
	if retryCount > 5 {
		retryCount = 5
	}
	return retryCount + 1
}

func adapterTimeout(timeoutMS int) time.Duration {
	if timeoutMS <= 0 {
		return 30 * time.Second
	}
	return time.Duration(timeoutMS) * time.Millisecond
}

func shouldRetryStatus(status int) bool {
	return status == http.StatusTooManyRequests || status >= 500
}

func sleepBeforeRetry(ctx context.Context, attempt int) {
	timer := time.NewTimer(time.Duration(attempt+1) * 100 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func normalizeCreateAdapterInput(input *CreateAdapterInput) {
	input.Name = strings.TrimSpace(input.Name)
	input.EndpointURL = strings.TrimSpace(input.EndpointURL)
	if input.AuthType == "" {
		input.AuthType = AuthHMAC
	}
	if input.Status == "" {
		input.Status = AdapterActive
	}
	if input.TimeoutMS <= 0 {
		input.TimeoutMS = 30000
	}
	if input.RetryCount < 0 {
		input.RetryCount = 0
	}
	if input.RetryCount == 0 {
		input.RetryCount = 3
	}
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
}

func normalizeExportBatchInput(input *CreateExportBatchInput) error {
	start, err := parseDate(input.PeriodStart)
	if err != nil {
		return err
	}
	end, err := parseDate(input.PeriodEnd)
	if err != nil {
		return err
	}
	if end.Before(start) {
		return fmt.Errorf("%w: period_end must be on or after period_start", ErrValidation)
	}
	input.periodStartTime = start
	input.periodEndTime = end
	if input.Currency == "" {
		input.Currency = "CNY"
	}
	input.Currency = strings.ToUpper(strings.TrimSpace(input.Currency))
	if input.IdempotencyKey == "" {
		input.IdempotencyKey = BatchIdempotencyKey(input.AdapterID.String(), input.PeriodStart, input.PeriodEnd, input.Currency)
	}
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
	return nil
}

func parseDate(raw string) (time.Time, error) {
	value, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: date must use YYYY-MM-DD", ErrValidation)
	}
	return value.UTC(), nil
}

func validateEndpoint(endpoint string) error {
	parsed, err := url.ParseRequestURI(endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%w: endpoint_url must be an absolute URL", ErrValidation)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%w: endpoint_url scheme must be http or https", ErrValidation)
	}
	return nil
}

func validAuthType(authType string) bool {
	return authType == AuthHMAC || authType == AuthBearer
}

func validAdapterStatus(status string) bool {
	return status == AdapterActive || status == AdapterDisabled || status == AdapterError
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func exportPayload(batch *ExportBatch) map[string]any {
	lines := make([]map[string]any, 0, len(batch.Lines))
	for _, line := range batch.Lines {
		lines = append(lines, map[string]any{
			"line_id":               line.ID.String(),
			"usage_ledger_id":       uuidString(line.UsageLedgerID),
			"cost_ledger_entry_id":  uuidString(line.CostLedgerEntryID),
			"project_cost_entry_id": uuidString(line.ProjectCostEntryID),
			"organization_id":       uuidString(line.OrganizationID),
			"department_id":         uuidString(line.DepartmentID),
			"project_id":            uuidString(line.ProjectID),
			"provider_id":           uuidString(line.ProviderID),
			"model_id":              uuidString(line.ModelID),
			"amount":                line.Amount,
			"currency":              line.Currency,
			"metadata":              line.Metadata,
		})
	}
	return map[string]any{
		"format_version":  "meta-org.finance.export.v1",
		"batch_id":        batch.ID.String(),
		"adapter_id":      batch.AdapterID.String(),
		"period_start":    batch.PeriodStart.Format("2006-01-02"),
		"period_end":      batch.PeriodEnd.Format("2006-01-02"),
		"currency":        batch.Currency,
		"total_amount":    batch.TotalAmount,
		"idempotency_key": batch.IdempotencyKey,
		"metadata":        batch.Metadata,
		"lines":           lines,
	}
}

func applyAuthHeaders(req *http.Request, adapter AdapterSecret, body []byte, idempotencyKey string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", idempotencyKey)
	req.Header.Set("X-Meta-Org-Adapter-ID", adapter.ID.String())
	if adapter.AuthType == AuthBearer {
		req.Header.Set("Authorization", "Bearer "+adapter.Secret)
		return
	}
	req.Header.Set("X-Meta-Org-Signature", SignPayload(body, adapter.Secret))
}

func verifyAdapterCallback(adapter AdapterSecret, body []byte, signature string, authorization string) bool {
	if adapter.AuthType == AuthBearer {
		return strings.TrimSpace(authorization) == "Bearer "+adapter.Secret
	}
	return VerifyPayload(body, signature, adapter.Secret)
}

func mapExternalBatchStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "":
		return ""
	case "exported", "submitted":
		return BatchExported
	case "accepted", "acknowledged":
		return BatchAcknowledged
	case "posted":
		return BatchPosted
	case "reconciled":
		return BatchReconciled
	case "rejected", "failed", "error":
		return BatchFailed
	default:
		return ""
	}
}

func mapLineStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "posted", "reconciled", "failed", "rejected", "exported", "acknowledged":
		return strings.ToLower(strings.TrimSpace(status))
	default:
		return "ready"
	}
}

func uuidFromPayload(payload map[string]any, key string) *uuid.UUID {
	raw := stringFromPayload(payload, key, "")
	if raw == "" {
		return nil
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &parsed
}

func stringFromPayload(payload map[string]any, key string, fallback string) string {
	if value, ok := payload[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return fallback
}

func firstNonEmptyString(payload map[string]any, keys ...string) string {
	for _, key := range keys {
		if value := stringFromPayload(payload, key, ""); value != "" {
			return value
		}
	}
	return ""
}

func externalAmountFromPayload(payload map[string]any) (any, bool) {
	for _, key := range []string{"external_amount", "reconciled_amount", "posted_amount"} {
		if value, ok := payload[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func externalAmountFloatFromPayload(payload map[string]any) (float64, bool) {
	value, ok := externalAmountFromPayload(payload)
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

func uuidString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func (s *Service) startFinanceTrace(ctx context.Context, category string, adapterID uuid.UUID, batchID *uuid.UUID, metadata map[string]any) *observability.Trace {
	if s.observability == nil {
		return nil
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["category"] = category
	metadata["adapter_id"] = adapterID.String()
	metadata["batch_id"] = uuidString(batchID)
	trace, err := s.observability.StartTrace(ctx, nil, metadata)
	if err != nil {
		return nil
	}
	return trace
}

func (s *Service) recordFinanceSpan(ctx context.Context, trace *observability.Trace, spanType observability.SpanType, entityID *uuid.UUID, entityType string, input map[string]any, output map[string]any, durationMS int) {
	if s.observability == nil || trace == nil {
		return
	}
	_, _ = s.observability.RecordSpan(ctx, observability.RecordSpanInput{
		TraceID:    trace.ID,
		SpanType:   spanType,
		EntityID:   entityID,
		EntityType: entityType,
		Input:      input,
		Output:     output,
		DurationMs: durationMS,
		Metadata: map[string]any{
			"category": string(spanType),
		},
	})
}

func (s *Service) recordFinanceMetric(ctx context.Context, name string, entityID *uuid.UUID, entityType string, value float64, metadata map[string]any) {
	if s.observability == nil {
		return
	}
	metricType := observability.MetricHealth
	if name == "finance_reconciliation_difference" {
		metricType = observability.MetricCost
	}
	_, _ = s.observability.RecordMetric(ctx, observability.RecordMetricInput{
		MetricType: metricType,
		MetricName: name,
		EntityID:   entityID,
		EntityType: entityType,
		Value:      value,
		Metadata:   metadata,
	})
}

func (s *Service) completeObservationTrace(ctx context.Context, trace *observability.Trace, status observability.TraceStatus) {
	if s.observability == nil || trace == nil {
		return
	}
	_ = s.observability.CompleteTrace(ctx, trace.ID, string(status))
}
