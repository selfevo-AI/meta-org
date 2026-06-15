package metaresource

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateResource(ctx context.Context, input CreateMetaResourceInput) (*MetaResource, error) {
	item := &MetaResource{}
	err := scanResource(r.db.QueryRow(ctx, `
		INSERT INTO meta_resources (
			resource_type, source_type, source_id, name, status, organization_id, department_id,
			owner_actor_id, owner_actor_type, capability_profile, cost_profile, capacity_profile,
			risk_profile, performance_profile, metadata
		)
		VALUES ($1, $2, $3, $4, COALESCE(NULLIF($5, ''), 'active'), $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		ON CONFLICT (resource_type, source_type, source_id)
			WHERE source_type <> '' AND source_id IS NOT NULL
		DO UPDATE SET name = EXCLUDED.name,
			status = EXCLUDED.status,
			organization_id = EXCLUDED.organization_id,
			department_id = EXCLUDED.department_id,
			owner_actor_id = EXCLUDED.owner_actor_id,
			owner_actor_type = EXCLUDED.owner_actor_type,
			capability_profile = EXCLUDED.capability_profile,
			cost_profile = EXCLUDED.cost_profile,
			capacity_profile = EXCLUDED.capacity_profile,
			risk_profile = EXCLUDED.risk_profile,
			performance_profile = EXCLUDED.performance_profile,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()
		RETURNING id, resource_type, source_type, source_id, name, status, organization_id, department_id,
			owner_actor_id, owner_actor_type, capability_profile, cost_profile, capacity_profile,
			risk_profile, performance_profile, metadata, created_at, updated_at
	`, input.ResourceType, input.SourceType, input.SourceID, input.Name, input.Status, input.OrganizationID, input.DepartmentID,
		input.OwnerActorID, input.OwnerActorType, mustJSON(input.CapabilityProfile), mustJSON(input.CostProfile), mustJSON(input.CapacityProfile),
		mustJSON(input.RiskProfile), mustJSON(input.PerformanceProfile), mustJSON(input.Metadata)), item)
	if err != nil {
		return nil, fmt.Errorf("create meta resource: %w", err)
	}
	return item, nil
}

