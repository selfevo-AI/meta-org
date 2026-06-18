# Phase 5: Evolution Domain Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the Evolution Domain — the Decision Weight Engine (core meta-algorithm), Sensing/Learning/Knowledge engines — completing all 9 backend domains.

**Architecture:** Decision Weight formula computed in-process (Go) with PostgreSQL persistence using heuristic scoring for all 6 dimensions. Meta-learning is manual (admin adjusts α parameters via API). Sensing/Learning/Knowledge engines are CRUD-based with stubs for autonomous logic. Follows existing 4-file domain pattern + migration.

**Tech Stack:** Go (chi router, pgxpool, google/uuid), PostgreSQL (JSONB, UUID), following existing domain patterns.

**Spec reference:** System design doc lines 389-457 (Evolution Domain), docs/superpowers/specs/2026-06-12-org-harness-system-design.md

---

### Task 1: Migration 010 — Evolution tables

**Files:**
- Create: `migrations/010_evolution.sql`

- [ ] **Step 1: Create 010_evolution.sql**

```sql
-- 010_evolution.sql

CREATE TABLE weight_scores (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id              UUID NOT NULL,
    actor_type            TEXT NOT NULL,
    overall_score         DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    expertise_score       DOUBLE PRECISION DEFAULT 0,
    track_record_score    DOUBLE PRECISION DEFAULT 0,
    reliability_score     DOUBLE PRECISION DEFAULT 0,
    recency_score         DOUBLE PRECISION DEFAULT 0,
    context_fit_score     DOUBLE PRECISION DEFAULT 0,
    principle_score       DOUBLE PRECISION DEFAULT 0,
    decision_count        INT NOT NULL DEFAULT 0,
    last_updated          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (actor_id, actor_type)
);

CREATE TABLE weight_alphas (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alpha_expertise     DOUBLE PRECISION NOT NULL DEFAULT 0.25,
    alpha_track_record  DOUBLE PRECISION NOT NULL DEFAULT 0.20,
    alpha_reliability   DOUBLE PRECISION NOT NULL DEFAULT 0.15,
    alpha_recency       DOUBLE PRECISION NOT NULL DEFAULT 0.10,
    alpha_context_fit   DOUBLE PRECISION NOT NULL DEFAULT 0.10,
    alpha_principle     DOUBLE PRECISION NOT NULL DEFAULT 0.20,
    version             INT NOT NULL DEFAULT 1,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE experiments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                TEXT NOT NULL,
    hypothesis          TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'proposed',
    mvru_id             UUID,
    alpha_overrides     JSONB,
    success_criteria    JSONB NOT NULL DEFAULT '{}',
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    conclusion          TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE knowledge_entries (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id         UUID REFERENCES workflow_instances(id) ON DELETE SET NULL,
    title               TEXT NOT NULL,
    content             TEXT NOT NULL,
    tags                TEXT[] DEFAULT '{}',
    source              TEXT NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE signals (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_type         TEXT NOT NULL,
    source              TEXT NOT NULL,
    priority            INT NOT NULL DEFAULT 0,
    data                JSONB NOT NULL DEFAULT '{}',
    acknowledged        BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_weight_actor ON weight_scores(actor_id, actor_type);
CREATE INDEX idx_weight_overall ON weight_scores(overall_score DESC);
CREATE INDEX idx_experiment_status ON experiments(status);
CREATE INDEX idx_knowledge_tags ON knowledge_entries USING GIN(tags);
CREATE INDEX idx_knowledge_source ON knowledge_entries(source);
CREATE INDEX idx_signals_priority ON signals(priority DESC, created_at DESC);
CREATE INDEX idx_signals_acknowledged ON signals(acknowledged);
```

- [ ] **Step 2: Commit**

```bash
git add migrations/010_evolution.sql && git commit -m "feat: add evolution migration with weight scores, alphas, experiments, knowledge, signals"
```

---

### Task 2: Evolution domain — models

**Files:**
- Create: `backend/internal/domain/evolution/model.go`
- Create directory: `backend/internal/domain/evolution/`

- [ ] **Step 1: Create model.go**

