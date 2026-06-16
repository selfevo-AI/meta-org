package metaresource

import (
	"time"

	"github.com/google/uuid"
)

const (
	ResourceHuman         = "human"
	ResourceInternalHuman = "internal_human"
	ResourceExternal      = "external_human"
	ResourceAgent         = "agent"
	ResourceInternalAgent = "internal_agent"
	ResourceExternalAgent = "external_agent"
	ResourceModelChannel  = "model_channel"
	ResourceTool          = "tool"
	ResourceMaterial      = "material"
	ResourceTime          = "time"
	ResourceCapability    = "capability"
	ResourceBudget        = "budget"

	StagePlan   = "plan"
	StageDo     = "do"
	StageChange = "change"
	StageAccept = "accept"
)

type MetaResource struct {
	ID                 uuid.UUID      `json:"id"`
	ResourceType       string         `json:"resource_type"`
	SourceType         string         `json:"source_type,omitempty"`
	SourceID           *uuid.UUID     `json:"source_id,omitempty"`
	Name               string         `json:"name"`
	Status             string         `json:"status"`
	OrganizationID     *uuid.UUID     `json:"organization_id,omitempty"`
	DepartmentID       *uuid.UUID     `json:"department_id,omitempty"`
	OwnerActorID       *uuid.UUID     `json:"owner_actor_id,omitempty"`
	OwnerActorType     string         `json:"owner_actor_type,omitempty"`
	CapabilityProfile  map[string]any `json:"capability_profile"`
	CostProfile        map[string]any `json:"cost_profile"`
	CapacityProfile    map[string]any `json:"capacity_profile"`
	RiskProfile        map[string]any `json:"risk_profile"`
	PerformanceProfile map[string]any `json:"performance_profile"`
	Metadata           map[string]any `json:"metadata"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
}

type DemandProfile struct {
	ID                    uuid.UUID      `json:"id"`
	RequirementID         *uuid.UUID     `json:"requirement_id,omitempty"`
	ProjectID             *uuid.UUID     `json:"project_id,omitempty"`
	Title                 string         `json:"title"`
	Goal                  string         `json:"goal"`
	Status                string         `json:"status"`
	AcceptanceCriteria    []any          `json:"acceptance_criteria"`
	RequiredCapabilities  []any          `json:"required_capabilities"`
	BudgetConstraints     map[string]any `json:"budget_constraints"`
	TimeConstraints       map[string]any `json:"time_constraints"`
	RiskConstraints       map[string]any `json:"risk_constraints"`
	ResourceFitCandidates []any          `json:"resource_fit_candidates"`
	Metadata              map[string]any `json:"metadata"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
}

type PDCACycle struct {
	ID              uuid.UUID      `json:"id"`
	DemandProfileID *uuid.UUID     `json:"demand_profile_id,omitempty"`
	RequirementID   *uuid.UUID     `json:"requirement_id,omitempty"`
	ProjectID       *uuid.UUID     `json:"project_id,omitempty"`
	Status          string         `json:"status"`
	CurrentStage    string         `json:"current_stage"`
	OutcomeScore    float64        `json:"outcome_score"`
	Summary         string         `json:"summary"`
	Metadata        map[string]any `json:"metadata"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty"`
}

type PDCAEvent struct {
	ID           uuid.UUID      `json:"id"`
	CycleID      uuid.UUID      `json:"cycle_id"`
	Stage        string         `json:"stage"`
	EventType    string         `json:"event_type"`
	SourceType   string         `json:"source_type,omitempty"`
	SourceID     *uuid.UUID     `json:"source_id,omitempty"`
	ActorID      *uuid.UUID     `json:"actor_id,omitempty"`
	ActorType    string         `json:"actor_type,omitempty"`
	ResourceRefs []any          `json:"resource_refs"`
	CostRefs     []any          `json:"cost_refs"`
	Evidence     map[string]any `json:"evidence"`
	Decision     string         `json:"decision"`
	NextAction   string         `json:"next_action"`
	Metadata     map[string]any `json:"metadata"`
	CreatedAt    time.Time      `json:"created_at"`
}

