package organization

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

type Repository interface {
	CreateOrganization(ctx context.Context, input CreateOrganizationInput) (*Organization, error)
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (*Organization, error)
	CreateMVRU(ctx context.Context, input CreateMVRUInput) (*MVRU, error)
	GetMVRUByID(ctx context.Context, id uuid.UUID) (*MVRU, error)
	ListMVRUs(ctx context.Context, orgID uuid.UUID) ([]MVRU, error)
	UpdateMVRUStatus(ctx context.Context, id uuid.UUID, status MVRUStatus) error
	AddMember(ctx context.Context, member MVRUMember) error
	RemoveMember(ctx context.Context, mvruID, userID, agentID *uuid.UUID) error
	CreateRelationship(ctx context.Context, rel MVRURelationship) (*MVRURelationship, error)
	GetOrgChart(ctx context.Context, orgID uuid.UUID) ([]MVRU, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateOrganization(ctx context.Context, input CreateOrganizationInput) (*Organization, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	return s.repo.CreateOrganization(ctx, input)
}

func (s *Service) GetOrganization(ctx context.Context, id uuid.UUID) (*Organization, error) {
	return s.repo.GetOrganizationByID(ctx, id)
}

func (s *Service) GetOrgChart(ctx context.Context, orgID uuid.UUID) ([]MVRU, error) {
	return s.repo.GetOrgChart(ctx, orgID)
}

func (s *Service) CreateMVRU(ctx context.Context, input CreateMVRUInput) (*MVRU, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	if input.Boundary == nil {
		input.Boundary = map[string]any{}
	}
	if input.Config == nil {
		input.Config = map[string]any{}
	}
	return s.repo.CreateMVRU(ctx, input)
}

func (s *Service) GetMVRU(ctx context.Context, id uuid.UUID) (*MVRU, error) {
	mvru, err := s.repo.GetMVRUByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mvru, nil
}

func (s *Service) ActivateMVRU(ctx context.Context, id uuid.UUID) error {
	return s.repo.UpdateMVRUStatus(ctx, id, MVRUActive)
}

func (s *Service) EvaluateMVRU(ctx context.Context, id uuid.UUID) error {
	return s.repo.UpdateMVRUStatus(ctx, id, MVRUEvaluating)
}

func (s *Service) AddMember(ctx context.Context, mvruID, roleID uuid.UUID, userID, agentID *uuid.UUID) error {
	if userID == nil && agentID == nil {
		return fmt.Errorf("%w: user_id or agent_id is required", ErrValidation)
	}
	return s.repo.AddMember(ctx, MVRUMember{
		MVRUID:  mvruID,
		UserID:  userID,
		AgentID: agentID,
		RoleID:  roleID,
	})
}

func (s *Service) RemoveMember(ctx context.Context, mvruID uuid.UUID, userID, agentID *uuid.UUID) error {
	return s.repo.RemoveMember(ctx, &mvruID, userID, agentID)
}

func (s *Service) CreateRelationship(ctx context.Context, sourceID, targetID uuid.UUID, relType string, config map[string]any) (*MVRURelationship, error) {
	return s.repo.CreateRelationship(ctx, MVRURelationship{
		SourceMVRUID: sourceID,
		TargetMVRUID: targetID,
		RelType:      relType,
		Config:       config,
	})
}