```go
package evolution

import (
	"time"

	"github.com/google/uuid"
)

type DecisionWeight struct {
	ID               uuid.UUID  `json:"id"`
	ActorID          uuid.UUID  `json:"actor_id"`
	ActorType        string     `json:"actor_type"`
	OverallScore     float64    `json:"overall_score"`
	ExpertiseScore   float64    `json:"expertise_score"`
	TrackRecordScore float64    `json:"track_record_score"`
	ReliabilityScore float64    `json:"reliability_score"`
	RecencyScore     float64    `json:"recency_score"`
	ContextFitScore  float64    `json:"context_fit_score"`
	PrincipleScore   float64    `json:"principle_score"`
	DecisionCount    int        `json:"decision_count"`
	LastUpdated      time.Time  `json:"last_updated"`
}

type AlphaConfig struct {
	ID              uuid.UUID `json:"id"`
	Expertise       float64   `json:"alpha_expertise"`
	TrackRecord     float64   `json:"alpha_track_record"`
	Reliability     float64   `json:"alpha_reliability"`
	Recency         float64   `json:"alpha_recency"`
	ContextFit      float64   `json:"alpha_context_fit"`
	Principle       float64   `json:"alpha_principle"`
	Version         int       `json:"version"`
	CreatedAt       time.Time `json:"created_at"`
}

type Experiment struct {
	ID              uuid.UUID          `json:"id"`
	Name            string             `json:"name"`
	Hypothesis      string             `json:"hypothesis"`
	Status          string             `json:"status"`
	MVRUID          *uuid.UUID         `json:"mvru_id,omitempty"`
	AlphaOverrides  map[string]any    `json:"alpha_overrides,omitempty"`
	SuccessCriteria map[string]any    `json:"success_criteria"`
	StartedAt       *time.Time         `json:"started_at,omitempty"`
	CompletedAt     *time.Time         `json:"completed_at,omitempty"`
	Conclusion      string             `json:"conclusion,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
}

type KnowledgeEntry struct {
	ID          uuid.UUID  `json:"id"`
	WorkflowID  *uuid.UUID `json:"workflow_id,omitempty"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	Tags        []string   `json:"tags"`
	Source      string     `json:"source"`
	CreatedAt   time.Time  `json:"created_at"`
}

type Signal struct {
	ID            uuid.UUID          `json:"id"`
	SignalType    string             `json:"signal_type"`
	Source        string             `json:"source"`
	Priority      int                `json:"priority"`
	Data          map[string]any    `json:"data"`
	Acknowledged  bool               `json:"acknowledged"`
	CreatedAt     time.Time          `json:"created_at"`
}

type WeightInput struct {
	ActorID       uuid.UUID      `json:"actor_id"`
	ActorType     string         `json:"actor_type"`
	TaskContext   map[string]any `json:"task_context,omitempty"`
	RequiredLevel string         `json:"required_level,omitempty"`
}

type OutcomeInput struct {
	ActorID        uuid.UUID      `json:"actor_id"`
	ActorType      string         `json:"actor_type"`
	OutcomeScore   float64        `json:"outcome_score"`
	TaskContext    map[string]any `json:"task_context,omitempty"`
}

type CreateExperimentInput struct {
	Name            string         `json:"name"`
	Hypothesis      string         `json:"hypothesis"`
	MVRUID          *uuid.UUID    `json:"mvru_id,omitempty"`
	AlphaOverrides  map[string]any `json:"alpha_overrides,omitempty"`
	SuccessCriteria map[string]any `json:"success_criteria,omitempty"`
}

type CreateKnowledgeInput struct {
	WorkflowID *uuid.UUID `json:"workflow_id,omitempty"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	Tags       []string   `json:"tags,omitempty"`
	Source     string     `json:"source"`
}

