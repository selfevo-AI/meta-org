package governance

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreatePermission(ctx context.Context, p *Permission) (*Permission, error) {
	perm := &Permission{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO permissions (level, name, description, behavior)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, level, name, description, behavior, created_at`,
		p.Level, p.Name, p.Description, p.Behavior,
	).Scan(&perm.ID, &perm.Level, &perm.Name, &perm.Description, &perm.Behavior, &perm.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create permission: %w", err)
	}
	return perm, nil
}

func (r *Repository) ListPermissions(ctx context.Context) ([]Permission, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, level, name, description, behavior, created_at
		 FROM permissions ORDER BY level, name`)
	if err != nil {
		return nil, fmt.Errorf("list permissions: %w", err)
	}
	defer rows.Close()

	var perms []Permission
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.ID, &p.Level, &p.Name, &p.Description, &p.Behavior, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		perms = append(perms, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list permissions iteration: %w", err)
	}
	return perms, nil
}

func (r *Repository) GetPermissionByLevel(ctx context.Context, level int) (*Permission, error) {
	p := &Permission{}
	err := r.db.QueryRow(ctx,
		`SELECT id, level, name, description, behavior, created_at
		 FROM permissions WHERE level = $1`, level,
	).Scan(&p.ID, &p.Level, &p.Name, &p.Description, &p.Behavior, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get permission by level: %w", err)
	}
	return p, nil
}

func (r *Repository) CreatePrinciple(ctx context.Context, input CreatePrincipleInput) (*Principle, error) {
	evalJSON, _ := json.Marshal(input.EvaluationLogic)
	prin := &Principle{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO principles (name, description, evaluation_logic, priority)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, name, description, evaluation_logic, priority, is_active, created_at, updated_at`,
		input.Name, input.Description, evalJSON, input.Priority,
	).Scan(&prin.ID, &prin.Name, &prin.Description, &evalJSON, &prin.Priority, &prin.IsActive, &prin.CreatedAt, &prin.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create principle: %w", err)
	}
	json.Unmarshal(evalJSON, &prin.EvaluationLogic)
	return prin, nil
}

func (r *Repository) ListPrinciples(ctx context.Context) ([]Principle, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, description, evaluation_logic, priority, is_active, created_at, updated_at
		 FROM principles ORDER BY priority DESC, name`)
	if err != nil {
		return nil, fmt.Errorf("list principles: %w", err)
	}
	defer rows.Close()

	var principles []Principle
	for rows.Next() {
		var p Principle
		var evalJSON []byte
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &evalJSON, &p.Priority, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan principle: %w", err)
		}
		json.Unmarshal(evalJSON, &p.EvaluationLogic)
		principles = append(principles, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list principles iteration: %w", err)
	}
	return principles, nil
}

func (r *Repository) GetPrinciple(ctx context.Context, id uuid.UUID) (*Principle, error) {
	p := &Principle{}
	var evalJSON []byte
	err := r.db.QueryRow(ctx,
		`SELECT id, name, description, evaluation_logic, priority, is_active, created_at, updated_at
		 FROM principles WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.Description, &evalJSON, &p.Priority, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get principle: %w", err)
	}
	json.Unmarshal(evalJSON, &p.EvaluationLogic)
	return p, nil
}

func (r *Repository) CreateControlRule(ctx context.Context, input CreateControlRuleInput) (*ControlRule, error) {
	condJSON, _ := json.Marshal(input.Condition)
	rule := &ControlRule{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO control_rules (principle_id, target_entity_type, target_entity_id, condition, action, priority)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, principle_id, target_entity_type, target_entity_id, condition, action, priority, is_active, created_at`,
		input.PrincipleID, input.TargetEntityType, input.TargetEntityID, condJSON, input.Action, input.Priority,
	).Scan(&rule.ID, &rule.PrincipleID, &rule.TargetEntityType, &rule.TargetEntityID, &condJSON, &rule.Action, &rule.Priority, &rule.IsActive, &rule.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create control rule: %w", err)
	}
	json.Unmarshal(condJSON, &rule.Condition)
	return rule, nil
}

