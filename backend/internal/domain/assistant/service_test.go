package assistant

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/selfevo-AI/meta-org/backend/internal/pkg/middleware"
)

func TestCreateSessionAutoModelUsesModuleDefault(t *testing.T) {
	agentID := uuid.New()
	providerID := uuid.New()
	repo := &fakeRepository{
		moduleDefault: &ModuleDefault{
			ModuleKey:    "project",
			TargetType:   "project",
			AgentID:      &agentID,
			ProviderID:   &providerID,
			ProviderType: "openai",
			Model:        "gpt-4o-mini",
		},
	}
	svc := NewService(repo, nil, nil)

	actorID := uuid.New()
	session, err := svc.CreateSession(context.Background(), actorID, "internal_human", CreateSessionInput{
		Title:      "project plan",
		ModuleKey:  "project",
		TargetType: "project",
		TargetID:   uuidPtr(uuid.New()),
		AutoModel:  true,
	})
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}

	if session.AgentID == nil || *session.AgentID != agentID {
		t.Fatalf("session agent = %v, want %s", session.AgentID, agentID)
	}
	if repo.createdSession.ProviderID == nil || *repo.createdSession.ProviderID != providerID {
		t.Fatalf("provider id = %v, want %s", repo.createdSession.ProviderID, providerID)
	}
	if repo.createdSession.ProviderType != "openai" || repo.createdSession.Model != "gpt-4o-mini" {
		t.Fatalf("model config = %s/%s, want openai/gpt-4o-mini", repo.createdSession.ProviderType, repo.createdSession.Model)
	}
}

func TestCreateSessionAutoModelFallsBackWhenModuleDefaultMissing(t *testing.T) {
	agentID := uuid.New()
	repo := &fakeRepository{
		moduleDefaultErr: pgx.ErrNoRows,
		fallbackDefault: &ModuleDefault{
			AgentID:      &agentID,
			ProviderType: "openai",
			Model:        "gpt-4o-mini",
		},
	}
	svc := NewService(repo, nil, nil)

	session, err := svc.CreateSession(context.Background(), uuid.New(), "internal_human", CreateSessionInput{
		Title:     "general request",
		ModuleKey: "meta_org",
		AutoModel: true,
	})
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if session.ProviderType != "openai" || session.Model != "gpt-4o-mini" {
		t.Fatalf("model config = %s/%s, want openai/gpt-4o-mini", session.ProviderType, session.Model)
	}
	if session.AgentID == nil || *session.AgentID != agentID {
		t.Fatalf("session agent = %v, want %s", session.AgentID, agentID)
	}
}

func TestCreateSessionUsesCurrentTenantOrganization(t *testing.T) {
	orgID := uuid.New()
	ctx := context.WithValue(context.Background(), middleware.TenantContextKey, &middleware.TenantContext{OrganizationID: &orgID})
	repo := &fakeRepository{}
	svc := NewService(repo, nil, nil)

	session, err := svc.CreateSession(ctx, uuid.New(), "internal_human", CreateSessionInput{Title: "tenant session"})
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if repo.createdSession.OrganizationID == nil || *repo.createdSession.OrganizationID != orgID {
		t.Fatalf("created session organization = %v, want %s", repo.createdSession.OrganizationID, orgID)
	}
	if session.OrganizationID == nil || *session.OrganizationID != orgID {
		t.Fatalf("session organization = %v, want %s", session.OrganizationID, orgID)
	}
}

func TestCreateSessionRejectsCrossTenantOrganization(t *testing.T) {
	orgID := uuid.New()
	otherOrgID := uuid.New()
	ctx := context.WithValue(context.Background(), middleware.TenantContextKey, &middleware.TenantContext{OrganizationID: &orgID})
	svc := NewService(&fakeRepository{}, nil, nil)

	_, err := svc.CreateSession(ctx, uuid.New(), "internal_human", CreateSessionInput{
		Title:          "cross tenant",
		OrganizationID: &otherOrgID,
	})
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("CreateSession error = %v, want ErrForbidden", err)
	}
}

