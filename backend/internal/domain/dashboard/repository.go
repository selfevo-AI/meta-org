package dashboard

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Identity(ctx context.Context) (IdentitySummary, error) {
	var s IdentitySummary
	if err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM users),
			(SELECT COUNT(*) FROM ai_agents WHERE is_active),
			(SELECT COUNT(*) FROM ai_agents),
			(SELECT COUNT(*) FROM roles)
	`).Scan(&s.Users, &s.ActiveAgents, &s.TotalAgents, &s.Roles); err != nil {
		return s, fmt.Errorf("query identity summary: %w", err)
	}
	return s, nil
}

func (r *Repository) Organization(ctx context.Context) (OrganizationSummary, error) {
	var s OrganizationSummary
	if err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM organizations),
			(SELECT COUNT(*) FROM muvrs),
			(SELECT COUNT(*) FROM mvru_members),
			(SELECT COUNT(*) FROM mvru_relationships)
	`).Scan(&s.Organizations, &s.MVRUs, &s.Members, &s.Relationships); err != nil {
		return s, fmt.Errorf("query organization summary: %w", err)
	}

	counts, err := r.countBy(ctx, `SELECT status::text, COUNT(*) FROM muvrs GROUP BY status`)
	if err != nil {
		return s, fmt.Errorf("query mvru status counts: %w", err)
	}
	s.MVRUsByStatus = withKnownKeys(counts, "designing", "active", "evaluating", "evolving", "dissolved")
	return s, nil
}

func (r *Repository) Workflow(ctx context.Context) (WorkflowSummary, error) {
	var s WorkflowSummary
	if err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM workflow_templates),
			(SELECT COUNT(*) FROM workflow_templates WHERE is_active),
			(SELECT COUNT(*) FROM workflow_instances),
			(SELECT COUNT(*) FROM decisions WHERE created_at >= NOW() - INTERVAL '7 days')
	`).Scan(&s.Templates, &s.ActiveTemplates, &s.Instances, &s.Decisions7d); err != nil {
		return s, fmt.Errorf("query workflow summary: %w", err)
	}

	instanceCounts, err := r.countBy(ctx, `SELECT status::text, COUNT(*) FROM workflow_instances GROUP BY status`)
	if err != nil {
		return s, fmt.Errorf("query workflow status counts: %w", err)
	}
	s.InstancesByStatus = withKnownKeys(instanceCounts, "active", "paused", "completed", "failed")

	taskCounts, err := r.countBy(ctx, `SELECT status::text, COUNT(*) FROM tasks GROUP BY status`)
	if err != nil {
		return s, fmt.Errorf("query task status counts: %w", err)
	}
	s.TasksByStatus = withKnownKeys(taskCounts, "pending", "assigned", "in_progress", "completed", "rejected")

	return s, nil
}

func (r *Repository) Capability(ctx context.Context) (CapabilitySummary, error) {
	var s CapabilitySummary
	if err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM capabilities),
			(SELECT COUNT(*) FROM capabilities WHERE is_active),
			(SELECT COUNT(*) FROM capability_bindings),
			(SELECT COUNT(*) FROM capability_invocations WHERE created_at >= NOW() - INTERVAL '24 hours'),
			(SELECT COUNT(*) FROM capability_invocations WHERE created_at >= NOW() - INTERVAL '24 hours' AND outcome IN ('failed', 'error', 'rejected')),
			(SELECT COALESCE(AVG(duration_ms), 0)::float8 FROM capability_invocations WHERE created_at >= NOW() - INTERVAL '24 hours'),
			(SELECT COALESCE(SUM(cost), 0)::float8 FROM capability_invocations WHERE created_at >= NOW() - INTERVAL '24 hours')
	`).Scan(
		&s.Capabilities,
		&s.ActiveCapabilities,
		&s.Bindings,
		&s.Invocations24h,
		&s.FailedInvocations24h,
		&s.AverageDurationMs,
		&s.Cost24h,
	); err != nil {
		return s, fmt.Errorf("query capability summary: %w", err)
	}
	return s, nil
}

