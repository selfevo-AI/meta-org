export const API_BASE = process.env.NEXT_PUBLIC_API_URL || 'http://127.0.0.1:8080/api/v1'

interface RequestOptions {
  method?: string
  body?: unknown
  token?: string
}

export async function apiRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const isFormData = typeof FormData !== 'undefined' && options.body instanceof FormData
  const headers: Record<string, string> = {}

  if (options.token) {
    headers['Authorization'] = `Bearer ${options.token}`
  }
  if (!isFormData) {
    headers['Content-Type'] = 'application/json'
  }
  const requestBody = options.body ? (isFormData ? (options.body as BodyInit) : JSON.stringify(options.body)) : undefined

  const response = await fetch(`${API_BASE}${path}`, {
    method: options.method || 'GET',
    headers,
    body: requestBody,
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Unknown error' }))
    throw new Error(error.error || `HTTP ${response.status}`)
  }

  return response.json()
}

export async function login(email: string, password: string): Promise<AuthResponse> {
  return apiRequest<AuthResponse>('/auth/login', {
    method: 'POST',
    body: { email, password },
  })
}

export async function registerUser(input: RegisterUserInput): Promise<UserResponse> {
  return apiRequest<UserResponse>('/auth/register', {
    method: 'POST',
    body: input,
  })
}

export async function listRoles(): Promise<Role[]> {
  return apiRequest<Role[]>('/roles')
}

export async function getDashboardOverview(token: string): Promise<DashboardOverview> {
  return apiRequest<DashboardOverview>('/dashboard/overview', { token })
}

export async function getMetaOrgOverview(token: string): Promise<MetaOrgOverview> {
  return apiRequest<MetaOrgOverview>('/meta-org/overview', { token })
}

export async function getMetaOrgInbox(token: string): Promise<InboxItem[]> {
  return apiRequest<InboxItem[]>('/meta-org/inbox', { token })
}

export async function listModelProviders(token: string): Promise<ModelProvider[]> {
  return apiRequest<ModelProvider[]>('/model-providers', { token })
}

export async function createModelProvider(token: string, input: CreateModelProviderInput): Promise<ModelProvider> {
  return apiRequest<ModelProvider>('/model-providers', { method: 'POST', token, body: input })
}

export async function rotateModelProviderKey(token: string, id: string, apiKey: string): Promise<ModelProvider> {
  return apiRequest<ModelProvider>(`/model-providers/${id}/rotate-key`, {
    method: 'POST',
    token,
    body: { api_key: apiKey },
  })
}

export async function testModelProvider(token: string, id: string, model?: string): Promise<{ status: string }> {
  return apiRequest<{ status: string }>(`/model-providers/${id}/test`, {
    method: 'POST',
    token,
    body: { model },
  })
}

export async function listModels(token: string): Promise<ModelCatalogItem[]> {
  return apiRequest<ModelCatalogItem[]>('/models', { token })
}

export async function createModel(token: string, input: CreateModelInput): Promise<ModelCatalogItem> {
  return apiRequest<ModelCatalogItem>('/models', { method: 'POST', token, body: input })
}

export async function listTools(token: string): Promise<ToolDefinition[]> {
  return apiRequest<ToolDefinition[]>('/tools', { token })
}

export async function listToolExecutions(token: string): Promise<ToolExecution[]> {
  return apiRequest<ToolExecution[]>('/tool-executions', { token })
}

export async function listInvocations(token: string): Promise<AIInvocation[]> {
  return apiRequest<AIInvocation[]>('/ai-gateway/invocations', { token })
}

export async function getAIInvocation(token: string, id: string): Promise<AIInvocation> {
  return apiRequest<AIInvocation>(`/ai-gateway/invocations/${id}`, { token })
}

export async function getAICostSummary(token: string): Promise<AICostSummary> {
  return apiRequest<AICostSummary>('/ai-gateway/cost-summary', { token })
}

export async function listFinanceAdapters(token: string): Promise<FinanceAdapter[]> {
  return apiRequest<FinanceAdapter[]>('/finance/adapters', { token })
}

