package assistant

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type DictionaryRepository interface {
	CreateDictionaryVersion(context.Context, DictionaryImportModel, *uuid.UUID) (uuid.UUID, error)
	CreateContextChangeProposal(context.Context, ContextChangeProposalInput) (uuid.UUID, error)
	CreateContextMigrationDraft(context.Context, ContextMigrationDraftInput) (uuid.UUID, error)
}

type SuggestionProvider interface {
	SuggestDictionaryChanges(context.Context, DictionaryImportModel) (map[string]any, error)
}

type DictionaryService struct {
	repo       DictionaryRepository
	suggestion SuggestionProvider
}

func NewDictionaryService(repo DictionaryRepository, suggestion SuggestionProvider) *DictionaryService {
	return &DictionaryService{repo: repo, suggestion: suggestion}
}

type DictionaryImportRequest struct {
	Model      DictionaryImportModel
	ImportedBy *uuid.UUID
}

type DictionaryImportResult struct {
	DictionaryVersionID uuid.UUID                  `json:"dictionary_version_id"`
	Validation          DictionaryValidationResult `json:"validation"`
	ProposalID          *uuid.UUID                 `json:"proposal_id,omitempty"`
	MigrationDraftIDs   []uuid.UUID                `json:"migration_draft_ids"`
}

type DictionaryValidationResult struct {
	Errors   []DictionaryValidationIssue `json:"errors"`
	Warnings []DictionaryValidationIssue `json:"warnings"`
}

type DictionaryValidationIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path"`
}

type ContextChangeProposalInput struct {
	DictionaryVersionID uuid.UUID
	ProposalType        string
	Title               string
	Summary             string
	Payload             map[string]any
	Status              string
}

type ContextMigrationDraftInput struct {
	DictionaryVersionID uuid.UUID
	Title               string
	Summary             string
	SQLUp               string
	SQLDown             string
	RiskLevel           string
	Metadata            map[string]any
}

func (s *DictionaryService) ValidateImport(model DictionaryImportModel) (DictionaryValidationResult, error) {
	result := DictionaryValidationResult{}
	if model.VersionKey == "" {
		result.Errors = append(result.Errors, DictionaryValidationIssue{Code: "missing_version_key", Message: "version_key is required", Path: "version_key"})
	}
	if model.ScopeLevel != ContextScopeSaaS && model.ScopeLevel != ContextScopeOrganization && model.ScopeLevel != ContextScopeModule {
		result.Errors = append(result.Errors, DictionaryValidationIssue{Code: "invalid_scope_level", Message: "scope_level must be saas, organization, or module", Path: "scope_level"})
	}
	entities := map[string]bool{}
	for _, entity := range model.Entities {
		if entity.EntityKey == "" {
			result.Errors = append(result.Errors, DictionaryValidationIssue{Code: "missing_entity_key", Message: "entity_key is required", Path: "entities"})
			continue
		}
		entities[entity.EntityKey] = true
	}
	for _, field := range model.Fields {
		if field.FieldKey == "" {
			result.Errors = append(result.Errors, DictionaryValidationIssue{Code: "missing_field_key", Message: "field_key is required", Path: "fields"})
		}
		if !entities[field.EntityKey] {
			result.Errors = append(result.Errors, DictionaryValidationIssue{Code: "unknown_entity", Message: "field references an unknown entity", Path: field.EntityKey + "." + field.FieldKey})
		}
		if field.IsFinanceField && field.BaseWeight > 0 && len(model.FinanceValidationProfiles) == 0 {
			result.Warnings = append(result.Warnings, DictionaryValidationIssue{Code: "finance_validation_missing", Message: "finance field has no validation profile", Path: field.EntityKey + "." + field.FieldKey})
		}
	}
	if len(result.Errors) > 0 {
		return result, fmt.Errorf("%w: dictionary import validation failed", ErrValidation)
	}
	return result, nil
}

func (s *DictionaryService) Import(ctx context.Context, input DictionaryImportRequest) (*DictionaryImportResult, error) {
	validation, err := s.ValidateImport(input.Model)
	if err != nil {
		return &DictionaryImportResult{Validation: validation}, err
	}
	if s.repo == nil {
		return nil, fmt.Errorf("%w: dictionary repository is not configured", ErrValidation)
	}
	versionID, err := s.repo.CreateDictionaryVersion(ctx, input.Model, input.ImportedBy)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{"version_key": input.Model.VersionKey, "module_key": input.Model.ModuleKey}
	if s.suggestion != nil {
		if suggestions, suggestErr := s.suggestion.SuggestDictionaryChanges(ctx, input.Model); suggestErr == nil {
			payload["ai_suggestions"] = suggestions
		}
	}
	proposalID, err := s.repo.CreateContextChangeProposal(ctx, ContextChangeProposalInput{
		DictionaryVersionID: versionID,
		ProposalType:        "dictionary_change",
		Title:               "Dictionary import " + input.Model.VersionKey,
		Summary:             "Review imported context dictionary before activation",
		Payload:             payload,
		Status:              ProposalPending,
	})
	if err != nil {
		return nil, err
	}
	draftIDs := []uuid.UUID{}
	for _, intent := range input.Model.MigrationIntents {
		draftID, err := s.repo.CreateContextMigrationDraft(ctx, ContextMigrationDraftInput{
			DictionaryVersionID: versionID,
			Title:               "Migration draft for " + intent.EntityKey + "." + intent.FieldKey,
			Summary:             intent.Reason,
			SQLUp:               "-- " + intent.IntentType + " " + intent.EntityKey + "." + intent.FieldKey,
			SQLDown:             "-- rollback " + intent.IntentType + " " + intent.EntityKey + "." + intent.FieldKey,
			RiskLevel:           "medium",
			Metadata:            map[string]any{"intent_type": intent.IntentType},
		})
		if err != nil {
			return nil, err
		}
		draftIDs = append(draftIDs, draftID)
	}
	return &DictionaryImportResult{DictionaryVersionID: versionID, Validation: validation, ProposalID: &proposalID, MigrationDraftIDs: draftIDs}, nil
}
