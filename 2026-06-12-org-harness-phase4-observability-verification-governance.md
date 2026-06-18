# Phase 4: Observability + Verification + Governance Domains Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build three interconnected backend domains — Observability (tracing & metrics), Verification (3D review), and Governance (permission & principle system) — completing all remaining core domains before Evolution (Phase 5).

**Architecture:** Each domain follows the existing 4-file pattern (model.go, repository.go, service.go, handler.go) with a migration SQL file, matching capability/workflow patterns. Services encapsulate business logic; handlers expose REST APIs under `/api/v1/`; repositories use pgxpool with JSONB for flexible data.

**Tech Stack:** Go (chi router, pgxpool, google/uuid), PostgreSQL (JSONB, UUID, TIMESTAMPTZ), following existing identity/organization/layer/capability/workflow patterns.

**Dependencies:**
- Task 1 (migrations) must come first (tables must exist before code)
- Tasks 2-4 (Observability), 5-7 (Verification), 8-10 (Governance) are independent and can be parallelized
- Task 11 (wiring) must come last

---

### Task 1: Create migrations 007 (observability), 008 (verification), 009 (governance)

**Files:**
- Create: `migrations/007_observability.sql`
- Create: `migrations/008_verification.sql`
- Create: `migrations/009_governance.sql`

- [ ] **Step 1: Create 007_observability.sql**

```sql
-- 007_observability.sql

CREATE TABLE traces (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id     UUID REFERENCES workflow_instances(id) ON DELETE SET NULL,
    status          TEXT NOT NULL DEFAULT 'active',
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    metadata        JSONB NOT NULL DEFAULT '{}'
);

CREATE TABLE spans (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    trace_id        UUID NOT NULL REFERENCES traces(id) ON DELETE CASCADE,
    parent_span_id  UUID REFERENCES spans(id) ON DELETE SET NULL,
    span_type       TEXT NOT NULL,
    entity_id       UUID,
    entity_type     TEXT,
    actor_id        UUID,
    actor_type      TEXT,
    input           JSONB,
    output          JSONB,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ,
    duration_ms     INT,
    metadata        JSONB NOT NULL DEFAULT '{}'
);

CREATE TABLE metrics (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_type     TEXT NOT NULL,
    metric_name     TEXT NOT NULL,
    entity_id       UUID,
    entity_type     TEXT,
    value           DOUBLE PRECISION NOT NULL,
    recorded_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata        JSONB NOT NULL DEFAULT '{}'
);

CREATE INDEX idx_spans_trace_id ON spans(trace_id);
CREATE INDEX idx_spans_actor ON spans(actor_id);
CREATE INDEX idx_spans_type ON spans(span_type);
CREATE INDEX idx_metrics_type ON metrics(metric_type, metric_name);
CREATE INDEX idx_metrics_recorded_at ON metrics(recorded_at);
```

- [ ] **Step 2: Create 008_verification.sql**

```sql
-- 008_verification.sql

CREATE TABLE verification_reports (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id         UUID REFERENCES workflow_instances(id) ON DELETE SET NULL,
    task_id             UUID REFERENCES tasks(id) ON DELETE SET NULL,
    result_score        DOUBLE PRECISION,
    path_score          DOUBLE PRECISION,
    environment_score   DOUBLE PRECISION,
    overall_score       DOUBLE PRECISION,
    conclusion          TEXT NOT NULL DEFAULT '',
    suggestions         JSONB NOT NULL DEFAULT '[]',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE review_assignments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id       UUID NOT NULL REFERENCES verification_reports(id) ON DELETE CASCADE,
    level           TEXT NOT NULL,
    reviewer_id     UUID,
    reviewer_type   TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    result          JSONB,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_verif_workflow ON verification_reports(workflow_id);
CREATE INDEX idx_verif_task ON verification_reports(task_id);
CREATE INDEX idx_review_report ON review_assignments(report_id);
CREATE INDEX idx_review_reviewer ON review_assignments(reviewer_id);
```

- [ ] **Step 3: Create 009_governance.sql**

```sql
-- 009_governance.sql

CREATE TABLE permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    level       INT NOT NULL CHECK (level BETWEEN 1 AND 4),
    name        TEXT NOT NULL UNIQUE,
    description TEXT,
    behavior    TEXT NOT NULL DEFAULT 'notify',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE principles (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name             TEXT NOT NULL UNIQUE,
    description      TEXT NOT NULL,
    evaluation_logic JSONB NOT NULL DEFAULT '{}',
    priority         INT NOT NULL DEFAULT 0,
    is_active        BOOLEAN NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE control_rules (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    principle_id        UUID REFERENCES principles(id) ON DELETE CASCADE,
    target_entity_type  TEXT NOT NULL,
    target_entity_id    UUID,
    condition           JSONB NOT NULL DEFAULT '{}',
    action              TEXT NOT NULL,
    priority            INT NOT NULL DEFAULT 0,
    is_active           BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_principles_active ON principles(is_active);
CREATE INDEX idx_control_principle ON control_rules(principle_id);
CREATE INDEX idx_control_target ON control_rules(target_entity_type, target_entity_id);
```

- [ ] **Step 4: Commit**

```bash
git add migrations/007_observability.sql migrations/008_verification.sql migrations/009_governance.sql
git commit -m "feat: add observability, verification, governance migrations"
```

---

### Task 2: Observability domain — models

**Files:**
- Create: `backend/internal/domain/observability/model.go`

- [ ] **Step 1: Create model.go**

