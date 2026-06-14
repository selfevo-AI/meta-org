package aigateway

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type secretBox interface {
	Encrypt(string) (string, error)
	Decrypt(string) (string, error)
}

type PostgresRepository struct {
	db  *pgxpool.Pool
	box secretBox
}

func NewRepository(db *pgxpool.Pool, box secretBox) *PostgresRepository {
	return &PostgresRepository{db: db, box: box}
}

func (r *PostgresRepository) CreateProvider(ctx context.Context, input CreateProviderInput) (*ModelProvider, error) {
	encrypted, err := r.box.Encrypt(input.APIKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt provider api key: %w", err)
	}
	tagsJSON, metaJSON, err := providerJSON(input.Tags, input.Metadata)
	if err != nil {
		return nil, err
	}
	provider := &ModelProvider{}
	err = scanProviderRow(r.db.QueryRow(ctx, `
		INSERT INTO model_providers (
			name, provider_type, base_url, encrypted_api_key, masked_api_key,
			status, timeout_ms, retry_count, risk_level, tags, metadata
		)
		VALUES ($1, $2, $3, $4, $5, COALESCE(NULLIF($6, ''), 'active'), COALESCE(NULLIF($7, 0), 60000), COALESCE(NULLIF($8, 0), 1), COALESCE(NULLIF($9, ''), 'medium'), $10, $11)
		RETURNING id, name, provider_type, base_url, masked_api_key, status, timeout_ms, retry_count,
			risk_level, tags, metadata, last_test_status, last_test_error, last_tested_at, created_at, updated_at
	`, input.Name, input.ProviderType, input.BaseURL, encrypted, maskSecret(input.APIKey), input.Status, input.TimeoutMS, input.RetryCount, input.RiskLevel, tagsJSON, metaJSON), provider)
	if err != nil {
		return nil, fmt.Errorf("create model provider: %w", err)
	}
	return provider, nil
}

