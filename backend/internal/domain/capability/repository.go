package capability

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

func (r *Repository) CreateCapability(ctx context.Context, input CreateCapabilityInput) (*Capability, error) {
	if input.Version == "" {
		input.Version = "1.0"
	}
	if input.PermissionLevel == "" {
		input.PermissionLevel = "L2"
	}

	inSchema, _ := json.Marshal(input.InputSchema)
	outSchema, _ := json.Marshal(input.OutputSchema)
	preconds, _ := json.Marshal(input.Preconditions)
	errHandling, _ := json.Marshal(input.ErrorHandling)
	costEst, _ := json.Marshal(input.CostEstimate)

	cap := &Capability{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO capabilities (name, version, description, input_schema, output_schema, preconditions, error_handling, permission_level, cost_estimate)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, name, version, description, input_schema, output_schema, preconditions, error_handling, permission_level, cost_estimate, is_active, created_at, updated_at`,
		input.Name, input.Version, input.Description, inSchema, outSchema, preconds, errHandling, input.PermissionLevel, costEst,
	).Scan(&cap.ID, &cap.Name, &cap.Version, &cap.Description, &inSchema, &outSchema, &preconds, &errHandling, &cap.PermissionLevel, &costEst, &cap.IsActive, &cap.CreatedAt, &cap.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create capability: %w", err)
	}
	json.Unmarshal(inSchema, &cap.InputSchema)
	json.Unmarshal(outSchema, &cap.OutputSchema)
	json.Unmarshal(preconds, &cap.Preconditions)
	json.Unmarshal(errHandling, &cap.ErrorHandling)
	json.Unmarshal(costEst, &cap.CostEstimate)
	return cap, nil
}

func (r *Repository) GetCapabilityByID(ctx context.Context, id uuid.UUID) (*Capability, error) {
	cap := &Capability{}
	var inSchema, outSchema, preconds, errHandling, costEst []byte
	err := r.db.QueryRow(ctx,
		`SELECT id, name, version, description, input_schema, output_schema, preconditions, error_handling, permission_level, cost_estimate, is_active, created_at, updated_at
		 FROM capabilities WHERE id = $1`, id,
	).Scan(&cap.ID, &cap.Name, &cap.Version, &cap.Description, &inSchema, &outSchema, &preconds, &errHandling, &cap.PermissionLevel, &costEst, &cap.IsActive, &cap.CreatedAt, &cap.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get capability: %w", err)
	}
	json.Unmarshal(inSchema, &cap.InputSchema)
	json.Unmarshal(outSchema, &cap.OutputSchema)
	json.Unmarshal(preconds, &cap.Preconditions)
	json.Unmarshal(errHandling, &cap.ErrorHandling)
	json.Unmarshal(costEst, &cap.CostEstimate)
	return cap, nil
}

func (r *Repository) ListCapabilities(ctx context.Context) ([]Capability, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, version, description, input_schema, output_schema, preconditions, error_handling, permission_level, cost_estimate, is_active, created_at, updated_at
		 FROM capabilities ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list capabilities: %w", err)
	}
	defer rows.Close()

	var caps []Capability
	for rows.Next() {
		var cap Capability
		var inSchema, outSchema, preconds, errHandling, costEst []byte
		if err := rows.Scan(&cap.ID, &cap.Name, &cap.Version, &cap.Description, &inSchema, &outSchema, &preconds, &errHandling, &cap.PermissionLevel, &costEst, &cap.IsActive, &cap.CreatedAt, &cap.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan capability: %w", err)
		}
		json.Unmarshal(inSchema, &cap.InputSchema)
		json.Unmarshal(outSchema, &cap.OutputSchema)
		json.Unmarshal(preconds, &cap.Preconditions)
		json.Unmarshal(errHandling, &cap.ErrorHandling)
		json.Unmarshal(costEst, &cap.CostEstimate)
		caps = append(caps, cap)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list capabilities iteration: %w", err)
	}
	return caps, nil
}

func (r *Repository) BindCapability(ctx context.Context, input BindCapabilityInput) (*CapabilityBinding, error) {
	configJSON, _ := json.Marshal(input.Config)
	binding := &CapabilityBinding{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO capability_bindings (capability_id, mvru_id, config) VALUES ($1, $2, $3)
		 RETURNING id, capability_id, mvru_id, config, created_at`,
		input.CapabilityID, input.MVRUID, configJSON,
	).Scan(&binding.ID, &binding.CapabilityID, &binding.MVRUID, &configJSON, &binding.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("bind capability: %w", err)
	}
	json.Unmarshal(configJSON, &binding.Config)
	return binding, nil
}

func (r *Repository) UnbindCapability(ctx context.Context, bindingID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `DELETE FROM capability_bindings WHERE id = $1`, bindingID)
	if err != nil {
		return fmt.Errorf("unbind capability: %w", err)
	}
	return nil
}