```go
package observability

import (
	"time"

	"github.com/google/uuid"
)

type Trace struct {
	ID          uuid.UUID          `json:"id"`
	WorkflowID  *uuid.UUID         `json:"workflow_id,omitempty"`
	Status      string             `json:"status"`
	StartedAt   time.Time          `json:"started_at"`
	CompletedAt *time.Time         `json:"completed_at,omitempty"`
	Metadata    map[string]any    `json:"metadata"`
	Spans       []Span             `json:"spans,omitempty"`
}

type Span struct {
	ID           uuid.UUID          `json:"id"`
	TraceID      uuid.UUID          `json:"trace_id"`
	ParentSpanID *uuid.UUID         `json:"parent_span_id,omitempty"`
	SpanType     string             `json:"span_type"`
	EntityID     *uuid.UUID         `json:"entity_id,omitempty"`
	EntityType   string             `json:"entity_type,omitempty"`
	ActorID      *uuid.UUID         `json:"actor_id,omitempty"`
	ActorType    string             `json:"actor_type,omitempty"`
	Input        map[string]any    `json:"input,omitempty"`
	Output       map[string]any    `json:"output,omitempty"`
	StartedAt    time.Time          `json:"started_at"`
	CompletedAt  *time.Time         `json:"completed_at,omitempty"`
	DurationMs   int                `json:"duration_ms,omitempty"`
	Metadata     map[string]any    `json:"metadata"`
}

type Metric struct {
	ID         uuid.UUID          `json:"id"`
	MetricType string             `json:"metric_type"`
	MetricName string             `json:"metric_name"`
	EntityID   *uuid.UUID         `json:"entity_id,omitempty"`
	EntityType string             `json:"entity_type,omitempty"`
	Value      float64            `json:"value"`
	RecordedAt time.Time          `json:"recorded_at"`
	Metadata   map[string]any    `json:"metadata"`
}

type CreateTraceInput struct {
	WorkflowID *uuid.UUID      `json:"workflow_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type RecordSpanInput struct {
	TraceID      uuid.UUID          `json:"trace_id"`
	ParentSpanID *uuid.UUID         `json:"parent_span_id,omitempty"`
	SpanType     string             `json:"span_type"`
	EntityID     *uuid.UUID         `json:"entity_id,omitempty"`
	EntityType   string             `json:"entity_type,omitempty"`
	ActorID      *uuid.UUID         `json:"actor_id,omitempty"`
	ActorType    string             `json:"actor_type,omitempty"`
	Input        map[string]any    `json:"input,omitempty"`
	Output       map[string]any    `json:"output,omitempty"`
	DurationMs   int                `json:"duration_ms,omitempty"`
	Metadata     map[string]any    `json:"metadata,omitempty"`
}

type RecordMetricInput struct {
	MetricType string         `json:"metric_type"`
	MetricName string         `json:"metric_name"`
	EntityID   *uuid.UUID    `json:"entity_id,omitempty"`
	EntityType string         `json:"entity_type,omitempty"`
	Value      float64        `json:"value"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type MetricsQuery struct {
	MetricType string     `json:"metric_type,omitempty"`
	MetricName string     `json:"metric_name,omitempty"`
	EntityID   *uuid.UUID `json:"entity_id,omitempty"`
	From       *time.Time `json:"from,omitempty"`
	To         *time.Time `json:"to,omitempty"`
	Limit      int        `json:"limit,omitempty"`
}
```

- [ ] **Step 2: Create the directory**

```bash
mkdir -p /root/HarnessCompany/backend/internal/domain/observability
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/domain/observability/model.go
git commit -m "feat: observability domain models"
```

---

### Task 3: Observability domain — repository

**Files:**
- Create: `backend/internal/domain/observability/repository.go`

- [ ] **Step 1: Create repository.go**

```go
package observability

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

func (r *Repository) CreateTrace(ctx context.Context, input CreateTraceInput) (*Trace, error) {
	meta, _ := json.Marshal(input.Metadata)
	t := &Trace{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO traces (workflow_id, metadata) VALUES ($1, $2)
		 RETURNING id, workflow_id, status, started_at, completed_at, metadata`,
		input.WorkflowID, meta,
	).Scan(&t.ID, &t.WorkflowID, &t.Status, &t.StartedAt, &t.CompletedAt, &meta)
	if err != nil {
		return nil, fmt.Errorf("create trace: %w", err)
	}
	json.Unmarshal(meta, &t.Metadata)
	return t, nil
}

func (r *Repository) GetTrace(ctx context.Context, id uuid.UUID) (*Trace, error) {
	t := &Trace{}
	var meta []byte
	err := r.db.QueryRow(ctx,
		`SELECT id, workflow_id, status, started_at, completed_at, metadata
		 FROM traces WHERE id = $1`, id,
	).Scan(&t.ID, &t.WorkflowID, &t.Status, &t.StartedAt, &t.CompletedAt, &meta)
	if err != nil {
		return nil, fmt.Errorf("get trace: %w", err)
	}
	json.Unmarshal(meta, &t.Metadata)
	return t, nil
}

func (r *Repository) ListTraces(ctx context.Context, limit int) ([]Trace, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.db.Query(ctx,
		`SELECT id, workflow_id, status, started_at, completed_at, metadata
		 FROM traces ORDER BY started_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list traces: %w", err)
	}
	defer rows.Close()

	var traces []Trace
	for rows.Next() {
		var t Trace
		var meta []byte
		if err := rows.Scan(&t.ID, &t.WorkflowID, &t.Status, &t.StartedAt, &t.CompletedAt, &meta); err != nil {
			return nil, fmt.Errorf("scan trace: %w", err)
		}
		json.Unmarshal(meta, &t.Metadata)
		traces = append(traces, t)
	}
	return traces, rows.Err()
}

func (r *Repository) RecordSpan(ctx context.Context, input RecordSpanInput) (*Span, error) {
	inJSON, _ := json.Marshal(input.Input)
	outJSON, _ := json.Marshal(input.Output)
	metaJSON, _ := json.Marshal(input.Metadata)
	s := &Span{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO spans (trace_id, parent_span_id, span_type, entity_id, entity_type, actor_id, actor_type, input, output, duration_ms, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id, trace_id, parent_span_id, span_type, entity_id, entity_type, actor_id, actor_type, input, output, started_at, completed_at, duration_ms, metadata`,
		input.TraceID, input.ParentSpanID, input.SpanType, input.EntityID, input.EntityType, input.ActorID, input.ActorType, inJSON, outJSON, input.DurationMs, metaJSON,
	).Scan(&s.ID, &s.TraceID, &s.ParentSpanID, &s.SpanType, &s.EntityID, &s.EntityType, &s.ActorID, &s.ActorType, &inJSON, &outJSON, &s.StartedAt, &s.CompletedAt, &s.DurationMs, &metaJSON)
	if err != nil {
		return nil, fmt.Errorf("record span: %w", err)
	}
	json.Unmarshal(inJSON, &s.Input)
	json.Unmarshal(outJSON, &s.Output)
	json.Unmarshal(metaJSON, &s.Metadata)
	return s, nil
}

