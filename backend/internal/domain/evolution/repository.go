package evolution

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

func (r *Repository) UpsertWeight(ctx context.Context, w *DecisionWeight) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO weight_scores (actor_id, actor_type, overall_score, expertise_score, track_record_score, reliability_score, recency_score, context_fit_score, principle_score, decision_count, last_updated)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		 ON CONFLICT (actor_id, actor_type) DO UPDATE SET
		     overall_score = EXCLUDED.overall_score,
		     expertise_score = EXCLUDED.expertise_score,
		     track_record_score = EXCLUDED.track_record_score,
		     reliability_score = EXCLUDED.reliability_score,
		     recency_score = EXCLUDED.recency_score,
		     context_fit_score = EXCLUDED.context_fit_score,
		     principle_score = EXCLUDED.principle_score,
		     decision_count = EXCLUDED.decision_count,
		     last_updated = NOW()
		 RETURNING id, actor_id, actor_type, overall_score, expertise_score, track_record_score, reliability_score, recency_score, context_fit_score, principle_score, decision_count, last_updated`,
		w.ActorID, w.ActorType, w.OverallScore, w.ExpertiseScore, w.TrackRecordScore, w.ReliabilityScore, w.RecencyScore, w.ContextFitScore, w.PrincipleScore, w.DecisionCount,
	).Scan(&w.ID, &w.ActorID, &w.ActorType, &w.OverallScore, &w.ExpertiseScore, &w.TrackRecordScore, &w.ReliabilityScore, &w.RecencyScore, &w.ContextFitScore, &w.PrincipleScore, &w.DecisionCount, &w.LastUpdated)
	if err != nil {
		return fmt.Errorf("upsert weight: %w", err)
	}
	return nil
}

func (r *Repository) GetWeight(ctx context.Context, actorID uuid.UUID, actorType string) (*DecisionWeight, error) {
	w := &DecisionWeight{}
	err := r.db.QueryRow(ctx,
		`SELECT id, actor_id, actor_type, overall_score, expertise_score, track_record_score, reliability_score, recency_score, context_fit_score, principle_score, decision_count, last_updated
		 FROM weight_scores WHERE actor_id = $1 AND actor_type = $2`,
		actorID, actorType,
	).Scan(&w.ID, &w.ActorID, &w.ActorType, &w.OverallScore, &w.ExpertiseScore, &w.TrackRecordScore, &w.ReliabilityScore, &w.RecencyScore, &w.ContextFitScore, &w.PrincipleScore, &w.DecisionCount, &w.LastUpdated)
	if err != nil {
		return nil, fmt.Errorf("get weight: %w", err)
	}
	return w, nil
}

func (r *Repository) ListWeights(ctx context.Context, limit int) ([]DecisionWeight, error) {
	if limit <= 0 {
		limit = 50
	} else if limit > 100 {
		limit = 100
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, actor_id, actor_type, overall_score, expertise_score, track_record_score, reliability_score, recency_score, context_fit_score, principle_score, decision_count, last_updated
		 FROM weight_scores ORDER BY overall_score DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list weights: %w", err)
	}
	defer rows.Close()

	var weights []DecisionWeight
	for rows.Next() {
		var w DecisionWeight
		if err := rows.Scan(&w.ID, &w.ActorID, &w.ActorType, &w.OverallScore, &w.ExpertiseScore, &w.TrackRecordScore, &w.ReliabilityScore, &w.RecencyScore, &w.ContextFitScore, &w.PrincipleScore, &w.DecisionCount, &w.LastUpdated); err != nil {
			return nil, fmt.Errorf("scan weight: %w", err)
		}
		weights = append(weights, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list weights iteration: %w", err)
	}
	return weights, nil
}

func (r *Repository) UpsertContextWeight(ctx context.Context, w *ContextDecisionWeight) error {
	contextJSON, _ := json.Marshal(w.Context)
	err := r.db.QueryRow(ctx,
		`INSERT INTO context_weight_scores (
		    actor_id, actor_type, scope_hash, organization_id, department_id, workflow_template_id,
		    workflow_stage, task_type, capability_id, risk_level, overall_score, expertise_score,
		    track_record_score, reliability_score, recency_score, context_fit_score, principle_score,
		    decision_count, context, last_updated
		 )
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, NOW())
		 ON CONFLICT (actor_id, actor_type, scope_hash) DO UPDATE SET
		    organization_id = EXCLUDED.organization_id,
		    department_id = EXCLUDED.department_id,
		    workflow_template_id = EXCLUDED.workflow_template_id,
		    workflow_stage = EXCLUDED.workflow_stage,
		    task_type = EXCLUDED.task_type,
		    capability_id = EXCLUDED.capability_id,
		    risk_level = EXCLUDED.risk_level,
		    overall_score = EXCLUDED.overall_score,
		    expertise_score = EXCLUDED.expertise_score,
		    track_record_score = EXCLUDED.track_record_score,
		    reliability_score = EXCLUDED.reliability_score,
		    recency_score = EXCLUDED.recency_score,
		    context_fit_score = EXCLUDED.context_fit_score,
		    principle_score = EXCLUDED.principle_score,
		    decision_count = EXCLUDED.decision_count,
		    context = EXCLUDED.context,
		    last_updated = NOW()
		 RETURNING id, actor_id, actor_type, scope_hash, organization_id, department_id, workflow_template_id,
		           workflow_stage, task_type, capability_id, risk_level, overall_score, expertise_score,
		           track_record_score, reliability_score, recency_score, context_fit_score, principle_score,
		           decision_count, context, last_updated`,
		w.ActorID, w.ActorType, w.ScopeHash, w.OrganizationID, w.DepartmentID, w.WorkflowTemplateID,
		w.WorkflowStage, w.TaskType, w.CapabilityID, w.RiskLevel, w.OverallScore, w.ExpertiseScore,
		w.TrackRecordScore, w.ReliabilityScore, w.RecencyScore, w.ContextFitScore, w.PrincipleScore,
		w.DecisionCount, contextJSON,
	).Scan(&w.ID, &w.ActorID, &w.ActorType, &w.ScopeHash, &w.OrganizationID, &w.DepartmentID, &w.WorkflowTemplateID,
		&w.WorkflowStage, &w.TaskType, &w.CapabilityID, &w.RiskLevel, &w.OverallScore, &w.ExpertiseScore,
		&w.TrackRecordScore, &w.ReliabilityScore, &w.RecencyScore, &w.ContextFitScore, &w.PrincipleScore,
		&w.DecisionCount, &contextJSON, &w.LastUpdated)
	if err != nil {
		return fmt.Errorf("upsert context weight: %w", err)
	}
	json.Unmarshal(contextJSON, &w.Context)
	return nil
}

func (r *Repository) GetContextWeight(ctx context.Context, actorID uuid.UUID, actorType string, scopeHash string) (*ContextDecisionWeight, error) {
	w := &ContextDecisionWeight{}
	var contextJSON []byte
	err := r.db.QueryRow(ctx,
		`SELECT id, actor_id, actor_type, scope_hash, organization_id, department_id, workflow_template_id,
		        workflow_stage, task_type, capability_id, risk_level, overall_score, expertise_score,
		        track_record_score, reliability_score, recency_score, context_fit_score, principle_score,
		        decision_count, context, last_updated
		 FROM context_weight_scores
		 WHERE actor_id = $1 AND actor_type = $2 AND scope_hash = $3`,
		actorID, actorType, scopeHash,
	).Scan(&w.ID, &w.ActorID, &w.ActorType, &w.ScopeHash, &w.OrganizationID, &w.DepartmentID, &w.WorkflowTemplateID,
		&w.WorkflowStage, &w.TaskType, &w.CapabilityID, &w.RiskLevel, &w.OverallScore, &w.ExpertiseScore,
		&w.TrackRecordScore, &w.ReliabilityScore, &w.RecencyScore, &w.ContextFitScore, &w.PrincipleScore,
		&w.DecisionCount, &contextJSON, &w.LastUpdated)
	if err != nil {
		return nil, fmt.Errorf("get context weight: %w", err)
	}
	json.Unmarshal(contextJSON, &w.Context)
	return w, nil
}

func (r *Repository) ListContextWeights(ctx context.Context, limit int) ([]ContextDecisionWeight, error) {
	if limit <= 0 {
		limit = 50
	} else if limit > 100 {
		limit = 100
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, actor_id, actor_type, scope_hash, organization_id, department_id, workflow_template_id,
		        workflow_stage, task_type, capability_id, risk_level, overall_score, expertise_score,
		        track_record_score, reliability_score, recency_score, context_fit_score, principle_score,
		        decision_count, context, last_updated
		 FROM context_weight_scores ORDER BY overall_score DESC, last_updated DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list context weights: %w", err)
	}
	defer rows.Close()

	var weights []ContextDecisionWeight
	for rows.Next() {
		var w ContextDecisionWeight
		var contextJSON []byte
		if err := rows.Scan(&w.ID, &w.ActorID, &w.ActorType, &w.ScopeHash, &w.OrganizationID, &w.DepartmentID, &w.WorkflowTemplateID,
			&w.WorkflowStage, &w.TaskType, &w.CapabilityID, &w.RiskLevel, &w.OverallScore, &w.ExpertiseScore,
			&w.TrackRecordScore, &w.ReliabilityScore, &w.RecencyScore, &w.ContextFitScore, &w.PrincipleScore,
			&w.DecisionCount, &contextJSON, &w.LastUpdated); err != nil {
			return nil, fmt.Errorf("scan context weight: %w", err)
		}
		json.Unmarshal(contextJSON, &w.Context)
		weights = append(weights, w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list context weights iteration: %w", err)
	}
	return weights, nil
}

func (r *Repository) GetAlpha(ctx context.Context) (*AlphaConfig, error) {
	a := &AlphaConfig{}
	err := r.db.QueryRow(ctx,
		`SELECT id, alpha_expertise, alpha_track_record, alpha_reliability, alpha_recency, alpha_context_fit, alpha_principle, version, created_at
		 FROM weight_alphas ORDER BY version DESC LIMIT 1`,
	).Scan(&a.ID, &a.Expertise, &a.TrackRecord, &a.Reliability, &a.Recency, &a.ContextFit, &a.Principle, &a.Version, &a.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get alpha: %w", err)
	}
	return a, nil
}

func (r *Repository) UpdateAlpha(ctx context.Context, a *AlphaConfig) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO weight_alphas (alpha_expertise, alpha_track_record, alpha_reliability, alpha_recency, alpha_context_fit, alpha_principle, version)
		 VALUES ($1, $2, $3, $4, $5, $6, (SELECT COALESCE(MAX(version),0)+1 FROM weight_alphas))
		 RETURNING id, alpha_expertise, alpha_track_record, alpha_reliability, alpha_recency, alpha_context_fit, alpha_principle, version, created_at`,
		a.Expertise, a.TrackRecord, a.Reliability, a.Recency, a.ContextFit, a.Principle,
	).Scan(&a.ID, &a.Expertise, &a.TrackRecord, &a.Reliability, &a.Recency, &a.ContextFit, &a.Principle, &a.Version, &a.CreatedAt)
	if err != nil {
		return fmt.Errorf("update alpha: %w", err)
	}
	return nil
}

