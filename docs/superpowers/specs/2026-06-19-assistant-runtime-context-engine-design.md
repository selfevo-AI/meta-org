# Assistant Runtime and Verified Context Engine Design

Date: 2026-06-19

## Purpose

This design upgrades the Meta-Org AI assistant from a session-level model/tool loop into a governed organizational assistant runtime. The first phase prioritizes framework and extension points over broad feature completion.

The design combines two foundations:

- Pi Agent Harness concepts from `D:\project\pi-main`: agent loop, harness, tool lifecycle, session entries, compaction, skills, and provider abstraction.
- Meta-Org business constraints: SaaS/organization/module permissions, workflow stages, finance validation, governance approvals, cost attribution, and audited business context.

The result is a Go-native runtime that keeps the current API surface mostly stable while making context, tools, data dictionaries, and future database evolution deterministic and extensible.

## Goals

- Split the current assistant service loop into stable backend boundaries: runtime, harness, context engine, tool runner, event sink, and dictionary service.
- Replace hardcoded context SQL as the core model with a Verified Context Engine backed by approved database metadata and business validation rules.
- Ensure context is controlled by business process, permissions, workflow state, and finance validation, not AI inference.
- Support data dictionary import from JSON/YAML/CSV/Excel through one normalized intermediate model.
- Allow AI to generate dictionary suggestions and migration drafts without directly activating rules or executing DDL.
- Preserve current assistant session APIs as much as possible so the frontend can continue working with minimal changes.
- Establish extension points for future business domains, context compaction, session replay, business skills, and database restructuring.

## Non-Goals

- Do not let AI execute DDL automatically.
- Do not migrate every existing module to the new dictionary framework in phase one.
- Do not build metadata-driven frontend forms in phase one.
- Do not introduce a graph database or vector retrieval as a required dependency.
- Do not implement full automatic weight learning. AI may suggest weights; active weights remain approved rules.
- Do not import the Pi TypeScript runtime as a dependency. Meta-Org implements the relevant framework concepts in Go.

## Pi Framework Mapping

Pi provides the agent engineering model. Meta-Org provides the organizational control layer.

| Pi concept | Meta-Org mapping | Adaptation |
|---|---|---|
| `packages/agent/src/agent-loop.ts` | `AssistantRuntime` | Go loop for turn lifecycle, tool calls, tool results, approval pause/resume, and event emission. |
| `packages/agent/src/harness/agent-harness.ts` | `AssistantHarness` | Runtime container for session, actor, module, workflow, context package, tool set, attribution, and event sink. |
| `packages/agent/src/harness/session/*` | Assistant runtime entries / assistant messages / assistant steps | Database-backed session history and event replay. Full branch navigation can come later. |
| `packages/agent/src/harness/compaction/*` | Context compaction extension point | Phase one records context package provenance and budgets; later phases add long-session summary compaction. |
| `packages/agent/src/harness/skills.ts` | `assistant_business_skills` evolution | Skills bind to module, target entity, tool allowlist, permission tier, and approval status. |
| `packages/ai/src/types.ts` | `aigateway` message/provider model | Reuse the ideas of unified messages, tools, reasoning, usage, diagnostics, and provider options without importing TS code. |

The important adaptation is that Pi's generic coding-agent harness becomes a business harness. It must obey permissions, workflows, finance validation, governance, and cost attribution before the model sees context or calls tools.

## Architecture

### AssistantService

`AssistantService` remains the application service behind the existing assistant handlers. It should become thin:

- Validate request input.
- Resolve authenticated actor and tenant scope.
- Load or create assistant sessions.
- Create runtime requests.
- Return runtime event streams.

It should not own the full turn loop, prompt assembly, context fetching, or tool control logic.

### AssistantRuntime

`AssistantRuntime` owns the turn lifecycle:

1. Load session.
2. Create `AssistantHarness`.
3. Append user message.
4. Build verified context package.
5. Build model messages.
6. Invoke AI Gateway.
7. Persist assistant message and LLM step.
8. Execute tool calls through `ToolRunner`.
9. Pause for approval when required.
10. Resume after approval by rebuilding context.
11. Update memory and proposals.
12. Emit completion or error events.

The runtime should expose:

```text
Run(ctx, request) -> event stream
Resume(ctx, request) -> event stream
```

### AssistantHarness

`AssistantHarness` is a deterministic container for one run or resumed run. It freezes:

- Actor ID, actor type, role, position, and authority context.
- SaaS, organization, and module scope.
- Session, target entity, workflow, task, and mode.
- Model provider, channel, model, service tier, and reasoning effort.
- Verified `ContextPackage`.
- Allowed tool set.
- Cost attribution metadata.
- Event sink and observability trace.

AI may not mutate harness boundaries inside a turn. A later turn may rebuild the harness, but only through the same validation chain.

### EventSink

`EventSink` centralizes persistence and streaming:

- `assistant_messages`
- `assistant_steps`
- Observability traces/spans/metrics
- SSE events
- Context package references

Runtime events should include at least:

- `run_started`
- `context_built`
- `message_appended`
- `llm_invoked`
- `tool_requested`
- `tool_blocked`
- `approval_required`
- `tool_completed`
- `context_invalidated`
- `proposal_created`
- `memory_updated`
- `run_completed`
- `run_failed`

Existing `assistant_steps` can carry these events in phase one, with richer `data` containing context package ID, rule versions, tool policy, approval reason, and validation summaries.

## Verified Context Engine

The current context resolver is too close to hardcoded module SQL. The new engine must be deterministic, database-backed, and controlled by business validation.

### Core Rule

`ContextEngine` only executes active, approved, verified context rules. AI can suggest rules but cannot activate them.

The engine exposes:

```text
BuildContextPackage(ctx, request)
ValidateDictionary(ctx, importModel)
PreviewContext(ctx, request)
```

### ContextRequest

Every request must include enough scope to make context deterministic:

- `ActorID`
- `ActorType`
- `OrganizationID`
- `ModuleKey`
- `WorkflowID`
- `TaskID`
- `TargetType`
- `TargetID`
- `Intent`
- `Mode`
- `RiskLevel`

### Business Validation Chain

Context is admitted through three validation gates:

1. Permission validation
   SaaS, organization, and module-level rules decide entity and field visibility. Sensitive fields need explicit active rules.

2. Workflow validation
   Fields and records are weighted by workflow stage. Requirement analysis, project execution, delivery acceptance, and finance accounting should not use the same context profile.

3. Finance validation
   Money, budget, cost, settlement, receivable, payable, invoice, export, and reconciliation fields require finance-domain validation. Unverified finance values may be included only as risk signals, not confirmed facts.

### ContextPackage

The engine returns a `ContextPackage`, not raw rows:

```text
ContextPackage
- id
- scope
- attention_core
- supporting_context
- risk_and_signals
- omissions
- weights
- validations
- provenance
```

`facts` from earlier discussions map to `attention_core` when they are required for the current decision, and to `supporting_context` when useful but not immediately decisive.

### Attention Core

The engine must not maximize context volume. It must produce a small, reliable attention core.

`AttentionCore` contains:

- Minimum facts required for the current task.
- Current target object state.
- Fields required for the next decision.
- Verified permission, workflow, finance, and governance conclusions.

Rules:

- `AttentionCore` has a hard budget, roughly 30%-40% of total prompt context.
- `SupportingContext` is ordered by weight and can be summarized or truncated.
- `RiskAndSignals` contains concise warnings and references.
- `Omissions` explains why something was excluded without leaking sensitive values.
- If context exceeds budget, the engine degrades to summary, references, and suggested follow-up queries.
- The model may make deterministic business judgments only from `AttentionCore`.

Design principle:

> ContextEngine does not use the model window as a database cache. It produces a small, verified, auditable AttentionCore.

### Weight Model

Field and record weight is a candidate score, not an automatic admission decision:

```text
final_weight =
  field_base_weight
  * domain_weight
  * workflow_stage_multiplier
  * permission_visibility_multiplier
  * finance_validation_multiplier
  * freshness_multiplier
  * relation_distance_multiplier
  * target_relevance_multiplier
  * risk_penalty
```

Hard constraints:

- Permission failure sets weight to zero.
- Financial conflict prevents use as confirmed fact.
- High-sensitive governance, finance, approval, and model-configuration fields require explicit active rules.
- AI-suggested weights remain proposals until approved.

### Query Safety

Dictionary metadata must not become arbitrary SQL execution. Phase one supports controlled templates only:

- Single entity by ID.
- Entity list by allowed filters.
- Whitelisted relationship joins.
- Restricted aggregates for budget, cost, status counts, and similar validated metrics.
- Forced tenant and organization scoping.
- Field selection from active physical mappings only.

## Data Dictionary and Database Evolution

### Dictionary Tables

The metadata model should be split into these logical table groups:

- `context_dictionary_versions`
- `context_business_domains`
- `context_entities`
- `context_fields`
- `context_physical_mappings`
- `context_rules`
- `context_change_proposals`
- `context_migration_drafts`
- `context_packages`

Use these table names by default. Implementation may adjust a name only when it conflicts with an existing table or repository convention, and the same separation of concepts must remain.

### Logical Model vs Physical Mapping

The dictionary distinguishes logical business model from physical database structure:

- Logical model: business domain, entity, field, relationship, workflow context profile, finance validation profile.
- Physical mapping: PostgreSQL table, column, join path, indexes, and migration drafts.

Future database restructuring should update physical mappings and migration drafts without changing assistant runtime logic.

### Import Flow

All formats flow through one intermediate model:

```text
DictionaryImportModel
- version
- source_type
- scope_level: saas / organization / module
- organization_id
- module_key
- domains
- entities
- fields
- relationships
- workflow_context_profiles
- finance_validation_profiles
- permissions
- migration_intents
```

Import sources:

- JSON/YAML for versioned technical definitions.
- CSV/Excel for business-maintained dictionaries.

### Change Flow

```text
Import file
-> normalize
-> validate structure
-> map business domains
-> check permissions
-> ask AI for suggestions
-> create change proposal
-> human approval
-> activate context rules
-> create migration draft when needed
-> human executes DDL outside AI
-> verify physical mapping
-> allow context usage
```

AI may:

- Explain field meanings.
- Suggest weights.
- Suggest relationships.
- Suggest desensitization.
- Detect conflicts.
- Draft SQL migrations and rollback notes.

AI may not:

- Activate rules.
- Raise permissions.
- Execute DDL.
- Treat imported fields as usable context before validation.

### Permission Levels

Dictionary and context changes are scoped at three levels:

- SaaS level: global defaults and cross-organization templates.
- Organization level: organization-specific overrides.
- Module/business-domain level: finance, project, organization, governance, and similar domain rules.

A module-level import that attempts to change SaaS-level or cross-module rules must produce a blocked proposal item rather than taking effect.

### Self-Consistency Rule

```text
AI suggestion != active rule
Active rule != physical database structure
Physical database structure != usable context
Usable context = active rule + permission validation + workflow validation + finance validation
```

This is the core safety model.

## Tool Governance

`ToolRunner` wraps the existing `toolruntime.Service`.

Before a tool call:

- Check module tool allowlist.
- Check business skill allowlist when a skill is active.
- Check actor authority at SaaS, organization, and module scope.
- Check the target entity is inside the current context package operation scope.
- Check governance decision.
- Force approval for finance, governance, model configuration, external, high-risk, or destructive operations.

After a tool call:

- Persist tool result.
- Record observability span.
- Record cost and attribution where applicable.
- Produce context invalidation hints.
- Mark finance or governance verification impacts.

Approval resume must rebuild context before continuing. The runtime should not keep reasoning from an old context package after a tool has changed business state.

## Error Handling

Runtime failures should be classified:

- `context_error`
- `provider_error`
- `tool_error`
- `governance_denied`
- `finance_validation_failed`
- `approval_rejected`
- `runtime_error`

This avoids treating all failures as model/provider problems and improves frontend diagnostics.

## Phase One Scope

Phase one builds the framework and extension points. It should not try to complete every module.

### Required Framework

- `AssistantRuntime`
- `AssistantHarness`
- `EventSink`
- `ContextEngine`
- `ContextRuleEvaluator`
- `ToolRunner`
- `DictionaryService`

### Seed Domains

Use three domains to prove the framework:

- `project`: requirements, projects, tasks, deliverables.
- `finance`: costs, budgets, settlements, receivables/payables.
- `governance`: permissions, access decisions, approvals.

These domains cover workflow, finance validation, and permission governance.

### Extension Requirements

The framework must allow future additions without changing the runtime loop:

- Add business domain through dictionary version and active rules.
- Add field through logical field, physical mapping, and rule.
- Add weight behavior through rule evaluator or approved configuration.
- Add finance validation through a validator adapter.
- Add tool through Tool Runtime registration and ToolRunner allowlist.
- Handle database restructuring through migration draft and physical mapping updates.

## Acceptance Criteria

- Existing assistant session API remains broadly compatible.
- The assistant service no longer owns the full turn loop.
- Runtime events are persisted and streamable.
- Context packages record dictionary version, rule version, permissions, workflow validation, finance validation, weights, omissions, and provenance.
- Tool calls pass through a unified before/after control boundary.
- Dictionary import can create proposals and migration drafts without executing DDL.
- Project, finance, and governance have seed dictionary/rule coverage.
- Tests cover runtime loop, context rule evaluation, dictionary validation, tool gating, and attention budget behavior.

## Testing Strategy

Backend tests should focus on deterministic behavior:

- Runtime loop with fake AI and fake tools.
- Approval pause and resume with context rebuild.
- Context package generation under permission allow/deny.
- Workflow-stage weight changes.
- Finance validation success, warning, and failure paths.
- Dictionary import normalization for JSON/YAML/CSV-like inputs.
- AI suggestion stored as proposal, not active rule.
- Tool allowlist and governance blocking.
- AttentionCore budget enforcement and supporting context truncation.

## Implementation Notes

- Keep existing user changes in the worktree intact.
- Avoid broad refactors outside assistant, context, dictionary metadata, and tool runtime integration.
- Add migrations append-only.
- Do not remove existing hardcoded context queries until seed dictionary coverage can replace them safely.
- A compatibility adapter may bridge old context resolver behavior into the new `ContextEngine` while migrating modules.
