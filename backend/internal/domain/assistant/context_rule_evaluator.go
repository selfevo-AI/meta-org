package assistant

import (
	"sort"

	"github.com/google/uuid"
)

const (
	ValidationVerified         = "verified"
	ValidationFinanceConflict  = "finance_conflict"
	ValidationPermissionDenied = "permission_denied"
)

type ContextRuleEvaluatorConfig struct {
	AttentionCoreRatio float64
}

type ContextRuleEvaluator struct {
	config ContextRuleEvaluatorConfig
}

type ContextRuleEvaluationInput struct {
	SessionID           uuid.UUID
	DictionaryVersionID *uuid.UUID
	Items               []ContextItem
	Omissions           []ContextOmission
	TokenBudget         int
	Validations         map[string]any
	Provenance          map[string]any
}

func NewContextRuleEvaluator(config ContextRuleEvaluatorConfig) *ContextRuleEvaluator {
	if config.AttentionCoreRatio <= 0 || config.AttentionCoreRatio > 1 {
		config.AttentionCoreRatio = 0.4
	}
	return &ContextRuleEvaluator{config: config}
}

func (e *ContextRuleEvaluator) BuildPackage(input ContextRuleEvaluationInput) ContextPackage {
	items := append([]ContextItem{}, input.Items...)
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].Weight > items[j].Weight
	})
	coreBudget := int(float64(input.TokenBudget) * e.config.AttentionCoreRatio)
	if coreBudget <= 0 {
		coreBudget = 512
	}
	pkg := ContextPackage{
		ID:                  uuid.New(),
		SessionID:           input.SessionID,
		DictionaryVersionID: input.DictionaryVersionID,
		Omissions:           append([]ContextOmission{}, input.Omissions...),
		Weights:             map[string]float64{},
		Validations:         copyMap(input.Validations),
		Provenance:          copyMap(input.Provenance),
		TokenBudget:         input.TokenBudget,
	}
	coreTokens := 0
	for _, item := range items {
		pkg.Weights[item.EntityKey+"."+item.FieldKey] = item.Weight
		switch item.ValidationState {
		case ValidationPermissionDenied:
			pkg.Omissions = append(pkg.Omissions, ContextOmission{EntityKey: item.EntityKey, FieldKey: item.FieldKey, Reason: "permission_denied"})
		case ValidationFinanceConflict:
			pkg.RiskAndSignals = append(pkg.RiskAndSignals, item)
		default:
			if coreTokens+item.EstimatedTokens <= coreBudget {
				pkg.AttentionCore = append(pkg.AttentionCore, item)
				coreTokens += item.EstimatedTokens
			} else {
				pkg.SupportingContext = append(pkg.SupportingContext, item)
			}
		}
	}
	return pkg
}
