package evolution

import (
	"time"

	"github.com/google/uuid"
)

type DecisionWeight struct {
	ID               uuid.UUID `json:"id"`
	ActorID          uuid.UUID `json:"actor_id"`
	ActorType        string    `json:"actor_type"`
	OverallScore     float64   `json:"overall_score"`
	ExpertiseScore   float64   `json:"expertise_score"`
	TrackRecordScore float64   `json:"track_record_score"`
	ReliabilityScore float64   `json:"reliability_score"`
	RecencyScore     float64   `json:"recency_score"`
	ContextFitScore  float64   `json:"context_fit_score"`
	PrincipleScore   float64   `json:"principle_score"`
	DecisionCount    int       `json:"decision_count"`
	LastUpdated      time.Time `json:"last_updated"`
}

type ContextWeightScope struct {
	OrganizationID     *uuid.UUID     `json:"organization_id,omitempty"`
	DepartmentID       *uuid.UUID     `json:"department_id,omitempty"`
	WorkflowTemplateID *uuid.UUID     `json:"workflow_template_id,omitempty"`
	WorkflowStage      string         `json:"workflow_stage,omitempty"`
	TaskType           string         `json:"task_type,omitempty"`
	CapabilityID       *uuid.UUID     `json:"capability_id,omitempty"`
	RiskLevel          string         `json:"risk_level,omitempty"`
	Context            map[string]any `json:"context,omitempty"`
}

type ContextDecisionWeight struct {
	DecisionWeight
	ScopeHash          string         `json:"scope_hash"`
	OrganizationID     *uuid.UUID     `json:"organization_id,omitempty"`
	DepartmentID       *uuid.UUID     `json:"department_id,omitempty"`
	WorkflowTemplateID *uuid.UUID     `json:"workflow_template_id,omitempty"`
	WorkflowStage      string         `json:"workflow_stage,omitempty"`
	TaskType           string         `json:"task_type,omitempty"`
	CapabilityID       *uuid.UUID     `json:"capability_id,omitempty"`
	RiskLevel          string         `json:"risk_level"`
	Context            map[string]any `json:"context"`
}

type AlphaConfig struct {
	ID          uuid.UUID `json:"id"`
	Expertise   float64   `json:"alpha_expertise"`
	TrackRecord float64   `json:"alpha_track_record"`
	Reliability float64   `json:"alpha_reliability"`
	Recency     float64   `json:"alpha_recency"`
	ContextFit  float64   `json:"alpha_context_fit"`
	Principle   float64   `json:"alpha_principle"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"created_at"`
}

type Experiment struct {
	ID              uuid.UUID      `json:"id"`
	Name            string         `json:"name"`
	Hypothesis      string         `json:"hypothesis"`
	Status          string         `json:"status"`
	MVRUID          *uuid.UUID     `json:"mvru_id,omitempty"`
	AlphaOverrides  map[string]any `json:"alpha_overrides,omitempty"`
	SuccessCriteria map[string]any `json:"success_criteria"`
	StartedAt       *time.Time     `json:"started_at,omitempty"`
	CompletedAt     *time.Time     `json:"completed_at,omitempty"`
	Conclusion      string         `json:"conclusion,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
}

type KnowledgeEntry struct {
	ID         uuid.UUID  `json:"id"`
	WorkflowID *uuid.UUID `json:"workflow_id,omitempty"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	Tags       []string   `json:"tags"`
	Source     string     `json:"source"`
	CreatedAt  time.Time  `json:"created_at"`
}

type Signal struct {
	ID           uuid.UUID      `json:"id"`
	SignalType   string         `json:"signal_type"`
	Source       string         `json:"source"`
	Priority     int            `json:"priority"`
	Data         map[string]any `json:"data"`
	Acknowledged bool           `json:"acknowledged"`
	CreatedAt    time.Time      `json:"created_at"`
}

type WeightInput struct {
	ActorID       uuid.UUID      `json:"actor_id"`
	ActorType     string         `json:"actor_type"`
	TaskContext   map[string]any `json:"task_context,omitempty"`
	RequiredLevel string         `json:"required_level,omitempty"`
}

type ContextWeightInput struct {
	ActorID       uuid.UUID          `json:"actor_id"`
	ActorType     string             `json:"actor_type"`
	Scope         ContextWeightScope `json:"scope"`
	RequiredLevel string             `json:"required_level,omitempty"`
}

type OutcomeInput struct {
	ActorID      uuid.UUID      `json:"actor_id"`
	ActorType    string         `json:"actor_type"`
	OutcomeScore float64        `json:"outcome_score"`
	TaskContext  map[string]any `json:"task_context,omitempty"`
}

type ContextOutcomeInput struct {
	ActorID      uuid.UUID          `json:"actor_id"`
	ActorType    string             `json:"actor_type"`
	Scope        ContextWeightScope `json:"scope"`
	OutcomeScore float64            `json:"outcome_score"`
}

type CreateExperimentInput struct {
	Name            string         `json:"name"`
	Hypothesis      string         `json:"hypothesis"`
	MVRUID          *uuid.UUID     `json:"mvru_id,omitempty"`
	AlphaOverrides  map[string]any `json:"alpha_overrides,omitempty"`
	SuccessCriteria map[string]any `json:"success_criteria,omitempty"`
}

type CreateKnowledgeInput struct {
	WorkflowID *uuid.UUID `json:"workflow_id,omitempty"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	Tags       []string   `json:"tags,omitempty"`
	Source     string     `json:"source"`
}

type CreateSignalInput struct {
	SignalType string         `json:"signal_type"`
	Source     string         `json:"source"`
	Priority   int            `json:"priority"`
	Data       map[string]any `json:"data,omitempty"`
}
