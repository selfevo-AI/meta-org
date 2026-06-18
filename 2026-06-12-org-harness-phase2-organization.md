# Phase 2: Organization + Layer Domain

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan.

**Goal:** Build Organization Domain (MVRU management, org chart, capability binding) and Layer Domain (strategic/tactical/operational layer matching).

**Architecture:** Go backend modules under `internal/domain/organization/` and `internal/domain/layer/`, new PostgreSQL migrations, new API routes registered via gateway router.

**Tech Stack:** Go 1.22, PostgreSQL 16, Chi router

---

### Task 1: Organization schema + migration

**Files:**
- Create: `migrations/003_organization.sql`

**Content:**
```sql
-- 003_organization.sql

CREATE TABLE organizations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TYPE mvru_status AS ENUM ('designing', 'active', 'evaluating', 'evolving', 'dissolved');

CREATE TABLE muvrs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    status          mvru_status NOT NULL DEFAULT 'designing',
    boundary        JSONB NOT NULL DEFAULT '{"data_permissions":[],"resource_quota":{},"network_policies":[]}',
    config          JSONB NOT NULL DEFAULT '{}',
    parent_id       UUID REFERENCES muvrs(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE teams (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mvru_id     UUID NOT NULL REFERENCES muvrs(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE mvru_members (
    mvru_id     UUID NOT NULL REFERENCES muvrs(id) ON DELETE CASCADE,
    user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
    agent_id    UUID REFERENCES ai_agents(id) ON DELETE CASCADE,
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    CONSTRAINT chk_one_actor CHECK (
        (user_id IS NOT NULL AND agent_id IS NULL) OR
        (user_id IS NULL AND agent_id IS NOT NULL)
    ),
    PRIMARY KEY (mvru_id, COALESCE(user_id, agent_id))
);

CREATE TABLE mvru_relationships (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_mvru_id  UUID NOT NULL REFERENCES muvrs(id) ON DELETE CASCADE,
    target_mvru_id  UUID NOT NULL REFERENCES muvrs(id) ON DELETE CASCADE,
    rel_type        VARCHAR(50) NOT NULL DEFAULT 'collaborate',
    config          JSONB DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_no_self_ref CHECK (source_mvru_id != target_mvru_id)
);

CREATE INDEX idx_muvrs_org ON muvrs(organization_id);
CREATE INDEX idx_teams_mvru ON teams(mvru_id);
```

**Commit:** `git add -A && git commit -m "feat: organization schema with MVRU, teams, members"`

---

### Task 2: Organization Domain — models

**Files:**
- Create: `backend/internal/domain/organization/model.go`

**Content:**
```go
package organization

import (
	"time"

	"github.com/google/uuid"
)

type MVRUStatus string

const (
	MVRUDesigning  MVRUStatus = "designing"
	MVRUActive     MVRUStatus = "active"
	MVRUEvaluating MVRUStatus = "evaluating"
	MVRUEvolving   MVRUStatus = "evolving"
	MVRUDissolved  MVRUStatus = "dissolved"
)

type Organization struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type MVRU struct {
	ID             uuid.UUID              `json:"id"`
	OrganizationID uuid.UUID              `json:"organization_id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	Status         MVRUStatus             `json:"status"`
	Boundary       map[string]any         `json:"boundary"`
	Config         map[string]any         `json:"config"`
	ParentID       *uuid.UUID             `json:"parent_id,omitempty"`
	Children       []MVRU                 `json:"children,omitempty"`
	Members        []MVRUMember           `json:"members,omitempty"`
	Relationships  []MVRURelationship     `json:"relationships,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

type Team struct {
	ID          uuid.UUID `json:"id"`
	MVRUID      uuid.UUID `json:"mvru_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type MVRUMember struct {
	MVRUID  uuid.UUID  `json:"mvru_id"`
	UserID  *uuid.UUID `json:"user_id,omitempty"`
	AgentID *uuid.UUID `json:"agent_id,omitempty"`
	RoleID  uuid.UUID  `json:"role_id"`
}

type MVRURelationship struct {
	ID            uuid.UUID `json:"id"`
	SourceMVRUID  uuid.UUID `json:"source_mvru_id"`
	TargetMVRUID  uuid.UUID `json:"target_mvru_id"`
	RelType       string    `json:"rel_type"`
	Config        map[string]any `json:"config,omitempty"`
}