func (r *Repository) ListResources(ctx context.Context, filter ListFilter) ([]MetaResource, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, resource_type, source_type, source_id, name, status, organization_id, department_id,
			owner_actor_id, owner_actor_type, capability_profile, cost_profile, capacity_profile,
			risk_profile, performance_profile, metadata, created_at, updated_at
		FROM meta_resources
		WHERE ($1 = '' OR resource_type = $1)
		  AND ($2 = '' OR status = $2)
		ORDER BY updated_at DESC
		LIMIT $3
	`, filter.ResourceType, filter.Status, normalizeLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list meta resources: %w", err)
	}
	defer rows.Close()
	return scanResources(rows)
}

func (r *Repository) ResourceSummary(ctx context.Context, limit int) (*ResourceSummary, error) {
	summary := &ResourceSummary{
		ByType:   map[string]int{},
		ByStatus: map[string]int{},
		Metadata: map[string]any{},
	}
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(*) FILTER (WHERE status = 'active')
		FROM meta_resources
	`).Scan(&summary.Total, &summary.Active); err != nil {
		return nil, fmt.Errorf("resource summary total: %w", err)
	}
	rows, err := r.db.Query(ctx, `SELECT resource_type, COUNT(*) FROM meta_resources GROUP BY resource_type`)
	if err != nil {
		return nil, fmt.Errorf("resource summary by type: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var key string
		var count int
		if err := rows.Scan(&key, &count); err != nil {
			return nil, fmt.Errorf("scan resource type summary: %w", err)
		}
		summary.ByType[key] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	statusRows, err := r.db.Query(ctx, `SELECT status, COUNT(*) FROM meta_resources GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("resource summary by status: %w", err)
	}
	defer statusRows.Close()
	for statusRows.Next() {
		var key string
		var count int
		if err := statusRows.Scan(&key, &count); err != nil {
			return nil, fmt.Errorf("scan resource status summary: %w", err)
		}
		summary.ByStatus[key] = count
	}
	if err := statusRows.Err(); err != nil {
		return nil, err
	}
	recent, err := r.ListResources(ctx, ListFilter{Limit: limit})
	if err != nil {
		return nil, err
	}
	summary.Recent = recent
	return summary, nil
}

func (r *Repository) CreateDemandProfile(ctx context.Context, input CreateDemandProfileInput) (*DemandProfile, error) {
	item := &DemandProfile{}
	err := scanDemandProfile(r.db.QueryRow(ctx, `
		INSERT INTO demand_profiles (
			requirement_id, project_id, title, goal, status, acceptance_criteria,
			required_capabilities, budget_constraints, time_constraints, risk_constraints,
			resource_fit_candidates, metadata
		)
		VALUES ($1, $2, $3, $4, COALESCE(NULLIF($5, ''), 'draft'), $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, requirement_id, project_id, title, goal, status, acceptance_criteria,
			required_capabilities, budget_constraints, time_constraints, risk_constraints,
			resource_fit_candidates, metadata, created_at, updated_at
	`, input.RequirementID, input.ProjectID, input.Title, input.Goal, input.Status, mustJSON(input.AcceptanceCriteria),
		mustJSON(input.RequiredCapabilities), mustJSON(input.BudgetConstraints), mustJSON(input.TimeConstraints),
		mustJSON(input.RiskConstraints), mustJSON(input.ResourceFitCandidates), mustJSON(input.Metadata)), item)
	if err != nil {
		return nil, fmt.Errorf("create demand profile: %w", err)
	}
	return item, nil
}

func (r *Repository) ListDemandProfiles(ctx context.Context, limit int) ([]DemandProfile, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, requirement_id, project_id, title, goal, status, acceptance_criteria,
			required_capabilities, budget_constraints, time_constraints, risk_constraints,
			resource_fit_candidates, metadata, created_at, updated_at
		FROM demand_profiles
		ORDER BY updated_at DESC
		LIMIT $1
	`, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list demand profiles: %w", err)
	}
	defer rows.Close()
	items := []DemandProfile{}
	for rows.Next() {
		var item DemandProfile
		if err := scanDemandProfile(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateCycle(ctx context.Context, input CreatePDCACycleInput) (*PDCACycle, error) {
	item := &PDCACycle{}
	err := scanCycle(r.db.QueryRow(ctx, `
		INSERT INTO pdca_cycles (
			demand_profile_id, requirement_id, project_id, status, current_stage, summary, metadata
		)
		VALUES ($1, $2, $3, COALESCE(NULLIF($4, ''), 'active'), COALESCE(NULLIF($5, ''), 'plan'), $6, $7)
		RETURNING id, demand_profile_id, requirement_id, project_id, status, current_stage,
			outcome_score, summary, metadata, created_at, updated_at, completed_at
	`, input.DemandProfileID, input.RequirementID, input.ProjectID, input.Status, input.CurrentStage, input.Summary, mustJSON(input.Metadata)), item)
	if err != nil {
		return nil, fmt.Errorf("create pdca cycle: %w", err)
	}
	return item, nil
}

func (r *Repository) ListCycles(ctx context.Context, limit int) ([]PDCACycle, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, demand_profile_id, requirement_id, project_id, status, current_stage,
			outcome_score, summary, metadata, created_at, updated_at, completed_at
		FROM pdca_cycles
		ORDER BY updated_at DESC
		LIMIT $1
	`, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list pdca cycles: %w", err)
	}
	defer rows.Close()
	items := []PDCACycle{}
	for rows.Next() {
		var item PDCACycle
		if err := scanCycle(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) CreateEvent(ctx context.Context, input CreatePDCAEventInput) (*PDCAEvent, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin create pdca event: %w", err)
	}
	defer tx.Rollback(ctx)

	item := &PDCAEvent{}
	err = scanEvent(tx.QueryRow(ctx, `
		INSERT INTO pdca_events (
			cycle_id, stage, event_type, source_type, source_id, actor_id, actor_type,
			resource_refs, cost_refs, evidence, decision, next_action, metadata
		)
		VALUES ($1, $2, COALESCE(NULLIF($3, ''), 'note'), $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING id, cycle_id, stage, event_type, source_type, source_id, actor_id, actor_type,
			resource_refs, cost_refs, evidence, decision, next_action, metadata, created_at
	`, input.CycleID, input.Stage, input.EventType, input.SourceType, input.SourceID, input.ActorID, input.ActorType,
		mustJSON(input.ResourceRefs), mustJSON(input.CostRefs), mustJSON(input.Evidence), input.Decision, input.NextAction, mustJSON(input.Metadata)), item)
	if err != nil {
		return nil, fmt.Errorf("create pdca event: %w", err)
	}
	status := "active"
	completedSQL := "NULL"
	if input.Stage == StageAccept && (strings.Contains(strings.ToLower(input.EventType), "accept") || strings.Contains(strings.ToLower(input.EventType), "close")) {
		status = "completed"
		completedSQL = "NOW()"
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf(`
		UPDATE pdca_cycles
		SET current_stage = $2, status = $3, updated_at = NOW(), completed_at = COALESCE(completed_at, %s)
		WHERE id = $1
	`, completedSQL), input.CycleID, input.Stage, status); err != nil {
		return nil, fmt.Errorf("update pdca cycle stage: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create pdca event: %w", err)
	}
	return item, nil
}

func (r *Repository) ListEvents(ctx context.Context, filter ListFilter) ([]PDCAEvent, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, cycle_id, stage, event_type, source_type, source_id, actor_id, actor_type,
			resource_refs, cost_refs, evidence, decision, next_action, metadata, created_at
		FROM pdca_events
		WHERE ($1::uuid IS NULL OR cycle_id = $1)
		ORDER BY created_at DESC
		LIMIT $2
	`, filter.CycleID, normalizeLimit(filter.Limit))
	if err != nil {
		return nil, fmt.Errorf("list pdca events: %w", err)
	}
	defer rows.Close()
	items := []PDCAEvent{}
	for rows.Next() {
		var item PDCAEvent
		if err := scanEvent(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repository) SyncExistingResources(ctx context.Context) (map[string]int, error) {
	result := map[string]int{}
	statements := []struct {
		key string
		sql string
	}{
		{"human", `
			INSERT INTO meta_resources (resource_type, source_type, source_id, name, owner_actor_id, owner_actor_type, capability_profile, metadata)
			SELECT 'human', 'users', id, name, id, 'internal_human',
				jsonb_build_object('roles', COALESCE((SELECT jsonb_agg(r.name) FROM user_roles ur JOIN roles r ON r.id = ur.role_id WHERE ur.user_id = users.id), '[]'::jsonb)),
				jsonb_build_object('email', email)
			FROM users
			ON CONFLICT (resource_type, source_type, source_id) WHERE source_type <> '' AND source_id IS NOT NULL
			DO UPDATE SET name = EXCLUDED.name, capability_profile = EXCLUDED.capability_profile, metadata = EXCLUDED.metadata, updated_at = NOW()
		`},
		{"agent", `
			INSERT INTO meta_resources (resource_type, source_type, source_id, name, status, owner_actor_id, owner_actor_type, capability_profile, risk_profile, metadata)
			SELECT 'agent', 'ai_agents', id, name, CASE WHEN is_active THEN 'active' ELSE 'inactive' END, id, 'agent',
				jsonb_build_object('model_type', model_type, 'capabilities', capabilities, 'service_class', service_class),
				jsonb_build_object('permission_level', permission_level, 'risk_level', risk_level),
				metadata
			FROM ai_agents
			ON CONFLICT (resource_type, source_type, source_id) WHERE source_type <> '' AND source_id IS NOT NULL
			DO UPDATE SET name = EXCLUDED.name, status = EXCLUDED.status, capability_profile = EXCLUDED.capability_profile, risk_profile = EXCLUDED.risk_profile, metadata = EXCLUDED.metadata, updated_at = NOW()
		`},
		{"external_human", `
			INSERT INTO meta_resources (resource_type, source_type, source_id, name, status, metadata)
			SELECT 'external_human', 'external_members', id, name, status, jsonb_build_object('email', email, 'vendor', vendor, 'contract_type', contract_type) || metadata
			FROM external_members
			ON CONFLICT (resource_type, source_type, source_id) WHERE source_type <> '' AND source_id IS NOT NULL
			DO UPDATE SET name = EXCLUDED.name, status = EXCLUDED.status, metadata = EXCLUDED.metadata, updated_at = NOW()
		`},
		{"model_channel", `
			INSERT INTO meta_resources (resource_type, source_type, source_id, name, status, owner_actor_id, owner_actor_type, capability_profile, cost_profile, capacity_profile, risk_profile, metadata)
			SELECT 'model_channel', 'model_provider_channels', c.id, c.name,
				CASE WHEN c.status = 'disabled' THEN 'inactive' WHEN c.status = 'quota_exhausted' THEN 'exhausted' ELSE 'active' END,
				COALESCE(c.user_id, c.agent_id), COALESCE(NULLIF(c.owner_type, ''), 'system'),
				jsonb_build_object('provider_type', p.provider_type, 'supported_model_patterns', c.supported_model_patterns, 'model_mapping', c.model_mapping),
				jsonb_build_object('rate_multiplier', c.rate_multiplier, 'quota_amount', c.quota_amount, 'quota_used', c.quota_used, 'quota_currency', c.quota_currency),
				jsonb_build_object('concurrency_limit', c.concurrency_limit, 'inflight_requests', c.inflight_requests, 'load_factor', c.load_factor),
				jsonb_build_object('health_status', c.health_status, 'last_error', c.last_error),
				c.metadata
			FROM model_provider_channels c
			JOIN model_providers p ON p.id = c.provider_id
			ON CONFLICT (resource_type, source_type, source_id) WHERE source_type <> '' AND source_id IS NOT NULL
			DO UPDATE SET name = EXCLUDED.name, status = EXCLUDED.status, capability_profile = EXCLUDED.capability_profile, cost_profile = EXCLUDED.cost_profile, capacity_profile = EXCLUDED.capacity_profile, risk_profile = EXCLUDED.risk_profile, metadata = EXCLUDED.metadata, updated_at = NOW()
		`},
		{"tool", `
			INSERT INTO meta_resources (resource_type, source_type, source_id, name, status, capability_profile, risk_profile, metadata)
			SELECT 'tool', 'tool_definitions', id, name, CASE WHEN is_active THEN 'active' ELSE 'inactive' END,
				jsonb_build_object('source_type', source_type, 'input_schema', input_schema, 'output_schema', output_schema),
				jsonb_build_object('default_policy', default_policy, 'risk_level', risk_level, 'required_level', required_level),
				metadata
			FROM tool_definitions
			ON CONFLICT (resource_type, source_type, source_id) WHERE source_type <> '' AND source_id IS NOT NULL
			DO UPDATE SET name = EXCLUDED.name, status = EXCLUDED.status, capability_profile = EXCLUDED.capability_profile, risk_profile = EXCLUDED.risk_profile, metadata = EXCLUDED.metadata, updated_at = NOW()
		`},
		{"capability", `
			INSERT INTO meta_resources (resource_type, source_type, source_id, name, status, capability_profile, cost_profile, risk_profile)
			SELECT 'capability', 'capabilities', id, name, CASE WHEN is_active THEN 'active' ELSE 'inactive' END,
				jsonb_build_object('version', version, 'input_schema', input_schema, 'output_schema', output_schema, 'preconditions', preconditions),
				cost_estimate,
				jsonb_build_object('permission_level', permission_level, 'error_handling', error_handling)
			FROM capabilities
			ON CONFLICT (resource_type, source_type, source_id) WHERE source_type <> '' AND source_id IS NOT NULL
			DO UPDATE SET name = EXCLUDED.name, status = EXCLUDED.status, capability_profile = EXCLUDED.capability_profile, cost_profile = EXCLUDED.cost_profile, risk_profile = EXCLUDED.risk_profile, updated_at = NOW()
		`},
	}
	for _, statement := range statements {
		tag, err := r.db.Exec(ctx, statement.sql)
		if err != nil {
			return result, fmt.Errorf("sync %s resources: %w", statement.key, err)
		}
		result[statement.key] = int(tag.RowsAffected())
	}
	return result, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanResource(row scanner, item *MetaResource) error {
	var sourceID, orgID, deptID, ownerID *uuid.UUID
	var capabilityJSON, costJSON, capacityJSON, riskJSON, performanceJSON, metadataJSON []byte
	if err := row.Scan(&item.ID, &item.ResourceType, &item.SourceType, &sourceID, &item.Name, &item.Status,
		&orgID, &deptID, &ownerID, &item.OwnerActorType, &capabilityJSON, &costJSON, &capacityJSON,
		&riskJSON, &performanceJSON, &metadataJSON, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return err
	}
	item.SourceID = sourceID
	item.OrganizationID = orgID
	item.DepartmentID = deptID
	item.OwnerActorID = ownerID
	item.CapabilityProfile = mapFromJSON(capabilityJSON)
	item.CostProfile = mapFromJSON(costJSON)
	item.CapacityProfile = mapFromJSON(capacityJSON)
	item.RiskProfile = mapFromJSON(riskJSON)
	item.PerformanceProfile = mapFromJSON(performanceJSON)
	item.Metadata = mapFromJSON(metadataJSON)
	return nil
}

func scanResources(rows pgx.Rows) ([]MetaResource, error) {
	items := []MetaResource{}
	for rows.Next() {
		var item MetaResource
		if err := scanResource(rows, &item); err != nil {
			return nil, fmt.Errorf("scan meta resource: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func scanDemandProfile(row scanner, item *DemandProfile) error {
	var acceptanceJSON, requiredJSON, budgetJSON, timeJSON, riskJSON, candidatesJSON, metadataJSON []byte
	if err := row.Scan(&item.ID, &item.RequirementID, &item.ProjectID, &item.Title, &item.Goal, &item.Status,
		&acceptanceJSON, &requiredJSON, &budgetJSON, &timeJSON, &riskJSON, &candidatesJSON, &metadataJSON,
		&item.CreatedAt, &item.UpdatedAt); err != nil {
		return err
	}
	item.AcceptanceCriteria = listFromJSON(acceptanceJSON)
	item.RequiredCapabilities = listFromJSON(requiredJSON)
	item.BudgetConstraints = mapFromJSON(budgetJSON)
	item.TimeConstraints = mapFromJSON(timeJSON)
	item.RiskConstraints = mapFromJSON(riskJSON)
	item.ResourceFitCandidates = listFromJSON(candidatesJSON)
	item.Metadata = mapFromJSON(metadataJSON)
	return nil
}

func scanCycle(row scanner, item *PDCACycle) error {
	var metadataJSON []byte
	if err := row.Scan(&item.ID, &item.DemandProfileID, &item.RequirementID, &item.ProjectID, &item.Status,
		&item.CurrentStage, &item.OutcomeScore, &item.Summary, &metadataJSON, &item.CreatedAt, &item.UpdatedAt,
		&item.CompletedAt); err != nil {
		return err
	}
	item.Metadata = mapFromJSON(metadataJSON)
	return nil
}

func scanEvent(row scanner, item *PDCAEvent) error {
	var resourceJSON, costJSON, evidenceJSON, metadataJSON []byte
	if err := row.Scan(&item.ID, &item.CycleID, &item.Stage, &item.EventType, &item.SourceType, &item.SourceID,
		&item.ActorID, &item.ActorType, &resourceJSON, &costJSON, &evidenceJSON, &item.Decision,
		&item.NextAction, &metadataJSON, &item.CreatedAt); err != nil {
		return err
	}
	item.ResourceRefs = listFromJSON(resourceJSON)
	item.CostRefs = listFromJSON(costJSON)
	item.Evidence = mapFromJSON(evidenceJSON)
	item.Metadata = mapFromJSON(metadataJSON)
	return nil
}

func mustJSON(value any) []byte {
	if value == nil {
		return []byte("{}")
	}
	switch v := value.(type) {
	case []map[string]any:
		if v == nil {
			return []byte("[]")
		}
	case []any:
		if v == nil {
			return []byte("[]")
		}
	}
	data, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return data
}

func mapFromJSON(data []byte) map[string]any {
	result := map[string]any{}
	if len(data) == 0 {
		return result
	}
	_ = json.Unmarshal(data, &result)
	return result
}

func listFromJSON(data []byte) []any {
	result := []any{}
	if len(data) == 0 {
		return result
	}
	if err := json.Unmarshal(data, &result); err == nil {
		return result
	}
	var object map[string]any
	if err := json.Unmarshal(data, &object); err == nil {
		return []any{object}
	}
	return result
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 100
	}
	if limit > 500 {
		return 500
	}
	return limit
}