func (r *Repository) GetSpansByTrace(ctx context.Context, traceID uuid.UUID) ([]Span, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, trace_id, parent_span_id, span_type, entity_id, entity_type, actor_id, actor_type, input, output, started_at, completed_at, duration_ms, metadata
		 FROM spans WHERE trace_id = $1 ORDER BY started_at`, traceID)
	if err != nil {
		return nil, fmt.Errorf("get spans by trace: %w", err)
	}
	defer rows.Close()

	var spans []Span
	for rows.Next() {
		var s Span
		var inJSON, outJSON, metaJSON []byte
		if err := rows.Scan(&s.ID, &s.TraceID, &s.ParentSpanID, &s.SpanType, &s.EntityID, &s.EntityType, &s.ActorID, &s.ActorType, &inJSON, &outJSON, &s.StartedAt, &s.CompletedAt, &s.DurationMs, &metaJSON); err != nil {
			return nil, fmt.Errorf("scan span: %w", err)
		}
		json.Unmarshal(inJSON, &s.Input)
		json.Unmarshal(outJSON, &s.Output)
		json.Unmarshal(metaJSON, &s.Metadata)
		spans = append(spans, s)
	}
	return spans, rows.Err()
}

func (r *Repository) RecordMetric(ctx context.Context, input RecordMetricInput) (*Metric, error) {
	meta, _ := json.Marshal(input.Metadata)
	m := &Metric{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO metrics (metric_type, metric_name, entity_id, entity_type, value, metadata)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, metric_type, metric_name, entity_id, entity_type, value, recorded_at, metadata`,
		input.MetricType, input.MetricName, input.EntityID, input.EntityType, input.Value, meta,
	).Scan(&m.ID, &m.MetricType, &m.MetricName, &m.EntityID, &m.EntityType, &m.Value, &m.RecordedAt, &meta)
	if err != nil {
		return nil, fmt.Errorf("record metric: %w", err)
	}
	json.Unmarshal(meta, &m.Metadata)
	return m, nil
}

func (r *Repository) QueryMetrics(ctx context.Context, q MetricsQuery) ([]Metric, error) {
	query := `SELECT id, metric_type, metric_name, entity_id, entity_type, value, recorded_at, metadata
		FROM metrics WHERE 1=1`
	args := []any{}
	argIdx := 1

	if q.MetricType != "" {
		query += fmt.Sprintf(" AND metric_type = $%d", argIdx)
		args = append(args, q.MetricType)
		argIdx++
	}
	if q.MetricName != "" {
		query += fmt.Sprintf(" AND metric_name = $%d", argIdx)
		args = append(args, q.MetricName)
		argIdx++
	}
	if q.EntityID != nil {
		query += fmt.Sprintf(" AND entity_id = $%d", argIdx)
		args = append(args, *q.EntityID)
		argIdx++
	}
	if q.From != nil {
		query += fmt.Sprintf(" AND recorded_at >= $%d", argIdx)
		args = append(args, *q.From)
		argIdx++
	}
	if q.To != nil {
		query += fmt.Sprintf(" AND recorded_at <= $%d", argIdx)
		args = append(args, *q.To)
		argIdx++
	}

	limit := 50
	if q.Limit > 0 && q.Limit <= 500 {
		limit = q.Limit
	}
	query += fmt.Sprintf(" ORDER BY recorded_at DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query metrics: %w", err)
	}
	defer rows.Close()

	var metrics []Metric
	for rows.Next() {
		var m Metric
		var meta []byte
		if err := rows.Scan(&m.ID, &m.MetricType, &m.MetricName, &m.EntityID, &m.EntityType, &m.Value, &m.RecordedAt, &meta); err != nil {
			return nil, fmt.Errorf("scan metric: %w", err)
		}
		json.Unmarshal(meta, &m.Metadata)
		metrics = append(metrics, m)
	}
	return metrics, rows.Err()
}

func (r *Repository) CompleteTrace(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE traces SET status = $1, completed_at = NOW() WHERE id = $2`,
		status, id)
	return err
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/domain/observability/repository.go
git commit -m "feat: observability domain repository with trace, span, metric CRUD"
```

---

### Task 4: Observability domain — service + handler

**Files:**
- Create: `backend/internal/domain/observability/service.go`
- Create: `backend/internal/domain/observability/handler.go`

- [ ] **Step 1: Create service.go**

```go
package observability

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) StartTrace(ctx context.Context, workflowID *uuid.UUID, metadata map[string]any) (*Trace, error) {
	return s.repo.CreateTrace(ctx, CreateTraceInput{
		WorkflowID: workflowID,
		Metadata:   metadata,
	})
}

func (s *Service) GetTrace(ctx context.Context, id uuid.UUID) (*Trace, error) {
	t, err := s.repo.GetTrace(ctx, id)
	if err != nil {
		return nil, err
	}
	spans, err := s.repo.GetSpansByTrace(ctx, id)
	if err != nil {
		return nil, err
	}
	t.Spans = spans
	return t, nil
}

func (s *Service) ListTraces(ctx context.Context, limit int) ([]Trace, error) {
	return s.repo.ListTraces(ctx, limit)
}

func (s *Service) RecordSpan(ctx context.Context, input RecordSpanInput) (*Span, error) {
	return s.repo.RecordSpan(ctx, input)
}

func (s *Service) RecordMetric(ctx context.Context, input RecordMetricInput) (*Metric, error) {
	return s.repo.RecordMetric(ctx, input)
}

func (s *Service) QueryMetrics(ctx context.Context, q MetricsQuery) ([]Metric, error) {
	return s.repo.QueryMetrics(ctx, q)
}

func (s *Service) CompleteTrace(ctx context.Context, id uuid.UUID, status string) error {
	return s.repo.CompleteTrace(ctx, id, status)
}
```

- [ ] **Step 2: Create handler.go**

```go
package observability

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
	r.Post("/traces", h.startTrace)
	r.Get("/traces", h.listTraces)
	r.Get("/traces/{id}", h.getTrace)
	r.Post("/traces/{id}/spans", h.recordSpan)
	r.Post("/traces/{id}/complete", h.completeTrace)
	r.Post("/metrics", h.recordMetric)
	r.Get("/metrics", h.queryMetrics)
}

func (h *Handler) startTrace(w http.ResponseWriter, r *http.Request) {
	var req struct {
		WorkflowID *string         `json:"workflow_id,omitempty"`
		Metadata   map[string]any `json:"metadata,omitempty"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	var wfID *uuid.UUID
	if req.WorkflowID != nil {
		parsed, err := uuid.Parse(*req.WorkflowID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid workflow_id"})
			return
		}
		wfID = &parsed
	}
	t, err := h.service.StartTrace(r.Context(), wfID, req.Metadata)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handler) listTraces(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	traces, err := h.service.ListTraces(r.Context(), limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, traces)
}

func (h *Handler) getTrace(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	t, err := h.service.GetTrace(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "trace not found"})
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) recordSpan(w http.ResponseWriter, r *http.Request) {
	traceID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid trace id"})
		return
	}
	var input RecordSpanInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	input.TraceID = traceID
	s, err := h.service.RecordSpan(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, s)
}

func (h *Handler) completeTrace(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if req.Status == "" {
		req.Status = "completed"
	}
	if err := h.service.CompleteTrace(r.Context(), id, req.Status); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "trace completed"})
}

