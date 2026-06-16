package assistant

import (
	"time"

	"github.com/google/uuid"
)

const (
	ModeBusinessProcess = "business_process"
	ModeSelfEvolution   = "self_evolution"

	StatusIdle             = "idle"
	StatusRunning          = "running"
	StatusApprovalRequired = "approval_required"
	StatusCompleted        = "completed"
	StatusFailed           = "failed"

	StepLLM        = "llm"
	StepToolCall   = "tool_call"
	StepToolResult = "tool_result"
	StepMemory     = "memory"
	StepApproval   = "approval"
	StepError      = "error"
)

type Scope struct {
	ModuleKey            string     `json:"module_key"`
	OrganizationID       *uuid.UUID `json:"organization_id,omitempty"`
	DepartmentID         *uuid.UUID `json:"department_id,omitempty"`
	PositionID           *uuid.UUID `json:"position_id,omitempty"`
	PositionAssignmentID *uuid.UUID `json:"position_assignment_id,omitempty"`
	ProjectID            *uuid.UUID `json:"project_id,omitempty"`
	WorkflowID           *uuid.UUID `json:"workflow_id,omitempty"`
	TaskID               *uuid.UUID `json:"task_id,omitempty"`
}

type Session struct {
	ID                   uuid.UUID      `json:"id"`
	Title                string         `json:"title"`
	Mode                 string         `json:"mode"`
	ModuleKey            string         `json:"module_key"`
	Status               string         `json:"status"`
	ActorID              uuid.UUID      `json:"actor_id"`
	ActorType            string         `json:"actor_type"`
	ProviderID           *uuid.UUID     `json:"provider_id,omitempty"`
	PreferredChannelID   *uuid.UUID     `json:"preferred_channel_id,omitempty"`
	ProviderType         string         `json:"provider_type"`
	Model                string         `json:"model"`
	ServiceTier          string         `json:"service_tier,omitempty"`
	ReasoningEffort      string         `json:"reasoning_effort,omitempty"`
	OrganizationID       *uuid.UUID     `json:"organization_id,omitempty"`
	DepartmentID         *uuid.UUID     `json:"department_id,omitempty"`
	PositionID           *uuid.UUID     `json:"position_id,omitempty"`
	PositionAssignmentID *uuid.UUID     `json:"position_assignment_id,omitempty"`
	ProjectID            *uuid.UUID     `json:"project_id,omitempty"`
	WorkflowID           *uuid.UUID     `json:"workflow_id,omitempty"`
	TaskID               *uuid.UUID     `json:"task_id,omitempty"`
	WorkingMemory        map[string]any `json:"working_memory"`
	Metadata             map[string]any `json:"metadata"`
	LastError            string         `json:"last_error,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

type Message struct {
	ID         uuid.UUID      `json:"id"`
	SessionID  uuid.UUID      `json:"session_id"`
	Role       string         `json:"role"`
	Content    string         `json:"content"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
	ToolName   string         `json:"tool_name,omitempty"`
	Metadata   map[string]any `json:"metadata"`
	CreatedAt  time.Time      `json:"created_at"`
}

type Step struct {
	ID                   uuid.UUID      `json:"id"`
	SessionID            uuid.UUID      `json:"session_id"`
	ModuleKey            string         `json:"module_key"`
	OrganizationID       *uuid.UUID     `json:"organization_id,omitempty"`
	DepartmentID         *uuid.UUID     `json:"department_id,omitempty"`
	PositionID           *uuid.UUID     `json:"position_id,omitempty"`
	PositionAssignmentID *uuid.UUID     `json:"position_assignment_id,omitempty"`
	InvocationID         *uuid.UUID     `json:"invocation_id,omitempty"`
	ToolExecutionID      *uuid.UUID     `json:"tool_execution_id,omitempty"`
	ToolApprovalID       *uuid.UUID     `json:"tool_approval_id,omitempty"`
	StepType             string         `json:"step_type"`
	Status               string         `json:"status"`
	Summary              string         `json:"summary"`
	Data                 map[string]any `json:"data"`
	Turn                 int            `json:"turn"`
	CreatedAt            time.Time      `json:"created_at"`
}

type Memory struct {
	ID              uuid.UUID      `json:"id"`
	Scope           Scope          `json:"scope"`
	ActorID         *uuid.UUID     `json:"actor_id,omitempty"`
	ActorType       string         `json:"actor_type,omitempty"`
	MemoryType      string         `json:"memory_type"`
	Title           string         `json:"title"`
	Content         string         `json:"content"`
	Data            map[string]any `json:"data"`
	SourceSessionID *uuid.UUID     `json:"source_session_id,omitempty"`
	SourceStepID    *uuid.UUID     `json:"source_step_id,omitempty"`
	Confidence      float64        `json:"confidence"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type CreateSessionInput struct {
	Title                string         `json:"title"`
	Mode                 string         `json:"mode"`
	ModuleKey            string         `json:"module_key"`
	ProviderID           *uuid.UUID     `json:"provider_id,omitempty"`
	PreferredChannelID   *uuid.UUID     `json:"preferred_channel_id,omitempty"`
	ProviderType         string         `json:"provider_type,omitempty"`
	Model                string         `json:"model,omitempty"`
	ServiceTier          string         `json:"service_tier,omitempty"`
	ReasoningEffort      string         `json:"reasoning_effort,omitempty"`
	OrganizationID       *uuid.UUID     `json:"organization_id,omitempty"`
	DepartmentID         *uuid.UUID     `json:"department_id,omitempty"`
	PositionID           *uuid.UUID     `json:"position_id,omitempty"`
	PositionAssignmentID *uuid.UUID     `json:"position_assignment_id,omitempty"`
	ProjectID            *uuid.UUID     `json:"project_id,omitempty"`
	WorkflowID           *uuid.UUID     `json:"workflow_id,omitempty"`
	TaskID               *uuid.UUID     `json:"task_id,omitempty"`
	Metadata             map[string]any `json:"metadata,omitempty"`
}

type RunInput struct {
	Message            string     `json:"message"`
	ProviderID         *uuid.UUID `json:"provider_id,omitempty"`
	PreferredChannelID *uuid.UUID `json:"preferred_channel_id,omitempty"`
	ProviderType       string     `json:"provider_type,omitempty"`
	Model              string     `json:"model,omitempty"`
	ServiceTier        string     `json:"service_tier,omitempty"`
	ReasoningEffort    string     `json:"reasoning_effort,omitempty"`
}

type RunEvent struct {
	Type    string         `json:"type"`
	Session *Session       `json:"session,omitempty"`
	Step    *Step          `json:"step,omitempty"`
	Message *Message       `json:"message,omitempty"`
	Delta   string         `json:"delta,omitempty"`
	Error   string         `json:"error,omitempty"`
	Done    bool           `json:"done,omitempty"`
	Data    map[string]any `json:"data,omitempty"`
}

type CreateMemoryInput struct {
	Scope           Scope
	ActorID         *uuid.UUID
	ActorType       string
	MemoryType      string
	Title           string
	Content         string
	Data            map[string]any
	SourceSessionID *uuid.UUID
	SourceStepID    *uuid.UUID
	Confidence      float64
}