func (r *PostgresRepository) ListProviders(ctx context.Context, limit int) ([]ModelProvider, error) {
	limit = normalizeLimit(limit)
	rows, err := r.db.Query(ctx, `
		SELECT id, name, provider_type, base_url, masked_api_key, status, timeout_ms, retry_count,
			risk_level, tags, metadata, last_test_status, last_test_error, last_tested_at, created_at, updated_at
		FROM model_providers
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list model providers: %w", err)
	}
	defer rows.Close()
	return scanProviders(rows)
}

func (r *PostgresRepository) UpdateProvider(ctx context.Context, id uuid.UUID, input UpdateProviderInput) (*ModelProvider, error) {
	tagsJSON, metaJSON, err := providerJSON(input.Tags, input.Metadata)
	if err != nil {
		return nil, err
	}
	provider := &ModelProvider{}
	err = scanProviderRow(r.db.QueryRow(ctx, `
		UPDATE model_providers
		SET name = COALESCE($2, name),
			base_url = COALESCE($3, base_url),
			status = COALESCE($4, status),
			timeout_ms = COALESCE($5, timeout_ms),
			retry_count = COALESCE($6, retry_count),
			risk_level = COALESCE($7, risk_level),
			tags = CASE WHEN $8::jsonb IS NULL THEN tags ELSE $8::jsonb END,
			metadata = CASE WHEN $9::jsonb IS NULL THEN metadata ELSE $9::jsonb END,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, provider_type, base_url, masked_api_key, status, timeout_ms, retry_count,
			risk_level, tags, metadata, last_test_status, last_test_error, last_tested_at, created_at, updated_at
	`, id, input.Name, input.BaseURL, input.Status, input.TimeoutMS, input.RetryCount, input.RiskLevel, nullableJSON(tagsJSON, input.Tags != nil), nullableJSON(metaJSON, input.Metadata != nil)), provider)
	if err != nil {
		return nil, fmt.Errorf("update model provider: %w", err)
	}
	return provider, nil
}

func (r *PostgresRepository) RotateProviderKey(ctx context.Context, id uuid.UUID, apiKey string) (*ModelProvider, error) {
	encrypted, err := r.box.Encrypt(apiKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt provider api key: %w", err)
	}
	provider := &ModelProvider{}
	err = scanProviderRow(r.db.QueryRow(ctx, `
		UPDATE model_providers
		SET encrypted_api_key = $2, masked_api_key = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, provider_type, base_url, masked_api_key, status, timeout_ms, retry_count,
			risk_level, tags, metadata, last_test_status, last_test_error, last_tested_at, created_at, updated_at
	`, id, encrypted, maskSecret(apiKey)), provider)
	if err != nil {
		return nil, fmt.Errorf("rotate model provider key: %w", err)
	}
	return provider, nil
}

func (r *PostgresRepository) UpdateProviderTestResult(ctx context.Context, id uuid.UUID, status string, message string) error {
	if _, err := r.db.Exec(ctx, `
		UPDATE model_providers
		SET last_test_status = $2, last_test_error = $3, last_tested_at = NOW(), updated_at = NOW()
		WHERE id = $1
	`, id, status, message); err != nil {
		return fmt.Errorf("update provider test result: %w", err)
	}
	return nil
}

func (r *PostgresRepository) GetProviderSecret(ctx context.Context, id uuid.UUID) (ProviderSecret, error) {
	var provider ProviderSecret
	var encrypted string
	err := r.db.QueryRow(ctx, `
		SELECT id, name, provider_type, base_url, encrypted_api_key, status
		FROM model_providers WHERE id = $1
	`, id).Scan(&provider.ID, &provider.Name, &provider.ProviderType, &provider.BaseURL, &encrypted, &provider.Status)
	if err != nil {
		return provider, fmt.Errorf("get provider secret: %w", err)
	}
	apiKey, err := r.box.Decrypt(encrypted)
	if err != nil {
		return provider, fmt.Errorf("decrypt provider api key: %w", err)
	}
	provider.APIKey = apiKey
	return provider, nil
}

func (r *PostgresRepository) CreateModel(ctx context.Context, input CreateModelInput) (*Model, error) {
	capsJSON, metaJSON, err := modelJSON(input.Capabilities, input.Metadata)
	if err != nil {
		return nil, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin create model: %w", err)
	}
	defer tx.Rollback(ctx)

	model := &Model{}
	err = scanModelRow(tx.QueryRow(ctx, `
		INSERT INTO models (
			provider_id, model_key, display_name, context_window, max_output_tokens,
			capabilities, status, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, COALESCE(NULLIF($7, ''), 'active'), $8)
		RETURNING id, provider_id, model_key, display_name, context_window, max_output_tokens,
			capabilities, status, metadata, created_at, updated_at
	`, input.ProviderID, input.ModelKey, input.DisplayName, input.ContextWindow, input.MaxOutputTokens, capsJSON, input.Status, metaJSON), model)
	if err != nil {
		return nil, fmt.Errorf("create model: %w", err)
	}
	if input.InputPricePer1K > 0 || input.OutputPricePer1K > 0 {
		currency := currencyOrDefault(input.Currency)
		if _, err := tx.Exec(ctx, `
			INSERT INTO model_price_versions (model_id, input_price_per_1k, output_price_per_1k, currency)
			VALUES ($1, $2, $3, $4)
		`, model.ID, input.InputPricePer1K, input.OutputPricePer1K, currency); err != nil {
			return nil, fmt.Errorf("create model price version: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create model: %w", err)
	}
	return model, nil
}

func (r *PostgresRepository) ListModels(ctx context.Context, providerID *uuid.UUID, limit int) ([]Model, error) {
	limit = normalizeLimit(limit)
	rows, err := r.db.Query(ctx, `
		SELECT id, provider_id, model_key, display_name, context_window, max_output_tokens,
			capabilities, status, metadata, created_at, updated_at
		FROM models
		WHERE ($1::uuid IS NULL OR provider_id = $1)
		ORDER BY created_at DESC
		LIMIT $2
	`, providerID, limit)
	if err != nil {
		return nil, fmt.Errorf("list models: %w", err)
	}
	defer rows.Close()
	return scanModels(rows)
}

func (r *PostgresRepository) UpdateModel(ctx context.Context, id uuid.UUID, input UpdateModelInput) (*Model, error) {
	capsJSON, metaJSON, err := modelJSON(input.Capabilities, input.Metadata)
	if err != nil {
		return nil, err
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin update model: %w", err)
	}
	defer tx.Rollback(ctx)

	model := &Model{}
	err = scanModelRow(tx.QueryRow(ctx, `
		UPDATE models
		SET display_name = COALESCE($2, display_name),
			context_window = COALESCE($3, context_window),
			max_output_tokens = COALESCE($4, max_output_tokens),
			capabilities = CASE WHEN $5::jsonb IS NULL THEN capabilities ELSE $5::jsonb END,
			status = COALESCE($6, status),
			metadata = CASE WHEN $7::jsonb IS NULL THEN metadata ELSE $7::jsonb END,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, provider_id, model_key, display_name, context_window, max_output_tokens,
			capabilities, status, metadata, created_at, updated_at
	`, id, input.DisplayName, input.ContextWindow, input.MaxOutputTokens, nullableJSON(capsJSON, input.Capabilities != nil), input.Status, nullableJSON(metaJSON, input.Metadata != nil)), model)
	if err != nil {
		return nil, fmt.Errorf("update model: %w", err)
	}
	if input.InputPricePer1K != nil || input.OutputPricePer1K != nil || input.Currency != nil {
		if _, err := tx.Exec(ctx, `UPDATE model_price_versions SET effective_to = NOW() WHERE model_id = $1 AND effective_to IS NULL`, id); err != nil {
			return nil, fmt.Errorf("close model price version: %w", err)
		}
		inputPrice := 0.0
		outputPrice := 0.0
		if input.InputPricePer1K != nil {
			inputPrice = *input.InputPricePer1K
		}
		if input.OutputPricePer1K != nil {
			outputPrice = *input.OutputPricePer1K
		}
		currency := "CNY"
		if input.Currency != nil {
			currency = *input.Currency
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO model_price_versions (model_id, input_price_per_1k, output_price_per_1k, currency)
			VALUES ($1, $2, $3, $4)
		`, id, inputPrice, outputPrice, currency); err != nil {
			return nil, fmt.Errorf("create model price version: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit update model: %w", err)
	}
	return model, nil
}

