package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/selfevo-AI/meta-org/backend/internal/domain/aigateway"
	"github.com/selfevo-AI/meta-org/backend/internal/domain/toolruntime"
)

var (
	ErrValidation = errors.New("validation error")
	ErrNotFound   = errors.New("not found")
)

type AIInvoker interface {
	Invoke(context.Context, aigateway.InvokeInput) (*aigateway.InvokeOutput, error)
}

type ToolExecutor interface {
	ExecuteTool(context.Context, toolruntime.ExecuteToolInput) (*toolruntime.ExecuteToolOutput, error)
	ListTools(context.Context, int) ([]toolruntime.ToolDefinition, error)
}

type Service struct {
	repo       Repository
	ai         AIInvoker
	tools      ToolExecutor
	maxTurns   int
	maxHistory int
}

func NewService(repo Repository, ai AIInvoker, tools ToolExecutor) *Service {
	return &Service{repo: repo, ai: ai, tools: tools, maxTurns: 12, maxHistory: 40}
}

func (s *Service) CreateSession(ctx context.Context, actorID uuid.UUID, actorType string, input CreateSessionInput) (*Session, error) {
	input.ModuleKey = normalizedModule(input.ModuleKey)
	if input.Mode == "" {
		input.Mode = ModeBusinessProcess
	}
	if input.Mode != ModeBusinessProcess && input.Mode != ModeSelfEvolution {
		return nil, fmt.Errorf("%w: unsupported assistant mode", ErrValidation)
	}
	if actorID == uuid.Nil || actorType == "" {
		return nil, fmt.Errorf("%w: actor is required", ErrValidation)
	}
	return s.repo.CreateSession(ctx, actorID, actorType, input)
}

func (s *Service) ListSessions(ctx context.Context, actorID uuid.UUID, actorType string, moduleKey string, limit int) ([]Session, error) {
	return s.repo.ListSessions(ctx, actorID, actorType, normalizedOptionalModule(moduleKey), limit)
}

func (s *Service) GetSession(ctx context.Context, id uuid.UUID, actorID uuid.UUID, actorType string) (*Session, error) {
	return s.repo.GetSession(ctx, id, actorID, actorType)
}

func (s *Service) ListSteps(ctx context.Context, sessionID uuid.UUID, actorID uuid.UUID, actorType string, limit int) ([]Step, error) {
	if _, err := s.repo.GetSession(ctx, sessionID, actorID, actorType); err != nil {
		return nil, err
	}
	return s.repo.ListSteps(ctx, sessionID, limit)
}

func (s *Service) Run(ctx context.Context, sessionID uuid.UUID, actorID uuid.UUID, actorType string, input RunInput) (<-chan RunEvent, error) {
	if strings.TrimSpace(input.Message) == "" {
		return nil, fmt.Errorf("%w: message is required", ErrValidation)
	}
	if s.ai == nil {
		return nil, fmt.Errorf("%w: ai gateway is not configured", ErrValidation)
	}
	if s.tools == nil {
		return nil, fmt.Errorf("%w: tool runtime is not configured", ErrValidation)
	}
	session, err := s.repo.GetSession(ctx, sessionID, actorID, actorType)
	if err != nil {
		return nil, err
	}
	if err := s.repo.UpdateSessionStatus(ctx, session.ID, StatusRunning, ""); err != nil {
		return nil, err
	}

	events := make(chan RunEvent)
	go s.runLoop(ctx, events, session, input)
	return events, nil
}