func (h *Handler) recordMetric(w http.ResponseWriter, r *http.Request) {
	var input RecordMetricInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	m, err := h.service.RecordMetric(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

func (h *Handler) queryMetrics(w http.ResponseWriter, r *http.Request) {
	q := MetricsQuery{
		MetricType: r.URL.Query().Get("metric_type"),
		MetricName: r.URL.Query().Get("metric_name"),
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		q.Limit, _ = strconv.Atoi(limitStr)
	}
	metrics, err := h.service.QueryMetrics(r.Context(), q)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, metrics)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON error: %v", err)
	}
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/domain/observability/service.go backend/internal/domain/observability/handler.go
git commit -m "feat: observability domain service and handler with 7 API endpoints"
```

---

### Task 5: Verification domain — models

**Files:**
- Create: `backend/internal/domain/verification/model.go`
- Create directory: `backend/internal/domain/verification/`

- [ ] **Step 1: Create model.go**

```go
package verification

import (
	"time"

	"github.com/google/uuid"
)

type VerificationReport struct {
	ID               uuid.UUID          `json:"id"`
	WorkflowID       *uuid.UUID         `json:"workflow_id,omitempty"`
	TaskID           *uuid.UUID         `json:"task_id,omitempty"`
	ResultScore      *float64           `json:"result_score,omitempty"`
	PathScore        *float64           `json:"path_score,omitempty"`
	EnvironmentScore *float64           `json:"environment_score,omitempty"`
	OverallScore     *float64           `json:"overall_score,omitempty"`
	Conclusion       string             `json:"conclusion"`
	Suggestions      []string           `json:"suggestions"`
	CreatedAt        time.Time          `json:"created_at"`
	Reviews          []ReviewAssignment `json:"reviews,omitempty"`
}

type ReviewAssignment struct {
	ID           uuid.UUID  `json:"id"`
	ReportID     uuid.UUID  `json:"report_id"`
	Level        string     `json:"level"`
	ReviewerID   *uuid.UUID `json:"reviewer_id,omitempty"`
	ReviewerType string     `json:"reviewer_type"`
	Status       string     `json:"status"`
	Result       map[string]any `json:"result,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type CreateReportInput struct {
	WorkflowID       *uuid.UUID      `json:"workflow_id,omitempty"`
	TaskID           *uuid.UUID      `json:"task_id,omitempty"`
	ResultScore      *float64        `json:"result_score,omitempty"`
	PathScore        *float64        `json:"path_score,omitempty"`
	EnvironmentScore *float64        `json:"environment_score,omitempty"`
	Conclusion       string          `json:"conclusion"`
	Suggestions      []string        `json:"suggestions,omitempty"`
}

type AssignReviewInput struct {
	ReportID     uuid.UUID  `json:"report_id"`
	Level        string     `json:"level"`
	ReviewerID   *uuid.UUID `json:"reviewer_id,omitempty"`
	ReviewerType string     `json:"reviewer_type"`
}

type CompleteReviewInput struct {
	Result map[string]any `json:"result"`
}
```

- [ ] **Step 2: Commit**

```bash
mkdir -p /root/HarnessCompany/backend/internal/domain/verification
git add backend/internal/domain/verification/model.go
git commit -m "feat: verification domain models"
```

---

### Task 6: Verification domain — repository

**Files:**
- Create: `backend/internal/domain/verification/repository.go`

- [ ] **Step 1: Create repository.go**

```go
package verification

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

func (r *Repository) CreateReport(ctx context.Context, input CreateReportInput) (*VerificationReport, error) {
	suggestions, _ := json.Marshal(input.Suggestions)
	rep := &VerificationReport{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO verification_reports (workflow_id, task_id, result_score, path_score, environment_score, conclusion, suggestions)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)
		 RETURNING id, workflow_id, task_id, result_score, path_score, environment_score, overall_score, conclusion, suggestions, created_at`,
		input.WorkflowID, input.TaskID, input.ResultScore, input.PathScore, input.EnvironmentScore, input.Conclusion, suggestions,
	).Scan(&rep.ID, &rep.WorkflowID, &rep.TaskID, &rep.ResultScore, &rep.PathScore, &rep.EnvironmentScore, &rep.OverallScore, &rep.Conclusion, &suggestions, &rep.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create verification report: %w", err)
	}
	json.Unmarshal(suggestions, &rep.Suggestions)
	return rep, nil
}

func (r *Repository) GetReport(ctx context.Context, id uuid.UUID) (*VerificationReport, error) {
	rep := &VerificationReport{}
	var suggestions []byte
	err := r.db.QueryRow(ctx,
		`SELECT id, workflow_id, task_id, result_score, path_score, environment_score, overall_score, conclusion, suggestions, created_at
		 FROM verification_reports WHERE id = $1`, id,
	).Scan(&rep.ID, &rep.WorkflowID, &rep.TaskID, &rep.ResultScore, &rep.PathScore, &rep.EnvironmentScore, &rep.OverallScore, &rep.Conclusion, &suggestions, &rep.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get verification report: %w", err)
	}
	json.Unmarshal(suggestions, &rep.Suggestions)
	return rep, nil
}

func (r *Repository) ListReports(ctx context.Context, workflowID *uuid.UUID, limit int) ([]VerificationReport, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var rows interface{ Close(); Next() bool; Err() error; Scan(...interface{}) error }
	var err error
	if workflowID != nil {
		rrows, rerr := r.db.Query(ctx,
			`SELECT id, workflow_id, task_id, result_score, path_score, environment_score, overall_score, conclusion, suggestions, created_at
			 FROM verification_reports WHERE workflow_id = $1 ORDER BY created_at DESC LIMIT $2`, *workflowID, limit)
		if rerr != nil {
			return nil, fmt.Errorf("list reports by workflow: %w", rerr)
		}
		rows = rrows
	} else {
		rrows, rerr := r.db.Query(ctx,
			`SELECT id, workflow_id, task_id, result_score, path_score, environment_score, overall_score, conclusion, suggestions, created_at
			 FROM verification_reports ORDER BY created_at DESC LIMIT $1`, limit)
		if rerr != nil {
			return nil, fmt.Errorf("list reports: %w", rerr)
		}
		rows = rrows
	}
	defer rows.Close()

	var reports []VerificationReport
	for rows.Next() {
		var rep VerificationReport
		var suggestions []byte
		if err := rows.Scan(&rep.ID, &rep.WorkflowID, &rep.TaskID, &rep.ResultScore, &rep.PathScore, &rep.EnvironmentScore, &rep.OverallScore, &rep.Conclusion, &suggestions, &rep.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}
		json.Unmarshal(suggestions, &rep.Suggestions)
		reports = append(reports, rep)
	}
	return reports, rows.Err()
}