func (r *PostgresRepository) ResolveInvocationTarget(ctx context.Context, input InvokeInput) (ResolvedModel, error) {
	var target ResolvedModel
	var encrypted string
	var inputPrice, outputPrice float64
	var priceVersionID pgtype.UUID
	var currency string
	err := r.db.QueryRow(ctx, `
		SELECT p.id, m.id, p.provider_type, p.base_url, p.encrypted_api_key, m.model_key,
			pv.id, COALESCE(pv.input_price_per_1k, 0)::float8, COALESCE(pv.output_price_per_1k, 0)::float8, COALESCE(pv.currency, 'CNY'),
			m.max_output_tokens
		FROM model_providers p
		JOIN models m ON m.provider_id = p.id
		LEFT JOIN LATERAL (
			SELECT id, input_price_per_1k, output_price_per_1k, currency
			FROM model_price_versions
			WHERE model_id = m.id AND (effective_to IS NULL OR effective_to > NOW())
			ORDER BY effective_from DESC
			LIMIT 1
		) pv ON true
		WHERE p.status = 'active'
		  AND m.status = 'active'
		  AND (($1::uuid IS NULL AND p.provider_type = $2) OR p.id = $1)
		  AND (($3::uuid IS NULL AND m.model_key = $4) OR m.id = $3)
		LIMIT 1
	`, input.ProviderID, input.ProviderType, input.ModelID, input.Model).
		Scan(&target.ProviderID, &target.ModelID, &target.ProviderType, &target.BaseURL, &encrypted, &target.Model, &priceVersionID, &inputPrice, &outputPrice, &currency, &target.MaxOutputTokens)
	if err != nil {
		return target, fmt.Errorf("resolve invocation target: %w", err)
	}
	apiKey, err := r.box.Decrypt(encrypted)
	if err != nil {
		return target, fmt.Errorf("decrypt provider api key: %w", err)
	}
	target.APIKey = apiKey
	if priceVersionID.Valid {
		id, err := uuid.FromBytes(priceVersionID.Bytes[:])
		if err != nil {
			return target, fmt.Errorf("decode price version id: %w", err)
		}
		target.PriceVersionID = &id
	}
	target.Price.InputPer1K = inputPrice
	target.Price.OutputPer1K = outputPrice
	target.Currency = currency
	return target, nil
}