type CreateSignalInput struct {
	SignalType string         `json:"signal_type"`
	Source     string         `json:"source"`
	Priority   int            `json:"priority"`
	Data       map[string]any `json:"data,omitempty"`
}
```

- [ ] **Step 2: Create directory and commit**

```bash
mkdir -p /root/HarnessCompany/backend/internal/domain/evolution
git add backend/internal/domain/evolution/model.go
git commit -m "feat: evolution domain models with DecisionWeight, AlphaConfig, Experiment, KnowledgeEntry, Signal"
```

---

### Task 3: Evolution domain — repository

**Files:**
- Create: `backend/internal/domain/evolution/repository.go`

- [ ] **Step 1: Create repository.go**

```go
package evolution

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UpsertWeight(ctx context.Context, dw *DecisionWeight) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO weight_scores (actor_id, actor_type, overall_score, expertise_score, track_record_score, reliability_score, recency_score, context_fit_score, principle_score, decision_count, last_updated)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (actor_id, actor_type) DO UPDATE SET
		   overall_score = $3, expertise_score = $4, track_record_score = $5,
		   reliability_score = $6, recency_score = $7, context_fit_score = $8,
		   principle_score = $9, decision_count = $10, last_updated = $11`,
		dw.ActorID, dw.ActorType, dw.OverallScore, dw.ExpertiseScore, dw.TrackRecordScore,
		dw.ReliabilityScore, dw.RecencyScore, dw.ContextFitScore, dw.PrincipleScore,
		dw.DecisionCount, dw.LastUpdated)
	return err
}

func (r *Repository) GetWeight(ctx context.Context, actorID uuid.UUID, actorType string) (*DecisionWeight, error) {
	dw := &DecisionWeight{}
	err := r.db.QueryRow(ctx,
		`SELECT id, actor_id, actor_type, overall_score, expertise_score, track_record_score, reliability_score, recency_score, context_fit_score, principle_score, decision_count, last_updated
		 FROM weight_scores WHERE actor_id = $1 AND actor_type = $2`,
		actorID, actorType,
	).Scan(&dw.ID, &dw.ActorID, &dw.ActorType, &dw.OverallScore, &dw.ExpertiseScore, &dw.TrackRecordScore,
		&dw.ReliabilityScore, &dw.RecencyScore, &dw.ContextFitScore, &dw.PrincipleScore,
		&dw.DecisionCount, &dw.LastUpdated)
	if err != nil {
		return nil, fmt.Errorf("get weight: %w", err)
	}
	return dw, nil
}

func (r *Repository) ListWeights(ctx context.Context, limit int) ([]DecisionWeight, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
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
		var dw DecisionWeight
		if err := rows.Scan(&dw.ID, &dw.ActorID, &dw.ActorType, &dw.OverallScore, &dw.ExpertiseScore,
			&dw.TrackRecordScore, &dw.ReliabilityScore, &dw.RecencyScore, &dw.ContextFitScore,
			&dw.PrincipleScore, &dw.DecisionCount, &dw.LastUpdated); err != nil {
			return nil, fmt.Errorf("scan weight: %w", err)
		}
		weights = append(weights, dw)
	}
	return weights, rows.Err()
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
	_, err := r.db.Exec(ctx,
		`INSERT INTO weight_alphas (alpha_expertise, alpha_track_record, alpha_reliability, alpha_recency, alpha_context_fit, alpha_principle, version)
		 VALUES ($1, $2, $3, $4, $5, $6, (SELECT COALESCE(MAX(version), 0) + 1 FROM weight_alphas))`,
		a.Expertise, a.TrackRecord, a.Reliability, a.Recency, a.ContextFit, a.Principle)
	return err
}

func (r *Repository) CreateExperiment(ctx context.Context, input CreateExperimentInput) (*Experiment, error) {
	overrides, _ := json.Marshal(input.AlphaOverrides)
	criteria, _ := json.Marshal(input.SuccessCriteria)
	e := &Experiment{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO experiments (name, hypothesis, mvru_id, alpha_overrides, success_criteria)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, name, hypothesis, status, mvru_id, alpha_overrides, success_criteria, started_at, completed_at, conclusion, created_at`,
		input.Name, input.Hypothesis, input.MVRUID, overrides, criteria,
	).Scan(&e.ID, &e.Name, &e.Hypothesis, &e.Status, &e.MVRUID, &overrides, &criteria, &e.StartedAt, &e.CompletedAt, &e.Conclusion, &e.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create experiment: %w", err)
	}
	json.Unmarshal(overrides, &e.AlphaOverrides)
	json.Unmarshal(criteria, &e.SuccessCriteria)
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
		var overrides, criteria []byte
		if err := rows.Scan(&e.ID, &e.Name, &e.Hypothesis, &e.Status, &e.MVRUID, &overrides, &criteria, &e.StartedAt, &e.CompletedAt, &e.Conclusion, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan experiment: %w", err)
		}
		json.Unmarshal(overrides, &e.AlphaOverrides)
		json.Unmarshal(criteria, &e.SuccessCriteria)
		experiments = append(experiments, e)
	}
	return experiments, rows.Err()
}

