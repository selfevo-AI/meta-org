package observability

import (
	"time"

	"github.com/google/uuid"
)

type TraceStatus string

const (
	TraceActive   TraceStatus = "active"
	TraceComplete TraceStatus = "completed"
	TraceFailed   TraceStatus = "failed"
)

type SpanType string

const (
	SpanDecision       SpanType = "decision"
	SpanInvocation     SpanType = "invocation"
	SpanTask           SpanType = "task"
	SpanWorkflow       SpanType = "workflow"
	SpanAIInvocation   SpanType = "ai_invocation"
	SpanAIStream       SpanType = "ai_stream"
	SpanToolExecution  SpanType = "tool_execution"
	SpanFinanceExport  SpanType = "finance_export"
	SpanFinanceWebhook SpanType = "finance_webhook"
)

type MetricType string

const (
	MetricEfficiency  MetricType = "efficiency"
	MetricQuality     MetricType = "quality"
	MetricCost        MetricType = "cost"
	MetricHealth      MetricType = "health"
	MetricUsage       MetricType = "usage"
	MetricReliability MetricType = "reliability"
)

type Trace struct {
	ID          uuid.UUID      `json:"id"`
	WorkflowID  *uuid.UUID     `json:"workflow_id,omitempty"`
	Status      TraceStatus    `json:"status"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt *time.Time     `json:"completed_at,omitempty"`
	Metadata    map[string]any `json:"metadata"`
	Spans       []Span         `json:"spans,omitempty"`
}

type Span struct {
	ID           uuid.UUID      `json:"id"`
	TraceID      uuid.UUID      `json:"trace_id"`
	ParentSpanID *uuid.UUID     `json:"parent_span_id,omitempty"`
	SpanType     SpanType       `json:"span_type"`
	EntityID     *uuid.UUID     `json:"entity_id,omitempty"`
	EntityType   string         `json:"entity_type,omitempty"`
	ActorID      *uuid.UUID     `json:"actor_id,omitempty"`
	ActorType    string         `json:"actor_type,omitempty"`
	Input        map[string]any `json:"input,omitempty"`
	Output       map[string]any `json:"output,omitempty"`
	StartedAt    time.Time      `json:"started_at"`
	CompletedAt  *time.Time     `json:"completed_at,omitempty"`
	DurationMs   int            `json:"duration_ms"`
	Metadata     map[string]any `json:"metadata"`
}

type Metric struct {
	ID         uuid.UUID      `json:"id"`
	MetricType MetricType     `json:"metric_type"`
	MetricName string         `json:"metric_name"`
	EntityID   *uuid.UUID     `json:"entity_id,omitempty"`
	EntityType string         `json:"entity_type,omitempty"`
	Value      float64        `json:"value"`
	RecordedAt time.Time      `json:"recorded_at"`
	Metadata   map[string]any `json:"metadata"`
}

type CreateTraceInput struct {
	WorkflowID *uuid.UUID     `json:"workflow_id,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type RecordSpanInput struct {
	TraceID      uuid.UUID      `json:"trace_id"`
	ParentSpanID *uuid.UUID     `json:"parent_span_id,omitempty"`
	SpanType     SpanType       `json:"span_type"`
	EntityID     *uuid.UUID     `json:"entity_id,omitempty"`
	EntityType   string         `json:"entity_type,omitempty"`
	ActorID      *uuid.UUID     `json:"actor_id,omitempty"`
	ActorType    string         `json:"actor_type,omitempty"`
	Input        map[string]any `json:"input,omitempty"`
	Output       map[string]any `json:"output,omitempty"`
	DurationMs   int            `json:"duration_ms"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type RecordMetricInput struct {
	MetricType MetricType     `json:"metric_type"`
	MetricName string         `json:"metric_name"`
	EntityID   *uuid.UUID     `json:"entity_id,omitempty"`
	EntityType string         `json:"entity_type,omitempty"`
	Value      float64        `json:"value"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type MetricsQuery struct {
	MetricType MetricType `json:"metric_type,omitempty"`
	MetricName string     `json:"metric_name,omitempty"`
	EntityID   *uuid.UUID `json:"entity_id,omitempty"`
	From       *time.Time `json:"from,omitempty"`
	To         *time.Time `json:"to,omitempty"`
	Limit      int        `json:"limit,omitempty"`
}
