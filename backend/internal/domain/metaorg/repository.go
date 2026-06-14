package metaorg

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Overview(ctx context.Context) (Overview, error) {
	health, err := r.health(ctx)
	if err != nil {
		return Overview{}, err
	}
	projects, err := r.projectSummary(ctx)
	if err != nil {
		return Overview{}, err
	}
	agents, err := r.agentSummary(ctx)
	if err != nil {
		return Overview{}, err
	}
	cost, err := r.costSummary(ctx)
	if err != nil {
		return Overview{}, err
	}
	risks, err := r.risks(ctx, 10)
	if err != nil {
		return Overview{}, err
	}
	activity, err := r.activity(ctx, 10)
	if err != nil {
		return Overview{}, err
	}
	return Overview{
		GeneratedAt: time.Now().UTC(),
		Health:      health,
		Projects:    projects,
		Agents:      agents,
		Cost:        cost,
		Risks:       risks,
		Activity:    activity,
	}, nil
}

func (r *PostgresRepository) Inbox(ctx context.Context, filter InboxFilter) ([]InboxItem, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}

	items := make([]InboxItem, 0, filter.Limit)
	if r.tableExists(ctx, "tool_approvals") {
		toolItems, err := r.toolApprovalInbox(ctx, filter.Limit)
		if err != nil {
			return nil, err
		}
		items = append(items, toolItems...)
	}
	signalItems, err := r.signalInbox(ctx, filter.Limit)
	if err != nil {
		return nil, err
	}
	items = append(items, signalItems...)

	reviewItems, err := r.reviewInbox(ctx, filter.Limit)
	if err != nil {
		return nil, err
	}
	items = append(items, reviewItems...)

	decisionItems, err := r.accessDecisionInbox(ctx, filter.Limit)
	if err != nil {
		return nil, err
	}
	items = append(items, decisionItems...)

	if r.tableExists(ctx, "finance_export_batches") {
		financeItems, err := r.financeInbox(ctx, filter.Limit)
		if err != nil {
			return nil, err
		}
		items = append(items, financeItems...)
	}

	items = filterInboxItems(items, filter.Type)
	sortInboxItems(items)
	if len(items) > filter.Limit {
		items = items[:filter.Limit]
	}
	return items, nil
}

func (r *PostgresRepository) health(ctx context.Context) (HealthSummary, error) {
	var summary HealthSummary
	if err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM requirements WHERE status IN ('draft', 'analyzed', 'approved')),
			(SELECT COUNT(*) FROM projects WHERE status IN ('planning', 'active', 'paused', 'delivering')),
			(SELECT COUNT(*) FROM ai_agents WHERE is_active)
	`).Scan(&summary.OpenRequirements, &summary.ActiveProjects, &summary.ActiveAgents); err != nil {
		return summary, fmt.Errorf("query meta-org health: %w", err)
	}
	summary.Currency = "CNY"
	summary.PendingApprovals = 0
	if r.tableExists(ctx, "tool_approvals") {
		if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM tool_approvals WHERE status = 'pending'`).Scan(&summary.PendingApprovals); err != nil {
			return summary, fmt.Errorf("query pending tool approvals: %w", err)
		}
	}
	if r.tableExists(ctx, "ai_usage_ledger") {
		if err := r.db.QueryRow(ctx, `
			SELECT COALESCE(SUM(amount), 0)::float8
			FROM ai_usage_ledger
			WHERE finance_export_line_id IS NULL
		`).Scan(&summary.UnexportedCost); err != nil {
			return summary, fmt.Errorf("query unexported ai cost: %w", err)
		}
	}
	return summary, nil
}

