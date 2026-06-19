package assistant

import (
	"context"

	"github.com/google/uuid"
)

type AssistantRuntimeRunner interface {
	Run(context.Context, AssistantRunRequest) (<-chan RunEvent, error)
	Resume(context.Context, AssistantResumeRequest) (<-chan RunEvent, error)
}

type AssistantRunRequest struct {
	SessionID uuid.UUID
	ActorID   uuid.UUID
	ActorType string
	Input     RunInput
}

type AssistantResumeRequest struct {
	SessionID uuid.UUID
	ActorID   uuid.UUID
	ActorType string
	Input     ResumeInput
}

type ContextPackageBuilder interface {
	BuildContextPackage(context.Context, ContextRequest) (*ContextPackage, error)
}

type AssistantRuntime struct {
	service       *Service
	contextEngine ContextPackageBuilder
	toolRunner    *ToolRunner
	eventSink     EventSink
}

func NewAssistantRuntime(service *Service, contextEngine ContextPackageBuilder, toolRunner *ToolRunner, eventSink EventSink) *AssistantRuntime {
	return &AssistantRuntime{service: service, contextEngine: contextEngine, toolRunner: toolRunner, eventSink: eventSink}
}

func (r *AssistantRuntime) Run(ctx context.Context, request AssistantRunRequest) (<-chan RunEvent, error) {
	session, err := r.service.repo.GetSession(ctx, request.SessionID, request.ActorID, request.ActorType)
	if err != nil {
		return nil, err
	}
	if r.contextEngine != nil {
		pkg, err := r.contextEngine.BuildContextPackage(ctx, ContextRequest{
			SessionID:      request.SessionID,
			ActorID:        request.ActorID,
			ActorType:      request.ActorType,
			OrganizationID: session.OrganizationID,
			ModuleKey:      session.ModuleKey,
			WorkflowID:     session.WorkflowID,
			TaskID:         session.TaskID,
			TargetType:     session.TargetType,
			TargetID:       session.TargetID,
			Intent:         request.Input.Message,
			Mode:           session.Mode,
			TokenBudget:    4096,
		})
		if err != nil {
			return nil, err
		}
		if r.eventSink != nil && pkg != nil {
			_, _ = r.eventSink.Emit(ctx, session, AssistantRuntimeEvent{
				Type:    RuntimeEventContextBuilt,
				Status:  StatusCompleted,
				Summary: "Verified context package built",
				Data: map[string]any{
					"context_package_id":   pkg.ID.String(),
					"attention_core_count": len(pkg.AttentionCore),
				},
			})
		}
	}
	return r.service.runLegacy(ctx, request.SessionID, request.ActorID, request.ActorType, request.Input)
}

func (r *AssistantRuntime) Resume(ctx context.Context, request AssistantResumeRequest) (<-chan RunEvent, error) {
	return r.service.resumeLegacy(ctx, request.SessionID, request.ActorID, request.ActorType, request.Input)
}
