package singleorg

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/selfevo-AI/meta-org/backend/internal/pkg/middleware"
)

func TestIdentitySessionProfileEnsuresDefaultOrganization(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	membershipID := uuid.New()
	repo := &fakeRepository{
		profile: &UserProfile{
			ID:    userID,
			Name:  "Alice",
			Email: "alice@example.com",
			Organizations: []OrganizationAccount{
				{ID: orgID, Name: "Default Organization", MembershipID: &membershipID, AuthorityTier: AuthorityOwner, IsOwner: true},
			},
		},
		ensured: &OrganizationAccount{ID: orgID, Name: "Default Organization", MembershipID: &membershipID, AuthorityTier: AuthorityOwner, IsOwner: true},
	}

	profile, err := NewService(repo).IdentitySessionProfile(context.Background(), userID)
	if err != nil {
		t.Fatalf("IdentitySessionProfile returned error: %v", err)
	}

	if !repo.ensureCalled {
		t.Fatalf("expected service to ensure a default organization")
	}
	if profile.OnboardingRequired {
		t.Fatalf("single-org profile should not require onboarding")
	}
	if profile.DefaultOrganizationID == nil || *profile.DefaultOrganizationID != orgID {
		t.Fatalf("default organization id = %v, want %s", profile.DefaultOrganizationID, orgID)
	}
	if len(profile.Organizations) != 1 || profile.Organizations[0].AuthorityTier != AuthorityOwner {
		t.Fatalf("organizations = %#v, want owner membership", profile.Organizations)
	}
}

func TestResolveTenantUsesRequestedHumanOrganizationWhenMembershipExists(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	membershipID := uuid.New()
	repo := &fakeRepository{
		profile: &UserProfile{
			ID:                    userID,
			DefaultOrganizationID: &orgID,
			Organizations:         []OrganizationAccount{{ID: orgID, Name: "Org"}},
		},
		humanMembership: &Membership{ID: membershipID, OrganizationID: orgID, AuthorityTier: AuthorityAdmin},
	}

	tenant, err := NewService(repo).ResolveTenant(context.Background(), middleware.AuthenticatedUser{ID: userID.String(), Type: "human"}, orgID.String())
	if err != nil {
		t.Fatalf("ResolveTenant returned error: %v", err)
	}

	if tenant.Mode != ModeSingleOrg {
		t.Fatalf("mode = %q, want %q", tenant.Mode, ModeSingleOrg)
	}
	if tenant.OrganizationID == nil || *tenant.OrganizationID != orgID {
		t.Fatalf("organization id = %v, want %s", tenant.OrganizationID, orgID)
	}
	if tenant.MembershipID == nil || *tenant.MembershipID != membershipID {
		t.Fatalf("membership id = %v, want %s", tenant.MembershipID, membershipID)
	}
	if tenant.AuthorityTier != AuthorityAdmin {
		t.Fatalf("authority tier = %q, want %q", tenant.AuthorityTier, AuthorityAdmin)
	}
}

func TestResolveTenantRejectsHumanOrganizationWithoutMembership(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	repo := &fakeRepository{
		profile:  &UserProfile{ID: userID, DefaultOrganizationID: &orgID},
		humanErr: errors.New("not a member"),
	}

	_, err := NewService(repo).ResolveTenant(context.Background(), middleware.AuthenticatedUser{ID: userID.String(), Type: "human"}, orgID.String())
	if !errors.Is(err, middleware.ErrTenantForbidden) {
		t.Fatalf("ResolveTenant error = %v, want ErrTenantForbidden", err)
	}
}

func TestResolveTenantRequiresRequestedOrganizationForAgent(t *testing.T) {
	agentID := uuid.New()

	_, err := NewService(&fakeRepository{}).ResolveTenant(context.Background(), middleware.AuthenticatedUser{ID: agentID.String(), Type: "ai"}, "")
	if !errors.Is(err, middleware.ErrTenantRequired) {
		t.Fatalf("ResolveTenant error = %v, want ErrTenantRequired", err)
	}
}

func TestResolveTenantAllowsAgentMembership(t *testing.T) {
	agentID := uuid.New()
	orgID := uuid.New()
	membershipID := uuid.New()
	repo := &fakeRepository{
		agentMembership: &Membership{ID: membershipID, OrganizationID: orgID, AuthorityTier: "executor"},
	}

	tenant, err := NewService(repo).ResolveTenant(context.Background(), middleware.AuthenticatedUser{ID: agentID.String(), Type: "ai"}, orgID.String())
	if err != nil {
		t.Fatalf("ResolveTenant returned error: %v", err)
	}

	if tenant.OrganizationID == nil || *tenant.OrganizationID != orgID {
		t.Fatalf("organization id = %v, want %s", tenant.OrganizationID, orgID)
	}
	if tenant.MembershipID == nil || *tenant.MembershipID != membershipID {
		t.Fatalf("membership id = %v, want %s", tenant.MembershipID, membershipID)
	}
}

type fakeRepository struct {
	profile         *UserProfile
	profileErr      error
	ensured         *OrganizationAccount
	ensureErr       error
	ensureCalled    bool
	humanMembership *Membership
	humanErr        error
	agentMembership *Membership
	agentErr        error
}

func (r *fakeRepository) GetUserProfile(context.Context, uuid.UUID) (*UserProfile, error) {
	return r.profile, r.profileErr
}

func (r *fakeRepository) EnsureSingleOrgForUser(context.Context, uuid.UUID) (*OrganizationAccount, error) {
	r.ensureCalled = true
	return r.ensured, r.ensureErr
}

func (r *fakeRepository) GetHumanMembership(context.Context, uuid.UUID, uuid.UUID) (*Membership, error) {
	return r.humanMembership, r.humanErr
}

func (r *fakeRepository) GetAgentMembership(context.Context, uuid.UUID, uuid.UUID) (*Membership, error) {
	return r.agentMembership, r.agentErr
}