type CreateOrganizationInput struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type CreateMVRUInput struct {
	OrganizationID uuid.UUID              `json:"organization_id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description,omitempty"`
	Boundary       map[string]any         `json:"boundary,omitempty"`
	Config         map[string]any         `json:"config,omitempty"`
	ParentID       *uuid.UUID             `json:"parent_id,omitempty"`
}
```

**Commit:** `git add -A && git commit -m "feat: organization domain models"`

---

### Task 3: Organization Domain — repository

**Files:**
- Create: `backend/internal/domain/organization/repository.go`

**Content:** Repository with:
- `CreateOrganization(ctx, input) (*Organization, error)`
- `GetOrganizationByID(ctx, id) (*Organization, error)`
- `CreateMVRU(ctx, input) (*MVRU, error)`
- `GetMVRUByID(ctx, id) (*MVRU, error)`
- `ListMVRUs(ctx, orgID) ([]MVRU, error)`
- `UpdateMVRUStatus(ctx, id, status) error`
- `AddMember(ctx, member) error`
- `RemoveMember(ctx, mvruID, userID, agentID) error`
- `CreateRelationship(ctx, rel) (*MVRURelationship, error)`
- `GetOrgChart(ctx, orgID) ([]MVRU, error)` — fetches full tree

Include: proper JSON scanning for boundary/config, join queries for members, error wrapping, `rows.Err()` check.

**Commit:** `git add -A && git commit -m "feat: organization repository with MVRU CRUD and org chart"`

---

### Task 4: Organization Domain — service + handler + routes

**Files:**
- Create: `backend/internal/domain/organization/service.go`
- Create: `backend/internal/domain/organization/handler.go`
- Modify: `backend/internal/gateway/router.go`
- Modify: `backend/cmd/server/main.go`

**service.go:**
- `CreateOrganization(ctx, input) (*Organization, error)` — validation + repo
- `GetOrganizationTree(ctx, orgID) (*MVRU, error)` — fetch org chart from repo
- `CreateMVRU(ctx, input) (*MVRU, error)` — validation + repo
- `ActivateMVRU(ctx, mvruID) error` — status transition
- `EvaluateMVRU(ctx, mvruID) error`
- `AddMember(ctx, mvruID, userOrAgentID, roleID) error`

**handler.go:**
- `POST /organizations` — create org
- `GET /organizations/{id}` — get org + chart
- `POST /muvrs` — create MVRU
- `GET /muvrs/{id}` — get MVRU detail
- `PATCH /muvrs/{id}/status` — update status
- `POST /muvrs/{id}/members` — add member
- `DELETE /muvrs/{id}/members/{userId}` — remove member
- `POST /relationships` — create relationship

Use `writeJSON` pattern from identity handler.

**router.go:** Add `OrganizationHandler` to `Dependencies` struct. Register routes under `/api/v1`.

**main.go:** Wire `organization.NewRepository(db)` → `organization.NewService(repo)` → `organization.NewHandler(svc)` → deps.

**Commit:** `git add -A && git commit -m "feat: organization API routes with CRUD and org chart"`

---

### Task 5: Layer Domain — schema + models

**Files:**
- Create: `migrations/004_layer.sql`
- Create: `backend/internal/domain/layer/model.go`

**Migration:**
```sql
-- 004_layer.sql

CREATE TYPE layer_type AS ENUM ('strategic', 'tactical', 'operational');

CREATE TABLE layer_configs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mvru_id         UUID NOT NULL REFERENCES muvrs(id) ON DELETE CASCADE,
    layer           layer_type NOT NULL,
    config          JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (mvru_id, layer)
);

CREATE TABLE layer_routing_rules (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_layer    layer_type NOT NULL,
    target_layer    layer_type NOT NULL,
    condition       JSONB NOT NULL DEFAULT '{}',
    priority        INT NOT NULL DEFAULT 0,
    is_active       BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_layer_config_mvru ON layer_configs(mvru_id);
```

**model.go:**
```go
package layer

import (
	"time"

	"github.com/google/uuid"
)

type LayerType string

const (
	LayerStrategic   LayerType = "strategic"
	LayerTactical    LayerType = "tactical"
	LayerOperational LayerType = "operational"
)

type LayerConfig struct {
	ID        uuid.UUID              `json:"id"`
	MVRUID    uuid.UUID              `json:"mvru_id"`
	Layer     LayerType              `json:"layer"`
	Config    map[string]any         `json:"config"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type LayerRoutingRule struct {
	ID          uuid.UUID              `json:"id"`
	SourceLayer LayerType              `json:"source_layer"`
	TargetLayer LayerType              `json:"target_layer"`
	Condition   map[string]any         `json:"condition"`
	Priority    int                    `json:"priority"`
	IsActive    bool                   `json:"is_active"`
}
```

**Commit:** `git add -A && git commit -m "feat: layer domain schema and models"`

---

### Task 6: Layer Domain — repository + classifier service

**Files:**
- Create: `backend/internal/domain/layer/repository.go`
- Create: `backend/internal/domain/layer/service.go`
- Create: `backend/internal/domain/layer/handler.go`
- Modify: `backend/internal/gateway/router.go`
- Modify: `backend/cmd/server/main.go`

**repository.go:** CRUD for layer_configs and routing_rules.

**service.go:**
- `ClassifyTask(ctx, task) (LayerType, error)` — computes complexity/risk/strategic scores and maps to layer
- `SetLayerConfig(ctx, mvruID, layer, config) error`
- `GetLayerConfig(ctx, mvruID) (*LayerConfig, error)`
- `GetLayerRoutingRules(ctx) ([]LayerRoutingRule, error)`

**handler.go:**
- `POST /layers/classify` — classify a task/operation
- `GET /layers/config/{mvruId}` — get layer config
- `PUT /layers/config/{mvruId}` — set layer config
- `GET /layers/rules` — list routing rules

**Wiring:** Add to Dependencies, register routes, wire in main.go.

**Commit:** `git add -A && git commit -m "feat: layer classifier service and API routes"`
