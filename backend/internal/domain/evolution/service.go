package evolution

import (
	"context"
	"errors"
	"math"

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
	w, err := s.repo.GetWeight(ctx, input.ActorID, input.ActorType)
	if err != nil {
		w = &DecisionWeight{
			ActorID:          input.ActorID,
			ActorType:        input.ActorType,
			OverallScore:     1.0,
			ExpertiseScore:   0.5,
			TrackRecordScore: 0.5,
			ReliabilityScore: 0.5,
			RecencyScore:     1.0,
			ContextFitScore:  0.5,
			PrincipleScore:   0.5,
			DecisionCount:    0,
		}
	}

	a, err := s.repo.GetAlpha(ctx)
	if err != nil {
		a = &AlphaConfig{
			Expertise:   0.25,
			TrackRecord: 0.20,
			Reliability: 0.15,
			Recency:     0.10,
			ContextFit:  0.10,
			Principle:   0.20,
		}
	}

	overall := a.Expertise*w.ExpertiseScore +
		a.TrackRecord*w.TrackRecordScore +
		a.Reliability*w.ReliabilityScore +
		a.Recency*w.RecencyScore +
		a.ContextFit*w.ContextFitScore +
		a.Principle*w.PrincipleScore

	w.OverallScore = math.Min(overall, 1.0)

	if err := s.repo.UpsertWeight(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *Service) RecordOutcome(ctx context.Context, input OutcomeInput) (*DecisionWeight, error) {
	w, err := s.repo.GetWeight(ctx, input.ActorID, input.ActorType)
	if err != nil {
		return nil, ErrNotFound
	}

	n := float64(w.DecisionCount + 1)
	w.TrackRecordScore = ((w.TrackRecordScore * (n - 1)) + input.OutcomeScore) / n
	w.RecencyScore = 1.0
	w.ContextFitScore = (w.ContextFitScore + 0.5) / 2
	w.DecisionCount = int(n)

	a, err := s.repo.GetAlpha(ctx)
	if err != nil {
		a = &AlphaConfig{
			Expertise:   0.25,
			TrackRecord: 0.20,
			Reliability: 0.15,
			Recency:     0.10,
			ContextFit:  0.10,
			Principle:   0.20,
		}
	}

	overall := a.Expertise*w.ExpertiseScore +
		a.TrackRecord*w.TrackRecordScore +
		a.Reliability*w.ReliabilityScore +
		a.Recency*w.RecencyScore +
		a.ContextFit*w.ContextFitScore +
		a.Principle*w.PrincipleScore

	w.OverallScore = math.Min(overall, 1.0)

	if err := s.repo.UpsertWeight(ctx, w); err != nil {
		return nil, err
	}
	return w, nil
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
	sum := a.Expertise + a.TrackRecord + a.Reliability + a.Recency + a.ContextFit + a.Principle
	if sum < 0.95 || sum > 1.05 {
		return ErrValidation
	}
	return s.repo.UpdateAlpha(ctx, a)
}

func (s *Service) CreateExperiment(ctx context.Context, input CreateExperimentInput) (*Experiment, error) {
	if input.Name == "" || input.Hypothesis == "" {
		return nil, ErrValidation
	}
	if input.SuccessCriteria == nil {
		input.SuccessCriteria = map[string]any{}
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
	if input.Title == "" || input.Content == "" {
		return nil, ErrValidation
	}
	if input.Source == "" {
		input.Source = "manual"
	}
	if input.Tags == nil {
		input.Tags = []string{}
	}
	return s.repo.CreateKnowledge(ctx, input)
}

func (s *Service) ListKnowledge(ctx context.Context, limit int) ([]KnowledgeEntry, error) {
	return s.repo.ListKnowledge(ctx, limit)
}

func (s *Service) CreateSignal(ctx context.Context, input CreateSignalInput) (*Signal, error) {
	if input.SignalType == "" {
		return nil, ErrValidation
	}
	if input.Data == nil {
		input.Data = map[string]any{}
	}
	return s.repo.CreateSignal(ctx, input)
}

func (s *Service) ListSignals(ctx context.Context, acknowledged *bool, limit int) ([]Signal, error) {
	return s.repo.ListSignals(ctx, acknowledged, limit)
}

func (s *Service) AcknowledgeSignal(ctx context.Context, id uuid.UUID) error {
	return s.repo.AcknowledgeSignal(ctx, id)
}