func TestConfirmProposalRequiresHumanAndIsIdempotent(t *testing.T) {
	proposalID := uuid.New()
	repo := &fakeRepository{
		proposal: &Proposal{
			ID:           proposalID,
			SessionID:    uuid.New(),
			ModuleKey:    "project",
			TargetType:   "project",
			TargetID:     uuidPtr(uuid.New()),
			ProposalType: "metadata_patch",
			Status:       ProposalPending,
			Payload:      map[string]any{"summary": "ship the plan"},
		},
	}
	applicator := &fakeProposalApplicator{}
	svc := NewService(repo, nil, nil, WithProposalApplicator(applicator))

	if _, err := svc.ConfirmProposal(context.Background(), proposalID, uuid.New(), "internal_agent"); err == nil {
		t.Fatalf("ConfirmProposal allowed a non-human actor")
	}

	reviewerID := uuid.New()
	applied, err := svc.ConfirmProposal(context.Background(), proposalID, reviewerID, "internal_human")
	if err != nil {
		t.Fatalf("ConfirmProposal returned error: %v", err)
	}
	if applied.Status != ProposalApplied {
		t.Fatalf("proposal status = %q, want %q", applied.Status, ProposalApplied)
	}
	if applicator.calls != 1 {
		t.Fatalf("applicator calls = %d, want 1", applicator.calls)
	}

	appliedAgain, err := svc.ConfirmProposal(context.Background(), proposalID, reviewerID, "internal_human")
	if err != nil {
		t.Fatalf("ConfirmProposal idempotent call returned error: %v", err)
	}
	if appliedAgain.Status != ProposalApplied {
		t.Fatalf("idempotent status = %q, want %q", appliedAgain.Status, ProposalApplied)
	}
	if applicator.calls != 1 {
		t.Fatalf("applicator calls after idempotent confirm = %d, want 1", applicator.calls)
	}
}

func TestConfirmProposalRequiresSessionAccess(t *testing.T) {
	proposalID := uuid.New()
	repo := &fakeRepository{
		proposal: &Proposal{
			ID:           proposalID,
			SessionID:    uuid.New(),
			ModuleKey:    "project",
			TargetType:   "project",
			TargetID:     uuidPtr(uuid.New()),
			ProposalType: "metadata_patch",
			Status:       ProposalPending,
			Payload:      map[string]any{"summary": "ship the plan"},
		},
		sessionErr: ErrNotFound,
	}
	applicator := &fakeProposalApplicator{}
	svc := NewService(repo, nil, nil, WithProposalApplicator(applicator))

	if _, err := svc.ConfirmProposal(context.Background(), proposalID, uuid.New(), "internal_human"); err == nil {
		t.Fatalf("ConfirmProposal allowed a reviewer without session access")
	}
	if applicator.calls != 0 {
		t.Fatalf("applicator calls = %d, want 0", applicator.calls)
	}
}

func TestBusinessSkillLifecycle(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo, nil, nil)

	created, err := svc.CreateBusinessSkill(context.Background(), uuid.New(), "internal_human", CreateBusinessSkillInput{
		ModuleKey:      "project",
		TargetType:     "project",
		Name:           "project risk reviewer",
		Description:    "review project delivery risk",
		TriggerIntent:  "review risk",
		PromptTemplate: "Review {{target.title}}",
		ToolAllowlist:  []string{"project.update"},
	})
	if err != nil {
		t.Fatalf("CreateBusinessSkill returned error: %v", err)
	}
	if created.Status != SkillDraft {
		t.Fatalf("created status = %q, want %q", created.Status, SkillDraft)
	}

	activated, err := svc.ActivateBusinessSkill(context.Background(), created.ID, uuid.New(), "internal_human")
	if err != nil {
		t.Fatalf("ActivateBusinessSkill returned error: %v", err)
	}
	if activated.Status != SkillActive {
		t.Fatalf("activated status = %q, want %q", activated.Status, SkillActive)
	}

	items, err := svc.ListBusinessSkills(context.Background(), "project", "")
	if err != nil {
		t.Fatalf("ListBusinessSkills returned error: %v", err)
	}
	if len(items) != 1 || items[0].ID != created.ID {
		t.Fatalf("listed skills = %v, want created skill", items)
	}
}