func (r *Repository) Observability(ctx context.Context) (ObservabilitySummary, error) {
	var s ObservabilitySummary
	if err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM traces WHERE status = 'active'),
			(SELECT COUNT(*) FROM traces WHERE status = 'completed'),
			(SELECT COUNT(*) FROM traces WHERE status = 'failed'),
			(SELECT COUNT(*) FROM spans WHERE started_at >= NOW() - INTERVAL '24 hours'),
			(SELECT COUNT(*) FROM metrics WHERE recorded_at >= NOW() - INTERVAL '24 hours')
	`).Scan(&s.ActiveTraces, &s.CompletedTraces, &s.FailedTraces, &s.Spans24h, &s.Metrics24h); err != nil {
		return s, fmt.Errorf("query observability summary: %w", err)
	}
	return s, nil
}

func (r *Repository) Verification(ctx context.Context) (VerificationSummary, error) {
	var s VerificationSummary
	if err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM verification_reports),
			(SELECT COALESCE(AVG(overall_score), 0)::float8 FROM verification_reports),
			(SELECT COUNT(*) FROM review_assignments WHERE status = 'pending')
	`).Scan(&s.Reports, &s.AverageScore, &s.PendingReviews); err != nil {
		return s, fmt.Errorf("query verification summary: %w", err)
	}
	return s, nil
}

func (r *Repository) Governance(ctx context.Context) (GovernanceSummary, error) {
	var s GovernanceSummary
	if err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM permissions),
			(SELECT COUNT(*) FROM principles WHERE is_active),
			(SELECT COUNT(*) FROM control_rules),
			(SELECT COUNT(*) FROM control_rules WHERE is_active)
	`).Scan(&s.Permissions, &s.ActivePrinciples, &s.ControlRules, &s.ActiveControlRules); err != nil {
		return s, fmt.Errorf("query governance summary: %w", err)
	}
	return s, nil
}

func (r *Repository) Evolution(ctx context.Context) (EvolutionSummary, error) {
	var s EvolutionSummary
	if err := r.db.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*) FROM weight_scores),
			(SELECT COUNT(*) FROM knowledge_entries),
			(SELECT COUNT(*) FROM signals WHERE NOT acknowledged),
			(SELECT COUNT(*) FROM signals WHERE NOT acknowledged AND priority >= 7)
	`).Scan(&s.WeightedActors, &s.KnowledgeEntries, &s.UnacknowledgedSignals, &s.HighPrioritySignals); err != nil {
		return s, fmt.Errorf("query evolution summary: %w", err)
	}

	counts, err := r.countBy(ctx, `SELECT status, COUNT(*) FROM experiments GROUP BY status`)
	if err != nil {
		return s, fmt.Errorf("query experiment status counts: %w", err)
	}
	s.ExperimentsByStatus = withKnownKeys(counts, "proposed", "running", "completed", "failed")
	return s, nil
}

func (r *Repository) RecentEvents(ctx context.Context, limit int) ([]RecentEvent, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, type, title, status, created_at
		FROM (
			SELECT id::text, 'workflow' AS type, 'Workflow instance' AS title, status::text, created_at
			FROM workflow_instances
			UNION ALL
			SELECT id::text, 'signal' AS type, signal_type AS title, CASE WHEN acknowledged THEN 'acknowledged' ELSE 'open' END AS status, created_at
			FROM signals
			UNION ALL
			SELECT id::text, 'verification' AS type, 'Verification report' AS title, 'reported' AS status, created_at
			FROM verification_reports
			UNION ALL
			SELECT id::text, 'experiment' AS type, name AS title, status, created_at
			FROM experiments
			UNION ALL
			SELECT id::text, 'trace' AS type, 'Execution trace' AS title, status, started_at AS created_at
			FROM traces
		) events
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query recent events: %w", err)
	}
	defer rows.Close()

	events := make([]RecentEvent, 0, limit)
	for rows.Next() {
		var event RecentEvent
		if err := rows.Scan(&event.ID, &event.Type, &event.Title, &event.Status, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan recent event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent events: %w", err)
	}
	return events, nil
}

func (r *Repository) countBy(ctx context.Context, query string) (map[string]int64, error) {
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