func (s *Service) runLoop(ctx context.Context, events chan<- RunEvent, session *Session, input RunInput) {
	defer close(events)
	send := func(event RunEvent) bool {
		select {
		case <-ctx.Done():
			return false
		case events <- event:
			return true
		}
	}
	fail := func(err error, turn int) {
		_ = s.repo.UpdateSessionStatus(context.Background(), session.ID, StatusFailed, err.Error())
		step, _ := s.repo.AddStep(context.Background(), session, AddStepInput{StepType: StepError, Status: StatusFailed, Summary: err.Error(), Turn: turn})
		send(RunEvent{Type: "error", Step: step, Error: err.Error(), Done: true})
	}

	userMessage, err := s.repo.AddMessage(ctx, session.ID, "user", input.Message, "", "", nil)
	if err != nil {
		fail(err, 0)
		return
	}
	if !send(RunEvent{Type: "message", Message: userMessage}) {
		return
	}

	scope := sessionScope(session)
	memories, err := s.repo.ListScopedMemories(ctx, scope, session.ActorID, session.ActorType, 12)
	if err != nil {
		fail(err, 0)
		return
	}
	history, err := s.repo.ListMessages(ctx, session.ID, s.maxHistory)
	if err != nil {
		fail(err, 0)
		return
	}
	toolDefs, err := s.tools.ListTools(ctx, 100)
	if err != nil {
		fail(err, 0)
		return
	}

	messages := buildAIMessages(session, memories, history)
	tools := gatewayTools(toolDefs)
	providerID, channelID, providerType, model, serviceTier, effort := runModelConfig(session, input)
	if (providerID == nil && providerType == "") || model == "" {
		fail(fmt.Errorf("%w: model provider and model are required", ErrValidation), 0)
		return
	}

	for turn := 1; turn <= s.maxTurns; turn++ {
		output, err := s.ai.Invoke(ctx, aigateway.InvokeInput{
			ProviderID:         providerID,
			PreferredChannelID: channelID,
			ProviderType:       providerType,
			Model:              model,
			Messages:           messages,
			MaxTokens:          2048,
			ServiceTier:        serviceTier,
			ReasoningEffort:    effort,
			Tools:              tools,
			Attribution: aigateway.Attribution{
				OrganizationID: session.OrganizationID,
				DepartmentID:   session.DepartmentID,
				ProjectID:      session.ProjectID,
				WorkflowID:     session.WorkflowID,
				TaskID:         session.TaskID,
				UserID:         actorIDForAttribution(session),
				SourceSurface:  "assistant:" + session.ModuleKey,
			},
			Metadata: map[string]any{
				"assistant_session_id":   session.ID.String(),
				"module_key":             session.ModuleKey,
				"position_id":            uuidString(session.PositionID),
				"position_assignment_id": uuidString(session.PositionAssignmentID),
			},
		})
		if err != nil {
			fail(err, turn)
			return
		}

		assistantMessage := aigateway.Message{Role: "assistant", Content: output.Content, ToolCalls: output.ToolCalls}
		messages = append(messages, assistantMessage)
		stored, err := s.repo.AddMessage(ctx, session.ID, "assistant", output.Content, "", "", map[string]any{"tool_calls": output.ToolCalls})
		if err != nil {
			fail(err, turn)
			return
		}
		llmStep, err := s.repo.AddStep(ctx, session, AddStepInput{
			InvocationID: &output.InvocationID,
			StepType:     StepLLM,
			Status:       StatusCompleted,
			Summary:      trimSummary(output.Content),
			Data:         map[string]any{"tool_call_count": len(output.ToolCalls), "model": output.Model, "provider_type": output.ProviderType},
			Turn:         turn,
		})
		if err != nil {
			fail(err, turn)
			return
		}
		if output.Content != "" && !send(RunEvent{Type: "message", Message: stored, Step: llmStep, Delta: output.Content}) {
			return
		}
		if len(output.ToolCalls) == 0 {
			s.finishRun(ctx, send, session, output.Content, llmStep)
			return
		}

		for index, call := range output.ToolCalls {
			callID := toolCallID(call, index)
			callStep, err := s.repo.AddStep(ctx, session, AddStepInput{
				InvocationID: &output.InvocationID,
				StepType:     StepToolCall,
				Status:       "requested",
				Summary:      call.Name,
				Data:         map[string]any{"tool_call_id": callID, "tool_name": call.Name, "arguments": call.Arguments},
				Turn:         turn,
			})
			if err != nil {
				fail(err, turn)
				return
			}
			if !send(RunEvent{Type: "tool_call", Step: callStep, Data: callStep.Data}) {
				return
			}
			result, err := s.tools.ExecuteTool(ctx, toolruntime.ExecuteToolInput{
				ToolName:       call.Name,
				InvocationID:   &output.InvocationID,
				ActorID:        session.ActorID,
				ActorType:      session.ActorType,
				OrganizationID: session.OrganizationID,
				DepartmentID:   session.DepartmentID,
				ProjectID:      session.ProjectID,
				WorkflowID:     session.WorkflowID,
				TaskID:         session.TaskID,
				IdempotencyKey: fmt.Sprintf("assistant:%s:%s:%d", session.ID, callID, turn),
				Arguments:      call.Arguments,
			})
			if err != nil {
				fail(err, turn)
				return
			}
			status := result.Execution.Status
			if result.Approval != nil {
				step, _ := s.repo.AddStep(ctx, session, AddStepInput{
					InvocationID:    &output.InvocationID,
					ToolExecutionID: &result.Execution.ID,
					ToolApprovalID:  &result.Approval.ID,
					StepType:        StepApproval,
					Status:          StatusApprovalRequired,
					Summary:         result.Execution.ResultSummary,
					Data:            map[string]any{"tool_call_id": callID, "tool_name": call.Name, "approval_id": result.Approval.ID.String()},
					Turn:            turn,
				})
				_ = s.repo.UpdateSessionStatus(ctx, session.ID, StatusApprovalRequired, "")
				send(RunEvent{Type: "approval_required", Step: step, Done: true, Data: step.Data})
				return
			}
			payload := map[string]any{"status": status, "summary": result.Execution.ResultSummary, "result": result.Execution.Result, "error": result.Execution.ErrorMessage}
			resultStep, err := s.repo.AddStep(ctx, session, AddStepInput{
				InvocationID:    &output.InvocationID,
				ToolExecutionID: &result.Execution.ID,
				StepType:        StepToolResult,
				Status:          status,
				Summary:         result.Execution.ResultSummary,
				Data:            payload,
				Turn:            turn,
			})
			if err != nil {
				fail(err, turn)
				return
			}
			if !send(RunEvent{Type: "tool_result", Step: resultStep, Data: payload}) {
				return
			}
			toolContent := marshalToolContent(payload)
			messages = append(messages, aigateway.Message{Role: "tool", Content: toolContent, ToolCallID: callID, ToolName: call.Name})
			if _, err := s.repo.AddMessage(ctx, session.ID, "tool", toolContent, callID, call.Name, payload); err != nil {
				fail(err, turn)
				return
			}
		}
	}
	fail(fmt.Errorf("%w: assistant reached max turns", ErrValidation), s.maxTurns)
}