// Helper to avoid interface{} pattern issue
func (r *Repository) listReportsByWorkflow(ctx context.Context, workflowID uuid.UUID, limit int) ([]VerificationReport, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, workflow_id, task_id, result_score, path_score, environment_score, overall_score, conclusion, suggestions, created_at
		 FROM verification_reports WHERE workflow_id = $1 ORDER BY created_at DESC LIMIT $2`, workflowID, limit)
	if err != nil {
		return nil, fmt.Errorf("list reports by workflow: %w", err)
	}
	defer rows.Close()

	var reports []VerificationReport
	for rows.Next() {
		var rep VerificationReport
		var suggestions []byte
		if err := rows.Scan(&rep.ID, &rep.WorkflowID, &rep.TaskID, &rep.ResultScore, &rep.PathScore, &rep.EnvironmentScore, &rep.OverallScore, &rep.Conclusion, &suggestions, &rep.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}
		json.Unmarshal(suggestions, &rep.Suggestions)
		reports = append(reports, rep)
	}
	return reports, rows.Err()
}

func (r *Repository) AssignReview(ctx context.Context, input AssignReviewInput) (*ReviewAssignment, error) {
	rev := &ReviewAssignment{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO review_assignments (report_id, level, reviewer_id, reviewer_type)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, report_id, level, reviewer_id, reviewer_type, status, result, completed_at, created_at`,
		input.ReportID, input.Level, input.ReviewerID, input.ReviewerType,
	).Scan(&rev.ID, &rev.ReportID, &rev.Level, &rev.ReviewerID, &rev.ReviewerType, &rev.Status, &rev.Result, &rev.CompletedAt, &rev.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("assign review: %w", err)
	}
	return rev, nil
}

func (r *Repository) CompleteReview(ctx context.Context, reviewID uuid.UUID, result map[string]any) error {
	resultJSON, _ := json.Marshal(result)
	_, err := r.db.Exec(ctx,
		`UPDATE review_assignments SET status = 'completed', result = $1, completed_at = NOW() WHERE id = $2`,
		resultJSON, reviewID)
	return err
}

func (r *Repository) GetReviewsByReport(ctx context.Context, reportID uuid.UUID) ([]ReviewAssignment, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, report_id, level, reviewer_id, reviewer_type, status, result, completed_at, created_at
		 FROM review_assignments WHERE report_id = $1 ORDER BY created_at`, reportID)
	if err != nil {
		return nil, fmt.Errorf("get reviews by report: %w", err)
	}
	defer rows.Close()

	var reviews []ReviewAssignment
	for rows.Next() {
		var rev ReviewAssignment
		if err := rows.Scan(&rev.ID, &rev.ReportID, &rev.Level, &rev.ReviewerID, &rev.ReviewerType, &rev.Status, &rev.Result, &rev.CompletedAt, &rev.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan review: %w", err)
		}
		reviews = append(reviews, rev)
	}
	return reviews, rows.Err()
}

func (r *Repository) UpdateOverallScore(ctx context.Context, reportID uuid.UUID, score float64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE verification_reports SET overall_score = $1 WHERE id = $2`,
		score, reportID)
	return err
}
```

Note: The `ListReports` method above has interface{} pattern issue. Use this cleaner implementation instead:

```go
func (r *Repository) ListReports(ctx context.Context, workflowID *uuid.UUID, limit int) ([]VerificationReport, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var query string
	var args []any

	if workflowID != nil {
		query = `SELECT id, workflow_id, task_id, result_score, path_score, environment_score, overall_score, conclusion, suggestions, created_at
				 FROM verification_reports WHERE workflow_id = $1 ORDER BY created_at DESC LIMIT $2`
		args = []any{*workflowID, limit}
	} else {
		query = `SELECT id, workflow_id, task_id, result_score, path_score, environment_score, overall_score, conclusion, suggestions, created_at
				 FROM verification_reports ORDER BY created_at DESC LIMIT $1`
		args = []any{limit}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list reports: %w", err)
	}
	defer rows.Close()

	var reports []VerificationReport
	for rows.Next() {
		var rep VerificationReport
		var suggestions []byte
		if err := rows.Scan(&rep.ID, &rep.WorkflowID, &rep.TaskID, &rep.ResultScore, &rep.PathScore, &rep.EnvironmentScore, &rep.OverallScore, &rep.Conclusion, &suggestions, &rep.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan report: %w", err)
		}
		json.Unmarshal(suggestions, &rep.Suggestions)
		reports = append(reports, rep)
	}
	return reports, rows.Err()
}
```

Remove the `listReportsByWorkflow` helper and the problematic `var rows interface{ ... }` pattern — use the above clean implementation with a single method.

- [ ] **Step 2: Commit**

```bash
git add backend/internal/domain/verification/repository.go
git commit -m "feat: verification domain repository with report and review CRUD"
```

---

### Task 7: Verification domain — service + handler

**Files:**
- Create: `backend/internal/domain/verification/service.go`
- Create: `backend/internal/domain/verification/handler.go`

- [ ] **Step 1: Create service.go**

```go
package verification

import (
	"context"
	"errors"
	"fmt"

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

func (s *Service) CreateReport(ctx context.Context, input CreateReportInput) (*VerificationReport, error) {
	overall := calculateOverallScore(input.ResultScore, input.PathScore, input.EnvironmentScore)
	rep, err := s.repo.CreateReport(ctx, input)
	if err != nil {
		return nil, err
	}
	if overall != nil {
		if err := s.repo.UpdateOverallScore(ctx, rep.ID, *overall); err != nil {
			return nil, fmt.Errorf("update overall score: %w", err)
		}
	}
	rep.OverallScore = overall
	return rep, nil
}

func (s *Service) GetReport(ctx context.Context, id uuid.UUID) (*VerificationReport, error) {
	rep, err := s.repo.GetReport(ctx, id)
	if err != nil {
		return nil, err
	}
	reviews, err := s.repo.GetReviewsByReport(ctx, id)
	if err != nil {
		return nil, err
	}
	rep.Reviews = reviews
	return rep, nil
}

func (s *Service) ListReports(ctx context.Context, workflowID *uuid.UUID, limit int) ([]VerificationReport, error) {
	return s.repo.ListReports(ctx, workflowID, limit)
}

func (s *Service) AssignReview(ctx context.Context, input AssignReviewInput) (*ReviewAssignment, error) {
	if input.Level != "L1" && input.Level != "L2" && input.Level != "L3" {
		return nil, fmt.Errorf("%w: level must be L1, L2, or L3", ErrValidation)
	}
	if input.ReviewerType != "machine" && input.ReviewerType != "ai" && input.ReviewerType != "expert" {
		return nil, fmt.Errorf("%w: reviewer_type must be machine, ai, or expert", ErrValidation)
	}
	return s.repo.AssignReview(ctx, input)
}

func (s *Service) CompleteReview(ctx context.Context, reviewID uuid.UUID, result map[string]any) error {
	return s.repo.CompleteReview(ctx, reviewID, result)
}

func calculateOverallScore(result, path, env *float64) *float64 {
	if result == nil && path == nil && env == nil {
		return nil
	}
	r, p, e := 1.0, 1.0, 1.0
	if result != nil {
		r = *result
	}
	if path != nil {
		p = *path
	}
	if env != nil {
		e = *env
	}
	score := r*0.4 + p*0.35 + e*0.25
	return &score
}
```

- [ ] **Step 2: Create handler.go**

```go
package verification

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
	r.Post("/verification/reports", h.createReport)
	r.Get("/verification/reports", h.listReports)
	r.Get("/verification/reports/{id}", h.getReport)
	r.Post("/verification/reports/{id}/reviews", h.assignReview)
	r.Patch("/verification/reviews/{id}", h.completeReview)
}