func (r *Repository) ListControlRules(ctx context.Context) ([]ControlRule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, principle_id, target_entity_type, target_entity_id, condition, action, priority, is_active, created_at
		 FROM control_rules ORDER BY priority DESC`)
	if err != nil {
		return nil, fmt.Errorf("list control rules: %w", err)
	}
	defer rows.Close()

	var rules []ControlRule
	for rows.Next() {
		var rule ControlRule
		var condJSON []byte
		if err := rows.Scan(&rule.ID, &rule.PrincipleID, &rule.TargetEntityType, &rule.TargetEntityID, &condJSON, &rule.Action, &rule.Priority, &rule.IsActive, &rule.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan control rule: %w", err)
		}
		json.Unmarshal(condJSON, &rule.Condition)
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list control rules iteration: %w", err)
	}
	return rules, nil
}

func (r *Repository) GetControlRulesByTarget(ctx context.Context, entityType string, entityID *uuid.UUID) ([]ControlRule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, principle_id, target_entity_type, target_entity_id, condition, action, priority, is_active, created_at
		 FROM control_rules
		 WHERE target_entity_type = $1 AND (target_entity_id = $2 OR target_entity_id IS NULL)
		 ORDER BY priority DESC`, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("get control rules by target: %w", err)
	}
	defer rows.Close()

	var rules []ControlRule
	for rows.Next() {
		var rule ControlRule
		var condJSON []byte
		if err := rows.Scan(&rule.ID, &rule.PrincipleID, &rule.TargetEntityType, &rule.TargetEntityID, &condJSON, &rule.Action, &rule.Priority, &rule.IsActive, &rule.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan control rule: %w", err)
		}
		json.Unmarshal(condJSON, &rule.Condition)
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get control rules by target iteration: %w", err)
	}
	return rules, nil
}

func (r *Repository) CreateAccessDecision(ctx context.Context, input AccessDecisionInput, decision, behavior, reason string, allowed bool, matchedRules []string) (*AccessDecision, error) {
	if input.Context == nil {
		input.Context = map[string]any{}
	}
	rulesJSON, _ := json.Marshal(matchedRules)
	contextJSON, _ := json.Marshal(input.Context)

	access := &AccessDecision{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO access_decisions (
		    actor_id, actor_type, action, resource, resource_id, organization_id, department_id,
		    workflow_id, task_id, capability_id, required_level, risk_level, decision, allowed,
		    behavior, reason, matched_rules, weight_snapshot, context
		 )
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		 RETURNING id, actor_id, actor_type, action, resource, resource_id, organization_id, department_id,
		           workflow_id, task_id, capability_id, required_level, risk_level, decision, allowed,
		           behavior, reason, matched_rules, weight_snapshot, context, created_at`,
		input.ActorID, input.ActorType, input.Action, input.Resource, input.ResourceID, input.OrganizationID, input.DepartmentID,
		input.WorkflowID, input.TaskID, input.CapabilityID, input.RequiredLevel, input.RiskLevel, decision, allowed,
		behavior, reason, rulesJSON, input.WeightSnapshot, contextJSON,
	).Scan(&access.ID, &access.ActorID, &access.ActorType, &access.Action, &access.Resource, &access.ResourceID, &access.OrganizationID, &access.DepartmentID,
		&access.WorkflowID, &access.TaskID, &access.CapabilityID, &access.RequiredLevel, &access.RiskLevel, &access.Decision, &access.Allowed,
		&access.Behavior, &access.Reason, &rulesJSON, &access.WeightSnapshot, &contextJSON, &access.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create access decision: %w", err)
	}
	json.Unmarshal(rulesJSON, &access.MatchedRules)
	json.Unmarshal(contextJSON, &access.Context)
	return access, nil
}

func (r *Repository) ListAccessDecisions(ctx context.Context, limit int) ([]AccessDecision, error) {
	if limit <= 0 {
		limit = 50
	} else if limit > 100 {
		limit = 100
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, actor_id, actor_type, action, resource, resource_id, organization_id, department_id,
		        workflow_id, task_id, capability_id, required_level, risk_level, decision, allowed,
		        behavior, reason, matched_rules, weight_snapshot, context, created_at
		 FROM access_decisions ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list access decisions: %w", err)
	}
	defer rows.Close()

	var decisions []AccessDecision
	for rows.Next() {
		var decision AccessDecision
		var rulesJSON, contextJSON []byte
		if err := rows.Scan(&decision.ID, &decision.ActorID, &decision.ActorType, &decision.Action, &decision.Resource, &decision.ResourceID, &decision.OrganizationID, &decision.DepartmentID,
			&decision.WorkflowID, &decision.TaskID, &decision.CapabilityID, &decision.RequiredLevel, &decision.RiskLevel, &decision.Decision, &decision.Allowed,
			&decision.Behavior, &decision.Reason, &rulesJSON, &decision.WeightSnapshot, &contextJSON, &decision.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan access decision: %w", err)
		}
		json.Unmarshal(rulesJSON, &decision.MatchedRules)
		json.Unmarshal(contextJSON, &decision.Context)
		decisions = append(decisions, decision)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list access decisions iteration: %w", err)
	}
	return decisions, nil
}
