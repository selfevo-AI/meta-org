package singleorg

import (
	"time"

	"github.com/google/uuid"
)

const (
	ModeSingleOrg = "single_org"

	OnboardingComplete = "complete"

	AuthorityOwner = "organization_creator"
	AuthorityAdmin = "organization_admin"
)

type UserProfile struct {
	ID                    uuid.UUID             `json:"id"`
	Name                  string                `json:"name"`
	Email                 string                `json:"email"`
	AccountStatus         string                `json:"account_status"`
	OnboardingStatus      string                `json:"onboarding_status"`
	OnboardingRequired    bool                  `json:"onboarding_required"`
	DefaultOrganizationID *uuid.UUID            `json:"default_organization_id,omitempty"`
	Organizations         []OrganizationAccount `json:"organizations"`
	CreatedAt             time.Time             `json:"-"`
	UpdatedAt             time.Time             `json:"-"`
}

type OrganizationAccount struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	Description   string     `json:"description,omitempty"`
	MembershipID  *uuid.UUID `json:"membership_id,omitempty"`
	AuthorityTier string     `json:"authority_tier,omitempty"`
	IsOwner       bool       `json:"is_owner"`
}

type Membership struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	AuthorityTier  string
}