func (s *Service) finishRun(ctx context.Context, send func(RunEvent) bool, session *Session, summary string, sourceStep *Step) {
	memory := copyMap(session.WorkingMemory)
	memory["last_summary"] = summary
	memory["last_module_key"] = session.ModuleKey
	memory["last_position_id"] = uuidString(session.PositionID)
	_ = s.repo.UpdateWorkingMemory(ctx, session.ID, memory)
	sourceStepID := sourceStep.ID
	created, err := s.repo.CreateMemory(ctx, CreateMemoryInput{
		Scope:           sessionScope(session),
		ActorID:         &session.ActorID,
		ActorType:       session.ActorType,
		MemoryType:      "lesson",
		Title:           "Assistant run summary",
		Content:         trimSummary(summary),
		Data:            map[string]any{"session_id": session.ID.String(), "mode": session.Mode},
		SourceSessionID: &session.ID,
		SourceStepID:    &sourceStepID,
		Confidence:      1,
	})
	if err == nil {
		step, _ := s.repo.AddStep(ctx, session, AddStepInput{
			StepType: StepMemory,
			Status:   StatusCompleted,
			Summary:  created.Title,
			Data:     map[string]any{"memory_id": created.ID.String(), "module_key": session.ModuleKey, "position_id": uuidString(session.PositionID)},
		})
		send(RunEvent{Type: "memory_updated", Step: step, Data: step.Data})
	}
	_ = s.repo.UpdateSessionStatus(ctx, session.ID, StatusCompleted, "")
	send(RunEvent{Type: "done", Done: true, Data: map[string]any{"status": StatusCompleted}})
}

