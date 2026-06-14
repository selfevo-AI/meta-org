package dashboard

import (
	"context"
	"fmt"
	"time"
)

type SummaryRepository interface {
	Identity(ctx context.Context) (IdentitySummary, error)
	Organization(ctx context.Context) (OrganizationSummary, error)
	Workflow(ctx context.Context) (WorkflowSummary, error)
	Capability(ctx context.Context) (CapabilitySummary, error)
	Observability(ctx context.Context) (ObservabilitySummary, error)
	Verification(ctx context.Context) (VerificationSummary, error)
	Governance(ctx context.Context) (GovernanceSummary, error)
	Evolution(ctx context.Context) (EvolutionSummary, error)
	RecentEvents(ctx context.Context, limit int) ([]RecentEvent, error)
}

type Service struct {
	repo SummaryRepository
	now  func() time.Time
}

func NewService(repo SummaryRepository) *Service {
	return &Service{
		repo: repo,
		now:  time.Now,
	}
}

func (s *Service) GetOverview(ctx context.Context) (*Overview, error) {
	identity, err := s.repo.Identity(ctx)
	if err != nil {
		return nil, fmt.Errorf("identity: %w", err)
	}
	organization, err := s.repo.Organization(ctx)
	if err != nil {
		return nil, fmt.Errorf("organization: %w", err)
	}
	workflow, err := s.repo.Workflow(ctx)
	if err != nil {
		return nil, fmt.Errorf("workflow: %w", err)
	}
	capability, err := s.repo.Capability(ctx)
	if err != nil {
		return nil, fmt.Errorf("capability: %w", err)
	}
	observability, err := s.repo.Observability(ctx)
	if err != nil {
		return nil, fmt.Errorf("observability: %w", err)
	}
	verification, err := s.repo.Verification(ctx)
	if err != nil {
		return nil, fmt.Errorf("verification: %w", err)
	}
	governance, err := s.repo.Governance(ctx)
	if err != nil {
		return nil, fmt.Errorf("governance: %w", err)
	}
	evolution, err := s.repo.Evolution(ctx)
	if err != nil {
		return nil, fmt.Errorf("evolution: %w", err)
	}
	events, err := s.repo.RecentEvents(ctx, 10)
	if err != nil {
		return nil, fmt.Errorf("recent events: %w", err)
	}

	return &Overview{
		GeneratedAt:   s.now().UTC(),
		Identity:      identity,
		Organization:  organization,
		Workflow:      workflow,
		Capability:    capability,
		Observability: observability,
		Verification:  verification,
		Governance:    governance,
		Evolution:     evolution,
		RecentEvents:  events,
	}, nil
}
