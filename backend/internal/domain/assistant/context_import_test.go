package assistant

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNormalizeDictionaryImportJSON(t *testing.T) {
	raw := `{"version_key":"v1","source_type":"json","scope_level":"module","module_key":"project","domains":[{"module_key":"project","name":"Project","scope_level":"module"}],"entities":[{"entity_key":"project","module_key":"project","display_name":"Project"}],"fields":[{"entity_key":"project","field_key":"status","display_name":"Status","data_type":"string","base_weight":3,"table_name":"projects","column_name":"status"}]}`

	model, err := NormalizeDictionaryImport(DictionaryImportSource{
		SourceType: ContextSourceJSON,
		SourceName: "project.json",
		Content:    []byte(raw),
	})
	if err != nil {
		t.Fatalf("NormalizeDictionaryImport returned error: %v", err)
	}
	if model.VersionKey != "v1" || model.ModuleKey != "project" {
		t.Fatalf("model = %s/%s, want v1/project", model.VersionKey, model.ModuleKey)
	}
	if len(model.Fields) != 1 || model.Fields[0].FieldKey != "status" {
		t.Fatalf("fields = %#v, want status", model.Fields)
	}
}

func TestNormalizeDictionaryImportCSV(t *testing.T) {
	csv := strings.Join([]string{
		"module_key,entity_key,field_key,display_name,data_type,base_weight,table_name,column_name,is_finance_field",
		"finance,cost_ledger_entry,amount,Amount,number,8,cost_ledger_entries,amount,true",
	}, "\n")

	model, err := NormalizeDictionaryImport(DictionaryImportSource{
		SourceType: ContextSourceCSV,
		SourceName: "finance.csv",
		Content:    []byte(csv),
		ScopeLevel: ContextScopeModule,
		ModuleKey:  "finance",
		VersionKey: "finance-v1",
	})
	if err != nil {
		t.Fatalf("NormalizeDictionaryImport returned error: %v", err)
	}
	if len(model.Fields) != 1 {
		t.Fatalf("fields len = %d, want 1", len(model.Fields))
	}
	field := model.Fields[0]
	if field.FieldKey != "amount" || !field.IsFinanceField || field.BaseWeight != 8 {
		t.Fatalf("field = %#v, want amount finance weight 8", field)
	}
}

func TestDictionaryServiceRejectsFieldWithoutEntity(t *testing.T) {
	svc := NewDictionaryService(nil, nil)
	model := DictionaryImportModel{
		VersionKey: "bad-v1",
		SourceType: ContextSourceJSON,
		ScopeLevel: ContextScopeModule,
		ModuleKey:  "project",
		Fields:     []ContextFieldInput{{EntityKey: "missing", FieldKey: "status"}},
	}

	result, err := svc.ValidateImport(model)
	if err == nil {
		t.Fatalf("ValidateImport returned nil error")
	}
	if len(result.Errors) != 1 {
		t.Fatalf("errors len = %d, want 1", len(result.Errors))
	}
	if result.Errors[0].Code != "unknown_entity" {
		t.Fatalf("error code = %s, want unknown_entity", result.Errors[0].Code)
	}
}

func TestDictionaryServiceCreatesMigrationDraftForIntent(t *testing.T) {
	repo := &fakeDictionaryRepository{}
	svc := NewDictionaryService(repo, nil)
	model := DictionaryImportModel{
		VersionKey: "project-v1",
		SourceType: ContextSourceJSON,
		ScopeLevel: ContextScopeModule,
		ModuleKey:  "project",
		Domains:    []ContextBusinessDomainInput{{ModuleKey: "project", Name: "Project", ScopeLevel: ContextScopeModule}},
		Entities:   []ContextEntityInput{{EntityKey: "project", ModuleKey: "project"}},
		Fields:     []ContextFieldInput{{EntityKey: "project", FieldKey: "priority", TableName: "projects", ColumnName: "priority"}},
		MigrationIntents: []ContextMigrationIntentInput{
			{IntentType: "add_column", EntityKey: "project", FieldKey: "priority", Reason: "prioritize work"},
		},
	}

	created, err := svc.Import(context.Background(), DictionaryImportRequest{Model: model})
	if err != nil {
		t.Fatalf("Import returned error: %v", err)
	}
	if created.DictionaryVersionID == uuid.Nil {
		t.Fatalf("dictionary version id is nil")
	}
	if len(repo.migrationDrafts) != 1 {
		t.Fatalf("migration drafts = %d, want 1", len(repo.migrationDrafts))
	}
	if repo.migrationDrafts[0].RiskLevel != "medium" {
		t.Fatalf("risk level = %s, want medium", repo.migrationDrafts[0].RiskLevel)
	}
}

type fakeDictionaryRepository struct {
	versionID       uuid.UUID
	proposals       []ContextChangeProposalInput
	migrationDrafts []ContextMigrationDraftInput
}

func (f *fakeDictionaryRepository) CreateDictionaryVersion(context.Context, DictionaryImportModel, *uuid.UUID) (uuid.UUID, error) {
	if f.versionID == uuid.Nil {
		f.versionID = uuid.New()
	}
	return f.versionID, nil
}

func (f *fakeDictionaryRepository) CreateContextChangeProposal(_ context.Context, input ContextChangeProposalInput) (uuid.UUID, error) {
	f.proposals = append(f.proposals, input)
	return uuid.New(), nil
}

func (f *fakeDictionaryRepository) CreateContextMigrationDraft(_ context.Context, input ContextMigrationDraftInput) (uuid.UUID, error) {
	f.migrationDrafts = append(f.migrationDrafts, input)
	return uuid.New(), nil
}