func (h *Handler) createReport(w http.ResponseWriter, r *http.Request) {
	var input CreateReportInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	rep, err := h.service.CreateReport(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, rep)
}

func (h *Handler) listReports(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	var wfID *uuid.UUID
	if wfStr := r.URL.Query().Get("workflow_id"); wfStr != "" {
		parsed, err := uuid.Parse(wfStr)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid workflow_id"})
			return
		}
		wfID = &parsed
	}
	reports, err := h.service.ListReports(r.Context(), wfID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, reports)
}

func (h *Handler) getReport(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	rep, err := h.service.GetReport(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "report not found"})
		return
	}
	writeJSON(w, http.StatusOK, rep)
}

func (h *Handler) assignReview(w http.ResponseWriter, r *http.Request) {
	reportID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid report id"})
		return
	}
	var input AssignReviewInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	input.ReportID = reportID
	rev, err := h.service.AssignReview(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, rev)
}

func (h *Handler) completeReview(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid review id"})
		return
	}
	var req CompleteReviewInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	if err := h.service.CompleteReview(r.Context(), id, req.Result); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "review completed"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON error: %v", err)
	}
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/domain/verification/service.go backend/internal/domain/verification/handler.go
git commit -m "feat: verification domain service and handler with 5 API endpoints"
```

---

### Task 8: Governance domain — models

**Files:**
- Create: `backend/internal/domain/governance/model.go`
- Create directory: `backend/internal/domain/governance/`

- [ ] **Step 1: Create model.go**

```go
package governance

import (
	"time"

	"github.com/google/uuid"
)

type Permission struct {
	ID          uuid.UUID `json:"id"`
	Level       int       `json:"level"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Behavior    string    `json:"behavior"`
	CreatedAt   time.Time `json:"created_at"`
}

type Principle struct {
	ID              uuid.UUID          `json:"id"`
	Name            string             `json:"name"`
	Description     string             `json:"description"`
	EvaluationLogic map[string]any    `json:"evaluation_logic"`
	Priority        int                `json:"priority"`
	IsActive        bool               `json:"is_active"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
}

type ControlRule struct {
	ID               uuid.UUID          `json:"id"`
	PrincipleID      *uuid.UUID         `json:"principle_id,omitempty"`
	TargetEntityType string             `json:"target_entity_type"`
	TargetEntityID   *uuid.UUID         `json:"target_entity_id,omitempty"`
	Condition        map[string]any    `json:"condition"`
	Action           string             `json:"action"`
	Priority         int                `json:"priority"`
	IsActive         bool               `json:"is_active"`
	CreatedAt        time.Time          `json:"created_at"`
}

type CreatePrincipleInput struct {
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	EvaluationLogic map[string]any `json:"evaluation_logic,omitempty"`
	Priority        int            `json:"priority,omitempty"`
}

type CreateControlRuleInput struct {
	PrincipleID      *uuid.UUID      `json:"principle_id,omitempty"`
	TargetEntityType string          `json:"target_entity_type"`
	TargetEntityID   *uuid.UUID      `json:"target_entity_id,omitempty"`
	Condition        map[string]any `json:"condition,omitempty"`
	Action           string          `json:"action"`
	Priority         int             `json:"priority,omitempty"`
}

type PermissionCheckInput struct {
	UserID     uuid.UUID `json:"user_id"`
	Action     string    `json:"action"`
	Resource   string    `json:"resource"`
	ResourceID *uuid.UUID `json:"resource_id,omitempty"`
}

type PermissionCheckResult struct {
	Allowed    bool     `json:"allowed"`
	Level      int      `json:"level"`
	Behavior   string   `json:"behavior"`
	Reason     string   `json:"reason"`
}
```

- [ ] **Step 2: Commit**

```bash
mkdir -p /root/HarnessCompany/backend/internal/domain/governance
git add backend/internal/domain/governance/model.go
git commit -m "feat: governance domain models"
```

---

### Task 9: Governance domain — repository

**Files:**
- Create: `backend/internal/domain/governance/repository.go`

- [ ] **Step 1: Create repository.go**

```go
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
	err := r.db.QueryRow(ctx,
		`INSERT INTO permissions (level, name, description, behavior)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, level, name, description, behavior, created_at`,
		p.Level, p.Name, p.Description, p.Behavior,
	).Scan(&p.ID, &p.Level, &p.Name, &p.Description, &p.Behavior, &p.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create permission: %w", err)
	}
	return p, nil
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
	return perms, rows.Err()
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
	eval, _ := json.Marshal(input.EvaluationLogic)
	p := &Principle{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO principles (name, description, evaluation_logic, priority)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, name, description, evaluation_logic, priority, is_active, created_at, updated_at`,
		input.Name, input.Description, eval, input.Priority,
	).Scan(&p.ID, &p.Name, &p.Description, &eval, &p.Priority, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create principle: %w", err)
	}
	json.Unmarshal(eval, &p.EvaluationLogic)
	return p, nil
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
		var eval []byte
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &eval, &p.Priority, &p.IsActive, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan principle: %w", err)
		}
		json.Unmarshal(eval, &p.EvaluationLogic)
		principles = append(principles, p)
	}
	return principles, rows.Err()
}

func (r *Repository) GetPrinciple(ctx context.Context, id uuid.UUID) (*Principle, error) {
	p := &Principle{}
	var eval []byte
	err := r.db.QueryRow(ctx,
		`SELECT id, name, description, evaluation_logic, priority, is_active, created_at, updated_at
		 FROM principles WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.Description, &eval, &p.Priority, &p.IsActive, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get principle: %w", err)
	}
	json.Unmarshal(eval, &p.EvaluationLogic)
	return p, nil
}