func (r *Repository) CreateExperiment(ctx context.Context, input CreateExperimentInput) (*Experiment, error) {
	overridesJSON, _ := json.Marshal(input.AlphaOverrides)
	criteriaJSON, _ := json.Marshal(input.SuccessCriteria)

	e := &Experiment{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO experiments (name, hypothesis, mvru_id, alpha_overrides, success_criteria)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, name, hypothesis, status, mvru_id, alpha_overrides, success_criteria, started_at, completed_at, conclusion, created_at`,
		input.Name, input.Hypothesis, input.MVRUID, overridesJSON, criteriaJSON,
	).Scan(&e.ID, &e.Name, &e.Hypothesis, &e.Status, &e.MVRUID, &overridesJSON, &criteriaJSON, &e.StartedAt, &e.CompletedAt, &e.Conclusion, &e.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create experiment: %w", err)
	}
	json.Unmarshal(overridesJSON, &e.AlphaOverrides)
	json.Unmarshal(criteriaJSON, &e.SuccessCriteria)
	return e, nil
}

func (r *Repository) ListExperiments(ctx context.Context) ([]Experiment, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, hypothesis, status, mvru_id, alpha_overrides, success_criteria, started_at, completed_at, conclusion, created_at
		 FROM experiments ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list experiments: %w", err)
	}
	defer rows.Close()

	var experiments []Experiment
	for rows.Next() {
		var e Experiment
		var overridesJSON, criteriaJSON []byte
		if err := rows.Scan(&e.ID, &e.Name, &e.Hypothesis, &e.Status, &e.MVRUID, &overridesJSON, &criteriaJSON, &e.StartedAt, &e.CompletedAt, &e.Conclusion, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan experiment: %w", err)
		}
		json.Unmarshal(overridesJSON, &e.AlphaOverrides)
		json.Unmarshal(criteriaJSON, &e.SuccessCriteria)
		experiments = append(experiments, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list experiments iteration: %w", err)
	}
	return experiments, nil
}

func (r *Repository) UpdateExperimentStatus(ctx context.Context, id uuid.UUID, status string, conclusion string) error {
	var query string
	if status == "completed" || status == "rolled_back" {
		query = `UPDATE experiments SET status = $1, conclusion = $2, completed_at = NOW() WHERE id = $3`
	} else {
		query = `UPDATE experiments SET status = $1, conclusion = $2 WHERE id = $3`
	}
	_, err := r.db.Exec(ctx, query, status, conclusion, id)
	if err != nil {
		return fmt.Errorf("update experiment status: %w", err)
	}
	return nil
}

func (r *Repository) CreateKnowledge(ctx context.Context, input CreateKnowledgeInput) (*KnowledgeEntry, error) {
	e := &KnowledgeEntry{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO knowledge_entries (workflow_id, title, content, tags, source)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, workflow_id, title, content, tags, source, created_at`,
		input.WorkflowID, input.Title, input.Content, input.Tags, input.Source,
	).Scan(&e.ID, &e.WorkflowID, &e.Title, &e.Content, &e.Tags, &e.Source, &e.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create knowledge: %w", err)
	}
	return e, nil
}