func (r *PostgresRepository) projectSummary(ctx context.Context) (ProjectSummary, error) {
	counts, err := r.countBy(ctx, `SELECT status::text, COUNT(*) FROM projects GROUP BY status`)
	if err != nil {
		return ProjectSummary{}, fmt.Errorf("query project status counts: %w", err)
	}
	var overBudget int64
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM projects p
		WHERE p.budget_amount > 0
		  AND COALESCE((SELECT SUM(amount) FROM project_cost_entries c WHERE c.project_id = p.id), 0) > p.budget_amount
	`).Scan(&overBudget); err != nil {
		return ProjectSummary{}, fmt.Errorf("query over-budget projects: %w", err)
	}
	return ProjectSummary{
		ByStatus:   withKnownKeys(counts, "planning", "active", "paused", "delivering", "completed", "closed", "cancelled"),
		OverBudget: overBudget,
	}, nil
}

func (r *PostgresRepository) agentSummary(ctx context.Context) (AgentSummary, error) {
	var summary AgentSummary
	if err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM ai_agents),
			(SELECT COUNT(*) FROM ai_agents WHERE is_active)
	`).Scan(&summary.Total, &summary.Active); err != nil {
		return summary, fmt.Errorf("query agent summary: %w", err)
	}
	counts, err := r.countBy(ctx, `SELECT risk_level::text, COUNT(*) FROM ai_agents GROUP BY risk_level`)
	if err != nil {
		return summary, fmt.Errorf("query agent risk counts: %w", err)
	}
	summary.ByRiskLevel = withKnownKeys(counts, "low", "medium", "high", "critical")
	return summary, nil
}