func (r *Repository) UpdateExperimentStatus(ctx context.Context, id uuid.UUID, status string, conclusion string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE experiments SET status = $1, conclusion = $2, completed_at = CASE WHEN $3 THEN NOW() ELSE completed_at END
		 WHERE id = $4`,
		status, conclusion, status == "completed" || status == "rolled_back", id)
	return err
}

func (r *Repository) CreateKnowledge(ctx context.Context, input CreateKnowledgeInput) (*KnowledgeEntry, error) {
	k := &KnowledgeEntry{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO knowledge_entries (workflow_id, title, content, tags, source)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, workflow_id, title, content, tags, source, created_at`,
		input.WorkflowID, input.Title, input.Content, input.Tags, input.Source,
	).Scan(&k.ID, &k.WorkflowID, &k.Title, &k.Content, &k.Tags, &k.Source, &k.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create knowledge entry: %w", err)
	}
	return k, nil
}

func (r *Repository) ListKnowledge(ctx context.Context, limit int) ([]KnowledgeEntry, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
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
		var k KnowledgeEntry
		if err := rows.Scan(&k.ID, &k.WorkflowID, &k.Title, &k.Content, &k.Tags, &k.Source, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan knowledge: %w", err)
		}
		entries = append(entries, k)
	}
	return entries, rows.Err()
}

func (r *Repository) CreateSignal(ctx context.Context, input CreateSignalInput) (*Signal, error) {
	data, _ := json.Marshal(input.Data)
	s := &Signal{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO signals (signal_type, source, priority, data)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, signal_type, source, priority, data, acknowledged, created_at`,
		input.SignalType, input.Source, input.Priority, data,
	).Scan(&s.ID, &s.SignalType, &s.Source, &s.Priority, &data, &s.Acknowledged, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create signal: %w", err)
	}
	json.Unmarshal(data, &s.Data)
	return s, nil
}

func (r *Repository) ListSignals(ctx context.Context, acknowledged *bool, limit int) ([]Signal, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	query := `SELECT id, signal_type, source, priority, data, acknowledged, created_at
		FROM signals`
	var args []any
	argIdx := 1

	if acknowledged != nil {
		query += fmt.Sprintf(" WHERE acknowledged = $%d", argIdx)
		args = append(args, *acknowledged)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY priority DESC, created_at DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list signals: %w", err)
	}
	defer rows.Close()

	var signals []Signal
	for rows.Next() {
		var s Signal
		var data []byte
		if err := rows.Scan(&s.ID, &s.SignalType, &s.Source, &s.Priority, &data, &s.Acknowledged, &s.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan signal: %w", err)
		}
		json.Unmarshal(data, &s.Data)
		signals = append(signals, s)
	}
	return signals, rows.Err()
}

func (r *Repository) AcknowledgeSignal(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE signals SET acknowledged = true WHERE id = $1`, id)
	return err
}
```

Note: The `UpdateExperimentStatus` function uses `CASE WHEN $3 THEN NOW() ELSE completed_at END` which won't work with boolean params directly. Use this corrected version:

```go
func (r *Repository) UpdateExperimentStatus(ctx context.Context, id uuid.UUID, status string, conclusion string) error {
	var query string
	if status == "completed" || status == "rolled_back" {
		query = `UPDATE experiments SET status = $1, conclusion = $2, completed_at = NOW() WHERE id = $3`
	} else {
		query = `UPDATE experiments SET status = $1, conclusion = $2 WHERE id = $3`
	}
	_, err := r.db.Exec(ctx, query, status, conclusion, id)
	return err
}
```

- [ ] **Step 2: Build and commit**

```bash
cd /root/HarnessCompany/backend && go build ./internal/domain/evolution/
git add backend/internal/domain/evolution/repository.go
git commit -m "feat: evolution domain repository with weight, alpha, experiment, knowledge, signal CRUD"
```

---

### Task 4: Evolution domain — weight computation service

**Files:**
- Create: `backend/internal/domain/evolution/service.go`

- [ ] **Step 1: Create service.go**

The service implements the Decision Weight formula:

```go
package evolution

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("validation error")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ComputeWeight(ctx context.Context, input WeightInput) (*DecisionWeight, error) {
	// Load existing weight or create default
	dw, err := s.repo.GetWeight(ctx, input.ActorID, input.ActorType)
	if err != nil {
		dw = &DecisionWeight{
			ActorID:   input.ActorID,
			ActorType: input.ActorType,
			OverallScore: 1.0,
			ExpertiseScore:   0.5,
			TrackRecordScore: 0.5,
			ReliabilityScore: 0.5,
			RecencyScore:     1.0,
			ContextFitScore:  0.5,
			PrincipleScore:   0.5,
			DecisionCount:    0,
			LastUpdated:      time.Now(),
		}
	}

	// Load current alpha config
	alpha, err := s.repo.GetAlpha(ctx)
	if err != nil {
		alpha = defaultAlpha()
	}

	// Compute overall score using weighted formula
	overall := alpha.Expertise*dw.ExpertiseScore +
		alpha.TrackRecord*dw.TrackRecordScore +
		alpha.Reliability*dw.ReliabilityScore +
		alpha.Recency*dw.RecencyScore +
		alpha.ContextFit*dw.ContextFitScore +
		alpha.Principle*dw.PrincipleScore

	dw.OverallScore = math.Min(overall, 1.0)
	dw.LastUpdated = time.Now()

	if err := s.repo.UpsertWeight(ctx, dw); err != nil {
		return nil, fmt.Errorf("persist weight: %w", err)
	}

	return dw, nil
}