export async function createFinanceAdapter(token: string, input: CreateFinanceAdapterInput): Promise<FinanceAdapter> {
  return apiRequest<FinanceAdapter>('/finance/adapters', { method: 'POST', token, body: input })
}

export async function testFinanceAdapter(token: string, id: string): Promise<{ status: string }> {
  return apiRequest<{ status: string }>(`/finance/adapters/${id}/test`, { method: 'POST', token })
}

export async function createFinanceExportBatch(
  token: string,
  input: CreateFinanceExportBatchInput,
): Promise<FinanceExportBatch> {
  return apiRequest<FinanceExportBatch>('/finance/export-batches', { method: 'POST', token, body: input })
}

export async function listFinanceExportBatches(token: string): Promise<FinanceExportBatch[]> {
  return apiRequest<FinanceExportBatch[]>('/finance/export-batches', { token })
}

export async function getFinanceExportBatch(token: string, id: string): Promise<FinanceExportBatch> {
  return apiRequest<FinanceExportBatch>(`/finance/export-batches/${id}`, { token })
}

export async function submitFinanceExportBatch(token: string, id: string): Promise<FinanceExportBatch> {
  return apiRequest<FinanceExportBatch>(`/finance/export-batches/${id}/submit`, { method: 'POST', token })
}

export async function listFinanceReconciliation(token: string): Promise<FinanceReconciliationItem[]> {
  return apiRequest<FinanceReconciliationItem[]>('/finance/reconciliation', { token })
}

export interface AuthResponse {
  token: string
  user_id: string
  user_type: 'human' | 'ai'
  expires_at: number
}

export interface UserResponse {
  id: string
  name: string
  email: string
  avatar_url?: string
  created_at: string
  updated_at: string
}

export interface RegisterUserInput {
  name: string
  email: string
  password: string
}

export interface Role {
  id: string
  name: string
  role_type: 'planner' | 'executor' | 'reviewer'
  description?: string
  permissions: string[]
}

