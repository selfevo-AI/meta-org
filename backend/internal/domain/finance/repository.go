package finance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

func (r *PostgresRepository) CreateAdapter(ctx context.Context, input CreateAdapterInput) (*FinanceAdapter, error) {
	encrypted, err := r.box.Encrypt(input.Secret)
	if err != nil {
		return nil, fmt.Errorf("encrypt finance adapter secret: %w", err)
	}
	adapter := &FinanceAdapter{}
	err = scanAdapter(r.db.QueryRow(ctx, `
		INSERT INTO finance_adapters (
			name, endpoint_url, auth_type, encrypted_secret, masked_secret,
			status, timeout_ms, retry_count, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, name, endpoint_url, auth_type, masked_secret, status,
			timeout_ms, retry_count, metadata, created_at, updated_at
	`, input.Name, input.EndpointURL, input.AuthType, encrypted, maskSecret(input.Secret), input.Status,
		input.TimeoutMS, input.RetryCount, mustJSON(input.Metadata)), adapter)
	if err != nil {
		return nil, fmt.Errorf("create finance adapter: %w", err)
	}
	return adapter, nil
}

func (r *PostgresRepository) ListAdapters(ctx context.Context, limit int) ([]FinanceAdapter, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, endpoint_url, auth_type, masked_secret, status,
			timeout_ms, retry_count, metadata, created_at, updated_at
		FROM finance_adapters
		ORDER BY created_at DESC
		LIMIT $1
	`, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list finance adapters: %w", err)
	}
	defer rows.Close()
	adapters := []FinanceAdapter{}
	for rows.Next() {
		var adapter FinanceAdapter
		if err := scanAdapter(rows, &adapter); err != nil {
			return nil, fmt.Errorf("scan finance adapter: %w", err)
		}
		adapters = append(adapters, adapter)
	}
	return adapters, rows.Err()
}

func (r *PostgresRepository) UpdateAdapter(ctx context.Context, id uuid.UUID, input UpdateAdapterInput) (*FinanceAdapter, error) {
	var encryptedSecret *string
	var maskedSecret *string
	if input.Secret != nil {
		encrypted, err := r.box.Encrypt(*input.Secret)
		if err != nil {
			return nil, fmt.Errorf("encrypt finance adapter secret: %w", err)
		}
		masked := maskSecret(*input.Secret)
		encryptedSecret = &encrypted
		maskedSecret = &masked
	}
	adapter := &FinanceAdapter{}
	err := scanAdapter(r.db.QueryRow(ctx, `
		UPDATE finance_adapters
		SET name = COALESCE($2, name),
			endpoint_url = COALESCE($3, endpoint_url),
			auth_type = COALESCE($4, auth_type),
			encrypted_secret = COALESCE($5, encrypted_secret),
			masked_secret = COALESCE($6, masked_secret),
			status = COALESCE($7, status),
			timeout_ms = COALESCE($8, timeout_ms),
			retry_count = COALESCE($9, retry_count),
			metadata = metadata || COALESCE($10::jsonb, '{}'::jsonb),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, endpoint_url, auth_type, masked_secret, status,
			timeout_ms, retry_count, metadata, created_at, updated_at
	`, id, input.Name, input.EndpointURL, input.AuthType, encryptedSecret, maskedSecret, input.Status,
		input.TimeoutMS, input.RetryCount, nullableJSON(input.Metadata)), adapter)
	if err != nil {
		return nil, fmt.Errorf("update finance adapter: %w", err)
	}
	return adapter, nil
}

func (r *PostgresRepository) GetAdapterSecret(ctx context.Context, id uuid.UUID) (AdapterSecret, error) {
	var adapter AdapterSecret
	var encrypted string
	err := r.db.QueryRow(ctx, `
		SELECT id, name, endpoint_url, auth_type, encrypted_secret, status, timeout_ms, retry_count
		FROM finance_adapters
		WHERE id = $1
	`, id).Scan(&adapter.ID, &adapter.Name, &adapter.EndpointURL, &adapter.AuthType, &encrypted,
		&adapter.Status, &adapter.TimeoutMS, &adapter.RetryCount)
	if err != nil {
		return adapter, fmt.Errorf("get finance adapter secret: %w", err)
	}
	secret, err := r.box.Decrypt(encrypted)
	if err != nil {
		return adapter, fmt.Errorf("decrypt finance adapter secret: %w", err)
	}
	adapter.Secret = secret
	return adapter, nil
}

func (r *PostgresRepository) CreateExportBatch(ctx context.Context, input CreateExportBatchInput) (*ExportBatch, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin finance export batch: %w", err)
	}
	defer tx.Rollback(ctx)

	existingID, err := r.findBatchByIdempotencyKey(ctx, tx, input.IdempotencyKey)
	if err != nil {
		return nil, err
	}
	if existingID != nil {
		batch, err := r.getExportBatch(ctx, tx, *existingID)
		if err != nil {
			return nil, err
		}
		if err := tx.Commit(ctx); err != nil {
			return nil, fmt.Errorf("commit finance export idempotent read: %w", err)
		}
		return batch, nil
	}

	batch := &ExportBatch{}
	err = scanBatch(tx.QueryRow(ctx, `
		INSERT INTO finance_export_batches (
			adapter_id, period_start, period_end, status, currency,
			idempotency_key, metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, adapter_id, period_start, period_end, status, currency,
			total_amount::float8, external_batch_id, error_message, idempotency_key,
			metadata, created_at, submitted_at, updated_at
	`, input.AdapterID, input.periodStartTime, input.periodEndTime, BatchDraft, input.Currency,
		input.IdempotencyKey, mustJSON(input.Metadata)), batch)
	if err != nil {
		return nil, fmt.Errorf("create finance export batch: %w", err)
	}

	costRows, err := r.exportableCostRows(ctx, tx, input.periodStartTime, input.periodEndTime, input.Currency)
	if err != nil {
		return nil, err
	}
	total := 0.0
	lines := make([]ExportLine, 0, len(costRows))
	for _, cost := range costRows {
		line := &ExportLine{}
		err := scanLine(tx.QueryRow(ctx, `
			INSERT INTO finance_export_lines (
				batch_id, usage_ledger_id, cost_ledger_entry_id, project_cost_entry_id,
				organization_id, department_id, project_id, provider_id, model_id, amount,
				currency, metadata
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			RETURNING id, batch_id, usage_ledger_id, cost_ledger_entry_id, project_cost_entry_id, organization_id,
				department_id, project_id, provider_id, model_id, amount::float8, currency,
				external_line_id, status, metadata, created_at
		`, batch.ID, cost.UsageLedgerID, cost.CostLedgerEntryID, cost.ProjectCostEntryID, cost.OrganizationID, cost.DepartmentID,
			cost.ProjectID, cost.ProviderID, cost.ModelID, cost.Amount, cost.Currency, mustJSON(map[string]any{
				"cost_ledger_created_at": cost.CostCreatedAt.Format(time.RFC3339),
				"cost_source_type":       cost.SourceType,
			})), line)
		if err != nil {
			return nil, fmt.Errorf("create finance export line: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE cost_ledger_entries
			SET finance_export_line_id = $1
			WHERE id = $2
		`, line.ID, cost.CostLedgerEntryID); err != nil {
			return nil, fmt.Errorf("link cost ledger to finance export line: %w", err)
		}
		if cost.UsageLedgerID != nil {
			if _, err := tx.Exec(ctx, `
				UPDATE ai_usage_ledger
				SET finance_export_line_id = $1
				WHERE id = $2
			`, line.ID, *cost.UsageLedgerID); err != nil {
				return nil, fmt.Errorf("link usage ledger to finance export line: %w", err)
			}
		}
		lines = append(lines, *line)
		total += line.Amount
	}
	status := BatchDraft
	if len(lines) > 0 {
		status = BatchReady
	}
	if err := scanBatch(tx.QueryRow(ctx, `
		UPDATE finance_export_batches
		SET total_amount = $2, status = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING id, adapter_id, period_start, period_end, status, currency,
			total_amount::float8, external_batch_id, error_message, idempotency_key,
			metadata, created_at, submitted_at, updated_at
	`, batch.ID, total, status), batch); err != nil {
		return nil, fmt.Errorf("update finance export batch total: %w", err)
	}
	batch.Lines = lines
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit finance export batch: %w", err)
	}
	return batch, nil
}

