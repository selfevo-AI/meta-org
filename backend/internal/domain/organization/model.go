package organization

import (
	"time"

	"github.com/google/uuid"
)

type MVRUStatus string

const (
	MVRUDesigning  MVRUStatus = "designing"
	MVRUActive     MVRUStatus = "active"
	MVRUEvaluating MVRUStatus = "evaluating"
	MVRUEvolving   MVRUStatus = "evolving"
	MVRUDissolved  MVRUStatus = "dissolved"
)

type Organization struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type MVRU struct {
	ID             uuid.UUID          `json:"id"`
	OrganizationID uuid.UUID          `json:"organization_id"`
	Name           string             `json:"name"`
	Description    string             `json:"description,omitempty"`
	Status         MVRUStatus         `json:"status"`
	Boundary       map[string]any     `json:"boundary"`
	Config         map[string]any     `json:"config"`
	ParentID       *uuid.UUID         `json:"parent_id,omitempty"`
	Children       []MVRU             `json:"children,omitempty"`
	Members        []MVRUMember       `json:"members,omitempty"`
	Relationships  []MVRURelationship `json:"relationships,omitempty"`
	CreatedAt      time.Time          `json:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at"`
}

type Team struct {
	ID          uuid.UUID `json:"id"`
	MVRUID      uuid.UUID `json:"mvru_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type MVRUMember struct {
	MVRUID  uuid.UUID  `json:"mvru_id"`
	UserID  *uuid.UUID `json:"user_id,omitempty"`
	AgentID *uuid.UUID `json:"agent_id,omitempty"`
	RoleID  uuid.UUID  `json:"role_id"`
}

type MVRURelationship struct {
	ID           uuid.UUID      `json:"id"`
	SourceMVRUID uuid.UUID      `json:"source_mvru_id"`
	TargetMVRUID uuid.UUID      `json:"target_mvru_id"`
	RelType      string         `json:"rel_type"`
	Config       map[string]any `json:"config,omitempty"`
}

type CreateOrganizationInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type CreateMVRUInput struct {
	OrganizationID uuid.UUID      `json:"organization_id"`
	Name           string         `json:"name"`
	Description    string         `json:"description,omitempty"`
	Boundary       map[string]any `json:"boundary,omitempty"`
	Config         map[string]any `json:"config,omitempty"`
	ParentID       *uuid.UUID     `json:"parent_id,omitempty"`
}
