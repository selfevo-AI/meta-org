package organization

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/harness-org/backend/internal/domain/evolution"
	"github.com/harness-org/backend/internal/domain/governance"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrValidation = errors.New("validation error")
)

type Repository interface {
	CreateOrganization(ctx context.Context, input CreateOrganizationInput) (*Organization, error)
	GetOrganizationByID(ctx context.Context, id uuid.UUID) (*Organization, error)
	ListOrganizations(ctx context.Context, limit int) ([]Organization, error)
	UpdateOrganization(ctx context.Context, id uuid.UUID, input UpdateOrganizationInput) (*Organization, error)
	CreateMVRU(ctx context.Context, input CreateMVRUInput) (*MVRU, error)
	GetMVRUByID(ctx context.Context, id uuid.UUID) (*MVRU, error)
	ListMVRUs(ctx context.Context, orgID uuid.UUID) ([]MVRU, error)
	UpdateMVRUStatus(ctx context.Context, id uuid.UUID, status MVRUStatus) error
	AddMember(ctx context.Context, member MVRUMember) error
	RemoveMember(ctx context.Context, mvruID, userID, agentID *uuid.UUID) error
	CreateRelationship(ctx context.Context, rel MVRURelationship) (*MVRURelationship, error)
	GetOrgChart(ctx context.Context, orgID uuid.UUID) ([]MVRU, error)
	CreateDepartment(ctx context.Context, input CreateDepartmentInput) (*Department, error)
	GetDepartmentByID(ctx context.Context, id uuid.UUID) (*Department, error)
	ListDepartments(ctx context.Context, orgID uuid.UUID) ([]Department, error)
	GetDepartmentTree(ctx context.Context, orgID uuid.UUID) ([]Department, error)
	UpdateDepartment(ctx context.Context, id uuid.UUID, input UpdateDepartmentInput) (*Department, error)
	CreatePosition(ctx context.Context, input CreatePositionInput) (*Position, error)
	GetPositionByID(ctx context.Context, id uuid.UUID) (*Position, error)
	ListPositions(ctx context.Context, orgID uuid.UUID, departmentID *uuid.UUID) ([]Position, error)
	UpdatePosition(ctx context.Context, id uuid.UUID, input UpdatePositionInput) (*Position, error)
	CreatePositionAssignment(ctx context.Context, input CreatePositionAssignmentInput) (*PositionAssignment, error)
	ListPositionAssignments(ctx context.Context, positionID uuid.UUID) ([]PositionAssignment, error)
	UpdatePositionAssignment(ctx context.Context, id uuid.UUID, input UpdatePositionAssignmentInput) (*PositionAssignment, error)
	RemovePositionAssignment(ctx context.Context, id uuid.UUID) error
	CreateExternalMember(ctx context.Context, input CreateExternalMemberInput) (*ExternalMember, error)
	GetExternalMemberByID(ctx context.Context, id uuid.UUID) (*ExternalMember, error)
	ListExternalMembers(ctx context.Context, limit int) ([]ExternalMember, error)
	UpdateExternalMember(ctx context.Context, id uuid.UUID, input UpdateExternalMemberInput) (*ExternalMember, error)
	AddOrganizationMember(ctx context.Context, input AddOrganizationMemberInput) (*OrganizationMembership, error)
	ListOrganizationMemberships(ctx context.Context, orgID uuid.UUID, departmentID *uuid.UUID, memberTypes []string) ([]OrganizationMembership, error)
	UpdateOrganizationMembership(ctx context.Context, id uuid.UUID, input UpdateOrganizationMembershipInput) (*OrganizationMembership, error)
	RemoveOrganizationMembership(ctx context.Context, id uuid.UUID) error
	LinkDepartmentMVRU(ctx context.Context, input LinkDepartmentMVRUInput) (*DepartmentMVRULink, error)
	ListDepartmentMVRULinks(ctx context.Context, departmentID uuid.UUID) ([]DepartmentMVRULink, error)
}

