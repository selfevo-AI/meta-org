package assistant

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresContextRepository struct {
	db *pgxpool.Pool
}

func NewContextRepository(db *pgxpool.Pool) *PostgresContextRepository {
	return &PostgresContextRepository{db: db}
}

func (r *PostgresContextRepository) CreateDictionaryVersion(ctx context.Context, model DictionaryImportModel, importedBy *uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO context_dictionary_versions (scope_level, organization_id, module_key, version_key, source_type, source_name, imported_by, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`, model.ScopeLevel, model.OrganizationID, model.ModuleKey, model.VersionKey, model.SourceType, model.SourceName, importedBy, mustJSON(map[string]any{"field_count": len(model.Fields)})).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create context dictionary version: %w", err)
	}
	return id, nil
}

func (r *PostgresContextRepository) CreateContextChangeProposal(ctx context.Context, input ContextChangeProposalInput) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO context_change_proposals (dictionary_version_id, proposal_type, title, summary, payload, status)
		VALUES ($1, $2, $3, $4, $5, COALESCE(NULLIF($6, ''), 'pending'))
		RETURNING id
	`, input.DictionaryVersionID, input.ProposalType, input.Title, input.Summary, mustJSON(input.Payload), input.Status).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create context change proposal: %w", err)
	}
	return id, nil
}

func (r *PostgresContextRepository) CreateContextMigrationDraft(ctx context.Context, input ContextMigrationDraftInput) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO context_migration_drafts (dictionary_version_id, title, summary, sql_up, sql_down, risk_level, metadata)
		VALUES ($1, $2, $3, $4, $5, COALESCE(NULLIF($6, ''), 'medium'), $7)
		RETURNING id
	`, input.DictionaryVersionID, input.Title, input.Summary, input.SQLUp, input.SQLDown, input.RiskLevel, mustJSON(input.Metadata)).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create context migration draft: %w", err)
	}
	return id, nil
}

func (r *PostgresContextRepository) CreateContextPackage(ctx context.Context, request ContextRequest, pkg ContextPackage) (*ContextPackage, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx, `
		INSERT INTO context_packages (
			session_id, dictionary_version_id, actor_id, actor_type, organization_id, module_key,
			target_type, target_id, workflow_id, task_id, attention_core, supporting_context,
			risk_and_signals, omissions, weights, validations, provenance, token_budget
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id
	`, request.SessionID, pkg.DictionaryVersionID, request.ActorID, request.ActorType, request.OrganizationID, request.ModuleKey,
		request.TargetType, request.TargetID, request.WorkflowID, request.TaskID, mustJSONValue(pkg.AttentionCore), mustJSONValue(pkg.SupportingContext),
		mustJSONValue(pkg.RiskAndSignals), mustJSONValue(pkg.Omissions), mustJSONValue(pkg.Weights), mustJSONValue(pkg.Validations),
		mustJSONValue(pkg.Provenance), pkg.TokenBudget).Scan(&id)
	if err != nil {
		return nil, fmt.Errorf("create context package: %w", err)
	}
	pkg.ID = id
	return &pkg, nil
}
