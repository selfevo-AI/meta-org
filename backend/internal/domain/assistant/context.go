package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkRecord struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Status    string         `json:"status"`
	CreatedAt string         `json:"created_at"`
	Data      map[string]any `json:"data,omitempty"`
}

type WorkRecordContext struct {
	ModuleKey string
	Records   []WorkRecord
	Error     string
}

type ContextResolver interface {
	Resolve(context.Context, *Session) WorkRecordContext
}

type DBContextResolver struct {
	db *pgxpool.Pool
}

func NewDBContextResolver(db *pgxpool.Pool) *DBContextResolver {
	return &DBContextResolver{db: db}
}

func (r *DBContextResolver) Resolve(ctx context.Context, session *Session) WorkRecordContext {
	result := WorkRecordContext{ModuleKey: session.ModuleKey}
	if r == nil || r.db == nil || session == nil {
		return result
	}
	records, err := r.queryRecords(ctx, session.ModuleKey, session.TargetType)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	if session.TargetID != nil {
		if selected, err := r.queryTargetRecord(ctx, session.ModuleKey, session.TargetType, *session.TargetID); err == nil {
			records = mergeSelectedRecord(selected, records)
		} else {
			result.Error = err.Error()
			return result
		}
	}
	result.Records = records
	return result
}

func (r *DBContextResolver) queryRecords(ctx context.Context, moduleKey string, targetType string) ([]WorkRecord, error) {
	query := targetContextQuery(targetType)
	if query == "" {
		query = contextQuery(moduleKey)
	}
	if query == "" {
		query = contextQuery("meta_org")
	}
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("resolve assistant context: %w", err)
	}
	defer rows.Close()

	records := []WorkRecord{}
	for rows.Next() {
		var record WorkRecord
		if err := rows.Scan(&record.ID, &record.Type, &record.Title, &record.Status, &record.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan assistant context: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read assistant context: %w", err)
	}
	return records, nil
}

func (r *DBContextResolver) queryTargetRecord(ctx context.Context, moduleKey string, targetType string, targetID uuid.UUID) (WorkRecord, error) {
	table, err := proposalTargetTable(moduleKey, targetType)
	if err != nil {
		return WorkRecord{}, err
	}
	var record WorkRecord
	var dataJSON []byte
	err = r.db.QueryRow(ctx, fmt.Sprintf(`
		WITH target AS (
			SELECT to_jsonb(t) AS data
			FROM %s t
			WHERE t.id = $1
			LIMIT 1
		)
		SELECT
			data->>'id',
			$2::text,
			COALESCE(
				NULLIF(data->>'title', ''),
				NULLIF(data->>'name', ''),
				NULLIF(data->>'settlement_number', ''),
				NULLIF(data->>'invoice_number', ''),
				NULLIF(data->>'description', ''),
				data->>'id',
				''
			),
			COALESCE(NULLIF(data->>'status', ''), ''),
			COALESCE(NULLIF(data->>'created_at', ''), ''),
			data
		FROM target
	`, table), targetID, firstNonEmpty(targetType, moduleKey)).Scan(
		&record.ID, &record.Type, &record.Title, &record.Status, &record.CreatedAt, &dataJSON,
	)
	if err != nil {
		return WorkRecord{}, fmt.Errorf("resolve assistant target context: %w", err)
	}
	record.Data = unmarshalRecordData(dataJSON)
	return record, nil
}

func mergeSelectedRecord(selected WorkRecord, records []WorkRecord) []WorkRecord {
	merged := []WorkRecord{selected}
	for _, record := range records {
		if record.ID == selected.ID && record.Type == selected.Type {
			continue
		}
		merged = append(merged, record)
	}
	return merged
}

func unmarshalRecordData(data []byte) map[string]any {
	if len(data) == 0 {
		return nil
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil
	}
	return value
}

