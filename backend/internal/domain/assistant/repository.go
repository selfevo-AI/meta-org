package assistant

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository interface {
	CreateSession(ctx context.Context, actorID uuid.UUID, actorType string, input CreateSessionInput) (*Session, error)
	ListSessions(ctx context.Context, actorID uuid.UUID, actorType string, moduleKey string, limit int) ([]Session, error)
	GetSession(ctx context.Context, id uuid.UUID, actorID uuid.UUID, actorType string) (*Session, error)
	UpdateSessionStatus(ctx context.Context, id uuid.UUID, status string, lastError string) error
	UpdateWorkingMemory(ctx context.Context, id uuid.UUID, memory map[string]any) error
	AddMessage(ctx context.Context, sessionID uuid.UUID, role string, content string, toolCallID string, toolName string, metadata map[string]any) (*Message, error)
	ListMessages(ctx context.Context, sessionID uuid.UUID, limit int) ([]Message, error)
	AddStep(ctx context.Context, session *Session, input AddStepInput) (*Step, error)
	ListSteps(ctx context.Context, sessionID uuid.UUID, limit int) ([]Step, error)
	ListScopedMemories(ctx context.Context, scope Scope, actorID uuid.UUID, actorType string, limit int) ([]Memory, error)
	CreateMemory(ctx context.Context, input CreateMemoryInput) (*Memory, error)
}

type AddStepInput struct {
	InvocationID    *uuid.UUID
	ToolExecutionID *uuid.UUID
	ToolApprovalID  *uuid.UUID
	StepType        string
	Status          string
	Summary         string
	Data            map[string]any
	Turn            int
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateSession(ctx context.Context, actorID uuid.UUID, actorType string, input CreateSessionInput) (*Session, error) {
	session := &Session{}
	err := scanSession(r.db.QueryRow(ctx, `
		INSERT INTO assistant_sessions (
			title, mode, module_key, actor_id, actor_type, provider_id, preferred_channel_id,
			provider_type, model, service_tier, reasoning_effort, organization_id, department_id,
			position_id, position_assignment_id, project_id, workflow_id, task_id, metadata
		)
		VALUES ($1, COALESCE(NULLIF($2, ''), 'business_process'), COALESCE(NULLIF($3, ''), 'general'),
			$4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING id, title, mode, module_key, status, actor_id, actor_type, provider_id, preferred_channel_id,
			provider_type, model, service_tier, reasoning_effort, organization_id, department_id,
			position_id, position_assignment_id, project_id, workflow_id, task_id, working_memory, metadata,
			last_error, created_at, updated_at
	`, input.Title, input.Mode, input.ModuleKey, actorID, actorType, input.ProviderID, input.PreferredChannelID,
		input.ProviderType, input.Model, input.ServiceTier, input.ReasoningEffort, input.OrganizationID,
		input.DepartmentID, input.PositionID, input.PositionAssignmentID, input.ProjectID, input.WorkflowID,
		input.TaskID, mustJSON(input.Metadata)), session)
	if err != nil {
		return nil, fmt.Errorf("create assistant session: %w", err)
	}
	return session, nil
}

func (r *PostgresRepository) ListSessions(ctx context.Context, actorID uuid.UUID, actorType string, moduleKey string, limit int) ([]Session, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, title, mode, module_key, status, actor_id, actor_type, provider_id, preferred_channel_id,
			provider_type, model, service_tier, reasoning_effort, organization_id, department_id,
			position_id, position_assignment_id, project_id, workflow_id, task_id, working_memory, metadata,
			last_error, created_at, updated_at
		FROM assistant_sessions
		WHERE actor_id = $1 AND actor_type = $2 AND ($3 = '' OR module_key = $3)
		ORDER BY updated_at DESC
		LIMIT $4
	`, actorID, actorType, moduleKey, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list assistant sessions: %w", err)
	}
	defer rows.Close()
	items := []Session{}
	for rows.Next() {
		var item Session
		if err := scanSession(rows, &item); err != nil {
			return nil, fmt.Errorf("scan assistant session: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) GetSession(ctx context.Context, id uuid.UUID, actorID uuid.UUID, actorType string) (*Session, error) {
	session := &Session{}
	err := scanSession(r.db.QueryRow(ctx, `
		SELECT id, title, mode, module_key, status, actor_id, actor_type, provider_id, preferred_channel_id,
			provider_type, model, service_tier, reasoning_effort, organization_id, department_id,
			position_id, position_assignment_id, project_id, workflow_id, task_id, working_memory, metadata,
			last_error, created_at, updated_at
		FROM assistant_sessions
		WHERE id = $1 AND actor_id = $2 AND actor_type = $3
	`, id, actorID, actorType), session)
	if err != nil {
		return nil, fmt.Errorf("get assistant session: %w", err)
	}
	return session, nil
}

func (r *PostgresRepository) UpdateSessionStatus(ctx context.Context, id uuid.UUID, status string, lastError string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE assistant_sessions
		SET status = $2, last_error = $3, updated_at = NOW()
		WHERE id = $1
	`, id, status, lastError)
	return err
}