func (r *PostgresRepository) CreateInvocation(ctx context.Context, input CreateInvocationInput) (*Invocation, error) {
	inv := &Invocation{}
	err := scanInvocationRow(r.db.QueryRow(ctx, `
		INSERT INTO ai_invocations (
			provider_id, model_id, mode, status, organization_id, department_id, project_id,
			requirement_id, workflow_id, task_id, agent_id, user_id, capability_id,
			source_surface, request_hash, estimated_input_tokens, estimated_output_tokens, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id, provider_id, model_id, mode, status,
			organization_id, department_id, project_id, requirement_id, workflow_id, task_id,
			agent_id, user_id, capability_id, source_surface,
			request_hash, provider_request_id,
			input_tokens, output_tokens, estimated_input_tokens, estimated_output_tokens,
			cost_amount::float8, currency, first_token_ms, duration_ms, error_type, error_message,
			metadata, created_at, completed_at
	`, input.ProviderID, input.ModelID, input.Mode, input.Status, input.Attribution.OrganizationID, input.Attribution.DepartmentID, input.Attribution.ProjectID,
		input.Attribution.RequirementID, input.Attribution.WorkflowID, input.Attribution.TaskID, input.Attribution.AgentID, input.Attribution.UserID,
		input.Attribution.CapabilityID, input.Attribution.SourceSurface, input.RequestHash, input.EstimatedInputTokens, input.EstimatedOutputTokens, mustJSON(input.Metadata)), inv)
	if err != nil {
		return nil, fmt.Errorf("create ai invocation: %w", err)
	}
	inv.Attribution = input.Attribution
	return inv, nil
}

func (r *PostgresRepository) CompleteInvocation(ctx context.Context, id uuid.UUID, input CompleteInvocationInput) error {
	if _, err := r.db.Exec(ctx, `
		UPDATE ai_invocations
		SET status = $2,
			provider_request_id = COALESCE(NULLIF($3, ''), provider_request_id),
			input_tokens = $4,
			output_tokens = $5,
			cost_amount = $6,
			currency = $7,
			duration_ms = $8,
			completed_at = NOW()
		WHERE id = $1
	`, id, StatusCompleted, input.ProviderRequestID, input.Usage.InputTokens, input.Usage.OutputTokens, input.CostAmount, currencyOrDefault(input.Currency), input.DurationMS); err != nil {
		return fmt.Errorf("complete ai invocation: %w", err)
	}
	return nil
}

