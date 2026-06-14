package toolruntime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/selfevo-AI/meta-org/backend/internal/domain/governance"
)

var (
	ErrValidation = errors.New("validation error")
	ErrNotFound   = errors.New("not found")
)

type ToolAdapter func(context.Context, ExecuteToolInput) (ToolResult, error)

type Repository interface {
	CreateTool(ctx context.Context, input CreateToolInput) (*ToolDefinition, error)
	ListTools(ctx context.Context, limit int) ([]ToolDefinition, error)
	UpdateTool(ctx context.Context, id uuid.UUID, input UpdateToolInput) (*ToolDefinition, error)
	GetToolByID(ctx context.Context, id uuid.UUID) (*ToolDefinition, error)
	GetToolByName(ctx context.Context, name string) (*ToolDefinition, error)
	CreateExecution(ctx context.Context, input CreateExecutionInput) (*ToolExecution, error)
	CompleteExecution(ctx context.Context, id uuid.UUID, input CompleteExecutionInput) (*ToolExecution, error)
	CreateApproval(ctx context.Context, executionID uuid.UUID, requestedBy *uuid.UUID, reason string) (*ToolApproval, error)
	ListExecutions(ctx context.Context, limit int) ([]ToolExecution, error)
	GetExecution(ctx context.Context, id uuid.UUID) (*ToolExecution, error)
	UpdateApproval(ctx context.Context, id uuid.UUID, status string, reviewedBy *uuid.UUID, reason string) (*ToolApproval, error)
}

type GovernanceService interface {
	DecideAccess(context.Context, governance.AccessDecisionInput) (*governance.AccessDecision, error)
}

type Service struct {
	repo       Repository
	governance GovernanceService
	adapters   map[string]ToolAdapter
}

func NewService(repo Repository, governanceSvc GovernanceService, adapters map[string]ToolAdapter) *Service {
	if adapters == nil {
		adapters = map[string]ToolAdapter{}
	}
	return &Service{repo: repo, governance: governanceSvc, adapters: adapters}
}

type CreateExecutionInput struct {
	ToolID             uuid.UUID
	InvocationID       *uuid.UUID
	ActorID            uuid.UUID
	ActorType          string
	OrganizationID     *uuid.UUID
	DepartmentID       *uuid.UUID
	ProjectID          *uuid.UUID
	WorkflowID         *uuid.UUID
	TaskID             *uuid.UUID
	IdempotencyKey     string
	Policy             string
	GovernanceDecision string
	Status             string
	Arguments          map[string]any
}

type CompleteExecutionInput struct {
	Status        string
	ResultSummary string
	Result        map[string]any
	ErrorMessage  string
	DurationMS    int
}

func EffectivePolicy(defaultPolicy string, governance GovernanceResult) string {
	if governance.Decision == "deny" {
		return PolicyDeny
	}
	if governance.Decision == "approve" {
		return PolicyApprove
	}
	if governance.Decision == "notify" && defaultPolicy == PolicyAuto {
		return PolicyNotify
	}
	return defaultPolicy
}

func (s *Service) CreateTool(ctx context.Context, input CreateToolInput) (*ToolDefinition, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	return s.repo.CreateTool(ctx, input)
}

func (s *Service) ListTools(ctx context.Context, limit int) ([]ToolDefinition, error) {
	return s.repo.ListTools(ctx, limit)
}

func (s *Service) UpdateTool(ctx context.Context, id uuid.UUID, input UpdateToolInput) (*ToolDefinition, error) {
	return s.repo.UpdateTool(ctx, id, input)
}

func (s *Service) ExecuteTool(ctx context.Context, input ExecuteToolInput) (*ExecuteToolOutput, error) {
	if input.ToolName == "" || input.ActorID == uuid.Nil || input.ActorType == "" {
		return nil, fmt.Errorf("%w: tool_name, actor_id, and actor_type are required", ErrValidation)
	}
	if input.Arguments == nil {
		input.Arguments = map[string]any{}
	}
	tool, err := s.repo.GetToolByName(ctx, input.ToolName)
	if err != nil {
		return nil, err
	}
	governanceResult, err := s.decide(ctx, tool, input)
	if err != nil {
		return nil, err
	}
	policy := EffectivePolicy(tool.DefaultPolicy, governanceResult)
	status := ExecutionRequested
	switch policy {
	case PolicyApprove:
		status = ExecutionApprovalRequired
	case PolicyDeny:
		status = ExecutionDenied
	case PolicyAuto, PolicyNotify:
		status = ExecutionRunning
	}
	execution, err := s.repo.CreateExecution(ctx, CreateExecutionInput{
		ToolID:             tool.ID,
		InvocationID:       input.InvocationID,
		ActorID:            input.ActorID,
		ActorType:          input.ActorType,
		OrganizationID:     input.OrganizationID,
		DepartmentID:       input.DepartmentID,
		ProjectID:          input.ProjectID,
		WorkflowID:         input.WorkflowID,
		TaskID:             input.TaskID,
		IdempotencyKey:     input.IdempotencyKey,
		Policy:             policy,
		GovernanceDecision: governanceResult.Decision,
		Status:             status,
		Arguments:          input.Arguments,
	})
	if err != nil {
		return nil, err
	}
	switch policy {
	case PolicyDeny:
		completed, err := s.repo.CompleteExecution(ctx, execution.ID, CompleteExecutionInput{
			Status:       ExecutionDenied,
			ErrorMessage: governanceResult.Reason,
		})
		if err != nil {
			return nil, err
		}
		return &ExecuteToolOutput{Execution: completed}, nil
	case PolicyApprove:
		approval, err := s.repo.CreateApproval(ctx, execution.ID, &input.ActorID, governanceResult.Reason)
		if err != nil {
			return nil, err
		}
		return &ExecuteToolOutput{Execution: execution, Approval: approval}, nil
	default:
		return s.runAdapter(ctx, tool, execution, input)
	}
}