func (r *PostgresRepository) costSummary(ctx context.Context) (CostSummary, error) {
	summary := CostSummary{
		Currency:   "CNY",
		ByProvider: map[string]float64{},
	}
	if err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(amount) FILTER (WHERE occurred_at >= date_trunc('day', NOW())), 0)::float8,
			COALESCE(SUM(amount) FILTER (WHERE occurred_at >= date_trunc('month', NOW())), 0)::float8
		FROM project_cost_entries
	`).Scan(&summary.Today, &summary.MonthToDate); err != nil {
		return summary, fmt.Errorf("query project cost summary: %w", err)
	}
	if r.tableExists(ctx, "ai_usage_ledger") {
		if err := r.db.QueryRow(ctx, `
			SELECT COALESCE(SUM(amount), 0)::float8
			FROM ai_usage_ledger
			WHERE finance_export_line_id IS NULL
		`).Scan(&summary.Unexported); err != nil {
			return summary, fmt.Errorf("query unexported usage summary: %w", err)
		}
	}
	return summary, nil
}

func (r *PostgresRepository) risks(ctx context.Context, limit int) ([]RiskItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, title, severity, source
		FROM (
			SELECT id::text, action || ' ' || resource AS title, risk_level AS severity, 'governance' AS source, created_at
			FROM access_decisions
			WHERE risk_level IN ('high', 'critical') OR NOT allowed
			UNION ALL
			SELECT id::text, signal_type AS title, CASE WHEN priority >= 9 THEN 'critical' ELSE 'high' END AS severity, 'evolution' AS source, created_at
			FROM signals
			WHERE NOT acknowledged AND priority >= 7
		) risks
		ORDER BY
			CASE severity WHEN 'critical' THEN 0 WHEN 'high' THEN 1 WHEN 'medium' THEN 2 ELSE 3 END,
			created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query risks: %w", err)
	}
	defer rows.Close()

	risks := []RiskItem{}
	for rows.Next() {
		var item RiskItem
		if err := rows.Scan(&item.ID, &item.Title, &item.Severity, &item.Source); err != nil {
			return nil, fmt.Errorf("scan risk: %w", err)
		}
		risks = append(risks, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate risks: %w", err)
	}
	return risks, nil
}

func (r *PostgresRepository) activity(ctx context.Context, limit int) ([]ActivityItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, type, title, status, created_at
		FROM (
			SELECT id::text, 'requirement' AS type, title, status::text, created_at FROM requirements
			UNION ALL
			SELECT id::text, 'project' AS type, name AS title, status::text, created_at FROM projects
			UNION ALL
			SELECT id::text, 'workflow' AS type, 'Workflow instance' AS title, status::text, created_at FROM workflow_instances
			UNION ALL
			SELECT id::text, 'signal' AS type, signal_type AS title, CASE WHEN acknowledged THEN 'acknowledged' ELSE 'open' END AS status, created_at FROM signals
		) events
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query activity: %w", err)
	}
	defer rows.Close()

	activity := []ActivityItem{}
	for rows.Next() {
		var item ActivityItem
		if err := rows.Scan(&item.ID, &item.Type, &item.Title, &item.Status, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan activity: %w", err)
		}
		activity = append(activity, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate activity: %w", err)
	}
	return activity, nil
}

func (r *PostgresRepository) signalInbox(ctx context.Context, limit int) ([]InboxItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, 'evolution_signal', signal_type, 'pending', CASE WHEN priority >= 9 THEN 'critical' WHEN priority >= 7 THEN 'high' ELSE 'medium' END, 'evolution', created_at
		FROM signals
		WHERE NOT acknowledged AND priority >= 7
		ORDER BY priority DESC, created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query signal inbox: %w", err)
	}
	return scanInboxRows(rows)
}

func (r *PostgresRepository) reviewInbox(ctx context.Context, limit int) ([]InboxItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, 'verification_review', 'Verification review', status, 'medium', 'verification', created_at
		FROM review_assignments
		WHERE status = 'pending'
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query review inbox: %w", err)
	}
	return scanInboxRows(rows)
}

func (r *PostgresRepository) accessDecisionInbox(ctx context.Context, limit int) ([]InboxItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, 'governance_decision', action || ' ' || resource, decision, CASE WHEN risk_level = 'critical' THEN 'critical' ELSE 'high' END, 'governance', created_at
		FROM access_decisions
		WHERE risk_level IN ('high', 'critical') OR decision IN ('approve', 'deny')
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query access decision inbox: %w", err)
	}
	return scanInboxRows(rows)
}

func (r *PostgresRepository) toolApprovalInbox(ctx context.Context, limit int) ([]InboxItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, 'tool_approval', 'Tool approval required', status, 'high', 'toolruntime', created_at
		FROM tool_approvals
		WHERE status = 'pending'
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query tool approval inbox: %w", err)
	}
	return scanInboxRows(rows)
}

func (r *PostgresRepository) financeInbox(ctx context.Context, limit int) ([]InboxItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id::text, 'finance_export', 'Finance export failed', status, 'high', 'finance', updated_at
		FROM finance_export_batches
		WHERE status = 'failed'
		ORDER BY updated_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query finance inbox: %w", err)
	}
	return scanInboxRows(rows)
}

func (r *PostgresRepository) tableExists(ctx context.Context, tableName string) bool {
	var exists bool
	if err := r.db.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, tableName).Scan(&exists); err != nil {
		return false
	}
	return exists
}

func (r *PostgresRepository) countBy(ctx context.Context, query string) (map[string]int64, error) {
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int64{}
	for rows.Next() {
		var key string
		var count int64
		if err := rows.Scan(&key, &count); err != nil {
			return nil, err
		}
		counts[key] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return counts, nil
}

func withKnownKeys(counts map[string]int64, keys ...string) map[string]int64 {
	result := make(map[string]int64, len(keys)+len(counts))
	for _, key := range keys {
		result[key] = counts[key]
	}
	for key, count := range counts {
		result[key] = count
	}
	return result
}

func scanInboxRows(rows pgx.Rows) ([]InboxItem, error) {
	defer rows.Close()

	items := []InboxItem{}
	for rows.Next() {
		var item InboxItem
		if err := rows.Scan(&item.ID, &item.Type, &item.Title, &item.Status, &item.Priority, &item.Source, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan inbox item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inbox items: %w", err)
	}
	return items, nil
}

func filterInboxItems(items []InboxItem, itemType string) []InboxItem {
	if itemType == "" {
		return items
	}
	filtered := make([]InboxItem, 0, len(items))
	for _, item := range items {
		if item.Type == itemType {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func sortInboxItems(items []InboxItem) {
	sort.SliceStable(items, func(i, j int) bool {
		leftRank := inboxPriorityRank(items[i].Priority)
		rightRank := inboxPriorityRank(items[j].Priority)
		if leftRank != rightRank {
			return leftRank < rightRank
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
}

func inboxPriorityRank(priority string) int {
	switch priority {
	case "critical":
		return 0
	case "high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 4
	}
}
