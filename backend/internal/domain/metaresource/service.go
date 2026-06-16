package metaresource

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

var ErrValidation = errors.New("validation error")

type Store interface {
	CreateResource(ctx context.Context, input CreateMetaResourceInput) (*MetaResource, error)
	GetResource(ctx context.Context, id uuid.UUID) (*MetaResource, error)
	ListResources(ctx context.Context, filter ListFilter) ([]MetaResource, error)
	ResourceSummary(ctx context.Context, limit int) (*ResourceSummary, error)
	CreateDemandProfile(ctx context.Context, input CreateDemandProfileInput) (*DemandProfile, error)
	ListDemandProfiles(ctx context.Context, limit int) ([]DemandProfile, error)
	CreateCycle(ctx context.Context, input CreatePDCACycleInput) (*PDCACycle, error)
	ListCycles(ctx context.Context, limit int) ([]PDCACycle, error)
	CreateEvent(ctx context.Context, input CreatePDCAEventInput) (*PDCAEvent, error)
	ListEvents(ctx context.Context, filter ListFilter) ([]PDCAEvent, error)
	SyncExistingResources(ctx context.Context) (map[string]int, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) CreateResource(ctx context.Context, input CreateMetaResourceInput) (*MetaResource, error) {
	input.ResourceType = normalize(input.ResourceType)
	input.Status = normalize(input.Status)
	if input.ResourceType == "" || strings.TrimSpace(input.Name) == "" {
		return nil, fmt.Errorf("%w: resource_type and name are required", ErrValidation)
	}
	if !validResourceType(input.ResourceType) {
		return nil, fmt.Errorf("%w: unsupported resource_type %q", ErrValidation, input.ResourceType)
	}
	if input.Status == "" {
		input.Status = "active"
	}
	return s.store.CreateResource(ctx, input)
}

func (s *Service) ListResources(ctx context.Context, filter ListFilter) ([]MetaResource, error) {
	filter.ResourceType = normalize(filter.ResourceType)
	filter.Status = normalize(filter.Status)
	return s.store.ListResources(ctx, filter)
}

func (s *Service) GetResource(ctx context.Context, id uuid.UUID) (*MetaResource, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("%w: id is required", ErrValidation)
	}
	return s.store.GetResource(ctx, id)
}

func (s *Service) ResourceSummary(ctx context.Context, limit int) (*ResourceSummary, error) {
	return s.store.ResourceSummary(ctx, limit)
}

func (s *Service) SyncExistingResources(ctx context.Context) (map[string]int, error) {
	return s.store.SyncExistingResources(ctx)
}

func (s *Service) CreateDemandProfile(ctx context.Context, input CreateDemandProfileInput) (*DemandProfile, error) {
	input.Status = normalize(input.Status)
	if strings.TrimSpace(input.Title) == "" {
		return nil, fmt.Errorf("%w: title is required", ErrValidation)
	}
	if input.Status == "" {
		input.Status = "draft"
	}
	return s.store.CreateDemandProfile(ctx, input)
}

func (s *Service) ListDemandProfiles(ctx context.Context, limit int) ([]DemandProfile, error) {
	return s.store.ListDemandProfiles(ctx, limit)
}

func (s *Service) CreateCycle(ctx context.Context, input CreatePDCACycleInput) (*PDCACycle, error) {
	input.Status = normalize(input.Status)
	input.CurrentStage = normalize(input.CurrentStage)
	if input.CurrentStage == "" {
		input.CurrentStage = StagePlan
	}
	if !validStage(input.CurrentStage) {
		return nil, fmt.Errorf("%w: unsupported current_stage %q", ErrValidation, input.CurrentStage)
	}
	if input.Status == "" {
		input.Status = "active"
	}
	return s.store.CreateCycle(ctx, input)
}

func (s *Service) ListCycles(ctx context.Context, limit int) ([]PDCACycle, error) {
	return s.store.ListCycles(ctx, limit)
}

func (s *Service) CreateEvent(ctx context.Context, input CreatePDCAEventInput) (*PDCAEvent, error) {
	input.Stage = normalize(input.Stage)
	input.EventType = normalize(input.EventType)
	if input.CycleID == uuid.Nil {
		return nil, fmt.Errorf("%w: cycle_id is required", ErrValidation)
	}
	if !validStage(input.Stage) {
		return nil, fmt.Errorf("%w: unsupported stage %q", ErrValidation, input.Stage)
	}
	if input.EventType == "" {
		input.EventType = "note"
	}
	return s.store.CreateEvent(ctx, input)
}

func (s *Service) ListEvents(ctx context.Context, filter ListFilter) ([]PDCAEvent, error) {
	return s.store.ListEvents(ctx, filter)
}

func normalize(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func validResourceType(value string) bool {
	switch value {
	case ResourceHuman, ResourceInternalHuman, ResourceExternal, ResourceAgent, ResourceInternalAgent, ResourceExternalAgent, ResourceModelChannel, ResourceTool, ResourceMaterial, ResourceTime, ResourceCapability, ResourceBudget, "resource":
		return true
	default:
		return false
	}
}

func validStage(value string) bool {
	switch value {
	case StagePlan, StageDo, StageChange, StageAccept:
		return true
	default:
		return false
	}
}
