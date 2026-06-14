# Meta-Org - AI-Native Organization Operating Platform

English | [简体中文](README.md)

Meta-Org is an AI-native organization operating platform for hybrid human and AI-agent teams. It brings human employees, AI agents, external collaborators, organization structure, project delivery, governance rules, and continuous learning into one operating system. The current product flow covers requirement intake, project formation, workflow execution, deliverable acceptance, cost tracking, and feedback capture.

The project is built around the **ETCLOVG** framework: Execution, Tooling, Context, Lifecycle, Observability, Verification, and Governance. This repository currently includes a Go backend, a Next.js frontend, PostgreSQL migrations, Docker Compose orchestration, JWT authentication, Meta-Org Home, organization and project workspaces, Developer Tools, AI Gateway, tool runtime, cost accounting, and generic finance exports.

## Product Goal

Meta-Org is not a simple task tracker. It focuses on how an organization can keep operating reliably after AI agents become active participants:

- **Humans and AI agents in one management model**: users, agents, and external members share identity, role, position, permission, and project assignment concepts.
- **Executable organization structure**: departments, positions, position assignments, MVRUs, workflow templates, and project members are used for scheduling, authorization, and evaluation.
- **Requirement-to-feedback lifecycle**: requirements can include uploaded documents, enter analysis workflows, be approved, convert into projects, bind members and workflows, track deliverables, record costs, and close feedback.
- **Governance embedded in workflows**: permissions, principles, control rules, risk levels, required permission levels, and decision weights participate in key project actions.
- **Self-evolution through records**: weight calculations, outcomes, experiments, knowledge entries, and signals preserve operational learning for future decisions.

## Core Concepts

| Concept | Description |
|---|---|
| ETCLOVG | Execution, Tooling, Context, Lifecycle, Observability, Verification, and Governance as seven organizational capabilities. |
| AI agents as first-class actors | Agents have identity, permission level, capabilities, origin, provider, risk level, and metadata, and can participate in projects and workflows. |
| MVRU | Minimal Viable Reconfigurable Unit, a small reconfigurable organization unit used to model adjustable structure, members, and relationships. The current API path uses `/muvrs`. |
| P-E-R workflow | Workflow templates and instances composed of Planner, Executor, and Reviewer stages, with tasks, decisions, and context. |
| Decision weight | A score for a human or agent based on capability, history, risk, and organization context. |
| Governance access decision | Access decisions are based on permissions, principles, control rules, risk level, required permission level, and weight snapshots. |
| Capability matching | Matching humans, agents, or capability resources by capability needs, risk, context, and candidates. |
| Self-evolution loop | Signals, experiments, verification, knowledge capture, and weight updates form a continuous improvement cycle. |

## Current Capabilities

### Business Lifecycle

The system currently supports a full project lifecycle:

1. **Requirement intake**: create requirements with organization, department, submitter, priority, risk level, budget, and metadata.
2. **Documents and analysis**: upload requirement documents, start requirement-analysis workflows, sync workflow output, and store analysis results.
3. **Requirement approval**: a human or agent actor approves requirements; approval can trigger governance checks and outcome recording.
4. **Project conversion**: convert approved requirements into projects while preserving organization, department, budget, risk, and context.
5. **Project staffing**: add project members, connect organization positions or position assignments, and match participants by capability and risk.
6. **Workflow binding**: bind workflow templates to projects, create workflow instances, and track tasks, decisions, and context.
7. **Deliverable management**: create, update, submit, accept, or reject deliverables.
8. **Cost management**: record cost entries, refresh cost summaries, and aggregate budget consumption by source type.
9. **Feedback evaluation**: create project evaluations, close feedback, and record outcomes and learnable signals in the evolution domain.

### Organization Capabilities

- Multi-organization model and current-organization lookup.
- Department tree, department status, ordering, and metadata.
- Positions, position permission levels, required capabilities, and position assignments.
- Human users, AI agents, and external members as organization members.
- Department-to-MVRU links.
- Connections between organization members, project members, positions, and position assignments.
- Member matching and capability matching for tasks and projects.

### Governance and Evolution

- Permissions, principles, control rules, and access decision records.
- Governance fields for AI agents: origin, service class, provider, contract reference, and risk level.
- Employee profiles, context weights, capability evaluations, and access-decision data structures.
- Weight computation, context-weight computation, outcome recording, experiments, knowledge entries, and signal acknowledgement.

### AI Operations, Tools, and Finance

