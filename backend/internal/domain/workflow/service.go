package workflow

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("validation error")
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateTemplate(ctx context.Context, input CreateWorkflowInput) (*WorkflowTemplate, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	if len(input.Stages) == 0 {
		input.Stages = defaultStages()
	}
	if input.AssigneeType == "" {
		input.AssigneeType = "either"
	}
	if input.RoutingRules == nil {
		input.RoutingRules = map[string]any{}
	}
	if input.VisualGraph == nil {
		input.VisualGraph = map[string]any{}
	}
	for i := range input.Stages {
		normalizeStage(&input.Stages[i])
	}
	return s.repo.CreateTemplate(ctx, input)
}

func (s *Service) GetTemplate(ctx context.Context, id uuid.UUID) (*WorkflowTemplate, error) {
	return s.repo.GetTemplate(ctx, id)
}

func (s *Service) ListTemplates(ctx context.Context) ([]WorkflowTemplate, error) {
	return s.repo.ListTemplates(ctx)
}

func (s *Service) StartWorkflow(ctx context.Context, input StartWorkflowInput) (*WorkflowInstance, error) {
	tmpl, err := s.repo.GetTemplate(ctx, input.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}
	if input.Context == nil {
		input.Context = map[string]any{}
	}

	return s.repo.CreateInstanceWithTasks(ctx, input, tmpl)
}

func (s *Service) GetWorkflow(ctx context.Context, id uuid.UUID) (*WorkflowInstance, error) {
	inst, err := s.repo.GetInstance(ctx, id)
	if err != nil {
		return nil, err
	}
	tasks, err := s.repo.GetTasksByWorkflow(ctx, id)
	if err != nil {
		return nil, err
	}
	inst.Tasks = tasks
	return inst, nil
}

func (s *Service) UpdateWorkflowStatus(ctx context.Context, id uuid.UUID, status WorkflowStatus) error {
	if !isValidWorkflowStatus(status) {
		return fmt.Errorf("%w: invalid workflow status", ErrValidation)
	}
	return s.repo.UpdateInstanceStatus(ctx, id, status)
}

func (s *Service) CompleteTask(ctx context.Context, taskID uuid.UUID, output map[string]any) error {
	if output == nil {
		output = map[string]any{}
	}
	return s.repo.CompleteTaskWithWorkflowProgress(ctx, taskID, output)
}

func (s *Service) RecordDecision(ctx context.Context, taskID uuid.UUID, decisionMakerID uuid.UUID, makerType string, reasoning string, outcome string, input, output map[string]any) (*Decision, error) {
	d := &Decision{
		TaskID:          taskID,
		DecisionMakerID: decisionMakerID,
		MakerType:       makerType,
		Weight:          1.0,
		Input:           input,
		Output:          output,
		Reasoning:       reasoning,
		Outcome:         outcome,
	}
	return s.repo.RecordDecision(ctx, d)
}

func (s *Service) GetContext(ctx context.Context, workflowID uuid.UUID) (*WorkflowContext, error) {
	return s.repo.GetWorkflowContext(ctx, workflowID)
}

func (s *Service) UpdateContext(ctx context.Context, wc *WorkflowContext) error {
	return s.repo.UpsertWorkflowContext(ctx, wc)
}

func defaultStages() []Stage {
	return []Stage{
		{Type: StagePlan, Name: "Planning", AssigneeType: "either", RequiredPermissionLevel: "L1", RiskLevel: "low"},
		{Type: StageExecute, Name: "Execution", AssigneeType: "either", RequiredPermissionLevel: "L2", RiskLevel: "medium"},
		{Type: StageReview, Name: "Review", AssigneeType: "internal", RequiredPermissionLevel: "L2", RiskLevel: "medium", PreferredActorTypes: []string{"internal_human"}},
	}
}

func normalizeStage(stage *Stage) {
	if stage.Name == "" {
		stage.Name = string(stage.Type)
	}
	if stage.ID == "" {
		stage.ID = string(stage.Type) + "-" + stage.Name
	}
	if stage.AssigneeType == "" {
		stage.AssigneeType = "either"
	}
	if stage.RequiredPermissionLevel == "" {
		stage.RequiredPermissionLevel = "L1"
	}
	if stage.RiskLevel == "" {
		stage.RiskLevel = "low"
	}
	if stage.EvaluationPolicy == nil {
		stage.EvaluationPolicy = map[string]any{"primary_reviewer": "human"}
	}
	if stage.MatchingPolicy == nil {
		stage.MatchingPolicy = map[string]any{"ranking": "capability_weight_access"}
	}
}

func isValidWorkflowStatus(status WorkflowStatus) bool {
	switch status {
	case WorkflowActive, WorkflowPaused, WorkflowCompleted, WorkflowFailed:
		return true
	default:
		return false
	}
}
