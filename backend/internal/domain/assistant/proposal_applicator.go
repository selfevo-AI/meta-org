package assistant

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DBProposalApplicator struct {
	db *pgxpool.Pool
}

func NewDBProposalApplicator(db *pgxpool.Pool) *DBProposalApplicator {
	return &DBProposalApplicator{db: db}
}

func (a *DBProposalApplicator) ApplyProposal(ctx context.Context, proposal *Proposal) (map[string]any, error) {
	if a == nil || a.db == nil {
		return nil, fmt.Errorf("%w: proposal applicator database is not configured", ErrValidation)
	}
	if proposal == nil || proposal.TargetID == nil {
		return nil, fmt.Errorf("%w: proposal target is required", ErrValidation)
	}
	table, err := proposalTargetTable(proposal.ModuleKey, proposal.TargetType)
	if err != nil {
		return nil, err
	}
	entry := map[string]any{
		"proposal_id":   proposal.ID.String(),
		"session_id":    proposal.SessionID.String(),
		"proposal_type": proposal.ProposalType,
		"title":         proposal.Title,
		"summary":       proposal.Summary,
		"payload":       proposal.Payload,
	}
	command, err := a.db.Exec(ctx, fmt.Sprintf(`
		UPDATE %s
		SET metadata = jsonb_set(
			COALESCE(metadata, '{}'::jsonb),
			'{assistant_confirmed_proposals}',
			COALESCE(metadata->'assistant_confirmed_proposals', '[]'::jsonb) || jsonb_build_array($2::jsonb),
			true
		)
		WHERE id = $1
	`, table), *proposal.TargetID, mustJSON(entry))
	if err != nil {
		return nil, fmt.Errorf("apply assistant proposal: %w", err)
	}
	if command.RowsAffected() == 0 {
		return nil, fmt.Errorf("%w: proposal target not found", ErrNotFound)
	}
	return map[string]any{
		"target_table": table,
		"target_id":    proposal.TargetID.String(),
		"writeback":    "metadata.assistant_confirmed_proposals",
	}, nil
}

func proposalTargetTable(moduleKey string, targetType string) (string, error) {
	key := targetType
	if key == "" {
		key = moduleKey
	}
	switch key {
	case "requirement":
		return "requirements", nil
	case "project":
		return "projects", nil
	case "deliverable", "delivery":
		return "deliverables", nil
	case "project_cost", "cost":
		return "project_cost_entries", nil
	case "project_evaluation", "feedback":
		return "project_evaluations", nil
	case "workflow", "workflow_instance":
		return "workflow_instances", nil
	case "task":
		return "tasks", nil
	case "finance_batch", "finance", "finance_accounting":
		return "finance_export_batches", nil
	case "finance_settlement", "settlement", "settlement_order":
		return "finance_settlement_orders", nil
	case "finance_receivable", "receivable":
		return "finance_receivables", nil
	case "finance_payable", "payable":
		return "finance_payables", nil
	case "cost_budget", "budget":
		return "cost_budgets", nil
	case "cost_rate_card", "rate_card":
		return "cost_rate_cards", nil
	case "cost_ledger_entry", "ledger_entry":
		return "cost_ledger_entries", nil
	default:
		return "", fmt.Errorf("%w: unsupported proposal target type %q", ErrValidation, key)
	}
}
