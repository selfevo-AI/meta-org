package project

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/harness-org/backend/internal/domain/evolution"
	"github.com/harness-org/backend/internal/domain/governance"
	"github.com/harness-org/backend/internal/domain/organization"
	"github.com/harness-org/backend/internal/domain/workflow"
	"github.com/harness-org/backend/internal/pkg/middleware"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("validation error")
	ErrForbidden  = errors.New("forbidden")
	ErrConflict   = errors.New("conflict")
)

type Service struct {
	repo         *Repository
	governance   *governance.Service
	evolution    *evolution.Service
	organization *organization.Service
	workflow     *workflow.Service
}

type ServiceOption func(*Service)

func WithGovernanceService(gov *governance.Service) ServiceOption {
	return func(s *Service) {
		s.governance = gov
	}
}

func WithEvolutionService(evo *evolution.Service) ServiceOption {
	return func(s *Service) {
		s.evolution = evo
	}
}

func WithOrganizationService(org *organization.Service) ServiceOption {
	return func(s *Service) {
		s.organization = org
	}
}

func WithWorkflowService(wf *workflow.Service) ServiceOption {
	return func(s *Service) {
		s.workflow = wf
	}
}

func NewService(repo *Repository, opts ...ServiceOption) *Service {
	s := &Service{repo: repo}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Service) CreateRequirement(ctx context.Context, input CreateRequirementInput) (*Requirement, error) {
	if input.Title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrValidation)
	}
	normalizeRequirementInput(&input)
	actorID, actorType, err := s.resolveActor(ctx, ActorInput{ActorID: input.CreatedByID, ActorType: input.CreatedByType})
	if err != nil {
		return nil, err
	}
	input.CreatedByID = &actorID
	input.CreatedByType = actorType
	if err := s.requireAccess(ctx, actorID, actorType, "requirement.create", "requirement", nil, input.OrganizationID, input.DepartmentID, nil, input.RequiredLevel, input.RiskLevel, nil); err != nil {
		return nil, err
	}
	return s.repo.CreateRequirement(ctx, input)
}

func (s *Service) ListRequirements(ctx context.Context, limit int) ([]Requirement, error) {
	requirements, err := s.repo.ListRequirements(ctx, limit)
	if requirements == nil {
		requirements = []Requirement{}
	}
	return requirements, err
}

func (s *Service) GetRequirement(ctx context.Context, id uuid.UUID) (*Requirement, error) {
	return s.repo.GetRequirement(ctx, id)
}

func (s *Service) UploadRequirementDocument(ctx context.Context, requirementID uuid.UUID, input UploadRequirementDocumentInput) (*RequirementDocument, error) {
	req, err := s.repo.GetRequirement(ctx, requirementID)
	if err != nil {
		return nil, err
	}
	actorID, actorType, err := s.resolveActor(ctx, input.ActorInput)
	if err != nil {
		return nil, err
	}
	if !isHumanActor(actorType) {
		return nil, fmt.Errorf("%w: only human employees can upload requirement documents", ErrForbidden)
	}
	if input.FileName == "" || len(input.Content) == 0 {
		return nil, fmt.Errorf("%w: file is required", ErrValidation)
	}
	if input.ContentType == "" {
		input.ContentType = "application/octet-stream"
	}
	input.SizeBytes = int64(len(input.Content))
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
	if err := s.requireAccess(ctx, actorID, actorType, "requirement.document.upload", "requirement", &requirementID, req.OrganizationID, req.DepartmentID, nil, req.RequiredLevel, req.RiskLevel, nil); err != nil {
		return nil, err
	}
	return s.repo.CreateRequirementDocument(ctx, requirementID, input, &actorID, actorType)
}

func (s *Service) ListRequirementDocuments(ctx context.Context, requirementID uuid.UUID) ([]RequirementDocument, error) {
	documents, err := s.repo.ListRequirementDocuments(ctx, requirementID)
	if documents == nil {
		documents = []RequirementDocument{}
	}
	return documents, err
}

func (s *Service) GetRequirementDocument(ctx context.Context, id uuid.UUID) (*RequirementDocumentContent, error) {
	return s.repo.GetRequirementDocument(ctx, id)
}

func (s *Service) StartRequirementAnalysisWorkflow(ctx context.Context, requirementID uuid.UUID, input StartRequirementAnalysisWorkflowInput) (*RequirementAnalysisWorkflow, error) {
	req, err := s.repo.GetRequirement(ctx, requirementID)
	if err != nil {
		return nil, err
	}
	if s.workflow == nil {
		return nil, fmt.Errorf("%w: workflow service is unavailable", ErrValidation)
	}
	if input.WorkflowTemplateID == uuid.Nil {
		return nil, fmt.Errorf("%w: workflow_template_id is required", ErrValidation)
	}
	actorID, actorType, err := s.resolveActor(ctx, input.ActorInput)
	if err != nil {
		return nil, err
	}
	if !isHumanActor(actorType) {
		return nil, fmt.Errorf("%w: only human employees can start requirement analysis workflows", ErrForbidden)
	}
	if err := s.requireAccess(ctx, actorID, actorType, "requirement.workflow.start", "requirement", &requirementID, req.OrganizationID, req.DepartmentID, nil, minLevel(req.RequiredLevel, "L2"), req.RiskLevel, nil); err != nil {
		return nil, err
	}
	if input.Purpose == "" {
		input.Purpose = "requirement_analysis"
	}
	if input.Context == nil {
		input.Context = map[string]any{}
	}
	documents, _ := s.ListRequirementDocuments(ctx, requirementID)
	input.Context["requirement_id"] = req.ID.String()
	input.Context["requirement_title"] = req.Title
	input.Context["requirement_description"] = req.Description
	input.Context["requirement_priority"] = req.Priority
	input.Context["requirement_risk_level"] = req.RiskLevel
	input.Context["requirement_required_level"] = req.RequiredLevel
	input.Context["requirement_documents"] = documents
	input.Context["analysis_result_contract"] = map[string]any{
		"generated_requirement": map[string]any{
			"title":          "string",
			"description":    "string",
			"priority":       "low|medium|high|critical",
			"risk_level":     "low|medium|high|critical",
			"required_level": "L1|L2|L3|L4",
			"analysis":       "object",
		},
	}
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
	input.Metadata["started_by_id"] = actorID.String()
	input.Metadata["started_by_type"] = actorType

	inst, err := s.workflow.StartWorkflow(ctx, workflow.StartWorkflowInput{
		TemplateID:     input.WorkflowTemplateID,
		OrganizationID: req.OrganizationID,
		DepartmentID:   req.DepartmentID,
		Context:        input.Context,
	})
	if err != nil {
		return nil, err
	}
	analysis, err := s.repo.CreateRequirementAnalysisWorkflow(ctx, requirementID, inst.ID, input)
	if err != nil {
		return nil, err
	}
	_, _ = s.repo.UpdateRequirement(ctx, requirementID, UpdateRequirementInput{
		Metadata: mergeMetadata(req.Metadata, map[string]any{
			"active_analysis_workflow_id": inst.ID.String(),
			"analysis_workflow_status":    "active",
		}),
	})
	return analysis, nil
}

