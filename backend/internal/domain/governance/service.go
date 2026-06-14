package governance

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

func (s *Service) CreatePermission(ctx context.Context, p *Permission) (*Permission, error) {
	if p.Name == "" {
		return nil, fmt.Errorf("%w: name is required", ErrValidation)
	}
	if p.Level < 1 || p.Level > 4 {
		return nil, fmt.Errorf("%w: level must be between 1 and 4", ErrValidation)
	}
	if p.Behavior != "auto" && p.Behavior != "notify" && p.Behavior != "approve" && p.Behavior != "deny" {
		p.Behavior = "notify"
	}
	return s.repo.CreatePermission(ctx, p)
}

func (s *Service) ListPermissions(ctx context.Context) ([]Permission, error) {
	return s.repo.ListPermissions(ctx)
}

func (s *Service) CreatePrinciple(ctx context.Context, input CreatePrincipleInput) (*Principle, error) {
	if input.Name == "" || input.Description == "" {
		return nil, fmt.Errorf("%w: name and description are required", ErrValidation)
	}
	return s.repo.CreatePrinciple(ctx, input)
}

func (s *Service) ListPrinciples(ctx context.Context) ([]Principle, error) {
	return s.repo.ListPrinciples(ctx)
}

func (s *Service) GetPrinciple(ctx context.Context, id uuid.UUID) (*Principle, error) {
	return s.repo.GetPrinciple(ctx, id)
}

func (s *Service) CreateControlRule(ctx context.Context, input CreateControlRuleInput) (*ControlRule, error) {
	if input.Action == "" {
		return nil, fmt.Errorf("%w: action is required", ErrValidation)
	}
	return s.repo.CreateControlRule(ctx, input)
}

func (s *Service) ListControlRules(ctx context.Context) ([]ControlRule, error) {
	return s.repo.ListControlRules(ctx)
}

func (s *Service) CheckPermission(ctx context.Context, input PermissionCheckInput) (*PermissionCheckResult, error) {
	decision, err := s.DecideAccess(ctx, AccessDecisionInput{
		ActorID:       input.UserID,
		ActorType:     "internal_human",
		Action:        input.Action,
		Resource:      input.Resource,
		ResourceID:    input.ResourceID,
		RequiredLevel: "L1",
		RiskLevel:     "low",
	})
	if err != nil {
		return nil, err
	}
	return &PermissionCheckResult{
		Allowed:  decision.Allowed,
		Level:    permissionLevelWeight(decision.RequiredLevel),
		Behavior: decision.Behavior,
		Reason:   decision.Reason,
	}, nil
}

func (s *Service) DecideAccess(ctx context.Context, input AccessDecisionInput) (*AccessDecision, error) {
	if input.ActorID == uuid.Nil {
		return nil, fmt.Errorf("%w: actor_id is required", ErrValidation)
	}
	if input.ActorType == "" || input.Action == "" || input.Resource == "" {
		return nil, fmt.Errorf("%w: actor_type, action, and resource are required", ErrValidation)
	}
	if input.RequiredLevel == "" {
		input.RequiredLevel = "L1"
	}
	if input.RiskLevel == "" {
		input.RiskLevel = "low"
	}
	if input.Context == nil {
		input.Context = map[string]any{}
	}

	behavior := "notify"
	matchedRules := []string{"audit-first-default"}
	if p, err := s.repo.GetPermissionByLevel(ctx, permissionLevelWeight(input.RequiredLevel)); err == nil && p.Behavior != "" {
		behavior = p.Behavior
		matchedRules = append(matchedRules, "permission-level:"+p.Name)
	}

	decision := behavior
	reason := "audit allowed"
	switch behavior {
	case "auto":
		decision = "allow"
		reason = "permission behavior auto"
	case "notify":
		decision = "notify"
		reason = "permission behavior notify"
	case "approve":
		decision = "approve"
		reason = "permission behavior requires approval"
	case "deny":
		decision = "deny"
		reason = "permission behavior denies access"
	}

	riskWeight := riskLevelWeight(input.RiskLevel)
	requiredWeight := permissionLevelWeight(input.RequiredLevel)
	if decision == "allow" || decision == "notify" {
		if riskWeight >= 3 || requiredWeight >= 4 {
			decision = "approve"
			behavior = "approve"
			reason = "high risk or L4 action requires human approval"
			matchedRules = append(matchedRules, "high-risk-approval")
		}
		if input.WeightSnapshot != nil && *input.WeightSnapshot < 0.35 && riskWeight >= 2 {
			decision = "approve"
			behavior = "approve"
			reason = "actor context weight below approval threshold"
			matchedRules = append(matchedRules, "low-weight-approval")
		}
		if input.WeightSnapshot != nil && *input.WeightSnapshot < 0.20 {
			decision = "deny"
			behavior = "deny"
			reason = "actor context weight below deny threshold"
			matchedRules = append(matchedRules, "low-weight-deny")
		}
	}
	if input.ActorType == "external_agent" && riskWeight >= 4 {
		decision = "approve"
		behavior = "approve"
		reason = "external service agent on critical risk requires human approval"
		matchedRules = append(matchedRules, "external-agent-critical-approval")
	}

	allowed := decision == "allow" || decision == "notify"
	return s.repo.CreateAccessDecision(ctx, input, decision, behavior, reason, allowed, matchedRules)
}

func (s *Service) ListAccessDecisions(ctx context.Context, limit int) ([]AccessDecision, error) {
	return s.repo.ListAccessDecisions(ctx, limit)
}

func permissionLevelWeight(level string) int {
	switch level {
	case "L1":
		return 1
	case "L2":
		return 2
	case "L3":
		return 3
	case "L4":
		return 4
	default:
		return 1
	}
}

func riskLevelWeight(level string) int {
	switch level {
	case "low":
		return 1
	case "medium":
		return 2
	case "high":
		return 3
	case "critical":
		return 4
	default:
		return 1
	}
}