func (r *PostgresRepository) ListExportBatches(ctx context.Context, limit int) ([]ExportBatch, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, adapter_id, period_start, period_end, status, currency,
			total_amount::float8, external_batch_id, error_message, idempotency_key,
			metadata, created_at, submitted_at, updated_at
		FROM finance_export_batches
		ORDER BY created_at DESC
		LIMIT $1
	`, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list finance export batches: %w", err)
	}
	defer rows.Close()
	batches := []ExportBatch{}
	for rows.Next() {
		var batch ExportBatch
		if err := scanBatch(rows, &batch); err != nil {
			return nil, fmt.Errorf("scan finance export batch: %w", err)
		}
		batches = append(batches, batch)
	}
	return batches, rows.Err()
}

func (r *PostgresRepository) GetExportBatch(ctx context.Context, id uuid.UUID) (*ExportBatch, error) {
	return r.getExportBatch(ctx, r.db, id)
}

func (r *PostgresRepository) UpdateExportBatchStatus(ctx context.Context, id uuid.UUID, input UpdateExportBatchStatusInput) (*ExportBatch, error) {
	batch := &ExportBatch{}
	err := scanBatch(r.db.QueryRow(ctx, `
		UPDATE finance_export_batches
		SET status = COALESCE(NULLIF($2, ''), status),
			external_batch_id = COALESCE(NULLIF($3, ''), external_batch_id),
			error_message = CASE
				WHEN $2 IN ('exported', 'acknowledged', 'posted', 'reconciled') THEN ''
				ELSE COALESCE(NULLIF($4, ''), error_message)
			END,
			metadata = metadata || COALESCE($5::jsonb, '{}'::jsonb),
			submitted_at = CASE WHEN $6 THEN COALESCE(submitted_at, NOW()) ELSE submitted_at END,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, adapter_id, period_start, period_end, status, currency,
			total_amount::float8, external_batch_id, error_message, idempotency_key,
			metadata, created_at, submitted_at, updated_at
	`, id, input.Status, input.ExternalBatchID, input.ErrorMessage, nullableJSON(input.Metadata), input.Submitted), batch)
	if err != nil {
		return nil, fmt.Errorf("update finance export batch status: %w", err)
	}
	lines, err := r.listBatchLines(ctx, r.db, id)
	if err != nil {
		return nil, err
	}
	batch.Lines = lines
	return batch, nil
}

func (r *PostgresRepository) RecordWebhookEvent(ctx context.Context, input RecordWebhookEventInput) (*WebhookEvent, error) {
	event := &WebhookEvent{}
	err := scanWebhookEvent(r.db.QueryRow(ctx, `
		INSERT INTO finance_webhook_events (
			adapter_id, batch_id, event_type, signature_valid, payload,
			processed, error_message
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, adapter_id, batch_id, event_type, signature_valid,
			payload, processed, error_message, created_at
	`, input.AdapterID, input.BatchID, input.EventType, input.SignatureValid, mustJSON(input.Payload),
		input.Processed, input.ErrorMessage), event)
	if err != nil {
		return nil, fmt.Errorf("record finance webhook event: %w", err)
	}
	return event, nil
}

func (r *PostgresRepository) UpdateExportLineStatus(ctx context.Context, id uuid.UUID, input UpdateExportLineStatusInput) (*ExportLine, error) {
	line := &ExportLine{}
	err := scanLine(r.db.QueryRow(ctx, `
		UPDATE finance_export_lines
		SET status = COALESCE(NULLIF($2, ''), status),
			external_line_id = COALESCE(NULLIF($3, ''), external_line_id),
			metadata = metadata || COALESCE($4::jsonb, '{}'::jsonb)
		WHERE id = $1
		RETURNING id, batch_id, usage_ledger_id, cost_ledger_entry_id, project_cost_entry_id, organization_id,
			department_id, project_id, provider_id, model_id, amount::float8, currency,
			external_line_id, status, metadata, created_at
	`, id, input.Status, input.ExternalLineID, nullableJSON(input.Metadata)), line)
	if err != nil {
		return nil, fmt.Errorf("update finance export line status: %w", err)
	}
	return line, nil
}

func (r *PostgresRepository) LinkProjectCostEntry(ctx context.Context, lineID uuid.UUID, entryID uuid.UUID) error {
	var usageID, costLedgerEntryID pgtype.UUID
	if err := r.db.QueryRow(ctx, `
		UPDATE finance_export_lines
		SET project_cost_entry_id = $2,
			metadata = metadata || '{"project_cost_posted": true}'::jsonb
		WHERE id = $1
		RETURNING usage_ledger_id, cost_ledger_entry_id
	`, lineID, entryID).Scan(&usageID, &costLedgerEntryID); err != nil {
		return fmt.Errorf("link finance line to project cost entry: %w", err)
	}
	if costLedgerEntryID.Valid {
		costLedgerUUID := uuidPtr(costLedgerEntryID)
		if costLedgerUUID != nil {
			if _, err := r.db.Exec(ctx, `
				UPDATE cost_ledger_entries
				SET metadata = metadata || jsonb_build_object('project_cost_entry_id', $2::text)
				WHERE id = $1
			`, *costLedgerUUID, entryID.String()); err != nil {
				return fmt.Errorf("link cost ledger to project cost entry: %w", err)
			}
		}
	}
	if usageID.Valid {
		usageUUID := uuidPtr(usageID)
		if usageUUID == nil {
			return nil
		}
		if _, err := r.db.Exec(ctx, `
			UPDATE ai_usage_ledger
			SET project_cost_entry_id = $2,
				posted_to_project_cost = TRUE
			WHERE id = $1
		`, *usageUUID, entryID); err != nil {
			return fmt.Errorf("link usage ledger to project cost entry: %w", err)
		}
	}
	return nil
}

func (r *PostgresRepository) ListReconciliation(ctx context.Context, limit int) ([]ReconciliationItem, error) {
	rows, err := r.db.Query(ctx, `
		WITH batches AS (
			SELECT id, adapter_id, status, currency, total_amount::float8 AS total_amount,
				CASE
					WHEN (metadata->>'external_amount') ~ '^-?[0-9]+(\.[0-9]+)?$'
						THEN (metadata->>'external_amount')::float8
					ELSE total_amount::float8
				END AS external_amount,
				external_batch_id, error_message, submitted_at, updated_at
			FROM finance_export_batches
			WHERE status IN ('exported', 'acknowledged', 'posted', 'reconciled', 'failed')
		)
		SELECT id, adapter_id, status, currency, total_amount,
			external_amount, external_amount - total_amount AS difference_amount,
			external_batch_id, error_message, submitted_at, updated_at
		FROM batches
		ORDER BY updated_at DESC
		LIMIT $1
	`, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list finance reconciliation: %w", err)
	}
	defer rows.Close()
	items := []ReconciliationItem{}
	for rows.Next() {
		var item ReconciliationItem
		if err := rows.Scan(&item.BatchID, &item.AdapterID, &item.Status, &item.Currency,
			&item.TotalAmount, &item.ExternalAmount, &item.DifferenceAmount, &item.ExternalBatchID,
			&item.ErrorMessage, &item.SubmittedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan finance reconciliation item: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

type batchStore interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (r *PostgresRepository) getExportBatch(ctx context.Context, store batchStore, id uuid.UUID) (*ExportBatch, error) {
	batch := &ExportBatch{}
	err := scanBatch(store.QueryRow(ctx, `
		SELECT id, adapter_id, period_start, period_end, status, currency,
			total_amount::float8, external_batch_id, error_message, idempotency_key,
			metadata, created_at, submitted_at, updated_at
		FROM finance_export_batches
		WHERE id = $1
	`, id), batch)
	if err != nil {
		return nil, fmt.Errorf("get finance export batch: %w", err)
	}
	lines, err := r.listBatchLines(ctx, store, id)
	if err != nil {
		return nil, err
	}
	batch.Lines = lines
	return batch, nil
}

func (r *PostgresRepository) listBatchLines(ctx context.Context, store batchStore, batchID uuid.UUID) ([]ExportLine, error) {
	rows, err := store.Query(ctx, `
		SELECT id, batch_id, usage_ledger_id, cost_ledger_entry_id, project_cost_entry_id, organization_id,
			department_id, project_id, provider_id, model_id, amount::float8, currency,
			external_line_id, status, metadata, created_at
		FROM finance_export_lines
		WHERE batch_id = $1
		ORDER BY created_at ASC
	`, batchID)
	if err != nil {
		return nil, fmt.Errorf("list finance export lines: %w", err)
	}
	defer rows.Close()
	lines := []ExportLine{}
	for rows.Next() {
		var line ExportLine
		if err := scanLine(rows, &line); err != nil {
			return nil, fmt.Errorf("scan finance export line: %w", err)
		}
		lines = append(lines, line)
	}
	return lines, rows.Err()
}

func (r *PostgresRepository) findBatchByIdempotencyKey(ctx context.Context, tx pgx.Tx, key string) (*uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `SELECT id FROM finance_export_batches WHERE idempotency_key = $1`, key).Scan(&id)
	if errorsIsNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find finance export batch by idempotency key: %w", err)
	}
	return &id, nil
}

type exportableCostRow struct {
	UsageLedgerID      *uuid.UUID
	CostLedgerEntryID  uuid.UUID
	ProjectCostEntryID *uuid.UUID
	OrganizationID     *uuid.UUID
	DepartmentID       *uuid.UUID
	ProjectID          *uuid.UUID
	ProviderID         *uuid.UUID
	ModelID            *uuid.UUID
	Amount             float64
	Currency           string
	SourceType         string
	CostCreatedAt      time.Time
}

func (r *PostgresRepository) exportableCostRows(ctx context.Context, tx pgx.Tx, start time.Time, end time.Time, currency string) ([]exportableCostRow, error) {
	rows, err := tx.Query(ctx, `
		SELECT c.id,
			u.id,
			CASE WHEN c.source_type = 'project_cost_entry' THEN c.source_id ELSE NULL END AS project_cost_entry_id,
			c.organization_id, c.department_id, c.project_id,
			i.provider_id, i.model_id, c.amount::float8, c.currency, c.source_type, c.created_at
		FROM cost_ledger_entries c
		LEFT JOIN ai_invocations i ON c.source_type = 'ai_invocation' AND i.id = c.source_id
		LEFT JOIN LATERAL (
			SELECT id
			FROM ai_usage_ledger
			WHERE invocation_id = i.id
			ORDER BY created_at ASC
			LIMIT 1
		) u ON TRUE
		WHERE c.finance_export_line_id IS NULL
			AND c.status = 'posted'
			AND c.ledger_type IN ('actual', 'adjustment')
			AND c.amount <> 0
			AND c.currency = $3
			AND c.occurred_at >= $1
			AND c.occurred_at < ($2::date + INTERVAL '1 day')
		ORDER BY c.occurred_at ASC, c.created_at ASC
		FOR UPDATE OF c SKIP LOCKED
	`, start, end, currency)
	if err != nil {
		return nil, fmt.Errorf("query exportable cost ledger: %w", err)
	}
	defer rows.Close()
	items := []exportableCostRow{}
	for rows.Next() {
		var item exportableCostRow
		var usageLedgerID, projectCostEntryID, organizationID, departmentID, projectID, providerID, modelID pgtype.UUID
		if err := rows.Scan(&item.CostLedgerEntryID, &usageLedgerID, &projectCostEntryID, &organizationID, &departmentID,
			&projectID, &providerID, &modelID, &item.Amount, &item.Currency, &item.SourceType, &item.CostCreatedAt); err != nil {
			return nil, fmt.Errorf("scan exportable cost ledger: %w", err)
		}
		item.UsageLedgerID = uuidPtr(usageLedgerID)
		item.ProjectCostEntryID = uuidPtr(projectCostEntryID)
		item.OrganizationID = uuidPtr(organizationID)
		item.DepartmentID = uuidPtr(departmentID)
		item.ProjectID = uuidPtr(projectID)
		item.ProviderID = uuidPtr(providerID)
		item.ModelID = uuidPtr(modelID)
		items = append(items, item)
	}
	return items, rows.Err()
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAdapter(row scanner, adapter *FinanceAdapter) error {
	var metadataJSON []byte
	if err := row.Scan(&adapter.ID, &adapter.Name, &adapter.EndpointURL, &adapter.AuthType,
		&adapter.MaskedSecret, &adapter.Status, &adapter.TimeoutMS, &adapter.RetryCount,
		&metadataJSON, &adapter.CreatedAt, &adapter.UpdatedAt); err != nil {
		return err
	}
	return json.Unmarshal(metadataJSON, &adapter.Metadata)
}

func scanBatch(row scanner, batch *ExportBatch) error {
	var metadataJSON []byte
	if err := row.Scan(&batch.ID, &batch.AdapterID, &batch.PeriodStart, &batch.PeriodEnd,
		&batch.Status, &batch.Currency, &batch.TotalAmount, &batch.ExternalBatchID,
		&batch.ErrorMessage, &batch.IdempotencyKey, &metadataJSON, &batch.CreatedAt,
		&batch.SubmittedAt, &batch.UpdatedAt); err != nil {
		return err
	}
	return json.Unmarshal(metadataJSON, &batch.Metadata)
}

func scanLine(row scanner, line *ExportLine) error {
	var metadataJSON []byte
	var usageLedgerID, costLedgerEntryID, projectCostEntryID, organizationID, departmentID, projectID, providerID, modelID pgtype.UUID
	if err := row.Scan(&line.ID, &line.BatchID, &usageLedgerID, &costLedgerEntryID, &projectCostEntryID, &organizationID,
		&departmentID, &projectID, &providerID, &modelID, &line.Amount, &line.Currency,
		&line.ExternalLineID, &line.Status, &metadataJSON, &line.CreatedAt); err != nil {
		return err
	}
	line.UsageLedgerID = uuidPtr(usageLedgerID)
	line.CostLedgerEntryID = uuidPtr(costLedgerEntryID)
	line.ProjectCostEntryID = uuidPtr(projectCostEntryID)
	line.OrganizationID = uuidPtr(organizationID)
	line.DepartmentID = uuidPtr(departmentID)
	line.ProjectID = uuidPtr(projectID)
	line.ProviderID = uuidPtr(providerID)
	line.ModelID = uuidPtr(modelID)
	return json.Unmarshal(metadataJSON, &line.Metadata)
}

func scanWebhookEvent(row scanner, event *WebhookEvent) error {
	var payloadJSON []byte
	var batchID pgtype.UUID
	if err := row.Scan(&event.ID, &event.AdapterID, &batchID, &event.EventType,
		&event.SignatureValid, &payloadJSON, &event.Processed, &event.ErrorMessage,
		&event.CreatedAt); err != nil {
		return err
	}
	event.BatchID = uuidPtr(batchID)
	return json.Unmarshal(payloadJSON, &event.Payload)
}

func uuidPtr(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	id := uuidPtrValue(value)
	if id == uuid.Nil {
		return nil
	}
	return &id
}

func uuidPtrValue(value pgtype.UUID) uuid.UUID {
	if !value.Valid {
		return uuid.Nil
	}
	id, err := uuid.FromBytes(value.Bytes[:])
	if err != nil {
		return uuid.Nil
	}
	return id
}

func mustJSON(value map[string]any) []byte {
	if value == nil {
		value = map[string]any{}
	}
	data, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return data
}

func nullableJSON(value map[string]any) any {
	if value == nil {
		return nil
	}
	return mustJSON(value)
}

func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return "****"
	}
	return secret[:4] + "****" + secret[len(secret)-4:]
}

func errorsIsNoRows(err error) bool {
	return err == pgx.ErrNoRows
}