func (r *Repository) CreateControlRule(ctx context.Context, input CreateControlRuleInput) (*ControlRule, error) {
	cond, _ := json.Marshal(input.Condition)
	rule := &ControlRule{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO control_rules (principle_id, target_entity_type, target_entity_id, condition, action, priority)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, principle_id, target_entity_type, target_entity_id, condition, action, priority, is_active, created_at`,
		input.PrincipleID, input.TargetEntityType, input.TargetEntityID, cond, input.Action, input.Priority,
	).Scan(&rule.ID, &rule.PrincipleID, &rule.TargetEntityType, &rule.TargetEntityID, &cond, &rule.Action, &rule.Priority, &rule.IsActive, &rule.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("create control rule: %w", err)
	}
	json.Unmarshal(cond, &rule.Condition)
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
		var cond []byte
		if err := rows.Scan(&rule.ID, &rule.PrincipleID, &rule.TargetEntityType, &rule.TargetEntityID, &cond, &rule.Action, &rule.Priority, &rule.IsActive, &rule.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan control rule: %w", err)
		}
		json.Unmarshal(cond, &rule.Condition)
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

func (r *Repository) GetControlRulesByTarget(ctx context.Context, entityType string, entityID *uuid.UUID) ([]ControlRule, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, principle_id, target_entity_type, target_entity_id, condition, action, priority, is_active, created_at
		 FROM control_rules WHERE target_entity_type = $1 AND (target_entity_id = $2 OR target_entity_id IS NULL)
		 ORDER BY priority DESC`, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("get control rules by target: %w", err)
	}
	defer rows.Close()

	var rules []ControlRule
	for rows.Next() {
		var rule ControlRule
		var cond []byte
		if err := rows.Scan(&rule.ID, &rule.PrincipleID, &rule.TargetEntityType, &rule.TargetEntityID, &cond, &rule.Action, &rule.Priority, &rule.IsActive, &rule.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan control rule: %w", err)
		}
		json.Unmarshal(cond, &rule.Condition)
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/domain/governance/repository.go
git commit -m "feat: governance domain repository with permission, principle, control rule CRUD"
```

---

### Task 10: Governance domain — service + handler

**Files:**
- Create: `backend/internal/domain/governance/service.go`
- Create: `backend/internal/domain/governance/handler.go`

- [ ] **Step 1: Create service.go**

```go
package governance

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("validation error")
	ErrDenied     = errors.New("permission denied")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreatePermission(ctx context.Context, p *Permission) (*Permission, error) {
	if p.Level < 1 || p.Level > 4 {
		return nil, fmt.Errorf("%w: level must be 1-4", ErrValidation)
	}
	if p.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	validBehaviors := map[string]bool{"auto": true, "notify": true, "approve": true, "deny": true}
	if !validBehaviors[p.Behavior] {
		p.Behavior = "notify"
	}
	return s.repo.CreatePermission(ctx, p)
}

func (s *Service) ListPermissions(ctx context.Context) ([]Permission, error) {
	return s.repo.ListPermissions(ctx)
}

func (s *Service) CreatePrinciple(ctx context.Context, input CreatePrincipleInput) (*Principle, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	if input.Description == "" {
		return nil, fmt.Errorf("%w: description is required", ErrValidation)
	}
	return s.repo.CreatePrinciple(ctx, input)
}

func (s *Service) ListPrinciples(ctx context.Context) ([]Principle, error) {
	return s.repo.ListPrinciples(ctx)
}

func (s *Service) GetPrinciple(ctx context.Context, id uuid.UUID) (*Principle, error) {
	return s.repo.GetPrinciple(ctx, id)
}

func (s *Service) CreateControlRule(ctx context.Context, input CreateControlRuleInput) (*ControlRule, error) {
	if input.Action == "" {
		return nil, fmt.Errorf("%w: action is required", ErrValidation)
	}
	return s.repo.CreateControlRule(ctx, input)
}

func (s *Service) ListControlRules(ctx context.Context) ([]ControlRule, error) {
	return s.repo.ListControlRules(ctx)
}

func (s *Service) CheckPermission(ctx context.Context, input PermissionCheckInput) (*PermissionCheckResult, error) {
	// Default deny
	result := &PermissionCheckResult{
		Allowed:  false,
		Level:    0,
		Behavior: "deny",
		Reason:   "no matching permission",
	}

	perms, err := s.repo.ListPermissions(ctx)
	if err != nil {
		return nil, err
	}

	for _, p := range perms {
		result.Level = p.Level
		result.Behavior = p.Behavior
		switch p.Behavior {
		case "auto":
			result.Allowed = true
			result.Reason = fmt.Sprintf("L%d auto-allowed", p.Level)
			return result, nil
		case "notify":
			result.Allowed = true
			result.Reason = fmt.Sprintf("L%d allowed with notification", p.Level)
			return result, nil
		case "approve":
			result.Allowed = false
			result.Reason = fmt.Sprintf("L%d requires approval", p.Level)
			return result, nil
		case "deny":
			result.Allowed = false
			result.Reason = fmt.Sprintf("L%d denied", p.Level)
			return result, nil
		}
	}

	return result, nil
}
```

- [ ] **Step 2: Create handler.go**

```go
package governance

import (
	"encoding/json"
	"log"
	"net/http"

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
	r.Post("/governance/permissions", h.createPermission)
	r.Get("/governance/permissions", h.listPermissions)
	r.Post("/governance/principles", h.createPrinciple)
	r.Get("/governance/principles", h.listPrinciples)
	r.Get("/governance/principles/{id}", h.getPrinciple)
	r.Post("/governance/control-rules", h.createControlRule)
	r.Get("/governance/control-rules", h.listControlRules)
	r.Post("/governance/check", h.checkPermission)
}

func (h *Handler) createPermission(w http.ResponseWriter, r *http.Request) {
	var p Permission
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&p); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	created, err := h.service.CreatePermission(r.Context(), &p)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (h *Handler) listPermissions(w http.ResponseWriter, r *http.Request) {
	perms, err := h.service.ListPermissions(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, perms)
}

func (h *Handler) createPrinciple(w http.ResponseWriter, r *http.Request) {
	var input CreatePrincipleInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	p, err := h.service.CreatePrinciple(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

func (h *Handler) listPrinciples(w http.ResponseWriter, r *http.Request) {
	principles, err := h.service.ListPrinciples(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, principles)
}

func (h *Handler) getPrinciple(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid id"})
		return
	}
	p, err := h.service.GetPrinciple(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "principle not found"})
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *Handler) createControlRule(w http.ResponseWriter, r *http.Request) {
	var input CreateControlRuleInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	rule, err := h.service.CreateControlRule(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

func (h *Handler) listControlRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.service.ListControlRules(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

func (h *Handler) checkPermission(w http.ResponseWriter, r *http.Request) {
	var input PermissionCheckInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	result, err := h.service.CheckPermission(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON error: %v", err)
	}
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/domain/governance/service.go backend/internal/domain/governance/handler.go
git commit -m "feat: governance domain service and handler with 8 API endpoints"
```

---

### Task 11: Wire all three domains into router + main.go + build

**Files:**
- Modify: `backend/internal/gateway/router.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: Update router.go**

```go
package gateway

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/harness-org/backend/internal/domain/capability"
	"github.com/harness-org/backend/internal/domain/governance"
	"github.com/harness-org/backend/internal/domain/identity"
	"github.com/harness-org/backend/internal/domain/layer"
	"github.com/harness-org/backend/internal/domain/observability"
	"github.com/harness-org/backend/internal/domain/organization"
	"github.com/harness-org/backend/internal/domain/verification"
	"github.com/harness-org/backend/internal/domain/workflow"
)

type Dependencies struct {
	IdentityHandler       *identity.Handler
	OrganizationHandler   *organization.Handler
	LayerHandler          *layer.Handler
	CapabilityHandler     *capability.Handler
	WorkflowHandler       *workflow.Handler
	ObservabilityHandler  *observability.Handler
	VerificationHandler   *verification.Handler
	GovernanceHandler     *governance.Handler
}

func RegisterRoutes(r *chi.Mux, deps *Dependencies) {
	if deps == nil {
		panic("gateway.RegisterRoutes: deps must not be nil")
	}
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", healthCheck)
		deps.IdentityHandler.RegisterRoutes(r)
		if deps.OrganizationHandler != nil {
			deps.OrganizationHandler.RegisterRoutes(r)
		}
		if deps.LayerHandler != nil {
			deps.LayerHandler.RegisterRoutes(r)
		}
		if deps.CapabilityHandler != nil {
			deps.CapabilityHandler.RegisterRoutes(r)
		}
		if deps.WorkflowHandler != nil {
			deps.WorkflowHandler.RegisterRoutes(r)
		}
		if deps.ObservabilityHandler != nil {
			deps.ObservabilityHandler.RegisterRoutes(r)
		}
		if deps.VerificationHandler != nil {
			deps.VerificationHandler.RegisterRoutes(r)
		}
		if deps.GovernanceHandler != nil {
			deps.GovernanceHandler.RegisterRoutes(r)
		}
	})
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
		log.Printf("health check write error: %v", err)
	}
}
```

- [ ] **Step 2: Update main.go**

```go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/harness-org/backend/internal/domain/capability"
	"github.com/harness-org/backend/internal/domain/governance"
	"github.com/harness-org/backend/internal/domain/identity"
	"github.com/harness-org/backend/internal/domain/layer"
	"github.com/harness-org/backend/internal/domain/observability"
	"github.com/harness-org/backend/internal/domain/organization"
	"github.com/harness-org/backend/internal/domain/verification"
	"github.com/harness-org/backend/internal/domain/workflow"
	"github.com/harness-org/backend/internal/gateway"
	"github.com/harness-org/backend/internal/pkg/config"
	"github.com/harness-org/backend/internal/pkg/database"
	"github.com/harness-org/backend/internal/pkg/server"
)

func main() {
	cfg := config.Load()

	connCtx, connCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer connCancel()

	db, err := database.Connect(connCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	if err := database.RunMigrations(context.Background(), db, cfg.MigrationsPath); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	identRepo := identity.NewRepository(db)
	identSvc := identity.NewService(identRepo, cfg.JWTSecret)
	identHandler := identity.NewHandler(identSvc)

	orgRepo := organization.NewRepository(db)
	orgSvc := organization.NewService(orgRepo)
	orgHandler := organization.NewHandler(orgSvc)

	layerRepo := layer.NewRepository(db)
	layerClassifier := layer.NewClassifierService(layerRepo)
	layerHandler := layer.NewHandler(layerClassifier)

	capRepo := capability.NewRepository(db)
	capRouter := capability.NewRouter(capRepo)
	capHandler := capability.NewHandler(capRepo, capRouter)

	wfRepo := workflow.NewRepository(db)
	wfSvc := workflow.NewService(wfRepo)
	wfHandler := workflow.NewHandler(wfSvc)

	obsRepo := observability.NewRepository(db)
	obsSvc := observability.NewService(obsRepo)
	obsHandler := observability.NewHandler(obsSvc)

	verRepo := verification.NewRepository(db)
	verSvc := verification.NewService(verRepo)
	verHandler := verification.NewHandler(verSvc)

	govRepo := governance.NewRepository(db)
	govSvc := governance.NewService(govRepo)
	govHandler := governance.NewHandler(govSvc)

	router := server.NewRouter(cfg.CorsOrigins)
	gateway.RegisterRoutes(router, &gateway.Dependencies{
		IdentityHandler:       identHandler,
		OrganizationHandler:   orgHandler,
		LayerHandler:          layerHandler,
		CapabilityHandler:     capHandler,
		WorkflowHandler:       wfHandler,
		ObservabilityHandler:  obsHandler,
		VerificationHandler:   verHandler,
		GovernanceHandler:     govHandler,
	})

	srv := server.New(router, cfg.ServerPort)
	go func() {
		log.Printf("server starting on :%d", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}
```

- [ ] **Step 3: Build and verify**

```bash
go build ./cmd/server/
```

Expected: No errors, binary compiled.

- [ ] **Step 4: Run go vet**

```bash
go vet ./internal/domain/observability/... ./internal/domain/verification/... ./internal/domain/governance/... ./internal/gateway/... ./cmd/server/...
```

Expected: No warnings.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/gateway/router.go backend/cmd/server/main.go
git commit -m "feat: wire observability, verification, governance handlers into router and main"
```

---

## Self-Review Checklist

- [ ] **Spec coverage:** Every domain from spec (Observability lines 297-323, Verification lines 325-358, Governance lines 361-386) has matching tasks
- [ ] **No placeholders:** All files contain complete, compilable Go code
- [ ] **Type consistency:** All method signatures match across model → repository → service → handler chains
- [ ] **Pattern consistency:** Follows existing 4-file domain pattern (model.go, repository.go, service.go, handler.go)
- [ ] **Migration order:** 007, 008, 009 continue the sequence from 006
- [ ] **26 new API endpoints total:** 8 observability + 5 verification + 8 governance = 21 new + update router/main
