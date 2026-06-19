package singleorg

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/selfevo-AI/meta-org/backend/internal/domain/identity"
	"github.com/selfevo-AI/meta-org/backend/internal/pkg/middleware"
)

var ErrValidation = errors.New("validation error")

type Repository interface {
	GetUserProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error)
	EnsureSingleOrgForUser(ctx context.Context, userID uuid.UUID) (*OrganizationAccount, error)
	GetHumanMembership(ctx context.Context, userID uuid.UUID, orgID uuid.UUID) (*Membership, error)
	GetAgentMembership(ctx context.Context, agentID uuid.UUID, orgID uuid.UUID) (*Membership, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*UserProfile, error) {
	profile, err := s.repo.GetUserProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, fmt.Errorf("%w: user profile not found", ErrValidation)
	}
	profile.OnboardingRequired = false
	if profile.Organizations == nil {
		profile.Organizations = []OrganizationAccount{}
	}
	if profile.DefaultOrganizationID == nil {
		org, err := s.repo.EnsureSingleOrgForUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		if org != nil {
			profile.DefaultOrganizationID = &org.ID
			if !containsOrganization(profile.Organizations, org.ID) {
				profile.Organizations = append(profile.Organizations, *org)
			}
		}
	}
	if profile.OnboardingStatus == "" {
		profile.OnboardingStatus = OnboardingComplete
	}
	return profile, nil
}

func (s *Service) IdentitySessionProfile(ctx context.Context, userID uuid.UUID) (*identity.SessionProfile, error) {
	profile, err := s.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	orgs := make([]identity.AuthOrganization, 0, len(profile.Organizations))
	for _, org := range profile.Organizations {
		orgs = append(orgs, identity.AuthOrganization{
			ID:            org.ID,
			Name:          org.Name,
			Description:   org.Description,
			MembershipID:  org.MembershipID,
			AuthorityTier: org.AuthorityTier,
			IsOwner:       org.IsOwner,
		})
	}
	return &identity.SessionProfile{
		OnboardingRequired:    false,
		DefaultOrganizationID: profile.DefaultOrganizationID,
		Organizations:         orgs,
	}, nil
}

func (s *Service) ResolveTenant(ctx context.Context, user middleware.AuthenticatedUser, requestedOrganizationID string) (*middleware.TenantContext, error) {
	actorID, err := uuid.Parse(user.ID)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid actor id", ErrValidation)
	}
	requested, err := parseOptionalUUID(requestedOrganizationID)
	if err != nil {
		return nil, err
	}

	if user.Type == "ai" {
		if requested == nil {
			return nil, middleware.ErrTenantRequired
		}
		membership, err := s.repo.GetAgentMembership(ctx, actorID, *requested)
		if err != nil || membership == nil {
			return nil, middleware.ErrTenantForbidden
		}
		membershipID := membership.ID
		orgID := membership.OrganizationID
		return &middleware.TenantContext{
			Mode:           ModeSingleOrg,
			UserID:         actorID,
			OrganizationID: &orgID,
			MembershipID:   &membershipID,
			AuthorityTier:  membership.AuthorityTier,
		}, nil
	}

	profile, err := s.GetProfile(ctx, actorID)
	if err != nil {
		return nil, err
	}
	orgID := requested
	if orgID == nil {
		orgID = profile.DefaultOrganizationID
	}
	if orgID == nil {
		return nil, middleware.ErrTenantRequired
	}

	membership, err := s.repo.GetHumanMembership(ctx, actorID, *orgID)
	if err != nil || membership == nil {
		return nil, middleware.ErrTenantForbidden
	}
	membershipID := membership.ID
	resolvedOrgID := membership.OrganizationID
	return &middleware.TenantContext{
		Mode:             ModeSingleOrg,
		UserID:           actorID,
		OrganizationID:   &resolvedOrgID,
		MembershipID:     &membershipID,
		AuthorityTier:    membership.AuthorityTier,
		OnboardingStatus: OnboardingComplete,
	}, nil
}

func parseOptionalUUID(raw string) (*uuid.UUID, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid organization id", ErrValidation)
	}
	return &id, nil
}

func containsOrganization(orgs []OrganizationAccount, id uuid.UUID) bool {
	for _, org := range orgs {
		if org.ID == id {
			return true
		}
	}
	return false
}