- Meta-Org Home aggregates organization health, project status, agent status, AI cost, risks, recent events, and inbox items.
- AI Gateway supports OpenAI, Anthropic, and Gemini provider configuration, encrypted secrets, model catalog, streaming calls, invocation logs, and cost summaries.
- Tool Runtime supports tool registry, governance decisions, approval policy, execution audit, and internal tool adapters.
- Developer Tools covers model providers, model catalog, tool registry, interface files, invocation logs, and cost summaries.
- Finance Exports support generic finance adapters, HMAC/Bearer auth, export batches, webhook callbacks, and reconciliation differences.

### Frontend Workspaces

The frontend is an operational single-page workspace:

- Login, registration, session persistence, and logout.
- Chinese and English language switching through `LanguageProvider` and `useI18n`.
- Dashboard overview for identity, organization, workflow, capability, observability, verification, governance, evolution, and recent events.
- Meta-Org Home for organization health, AI cost, risks, inbox items, and contextual AI assistance.
- Draggable sidebar menu groups: business lifecycle, organization capabilities, governance evolution, and system tools.
- Organization workspace for organizations, departments, positions, members, external members, position assignments, MVRU links, and matching.
- Control workspaces for governance, weights, capability evaluations, workflow design, and workflow matching.
- Project lifecycle workspace for requirements, projects, delivery, costs, and feedback.
- Developer Tools for providers, models, tools, interface files, invocation logs, and cost summaries.
- Finance Exports for adapters, export batches, reconciliation, and failed callbacks.
- Context AI Assistant for Meta-Org, organization, project, governance, and developer-tool contexts with streaming and cost display.
- API Workbench for browsing and calling backend APIs by domain, with path parameters, query parameters, request templates, and auth token support.

## Technical Architecture

| Layer | Current Implementation |
|---|---|
| Frontend | Next.js 16 App Router, React 19, TypeScript, Tailwind CSS, lucide-react, @xyflow/react |
| Backend | Go 1.22, Chi Router v5, modular domain-oriented monolith, pgx PostgreSQL driver |
| Database | PostgreSQL 16, root-level SQL migrations, automatically applied by the backend on startup |
| Authentication | JWT Bearer Token with separate public and protected route groups |
| Deployment | Docker Compose starts PostgreSQL, backend, and frontend |

### Backend Structure

The backend entry point is `backend/cmd/server/main.go`. Startup flow:

1. Load environment configuration.
2. Connect to PostgreSQL.
3. Run SQL migrations from `migrations/`.
4. Initialize repositories, services, and handlers for each domain.
5. Register `/api/v1` routes in `backend/internal/gateway/router.go`.
6. Start the HTTP server with graceful shutdown.

Backend domains live under `backend/internal/domain/<domain>/` and typically contain:

- `model.go`: API and database models.
- `repository.go`: PostgreSQL persistence.
- `service.go`: business rules and cross-domain orchestration.
- `handler.go`: HTTP request parsing and responses.

Shared packages live under `backend/internal/pkg/` and cover configuration, database access, migrations, middleware, and server setup.

### Backend Domains

| Domain | Responsibility |
|---|---|
| `identity` | Users, AI agents, roles, login, registration, and agent authentication. |
| `organization` | Organizations, department tree, positions, position assignments, external members, organization members, MVRUs, relationships, and matching. |
| `layer` | Strategic, tactical, and execution layer classification and MVRU layer configuration. |
| `capability` | Capability catalog, capability bindings, capability matching, and capability evaluations. |
| `dashboard` | Aggregated statistics and recent events for the system overview. |
| `metaorg` | Meta-Org Home, organization health, risks, activity, and inbox aggregation. |
| `aigateway` | Model providers, model catalog, streaming calls, invocation logs, and AI usage cost. |
| `toolruntime` | Tool registry, governance policy, approvals, execution audit, and internal tool adapters. |
| `finance` | Generic finance adapters, export batches, webhook callbacks, and reconciliation. |
| `workflow` | Workflow templates, instances, tasks, decisions, and context. |
| `project` | Requirements, documents, requirement-analysis workflows, projects, members, project workflows, delivery, costs, and feedback. |
| `governance` | Permissions, governance principles, control rules, permission checks, and access decisions. |
| `evolution` | Decision weights, context weights, experiments, knowledge entries, signals, and outcome recording. |
| `observability` | Traces, spans, metrics, and execution telemetry. |
| `verification` | Verification reports, review assignments, review completion, and scoring. |

### Frontend Structure