func (r *PostgresRepository) FailInvocation(ctx context.Context, id uuid.UUID, input FailInvocationInput) error {
	status := StatusFailed
	if input.Cancelled {
		status = StatusCancelled
	}
	if _, err := r.db.Exec(ctx, `
		UPDATE ai_invocations
		SET status = $2, error_type = $3, error_message = $4, duration_ms = $5, completed_at = NOW()
		WHERE id = $1
	`, id, status, input.ErrorType, input.Message, input.DurationMS); err != nil {
		return fmt.Errorf("fail ai invocation: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CreateUsageLedger(ctx context.Context, input CreateUsageLedgerInput) error {
	if _, err := r.db.Exec(ctx, `
		INSERT INTO ai_usage_ledger (
			invocation_id, model_price_version_id, ledger_type, amount, currency,
			input_tokens, output_tokens, reason
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, input.InvocationID, input.ModelPriceVersionID, input.LedgerType, input.Amount, currencyOrDefault(input.Currency), input.Usage.InputTokens, input.Usage.OutputTokens, input.Reason); err != nil {
		return fmt.Errorf("create ai usage ledger: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListInvocations(ctx context.Context, limit int) ([]Invocation, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, provider_id, model_id, mode, status,
			organization_id, department_id, project_id, requirement_id, workflow_id, task_id,
			agent_id, user_id, capability_id, source_surface,
			request_hash, provider_request_id,
			input_tokens, output_tokens, estimated_input_tokens, estimated_output_tokens,
			cost_amount::float8, currency, first_token_ms, duration_ms, error_type, error_message,
			metadata, created_at, completed_at
		FROM ai_invocations
		ORDER BY created_at DESC
		LIMIT $1
	`, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list ai invocations: %w", err)
	}
	defer rows.Close()
	return scanInvocations(rows)
}

func (r *PostgresRepository) GetInvocation(ctx context.Context, id uuid.UUID) (*Invocation, error) {
	inv := &Invocation{}
	err := scanInvocationRow(r.db.QueryRow(ctx, `
		SELECT id, provider_id, model_id, mode, status,
			organization_id, department_id, project_id, requirement_id, workflow_id, task_id,
			agent_id, user_id, capability_id, source_surface,
			request_hash, provider_request_id,
			input_tokens, output_tokens, estimated_input_tokens, estimated_output_tokens,
			cost_amount::float8, currency, first_token_ms, duration_ms, error_type, error_message,
			metadata, created_at, completed_at
		FROM ai_invocations
		WHERE id = $1
	`, id), inv)
	if err != nil {
		return nil, fmt.Errorf("get ai invocation: %w", err)
	}
	return inv, nil
}

func (r *PostgresRepository) CostSummary(ctx context.Context) (*GatewayCostSummary, error) {
	summary := &GatewayCostSummary{Currency: "CNY", ByProvider: map[string]float64{}}
	if err := r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount), 0)::float8,
			COALESCE(SUM(amount) FILTER (WHERE finance_export_line_id IS NULL), 0)::float8,
			COALESCE(MAX(currency), 'CNY')
		FROM ai_usage_ledger
	`).Scan(&summary.Total, &summary.Unexported, &summary.Currency); err != nil {
		return nil, fmt.Errorf("query ai cost summary: %w", err)
	}
	rows, err := r.db.Query(ctx, `
		SELECT p.provider_type, COALESCE(SUM(l.amount), 0)::float8
		FROM ai_usage_ledger l
		JOIN ai_invocations i ON i.id = l.invocation_id
		JOIN model_providers p ON p.id = i.provider_id
		GROUP BY p.provider_type
	`)
	if err != nil {
		return nil, fmt.Errorf("query ai cost by provider: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var provider string
		var amount float64
		if err := rows.Scan(&provider, &amount); err != nil {
			return nil, fmt.Errorf("scan ai cost by provider: %w", err)
		}
		summary.ByProvider[provider] = amount
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ai cost by provider: %w", err)
	}
	return summary, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanProviderRow(row scanner, provider *ModelProvider) error {
	var tagsJSON, metaJSON []byte
	var lastTestedAt pgtype.Timestamptz
	if err := row.Scan(&provider.ID, &provider.Name, &provider.ProviderType, &provider.BaseURL, &provider.MaskedAPIKey, &provider.Status,
		&provider.TimeoutMS, &provider.RetryCount, &provider.RiskLevel, &tagsJSON, &metaJSON, &provider.LastTestStatus,
		&provider.LastTestError, &lastTestedAt, &provider.CreatedAt, &provider.UpdatedAt); err != nil {
		return err
	}
	if lastTestedAt.Valid {
		t := lastTestedAt.Time
		provider.LastTestedAt = &t
	}
	return hydrateProviderJSON(provider, tagsJSON, metaJSON)
}

func scanModelRow(row scanner, model *Model) error {
	var capJSON, metaJSON []byte
	if err := row.Scan(&model.ID, &model.ProviderID, &model.ModelKey, &model.DisplayName, &model.ContextWindow,
		&model.MaxOutputTokens, &capJSON, &model.Status, &metaJSON, &model.CreatedAt, &model.UpdatedAt); err != nil {
		return err
	}
	return hydrateModelJSON(model, capJSON, metaJSON)
}

func scanInvocationRow(row scanner, inv *Invocation) error {
	var metaJSON []byte
	var completedAt pgtype.Timestamptz
	var organizationID, departmentID, projectID, requirementID, workflowID, taskID pgtype.UUID
	var agentID, userID, capabilityID pgtype.UUID
	if err := row.Scan(&inv.ID, &inv.ProviderID, &inv.ModelID, &inv.Mode, &inv.Status,
		&organizationID, &departmentID, &projectID, &requirementID, &workflowID, &taskID,
		&agentID, &userID, &capabilityID, &inv.Attribution.SourceSurface,
		&inv.RequestHash, &inv.ProviderRequestID,
		&inv.InputTokens, &inv.OutputTokens, &inv.EstimatedInputTokens, &inv.EstimatedOutputTokens,
		&inv.CostAmount, &inv.Currency, &inv.FirstTokenMS, &inv.DurationMS, &inv.ErrorType, &inv.ErrorMessage,
		&metaJSON, &inv.CreatedAt, &completedAt); err != nil {
		return err
	}
	inv.Attribution.OrganizationID = uuidPointer(organizationID)
	inv.Attribution.DepartmentID = uuidPointer(departmentID)
	inv.Attribution.ProjectID = uuidPointer(projectID)
	inv.Attribution.RequirementID = uuidPointer(requirementID)
	inv.Attribution.WorkflowID = uuidPointer(workflowID)
	inv.Attribution.TaskID = uuidPointer(taskID)
	inv.Attribution.AgentID = uuidPointer(agentID)
	inv.Attribution.UserID = uuidPointer(userID)
	inv.Attribution.CapabilityID = uuidPointer(capabilityID)
	if completedAt.Valid {
		t := completedAt.Time
		inv.CompletedAt = &t
	}
	if err := json.Unmarshal(metaJSON, &inv.Metadata); err != nil {
		return fmt.Errorf("unmarshal invocation metadata: %w", err)
	}
	return nil
}

func uuidPointer(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	id, err := uuid.FromBytes(value.Bytes[:])
	if err != nil {
		return nil
	}
	return &id
}

func scanProviders(rows pgx.Rows) ([]ModelProvider, error) {
	items := []ModelProvider{}
	for rows.Next() {
		var item ModelProvider
		if err := scanProviderRow(rows, &item); err != nil {
			return nil, fmt.Errorf("scan model provider: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanModels(rows pgx.Rows) ([]Model, error) {
	items := []Model{}
	for rows.Next() {
		var item Model
		if err := scanModelRow(rows, &item); err != nil {
			return nil, fmt.Errorf("scan model: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanInvocations(rows pgx.Rows) ([]Invocation, error) {
	items := []Invocation{}
	for rows.Next() {
		var item Invocation
		if err := scanInvocationRow(rows, &item); err != nil {
			return nil, fmt.Errorf("scan invocation: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func providerJSON(tags []string, metadata map[string]any) ([]byte, []byte, error) {
	if tags == nil {
		tags = []string{}
	}
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal provider tags: %w", err)
	}
	metaJSON, err := json.Marshal(nonNilMap(metadata))
	if err != nil {
		return nil, nil, fmt.Errorf("marshal provider metadata: %w", err)
	}
	return tagsJSON, metaJSON, nil
}

func modelJSON(capabilities []string, metadata map[string]any) ([]byte, []byte, error) {
	if capabilities == nil {
		capabilities = []string{}
	}
	capJSON, err := json.Marshal(capabilities)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal model capabilities: %w", err)
	}
	metaJSON, err := json.Marshal(nonNilMap(metadata))
	if err != nil {
		return nil, nil, fmt.Errorf("marshal model metadata: %w", err)
	}
	return capJSON, metaJSON, nil
}

func hydrateProviderJSON(provider *ModelProvider, tagsJSON, metaJSON []byte) error {
	if err := json.Unmarshal(tagsJSON, &provider.Tags); err != nil {
		return fmt.Errorf("unmarshal provider tags: %w", err)
	}
	if err := json.Unmarshal(metaJSON, &provider.Metadata); err != nil {
		return fmt.Errorf("unmarshal provider metadata: %w", err)
	}
	return nil
}

func hydrateModelJSON(model *Model, capJSON, metaJSON []byte) error {
	if err := json.Unmarshal(capJSON, &model.Capabilities); err != nil {
		return fmt.Errorf("unmarshal model capabilities: %w", err)
	}
	if err := json.Unmarshal(metaJSON, &model.Metadata); err != nil {
		return fmt.Errorf("unmarshal model metadata: %w", err)
	}
	return nil
}

func nonNilMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	return input
}

func nullableJSON(data []byte, include bool) any {
	if !include {
		return nil
	}
	return data
}

func mustJSON(input map[string]any) []byte {
	data, err := json.Marshal(nonNilMap(input))
	if err != nil {
		return []byte("{}")
	}
	return data
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

func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return "****"
	}
	return secret[:4] + "****" + secret[len(secret)-4:]
}