func (s *Service) TestTool(ctx context.Context, id uuid.UUID, input ExecuteToolInput) (*ExecuteToolOutput, error) {
	tool, err := s.repo.GetToolByID(ctx, id)
	if err != nil {
		return nil, err
	}
	input.ToolName = tool.Name
	return s.ExecuteTool(ctx, input)
}

func (s *Service) ListExecutions(ctx context.Context, limit int) ([]ToolExecution, error) {
	return s.repo.ListExecutions(ctx, limit)
}

func (s *Service) GetExecution(ctx context.Context, id uuid.UUID) (*ToolExecution, error) {
	return s.repo.GetExecution(ctx, id)
}

func (s *Service) Approve(ctx context.Context, id uuid.UUID, reviewedBy *uuid.UUID, reason string) (*ToolApproval, error) {
	return s.repo.UpdateApproval(ctx, id, ApprovalApproved, reviewedBy, reason)
}

func (s *Service) Reject(ctx context.Context, id uuid.UUID, reviewedBy *uuid.UUID, reason string) (*ToolApproval, error) {
	return s.repo.UpdateApproval(ctx, id, ApprovalRejected, reviewedBy, reason)
}

func (s *Service) runAdapter(ctx context.Context, tool *ToolDefinition, execution *ToolExecution, input ExecuteToolInput) (*ExecuteToolOutput, error) {
	adapter, ok := s.adapters[tool.Name]
	if !ok {
		completed, err := s.repo.CompleteExecution(ctx, execution.ID, CompleteExecutionInput{
			Status:       ExecutionFailed,
			ErrorMessage: "tool adapter is not configured",
		})
		if err != nil {
			return nil, err
		}
		return &ExecuteToolOutput{Execution: completed}, fmt.Errorf("%w: tool adapter is not configured", ErrNotFound)
	}
	started := time.Now()
	result, err := adapter(ctx, input)
	if err != nil {
		completed, updateErr := s.repo.CompleteExecution(ctx, execution.ID, CompleteExecutionInput{
			Status:       ExecutionFailed,
			ErrorMessage: err.Error(),
			DurationMS:   int(time.Since(started).Milliseconds()),
		})
		if updateErr != nil {
			return nil, updateErr
		}
		return &ExecuteToolOutput{Execution: completed}, err
	}
	completed, err := s.repo.CompleteExecution(ctx, execution.ID, CompleteExecutionInput{
		Status:        ExecutionCompleted,
		ResultSummary: result.Summary,
		Result:        result.Data,
		DurationMS:    int(time.Since(started).Milliseconds()),
	})
	if err != nil {
		return nil, err
	}
	return &ExecuteToolOutput{Execution: completed}, nil
}

func (s *Service) decide(ctx context.Context, tool *ToolDefinition, input ExecuteToolInput) (GovernanceResult, error) {
	if s.governance == nil {
		return GovernanceResult{Decision: "notify", Allowed: true, Reason: "governance service not configured"}, nil
	}
	decision, err := s.governance.DecideAccess(ctx, governance.AccessDecisionInput{
		ActorID:        input.ActorID,
		ActorType:      input.ActorType,
		Action:         tool.Name,
		Resource:       "tool",
		ResourceID:     &tool.ID,
		OrganizationID: input.OrganizationID,
		DepartmentID:   input.DepartmentID,
		WorkflowID:     input.WorkflowID,
		TaskID:         input.TaskID,
		RequiredLevel:  tool.RequiredLevel,
		RiskLevel:      tool.RiskLevel,
		Context: map[string]any{
			"tool_source_type": tool.SourceType,
			"project_id":       input.ProjectID,
			"idempotency_key":  input.IdempotencyKey,
		},
	})
	if err != nil {
		return GovernanceResult{}, err
	}
	return GovernanceResult{Decision: decision.Decision, Allowed: decision.Allowed, Reason: decision.Reason}, nil
}