func TestListContextTargetsPassesTargetType(t *testing.T) {
	resolver := &fakeContextResolver{
		result: WorkRecordContext{
			ModuleKey: "finance",
			Records:   []WorkRecord{{ID: uuid.New().String(), Type: "finance_receivable"}},
		},
	}
	svc := NewService(&fakeRepository{}, nil, nil, WithContextResolver(resolver))

	records, err := svc.ListContextTargets(context.Background(), "finance", "finance_receivable", 10)
	if err != nil {
		t.Fatalf("ListContextTargets returned error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("records len = %d, want 1", len(records))
	}
	if resolver.session == nil || resolver.session.TargetType != "finance_receivable" {
		t.Fatalf("resolver target type = %v, want finance_receivable", resolver.session)
	}
}

func TestListBusinessSkillsFiltersByTargetType(t *testing.T) {
	repo := &fakeRepository{}
	svc := NewService(repo, nil, nil)

	if _, err := svc.ListBusinessSkills(context.Background(), "finance", "finance_receivable"); err != nil {
		t.Fatalf("ListBusinessSkills returned error: %v", err)
	}
	if repo.lastSkillListModuleKey != "finance" || repo.lastSkillListTargetType != "finance_receivable" {
		t.Fatalf("skill filters = %s/%s, want finance/finance_receivable", repo.lastSkillListModuleKey, repo.lastSkillListTargetType)
	}
}

func TestRunBusinessSkillRecordsSessionAndTargetContext(t *testing.T) {
	skillID := uuid.New()
	sessionID := uuid.New()
	targetID := uuid.New()
	repo := &fakeRepository{
		skill: &BusinessSkill{
			ID:             skillID,
			ModuleKey:      "finance",
			TargetType:     "finance_receivable",
			Name:           "receivable reviewer",
			PromptTemplate: "Review receivable",
			Status:         SkillActive,
		},
	}
	svc := NewService(repo, nil, nil)

	run, err := svc.RunBusinessSkill(context.Background(), skillID, uuid.New(), "internal_human", map[string]any{
		"session_id":  sessionID.String(),
		"target_type": "finance_receivable",
		"target_id":   targetID.String(),
	})
	if err != nil {
		t.Fatalf("RunBusinessSkill returned error: %v", err)
	}
	if run.SessionID == nil || *run.SessionID != sessionID {
		t.Fatalf("run session id = %v, want %s", run.SessionID, sessionID)
	}
	if run.TargetID == nil || *run.TargetID != targetID {
		t.Fatalf("run target id = %v, want %s", run.TargetID, targetID)
	}
	if run.TargetType != "finance_receivable" {
		t.Fatalf("run target type = %q, want finance_receivable", run.TargetType)
	}
}

func TestSystemPromptIncludesSelectedTarget(t *testing.T) {
	targetID := uuid.New()
	prompt := systemPrompt(&Session{
		Mode:          ModeBusinessProcess,
		ModuleKey:     "project",
		TargetType:    "project",
		TargetID:      &targetID,
		WorkingMemory: map[string]any{},
	}, nil, WorkRecordContext{})

	if !strings.Contains(prompt, "target_type=project") {
		t.Fatalf("prompt does not include target_type: %s", prompt)
	}
	if !strings.Contains(prompt, "target_id="+targetID.String()) {
		t.Fatalf("prompt does not include target_id: %s", prompt)
	}
}

func TestSystemPromptIncludesWorkRecordData(t *testing.T) {
	prompt := systemPrompt(&Session{
		Mode:          ModeBusinessProcess,
		ModuleKey:     "finance",
		WorkingMemory: map[string]any{},
	}, nil, WorkRecordContext{
		ModuleKey: "finance",
		Records: []WorkRecord{
			{
				ID:     uuid.New().String(),
				Type:   "finance_receivable",
				Title:  "INV-001",
				Status: "unpaid",
				Data: map[string]any{
					"customer_name": "Acme",
					"amount":        float64(1200),
				},
			},
		},
	})

	if !strings.Contains(prompt, "customer_name") || !strings.Contains(prompt, "Acme") {
		t.Fatalf("prompt does not include work record data: %s", prompt)
	}
	if !strings.Contains(prompt, "amount") || !strings.Contains(prompt, "1200") {
		t.Fatalf("prompt does not include numeric work record data: %s", prompt)
	}
}

type fakeRepository struct {
	createdSession          CreateSessionInput
	moduleDefault           *ModuleDefault
	moduleDefaultErr        error
	fallbackDefault         *ModuleDefault
	proposal                *Proposal
	skill                   *BusinessSkill
	lastSkillRun            CreateSkillRunInput
	lastSkillListModuleKey  string
	lastSkillListTargetType string
	lastStep                AddStepInput
	session                 *Session
	sessionErr              error
}

func (f *fakeRepository) CreateSession(_ context.Context, actorID uuid.UUID, actorType string, input CreateSessionInput) (*Session, error) {
	f.createdSession = input
	return &Session{
		ID:             uuid.New(),
		Title:          input.Title,
		Mode:           input.Mode,
		ModuleKey:      input.ModuleKey,
		Status:         StatusIdle,
		ActorID:        actorID,
		ActorType:      actorType,
		AgentID:        input.AgentID,
		ProviderID:     input.ProviderID,
		ProviderType:   input.ProviderType,
		Model:          input.Model,
		OrganizationID: input.OrganizationID,
		TargetType:     input.TargetType,
		TargetID:       input.TargetID,
		WorkingMemory:  map[string]any{},
		Metadata:       map[string]any{},
	}, nil
}

func (f *fakeRepository) GetModuleDefault(context.Context, string, string) (*ModuleDefault, error) {
	if f.moduleDefaultErr != nil {
		return nil, f.moduleDefaultErr
	}
	if f.moduleDefault == nil {
		return nil, ErrNotFound
	}
	return f.moduleDefault, nil
}

func (f *fakeRepository) FindDefaultModel(context.Context) (*ModuleDefault, error) {
	if f.fallbackDefault != nil {
		return f.fallbackDefault, nil
	}
	return &ModuleDefault{ProviderType: "openai", Model: "gpt-4o-mini"}, nil
}

func (f *fakeRepository) ListSessions(context.Context, uuid.UUID, string, string, int) ([]Session, error) {
	return []Session{}, nil
}

func (f *fakeRepository) GetSession(_ context.Context, id uuid.UUID, actorID uuid.UUID, actorType string) (*Session, error) {
	if f.sessionErr != nil {
		return nil, f.sessionErr
	}
	if f.session != nil {
		return f.session, nil
	}
	return &Session{ID: id, ActorID: actorID, ActorType: actorType, WorkingMemory: map[string]any{}}, nil
}

func (f *fakeRepository) UpdateSessionStatus(context.Context, uuid.UUID, string, string) error {
	return nil
}

func (f *fakeRepository) UpdateWorkingMemory(context.Context, uuid.UUID, map[string]any) error {
	return nil
}

func (f *fakeRepository) AddMessage(context.Context, uuid.UUID, string, string, string, string, map[string]any) (*Message, error) {
	return &Message{}, nil
}

func (f *fakeRepository) ListMessages(context.Context, uuid.UUID, int) ([]Message, error) {
	return []Message{}, nil
}

func (f *fakeRepository) AddStep(_ context.Context, _ *Session, input AddStepInput) (*Step, error) {
	f.lastStep = input
	return &Step{ID: uuid.New(), StepType: input.StepType, Status: input.Status, Data: input.Data}, nil
}

func (f *fakeRepository) ListSteps(context.Context, uuid.UUID, int) ([]Step, error) {
	return []Step{}, nil
}

func (f *fakeRepository) ListScopedMemories(context.Context, Scope, uuid.UUID, string, int) ([]Memory, error) {
	return []Memory{}, nil
}

func (f *fakeRepository) CreateMemory(context.Context, CreateMemoryInput) (*Memory, error) {
	return &Memory{}, nil
}

func (f *fakeRepository) CreateProposal(_ context.Context, input CreateProposalInput) (*Proposal, error) {
	f.proposal = &Proposal{
		ID:           uuid.New(),
		SessionID:    input.SessionID,
		ModuleKey:    input.ModuleKey,
		TargetType:   input.TargetType,
		TargetID:     input.TargetID,
		ProposalType: input.ProposalType,
		Status:       ProposalPending,
		Payload:      input.Payload,
	}
	return f.proposal, nil
}

func (f *fakeRepository) ListProposals(context.Context, uuid.UUID, int) ([]Proposal, error) {
	if f.proposal == nil {
		return []Proposal{}, nil
	}
	return []Proposal{*f.proposal}, nil
}

func (f *fakeRepository) GetProposal(context.Context, uuid.UUID) (*Proposal, error) {
	if f.proposal == nil {
		return nil, ErrNotFound
	}
	return f.proposal, nil
}

func (f *fakeRepository) MarkProposalApplied(_ context.Context, id uuid.UUID, reviewerID uuid.UUID, result map[string]any) (*Proposal, error) {
	if f.proposal == nil || f.proposal.ID != id {
		return nil, ErrNotFound
	}
	f.proposal.Status = ProposalApplied
	f.proposal.ReviewerID = &reviewerID
	f.proposal.ApplyResult = result
	return f.proposal, nil
}

func (f *fakeRepository) MarkProposalRejected(_ context.Context, id uuid.UUID, reviewerID uuid.UUID, reason string) (*Proposal, error) {
	if f.proposal == nil || f.proposal.ID != id {
		return nil, ErrNotFound
	}
	f.proposal.Status = ProposalRejected
	f.proposal.ReviewerID = &reviewerID
	f.proposal.ReviewReason = reason
	return f.proposal, nil
}

func (f *fakeRepository) CreateBusinessSkill(_ context.Context, input CreateBusinessSkillInput, actorID uuid.UUID, actorType string) (*BusinessSkill, error) {
	f.skill = &BusinessSkill{
		ID:             uuid.New(),
		ModuleKey:      input.ModuleKey,
		TargetType:     input.TargetType,
		Name:           input.Name,
		Description:    input.Description,
		TriggerIntent:  input.TriggerIntent,
		PromptTemplate: input.PromptTemplate,
		ToolAllowlist:  input.ToolAllowlist,
		Status:         SkillDraft,
		CreatedBy:      &actorID,
		CreatedByType:  actorType,
	}
	return f.skill, nil
}

func (f *fakeRepository) ListBusinessSkills(_ context.Context, moduleKey string, targetType string, _ int) ([]BusinessSkill, error) {
	f.lastSkillListModuleKey = moduleKey
	f.lastSkillListTargetType = targetType
	if f.skill == nil {
		return []BusinessSkill{}, nil
	}
	return []BusinessSkill{*f.skill}, nil
}

func (f *fakeRepository) GetBusinessSkill(context.Context, uuid.UUID) (*BusinessSkill, error) {
	if f.skill == nil {
		return nil, ErrNotFound
	}
	return f.skill, nil
}

func (f *fakeRepository) ActivateBusinessSkill(_ context.Context, id uuid.UUID, reviewerID uuid.UUID) (*BusinessSkill, error) {
	if f.skill == nil || f.skill.ID != id {
		return nil, ErrNotFound
	}
	f.skill.Status = SkillActive
	f.skill.ReviewedBy = &reviewerID
	return f.skill, nil
}

func (f *fakeRepository) CreateSkillRun(_ context.Context, input CreateSkillRunInput) (*SkillRun, error) {
	f.lastSkillRun = input
	return &SkillRun{
		ID:            uuid.New(),
		SkillID:       input.SkillID,
		SessionID:     input.SessionID,
		ModuleKey:     input.ModuleKey,
		TargetType:    input.TargetType,
		TargetID:      input.TargetID,
		Input:         input.Input,
		Output:        input.Output,
		Status:        input.Status,
		ErrorMessage:  input.ErrorMessage,
		CreatedBy:     input.CreatedBy,
		CreatedByType: input.CreatedByType,
	}, nil
}

type fakeProposalApplicator struct {
	calls int
	err   error
}

func (f *fakeProposalApplicator) ApplyProposal(_ context.Context, proposal *Proposal) (map[string]any, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	if proposal == nil {
		return nil, errors.New("missing proposal")
	}
	return map[string]any{"target_type": proposal.TargetType, "target_id": uuidString(proposal.TargetID)}, nil
}

func uuidPtr(id uuid.UUID) *uuid.UUID {
	return &id
}

type fakeContextResolver struct {
	session *Session
	result  WorkRecordContext
}

func (f *fakeContextResolver) Resolve(_ context.Context, session *Session) WorkRecordContext {
	f.session = session
	return f.result
}