func targetContextQuery(targetType string) string {
	switch strings.ToLower(strings.TrimSpace(targetType)) {
	case "requirement":
		return `SELECT id::text, 'requirement', title, status, created_at::text FROM requirements ORDER BY created_at DESC LIMIT 12`
	case "project":
		return `SELECT id::text, 'project', name, status, created_at::text FROM projects ORDER BY created_at DESC LIMIT 12`
	case "deliverable", "delivery":
		return `SELECT id::text, 'deliverable', name, status, created_at::text FROM deliverables ORDER BY created_at DESC LIMIT 12`
	case "project_cost", "cost":
		return `SELECT id::text, 'project_cost', COALESCE(NULLIF(description, ''), source_type), 'posted', created_at::text FROM project_cost_entries ORDER BY created_at DESC LIMIT 12`
	case "project_evaluation", "feedback":
		return `SELECT id::text, 'project_evaluation', evaluator_type, 'completed', created_at::text FROM project_evaluations ORDER BY created_at DESC LIMIT 12`
	case "workflow", "workflow_instance":
		return `SELECT id::text, 'workflow_instance', status, status, created_at::text FROM workflow_instances ORDER BY created_at DESC LIMIT 12`
	case "task":
		return `SELECT id::text, 'task', stage_type, status, created_at::text FROM tasks ORDER BY created_at DESC LIMIT 12`
	case "finance_settlement", "settlement", "settlement_order", "finance_accounting":
		return `SELECT id::text, 'finance_settlement', COALESCE(NULLIF(title, ''), settlement_number), status, created_at::text FROM finance_settlement_orders ORDER BY created_at DESC LIMIT 12`
	case "finance_receivable", "receivable":
		return `SELECT id::text, 'finance_receivable', COALESCE(NULLIF(invoice_number, ''), customer_name), status, created_at::text FROM finance_receivables ORDER BY created_at DESC LIMIT 12`
	case "finance_payable", "payable":
		return `SELECT id::text, 'finance_payable', COALESCE(NULLIF(invoice_number, ''), vendor_name), status, created_at::text FROM finance_payables ORDER BY created_at DESC LIMIT 12`
	case "cost_budget", "budget":
		return `SELECT id::text, 'cost_budget', scope_type, status, created_at::text FROM cost_budgets ORDER BY created_at DESC LIMIT 12`
	case "cost_rate_card", "rate_card":
		return `SELECT id::text, 'cost_rate_card', subject_type || ':' || rate_type, status, created_at::text FROM cost_rate_cards ORDER BY created_at DESC LIMIT 12`
	case "cost_ledger_entry", "ledger_entry":
		return `SELECT id::text, 'cost_ledger_entry', cost_category || ':' || source_type, status, created_at::text FROM cost_ledger_entries ORDER BY created_at DESC LIMIT 12`
	default:
		return ""
	}
}

