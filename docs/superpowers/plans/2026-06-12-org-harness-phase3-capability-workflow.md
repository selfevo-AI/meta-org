# Phase 3: Capability Domain + Workflow Domain

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan.

**Goal:** Build Capability Domain (capability registry, smart routing) and Workflow Domain (P-E-R flow, context management, human-AI collaboration).

**Architecture:** Go backend modules under `internal/domain/capability/` and `internal/domain/workflow/`, new migrations, new API routes.

**Tech Stack:** Go 1.22, PostgreSQL 16, Chi router

---

### Task 1: Capability schema + migration

**Create:** `migrations/005_capability.sql`

```sql
-- 005_capability.sql

CREATE TABLE capabilities (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    version         VARCHAR(50) NOT NULL DEFAULT '1.0',
    description     TEXT,
    input_schema    JSONB NOT NULL DEFAULT '{}',
    output_schema   JSONB NOT NULL DEFAULT '{}',
    preconditions   JSONB NOT NULL DEFAULT '[]',
    error_handling  JSONB NOT NULL DEFAULT '{}',
    permission_level permission_level NOT NULL DEFAULT 'L2',
    cost_estimate   JSONB NOT NULL DEFAULT '{}',
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (name, version)
);

CREATE TABLE capability_bindings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    capability_id   UUID NOT NULL REFERENCES capabilities(id) ON DELETE CASCADE,
    mvru_id         UUID NOT NULL REFERENCES muvrs(id) ON DELETE CASCADE,
    config          JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (capability_id, mvru_id)
);

CREATE TABLE capability_invocations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    capability_id   UUID NOT NULL REFERENCES capabilities(id) ON DELETE CASCADE,
    caller_id       UUID NOT NULL,
    caller_type     VARCHAR(10) NOT NULL,
    input           JSONB,
    output          JSONB,
    duration_ms     INT,
    cost            NUMERIC(12,4),
    outcome         VARCHAR(20),
    trace_id        UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cap_name ON capabilities(name);
CREATE INDEX idx_cap_bind_mvru ON capability_bindings(mvru_id);
CREATE INDEX idx_cap_inv_caller ON capability_invocations(caller_id);
```

---

### Task 2: Capability Domain — models + repository

**Create:** `backend/internal/domain/capability/model.go`
**Create:** `backend/internal/domain/capability/repository.go`

Models: Capability, CapabilityBinding, CapabilityInvocation, CreateCapabilityInput, BindCapabilityInput.

Repository: CreateCapability, GetCapabilityByID, ListCapabilities, BindCapability, UnbindCapability, ListBoundCapabilities, RecordInvocation, GetInvocationHistory.

---

### Task 3: Capability Domain — service + handler + routes

**Create:** `backend/internal/domain/capability/service.go` — Router service with `MatchTask(task) → []RankedCapability` using embedding similarity and history.

**Create:** `backend/internal/domain/capability/handler.go` — CRUD + match endpoints.

**Modify:** `backend/internal/gateway/router.go`, `backend/cmd/server/main.go`

API:
- `POST /capabilities` — register capability
- `GET /capabilities` — list all
- `GET /capabilities/{id}` — get detail
- `POST /capabilities/match` — match task to capabilities (router)
- `POST /bindings` — bind capability to MVRU
- `DELETE /bindings/{id}` — unbind
- `GET /bindings?mvru_id=X` — list bindings

---

### Task 4: Workflow schema + migration

**Create:** `migrations/006_workflow.sql`

Tables:
- `workflow_templates` (id, name, stages JSONB, assignee_type, required_weight, routing_rules JSONB)
- `workflow_instances` (id, template_id, status, current_stage, context JSONB, trace_id)
- `tasks` (id, workflow_id, stage, assignee_id, assignee_type, input JSONB, output JSONB, weight_snapshot NUMERIC, status)
- `decisions` (id, task_id, decision_maker_id, maker_type, weight NUMERIC, input JSONB, output JSONB, reasoning TEXT, outcome VARCHAR(50))
- `workflow_contexts` (id, workflow_id, working_memory JSONB, injected_experience JSONB, principle_notes TEXT)

Enums: `workflow_status` (active, paused, completed, failed), `task_status` (pending, assigned, in_progress, completed, rejected)

---

### Task 5: Workflow Domain — models

**Create:** `backend/internal/domain/workflow/model.go`

Models: WorkflowTemplate, WorkflowInstance, Task, Decision, WorkflowContext, PERTaskList (Planner-Executor-Reviewer task breakdown), CreateWorkflowInput, Stage enum types.

---

### Task 6: Workflow Domain — repository

**Create:** `backend/internal/domain/workflow/repository.go`

CRUD for workflows, tasks, decisions. Methods: CreateTemplate, GetTemplate, CreateInstance, GetInstance, UpdateStatus, CreateTask, UpdateTask, GetPendingTasks, RecordDecision, GetWorkflowContext, UpdateContext.

---

### Task 7: Workflow Domain — service + handler + routes

**Create:** `backend/internal/domain/workflow/service.go` — P-E-R orchestration, dynamic flow routing based on risk, context injection.

**Create:** `backend/internal/domain/workflow/handler.go`

**Modify:** router.go, main.go

API:
- `POST /workflows/templates` — create template
- `GET /workflows/templates` — list templates
- `POST /workflows/instances` — start workflow
- `GET /workflows/instances/{id}` — get instance + tasks
- `PATCH /tasks/{id}/status` — update task status
- `POST /tasks/{id}/decisions` — record decision
- `GET /workflows/instances/{id}/context` — get context
