package workflow

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateTemplate(ctx context.Context, input CreateWorkflowInput) (*WorkflowTemplate, error) {
	stagesJSON, _ := json.Marshal(input.Stages)
	rulesJSON, _ := json.Marshal(input.RoutingRules)
	graphJSON, _ := json.Marshal(input.VisualGraph)

	t := &WorkflowTemplate{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO workflow_templates (
		    organization_id, department_id, name, description, stages, assignee_type, required_weight, routing_rules, visual_graph
		 )
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, organization_id, department_id, name, description, stages, assignee_type, required_weight,
		           routing_rules, visual_graph, is_active, created_at, updated_at`,
		input.OrganizationID, input.DepartmentID, input.Name, input.Description, stagesJSON, input.AssigneeType, input.RequiredWeight, rulesJSON, graphJSON,
	).Scan(&t.ID, &t.OrganizationID, &t.DepartmentID, &t.Name, &t.Description, &stagesJSON, &t.AssigneeType, &t.RequiredWeight, &rulesJSON, &graphJSON, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create template: %w", err)
	}
	json.Unmarshal(stagesJSON, &t.Stages)
	json.Unmarshal(rulesJSON, &t.RoutingRules)
	json.Unmarshal(graphJSON, &t.VisualGraph)
	return t, nil
}

func (r *Repository) GetTemplate(ctx context.Context, id uuid.UUID) (*WorkflowTemplate, error) {
	t := &WorkflowTemplate{}
	var stagesJSON, rulesJSON, graphJSON []byte
	err := r.db.QueryRow(ctx,
		`SELECT id, organization_id, department_id, name, description, stages, assignee_type, required_weight,
		        routing_rules, visual_graph, is_active, created_at, updated_at
		 FROM workflow_templates WHERE id = $1`, id,
	).Scan(&t.ID, &t.OrganizationID, &t.DepartmentID, &t.Name, &t.Description, &stagesJSON, &t.AssigneeType, &t.RequiredWeight, &rulesJSON, &graphJSON, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}
	json.Unmarshal(stagesJSON, &t.Stages)
	json.Unmarshal(rulesJSON, &t.RoutingRules)
	json.Unmarshal(graphJSON, &t.VisualGraph)
	return t, nil
}

func (r *Repository) ListTemplates(ctx context.Context) ([]WorkflowTemplate, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, organization_id, department_id, name, description, stages, assignee_type, required_weight,
		        routing_rules, visual_graph, is_active, created_at, updated_at
		 FROM workflow_templates ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list templates: %w", err)
	}
	defer rows.Close()

	var templates []WorkflowTemplate
	for rows.Next() {
		var t WorkflowTemplate
		var stagesJSON, rulesJSON, graphJSON []byte
		if err := rows.Scan(&t.ID, &t.OrganizationID, &t.DepartmentID, &t.Name, &t.Description, &stagesJSON, &t.AssigneeType, &t.RequiredWeight, &rulesJSON, &graphJSON, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan template: %w", err)
		}
		json.Unmarshal(stagesJSON, &t.Stages)
		json.Unmarshal(rulesJSON, &t.RoutingRules)
		json.Unmarshal(graphJSON, &t.VisualGraph)
		templates = append(templates, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list templates iteration: %w", err)
	}
	return templates, nil
}

func (r *Repository) CreateInstance(ctx context.Context, input StartWorkflowInput) (*WorkflowInstance, error) {
	contextJSON, _ := json.Marshal(input.Context)
	inst := &WorkflowInstance{}
	err := r.db.QueryRow(ctx,
		`INSERT INTO workflow_instances (template_id, organization_id, department_id, project_id, context)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, template_id, organization_id, department_id, project_id, status, current_stage, context, trace_id, created_at, updated_at`,
		input.TemplateID, input.OrganizationID, input.DepartmentID, input.ProjectID, contextJSON,
	).Scan(&inst.ID, &inst.TemplateID, &inst.OrganizationID, &inst.DepartmentID, &inst.ProjectID, &inst.Status, &inst.CurrentStage, &contextJSON, &inst.TraceID, &inst.CreatedAt, &inst.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create instance: %w", err)
	}
	json.Unmarshal(contextJSON, &inst.Context)
	return inst, nil
}

func (r *Repository) CreateInstanceWithTasks(ctx context.Context, input StartWorkflowInput, tmpl *WorkflowTemplate) (*WorkflowInstance, error) {
	contextJSON, err := json.Marshal(input.Context)
	if err != nil {
		return nil, fmt.Errorf("marshal workflow context: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin workflow transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	inst := &WorkflowInstance{}
	err = tx.QueryRow(ctx,
		`INSERT INTO workflow_instances (template_id, organization_id, department_id, project_id, context)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, template_id, organization_id, department_id, project_id, status, current_stage, context, trace_id, created_at, updated_at`,
		input.TemplateID, input.OrganizationID, input.DepartmentID, input.ProjectID, contextJSON,
	).Scan(&inst.ID, &inst.TemplateID, &inst.OrganizationID, &inst.DepartmentID, &inst.ProjectID, &inst.Status, &inst.CurrentStage, &contextJSON, &inst.TraceID, &inst.CreatedAt, &inst.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create instance: %w", err)
	}
	if err := json.Unmarshal(contextJSON, &inst.Context); err != nil {
		return nil, fmt.Errorf("unmarshal workflow context: %w", err)
	}

	for i, stage := range tmpl.Stages {
		task := Task{
			WorkflowID:     inst.ID,
			Stage:          i,
			StageType:      stage.Type,
			AssigneeType:   stage.AssigneeType,
			Input:          taskInputFromStage(input.Context, stage),
			WeightSnapshot: tmpl.RequiredWeight,
			Status:         TaskPending,
		}
		if i == 0 {
			task.Status = TaskAssigned
		}

		taskInputJSON, err := json.Marshal(task.Input)
		if err != nil {
			return nil, fmt.Errorf("marshal task input for stage %d: %w", i, err)
		}
		var outputJSON []byte
		err = tx.QueryRow(ctx,
			`INSERT INTO tasks (workflow_id, stage, stage_type, assignee_id, assignee_type, input, weight_snapshot, status)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 RETURNING id, workflow_id, stage, stage_type, assignee_id, assignee_type, input, output, weight_snapshot, status, created_at, updated_at`,
			task.WorkflowID, task.Stage, task.StageType, task.AssigneeID, task.AssigneeType, taskInputJSON, task.WeightSnapshot, task.Status,
		).Scan(&task.ID, &task.WorkflowID, &task.Stage, &task.StageType, &task.AssigneeID, &task.AssigneeType, &taskInputJSON, &outputJSON, &task.WeightSnapshot, &task.Status, &task.CreatedAt, &task.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("create task for stage %d: %w", i, err)
		}
		if err := json.Unmarshal(taskInputJSON, &task.Input); err != nil {
			return nil, fmt.Errorf("unmarshal task input for stage %d: %w", i, err)
		}
		if outputJSON != nil {
			if err := json.Unmarshal(outputJSON, &task.Output); err != nil {
				return nil, fmt.Errorf("unmarshal task output for stage %d: %w", i, err)
			}
		}
		inst.Tasks = append(inst.Tasks, task)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit workflow transaction: %w", err)
	}
	return inst, nil
}

func taskInputFromStage(base map[string]any, stage Stage) map[string]any {
	taskInput := map[string]any{}
	for key, value := range base {
		taskInput[key] = value
	}
	taskInput["stage_id"] = stage.ID
	taskInput["stage_name"] = stage.Name
	if stage.PositionID != nil {
		taskInput["position_id"] = stage.PositionID.String()
	}
	taskInput["position_code"] = stage.PositionCode
	taskInput["required_roles"] = stage.RequiredRoles
	taskInput["required_tools"] = stage.RequiredTools
	taskInput["required_capabilities"] = stage.RequiredCapabilities
	taskInput["required_permission_level"] = stage.RequiredPermissionLevel
	taskInput["risk_level"] = stage.RiskLevel
	taskInput["preferred_actor_types"] = stage.PreferredActorTypes
	if stage.EvaluationPolicy != nil {
		taskInput["evaluation_policy"] = stage.EvaluationPolicy
	}
	if stage.MatchingPolicy != nil {
		taskInput["matching_policy"] = stage.MatchingPolicy
	}
	return taskInput
}

func (r *Repository) GetInstance(ctx context.Context, id uuid.UUID) (*WorkflowInstance, error) {
	inst := &WorkflowInstance{}
	var contextJSON []byte
	err := r.db.QueryRow(ctx,
		`SELECT id, template_id, organization_id, department_id, project_id, status, current_stage, context, trace_id, created_at, updated_at
		 FROM workflow_instances WHERE id = $1`, id,
	).Scan(&inst.ID, &inst.TemplateID, &inst.OrganizationID, &inst.DepartmentID, &inst.ProjectID, &inst.Status, &inst.CurrentStage, &contextJSON, &inst.TraceID, &inst.CreatedAt, &inst.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get instance: %w", err)
	}
	json.Unmarshal(contextJSON, &inst.Context)
	return inst, nil
}

func (r *Repository) UpdateInstanceStatus(ctx context.Context, id uuid.UUID, status WorkflowStatus) error {
	_, err := r.db.Exec(ctx, `UPDATE workflow_instances SET status = $1, updated_at = NOW() WHERE id = $2`, status, id)
	if err != nil {
		return fmt.Errorf("update instance status: %w", err)
	}
	return nil
}

func (r *Repository) UpdateInstanceStage(ctx context.Context, id uuid.UUID, stage int) error {
	_, err := r.db.Exec(ctx, `UPDATE workflow_instances SET current_stage = $1, updated_at = NOW() WHERE id = $2`, stage, id)
	if err != nil {
		return fmt.Errorf("update instance stage: %w", err)
	}
	return nil
}

func (r *Repository) CreateTask(ctx context.Context, task *Task) (*Task, error) {
	inputJSON, _ := json.Marshal(task.Input)
	err := r.db.QueryRow(ctx,
		`INSERT INTO tasks (workflow_id, stage, stage_type, assignee_id, assignee_type, input, weight_snapshot, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, workflow_id, stage, stage_type, assignee_id, assignee_type, input, output, weight_snapshot, status, created_at, updated_at`,
		task.WorkflowID, task.Stage, task.StageType, task.AssigneeID, task.AssigneeType, inputJSON, task.WeightSnapshot, task.Status,
	).Scan(&task.ID, &task.WorkflowID, &task.Stage, &task.StageType, &task.AssigneeID, &task.AssigneeType, &inputJSON, &task.Output, &task.WeightSnapshot, &task.Status, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	json.Unmarshal(inputJSON, &task.Input)
	return task, nil
}

func (r *Repository) UpdateTaskStatus(ctx context.Context, id uuid.UUID, status TaskStatus, output map[string]any) error {
	outputJSON, _ := json.Marshal(output)
	_, err := r.db.Exec(ctx,
		`UPDATE tasks SET status = $1, output = $2, updated_at = NOW() WHERE id = $3`, status, outputJSON, id)
	if err != nil {
		return fmt.Errorf("update task status: %w", err)
	}
	return nil
}

func (r *Repository) CompleteTaskWithWorkflowProgress(ctx context.Context, taskID uuid.UUID, output map[string]any) error {
	outputJSON, err := json.Marshal(output)
	if err != nil {
		return fmt.Errorf("marshal task output: %w", err)
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin task transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	task := &Task{}
	var inputJSON, existingOutputJSON []byte
	err = tx.QueryRow(ctx,
		`SELECT id, workflow_id, stage, stage_type, assignee_id, assignee_type, input, output, weight_snapshot, status, created_at, updated_at
		 FROM tasks WHERE id = $1 FOR UPDATE`, taskID,
	).Scan(&task.ID, &task.WorkflowID, &task.Stage, &task.StageType, &task.AssigneeID, &task.AssigneeType, &inputJSON, &existingOutputJSON, &task.WeightSnapshot, &task.Status, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return fmt.Errorf("get task: %w", err)
	}
	if err := json.Unmarshal(inputJSON, &task.Input); err != nil {
		return fmt.Errorf("unmarshal task input: %w", err)
	}
	if existingOutputJSON != nil {
		if err := json.Unmarshal(existingOutputJSON, &task.Output); err != nil {
			return fmt.Errorf("unmarshal existing task output: %w", err)
		}
	}

	if _, err := tx.Exec(ctx,
		`UPDATE tasks SET status = $1, output = $2, updated_at = NOW() WHERE id = $3`,
		TaskCompleted, outputJSON, taskID,
	); err != nil {
		return fmt.Errorf("update task status: %w", err)
	}

	inst := &WorkflowInstance{}
	var contextJSON []byte
	err = tx.QueryRow(ctx,
		`SELECT id, template_id, organization_id, department_id, project_id, status, current_stage, context, trace_id, created_at, updated_at
		 FROM workflow_instances WHERE id = $1 FOR UPDATE`, task.WorkflowID,
	).Scan(&inst.ID, &inst.TemplateID, &inst.OrganizationID, &inst.DepartmentID, &inst.ProjectID, &inst.Status, &inst.CurrentStage, &contextJSON, &inst.TraceID, &inst.CreatedAt, &inst.UpdatedAt)
	if err != nil {
		return fmt.Errorf("get workflow instance: %w", err)
	}
	if err := json.Unmarshal(contextJSON, &inst.Context); err != nil {
		return fmt.Errorf("unmarshal workflow context: %w", err)
	}

	var stagesJSON []byte
	var stages []Stage
	if err := tx.QueryRow(ctx, `SELECT stages FROM workflow_templates WHERE id = $1`, inst.TemplateID).Scan(&stagesJSON); err != nil {
		return fmt.Errorf("get workflow template stages: %w", err)
	}
	if err := json.Unmarshal(stagesJSON, &stages); err != nil {
		return fmt.Errorf("unmarshal workflow template stages: %w", err)
	}

	nextStage := inst.CurrentStage + 1
	nextStatus := WorkflowActive
	if nextStage >= len(stages) {
		nextStatus = WorkflowCompleted
	}
	if _, err := tx.Exec(ctx,
		`UPDATE workflow_instances SET current_stage = $1, status = $2, updated_at = NOW() WHERE id = $3`,
		nextStage, nextStatus, inst.ID,
	); err != nil {
		return fmt.Errorf("update workflow progress: %w", err)
	}

	if nextStatus != WorkflowCompleted {
		if _, err := tx.Exec(ctx,
			`UPDATE tasks SET status = $1, updated_at = NOW() WHERE workflow_id = $2 AND stage = $3 AND status = $4`,
			TaskAssigned, inst.ID, nextStage, TaskPending,
		); err != nil {
			return fmt.Errorf("assign next task: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit task transaction: %w", err)
	}
	return nil
}

func (r *Repository) GetTasksByWorkflow(ctx context.Context, workflowID uuid.UUID) ([]Task, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, workflow_id, stage, stage_type, assignee_id, assignee_type, input, output, weight_snapshot, status, created_at, updated_at
		 FROM tasks WHERE workflow_id = $1 ORDER BY stage, created_at`, workflowID)
	if err != nil {
		return nil, fmt.Errorf("get tasks: %w", err)
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var task Task
		var inputJSON, outputJSON []byte
		if err := rows.Scan(&task.ID, &task.WorkflowID, &task.Stage, &task.StageType, &task.AssigneeID, &task.AssigneeType, &inputJSON, &outputJSON, &task.WeightSnapshot, &task.Status, &task.CreatedAt, &task.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		json.Unmarshal(inputJSON, &task.Input)
		if outputJSON != nil {
			json.Unmarshal(outputJSON, &task.Output)
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("get tasks iteration: %w", err)
	}
	return tasks, nil
}

func (r *Repository) RecordDecision(ctx context.Context, d *Decision) (*Decision, error) {
	inputJSON, _ := json.Marshal(d.Input)
	outputJSON, _ := json.Marshal(d.Output)
	err := r.db.QueryRow(ctx,
		`INSERT INTO decisions (task_id, decision_maker_id, maker_type, weight, input, output, reasoning, outcome)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id, task_id, decision_maker_id, maker_type, weight, input, output, reasoning, outcome, created_at`,
		d.TaskID, d.DecisionMakerID, d.MakerType, d.Weight, inputJSON, outputJSON, d.Reasoning, d.Outcome,
	).Scan(&d.ID, &d.TaskID, &d.DecisionMakerID, &d.MakerType, &d.Weight, &inputJSON, &outputJSON, &d.Reasoning, &d.Outcome, &d.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("record decision: %w", err)
	}
	json.Unmarshal(inputJSON, &d.Input)
	json.Unmarshal(outputJSON, &d.Output)
	return d, nil
}

func (r *Repository) GetTaskByID(ctx context.Context, id uuid.UUID) (*Task, error) {
	task := &Task{}
	var inputJSON, outputJSON []byte
	err := r.db.QueryRow(ctx,
		`SELECT id, workflow_id, stage, stage_type, assignee_id, assignee_type, input, output, weight_snapshot, status, created_at, updated_at
		 FROM tasks WHERE id = $1`, id,
	).Scan(&task.ID, &task.WorkflowID, &task.Stage, &task.StageType, &task.AssigneeID, &task.AssigneeType, &inputJSON, &outputJSON, &task.WeightSnapshot, &task.Status, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get task by id: %w", err)
	}
	json.Unmarshal(inputJSON, &task.Input)
	if outputJSON != nil {
		json.Unmarshal(outputJSON, &task.Output)
	}
	return task, nil
}

func (r *Repository) GetWorkflowContext(ctx context.Context, workflowID uuid.UUID) (*WorkflowContext, error) {
	wc := &WorkflowContext{}
	var memJSON, expJSON []byte
	err := r.db.QueryRow(ctx,
		`SELECT id, workflow_id, working_memory, injected_experience, principle_notes, created_at, updated_at
		 FROM workflow_contexts WHERE workflow_id = $1`, workflowID,
	).Scan(&wc.ID, &wc.WorkflowID, &memJSON, &expJSON, &wc.PrincipleNotes, &wc.CreatedAt, &wc.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get workflow context: %w", err)
	}
	json.Unmarshal(memJSON, &wc.WorkingMemory)
	json.Unmarshal(expJSON, &wc.InjectedExperience)
	return wc, nil
}

func (r *Repository) UpsertWorkflowContext(ctx context.Context, wc *WorkflowContext) error {
	memJSON, _ := json.Marshal(wc.WorkingMemory)
	expJSON, _ := json.Marshal(wc.InjectedExperience)
	_, err := r.db.Exec(ctx,
		`INSERT INTO workflow_contexts (workflow_id, working_memory, injected_experience, principle_notes)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (workflow_id) DO UPDATE SET working_memory = $2, injected_experience = $3, principle_notes = $4, updated_at = NOW()`,
		wc.WorkflowID, memJSON, expJSON, wc.PrincipleNotes)
	if err != nil {
		return fmt.Errorf("upsert workflow context: %w", err)
	}
	return nil
}