func buildAIMessages(session *Session, memories []Memory, history []Message) []aigateway.Message {
	messages := []aigateway.Message{{Role: "system", Content: systemPrompt(session, memories)}}
	for _, item := range history {
		msg := aigateway.Message{Role: item.Role, Content: item.Content, ToolCallID: item.ToolCallID, ToolName: item.ToolName}
		if item.Role == "assistant" {
			if calls, ok := item.Metadata["tool_calls"]; ok {
				data, _ := json.Marshal(calls)
				_ = json.Unmarshal(data, &msg.ToolCalls)
			}
		}
		messages = append(messages, msg)
	}
	return messages
}

func systemPrompt(session *Session, memories []Memory) string {
	var b strings.Builder
	b.WriteString("You are a business process and system self-evolution assistant for this organization.\n")
	b.WriteString("Use tools when they are needed to analyze requirements, operate project workflows, prepare governance context, or create evolution knowledge/signals.\n")
	b.WriteString("Memory isolation rule: only use memories and work records from the exact module_key and organization position scope shown here. Do not infer or import context from other modules or positions.\n")
	b.WriteString(fmt.Sprintf("mode=%s module_key=%s organization_id=%s department_id=%s position_id=%s position_assignment_id=%s\n",
		session.Mode, session.ModuleKey, uuidString(session.OrganizationID), uuidString(session.DepartmentID),
		uuidString(session.PositionID), uuidString(session.PositionAssignmentID)))
	if len(memories) > 0 {
		b.WriteString("Scoped memories:\n")
		for _, memory := range memories {
			b.WriteString("- ")
			b.WriteString(memory.Title)
			if memory.Content != "" {
				b.WriteString(": ")
				b.WriteString(memory.Content)
			}
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func gatewayTools(tools []toolruntime.ToolDefinition) []aigateway.ToolDefinition {
	result := []aigateway.ToolDefinition{}
	for _, tool := range tools {
		if !tool.IsActive {
			continue
		}
		result = append(result, aigateway.ToolDefinition{Name: tool.Name, Description: tool.Description, Schema: tool.InputSchema})
	}
	return result
}

func runModelConfig(session *Session, input RunInput) (*uuid.UUID, *uuid.UUID, string, string, string, string) {
	providerID := session.ProviderID
	if input.ProviderID != nil {
		providerID = input.ProviderID
	}
	channelID := session.PreferredChannelID
	if input.PreferredChannelID != nil {
		channelID = input.PreferredChannelID
	}
	providerType := firstNonEmpty(input.ProviderType, session.ProviderType)
	model := firstNonEmpty(input.Model, session.Model)
	serviceTier := firstNonEmpty(input.ServiceTier, session.ServiceTier)
	effort := firstNonEmpty(input.ReasoningEffort, session.ReasoningEffort)
	return providerID, channelID, providerType, model, serviceTier, effort
}

func sessionScope(session *Session) Scope {
	return Scope{
		ModuleKey:            session.ModuleKey,
		OrganizationID:       session.OrganizationID,
		DepartmentID:         session.DepartmentID,
		PositionID:           session.PositionID,
		PositionAssignmentID: session.PositionAssignmentID,
		ProjectID:            session.ProjectID,
		WorkflowID:           session.WorkflowID,
		TaskID:               session.TaskID,
	}
}

func normalizedModule(value string) string {
	if strings.TrimSpace(value) == "" {
		return "general"
	}
	return strings.TrimSpace(value)
}

func normalizedOptionalModule(value string) string {
	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func trimSummary(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 600 {
		return value[:600]
	}
	return value
}

func marshalToolContent(input map[string]any) string {
	data, err := json.Marshal(input)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func toolCallID(call aigateway.ToolCall, index int) string {
	if call.ID != "" {
		return call.ID
	}
	return fmt.Sprintf("tool_call_%d", index+1)
}

func actorIDForAttribution(session *Session) *uuid.UUID {
	if session.ActorType == "human" {
		return &session.ActorID
	}
	return nil
}

func uuidString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func copyMap(input map[string]any) map[string]any {
	output := map[string]any{}
	for key, value := range input {
		output[key] = value
	}
	return output
}