type Service struct {
	repo       Repository
	governance *governance.Service
	evolution  *evolution.Service
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

func NewService(repo Repository, opts ...ServiceOption) *Service {
	s := &Service{repo: repo}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *Service) CreateOrganization(ctx context.Context, input CreateOrganizationInput) (*Organization, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	return s.repo.CreateOrganization(ctx, input)
}

func (s *Service) GetCurrentOrganization(ctx context.Context) (*Organization, error) {
	organizations, err := s.repo.ListOrganizations(ctx, 1)
	if err != nil {
		return nil, err
	}
	if len(organizations) > 0 {
		return &organizations[0], nil
	}
	return s.repo.CreateOrganization(ctx, CreateOrganizationInput{
		Name:        "Default Organization",
		Description: "Single organization workspace",
	})
}

func (s *Service) GetOrganization(ctx context.Context, id uuid.UUID) (*Organization, error) {
	return s.repo.GetOrganizationByID(ctx, id)
}

func (s *Service) ListOrganizations(ctx context.Context, limit int) ([]Organization, error) {
	organizations, err := s.repo.ListOrganizations(ctx, limit)
	if organizations == nil {
		organizations = []Organization{}
	}
	return organizations, err
}

func (s *Service) UpdateOrganization(ctx context.Context, id uuid.UUID, input UpdateOrganizationInput) (*Organization, error) {
	if input.Name == "" && input.Description == "" {
		return nil, fmt.Errorf("%w: name or description is required", ErrValidation)
	}
	return s.repo.UpdateOrganization(ctx, id, input)
}

func (s *Service) GetOrgChart(ctx context.Context, orgID uuid.UUID) ([]MVRU, error) {
	return s.repo.GetOrgChart(ctx, orgID)
}

func (s *Service) CreateMVRU(ctx context.Context, input CreateMVRUInput) (*MVRU, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	if input.Boundary == nil {
		input.Boundary = map[string]any{}
	}
	if input.Config == nil {
		input.Config = map[string]any{}
	}
	return s.repo.CreateMVRU(ctx, input)
}

func (s *Service) GetMVRU(ctx context.Context, id uuid.UUID) (*MVRU, error) {
	mvru, err := s.repo.GetMVRUByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mvru, nil
}

func (s *Service) ActivateMVRU(ctx context.Context, id uuid.UUID) error {
	return s.repo.UpdateMVRUStatus(ctx, id, MVRUActive)
}

func (s *Service) EvaluateMVRU(ctx context.Context, id uuid.UUID) error {
	return s.repo.UpdateMVRUStatus(ctx, id, MVRUEvaluating)
}

func (s *Service) AddMember(ctx context.Context, mvruID, roleID uuid.UUID, userID, agentID *uuid.UUID) error {
	if userID == nil && agentID == nil {
		return fmt.Errorf("%w: user_id or agent_id is required", ErrValidation)
	}
	return s.repo.AddMember(ctx, MVRUMember{
		MVRUID:  mvruID,
		UserID:  userID,
		AgentID: agentID,
		RoleID:  roleID,
	})
}

func (s *Service) RemoveMember(ctx context.Context, mvruID uuid.UUID, userID, agentID *uuid.UUID) error {
	return s.repo.RemoveMember(ctx, &mvruID, userID, agentID)
}

func (s *Service) CreateRelationship(ctx context.Context, sourceID, targetID uuid.UUID, relType string, config map[string]any) (*MVRURelationship, error) {
	return s.repo.CreateRelationship(ctx, MVRURelationship{
		SourceMVRUID: sourceID,
		TargetMVRUID: targetID,
		RelType:      relType,
		Config:       config,
	})
}

func (s *Service) CreateDepartment(ctx context.Context, orgID uuid.UUID, input CreateDepartmentInput) (*Department, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	input.OrganizationID = orgID
	normalizeDepartmentInput(&input)
	if !isValidOrgStatus(input.Status) {
		return nil, fmt.Errorf("%w: invalid department status", ErrValidation)
	}
	return s.repo.CreateDepartment(ctx, input)
}

func (s *Service) GetDepartment(ctx context.Context, id uuid.UUID) (*Department, error) {
	return s.repo.GetDepartmentByID(ctx, id)
}

func (s *Service) ListDepartments(ctx context.Context, orgID uuid.UUID) ([]Department, error) {
	departments, err := s.repo.ListDepartments(ctx, orgID)
	if departments == nil {
		departments = []Department{}
	}
	return departments, err
}

func (s *Service) GetDepartmentTree(ctx context.Context, orgID uuid.UUID) ([]Department, error) {
	tree, err := s.repo.GetDepartmentTree(ctx, orgID)
	if tree == nil {
		tree = []Department{}
	}
	return tree, err
}

func (s *Service) UpdateDepartment(ctx context.Context, id uuid.UUID, input UpdateDepartmentInput) (*Department, error) {
	if input.Status != "" && !isValidOrgStatus(input.Status) {
		return nil, fmt.Errorf("%w: invalid department status", ErrValidation)
	}
	if input.ParentID != nil && *input.ParentID == id {
		return nil, fmt.Errorf("%w: department cannot be its own parent", ErrValidation)
	}
	return s.repo.UpdateDepartment(ctx, id, input)
}

func (s *Service) CreatePosition(ctx context.Context, departmentID uuid.UUID, input CreatePositionInput) (*Position, error) {
	dept, err := s.repo.GetDepartmentByID(ctx, departmentID)
	if err != nil {
		return nil, err
	}
	input.OrganizationID = dept.OrganizationID
	input.DepartmentID = departmentID
	normalizePositionInput(&input)
	if input.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	return s.repo.CreatePosition(ctx, input)
}

func (s *Service) GetPosition(ctx context.Context, id uuid.UUID) (*Position, error) {
	return s.repo.GetPositionByID(ctx, id)
}

func (s *Service) ListPositions(ctx context.Context, orgID uuid.UUID, departmentID *uuid.UUID) ([]Position, error) {
	positions, err := s.repo.ListPositions(ctx, orgID, departmentID)
	if positions == nil {
		positions = []Position{}
	}
	return positions, err
}

func (s *Service) UpdatePosition(ctx context.Context, id uuid.UUID, input UpdatePositionInput) (*Position, error) {
	return s.repo.UpdatePosition(ctx, id, input)
}

func (s *Service) CreatePositionAssignment(ctx context.Context, positionID uuid.UUID, input CreatePositionAssignmentInput) (*PositionAssignment, error) {
	position, err := s.repo.GetPositionByID(ctx, positionID)
	if err != nil {
		return nil, err
	}
	input.PositionID = positionID
	normalizePositionAssignmentInput(&input)
	if input.ActorID == uuid.Nil {
		return nil, fmt.Errorf("%w: actor_id is required", ErrValidation)
	}
	if !isValidPositionActorType(input.ActorType) {
		return nil, fmt.Errorf("%w: invalid actor_type", ErrValidation)
	}
	input.Metadata = mergeMetadata(input.Metadata, map[string]any{
		"organization_id": position.OrganizationID.String(),
		"department_id":   position.DepartmentID.String(),
	})
	return s.repo.CreatePositionAssignment(ctx, input)
}

func (s *Service) ListPositionAssignments(ctx context.Context, positionID uuid.UUID) ([]PositionAssignment, error) {
	assignments, err := s.repo.ListPositionAssignments(ctx, positionID)
	if assignments == nil {
		assignments = []PositionAssignment{}
	}
	return assignments, err
}

func (s *Service) UpdatePositionAssignment(ctx context.Context, id uuid.UUID, input UpdatePositionAssignmentInput) (*PositionAssignment, error) {
	return s.repo.UpdatePositionAssignment(ctx, id, input)
}

func (s *Service) RemovePositionAssignment(ctx context.Context, id uuid.UUID) error {
	return s.repo.RemovePositionAssignment(ctx, id)
}

func (s *Service) CreateExternalMember(ctx context.Context, input CreateExternalMemberInput) (*ExternalMember, error) {
	if input.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
	if !isValidOrgStatus(input.Status) {
		return nil, fmt.Errorf("%w: invalid external member status", ErrValidation)
	}
	return s.repo.CreateExternalMember(ctx, input)
}

func (s *Service) GetExternalMember(ctx context.Context, id uuid.UUID) (*ExternalMember, error) {
	return s.repo.GetExternalMemberByID(ctx, id)
}

func (s *Service) ListExternalMembers(ctx context.Context, limit int) ([]ExternalMember, error) {
	members, err := s.repo.ListExternalMembers(ctx, limit)
	if members == nil {
		members = []ExternalMember{}
	}
	return members, err
}

func (s *Service) UpdateExternalMember(ctx context.Context, id uuid.UUID, input UpdateExternalMemberInput) (*ExternalMember, error) {
	if input.Status != "" && !isValidOrgStatus(input.Status) {
		return nil, fmt.Errorf("%w: invalid external member status", ErrValidation)
	}
	return s.repo.UpdateExternalMember(ctx, id, input)
}

func (s *Service) AddOrganizationMember(ctx context.Context, departmentID uuid.UUID, input AddOrganizationMemberInput) (*OrganizationMembership, error) {
	input.DepartmentID = departmentID
	if input.Status == "" {
		input.Status = "active"
	}
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
	if !isValidOrgStatus(input.Status) {
		return nil, fmt.Errorf("%w: invalid membership status", ErrValidation)
	}
	if err := validateMembershipActor(input); err != nil {
		return nil, err
	}
	return s.repo.AddOrganizationMember(ctx, input)
}

func (s *Service) ListOrganizationMemberships(ctx context.Context, orgID uuid.UUID, departmentID *uuid.UUID, memberTypes []string) ([]OrganizationMembership, error) {
	for _, memberType := range memberTypes {
		if !isValidMemberType(memberType) {
			return nil, fmt.Errorf("%w: invalid member type", ErrValidation)
		}
	}
	if memberTypes == nil {
		memberTypes = []string{}
	}
	memberships, err := s.repo.ListOrganizationMemberships(ctx, orgID, departmentID, memberTypes)
	if memberships == nil {
		memberships = []OrganizationMembership{}
	}
	return memberships, err
}

func (s *Service) UpdateOrganizationMembership(ctx context.Context, id uuid.UUID, input UpdateOrganizationMembershipInput) (*OrganizationMembership, error) {
	if input.Status != "" && !isValidOrgStatus(input.Status) {
		return nil, fmt.Errorf("%w: invalid membership status", ErrValidation)
	}
	return s.repo.UpdateOrganizationMembership(ctx, id, input)
}

func (s *Service) RemoveOrganizationMembership(ctx context.Context, id uuid.UUID) error {
	return s.repo.RemoveOrganizationMembership(ctx, id)
}

func (s *Service) LinkDepartmentMVRU(ctx context.Context, departmentID uuid.UUID, input LinkDepartmentMVRUInput) (*DepartmentMVRULink, error) {
	input.DepartmentID = departmentID
	if input.MVRUID == uuid.Nil {
		return nil, fmt.Errorf("%w: mvru_id is required", ErrValidation)
	}
	if input.LinkType == "" {
		input.LinkType = "execution"
	}
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
	return s.repo.LinkDepartmentMVRU(ctx, input)
}

func (s *Service) ListDepartmentMVRULinks(ctx context.Context, departmentID uuid.UUID) ([]DepartmentMVRULink, error) {
	return s.repo.ListDepartmentMVRULinks(ctx, departmentID)
}

func (s *Service) MatchMembers(ctx context.Context, input MatchMembersInput) ([]MemberMatchCandidate, error) {
	if input.OrganizationID == uuid.Nil {
		return nil, fmt.Errorf("%w: organization_id is required", ErrValidation)
	}
	if input.TaskDescription == "" {
		return nil, fmt.Errorf("%w: task_description is required", ErrValidation)
	}
	if input.PositionID != nil {
		position, err := s.repo.GetPositionByID(ctx, *input.PositionID)
		if err != nil {
			return nil, err
		}
		assignments, err := s.repo.ListPositionAssignments(ctx, *input.PositionID)
		if err != nil {
			return nil, err
		}
		candidates := make([]MemberMatchCandidate, 0, len(assignments))
		for _, assignment := range assignments {
			if assignment.Status == "archived" || !matchesMemberTypes(assignment.ActorType, input.MemberTypes) {
				continue
			}
			score := 0.72
			reason := "position assignment"
			if assignment.AssignmentType == "primary" {
				score += 0.12
				reason += ", primary"
			}
			if len(input.RequiredCapabilities) > 0 {
				score += capabilityOverlapScore(position.RequiredCapabilities, input.RequiredCapabilities)
				reason += ", capability fit"
			}
			weightSnapshot := 0.5
			if s.evolution != nil {
				weight, err := s.evolution.ComputeContextWeight(ctx, evolution.ContextWeightInput{
					ActorID:       assignment.ActorID,
					ActorType:     assignment.ActorType,
					RequiredLevel: firstNonEmpty(input.RequiredLevel, position.PermissionLevel),
					Scope: evolution.ContextWeightScope{
						OrganizationID:     &assignment.OrganizationID,
						DepartmentID:       &assignment.DepartmentID,
						WorkflowTemplateID: input.WorkflowTemplateID,
						TaskType:           input.TaskDescription,
						RiskLevel:          normalizeRiskLevel(input.RiskLevel),
						Context: map[string]any{
							"position_id":           position.ID.String(),
							"position_name":         position.Name,
							"required_capabilities": input.RequiredCapabilities,
						},
					},
				})
				if err == nil {
					weightSnapshot = weight.OverallScore
					reason += ", contextual weight"
				}
			}
			accessDecision := "notify"
			accessAllowed := true
			requiresApproval := false
			if s.governance != nil {
				access, err := s.governance.DecideAccess(ctx, governance.AccessDecisionInput{
					ActorID:        assignment.ActorID,
					ActorType:      assignment.ActorType,
					Action:         "workflow.assign",
					Resource:       "position",
					ResourceID:     &position.ID,
					OrganizationID: &assignment.OrganizationID,
					DepartmentID:   &assignment.DepartmentID,
					RequiredLevel:  firstNonEmpty(input.RequiredLevel, position.PermissionLevel),
					RiskLevel:      normalizeRiskLevel(input.RiskLevel),
					WeightSnapshot: &weightSnapshot,
					Context: map[string]any{
						"task_description":       input.TaskDescription,
						"position_assignment_id": assignment.ID.String(),
						"workflow_template_id":   uuidString(input.WorkflowTemplateID),
					},
				})
				if err == nil {
					accessDecision = access.Decision
					accessAllowed = access.Allowed
					requiresApproval = access.Decision == "approve"
					reason += ", access " + access.Decision
				}
			}
			if accessDecision == "deny" {
				continue
			}
			score = (score * 0.6) + (weightSnapshot * 0.3)
			if accessAllowed {
				score += 0.1
			}
			if requiresApproval {
				score -= 0.05
			}
			candidates = append(candidates, MemberMatchCandidate{
				MembershipID:         assignment.ID,
				DepartmentID:         assignment.DepartmentID,
				PositionID:           &position.ID,
				PositionName:         position.Name,
				PositionAssignmentID: &assignment.ID,
				MemberType:           memberTypeFromActorType(assignment.ActorType),
				MemberID:             assignment.ActorID,
				MemberName:           assignment.ActorName,
				Title:                position.Name,
				Score:                clampScore(score),
				WeightSnapshot:       weightSnapshot,
				AccessDecision:       accessDecision,
				AccessAllowed:        accessAllowed,
				RequiresApproval:     requiresApproval,
				Reason:               reason,
				CapabilityMatchPath:  "/api/v1/capabilities/match",
				WorkflowAssignHint:   "Use member_id as task assignee_id and actor_type as assignee_type.",
			})
		}
		sort.SliceStable(candidates, func(i, j int) bool {
			return candidates[i].Score > candidates[j].Score
		})
		if len(candidates) > 10 {
			candidates = candidates[:10]
		}
		return candidates, nil
	}
	memberships, err := s.ListOrganizationMemberships(ctx, input.OrganizationID, input.DepartmentID, input.MemberTypes)
	if err != nil {
		return nil, err
	}

	candidates := make([]MemberMatchCandidate, 0, len(memberships))
	for _, membership := range memberships {
		if membership.Status == "archived" {
			continue
		}
		memberID, ok := membershipActorID(membership)
		if !ok {
			continue
		}
		baseScore, reason := scoreMembership(membership, input)
		actorType := membershipActorType(membership)
		weightSnapshot := 0.5
		if s.evolution != nil {
			weight, err := s.evolution.ComputeContextWeight(ctx, evolution.ContextWeightInput{
				ActorID:       memberID,
				ActorType:     actorType,
				RequiredLevel: input.RequiredLevel,
				Scope: evolution.ContextWeightScope{
					OrganizationID:     &input.OrganizationID,
					DepartmentID:       &membership.DepartmentID,
					WorkflowTemplateID: input.WorkflowTemplateID,
					TaskType:           input.TaskDescription,
					RiskLevel:          normalizeRiskLevel(input.RiskLevel),
					Context: map[string]any{
						"required_capabilities": input.RequiredCapabilities,
						"required_level":        input.RequiredLevel,
						"member_type":           membership.MemberType,
						"membership_id":         membership.ID.String(),
					},
				},
			})
			if err == nil {
				weightSnapshot = weight.OverallScore
				reason += ", contextual weight"
			}
		}

		accessDecision := "notify"
		accessAllowed := true
		requiresApproval := false
		if s.governance != nil {
			access, err := s.governance.DecideAccess(ctx, governance.AccessDecisionInput{
				ActorID:        memberID,
				ActorType:      actorType,
				Action:         "workflow.assign",
				Resource:       "organization_member",
				ResourceID:     &membership.ID,
				OrganizationID: &input.OrganizationID,
				DepartmentID:   &membership.DepartmentID,
				RequiredLevel:  input.RequiredLevel,
				RiskLevel:      normalizeRiskLevel(input.RiskLevel),
				WeightSnapshot: &weightSnapshot,
				Context: map[string]any{
					"task_description":      input.TaskDescription,
					"required_capabilities": input.RequiredCapabilities,
					"workflow_template_id":  uuidString(input.WorkflowTemplateID),
				},
			})
			if err == nil {
				accessDecision = access.Decision
				accessAllowed = access.Allowed
				requiresApproval = access.Decision == "approve"
				reason += ", access " + access.Decision
			}
		}
		if accessDecision == "deny" {
			continue
		}
		score := (baseScore * 0.55) + (weightSnapshot * 0.35)
		if accessAllowed {
			score += 0.10
		}
		if requiresApproval {
			score -= 0.05
		}
		if score > 1 {
			score = 1
		}
		if score < 0 {
			score = 0
		}
		candidates = append(candidates, MemberMatchCandidate{
			MembershipID:        membership.ID,
			DepartmentID:        membership.DepartmentID,
			MemberType:          membership.MemberType,
			MemberID:            memberID,
			MemberName:          membership.MemberName,
			Title:               membership.Title,
			Score:               score,
			WeightSnapshot:      weightSnapshot,
			AccessDecision:      accessDecision,
			AccessAllowed:       accessAllowed,
			RequiresApproval:    requiresApproval,
			Reason:              reason,
			CapabilityMatchPath: "/api/v1/capabilities/match",
			WorkflowAssignHint:  "Use member_id as task assignee_id and member_type as assignee_type.",
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})
	if len(candidates) > 10 {
		candidates = candidates[:10]
	}
	return candidates, nil
}

func (s *Service) MatchCapabilities(ctx context.Context, input MatchCapabilitiesInput) (*CapabilityMatchBridge, error) {
	if input.TaskDescription == "" {
		return nil, fmt.Errorf("%w: task_description is required", ErrValidation)
	}
	if input.Context == nil {
		input.Context = map[string]any{}
	}
	if input.DepartmentID != nil {
		input.Context["department_id"] = input.DepartmentID.String()
	}
	if len(input.RequiredCapabilities) > 0 {
		input.Context["required_capabilities"] = input.RequiredCapabilities
	}
	if input.RiskLevel == "" {
		input.RiskLevel = "low"
	}
	return &CapabilityMatchBridge{
		DepartmentID:         input.DepartmentID,
		TaskDescription:      input.TaskDescription,
		RequiredCapabilities: input.RequiredCapabilities,
		RequiredLevel:        input.RequiredLevel,
		RiskLevel:            input.RiskLevel,
		CapabilityMatchPath:  "/api/v1/capabilities/match",
		ContextWeightPath:    "/api/v1/evolution/context-weights/compute",
		AccessDecisionPath:   "/api/v1/governance/access/decide",
		WorkflowStartPath:    "/api/v1/workflows/instances",
		Context:              input.Context,
	}, nil
}

func normalizeDepartmentInput(input *CreateDepartmentInput) {
	if input.Status == "" {
		input.Status = "active"
	}
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
}

func normalizePositionInput(input *CreatePositionInput) {
	if input.Status == "" {
		input.Status = "active"
	}
	if input.PermissionLevel == "" {
		input.PermissionLevel = "L1"
	}
	if input.RequiredCapabilities == nil {
		input.RequiredCapabilities = []string{}
	}
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
}

func normalizePositionAssignmentInput(input *CreatePositionAssignmentInput) {
	if input.AssignmentType == "" {
		input.AssignmentType = "candidate"
	}
	if input.AllocationPercent <= 0 {
		input.AllocationPercent = 100
	}
	if input.Status == "" {
		input.Status = "active"
	}
	if input.Metadata == nil {
		input.Metadata = map[string]any{}
	}
}

func isValidPositionActorType(actorType string) bool {
	switch actorType {
	case "internal_human", "external_human", "internal_agent", "external_agent":
		return true
	default:
		return false
	}
}

func matchesMemberTypes(actorType string, memberTypes []string) bool {
	if len(memberTypes) == 0 {
		return true
	}
	memberType := memberTypeFromActorType(actorType)
	for _, allowed := range memberTypes {
		if allowed == memberType || allowed == actorType {
			return true
		}
	}
	return false
}

func memberTypeFromActorType(actorType string) string {
	switch actorType {
	case "internal_human":
		return "internal"
	case "external_human":
		return "external"
	case "internal_agent", "external_agent":
		return "agent"
	default:
		return actorType
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func capabilityOverlapScore(positionCapabilities, requiredCapabilities []string) float64 {
	if len(positionCapabilities) == 0 || len(requiredCapabilities) == 0 {
		return 0
	}
	lookup := make(map[string]struct{}, len(positionCapabilities))
	for _, capability := range positionCapabilities {
		lookup[strings.ToLower(strings.TrimSpace(capability))] = struct{}{}
	}
	matches := 0
	for _, capability := range requiredCapabilities {
		if _, ok := lookup[strings.ToLower(strings.TrimSpace(capability))]; ok {
			matches++
		}
	}
	return float64(matches) / float64(len(requiredCapabilities)) * 0.12
}

func clampScore(score float64) float64 {
	if score > 1 {
		return 1
	}
	if score < 0 {
		return 0
	}
	return score
}

func mergeMetadata(base map[string]any, extra map[string]any) map[string]any {
	if base == nil {
		base = map[string]any{}
	}
	for key, value := range extra {
		base[key] = value
	}
	return base
}

func isValidOrgStatus(status string) bool {
	switch status {
	case "active", "inactive", "archived":
		return true
	default:
		return false
	}
}

func isValidMemberType(memberType string) bool {
	switch memberType {
	case "internal", "external", "agent":
		return true
	default:
		return false
	}
}

func validateMembershipActor(input AddOrganizationMemberInput) error {
	if !isValidMemberType(input.MemberType) {
		return fmt.Errorf("%w: invalid member type", ErrValidation)
	}
	switch input.MemberType {
	case "internal":
		if input.UserID == nil || input.ExternalMemberID != nil || input.AgentID != nil {
			return fmt.Errorf("%w: internal membership requires only user_id", ErrValidation)
		}
	case "external":
		if input.ExternalMemberID == nil || input.UserID != nil || input.AgentID != nil {
			return fmt.Errorf("%w: external membership requires only external_member_id", ErrValidation)
		}
	case "agent":
		if input.AgentID == nil || input.UserID != nil || input.ExternalMemberID != nil {
			return fmt.Errorf("%w: agent membership requires only agent_id", ErrValidation)
		}
	}
	return nil
}

func membershipActorID(membership OrganizationMembership) (uuid.UUID, bool) {
	switch membership.MemberType {
	case "internal":
		if membership.UserID != nil {
			return *membership.UserID, true
		}
	case "external":
		if membership.ExternalMemberID != nil {
			return *membership.ExternalMemberID, true
		}
	case "agent":
		if membership.AgentID != nil {
			return *membership.AgentID, true
		}
	}
	return uuid.Nil, false
}

func membershipActorType(membership OrganizationMembership) string {
	switch membership.MemberType {
	case "internal":
		return "internal_human"
	case "external":
		return "external_human"
	case "agent":
		if origin, ok := membership.Metadata["agent_origin"].(string); ok && origin == "external" {
			return "external_agent"
		}
		return "internal_agent"
	default:
		return membership.MemberType
	}
}

func scoreMembership(membership OrganizationMembership, input MatchMembersInput) (float64, string) {
	score := 0.45
	reasons := []string{"department member"}
	if membership.Status == "active" {
		score += 0.25
		reasons = append(reasons, "active")
	}
	if titleMatchesTask(membership.Title, input.TaskDescription) {
		score += 0.15
		reasons = append(reasons, "title matches task")
	}
	if len(input.RequiredCapabilities) > 0 && membership.MemberType == "agent" {
		score += 0.1
		reasons = append(reasons, "agent capability candidate")
	}
	if input.WorkflowTemplateID != nil {
		score += 0.05
		reasons = append(reasons, "workflow context available")
	}
	if score > 1 {
		score = 1
	}
	return score, strings.Join(reasons, ", ")
}

func titleMatchesTask(title, task string) bool {
	title = strings.ToLower(title)
	task = strings.ToLower(task)
	for _, word := range strings.Fields(task) {
		if len(word) > 3 && strings.Contains(title, word) {
			return true
		}
	}
	return false
}

func normalizeRiskLevel(riskLevel string) string {
	switch riskLevel {
	case "low", "medium", "high", "critical":
		return riskLevel
	default:
		return "low"
	}
}

func uuidString(id *uuid.UUID) string {
	if id == nil {
		return ""
	}
	return id.String()
}