| Path | Description |
|---|---|
| `frontend/src/app/page.tsx` | Main app entry, login/register, layout, overview, menu, and workspace switching. |
| `frontend/src/app/organization-workspace.tsx` | Organization, department, position, member, external member, and MVRU operations. |
| `frontend/src/app/control-workspaces.tsx` | Governance, weights, capability evaluations, workflow design, and workflow matching. |
| `frontend/src/app/project-lifecycle-workspace.tsx` | Requirement, project, delivery, cost, and feedback workspace. |
| `frontend/src/app/api-workbench.tsx` | Generic API calling panel. |
| `frontend/src/app/ai-assistant.tsx` | Context AI Assistant and SSE streaming response panel. |
| `frontend/src/app/developer-tools-workspace.tsx` | Model, tool, interface file, invocation log, and cost views. |
| `frontend/src/app/finance-workspace.tsx` | Finance adapter, export batch, reconciliation, and failed callback views. |
| `frontend/src/lib/api.ts` | API request wrapper, base types, and dashboard data shapes. |
| `frontend/src/lib/operations.ts` | API Workbench domain, path, parameter, and body-template metadata. |
| `frontend/src/lib/i18n.tsx` | Chinese and English language packs and i18n provider. |
| `frontend/src/lib/auth.ts` | Token and session storage. |

## Database Migrations

The backend applies SQL files from the root `migrations/` directory at startup. The current migration set goes through `018`:

| Migration | Topic |
|---|---|
| `001_identity.sql` | schema migrations, users, ai_agents, roles, user_roles, agent_roles. |
| `002_seed_roles.sql` | Seed planner, executor, and reviewer roles. |
| `003_organization.sql` | organizations, muvrs, teams, mvru_members, mvru_relationships. |
| `004_layer.sql` | layer_configs, layer_routing_rules. |
| `005_capability.sql` | capabilities, capability_bindings, capability_invocations. |
| `006_workflow.sql` | workflow_templates, workflow_instances, tasks, decisions, workflow_contexts. |
| `007_observability.sql` | traces, spans, metrics. |
| `008_verification.sql` | verification_reports, review_assignments. |
| `009_governance.sql` | permissions, principles, control_rules. |
| `010_evolution.sql` | weight_scores, weight_alphas, experiments, knowledge_entries, signals. |
| `011_organization_tree.sql` | departments, external_members, organization_memberships, department_mvru_links. |
| `012_policy_weight_evaluation.sql` | Agent governance fields, employee_profiles, access_decisions, context_weight_scores, capability_evaluations. |
| `013_project_lifecycle.sql` | requirements, projects, project_members, project_workflows, deliverables, project_cost_entries, project_evaluations. |
| `014_requirement_documents_workflow_analysis.sql` | requirement_documents, requirement_analysis_workflows. |
| `015_single_org_positions_workflow_graph.sql` | positions, position_assignments, plus organization, department, and position links for workflows and project members. |
| `016_ai_gateway.sql` | Model providers, model catalog, price versions, AI invocations, and AI usage ledger. |
| `017_tool_runtime.sql` | Tool definitions, interface files, tool executions, approvals, and initial internal tools. |
| `018_finance_exports.sql` | Finance adapters, export batches, export lines, webhook events, and AI cost posting constraints. |

## API Overview

All API routes are mounted under `/api/v1`.

Public routes:

- `GET /health`
- `POST /auth/login`
- `POST /auth/register`
- `POST /agents/auth`
- `GET /roles`

All other business routes require a JWT Bearer Token.

| Domain | Main Routes |
|---|---|
| Dashboard | `GET /dashboard/overview` |
| Meta-Org | `GET /meta-org/overview`, `GET /meta-org/inbox` |
| Identity | `POST /agents/register`, `GET /agents` |
| AI Gateway | Model providers, model catalog, `POST /ai-gateway/invoke`, `GET /ai-gateway/stream`, invocation logs, and cost summary routes |
| Tool Runtime | Tool definition, tool test, tool execution log, and tool approval routes |
| Finance | Finance adapter, export batch, submit export, webhook callback, and reconciliation routes |
| Organization | `GET/POST/PATCH /organizations`, `GET /organization/current`, plus department, department tree, position, position assignment, organization member, external member, MVRU, relationship, member-matching, and capability-matching routes |
| Layer | `POST /layers/classify`, `GET/PUT /layers/config/{mvruId}`, `GET /layers/rules` |
| Capability | `GET/POST /capabilities`, `GET /capabilities/{id}`, `POST /capabilities/match`, capability evaluation, binding, and unbinding routes |
| Workflow | Workflow template, instance, status, task completion, decision recording, and context read/write routes |
| Project Lifecycle | Requirement, requirement document, requirement-analysis workflow, approval, project conversion, project member, project workflow, project overview, deliverable, cost, and feedback routes |
| Governance | Permission, principle, control rule, permission check, access decision, and access decision list routes |
| Evolution | Weight computation, outcome recording, context weights, alpha config, experiment, knowledge, signal, and signal acknowledgement routes |
| Observability | Trace, span, trace completion, metric recording, and metric query routes |
| Verification | Verification report, report query, review assignment, and review completion routes |