func (r *Repository) ListBoundCapabilities(ctx context.Context, mvruID uuid.UUID) ([]CapabilityBinding, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, capability_id, mvru_id, config, created_at
		 FROM capability_bindings WHERE mvru_id = $1 ORDER BY created_at`, mvruID)
	if err != nil {
		return nil, fmt.Errorf("list bound capabilities: %w", err)
	}
	defer rows.Close()

	var bindings []CapabilityBinding
	for rows.Next() {
		var b CapabilityBinding
		var configJSON []byte
		if err := rows.Scan(&b.ID, &b.CapabilityID, &b.MVRUID, &configJSON, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan binding: %w", err)
		}
		json.Unmarshal(configJSON, &b.Config)
		bindings = append(bindings, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list bindings iteration: %w", err)
	}
	return bindings, nil
}

func (r *Repository) RecordInvocation(ctx context.Context, inv CapabilityInvocation) (*CapabilityInvocation, error) {
	inputJSON, _ := json.Marshal(inv.Input)
	outputJSON, _ := json.Marshal(inv.Output)
	err := r.db.QueryRow(ctx,
		`INSERT INTO capability_invocations (capability_id, caller_id, caller_type, input, output, duration_ms, cost, outcome, trace_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, capability_id, caller_id, caller_type, input, output, duration_ms, cost, outcome, trace_id, created_at`,
		inv.CapabilityID, inv.CallerID, inv.CallerType, inputJSON, outputJSON, inv.DurationMs, inv.Cost, inv.Outcome, inv.TraceID,
	).Scan(&inv.ID, &inv.CapabilityID, &inv.CallerID, &inv.CallerType, &inputJSON, &outputJSON, &inv.DurationMs, &inv.Cost, &inv.Outcome, &inv.TraceID, &inv.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("record invocation: %w", err)
	}
	json.Unmarshal(inputJSON, &inv.Input)
	json.Unmarshal(outputJSON, &inv.Output)
	return &inv, nil
}

func (r *Repository) CreateCapabilityEvaluation(ctx context.Context, input CreateCapabilityEvaluationInput, overallScore float64) (*CapabilityEvaluation, error) {
	evidenceJSON, _ := json.Marshal(input.Evidence)
	eval := &CapabilityEvaluation{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO capability_evaluations (
		    capability_id, actor_id, actor_type, workflow_id, task_id, evaluator_id, evaluator_type,
		    quality_score, reliability_score, cost_score, latency_score, risk_score, compliance_score,
		    overall_score, evidence, conclusion
		 )
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		 RETURNING id, capability_id, actor_id, COALESCE(actor_type, ''), workflow_id, task_id, evaluator_id, evaluator_type,
		           quality_score, reliability_score, cost_score, latency_score, risk_score, compliance_score,
		           overall_score, evidence, COALESCE(conclusion, ''), created_at`,
		input.CapabilityID, input.ActorID, input.ActorType, input.WorkflowID, input.TaskID, input.EvaluatorID, input.EvaluatorType,
		input.QualityScore, input.ReliabilityScore, input.CostScore, input.LatencyScore, input.RiskScore, input.ComplianceScore,
		overallScore, evidenceJSON, input.Conclusion,
	).Scan(&eval.ID, &eval.CapabilityID, &eval.ActorID, &eval.ActorType, &eval.WorkflowID, &eval.TaskID, &eval.EvaluatorID, &eval.EvaluatorType,
		&eval.QualityScore, &eval.ReliabilityScore, &eval.CostScore, &eval.LatencyScore, &eval.RiskScore, &eval.ComplianceScore,
		&eval.OverallScore, &evidenceJSON, &eval.Conclusion, &eval.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create capability evaluation: %w", err)
	}
	json.Unmarshal(evidenceJSON, &eval.Evidence)
	return eval, nil
}

func (r *Repository) ListCapabilityEvaluations(ctx context.Context, capabilityID *uuid.UUID, limit int) ([]CapabilityEvaluation, error) {
	if limit <= 0 {
		limit = 50
	} else if limit > 100 {
		limit = 100
	}
	query := `SELECT id, capability_id, actor_id, COALESCE(actor_type, ''), workflow_id, task_id, evaluator_id, evaluator_type,
	                 quality_score, reliability_score, cost_score, latency_score, risk_score, compliance_score,
	                 overall_score, evidence, COALESCE(conclusion, ''), created_at
	          FROM capability_evaluations`
	args := []any{}
	if capabilityID != nil {
		query += ` WHERE capability_id = $1 ORDER BY created_at DESC LIMIT $2`
		args = append(args, *capabilityID, limit)
	} else {
		query += ` ORDER BY created_at DESC LIMIT $1`
		args = append(args, limit)
	}
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list capability evaluations: %w", err)
	}
	defer rows.Close()

	var evaluations []CapabilityEvaluation
	for rows.Next() {
		var eval CapabilityEvaluation
		var evidenceJSON []byte
		if err := rows.Scan(&eval.ID, &eval.CapabilityID, &eval.ActorID, &eval.ActorType, &eval.WorkflowID, &eval.TaskID, &eval.EvaluatorID, &eval.EvaluatorType,
			&eval.QualityScore, &eval.ReliabilityScore, &eval.CostScore, &eval.LatencyScore, &eval.RiskScore, &eval.ComplianceScore,
			&eval.OverallScore, &evidenceJSON, &eval.Conclusion, &eval.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan capability evaluation: %w", err)
		}
		json.Unmarshal(evidenceJSON, &eval.Evidence)
		evaluations = append(evaluations, eval)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list capability evaluations iteration: %w", err)
	}
	return evaluations, nil
}