func (r *PostgresRepository) UpdateWorkingMemory(ctx context.Context, id uuid.UUID, memory map[string]any) error {
	_, err := r.db.Exec(ctx, `
		UPDATE assistant_sessions
		SET working_memory = $2, updated_at = NOW()
		WHERE id = $1
	`, id, mustJSON(memory))
	return err
}

func (r *PostgresRepository) AddMessage(ctx context.Context, sessionID uuid.UUID, role string, content string, toolCallID string, toolName string, metadata map[string]any) (*Message, error) {
	message := &Message{}
	err := scanMessage(r.db.QueryRow(ctx, `
		INSERT INTO assistant_messages (session_id, role, content, tool_call_id, tool_name, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, session_id, role, content, tool_call_id, tool_name, metadata, created_at
	`, sessionID, role, content, toolCallID, toolName, mustJSON(metadata)), message)
	if err != nil {
		return nil, fmt.Errorf("add assistant message: %w", err)
	}
	return message, nil
}

func (r *PostgresRepository) ListMessages(ctx context.Context, sessionID uuid.UUID, limit int) ([]Message, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, session_id, role, content, tool_call_id, tool_name, metadata, created_at
		FROM assistant_messages
		WHERE session_id = $1
		ORDER BY created_at ASC
		LIMIT $2
	`, sessionID, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list assistant messages: %w", err)
	}
	defer rows.Close()
	items := []Message{}
	for rows.Next() {
		var item Message
		if err := scanMessage(rows, &item); err != nil {
			return nil, fmt.Errorf("scan assistant message: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) AddStep(ctx context.Context, session *Session, input AddStepInput) (*Step, error) {
	step := &Step{}
	err := scanStep(r.db.QueryRow(ctx, `
		INSERT INTO assistant_steps (
			session_id, module_key, organization_id, department_id, position_id, position_assignment_id,
			invocation_id, tool_execution_id, tool_approval_id, step_type, status, summary, data, turn
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, COALESCE(NULLIF($11, ''), 'completed'), $12, $13, $14)
		RETURNING id, session_id, module_key, organization_id, department_id, position_id, position_assignment_id,
			invocation_id, tool_execution_id, tool_approval_id, step_type, status, summary, data, turn, created_at
	`, session.ID, session.ModuleKey, session.OrganizationID, session.DepartmentID, session.PositionID,
		session.PositionAssignmentID, input.InvocationID, input.ToolExecutionID, input.ToolApprovalID,
		input.StepType, input.Status, input.Summary, mustJSON(input.Data), input.Turn), step)
	if err != nil {
		return nil, fmt.Errorf("add assistant step: %w", err)
	}
	return step, nil
}

func (r *PostgresRepository) ListSteps(ctx context.Context, sessionID uuid.UUID, limit int) ([]Step, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, session_id, module_key, organization_id, department_id, position_id, position_assignment_id,
			invocation_id, tool_execution_id, tool_approval_id, step_type, status, summary, data, turn, created_at
		FROM assistant_steps
		WHERE session_id = $1
		ORDER BY created_at ASC
		LIMIT $2
	`, sessionID, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list assistant steps: %w", err)
	}
	defer rows.Close()
	items := []Step{}
	for rows.Next() {
		var item Step
		if err := scanStep(rows, &item); err != nil {
			return nil, fmt.Errorf("scan assistant step: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) ListScopedMemories(ctx context.Context, scope Scope, actorID uuid.UUID, actorType string, limit int) ([]Memory, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, module_key, organization_id, department_id, position_id, position_assignment_id,
			actor_id, actor_type, memory_type, title, content, data, source_session_id, source_step_id,
			confidence, created_at, updated_at
		FROM assistant_memories
		WHERE module_key = COALESCE(NULLIF($1, ''), 'general')
			AND organization_id IS NOT DISTINCT FROM $2
			AND department_id IS NOT DISTINCT FROM $3
			AND position_id IS NOT DISTINCT FROM $4
			AND position_assignment_id IS NOT DISTINCT FROM $5
			AND (actor_id IS NULL OR (actor_id = $6 AND actor_type = $7))
		ORDER BY updated_at DESC
		LIMIT $8
	`, scope.ModuleKey, scope.OrganizationID, scope.DepartmentID, scope.PositionID, scope.PositionAssignmentID,
		actorID, actorType, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list assistant memories: %w", err)
	}
	defer rows.Close()
	items := []Memory{}
	for rows.Next() {
		var item Memory
		if err := scanMemory(rows, &item); err != nil {
			return nil, fmt.Errorf("scan assistant memory: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *PostgresRepository) CreateMemory(ctx context.Context, input CreateMemoryInput) (*Memory, error) {
	memory := &Memory{}
	err := scanMemory(r.db.QueryRow(ctx, `
		INSERT INTO assistant_memories (
			module_key, organization_id, department_id, position_id, position_assignment_id,
			actor_id, actor_type, memory_type, title, content, data, source_session_id, source_step_id, confidence
		)
		VALUES (COALESCE(NULLIF($1, ''), 'general'), $2, $3, $4, $5, $6, $7,
			COALESCE(NULLIF($8, ''), 'lesson'), $9, $10, $11, $12, $13, COALESCE(NULLIF($14, 0), 1))
		RETURNING id, module_key, organization_id, department_id, position_id, position_assignment_id,
			actor_id, actor_type, memory_type, title, content, data, source_session_id, source_step_id,
			confidence, created_at, updated_at
	`, input.Scope.ModuleKey, input.Scope.OrganizationID, input.Scope.DepartmentID, input.Scope.PositionID,
		input.Scope.PositionAssignmentID, input.ActorID, input.ActorType, input.MemoryType, input.Title,
		input.Content, mustJSON(input.Data), input.SourceSessionID, input.SourceStepID, input.Confidence), memory)
	if err != nil {
		return nil, fmt.Errorf("create assistant memory: %w", err)
	}
	return memory, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanSession(row scanner, session *Session) error {
	var providerID, channelID, orgID, deptID, positionID, assignmentID, projectID, workflowID, taskID pgtype.UUID
	var workingJSON, metaJSON []byte
	if err := row.Scan(&session.ID, &session.Title, &session.Mode, &session.ModuleKey, &session.Status,
		&session.ActorID, &session.ActorType, &providerID, &channelID, &session.ProviderType, &session.Model,
		&session.ServiceTier, &session.ReasoningEffort, &orgID, &deptID, &positionID, &assignmentID,
		&projectID, &workflowID, &taskID, &workingJSON, &metaJSON, &session.LastError, &session.CreatedAt,
		&session.UpdatedAt); err != nil {
		return err
	}
	session.ProviderID = uuidPointer(providerID)
	session.PreferredChannelID = uuidPointer(channelID)
	session.OrganizationID = uuidPointer(orgID)
	session.DepartmentID = uuidPointer(deptID)
	session.PositionID = uuidPointer(positionID)
	session.PositionAssignmentID = uuidPointer(assignmentID)
	session.ProjectID = uuidPointer(projectID)
	session.WorkflowID = uuidPointer(workflowID)
	session.TaskID = uuidPointer(taskID)
	session.WorkingMemory = unmarshalMap(workingJSON)
	session.Metadata = unmarshalMap(metaJSON)
	return nil
}

func scanMessage(row scanner, message *Message) error {
	var metaJSON []byte
	if err := row.Scan(&message.ID, &message.SessionID, &message.Role, &message.Content, &message.ToolCallID,
		&message.ToolName, &metaJSON, &message.CreatedAt); err != nil {
		return err
	}
	message.Metadata = unmarshalMap(metaJSON)
	return nil
}

func scanStep(row scanner, step *Step) error {
	var orgID, deptID, positionID, assignmentID, invocationID, executionID, approvalID pgtype.UUID
	var dataJSON []byte
	if err := row.Scan(&step.ID, &step.SessionID, &step.ModuleKey, &orgID, &deptID, &positionID, &assignmentID,
		&invocationID, &executionID, &approvalID, &step.StepType, &step.Status, &step.Summary,
		&dataJSON, &step.Turn, &step.CreatedAt); err != nil {
		return err
	}
	step.OrganizationID = uuidPointer(orgID)
	step.DepartmentID = uuidPointer(deptID)
	step.PositionID = uuidPointer(positionID)
	step.PositionAssignmentID = uuidPointer(assignmentID)
	step.InvocationID = uuidPointer(invocationID)
	step.ToolExecutionID = uuidPointer(executionID)
	step.ToolApprovalID = uuidPointer(approvalID)
	step.Data = unmarshalMap(dataJSON)
	return nil
}

func scanMemory(row scanner, memory *Memory) error {
	var orgID, deptID, positionID, assignmentID, actorID, sessionID, stepID pgtype.UUID
	var dataJSON []byte
	if err := row.Scan(&memory.ID, &memory.Scope.ModuleKey, &orgID, &deptID, &positionID, &assignmentID,
		&actorID, &memory.ActorType, &memory.MemoryType, &memory.Title, &memory.Content, &dataJSON,
		&sessionID, &stepID, &memory.Confidence, &memory.CreatedAt, &memory.UpdatedAt); err != nil {
		return err
	}
	memory.Scope.OrganizationID = uuidPointer(orgID)
	memory.Scope.DepartmentID = uuidPointer(deptID)
	memory.Scope.PositionID = uuidPointer(positionID)
	memory.Scope.PositionAssignmentID = uuidPointer(assignmentID)
	memory.ActorID = uuidPointer(actorID)
	memory.SourceSessionID = uuidPointer(sessionID)
	memory.SourceStepID = uuidPointer(stepID)
	memory.Data = unmarshalMap(dataJSON)
	return nil
}

func mustJSON(input map[string]any) []byte {
	if input == nil {
		input = map[string]any{}
	}
	data, err := json.Marshal(input)
	if err != nil {
		return []byte("{}")
	}
	return data
}

func unmarshalMap(data []byte) map[string]any {
	value := map[string]any{}
	if len(data) == 0 {
		return value
	}
	_ = json.Unmarshal(data, &value)
	if value == nil {
		return map[string]any{}
	}
	return value
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func uuidPointer(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	id, err := uuid.FromBytes(value.Bytes[:])
	if err != nil {
		return nil
	}
	return &id
}