func (s *Service) RecordOutcome(ctx context.Context, input OutcomeInput) (*DecisionWeight, error) {
	// Get current weight
	dw, err := s.repo.GetWeight(ctx, input.ActorID, input.ActorType)
	if err != nil {
		return nil, fmt.Errorf("weight not found for actor: %w", err)
	}

	// Update decision count
	dw.DecisionCount++

	// Update track record with moving average
	n := float64(dw.DecisionCount)
	dw.TrackRecordScore = ((dw.TrackRecordScore * (n - 1)) + input.OutcomeScore) / n

	// Update recency score (reset to 1.0 on new decision)
	dw.RecencyScore = 1.0

	// Context fit smoothing
	if input.TaskContext != nil {
		dw.ContextFitScore = (dw.ContextFitScore + 0.5) / 2
	}

	// Recompute overall
	alpha, err := s.repo.GetAlpha(ctx)
	if err != nil {
		alpha = defaultAlpha()
	}

	overall := alpha.Expertise*dw.ExpertiseScore +
		alpha.TrackRecord*dw.TrackRecordScore +
		alpha.Reliability*dw.ReliabilityScore +
		alpha.Recency*dw.RecencyScore +
		alpha.ContextFit*dw.ContextFitScore +
		alpha.Principle*dw.PrincipleScore

	dw.OverallScore = math.Min(overall, 1.0)
	dw.LastUpdated = time.Now()

	if err := s.repo.UpsertWeight(ctx, dw); err != nil {
		return nil, fmt.Errorf("persist weight after outcome: %w", err)
	}

	return dw, nil
}

func (s *Service) GetWeight(ctx context.Context, actorID uuid.UUID, actorType string) (*DecisionWeight, error) {
	return s.repo.GetWeight(ctx, actorID, actorType)
}

func (s *Service) ListWeights(ctx context.Context, limit int) ([]DecisionWeight, error) {
	return s.repo.ListWeights(ctx, limit)
}

func (s *Service) GetAlpha(ctx context.Context) (*AlphaConfig, error) {
	return s.repo.GetAlpha(ctx)
}

func (s *Service) UpdateAlpha(ctx context.Context, a *AlphaConfig) error {
	// Validate sum of alphas is approximately 1.0
	sum := a.Expertise + a.TrackRecord + a.Reliability + a.Recency + a.ContextFit + a.Principle
	if sum < 0.95 || sum > 1.05 {
		return fmt.Errorf("%w: alpha values must sum to approximately 1.0 (got %f)", ErrValidation, sum)
	}
	return s.repo.UpdateAlpha(ctx, a)
}

func (s *Service) CreateExperiment(ctx context.Context, input CreateExperimentInput) (*Experiment, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	if input.Hypothesis == "" {
		return nil, fmt.Errorf("%w: hypothesis is required", ErrValidation)
	}
	return s.repo.CreateExperiment(ctx, input)
}