func (r *Repository) ListKnowledge(ctx context.Context, limit int) ([]KnowledgeEntry, error) {
	if limit <= 0 {
		limit = 50
	} else if limit > 100 {
		limit = 100
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, workflow_id, title, content, tags, source, created_at
		 FROM knowledge_entries ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list knowledge: %w", err)
	}
	defer rows.Close()

	var entries []KnowledgeEntry
	for rows.Next() {
		var e KnowledgeEntry
		if err := rows.Scan(&e.ID, &e.WorkflowID, &e.Title, &e.Content, &e.Tags, &e.Source, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan knowledge: %w", err)
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list knowledge iteration: %w", err)
	}
	return entries, nil
}

func (r *Repository) CreateSignal(ctx context.Context, input CreateSignalInput) (*Signal, error) {
	dataJSON, _ := json.Marshal(input.Data)
	s := &Signal{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO signals (signal_type, source, priority, data)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, signal_type, source, priority, data, acknowledged, created_at`,
		input.SignalType, input.Source, input.Priority, dataJSON,
	).Scan(&s.ID, &s.SignalType, &s.Source, &s.Priority, &dataJSON, &s.Acknowledged, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create signal: %w", err)
	}
	json.Unmarshal(dataJSON, &s.Data)
	return s, nil
}

func (r *Repository) ListSignals(ctx context.Context, acknowledged *bool, limit int) ([]Signal, error) {
	if limit <= 0 {
		limit = 50
	} else if limit > 100 {
		limit = 100
	}

	where := "WHERE 1=1"
	args := []any{}
	argIdx := 1

	if acknowledged != nil {
		where += fmt.Sprintf(" AND acknowledged = $%d", argIdx)
		args = append(args, *acknowledged)
		argIdx++
	}

	query := fmt.Sprintf(`SELECT id, signal_type, source, priority, data, acknowledged, created_at
		FROM signals %s ORDER BY priority DESC, created_at DESC LIMIT $%d`, where, argIdx)
	args = append(args, limit)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list signals: %w", err)
	}
	defer rows.Close()

	var signals []Signal
	for rows.Next() {
		var s Signal
		var dataJSON []byte
		if err := rows.Scan(&s.ID, &s.SignalType, &s.Source, &s.Priority, &dataJSON, &s.Acknowledged, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan signal: %w", err)
		}
		json.Unmarshal(dataJSON, &s.Data)
		signals = append(signals, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list signals iteration: %w", err)
	}
	return signals, nil
}

func (r *Repository) AcknowledgeSignal(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE signals SET acknowledged = true WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("acknowledge signal: %w", err)
	}
	return nil
}
