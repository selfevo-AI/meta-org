package assistant

import "context"

type AssistantRuntimeEvent struct {
	Type    string
	Status  string
	Summary string
	Data    map[string]any
	Turn    int
}

type EventSink interface {
	Emit(context.Context, *Session, AssistantRuntimeEvent) (RunEvent, error)
}

type MemoryEventSink struct {
	repo Repository
}

func NewMemoryEventSink(repo Repository) *MemoryEventSink {
	return &MemoryEventSink{repo: repo}
}

func (s *MemoryEventSink) Emit(ctx context.Context, session *Session, event AssistantRuntimeEvent) (RunEvent, error) {
	stepType := runtimeEventStepType(event.Type)
	step, err := s.repo.AddStep(ctx, session, AddStepInput{
		StepType: stepType,
		Status:   firstNonEmpty(event.Status, StatusCompleted),
		Summary:  event.Summary,
		Data:     event.Data,
		Turn:     event.Turn,
	})
	if err != nil {
		return RunEvent{}, err
	}
	return RunEvent{Type: event.Type, Step: step, Data: event.Data}, nil
}

func runtimeEventStepType(eventType string) string {
	switch eventType {
	case RuntimeEventContextBuilt:
		return StepContext
	case RuntimeEventToolRequested:
		return StepToolCall
	case RuntimeEventToolCompleted:
		return StepToolResult
	case RuntimeEventApprovalRequired:
		return StepApproval
	case RuntimeEventRunFailed:
		return StepError
	default:
		return StepLLM
	}
}