func (s *Service) ListExperiments(ctx context.Context) ([]Experiment, error) {
	return s.repo.ListExperiments(ctx)
}

func (s *Service) UpdateExperimentStatus(ctx context.Context, id uuid.UUID, status string, conclusion string) error {
	return s.repo.UpdateExperimentStatus(ctx, id, status, conclusion)
}

func (s *Service) CreateKnowledge(ctx context.Context, input CreateKnowledgeInput) (*KnowledgeEntry, error) {
	if input.Title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrValidation)
	}
	if input.Content == "" {
		return nil, fmt.Errorf("%w: content is required", ErrValidation)
	}
	if input.Source == "" {
		input.Source = "manual"
	}
	return s.repo.CreateKnowledge(ctx, input)
}

func (s *Service) ListKnowledge(ctx context.Context, limit int) ([]KnowledgeEntry, error) {
	return s.repo.ListKnowledge(ctx, limit)
}

func (s *Service) CreateSignal(ctx context.Context, input CreateSignalInput) (*Signal, error) {
	if input.SignalType == "" {
		return nil, fmt.Errorf("%w: signal_type is required", ErrValidation)
	}
	return s.repo.CreateSignal(ctx, input)
}

func (s *Service) ListSignals(ctx context.Context, acknowledged *bool, limit int) ([]Signal, error) {
	return s.repo.ListSignals(ctx, acknowledged, limit)
}

func (s *Service) AcknowledgeSignal(ctx context.Context, id uuid.UUID) error {
	return s.repo.AcknowledgeSignal(ctx, id)
}

func defaultAlpha() *AlphaConfig {
	return &AlphaConfig{
		Expertise:   0.25,
		TrackRecord: 0.20,
		Reliability: 0.15,
		Recency:     0.10,
		ContextFit:  0.10,
		Principle:   0.20,
		Version:     1,
	}
}
```

- [ ] **Step 2: Build and commit**

```bash
cd /root/HarnessCompany/backend && go build ./internal/domain/evolution/
git add backend/internal/domain/evolution/service.go
git commit -m "feat: evolution domain service with Decision Weight computation and meta-learning"
```

---

### Task 5: Evolution domain — handler

**Files:**
- Create: `backend/internal/domain/evolution/handler.go`

- [ ] **Step 1: Create handler.go**

```go
package evolution

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/evolution/weights/compute", h.computeWeight)
	r.Post("/evolution/weights/outcome", h.recordOutcome)
	r.Get("/evolution/weights/{actorType}/{actorID}", h.getWeight)
	r.Get("/evolution/weights", h.listWeights)
	r.Get("/evolution/alphas", h.getAlpha)
	r.Put("/evolution/alphas", h.updateAlpha)
	r.Post("/evolution/experiments", h.createExperiment)
	r.Get("/evolution/experiments", h.listExperiments)
	r.Patch("/evolution/experiments/{id}", h.updateExperimentStatus)
	r.Post("/evolution/knowledge", h.createKnowledge)
	r.Get("/evolution/knowledge", h.listKnowledge)
	r.Post("/evolution/signals", h.createSignal)
	r.Get("/evolution/signals", h.listSignals)
	r.Post("/evolution/signals/{id}/acknowledge", h.acknowledgeSignal)
}

func (h *Handler) computeWeight(w http.ResponseWriter, r *http.Request) {
	var input WeightInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	dw, err := h.service.ComputeWeight(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, dw)
}

func (h *Handler) recordOutcome(w http.ResponseWriter, r *http.Request) {
	var input OutcomeInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	dw, err := h.service.RecordOutcome(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, dw)
}

func (h *Handler) getWeight(w http.ResponseWriter, r *http.Request) {
	actorType := chi.URLParam(r, "actorType")
	actorID, err := uuid.Parse(chi.URLParam(r, "actorID"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid actor id"})
		return
	}
	dw, err := h.service.GetWeight(r.Context(), actorID, actorType)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "weight not found"})
		return
	}
	writeJSON(w, http.StatusOK, dw)
}

func (h *Handler) listWeights(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	weights, err := h.service.ListWeights(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, weights)
}

func (h *Handler) getAlpha(w http.ResponseWriter, r *http.Request) {
	alpha, err := h.service.GetAlpha(r.Context())
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "alpha config not found"})
		return
	}
	writeJSON(w, http.StatusOK, alpha)
}

