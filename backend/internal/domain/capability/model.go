package capability

import (
	"time"

	"github.com/google/uuid"
)

type Capability struct {
	ID              uuid.UUID              `json:"id"`
	Name            string                 `json:"name"`
	Version         string                 `json:"version"`
	Description     string                 `json:"description,omitempty"`
	InputSchema     map[string]any         `json:"input_schema"`
	OutputSchema    map[string]any         `json:"output_schema"`
	Preconditions   []string               `json:"preconditions"`
	ErrorHandling   map[string]any         `json:"error_handling"`
	PermissionLevel string                 `json:"permission_level"`
	CostEstimate    map[string]any         `json:"cost_estimate"`
	IsActive        bool                   `json:"is_active"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type CapabilityBinding struct {
	ID            uuid.UUID              `json:"id"`
	CapabilityID  uuid.UUID              `json:"capability_id"`
	MVRUID        uuid.UUID              `json:"mvru_id"`
	Config        map[string]any         `json:"config"`
	CreatedAt     time.Time              `json:"created_at"`
}

type CapabilityInvocation struct {
	ID            uuid.UUID              `json:"id"`
	CapabilityID  uuid.UUID              `json:"capability_id"`
	CallerID      uuid.UUID              `json:"caller_id"`
	CallerType    string                 `json:"caller_type"`
	Input         map[string]any         `json:"input"`
	Output        map[string]any         `json:"output"`
	DurationMs    int                    `json:"duration_ms"`
	Cost          float64                `json:"cost"`
	Outcome       string                 `json:"outcome"`
	TraceID       *uuid.UUID             `json:"trace_id,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
}

type CreateCapabilityInput struct {
	Name            string         `json:"name"`
	Version         string         `json:"version,omitempty"`
	Description     string         `json:"description,omitempty"`
	InputSchema     map[string]any `json:"input_schema,omitempty"`
	OutputSchema    map[string]any `json:"output_schema,omitempty"`
	Preconditions   []string       `json:"preconditions,omitempty"`
	ErrorHandling   map[string]any `json:"error_handling,omitempty"`
	PermissionLevel string         `json:"permission_level,omitempty"`
	CostEstimate    map[string]any `json:"cost_estimate,omitempty"`
}

type BindCapabilityInput struct {
	CapabilityID uuid.UUID              `json:"capability_id"`
	MVRUID       uuid.UUID              `json:"mvru_id"`
	Config       map[string]any         `json:"config,omitempty"`
}
