package governance

import (
	"time"

	"github.com/google/uuid"
)

type Permission struct {
	ID          uuid.UUID `json:"id"`
	Level       int       `json:"level"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Behavior    string    `json:"behavior"`
	CreatedAt   time.Time `json:"created_at"`
}

type Principle struct {
	ID              uuid.UUID      `json:"id"`
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	EvaluationLogic map[string]any `json:"evaluation_logic"`
	Priority        int            `json:"priority"`
	IsActive        bool           `json:"is_active"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type ControlRule struct {
	ID               uuid.UUID      `json:"id"`
	PrincipleID      *uuid.UUID     `json:"principle_id,omitempty"`
	TargetEntityType string         `json:"target_entity_type"`
	TargetEntityID   *uuid.UUID     `json:"target_entity_id,omitempty"`
	Condition        map[string]any `json:"condition"`
	Action           string         `json:"action"`
	Priority         int            `json:"priority"`
	IsActive         bool           `json:"is_active"`
	CreatedAt        time.Time      `json:"created_at"`
}

type CreatePrincipleInput struct {
	Name            string         `json:"name"`
	Description     string         `json:"description"`
	EvaluationLogic map[string]any `json:"evaluation_logic,omitempty"`
	Priority        int            `json:"priority,omitempty"`
}

type CreateControlRuleInput struct {
	PrincipleID      *uuid.UUID     `json:"principle_id,omitempty"`
	TargetEntityType string         `json:"target_entity_type"`
	TargetEntityID   *uuid.UUID     `json:"target_entity_id,omitempty"`
	Condition        map[string]any `json:"condition,omitempty"`
	Action           string         `json:"action"`
	Priority         int            `json:"priority,omitempty"`
}

type PermissionCheckInput struct {
	UserID     uuid.UUID  `json:"user_id"`
	Action     string     `json:"action"`
	Resource   string     `json:"resource"`
	ResourceID *uuid.UUID `json:"resource_id,omitempty"`
}

type PermissionCheckResult struct {
	Allowed  bool   `json:"allowed"`
	Level    int    `json:"level"`
	Behavior string `json:"behavior"`
	Reason   string `json:"reason"`
}

type AccessDecisionInput struct {
	ActorID        uuid.UUID      `json:"actor_id"`
	ActorType      string         `json:"actor_type"`
	Action         string         `json:"action"`
	Resource       string         `json:"resource"`
	ResourceID     *uuid.UUID     `json:"resource_id,omitempty"`
	OrganizationID *uuid.UUID     `json:"organization_id,omitempty"`
	DepartmentID   *uuid.UUID     `json:"department_id,omitempty"`
	WorkflowID     *uuid.UUID     `json:"workflow_id,omitempty"`
	TaskID         *uuid.UUID     `json:"task_id,omitempty"`
	CapabilityID   *uuid.UUID     `json:"capability_id,omitempty"`
	RequiredLevel  string         `json:"required_level,omitempty"`
	RiskLevel      string         `json:"risk_level,omitempty"`
	WeightSnapshot *float64       `json:"weight_snapshot,omitempty"`
	Context        map[string]any `json:"context,omitempty"`
}

type AccessDecision struct {
	ID             uuid.UUID      `json:"id"`
	ActorID        uuid.UUID      `json:"actor_id"`
	ActorType      string         `json:"actor_type"`
	Action         string         `json:"action"`
	Resource       string         `json:"resource"`
	ResourceID     *uuid.UUID     `json:"resource_id,omitempty"`
	OrganizationID *uuid.UUID     `json:"organization_id,omitempty"`
	DepartmentID   *uuid.UUID     `json:"department_id,omitempty"`
	WorkflowID     *uuid.UUID     `json:"workflow_id,omitempty"`
	TaskID         *uuid.UUID     `json:"task_id,omitempty"`
	CapabilityID   *uuid.UUID     `json:"capability_id,omitempty"`
	RequiredLevel  string         `json:"required_level"`
	RiskLevel      string         `json:"risk_level"`
	Decision       string         `json:"decision"`
	Allowed        bool           `json:"allowed"`
	Behavior       string         `json:"behavior"`
	Reason         string         `json:"reason"`
	MatchedRules   []string       `json:"matched_rules"`
	WeightSnapshot *float64       `json:"weight_snapshot,omitempty"`
	Context        map[string]any `json:"context"`
	CreatedAt      time.Time      `json:"created_at"`
}
