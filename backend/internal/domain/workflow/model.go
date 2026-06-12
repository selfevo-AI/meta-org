package workflow

import (
	"time"

	"github.com/google/uuid"
)

type WorkflowStatus string

const (
	WorkflowActive    WorkflowStatus = "active"
	WorkflowPaused    WorkflowStatus = "paused"
	WorkflowCompleted WorkflowStatus = "completed"
	WorkflowFailed    WorkflowStatus = "failed"
)

type TaskStatus string

const (
	TaskPending     TaskStatus = "pending"
	TaskAssigned    TaskStatus = "assigned"
	TaskInProgress  TaskStatus = "in_progress"
	TaskCompleted   TaskStatus = "completed"
	TaskRejected    TaskStatus = "rejected"
)

type StageType string

const (
	StagePlan    StageType = "plan"
	StageExecute StageType = "execute"
	StageReview  StageType = "review"
)

type Stage struct {
	Type          StageType `json:"type"`
	Name          string    `json:"name"`
	AssigneeType  string    `json:"assignee_type"`
	RequiredTools []string  `json:"required_tools,omitempty"`
}

type WorkflowTemplate struct {
	ID             uuid.UUID              `json:"id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	Stages         []Stage                `json:"stages"`
	AssigneeType   string                 `json:"assignee_type"`
	RequiredWeight float64                `json:"required_weight"`
	RoutingRules   map[string]any         `json:"routing_rules"`
	IsActive       bool                   `json:"is_active"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type WorkflowInstance struct {
	ID           uuid.UUID              `json:"id"`
	TemplateID   uuid.UUID              `json:"template_id"`
	Status       WorkflowStatus         `json:"status"`
	CurrentStage int                    `json:"current_stage"`
	Context      map[string]any         `json:"context"`
	TraceID      *uuid.UUID             `json:"trace_id,omitempty"`
	Tasks        []Task                 `json:"tasks,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type Task struct {
	ID             uuid.UUID              `json:"id"`
	WorkflowID     uuid.UUID              `json:"workflow_id"`
	Stage          int                    `json:"stage"`
	StageType      StageType              `json:"stage_type"`
	AssigneeID     *uuid.UUID             `json:"assignee_id,omitempty"`
	AssigneeType   string                 `json:"assignee_type,omitempty"`
	Input          map[string]any         `json:"input"`
	Output         map[string]any         `json:"output"`
	WeightSnapshot float64                `json:"weight_snapshot"`
	Status         TaskStatus             `json:"status"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type Decision struct {
	ID              uuid.UUID              `json:"id"`
	TaskID          uuid.UUID              `json:"task_id"`
	DecisionMakerID uuid.UUID              `json:"decision_maker_id"`
	MakerType       string                 `json:"maker_type"`
	Weight          float64                `json:"weight"`
	Input           map[string]any         `json:"input"`
	Output          map[string]any         `json:"output"`
	Reasoning       string                 `json:"reasoning"`
	Outcome         string                 `json:"outcome"`
	CreatedAt       time.Time              `json:"created_at"`
}

type WorkflowContext struct {
	ID                 uuid.UUID              `json:"id"`
	WorkflowID         uuid.UUID              `json:"workflow_id"`
	WorkingMemory      map[string]any         `json:"working_memory"`
	InjectedExperience []map[string]any       `json:"injected_experience"`
	PrincipleNotes     string                 `json:"principle_notes,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`
}

type CreateWorkflowInput struct {
	Name           string         `json:"name"`
	Description    string         `json:"description,omitempty"`
	Stages         []Stage        `json:"stages"`
	AssigneeType   string         `json:"assignee_type"`
	RequiredWeight float64        `json:"required_weight"`
	RoutingRules   map[string]any `json:"routing_rules,omitempty"`
}

type StartWorkflowInput struct {
	TemplateID uuid.UUID              `json:"template_id"`
	Context    map[string]any         `json:"context,omitempty"`
}
