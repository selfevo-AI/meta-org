package assistant

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ContextPackageRepository interface {
	CreateContextPackage(context.Context, ContextRequest, ContextPackage) (*ContextPackage, error)
}

type VerifiedContextEngineConfig struct {
	Resolver   ContextResolver
	Evaluator  *ContextRuleEvaluator
	Repository ContextPackageRepository
}

type VerifiedContextEngine struct {
	resolver   ContextResolver
	evaluator  *ContextRuleEvaluator
	repository ContextPackageRepository
}

func NewVerifiedContextEngine(config VerifiedContextEngineConfig) *VerifiedContextEngine {
	if config.Evaluator == nil {
		config.Evaluator = NewContextRuleEvaluator(ContextRuleEvaluatorConfig{AttentionCoreRatio: 0.4})
	}
	return &VerifiedContextEngine{resolver: config.Resolver, evaluator: config.Evaluator, repository: config.Repository}
}

func (e *VerifiedContextEngine) BuildContextPackage(ctx context.Context, request ContextRequest) (*ContextPackage, error) {
	if request.ActorID == uuid.Nil || request.ActorType == "" {
		return nil, fmt.Errorf("%w: actor is required for context package", ErrValidation)
	}
	if e.resolver == nil {
		return nil, fmt.Errorf("%w: context resolver is not configured", ErrValidation)
	}
	session := &Session{
		ID:             request.SessionID,
		ModuleKey:      normalizedModule(request.ModuleKey),
		TargetType:     request.TargetType,
		TargetID:       request.TargetID,
		ActorID:        request.ActorID,
		ActorType:      request.ActorType,
		OrganizationID: request.OrganizationID,
		WorkflowID:     request.WorkflowID,
		TaskID:         request.TaskID,
	}
	workContext := e.resolver.Resolve(ctx, session)
	if workContext.Error != "" {
		return nil, fmt.Errorf("%w: %s", ErrValidation, workContext.Error)
	}
	items := make([]ContextItem, 0, len(workContext.Records))
	for _, record := range workContext.Records {
		items = append(items, ContextItem{
			EntityKey:       record.Type,
			FieldKey:        "record",
			RecordID:        record.ID,
			Value:           map[string]any{"title": record.Title, "status": record.Status, "created_at": record.CreatedAt, "data": record.Data},
			Weight:          5,
			EstimatedTokens: estimateContextRecordTokens(record),
			ValidationState: ValidationVerified,
			Source:          "compatibility_resolver",
		})
	}
	pkg := e.evaluator.BuildPackage(ContextRuleEvaluationInput{
		SessionID:   request.SessionID,
		Items:       items,
		TokenBudget: firstPositive(request.TokenBudget, 4096),
		Validations: map[string]any{"permission": "compatibility_checked", "workflow": "compatibility_checked", "finance": "not_applicable"},
		Provenance:  map[string]any{"source": "compatibility_resolver", "module_key": normalizedModule(request.ModuleKey)},
	})
	if e.repository != nil {
		return e.repository.CreateContextPackage(ctx, request, pkg)
	}
	return &pkg, nil
}

func estimateContextRecordTokens(record WorkRecord) int {
	total := len(record.ID) + len(record.Type) + len(record.Title) + len(record.Status) + len(record.CreatedAt)
	if len(record.Data) > 0 {
		total += len(marshalToolContent(record.Data))
	}
	tokens := total / 4
	if tokens < 16 {
		return 16
	}
	return tokens
}

func firstPositive(values ...int) int {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}
	return 0
}
