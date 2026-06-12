package observability

import (
	"context"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) StartTrace(ctx context.Context, workflowID *uuid.UUID, metadata map[string]any) (*Trace, error) {
	t, err := s.repo.CreateTrace(ctx, CreateTraceInput{
		WorkflowID: workflowID,
		Metadata:   metadata,
	})
	if err != nil {
		return nil, err
	}
	t.Spans = []Span{}
	return t, nil
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