type ResourceSummary struct {
	Total            int            `json:"total"`
	Active           int            `json:"active"`
	ByType           map[string]int `json:"by_type"`
	ByStatus         map[string]int `json:"by_status"`
	AverageCostScore float64        `json:"average_cost_score"`
	AverageFitScore  float64        `json:"average_fit_score"`
	Recent           []MetaResource `json:"recent"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type CreateMetaResourceInput struct {
	ResourceType       string         `json:"resource_type"`
	SourceType         string         `json:"source_type,omitempty"`
	SourceID           *uuid.UUID     `json:"source_id,omitempty"`
	Name               string         `json:"name"`
	Status             string         `json:"status,omitempty"`
	OrganizationID     *uuid.UUID     `json:"organization_id,omitempty"`
	DepartmentID       *uuid.UUID     `json:"department_id,omitempty"`
	OwnerActorID       *uuid.UUID     `json:"owner_actor_id,omitempty"`
	OwnerActorType     string         `json:"owner_actor_type,omitempty"`
	CapabilityProfile  map[string]any `json:"capability_profile,omitempty"`
	CostProfile        map[string]any `json:"cost_profile,omitempty"`
	CapacityProfile    map[string]any `json:"capacity_profile,omitempty"`
	RiskProfile        map[string]any `json:"risk_profile,omitempty"`
	PerformanceProfile map[string]any `json:"performance_profile,omitempty"`
	Metadata           map[string]any `json:"metadata,omitempty"`
}

type CreateDemandProfileInput struct {
	RequirementID         *uuid.UUID     `json:"requirement_id,omitempty"`
	ProjectID             *uuid.UUID     `json:"project_id,omitempty"`
	Title                 string         `json:"title"`
	Goal                  string         `json:"goal,omitempty"`
	Status                string         `json:"status,omitempty"`
	AcceptanceCriteria    []any          `json:"acceptance_criteria,omitempty"`
	RequiredCapabilities  []any          `json:"required_capabilities,omitempty"`
	BudgetConstraints     map[string]any `json:"budget_constraints,omitempty"`
	TimeConstraints       map[string]any `json:"time_constraints,omitempty"`
	RiskConstraints       map[string]any `json:"risk_constraints,omitempty"`
	ResourceFitCandidates []any          `json:"resource_fit_candidates,omitempty"`
	Metadata              map[string]any `json:"metadata,omitempty"`
}

type CreatePDCACycleInput struct {
	DemandProfileID *uuid.UUID     `json:"demand_profile_id,omitempty"`
	RequirementID   *uuid.UUID     `json:"requirement_id,omitempty"`
	ProjectID       *uuid.UUID     `json:"project_id,omitempty"`
	Status          string         `json:"status,omitempty"`
	CurrentStage    string         `json:"current_stage,omitempty"`
	Summary         string         `json:"summary,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type CreatePDCAEventInput struct {
	CycleID      uuid.UUID      `json:"cycle_id"`
	Stage        string         `json:"stage"`
	EventType    string         `json:"event_type,omitempty"`
	SourceType   string         `json:"source_type,omitempty"`
	SourceID     *uuid.UUID     `json:"source_id,omitempty"`
	ActorID      *uuid.UUID     `json:"actor_id,omitempty"`
	ActorType    string         `json:"actor_type,omitempty"`
	ResourceRefs []any          `json:"resource_refs,omitempty"`
	CostRefs     []any          `json:"cost_refs,omitempty"`
	Evidence     map[string]any `json:"evidence,omitempty"`
	Decision     string         `json:"decision,omitempty"`
	NextAction   string         `json:"next_action,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

type ListFilter struct {
	Limit        int
	ResourceType string
	Status       string
	CycleID      *uuid.UUID
}
