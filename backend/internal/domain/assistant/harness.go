package assistant

import "github.com/google/uuid"

type AssistantHarnessInput struct {
	SessionID        uuid.UUID
	ActorID          uuid.UUID
	ActorType        string
	OrganizationID   *uuid.UUID
	DepartmentID     *uuid.UUID
	PositionID       *uuid.UUID
	ProjectID        *uuid.UUID
	WorkflowID       *uuid.UUID
	TaskID           *uuid.UUID
	ModuleKey        string
	TargetType       string
	TargetID         *uuid.UUID
	ContextPackageID uuid.UUID
	ProviderID       *uuid.UUID
	ChannelID        *uuid.UUID
	ProviderType     string
	Model            string
	ServiceTier      string
	ReasoningEffort  string
}

type AssistantHarness struct {
	SessionID        uuid.UUID
	ActorID          uuid.UUID
	ActorType        string
	OrganizationID   *uuid.UUID
	DepartmentID     *uuid.UUID
	PositionID       *uuid.UUID
	ProjectID        *uuid.UUID
	WorkflowID       *uuid.UUID
	TaskID           *uuid.UUID
	ModuleKey        string
	TargetType       string
	TargetID         *uuid.UUID
	ContextPackageID uuid.UUID
	ProviderID       *uuid.UUID
	ChannelID        *uuid.UUID
	ProviderType     string
	Model            string
	ServiceTier      string
	ReasoningEffort  string
}

func NewAssistantHarness(input AssistantHarnessInput) AssistantHarness {
	return AssistantHarness{
		SessionID:        input.SessionID,
		ActorID:          input.ActorID,
		ActorType:        input.ActorType,
		OrganizationID:   cloneUUID(input.OrganizationID),
		DepartmentID:     cloneUUID(input.DepartmentID),
		PositionID:       cloneUUID(input.PositionID),
		ProjectID:        cloneUUID(input.ProjectID),
		WorkflowID:       cloneUUID(input.WorkflowID),
		TaskID:           cloneUUID(input.TaskID),
		ModuleKey:        normalizedModule(input.ModuleKey),
		TargetType:       input.TargetType,
		TargetID:         cloneUUID(input.TargetID),
		ContextPackageID: input.ContextPackageID,
		ProviderID:       cloneUUID(input.ProviderID),
		ChannelID:        cloneUUID(input.ChannelID),
		ProviderType:     input.ProviderType,
		Model:            input.Model,
		ServiceTier:      input.ServiceTier,
		ReasoningEffort:  input.ReasoningEffort,
	}
}

func cloneUUID(id *uuid.UUID) *uuid.UUID {
	if id == nil {
		return nil
	}
	copied := *id
	return &copied
}