func contextQuery(moduleKey string) string {
	switch strings.ToLower(moduleKey) {
	case "requirement":
		return `
			SELECT id::text, 'requirement', title, status, created_at::text FROM requirements ORDER BY created_at DESC LIMIT 12`
	case "project", "delivery", "feedback":
		return `
			SELECT id::text, 'project', name, status, created_at::text FROM projects ORDER BY created_at DESC LIMIT 12`
	case "project_cost", "cost":
		return `
			SELECT id::text, 'project_cost', COALESCE(NULLIF(description, ''), source_type), 'posted', created_at::text FROM project_cost_entries ORDER BY created_at DESC LIMIT 12`
	case "meta_resource":
		return `
			(SELECT id::text, 'meta_resource', name, status, created_at::text FROM meta_resources ORDER BY created_at DESC LIMIT 8)
			UNION ALL
			(SELECT id::text, 'pdca_cycle', COALESCE(NULLIF(summary, ''), current_stage), status, created_at::text FROM pdca_cycles ORDER BY created_at DESC LIMIT 4)
			LIMIT 12`
	case "organization":
		return `
			(SELECT id::text, 'department', name, status, created_at::text FROM departments ORDER BY created_at DESC LIMIT 6)
			UNION ALL
			(SELECT id::text, 'position', name, status, created_at::text FROM positions ORDER BY created_at DESC LIMIT 6)
			LIMIT 12`
	case "workflow":
		return `
			(SELECT id::text, 'workflow_instance', status, status, created_at::text FROM workflow_instances ORDER BY created_at DESC LIMIT 6)
			UNION ALL
			(SELECT id::text, 'task', stage_type, status, created_at::text FROM tasks ORDER BY created_at DESC LIMIT 6)
			LIMIT 12`
	case "capability":
		return `
			SELECT id::text, 'capability', name, CASE WHEN is_active THEN 'active' ELSE 'inactive' END, created_at::text FROM capabilities ORDER BY created_at DESC LIMIT 12`
	case "governance":
		return `
			(SELECT id::text, 'principle', name, CASE WHEN is_active THEN 'active' ELSE 'inactive' END, created_at::text FROM principles ORDER BY created_at DESC LIMIT 6)
			UNION ALL
			(SELECT id::text, 'access_decision', action, decision, created_at::text FROM access_decisions ORDER BY created_at DESC LIMIT 6)
			LIMIT 12`
	case "self_evolution":
		return `
			(SELECT id::text, 'signal', signal_type, CASE WHEN acknowledged THEN 'acknowledged' ELSE 'open' END, created_at::text FROM signals ORDER BY created_at DESC LIMIT 6)
			UNION ALL
			(SELECT id::text, 'knowledge', title, source, created_at::text FROM knowledge_entries ORDER BY created_at DESC LIMIT 6)
			LIMIT 12`
	case "verification":
		return `
			SELECT id::text, 'verification_report', conclusion, 'completed', created_at::text FROM verification_reports ORDER BY created_at DESC LIMIT 12`
	case "model_settings":
		return `
			(SELECT id::text, 'model', model_key, status, created_at::text FROM models ORDER BY created_at DESC LIMIT 6)
			UNION ALL
			(SELECT id::text, 'ai_invocation', requested_model, status, created_at::text FROM ai_invocations ORDER BY created_at DESC LIMIT 6)
			LIMIT 12`
	case "costing":
		return `
			(SELECT id::text, 'rate_card', subject_type || ':' || rate_type, status, created_at::text FROM cost_rate_cards ORDER BY created_at DESC LIMIT 6)
			UNION ALL
			(SELECT id::text, 'budget', scope_type, status, created_at::text FROM cost_budgets ORDER BY created_at DESC LIMIT 6)
			UNION ALL
			(SELECT id::text, 'cost_ledger_entry', cost_category || ':' || source_type, status, created_at::text FROM cost_ledger_entries ORDER BY created_at DESC LIMIT 6)
			LIMIT 12`
	case "finance":
		return `
			(SELECT id::text, 'finance_settlement', COALESCE(NULLIF(title, ''), settlement_number), status, created_at::text FROM finance_settlement_orders ORDER BY created_at DESC LIMIT 4)
			UNION ALL
			(SELECT id::text, 'finance_receivable', COALESCE(NULLIF(invoice_number, ''), customer_name), status, created_at::text FROM finance_receivables ORDER BY created_at DESC LIMIT 4)
			UNION ALL
			(SELECT id::text, 'finance_payable', COALESCE(NULLIF(invoice_number, ''), vendor_name), status, created_at::text FROM finance_payables ORDER BY created_at DESC LIMIT 4)
			LIMIT 12`
	case "meta_org":
		return `
			(SELECT id::text, 'requirement', title, status, created_at::text FROM requirements ORDER BY created_at DESC LIMIT 4)
			UNION ALL
			(SELECT id::text, 'project', name, status, created_at::text FROM projects ORDER BY created_at DESC LIMIT 4)
			UNION ALL
			(SELECT id::text, 'ai_invocation', requested_model, status, created_at::text FROM ai_invocations ORDER BY created_at DESC LIMIT 4)
			LIMIT 12`
	default:
		return ""
	}
}