Frontend API Workbench metadata lives in `frontend/src/lib/operations.ts`. It groups operations by MetaOrg, DeveloperTools, Finance, Dashboard, Identity, Organization, Layer, Capability, Workflow, Observability, Verification, Governance, Evolution, Requirement, Project, Delivery, Cost, and Feedback.

## Quick Start

Start the full local environment with Docker Compose:

```bash
docker compose up --build
```

Service addresses:

- PostgreSQL: `localhost:5432`
- Go API: `http://localhost:8080`
- API health: `http://localhost:8080/api/v1/health`
- Next.js frontend: `http://localhost:3000`

Default Docker environment values are defined in `docker-compose.yml`:

- Database: `postgres://postgres:postgres@postgres:5432/meta_org?sslmode=disable`
- Backend port: `8080`
- Model and finance secret encryption: `MODEL_SECRET_KEY=0123456789abcdef0123456789abcdef`
- Frontend API URL: `http://localhost:8080/api/v1`

## Local Development

Backend:

```bash
cd backend
go run ./cmd/server
go test ./...
go build ./cmd/server
```

When running the backend outside Docker, provide PostgreSQL and set:

```bash
MIGRATIONS_PATH=../migrations
```

PowerShell:

```powershell
$env:MIGRATIONS_PATH = '../migrations'
go run ./cmd/server
```

Frontend:

```bash
cd frontend
npm install
npm run dev
npm run lint
npm run build
```

The frontend defaults to:

```bash
NEXT_PUBLIC_API_URL=http://127.0.0.1:8080/api/v1
```

## Configuration

Backend configuration is loaded in `backend/internal/pkg/config/config.go`:

| Environment Variable | Default | Description |
|---|---|---|
| `SERVER_PORT` | `8080` | Backend listen port. |
| `DATABASE_URL` | `postgres://postgres:postgres@localhost:5432/meta_org?sslmode=disable` | PostgreSQL connection string. |
| `JWT_SECRET` | `dev-secret-change-in-production` | JWT signing secret. Replace in production. |
| `MODEL_SECRET_KEY` | `0123456789abcdef0123456789abcdef` | 32-character key for model provider and finance adapter secret encryption. Replace in production. |
| `CORS_ORIGINS` | `http://localhost:3000,http://127.0.0.1:3000` | Frontend origins allowed to call the API. |
| `MIGRATIONS_PATH` | `migrations` | SQL migration directory; when running from `backend/`, usually set it to `../migrations`. |

Frontend configuration:

| Environment Variable | Default | Description |
|---|---|---|
| `NEXT_PUBLIC_API_URL` | `http://127.0.0.1:8080/api/v1` | Browser-side API base URL. |

## Project Structure

```text
backend/
  cmd/server/                 Backend entry point
  internal/domain/            Domain modules
  internal/gateway/           Route registration
  internal/pkg/               Config, database, migrations, middleware, server
frontend/
  src/app/                    Next.js App Router pages and workspaces
  src/lib/                    API, auth, i18n, API Workbench metadata
docker-compose.yml            Full local environment orchestration
migrations/                   PostgreSQL SQL migrations 001-018
docs/operations/              Production operations and finance adapter protocol docs
.github/workflows/            GitHub Actions CI
```

## Current Status and Boundaries

The codebase now provides a single-enterprise Meta-Org entry, organization management, project lifecycle, AI Gateway, tool runtime loop, cost accounting, finance exports, governance, evolution, observability, and verification foundation. It is suitable as a production v1 base for 10-50 humans and 50-250+ agents.

When upgrading from the old `harness_org` database to `meta_org`, explicitly back up and migrate data first. The system does not automatically delete or overwrite the old database.

Important next steps:

- Expand model capabilities, agent executors, and external tool runtimes.
- Expand the MVRU sandbox concept from data model to isolated execution environment.
- Add automated frontend-state and end-to-end tests for critical workflows.
- Improve production-grade secret management, audit reports, alerts, and permission-policy visualization.
- Extend multi-organization tenant boundaries, approval-flow templates, and finer-grained operation audit trails.