export interface AIAgent {
  id: string
  name: string
  model_type: string
  capabilities: string[]
  permission_level: string
  agent_origin: 'internal' | 'external'
  provider?: string
  service_class: string
  vendor?: string
  contract_ref?: string
  risk_level: 'low' | 'medium' | 'high' | 'critical'
  metadata: Record<string, unknown>
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface DashboardOverview {
  generated_at: string
  identity: {
    users: number
    active_agents: number
    total_agents: number
    roles: number
  }
  organization: {
    organizations: number
    mvrus: number
    mvrus_by_status: Record<string, number>
    members: number
    relationships: number
  }
  workflow: {
    templates: number
    active_templates: number
    instances: number
    instances_by_status: Record<string, number>
    tasks_by_status: Record<string, number>
    decisions_7d: number
  }
  capability: {
    capabilities: number
    active_capabilities: number
    bindings: number
    invocations_24h: number
    failed_invocations_24h: number
    average_duration_ms: number
    cost_24h: number
  }
  observability: {
    active_traces: number
    completed_traces: number
    failed_traces: number
    spans_24h: number
    metrics_24h: number
  }
  verification: {
    reports: number
    average_score: number
    pending_reviews: number
  }
  governance: {
    permissions: number
    active_principles: number
    control_rules: number
    active_control_rules: number
  }
  evolution: {
    weighted_actors: number
    experiments_by_status: Record<string, number>
    knowledge_entries: number
    unacknowledged_signals: number
    high_priority_signals: number
  }
  recent_events: RecentEvent[]
}

export interface MetaOrgOverview {
  generated_at: string
  health: {
    open_requirements: number
    active_projects: number
    active_agents: number
    pending_approvals: number
    unexported_cost: number
    currency: string
  }
  projects: {
    by_status: Record<string, number>
    over_budget: number
  }
  agents: {
    total: number
    active: number
    by_risk_level: Record<string, number>
  }
  cost: {
    today: number
    month_to_date: number
    unexported: number
    currency: string
    by_provider: Record<string, number>
  }
  risks: Array<{ id: string; title: string; severity: string; source: string }>
  activity: RecentEvent[]
}

export interface InboxItem {
  id: string
  type: string
  title: string
  status: string
  priority: string
  source?: string
  created_at: string
}

export interface RecentEvent {
  id: string
  type: string
  title: string
  status?: string
  created_at: string
}

export interface ModelProvider {
  id: string
  name: string
  provider_type: 'openai' | 'anthropic' | 'gemini'
  base_url: string
  masked_api_key: string
  status: string
  timeout_ms: number
  retry_count: number
  risk_level: string
  tags: string[]
  metadata: Record<string, unknown>
  last_test_status: string
  last_test_error?: string
  last_tested_at?: string
  created_at: string
  updated_at: string
}

export interface CreateModelProviderInput {
  name: string
  provider_type: 'openai' | 'anthropic' | 'gemini'
  base_url?: string
  api_key: string
  risk_level?: string
  timeout_ms?: number
  retry_count?: number
  tags?: string[]
  metadata?: Record<string, unknown>
}

export interface ModelCatalogItem {
  id: string
  provider_id: string
  model_key: string
  display_name: string
  context_window: number
  max_output_tokens: number
  capabilities: string[]
  status: string
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CreateModelInput {
  provider_id: string
  model_key: string
  display_name?: string
  context_window?: number
  max_output_tokens?: number
  capabilities?: string[]
  status?: string
  input_price_per_1k?: number
  output_price_per_1k?: number
  currency?: string
  metadata?: Record<string, unknown>
}

export interface ToolDefinition {
  id: string
  name: string
  description: string
  source_type: string
  default_policy: string
  risk_level: string
  required_level: string
  status: string
  input_schema: Record<string, unknown>
  output_schema: Record<string, unknown>
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface ToolExecution {
  id: string
  tool_id: string
  tool_name?: string
  actor_id: string
  actor_type: string
  policy: string
  status: string
  result_summary?: string
  error_message?: string
  created_at: string
  completed_at?: string
}

export interface AIInvocation {
  id: string
  provider_id: string
  model_id: string
  mode: string
  status: string
  cost_amount: number
  currency: string
  input_tokens: number
  output_tokens: number
  error_message?: string
  created_at: string
  completed_at?: string
}

export interface AICostSummary {
  total: number
  unexported: number
  currency: string
  by_provider: Record<string, number>
}

export interface FinanceAdapter {
  id: string
  name: string
  endpoint_url: string
  auth_type: 'hmac' | 'bearer'
  masked_secret: string
  status: string
  timeout_ms: number
  retry_count: number
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CreateFinanceAdapterInput {
  name: string
  endpoint_url: string
  auth_type: 'hmac' | 'bearer'
  secret: string
  timeout_ms?: number
  retry_count?: number
  metadata?: Record<string, unknown>
}

export interface FinanceExportLine {
  id: string
  batch_id: string
  usage_ledger_id?: string
  project_cost_entry_id?: string
  project_id?: string
  provider_id?: string
  model_id?: string
  amount: number
  currency: string
  external_line_id: string
  status: string
  metadata: Record<string, unknown>
  created_at: string
}

export interface FinanceExportBatch {
  id: string
  adapter_id: string
  period_start: string
  period_end: string
  status: string
  currency: string
  total_amount: number
  external_batch_id: string
  error_message: string
  idempotency_key: string
  metadata: Record<string, unknown>
  lines?: FinanceExportLine[]
  created_at: string
  submitted_at?: string
  updated_at: string
}

export interface CreateFinanceExportBatchInput {
  adapter_id: string
  period_start: string
  period_end: string
  currency?: string
  metadata?: Record<string, unknown>
}

export interface FinanceReconciliationItem {
  batch_id: string
  adapter_id: string
  status: string
  currency: string
  total_amount: number
  external_amount: number
  difference_amount: number
  external_batch_id: string
  error_message: string
  submitted_at?: string
  updated_at: string
}