func (s *Service) ListRequirementAnalysisWorkflows(ctx context.Context, requirementID uuid.UUID) ([]RequirementAnalysisWorkflow, error) {
	workflows, err := s.repo.ListRequirementAnalysisWorkflows(ctx, requirementID)
	if workflows == nil {
		workflows = []RequirementAnalysisWorkflow{}
	}
	return workflows, err
}

func (s *Service) SyncRequirementAnalysisWorkflow(ctx context.Context, requirementID uuid.UUID, input SyncRequirementAnalysisWorkflowInput) (map[string]any, error) {
	req, err := s.repo.GetRequirement(ctx, requirementID)
	if err != nil {
		return nil, err
	}
	if s.workflow == nil {
		return nil, fmt.Errorf("%w: workflow service is unavailable", ErrValidation)
	}
	actorID, actorType, err := s.resolveActor(ctx, input.ActorInput)
	if err != nil {
		return nil, err
	}
	if err := s.requireAccess(ctx, actorID, actorType, "requirement.workflow.sync", "requirement", &requirementID, req.OrganizationID, req.DepartmentID, nil, req.RequiredLevel, req.RiskLevel, nil); err != nil {
		return nil, err
	}

	workflowID := input.WorkflowID
	if workflowID == uuid.Nil {
		workflows, err := s.ListRequirementAnalysisWorkflows(ctx, requirementID)
		if err != nil {
			return nil, err
		}
		if len(workflows) == 0 {
			return nil, fmt.Errorf("%w: no analysis workflow found", ErrNotFound)
		}
		workflowID = workflows[0].WorkflowID
	}
	link, err := s.repo.GetRequirementAnalysisWorkflow(ctx, requirementID, workflowID)
	if err != nil {
		return nil, err
	}
	inst, err := s.workflow.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	if inst.Status != workflow.WorkflowCompleted {
		_, _ = s.repo.UpdateRequirementAnalysisWorkflow(ctx, link.ID, string(inst.Status), nil, map[string]any{
			"last_sync_attempt": "workflow_not_completed",
		})
		return nil, fmt.Errorf("%w: workflow is %s", ErrConflict, inst.Status)
	}

	result := s.requirementAnalysisResult(ctx, inst)
	generated := generatedRequirementFromResult(result)
	update := UpdateRequirementInput{
		Status:   "analyzed",
		Analysis: mergeMetadata(req.Analysis, result),
		Metadata: mergeMetadata(req.Metadata, map[string]any{
			"active_analysis_workflow_id": workflowID.String(),
			"analysis_workflow_status":    "completed",
		}),
	}
	if title := stringFromMap(generated, "title"); title != "" {
		update.Title = title
	}
	if description := stringFromMap(generated, "description"); description != "" {
		update.Description = description
	}
	if priority := stringFromMap(generated, "priority"); priority != "" {
		update.Priority = normalizePriority(priority)
	}
	if riskLevel := stringFromMap(generated, "risk_level"); riskLevel != "" {
		update.RiskLevel = normalizeRisk(riskLevel)
	}
	if requiredLevel := stringFromMap(generated, "required_level"); requiredLevel != "" {
		update.RequiredLevel = normalizeLevel(requiredLevel)
	}
	if analysis, ok := mapFromAny(generated["analysis"]); ok {
		update.Analysis = mergeMetadata(update.Analysis, analysis)
	}

	updatedRequirement, err := s.repo.UpdateRequirement(ctx, requirementID, update)
	if err != nil {
		return nil, err
	}
	analysisWorkflow, err := s.repo.UpdateRequirementAnalysisWorkflow(ctx, link.ID, "completed", result, map[string]any{
		"synced_requirement_id": updatedRequirement.ID.String(),
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"requirement":       updatedRequirement,
		"analysis_workflow": analysisWorkflow,
		"workflow":          inst,
	}, nil
}

func (s *Service) UpdateRequirement(ctx context.Context, id uuid.UUID, input UpdateRequirementInput) (*Requirement, error) {
	current, err := s.repo.GetRequirement(ctx, id)
	if err != nil {
		return nil, err
	}
	actorID, actorType, err := s.resolveActor(ctx, ActorInput{})
	if err != nil {
		return nil, err
	}
	if err := s.requireAccess(ctx, actorID, actorType, "requirement.update", "requirement", &id, current.OrganizationID, current.DepartmentID, nil, current.RequiredLevel, current.RiskLevel, nil); err != nil {
		return nil, err
	}
	normalizeRequirementUpdate(&input)
	return s.repo.UpdateRequirement(ctx, id, input)
}

func (s *Service) AnalyzeRequirement(ctx context.Context, id uuid.UUID, input AnalyzeRequirementInput) (*Requirement, error) {
	req, err := s.repo.GetRequirement(ctx, id)
	if err != nil {
		return nil, err
	}
	actorID, actorType, err := s.resolveActor(ctx, input.ActorInput)
	if err != nil {
		return nil, err
	}
	if err := s.requireAccess(ctx, actorID, actorType, "requirement.analyze", "requirement", &id, req.OrganizationID, req.DepartmentID, nil, req.RequiredLevel, req.RiskLevel, nil); err != nil {
		return nil, err
	}
	analysis := buildRequirementAnalysis(req, input.Notes)
	return s.repo.UpdateRequirement(ctx, id, UpdateRequirementInput{
		Status:   "analyzed",
		Analysis: analysis,
	})
}

func (s *Service) ApproveRequirement(ctx context.Context, id uuid.UUID, input ActorInput) (*Requirement, error) {
	req, err := s.repo.GetRequirement(ctx, id)
	if err != nil {
		return nil, err
	}
	actorID, actorType, err := s.resolveActor(ctx, input)
	if err != nil {
		return nil, err
	}
	if err := s.requireAccess(ctx, actorID, actorType, "requirement.approve", "requirement", &id, req.OrganizationID, req.DepartmentID, nil, minLevel(req.RequiredLevel, "L2"), req.RiskLevel, nil); err != nil {
		return nil, err
	}
	return s.repo.UpdateRequirement(ctx, id, UpdateRequirementInput{Status: "approved"})
}

func (s *Service) ConvertRequirementToProject(ctx context.Context, id uuid.UUID, input ConvertRequirementInput) (*Project, error) {
	req, err := s.repo.GetRequirement(ctx, id)
	if err != nil {
		return nil, err
	}
	actorID, actorType, err := s.resolveActor(ctx, input.ActorInput)
	if err != nil {
		return nil, err
	}
	if err := s.requireAccess(ctx, actorID, actorType, "project.create", "project", nil, req.OrganizationID, req.DepartmentID, nil, req.RequiredLevel, req.RiskLevel, nil); err != nil {
		return nil, err
	}
	name := input.Name
	if name == "" {
		name = req.Title
	}
	description := input.Description
	if description == "" {
		description = req.Description
	}
	metadata := input.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["converted_from_requirement"] = req.ID.String()
	proj, err := s.repo.CreateProject(ctx, CreateProjectInput{
		RequirementID:  &req.ID,
		OrganizationID: req.OrganizationID,
		DepartmentID:   req.DepartmentID,
		Name:           name,
		Description:    description,
		Status:         "planning",
		Priority:       req.Priority,
		RiskLevel:      req.RiskLevel,
		RequiredLevel:  req.RequiredLevel,
		BudgetAmount:   input.BudgetAmount,
		Metadata:       metadata,
	})
	if err != nil {
		return nil, err
	}
	_, _ = s.repo.UpdateRequirement(ctx, id, UpdateRequirementInput{Status: "converted"})
	return proj, nil
}

func (s *Service) CreateProject(ctx context.Context, input CreateProjectInput) (*Project, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	normalizeProjectInput(&input)
	actorID, actorType, err := s.resolveActor(ctx, input.ActorInput)
	if err != nil {
		return nil, err
	}
	if input.RequirementID != nil {
		if req, err := s.repo.GetRequirement(ctx, *input.RequirementID); err == nil {
			if input.OrganizationID == nil {
				input.OrganizationID = req.OrganizationID
			}
			if input.DepartmentID == nil {
				input.DepartmentID = req.DepartmentID
			}
			if input.Description == "" {
				input.Description = req.Description
			}
		}
	}
	if err := s.requireAccess(ctx, actorID, actorType, "project.create", "project", nil, input.OrganizationID, input.DepartmentID, nil, input.RequiredLevel, input.RiskLevel, nil); err != nil {
		return nil, err
	}
	return s.repo.CreateProject(ctx, input)
}

func (s *Service) ListProjects(ctx context.Context, limit int) ([]Project, error) {
	projects, err := s.repo.ListProjects(ctx, limit)
	if projects == nil {
		projects = []Project{}
	}
	return projects, err
}

func (s *Service) GetProject(ctx context.Context, id uuid.UUID) (*Project, error) {
	return s.repo.GetProject(ctx, id)
}

func (s *Service) UpdateProject(ctx context.Context, id uuid.UUID, input UpdateProjectInput) (*Project, error) {
	current, err := s.repo.GetProject(ctx, id)
	if err != nil {
		return nil, err
	}
	actorID, actorType, err := s.resolveActor(ctx, ActorInput{})
	if err != nil {
		return nil, err
	}
	if err := s.requireAccess(ctx, actorID, actorType, "project.update", "project", &id, current.OrganizationID, current.DepartmentID, nil, current.RequiredLevel, current.RiskLevel, nil); err != nil {
		return nil, err
	}
	normalizeProjectUpdate(&input)
	return s.repo.UpdateProject(ctx, id, input)
}

func (s *Service) AddProjectMember(ctx context.Context, projectID uuid.UUID, input AddProjectMemberInput) (*ProjectMember, error) {
	proj, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	actorID, actorType, err := s.resolveActor(ctx, input.ActorInput)
	if err != nil {
		return nil, err
	}
	if input.MemberActorID == uuid.Nil || input.MemberActorType == "" {
		return nil, fmt.Errorf("%w: member_actor_id and member_actor_type are required", ErrValidation)
	}
	input.ProjectID = projectID
	normalizeProjectMemberInput(&input)
	if err := s.requireAccess(ctx, actorID, actorType, "project.assign", "project", &projectID, proj.OrganizationID, proj.DepartmentID, nil, minLevel(input.PermissionLevel, proj.RequiredLevel), proj.RiskLevel, nil); err != nil {
		return nil, err
	}
	return s.repo.AddProjectMember(ctx, input)
}

func (s *Service) ListProjectMembers(ctx context.Context, projectID uuid.UUID) ([]ProjectMember, error) {
	members, err := s.repo.ListProjectMembers(ctx, projectID)
	if members == nil {
		members = []ProjectMember{}
	}
	return members, err
}

func (s *Service) BindProjectWorkflow(ctx context.Context, projectID uuid.UUID, input BindProjectWorkflowInput) (*ProjectWorkflow, error) {
	proj, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	actorID, actorType, err := s.resolveActor(ctx, input.ActorInput)
	if err != nil {
		return nil, err
	}
	if input.WorkflowID == nil && input.WorkflowTemplateID == nil {
		return nil, fmt.Errorf("%w: workflow_id or workflow_template_id is required", ErrValidation)
	}
	if err := s.requireAccess(ctx, actorID, actorType, "project.workflow.bind", "project", &projectID, proj.OrganizationID, proj.DepartmentID, nil, proj.RequiredLevel, proj.RiskLevel, nil); err != nil {
		return nil, err
	}
	normalizeProjectWorkflowInput(&input)
	workflowID := uuid.Nil
	if input.WorkflowID != nil {
		workflowID = *input.WorkflowID
	} else if s.workflow != nil && input.WorkflowTemplateID != nil {
		instance, err := s.workflow.StartWorkflow(ctx, workflow.StartWorkflowInput{
			TemplateID:     *input.WorkflowTemplateID,
			OrganizationID: proj.OrganizationID,
			DepartmentID:   proj.DepartmentID,
			ProjectID:      &projectID,
			Context: map[string]any{
				"project_id":        projectID.String(),
				"project_name":      proj.Name,
				"requirement_id":    uuidString(proj.RequirementID),
				"organization_id":   uuidString(proj.OrganizationID),
				"department_id":     uuidString(proj.DepartmentID),
				"required_level":    proj.RequiredLevel,
				"risk_level":        proj.RiskLevel,
				"lifecycle_purpose": input.Purpose,
			},
		})
		if err != nil {
			return nil, err
		}
		workflowID = instance.ID
	}
	if workflowID == uuid.Nil {
		return nil, fmt.Errorf("%w: workflow_id is required when workflow service is unavailable", ErrValidation)
	}
	return s.repo.BindProjectWorkflow(ctx, input, projectID, workflowID)
}

func (s *Service) ListProjectWorkflows(ctx context.Context, projectID uuid.UUID) ([]ProjectWorkflow, error) {
	workflows, err := s.repo.ListProjectWorkflows(ctx, projectID)
	if workflows == nil {
		workflows = []ProjectWorkflow{}
	}
	return workflows, err
}

func (s *Service) MatchProjectActors(ctx context.Context, projectID uuid.UUID, input MatchProjectActorsInput) ([]organization.MemberMatchCandidate, error) {
	proj, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if s.organization == nil {
		return []organization.MemberMatchCandidate{}, nil
	}
	if proj.OrganizationID == nil {
		return nil, fmt.Errorf("%w: project has no organization_id", ErrValidation)
	}
	if input.TaskDescription == "" {
		input.TaskDescription = proj.Description
	}
	if input.RequiredLevel == "" {
		input.RequiredLevel = proj.RequiredLevel
	}
	if input.RiskLevel == "" {
		input.RiskLevel = proj.RiskLevel
	}
	return s.organization.MatchMembers(ctx, organization.MatchMembersInput{
		OrganizationID:       *proj.OrganizationID,
		DepartmentID:         proj.DepartmentID,
		TaskDescription:      input.TaskDescription,
		WorkflowTemplateID:   input.WorkflowTemplateID,
		RequiredCapabilities: input.RequiredCapabilities,
		RequiredLevel:        input.RequiredLevel,
		RiskLevel:            input.RiskLevel,
		MemberTypes:          input.MemberTypes,
	})
}

func (s *Service) UpdateProjectStatus(ctx context.Context, projectID uuid.UUID, input UpdateProjectStatusInput) (*Project, error) {
	proj, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	actorID, actorType, err := s.resolveActor(ctx, input.ActorInput)
	if err != nil {
		return nil, err
	}
	if !isValidProjectStatus(input.Status) {
		return nil, fmt.Errorf("%w: invalid project status", ErrValidation)
	}
	if err := s.requireAccess(ctx, actorID, actorType, "project.status", "project", &projectID, proj.OrganizationID, proj.DepartmentID, nil, proj.RequiredLevel, proj.RiskLevel, nil); err != nil {
		return nil, err
	}
	metadata := mergeMetadata(proj.Metadata, map[string]any{"last_status_note": input.Note})
	return s.repo.UpdateProject(ctx, projectID, UpdateProjectInput{Status: input.Status, Metadata: metadata})
}

func (s *Service) GetProjectOverview(ctx context.Context, projectID uuid.UUID) (*ProjectOverview, error) {
	proj, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	var req *Requirement
	if proj.RequirementID != nil {
		req, _ = s.repo.GetRequirement(ctx, *proj.RequirementID)
	}
	members, err := s.ListProjectMembers(ctx, projectID)
	if err != nil {
		return nil, err
	}
	workflows, err := s.ListProjectWorkflows(ctx, projectID)
	if err != nil {
		return nil, err
	}
	deliverables, err := s.ListDeliverables(ctx, projectID)
	if err != nil {
		return nil, err
	}
	costSummary, err := s.GetCostSummary(ctx, projectID)
	if err != nil {
		return nil, err
	}
	evaluations, err := s.ListProjectEvaluations(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return &ProjectOverview{
		Project:      proj,
		Requirement:  req,
		Members:      members,
		Workflows:    workflows,
		Deliverables: deliverables,
		CostSummary:  costSummary,
		Evaluations:  evaluations,
	}, nil
}

func (s *Service) CreateDeliverable(ctx context.Context, projectID uuid.UUID, input CreateDeliverableInput) (*Deliverable, error) {
	proj, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	actorID, actorType, err := s.resolveActor(ctx, input.ActorInput)
	if err != nil {
		return nil, err
	}
	normalizeDeliverableInput(&input)
	action := "deliverable.write"
	if input.Status == "submitted" {
		action = "deliverable.submit"
	}
	if err := s.requireAccess(ctx, actorID, actorType, action, "project", &projectID, proj.OrganizationID, proj.DepartmentID, nil, proj.RequiredLevel, proj.RiskLevel, nil); err != nil {
		return nil, err
	}
	return s.repo.CreateDeliverable(ctx, projectID, input, &actorID, actorType)
}

func (s *Service) ListDeliverables(ctx context.Context, projectID uuid.UUID) ([]Deliverable, error) {
	deliverables, err := s.repo.ListDeliverables(ctx, projectID)
	if deliverables == nil {
		deliverables = []Deliverable{}
	}
	return deliverables, err
}

func (s *Service) UpdateDeliverable(ctx context.Context, id uuid.UUID, input UpdateDeliverableInput) (*Deliverable, error) {
	deliverable, err := s.repo.GetDeliverable(ctx, id)
	if err != nil {
		return nil, err
	}
	proj, err := s.repo.GetProject(ctx, deliverable.ProjectID)
	if err != nil {
		return nil, err
	}
	actorID, actorType, err := s.resolveActor(ctx, ActorInput{})
	if err != nil {
		return nil, err
	}
	if err := s.requireAccess(ctx, actorID, actorType, "deliverable.write", "deliverable", &id, proj.OrganizationID, proj.DepartmentID, nil, proj.RequiredLevel, proj.RiskLevel, nil); err != nil {
		return nil, err
	}
	return s.repo.UpdateDeliverable(ctx, id, input)
}

func (s *Service) SubmitDeliverable(ctx context.Context, id uuid.UUID, input DeliverableActionInput) (*Deliverable, error) {
	return s.changeDeliverableStatus(ctx, id, input, "submitted", "deliverable.submit")
}

func (s *Service) AcceptDeliverable(ctx context.Context, id uuid.UUID, input DeliverableActionInput) (*Deliverable, error) {
	return s.changeDeliverableStatus(ctx, id, input, "accepted", "deliverable.accept")
}

func (s *Service) RejectDeliverable(ctx context.Context, id uuid.UUID, input DeliverableActionInput) (*Deliverable, error) {
	if input.Evidence == nil {
		input.Evidence = map[string]any{}
	}
	input.Evidence["reject_reason"] = input.Reason
	return s.changeDeliverableStatus(ctx, id, input, "rejected", "deliverable.accept")
}

func (s *Service) CreateCostEntry(ctx context.Context, projectID uuid.UUID, input CreateCostEntryInput) (*CostEntry, error) {
	proj, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	actorID, actorType, err := s.resolveActor(ctx, input.ActorInput)
	if err != nil {
		return nil, err
	}
	normalizeCostEntryInput(&input)
	if err := s.requireAccess(ctx, actorID, actorType, "cost.write", "project", &projectID, proj.OrganizationID, proj.DepartmentID, nil, proj.RequiredLevel, proj.RiskLevel, nil); err != nil {
		return nil, err
	}
	return s.repo.CreateCostEntry(ctx, projectID, input)
}

func (s *Service) ListCostEntries(ctx context.Context, projectID uuid.UUID) ([]CostEntry, error) {
	entries, err := s.repo.ListCostEntries(ctx, projectID)
	if entries == nil {
		entries = []CostEntry{}
	}
	return entries, err
}

func (s *Service) GetCostSummary(ctx context.Context, projectID uuid.UUID) (*CostSummary, error) {
	return s.repo.GetCostSummary(ctx, projectID)
}

func (s *Service) RefreshCost(ctx context.Context, projectID uuid.UUID, actorInput ActorInput) ([]CostEntry, error) {
	proj, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	actorID, actorType, err := s.resolveActor(ctx, actorInput)
	if err != nil {
		return nil, err
	}
	if err := s.requireAccess(ctx, actorID, actorType, "cost.write", "project", &projectID, proj.OrganizationID, proj.DepartmentID, nil, proj.RequiredLevel, proj.RiskLevel, nil); err != nil {
		return nil, err
	}
	members, err := s.ListProjectMembers(ctx, projectID)
	if err != nil {
		return nil, err
	}
	entries := []CostEntry{}
	for _, member := range members {
		if member.CostRate <= 0 || member.AllocationPercent <= 0 || member.Status == "archived" {
			continue
		}
		amount := member.CostRate * (member.AllocationPercent / 100)
		entry, err := s.repo.CreateCostEntry(ctx, projectID, CreateCostEntryInput{
			SourceType:     "member_allocation",
			EntryActorID:   &member.ActorID,
			EntryActorType: member.ActorType,
			Amount:         amount,
			Currency:       "CNY",
			Description:    "member allocation cost snapshot",
			Metadata: map[string]any{
				"project_member_id":  member.ID.String(),
				"allocation_percent": member.AllocationPercent,
				"cost_rate":          member.CostRate,
				"refresh_actor_id":   actorID.String(),
				"refresh_actor_type": actorType,
			},
		})
		if err != nil {
			return nil, err
		}
		entries = append(entries, *entry)
	}
	return entries, nil
}

func (s *Service) CreateProjectEvaluation(ctx context.Context, projectID uuid.UUID, input CreateProjectEvaluationInput) (*ProjectEvaluation, error) {
	proj, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	actorID, actorType, err := s.resolveActor(ctx, input.ActorInput)
	if err != nil {
		return nil, err
	}
	if err := s.requireAccess(ctx, actorID, actorType, "evaluation.create", "project", &projectID, proj.OrganizationID, proj.DepartmentID, input.CapabilityID, proj.RequiredLevel, proj.RiskLevel, nil); err != nil {
		return nil, err
	}
	normalizeProjectEvaluationInput(&input)
	overall := projectEvaluationOverall(input)
	eval, err := s.repo.CreateProjectEvaluation(ctx, projectID, input, &actorID, actorType, overall)
	if err != nil {
		return nil, err
	}
	s.recordEvaluationOutcome(ctx, proj, eval)
	return eval, nil
}

func (s *Service) ListProjectEvaluations(ctx context.Context, projectID uuid.UUID) ([]ProjectEvaluation, error) {
	evaluations, err := s.repo.ListProjectEvaluations(ctx, projectID)
	if evaluations == nil {
		evaluations = []ProjectEvaluation{}
	}
	return evaluations, err
}

func (s *Service) CloseFeedback(ctx context.Context, projectID uuid.UUID, input CloseFeedbackInput) (map[string]any, error) {
	proj, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	actorID, actorType, err := s.resolveActor(ctx, input.ActorInput)
	if err != nil {
		return nil, err
	}
	if err := s.requireAccess(ctx, actorID, actorType, "evaluation.create", "project", &projectID, proj.OrganizationID, proj.DepartmentID, nil, proj.RequiredLevel, proj.RiskLevel, nil); err != nil {
		return nil, err
	}
	outcomeScore := clampScore(input.OutcomeScore)
	if input.OutcomeScore == 0 {
		outcomeScore = 0.75
	}
	updated := 0
	evaluations, err := s.ListProjectEvaluations(ctx, projectID)
	if err != nil {
		return nil, err
	}
	for _, eval := range evaluations {
		if eval.ActorID == nil || eval.ActorType == "" {
			continue
		}
		score := eval.OverallScore
		if input.OutcomeScore > 0 {
			score = outcomeScore
		}
		s.recordOutcome(ctx, proj, *eval.ActorID, eval.ActorType, eval.CapabilityID, score, map[string]any{
			"project_evaluation_id": eval.ID.String(),
			"source":                "project_close_feedback",
		})
		updated++
	}
	if updated == 0 {
		members, _ := s.ListProjectMembers(ctx, projectID)
		for _, member := range members {
			s.recordOutcome(ctx, proj, member.ActorID, member.ActorType, nil, outcomeScore, map[string]any{
				"project_member_id": member.ID.String(),
				"source":            "project_close_feedback",
			})
			updated++
		}
	}
	updatedProject, _ := s.repo.UpdateProject(ctx, projectID, UpdateProjectInput{
		Status:   "closed",
		Metadata: mergeMetadata(proj.Metadata, map[string]any{"close_feedback": input.Conclusion}),
	})
	return map[string]any{
		"project":        updatedProject,
		"outcome_score":  outcomeScore,
		"updated_actors": updated,
	}, nil
}

func (s *Service) changeDeliverableStatus(ctx context.Context, id uuid.UUID, input DeliverableActionInput, status string, action string) (*Deliverable, error) {
	deliverable, err := s.repo.GetDeliverable(ctx, id)
	if err != nil {
		return nil, err
	}
	proj, err := s.repo.GetProject(ctx, deliverable.ProjectID)
	if err != nil {
		return nil, err
	}
	actorID, actorType, err := s.resolveActor(ctx, input.ActorInput)
	if err != nil {
		return nil, err
	}
	if err := s.requireAccess(ctx, actorID, actorType, action, "deliverable", &id, proj.OrganizationID, proj.DepartmentID, nil, proj.RequiredLevel, proj.RiskLevel, nil); err != nil {
		return nil, err
	}
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
	if input.Reason != "" {
		input.Metadata["reason"] = input.Reason
	}
	return s.repo.UpdateDeliverableStatus(ctx, id, status, &actorID, actorType, input.Evidence, input.Metadata)
}

func (s *Service) resolveActor(ctx context.Context, input ActorInput) (uuid.UUID, string, error) {
	if input.ActorID != nil && *input.ActorID != uuid.Nil {
		actorType := input.ActorType
		if actorType == "" {
			actorType = "internal_human"
		}
		return *input.ActorID, normalizeActorType(actorType), nil
	}
	user, ok := middleware.UserFromContext(ctx)
	if !ok {
		return uuid.Nil, "", fmt.Errorf("%w: actor_id is required", ErrValidation)
	}
	actorID, err := uuid.Parse(user.ID)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("%w: invalid authenticated actor", ErrValidation)
	}
	return actorID, normalizeAuthActorType(user.Type), nil
}

func (s *Service) requireAccess(ctx context.Context, actorID uuid.UUID, actorType string, action string, resource string, resourceID *uuid.UUID, organizationID *uuid.UUID, departmentID *uuid.UUID, capabilityID *uuid.UUID, requiredLevel string, riskLevel string, weightSnapshot *float64) error {
	if s.governance == nil {
		return nil
	}
	decision, err := s.governance.DecideAccess(ctx, governance.AccessDecisionInput{
		ActorID:        actorID,
		ActorType:      actorType,
		Action:         action,
		Resource:       resource,
		ResourceID:     resourceID,
		OrganizationID: organizationID,
		DepartmentID:   departmentID,
		CapabilityID:   capabilityID,
		RequiredLevel:  normalizeLevel(requiredLevel),
		RiskLevel:      normalizeRisk(riskLevel),
		WeightSnapshot: weightSnapshot,
		Context: map[string]any{
			"domain": "project_lifecycle",
		},
	})
	if err != nil {
		return err
	}
	if !decision.Allowed {
		return fmt.Errorf("%w: %s", ErrForbidden, decision.Reason)
	}
	return nil
}

func (s *Service) recordEvaluationOutcome(ctx context.Context, proj *Project, eval *ProjectEvaluation) {
	if eval.ActorID == nil || eval.ActorType == "" {
		return
	}
	s.recordOutcome(ctx, proj, *eval.ActorID, eval.ActorType, eval.CapabilityID, eval.OverallScore, map[string]any{
		"project_evaluation_id": eval.ID.String(),
		"source":                "project_evaluation",
	})
}

func (s *Service) recordOutcome(ctx context.Context, proj *Project, actorID uuid.UUID, actorType string, capabilityID *uuid.UUID, outcomeScore float64, extra map[string]any) {
	if s.evolution == nil {
		return
	}
	context := map[string]any{
		"project_id":      proj.ID.String(),
		"project_name":    proj.Name,
		"requirement_id":  uuidString(proj.RequirementID),
		"required_level":  proj.RequiredLevel,
		"lifecycle_stage": "feedback",
	}
	for key, value := range extra {
		context[key] = value
	}
	_, _ = s.evolution.RecordContextOutcome(ctx, evolution.ContextOutcomeInput{
		ActorID:      actorID,
		ActorType:    actorType,
		OutcomeScore: clampScore(outcomeScore),
		Scope: evolution.ContextWeightScope{
			OrganizationID: proj.OrganizationID,
			DepartmentID:   proj.DepartmentID,
			TaskType:       "project_delivery",
			CapabilityID:   capabilityID,
			RiskLevel:      normalizeRisk(proj.RiskLevel),
			Context:        context,
		},
	})
}

func normalizeRequirementInput(input *CreateRequirementInput) {
	if input.Source == "" {
		input.Source = "manual"
	}
	input.Priority = normalizePriority(input.Priority)
	input.RiskLevel = normalizeRisk(input.RiskLevel)
	input.RequiredLevel = normalizeLevel(input.RequiredLevel)
	if input.Analysis == nil {
		input.Analysis = map[string]any{}
	}
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
}

func normalizeRequirementUpdate(input *UpdateRequirementInput) {
	if input.Priority != "" {
		input.Priority = normalizePriority(input.Priority)
	}
	if input.RiskLevel != "" {
		input.RiskLevel = normalizeRisk(input.RiskLevel)
	}
	if input.RequiredLevel != "" {
		input.RequiredLevel = normalizeLevel(input.RequiredLevel)
	}
}

func normalizeProjectInput(input *CreateProjectInput) {
	if input.Status == "" {
		input.Status = "planning"
	}
	if input.Priority == "" {
		input.Priority = "medium"
	}
	input.Priority = normalizePriority(input.Priority)
	input.RiskLevel = normalizeRisk(input.RiskLevel)
	input.RequiredLevel = normalizeLevel(input.RequiredLevel)
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
}

func normalizeProjectUpdate(input *UpdateProjectInput) {
	if input.Priority != "" {
		input.Priority = normalizePriority(input.Priority)
	}
	if input.RiskLevel != "" {
		input.RiskLevel = normalizeRisk(input.RiskLevel)
	}
	if input.RequiredLevel != "" {
		input.RequiredLevel = normalizeLevel(input.RequiredLevel)
	}
}

func normalizeProjectMemberInput(input *AddProjectMemberInput) {
	if input.Role == "" {
		input.Role = "contributor"
	}
	if input.AllocationPercent <= 0 {
		input.AllocationPercent = 100
	}
	if input.AllocationPercent > 100 {
		input.AllocationPercent = 100
	}
	input.PermissionLevel = normalizeLevel(input.PermissionLevel)
	if input.Status == "" {
		input.Status = "active"
	}
	if input.Capabilities == nil {
		input.Capabilities = []string{}
	}
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
	input.MemberActorType = normalizeActorType(input.MemberActorType)
}

func normalizeProjectWorkflowInput(input *BindProjectWorkflowInput) {
	if input.Purpose == "" {
		input.Purpose = "delivery"
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
}

func normalizeDeliverableInput(input *CreateDeliverableInput) {
	if input.DeliverableType == "" {
		input.DeliverableType = "artifact"
	}
	if input.Version == "" {
		input.Version = "1.0"
	}
	if input.Status == "" {
		input.Status = "draft"
	}
	if input.Evidence == nil {
		input.Evidence = map[string]any{}
	}
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
}

func normalizeCostEntryInput(input *CreateCostEntryInput) {
	if input.SourceType == "" {
		input.SourceType = "manual"
	}
	if input.Currency == "" {
		input.Currency = "CNY"
	}
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
	input.EntryActorType = normalizeActorType(input.EntryActorType)
}

func normalizeProjectEvaluationInput(input *CreateProjectEvaluationInput) {
	input.EvaluatedActorType = normalizeActorType(input.EvaluatedActorType)
	input.QualityScore = clampScore(input.QualityScore)
	input.DeliveryScore = clampScore(input.DeliveryScore)
	input.CostScore = clampScore(input.CostScore)
	input.CollaborationScore = clampScore(input.CollaborationScore)
	if input.Evidence == nil {
		input.Evidence = map[string]any{}
	}
}

func buildRequirementAnalysis(req *Requirement, notes string) map[string]any {
	text := strings.ToLower(req.Title + " " + req.Description + " " + notes)
	capabilities := []string{}
	for _, keyword := range []string{"analysis", "review", "delivery", "integration", "compliance", "finance", "data", "workflow"} {
		if strings.Contains(text, keyword) {
			capabilities = append(capabilities, keyword)
		}
	}
	if len(capabilities) == 0 {
		capabilities = append(capabilities, "planning", "delivery", "review")
	}
	return map[string]any{
		"suggested_capabilities": capabilities,
		"suggested_stages": []string{
			"需求拆解",
			"方案设计",
			"执行交付",
			"人工验收",
			"反馈评估",
		},
		"risk_level":     req.RiskLevel,
		"required_level": req.RequiredLevel,
		"priority":       req.Priority,
		"notes":          notes,
	}
}

func (s *Service) requirementAnalysisResult(ctx context.Context, inst *workflow.WorkflowInstance) map[string]any {
	result := map[string]any{
		"workflow_id":   inst.ID.String(),
		"template_id":   inst.TemplateID.String(),
		"status":        string(inst.Status),
		"current_stage": inst.CurrentStage,
		"context":       inst.Context,
	}
	taskOutputs := []map[string]any{}
	for _, task := range inst.Tasks {
		if task.Output == nil {
			continue
		}
		taskOutputs = append(taskOutputs, map[string]any{
			"task_id":    task.ID.String(),
			"stage":      task.Stage,
			"stage_type": string(task.StageType),
			"output":     task.Output,
		})
	}
	result["task_outputs"] = taskOutputs

	if wc, err := s.workflow.GetContext(ctx, inst.ID); err == nil {
		result["workflow_working_memory"] = wc.WorkingMemory
		result["workflow_principle_notes"] = wc.PrincipleNotes
		if generated, ok := candidateRequirement(wc.WorkingMemory); ok {
			result["generated_requirement"] = generated
		}
	}
	if _, ok := result["generated_requirement"]; !ok {
		if generated, ok := candidateRequirement(inst.Context); ok {
			result["generated_requirement"] = generated
		}
	}
	if _, ok := result["generated_requirement"]; !ok {
		for i := len(inst.Tasks) - 1; i >= 0; i-- {
			if generated, ok := candidateRequirement(inst.Tasks[i].Output); ok {
				result["generated_requirement"] = generated
				break
			}
		}
	}
	return result
}

func generatedRequirementFromResult(result map[string]any) map[string]any {
	if generated, ok := mapFromAny(result["generated_requirement"]); ok {
		return generated
	}
	return map[string]any{}
}

func candidateRequirement(payload map[string]any) (map[string]any, bool) {
	if payload == nil {
		return nil, false
	}
	for _, key := range []string{"generated_requirement", "requirement", "requirement_result"} {
		if generated, ok := mapFromAny(payload[key]); ok {
			return generated, true
		}
	}
	if _, ok := payload["title"].(string); ok {
		return payload, true
	}
	if _, ok := payload["description"].(string); ok {
		return payload, true
	}
	if _, ok := payload["analysis"]; ok {
		return payload, true
	}
	return nil, false
}

func mapFromAny(value any) (map[string]any, bool) {
	if value == nil {
		return nil, false
	}
	if mapped, ok := value.(map[string]any); ok {
		return mapped, true
	}
	if mapped, ok := value.(map[string]string); ok {
		result := map[string]any{}
		for key, item := range mapped {
			result[key] = item
		}
		return result, true
	}
	return nil, false
}

func stringFromMap(value map[string]any, key string) string {
	raw, _ := value[key].(string)
	return strings.TrimSpace(raw)
}

func projectEvaluationOverall(input CreateProjectEvaluationInput) float64 {
	return clampScore(input.QualityScore*0.35 + input.DeliveryScore*0.30 + input.CostScore*0.15 + input.CollaborationScore*0.20)
}

func clampScore(score float64) float64 {
	return math.Min(math.Max(score, 0), 1)
}

func normalizePriority(priority string) string {
	switch priority {
	case "low", "medium", "high", "critical":
		return priority
	default:
		return "medium"
	}
}

func normalizeRisk(risk string) string {
	switch risk {
	case "low", "medium", "high", "critical":
		return risk
	default:
		return "low"
	}
}

func normalizeLevel(level string) string {
	switch level {
	case "L1", "L2", "L3", "L4":
		return level
	default:
		return "L1"
	}
}

func minLevel(primary string, fallback string) string {
	if primary != "" {
		return normalizeLevel(primary)
	}
	return normalizeLevel(fallback)
}

func normalizeActorType(actorType string) string {
	switch actorType {
	case "human":
		return "internal_human"
	case "ai":
		return "internal_agent"
	case "internal", "internal_human":
		return "internal_human"
	case "external", "external_human":
		return "external_human"
	case "agent", "internal_agent":
		return "internal_agent"
	case "external_agent":
		return "external_agent"
	default:
		return actorType
	}
}

func normalizeAuthActorType(userType string) string {
	if userType == "ai" {
		return "internal_agent"
	}
	return "internal_human"
}

func isHumanActor(actorType string) bool {
	return actorType == "internal_human" || actorType == "external_human"
}

func isValidProjectStatus(status string) bool {
	switch status {
	case "planning", "active", "paused", "delivering", "completed", "closed", "cancelled":
		return true
	default:
		return false
	}
}

func mergeMetadata(base map[string]any, patch map[string]any) map[string]any {
	merged := map[string]any{}
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range patch {
		if value != "" && value != nil {
			merged[key] = value
		}
	}
	return merged
}

func uuidString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}