func (h *Handler) updateAlpha(w http.ResponseWriter, r *http.Request) {
	var a AlphaConfig
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&a); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if err := h.service.UpdateAlpha(r.Context(), &a); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "alpha updated"})
}

func (h *Handler) createExperiment(w http.ResponseWriter, r *http.Request) {
	var input CreateExperimentInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	e, err := h.service.CreateExperiment(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (h *Handler) listExperiments(w http.ResponseWriter, r *http.Request) {
	experiments, err := h.service.ListExperiments(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, experiments)
}

func (h *Handler) updateExperimentStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid experiment id"})
		return
	}
	var req struct {
		Status     string `json:"status"`
		Conclusion string `json:"conclusion,omitempty"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if err := h.service.UpdateExperimentStatus(r.Context(), id, req.Status, req.Conclusion); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "experiment updated"})
}

func (h *Handler) createKnowledge(w http.ResponseWriter, r *http.Request) {
	var input CreateKnowledgeInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	k, err := h.service.CreateKnowledge(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, k)
}

func (h *Handler) listKnowledge(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	entries, err := h.service.ListKnowledge(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (h *Handler) createSignal(w http.ResponseWriter, r *http.Request) {
	var input CreateSignalInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	s, err := h.service.CreateSignal(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, s)
}

func (h *Handler) listSignals(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	var acknowledged *bool
	if ackStr := r.URL.Query().Get("acknowledged"); ackStr != "" {
		b := ackStr == "true"
		acknowledged = &b
	}
	signals, err := h.service.ListSignals(r.Context(), acknowledged, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, signals)
}

func (h *Handler) acknowledgeSignal(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid signal id"})
		return
	}
	if err := h.service.AcknowledgeSignal(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "signal acknowledged"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON error: %v", err)
	}
}
```

- [ ] **Step 2: Build and commit**

```bash
cd /root/HarnessCompany/backend && go build ./internal/domain/evolution/
git add backend/internal/domain/evolution/handler.go
git commit -m "feat: evolution domain handler with 14 API endpoints for weight engine"
```

---

### Task 6: Wire Evolution into router + main.go + build

**Files:**
- Modify: `backend/internal/gateway/router.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Update router.go imports and Dependencies**

Add to imports:
```go
	"github.com/harness-org/backend/internal/domain/evolution"
```

Add to Dependencies struct:
```go
	EvolutionHandler *evolution.Handler
```

Add to RegisterRoutes body (before closing `}`):
```go
		if deps.EvolutionHandler != nil {
			deps.EvolutionHandler.RegisterRoutes(r)
		}
```

- [ ] **Step 2: Update main.go**

Add to imports:
```go
	"github.com/harness-org/backend/internal/domain/evolution"
```

Add before `router := server.NewRouter(...)`:
```go
	evoRepo := evolution.NewRepository(db)
	evoSvc := evolution.NewService(evoRepo)
	evoHandler := evolution.NewHandler(evoSvc)
```

Add to Dependencies literal:
```go
		EvolutionHandler: evoHandler,
```

- [ ] **Step 3: Build and vet**

```bash
cd /root/HarnessCompany/backend && go build ./cmd/server/
go vet ./internal/domain/evolution/... ./internal/gateway/... ./cmd/server/...
```

- [ ] **Step 4: Commit**

```bash
git add backend/internal/gateway/router.go backend/cmd/server/main.go
git commit -m "feat: wire evolution domain handler into router and main"
```

---

## Self-Review Checklist

- [ ] **Spec coverage:** Covers all 4 Evolution subsystems: Decision Weight Engine (tasks 3-4), Sensing Engine (signals in tasks 3,5), Learning Engine (experiments in tasks 3,5), Knowledge Engine (tasks 3,5)
- [ ] **Weight formula implemented:** Σ(αᵢ × scoreᵢ) with 6 dimensions, alpha validation, outcome-based track record update
- [ ] **No placeholders:** All files contain complete, compilable Go code
- [ ] **Type consistency:** All method signatures match across model→repository→service→handler chains
- [ ] **14 new API endpoints:** compute/outcome/get/list weights, get/update alphas, CRUD experiments, CRUD knowledge, CRUD signals
- [ ] **All 9 domains now wired:** Identity, Organization, Layer, Capability, Workflow, Observability, Verification, Governance, Evolution
