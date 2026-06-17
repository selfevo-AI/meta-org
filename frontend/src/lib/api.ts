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

export async function listDataTables(token: string, category?: string): Promise<DataTable[]> {
  const query = category ? `?category=${encodeURIComponent(category)}` : ''
  return apiRequest<DataTable[]>(`/governance/data/tables${query}`, { token })
}

export async function listDataFields(token: string, tableName: string): Promise<DataField[]> {
  return apiRequest<DataField[]>(`/governance/data/tables/${encodeURIComponent(tableName)}/fields`, { token })
}

export async function getUserFieldPreference(token: string, tableName: string): Promise<UserFieldPreference> {
  return apiRequest<UserFieldPreference>(`/governance/data/field-preferences/${encodeURIComponent(tableName)}`, { token })
}

export async function saveUserFieldPreference(token: string, tableName: string, input: SaveUserFieldPreferenceInput): Promise<UserFieldPreference> {
  return apiRequest<UserFieldPreference>(`/governance/data/field-preferences/${encodeURIComponent(tableName)}`, {
    method: 'PUT',
    token,
    body: input,
  })
}

export async function createFieldPermissionRule(token: string, input: CreateFieldPermissionRuleInput): Promise<FieldPermissionRule> {
  return apiRequest<FieldPermissionRule>('/governance/data/field-permissions', {
    method: 'POST',
    token,
    body: input,
  })
}

export async function listFieldPermissionRules(token: string, tableName?: string): Promise<FieldPermissionRule[]> {
  const query = tableName ? `?table=${encodeURIComponent(tableName)}` : ''
  return apiRequest<FieldPermissionRule[]>(`/governance/data/field-permissions${query}`, { token })
}

export async function checkFieldAccess(token: string, input: FieldAccessCheckInput): Promise<FieldAccessCheckResult> {
  return apiRequest<FieldAccessCheckResult>('/governance/data/field-access/check', {
    method: 'POST',
    token,
    body: input,
  })
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

export async function listMetaResources(token: string, filter: { resource_type?: string; status?: string } = {}): Promise<MetaResource[]> {
  const params = new URLSearchParams()
  if (filter.resource_type) params.set('resource_type', filter.resource_type)
  if (filter.status) params.set('status', filter.status)
  const query = params.toString() ? `?${params.toString()}` : ''
  return apiRequest<MetaResource[]>(`/meta-resources${query}`, { token })
}

export async function createMetaResource(token: string, input: CreateMetaResourceInput): Promise<MetaResource> {
  return apiRequest<MetaResource>('/meta-resources', { method: 'POST', token, body: input })
}

export async function syncExistingMetaResources(token: string): Promise<Record<string, number>> {
  return apiRequest<Record<string, number>>('/meta-resources/sync-existing', { method: 'POST', token })
}

export async function getMetaResourceSummary(token: string): Promise<MetaResourceSummary> {
  return apiRequest<MetaResourceSummary>('/meta-resources/summary', { token })
}

export async function listDemandProfiles(token: string): Promise<DemandProfile[]> {
  return apiRequest<DemandProfile[]>('/demand-profiles', { token })
}

export async function createDemandProfile(token: string, input: CreateDemandProfileInput): Promise<DemandProfile> {
  return apiRequest<DemandProfile>('/demand-profiles', { method: 'POST', token, body: input })
}

export async function listPDCACycles(token: string): Promise<PDCACycle[]> {
  return apiRequest<PDCACycle[]>('/pdca-cycles', { token })
}

export async function createPDCACycle(token: string, input: CreatePDCACycleInput): Promise<PDCACycle> {
  return apiRequest<PDCACycle>('/pdca-cycles', { method: 'POST', token, body: input })
}

export async function listPDCAEvents(token: string, cycleID?: string): Promise<PDCAEvent[]> {
  const query = cycleID ? `?cycle_id=${encodeURIComponent(cycleID)}` : ''
  return apiRequest<PDCAEvent[]>(`/pdca-events${query}`, { token })
}

export async function createPDCAEvent(token: string, input: CreatePDCAEventInput): Promise<PDCAEvent> {
  return apiRequest<PDCAEvent>('/pdca-events', { method: 'POST', token, body: input })
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

export async function listProviderChannels(token: string, providerID?: string): Promise<ProviderChannel[]> {
  const query = providerID ? `?provider_id=${encodeURIComponent(providerID)}` : ''
  return apiRequest<ProviderChannel[]>(`/model-provider-channels${query}`, { token })
}

export async function createProviderChannel(token: string, providerID: string, input: CreateProviderChannelInput): Promise<ProviderChannel> {
  return apiRequest<ProviderChannel>(`/model-providers/${providerID}/channels`, { method: 'POST', token, body: input })
}

export async function updateProviderChannel(token: string, id: string, input: UpdateProviderChannelInput): Promise<ProviderChannel> {
  return apiRequest<ProviderChannel>(`/model-provider-channels/${id}`, { method: 'PATCH', token, body: input })
}

export async function rotateProviderChannelKey(token: string, id: string, apiKey: string): Promise<ProviderChannel> {
  return apiRequest<ProviderChannel>(`/model-provider-channels/${id}/rotate-key`, {
    method: 'POST',
    token,
    body: { api_key: apiKey },
  })
}

export async function testProviderChannel(token: string, id: string, model?: string): Promise<{ status: string }> {
  return apiRequest<{ status: string }>(`/model-provider-channels/${id}/test`, {
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

export async function approveToolApproval(token: string, id: string, reason = 'approved from human review console'): Promise<ToolApproval> {
  return apiRequest<ToolApproval>(`/tool-approvals/${id}/approve`, { method: 'POST', token, body: { reason } })
}

export async function rejectToolApproval(token: string, id: string, reason = 'rejected from human review console'): Promise<ToolApproval> {
  return apiRequest<ToolApproval>(`/tool-approvals/${id}/reject`, { method: 'POST', token, body: { reason } })
}

export async function listInvocations(token: string): Promise<AIInvocation[]> {
  return apiRequest<AIInvocation[]>('/ai-gateway/invocations', { token })
}

export async function getAIInvocation(token: string, id: string): Promise<AIInvocation> {
  return apiRequest<AIInvocation>(`/ai-gateway/invocations/${id}`, { token })
}

export async function createAssistantSession(token: string, input: CreateAssistantSessionInput): Promise<AssistantSession> {
  return apiRequest<AssistantSession>('/assistant/sessions', { method: 'POST', token, body: input })
}

export async function listAssistantSessions(token: string, moduleKey?: string): Promise<AssistantSession[]> {
  const query = moduleKey ? `?module_key=${encodeURIComponent(moduleKey)}` : ''
  return apiRequest<AssistantSession[]>(`/assistant/sessions${query}`, { token })
}

export async function listAssistantSteps(token: string, sessionID: string): Promise<AssistantStep[]> {
  return apiRequest<AssistantStep[]>(`/assistant/sessions/${sessionID}/steps`, { token })
}

export async function getAICostSummary(token: string): Promise<AICostSummary> {
  return apiRequest<AICostSummary>('/ai-gateway/cost-summary', { token })
}

export async function listRoutingRules(token: string): Promise<AIRoutingRule[]> {
  return apiRequest<AIRoutingRule[]>('/ai-gateway/routing-rules', { token })
}

export async function createRoutingRule(token: string, input: CreateAIRoutingRuleInput): Promise<AIRoutingRule> {
  return apiRequest<AIRoutingRule>('/ai-gateway/routing-rules', { method: 'POST', token, body: input })
}

export async function getAIUsageAnalysis(token: string): Promise<AIUsageAnalysis> {
  return apiRequest<AIUsageAnalysis>('/ai-gateway/usage-analysis', { token })
}

export async function estimateAICost(token: string, input: EstimateAICostInput): Promise<EstimateAICostOutput> {
  return apiRequest<EstimateAICostOutput>('/ai-gateway/estimate-cost', { method: 'POST', token, body: input })
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
  return apiRequest<FinanceExportBatch>('/finance/accounting-batches', { method: 'POST', token, body: input })
}

export async function listFinanceExportBatches(token: string): Promise<FinanceExportBatch[]> {
  return apiRequest<FinanceExportBatch[]>('/finance/accounting-batches', { token })
}

export async function getFinanceExportBatch(token: string, id: string): Promise<FinanceExportBatch> {
  return apiRequest<FinanceExportBatch>(`/finance/accounting-batches/${id}`, { token })
}

export async function submitFinanceExportBatch(token: string, id: string): Promise<FinanceExportBatch> {
  return apiRequest<FinanceExportBatch>(`/finance/accounting-batches/${id}/submit`, { method: 'POST', token })
}

export async function listFinanceReconciliation(token: string): Promise<FinanceReconciliationItem[]> {
  return apiRequest<FinanceReconciliationItem[]>('/finance/reconciliation', { token })
}

export async function importFinanceExpenses(token: string, input: ImportFinanceExpensesInput): Promise<FinanceImportResult> {
  return apiRequest<FinanceImportResult>('/finance/imports', { method: 'POST', token, body: input })
}

export async function importFinanceExpenseFile(token: string, adapterID: string, file: File): Promise<FinanceImportResult> {
  const form = new FormData()
  form.append('adapter_id', adapterID)
  form.append('file', file)
  return apiRequest<FinanceImportResult>('/finance/imports/files', { method: 'POST', token, body: form })
}

export async function pullFinanceExpenses(token: string, adapterID: string): Promise<FinanceImportResult> {
  return apiRequest<FinanceImportResult>(`/finance/imports/${adapterID}/pull`, { method: 'POST', token })
}

export async function listFinanceImportBatches(token: string): Promise<FinanceImportBatch[]> {
  return apiRequest<FinanceImportBatch[]>('/finance/import-batches', { token })
}

export async function listFinanceImportRecords(token: string): Promise<FinanceImportRecord[]> {
  return apiRequest<FinanceImportRecord[]>('/finance/import-records', { token })
}

export async function createFinanceSettlementOrder(token: string, input: CreateFinanceSettlementOrderInput): Promise<FinanceSettlementOrder> {
  return apiRequest<FinanceSettlementOrder>('/finance/settlement-orders', { method: 'POST', token, body: input })
}

export async function listFinanceSettlementOrders(token: string): Promise<FinanceSettlementOrder[]> {
  return apiRequest<FinanceSettlementOrder[]>('/finance/settlement-orders', { token })
}

export async function updateFinanceSettlementOrder(token: string, id: string, input: CreateFinanceSettlementOrderInput): Promise<FinanceSettlementOrder> {
  return apiRequest<FinanceSettlementOrder>(`/finance/settlement-orders/${id}`, { method: 'PATCH', token, body: input })
}

export async function postFinanceSettlementOrder(token: string, id: string): Promise<FinanceReceivable> {
  return apiRequest<FinanceReceivable>(`/finance/settlement-orders/${id}/post`, { method: 'POST', token })
}

export async function voidFinanceSettlementOrder(token: string, id: string, reason: string): Promise<FinanceSettlementOrder> {
  return apiRequest<FinanceSettlementOrder>(`/finance/settlement-orders/${id}/void`, { method: 'POST', token, body: { reason } })
}

export async function createFinanceReceivable(token: string, input: CreateFinanceReceivableInput): Promise<FinanceReceivable> {
  return apiRequest<FinanceReceivable>('/finance/receivables', { method: 'POST', token, body: input })
}

export async function listFinanceReceivables(token: string): Promise<FinanceReceivable[]> {
  return apiRequest<FinanceReceivable[]>('/finance/receivables', { token })
}

export async function updateFinanceReceivable(token: string, id: string, input: CreateFinanceReceivableInput): Promise<FinanceReceivable> {
  return apiRequest<FinanceReceivable>(`/finance/receivables/${id}`, { method: 'PATCH', token, body: input })
}

export async function voidFinanceReceivable(token: string, id: string, reason: string): Promise<FinanceReceivable> {
  return apiRequest<FinanceReceivable>(`/finance/receivables/${id}/void`, { method: 'POST', token, body: { reason } })
}

export async function createFinanceReceipt(token: string, input: CreateFinanceReceiptInput): Promise<FinanceReceipt> {
  return apiRequest<FinanceReceipt>('/finance/receipts', { method: 'POST', token, body: input })
}

export async function listFinanceReceipts(token: string): Promise<FinanceReceipt[]> {
  return apiRequest<FinanceReceipt[]>('/finance/receipts', { token })
}

export async function allocateFinanceReceipt(token: string, receiptID: string, input: AllocateFinanceReceiptInput): Promise<FinanceReceiptAllocation> {
  return apiRequest<FinanceReceiptAllocation>(`/finance/receipts/${receiptID}/allocate`, { method: 'POST', token, body: input })
}

export async function createFinancePayable(token: string, input: CreateFinancePayableInput): Promise<FinancePayable> {
  return apiRequest<FinancePayable>('/finance/payables', { method: 'POST', token, body: input })
}

export async function listFinancePayables(token: string): Promise<FinancePayable[]> {
  return apiRequest<FinancePayable[]>('/finance/payables', { token })
}

export async function updateFinancePayable(token: string, id: string, input: CreateFinancePayableInput): Promise<FinancePayable> {
  return apiRequest<FinancePayable>(`/finance/payables/${id}`, { method: 'PATCH', token, body: input })
}

export async function voidFinancePayable(token: string, id: string, reason: string): Promise<FinancePayable> {
  return apiRequest<FinancePayable>(`/finance/payables/${id}/void`, { method: 'POST', token, body: { reason } })
}

export async function createFinancePayment(token: string, input: CreateFinancePaymentInput): Promise<FinancePayment> {
  return apiRequest<FinancePayment>('/finance/payments', { method: 'POST', token, body: input })
}

export async function listFinancePayments(token: string): Promise<FinancePayment[]> {
  return apiRequest<FinancePayment[]>('/finance/payments', { token })
}

export async function updateFinancePayment(token: string, id: string, input: CreateFinancePaymentInput): Promise<FinancePayment> {
  return apiRequest<FinancePayment>(`/finance/payments/${id}`, { method: 'PATCH', token, body: input })
}

export async function voidFinancePayment(token: string, id: string, reason: string): Promise<FinancePayment> {
  return apiRequest<FinancePayment>(`/finance/payments/${id}/void`, { method: 'POST', token, body: { reason } })
}

export async function allocateFinancePayment(token: string, paymentID: string, input: AllocateFinancePaymentInput): Promise<FinancePaymentAllocation> {
  return apiRequest<FinancePaymentAllocation>(`/finance/payments/${paymentID}/allocate`, { method: 'POST', token, body: input })
}

export async function listCurrencies(token: string): Promise<Currency[]> {
  return apiRequest<Currency[]>('/costing/currencies', { token })
}

export async function upsertCurrency(token: string, input: CreateCurrencyInput): Promise<Currency> {
  return apiRequest<Currency>('/costing/currencies', { method: 'POST', token, body: input })
}

export async function voidCurrency(token: string, code: string): Promise<Currency> {
  return apiRequest<Currency>(`/costing/currencies/${encodeURIComponent(code)}/void`, { method: 'POST', token })
}

export async function listExchangeRates(token: string): Promise<ExchangeRateVersion[]> {
  return apiRequest<ExchangeRateVersion[]>('/costing/exchange-rates', { token })
}

export async function createExchangeRate(token: string, input: CreateExchangeRateInput): Promise<ExchangeRateVersion> {
  return apiRequest<ExchangeRateVersion>('/costing/exchange-rates', { method: 'POST', token, body: input })
}

export async function updateExchangeRate(token: string, id: string, input: CreateExchangeRateInput): Promise<ExchangeRateVersion> {
  return apiRequest<ExchangeRateVersion>(`/costing/exchange-rates/${id}`, { method: 'PATCH', token, body: input })
}

export async function voidExchangeRate(token: string, id: string): Promise<ExchangeRateVersion> {
  return apiRequest<ExchangeRateVersion>(`/costing/exchange-rates/${id}/void`, { method: 'POST', token })
}

export async function convertCostAmount(token: string, input: ConvertCostInput): Promise<ConversionResult> {
  return apiRequest<ConversionResult>('/costing/convert', { method: 'POST', token, body: input })
}

export async function listCostRateCards(token: string): Promise<CostRateCard[]> {
  return apiRequest<CostRateCard[]>('/costing/rate-cards', { token })
}

export async function createCostRateCard(token: string, input: CreateCostRateCardInput): Promise<CostRateCard> {
  return apiRequest<CostRateCard>('/costing/rate-cards', { method: 'POST', token, body: input })
}

export async function updateCostRateCard(token: string, id: string, input: CreateCostRateCardInput): Promise<CostRateCard> {
  return apiRequest<CostRateCard>(`/costing/rate-cards/${id}`, { method: 'PATCH', token, body: input })
}

export async function voidCostRateCard(token: string, id: string): Promise<CostRateCard> {
  return apiRequest<CostRateCard>(`/costing/rate-cards/${id}/void`, { method: 'POST', token })
}

export async function listCostBudgets(token: string): Promise<CostBudget[]> {
  return apiRequest<CostBudget[]>('/costing/budgets', { token })
}

export async function createCostBudget(token: string, input: CreateCostBudgetInput): Promise<CostBudget> {
  return apiRequest<CostBudget>('/costing/budgets', { method: 'POST', token, body: input })
}

export async function updateCostBudget(token: string, id: string, input: CreateCostBudgetInput): Promise<CostBudget> {
  return apiRequest<CostBudget>(`/costing/budgets/${id}`, { method: 'PATCH', token, body: input })
}

export async function voidCostBudget(token: string, id: string): Promise<CostBudget> {
  return apiRequest<CostBudget>(`/costing/budgets/${id}/void`, { method: 'POST', token })
}

export async function listCostLedgerEntries(token: string): Promise<CostLedgerEntry[]> {
  return apiRequest<CostLedgerEntry[]>('/costing/ledger-entries', { token })
}

export async function createCostLedgerEntry(token: string, input: CreateCostLedgerEntryInput): Promise<CostLedgerEntry> {
  return apiRequest<CostLedgerEntry>('/costing/ledger-entries', { method: 'POST', token, body: input })
}

export async function updateCostLedgerEntry(token: string, id: string, input: CreateCostLedgerEntryInput): Promise<CostLedgerEntry> {
  return apiRequest<CostLedgerEntry>(`/costing/ledger-entries/${id}`, { method: 'PATCH', token, body: input })
}

export async function voidCostLedgerEntry(token: string, id: string): Promise<CostLedgerEntry> {
  return apiRequest<CostLedgerEntry>(`/costing/ledger-entries/${id}/void`, { method: 'POST', token })
}

export async function getCostSummary(token: string): Promise<CostSummary> {
  return apiRequest<CostSummary>('/costing/summary', { token })
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

export interface DataTable {
  table_name: string
  master_table_name: string
  detail_table_name: string
  key_prefix: string
  display_name: string
  category: string
  is_base_data: boolean
  is_business_scenario: boolean
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface DataField {
  table_name: string
  field_name: string
  data_type: string
  display_name: string
  is_master_key: boolean
  is_sub_key: boolean
  is_visible_default: boolean
  permission_level: string
  display_order: number
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface UserFieldPreference {
  actor_id: string
  table_name: string
  visible_fields: string[]
  field_order: string[]
  field_widths: Record<string, number>
  created_at?: string
  updated_at?: string
}

export interface SaveUserFieldPreferenceInput {
  visible_fields: string[]
  field_order: string[]
  field_widths: Record<string, number>
}

export interface FieldPermissionRule {
  id: string
  table_name: string
  field_name: string
  actor_type: string
  actor_id?: string
  role_id?: string
  action: 'read' | 'write' | 'delete' | 'admin'
  behavior: 'allow' | 'notify' | 'approve' | 'deny'
  required_level: string
  reason: string
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CreateFieldPermissionRuleInput {
  table_name: string
  field_name?: string
  actor_type?: string
  actor_id?: string
  role_id?: string
  action: 'read' | 'write' | 'delete' | 'admin'
  behavior?: 'allow' | 'notify' | 'approve' | 'deny'
  required_level?: string
  reason?: string
  metadata?: Record<string, unknown>
}

export interface FieldAccessCheckInput {
  actor_id?: string
  actor_type?: string
  table_name: string
  field_name?: string
  action: 'read' | 'write' | 'delete' | 'admin'
}

export interface FieldAccessCheckResult {
  allowed: boolean
  behavior: string
  required_level: string
  reason: string
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

export interface MetaResource {
  id: string
  resource_type: string
  source_type?: string
  source_id?: string
  name: string
  status: string
  organization_id?: string
  department_id?: string
  owner_actor_id?: string
  owner_actor_type?: string
  capability_profile: Record<string, unknown>
  cost_profile: Record<string, unknown>
  capacity_profile: Record<string, unknown>
  risk_profile: Record<string, unknown>
  performance_profile: Record<string, unknown>
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CreateMetaResourceInput {
  resource_type: string
  source_type?: string
  source_id?: string
  name: string
  status?: string
  organization_id?: string
  department_id?: string
  owner_actor_id?: string
  owner_actor_type?: string
  capability_profile?: Record<string, unknown>
  cost_profile?: Record<string, unknown>
  capacity_profile?: Record<string, unknown>
  risk_profile?: Record<string, unknown>
  performance_profile?: Record<string, unknown>
  metadata?: Record<string, unknown>
}

export interface MetaResourceSummary {
  total: number
  active: number
  by_type: Record<string, number>
  by_status: Record<string, number>
  average_cost_score: number
  average_fit_score: number
  recent: MetaResource[]
  metadata?: Record<string, unknown>
}

export interface DemandProfile {
  id: string
  requirement_id?: string
  project_id?: string
  title: string
  goal: string
  status: string
  acceptance_criteria: unknown[]
  required_capabilities: unknown[]
  budget_constraints: Record<string, unknown>
  time_constraints: Record<string, unknown>
  risk_constraints: Record<string, unknown>
  resource_fit_candidates: unknown[]
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CreateDemandProfileInput {
  requirement_id?: string
  project_id?: string
  title: string
  goal?: string
  status?: string
  acceptance_criteria?: unknown[]
  required_capabilities?: unknown[]
  budget_constraints?: Record<string, unknown>
  time_constraints?: Record<string, unknown>
  risk_constraints?: Record<string, unknown>
  resource_fit_candidates?: unknown[]
  metadata?: Record<string, unknown>
}

export interface PDCACycle {
  id: string
  demand_profile_id?: string
  requirement_id?: string
  project_id?: string
  status: string
  current_stage: string
  outcome_score: number
  summary: string
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
  completed_at?: string
}

export interface CreatePDCACycleInput {
  demand_profile_id?: string
  requirement_id?: string
  project_id?: string
  status?: string
  current_stage?: string
  summary?: string
  metadata?: Record<string, unknown>
}

export interface PDCAEvent {
  id: string
  cycle_id: string
  stage: string
  event_type: string
  source_type?: string
  source_id?: string
  actor_id?: string
  actor_type?: string
  resource_refs: unknown[]
  cost_refs: unknown[]
  evidence: Record<string, unknown>
  decision: string
  next_action: string
  metadata: Record<string, unknown>
  created_at: string
}

export interface CreatePDCAEventInput {
  cycle_id: string
  stage: string
  event_type?: string
  source_type?: string
  source_id?: string
  actor_id?: string
  actor_type?: string
  resource_refs?: unknown[]
  cost_refs?: unknown[]
  evidence?: Record<string, unknown>
  decision?: string
  next_action?: string
  metadata?: Record<string, unknown>
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

export interface ProviderChannel {
  id: string
  provider_id: string
  name: string
  base_url: string
  masked_api_key: string
  owner_type?: string
  user_id?: string
  agent_id?: string
  status: string
  priority: number
  concurrency_limit: number
  inflight_requests: number
  load_factor: number
  rate_multiplier: number
  quota_amount: number
  quota_used: number
  quota_currency: string
  supported_model_patterns: string[]
  model_mapping: Record<string, string>
  health_status: string
  last_error?: string
  last_tested_at?: string
  last_used_at?: string
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CreateProviderChannelInput {
  provider_id?: string
  name: string
  base_url?: string
  api_key: string
  owner_type?: string
  user_id?: string
  agent_id?: string
  status?: string
  priority?: number
  concurrency_limit?: number
  load_factor?: number
  rate_multiplier?: number
  quota_amount?: number
  quota_currency?: string
  supported_model_patterns?: string[]
  model_mapping?: Record<string, string>
  metadata?: Record<string, unknown>
}

export type UpdateProviderChannelInput = Partial<Omit<CreateProviderChannelInput, 'api_key' | 'provider_id'>>

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
  cache_creation_price_per_1k?: number
  cache_read_price_per_1k?: number
  cache_creation_5m_price_per_1k?: number
  cache_creation_1h_price_per_1k?: number
  image_output_price_per_1k?: number
  priority_input_price_per_1k?: number
  priority_output_price_per_1k?: number
  priority_cache_read_price_per_1k?: number
  long_context_threshold?: number
  long_context_input_multiplier?: number
  long_context_output_multiplier?: number
  billing_mode?: string
  pricing_source?: string
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
  tool_category: string
  approval_tier_required: string
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
  requested_by_human_id?: string
  policy: string
  status: string
  result_summary?: string
  error_message?: string
  created_at: string
  completed_at?: string
}

export interface ToolApproval {
  id: string
  execution_id: string
  status: string
  requested_by?: string
  reviewed_by?: string
  approved_by_human_id?: string
  reason?: string
  expires_at?: string
  created_at: string
  reviewed_at?: string
}

export interface AIInvocation {
  id: string
  provider_id: string
  model_id: string
  channel_id?: string
  mode: string
  status: string
  attribution?: Record<string, unknown>
  requested_model?: string
  upstream_model?: string
  model_mapping_chain?: string
  service_tier?: string
  reasoning_effort?: string
  request_hash?: string
  provider_request_id?: string
  cost_amount: number
  cost_breakdown?: CostBreakdown
  currency: string
  input_tokens: number
  output_tokens: number
  cache_creation_tokens: number
  cache_read_tokens: number
  cache_creation_5m_tokens: number
  cache_creation_1h_tokens: number
  image_output_tokens: number
  estimated_input_tokens: number
  estimated_output_tokens: number
  duration_ms: number
  error_message?: string
  metadata?: Record<string, unknown>
  created_at: string
  completed_at?: string
}

export interface AssistantSession {
  id: string
  title: string
  mode: 'business_process' | 'self_evolution'
  module_key: string
  status: string
  provider_type?: string
  model?: string
  service_tier?: string
  reasoning_effort?: string
  organization_id?: string
  department_id?: string
  position_id?: string
  position_assignment_id?: string
  project_id?: string
  workflow_id?: string
  task_id?: string
  working_memory: Record<string, unknown>
  metadata: Record<string, unknown>
  last_error?: string
  created_at: string
  updated_at: string
}

export interface CreateAssistantSessionInput {
  title?: string
  mode?: 'business_process' | 'self_evolution'
  module_key: string
  provider_id?: string
  preferred_channel_id?: string
  provider_type?: 'openai' | 'anthropic' | 'gemini'
  model?: string
  service_tier?: string
  reasoning_effort?: string
  organization_id?: string
  department_id?: string
  position_id?: string
  position_assignment_id?: string
  project_id?: string
  workflow_id?: string
  task_id?: string
  metadata?: Record<string, unknown>
}

export interface AssistantStep {
  id: string
  session_id: string
  module_key: string
  organization_id?: string
  department_id?: string
  position_id?: string
  position_assignment_id?: string
  invocation_id?: string
  tool_execution_id?: string
  tool_approval_id?: string
  step_type: string
  status: string
  summary: string
  data: Record<string, unknown>
  turn: number
  created_at: string
}

export interface AICostSummary {
  total: number
  unexported: number
  currency: string
  by_provider: Record<string, number>
  by_channel?: Record<string, number>
}

export interface TokenUsage {
  input_tokens: number
  output_tokens: number
  cache_creation_tokens?: number
  cache_read_tokens?: number
  cache_creation_5m_tokens?: number
  cache_creation_1h_tokens?: number
  image_output_tokens?: number
}

export interface CostBreakdown {
  input_cost: number
  output_cost: number
  cache_creation_cost: number
  cache_read_cost: number
  image_output_cost: number
  total_cost: number
  actual_cost: number
  rate_multiplier: number
  billing_mode: string
  service_tier?: string
}

export interface AIRoutingRule {
  id: string
  name: string
  provider_id?: string
  channel_id?: string
  match_scope: string
  match_value: string
  model_pattern: string
  priority: number
  status: string
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CreateAIRoutingRuleInput {
  name: string
  provider_id?: string
  channel_id?: string
  match_scope?: string
  match_value?: string
  model_pattern?: string
  priority?: number
  status?: string
  metadata?: Record<string, unknown>
}

export interface AIUsageAnalysis {
  currency: string
  total_cost: number
  invocation_count: number
  by_provider: Record<string, number>
  by_channel: Record<string, number>
  by_model: Record<string, number>
  by_actor: Record<string, number>
  recent: AIInvocation[]
}

export interface EstimateAICostInput {
  model: string
  provider_id?: string
  provider_type?: string
  usage: TokenUsage
  service_tier?: string
  rate_multiplier?: number
}

export interface EstimateAICostOutput {
  model: string
  cost_breakdown: CostBreakdown
  currency: string
}

export interface FinanceAdapter {
  id: string
  name: string
  endpoint_url: string
  auth_type: 'hmac' | 'bearer'
  adapter_type: string
  direction: string
  masked_secret: string
  status: string
  timeout_ms: number
  retry_count: number
  field_mapping: Record<string, unknown>
  pull_config: Record<string, unknown>
  last_sync_at?: string
  last_sync_status: string
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CreateFinanceAdapterInput {
  name: string
  endpoint_url: string
  auth_type: 'hmac' | 'bearer'
  adapter_type?: string
  direction?: string
  secret: string
  timeout_ms?: number
  retry_count?: number
  field_mapping?: Record<string, unknown>
  pull_config?: Record<string, unknown>
  metadata?: Record<string, unknown>
}

export interface FinanceExportLine {
  id: string
  batch_id: string
  usage_ledger_id?: string
  cost_ledger_entry_id?: string
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

export interface ImportFinanceExpensesInput {
  adapter_id: string
  source_type?: string
  file_name?: string
  records: Array<Record<string, unknown>>
  metadata?: Record<string, unknown>
}

export interface FinanceImportBatch {
  id: string
  adapter_id?: string
  source_type: string
  file_name: string
  status: string
  total_records: number
  processed_records: number
  failed_records: number
  metadata: Record<string, unknown>
  created_at: string
  completed_at?: string
}

export interface FinanceImportRecord {
  id: string
  batch_id: string
  adapter_id?: string
  external_record_id: string
  expense_type: string
  raw_payload: Record<string, unknown>
  normalized_payload: Record<string, unknown>
  cost_ledger_entry_id?: string
  payable_id?: string
  status: string
  error_message: string
  metadata: Record<string, unknown>
  created_at: string
}

export interface FinanceImportResult {
  batch: FinanceImportBatch
  records: FinanceImportRecord[]
}

export interface FinanceSettlementLine {
  id: string
  settlement_order_id: string
  line_type: string
  source_type: string
  source_id?: string
  deliverable_id?: string
  description: string
  quantity: number
  unit_price: number
  amount: number
  tax_amount: number
  total_amount: number
  metadata: Record<string, unknown>
  created_at: string
}

export interface FinanceSettlementOrder {
  id: string
  settlement_number: string
  project_id?: string
  requirement_id?: string
  deliverable_id?: string
  customer_id: string
  customer_name: string
  title: string
  description: string
  subtotal: number
  tax_amount: number
  total_amount: number
  currency: string
  settlement_date?: string
  due_date?: string
  status: string
  receivable_id?: string
  metadata: Record<string, unknown>
  lines?: FinanceSettlementLine[]
  created_at: string
  updated_at: string
}

export interface CreateFinanceSettlementOrderInput {
  settlement_number?: string
  project_id?: string
  requirement_id?: string
  deliverable_id?: string
  customer_id?: string
  customer_name?: string
  title?: string
  description?: string
  currency?: string
  settlement_date?: string
  due_date?: string
  status?: string
  metadata?: Record<string, unknown>
  lines?: Array<{
    line_type?: string
    source_type?: string
    source_id?: string
    deliverable_id?: string
    description?: string
    quantity?: number
    unit_price?: number
    amount: number
    tax_amount?: number
    metadata?: Record<string, unknown>
  }>
}

export interface FinanceReceivable {
  id: string
  receivable_type: string
  settlement_order_id?: string
  source_type: string
  source_id?: string
  external_receivable_id: string
  invoice_number: string
  customer_id: string
  customer_name: string
  project_id?: string
  requirement_id?: string
  organization_id?: string
  department_id?: string
  account_code: string
  account_name: string
  amount: number
  tax_amount: number
  currency: string
  period_start?: string
  period_end?: string
  invoice_date?: string
  due_date?: string
  status: string
  received_amount: number
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CreateFinanceReceivableInput {
  receivable_type?: string
  settlement_order_id?: string
  source_type?: string
  source_id?: string
  external_receivable_id?: string
  invoice_number?: string
  customer_id?: string
  customer_name?: string
  project_id?: string
  requirement_id?: string
  amount: number
  tax_amount?: number
  currency?: string
  invoice_date?: string
  due_date?: string
  status?: string
  metadata?: Record<string, unknown>
}

export interface FinanceReceipt {
  id: string
  receipt_number: string
  external_receipt_id: string
  payment_method: string
  payer_account: string
  receiver_account: string
  customer_id: string
  customer_name: string
  amount: number
  currency: string
  received_at?: string
  status: string
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CreateFinanceReceiptInput {
  receipt_number?: string
  external_receipt_id?: string
  payment_method?: string
  payer_account?: string
  receiver_account?: string
  customer_id?: string
  customer_name?: string
  amount: number
  currency?: string
  received_at?: string
  status?: string
  metadata?: Record<string, unknown>
}

export interface AllocateFinanceReceiptInput {
  receivable_id: string
  amount: number
  currency?: string
  metadata?: Record<string, unknown>
}

export interface FinanceReceiptAllocation {
  id: string
  receipt_id: string
  receivable_id: string
  amount: number
  currency: string
  metadata: Record<string, unknown>
  created_at: string
}

export interface FinancePayable {
  id: string
  payable_type: string
  source_type: string
  external_payable_id: string
  invoice_number: string
  vendor_id: string
  vendor_name: string
  employee_id: string
  employee_name: string
  project_id?: string
  account_code: string
  account_name: string
  cost_center_code: string
  cost_center_name: string
  amount: number
  tax_amount: number
  currency: string
  period_start?: string
  period_end?: string
  invoice_date?: string
  due_date?: string
  status: string
  paid_amount: number
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CreateFinancePayableInput {
  payable_type?: string
  external_payable_id?: string
  invoice_number?: string
  vendor_id?: string
  vendor_name?: string
  employee_id?: string
  employee_name?: string
  project_id?: string
  account_code?: string
  account_name?: string
  cost_center_code?: string
  cost_center_name?: string
  amount: number
  tax_amount?: number
  currency?: string
  invoice_date?: string
  due_date?: string
  status?: string
  metadata?: Record<string, unknown>
}

export interface FinancePayment {
  id: string
  payment_number: string
  external_payment_id: string
  payment_method: string
  payer_account: string
  payee_account: string
  vendor_id: string
  vendor_name: string
  employee_id: string
  employee_name: string
  amount: number
  currency: string
  paid_at?: string
  status: string
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CreateFinancePaymentInput {
  payment_number?: string
  external_payment_id?: string
  payment_method?: string
  payer_account?: string
  payee_account?: string
  vendor_id?: string
  vendor_name?: string
  employee_id?: string
  employee_name?: string
  amount: number
  currency?: string
  paid_at?: string
  status?: string
  metadata?: Record<string, unknown>
}

export interface AllocateFinancePaymentInput {
  payable_id: string
  amount: number
  currency?: string
  metadata?: Record<string, unknown>
}

export interface FinancePaymentAllocation {
  id: string
  payment_id: string
  payable_id: string
  amount: number
  currency: string
  metadata: Record<string, unknown>
  created_at: string
}

export interface Currency {
  code: string
  name: string
  currency_type: string
  symbol: string
  precision_digits: number
  chain_id?: string
  contract_address?: string
  external_source?: string
  is_active: boolean
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CreateCurrencyInput {
  code: string
  name?: string
  currency_type?: string
  symbol?: string
  precision_digits?: number
  chain_id?: string
  contract_address?: string
  external_source?: string
  is_active?: boolean
  metadata?: Record<string, unknown>
}

export interface ExchangeRateVersion {
  id: string
  from_currency: string
  to_currency: string
  rate: number
  source: string
  provider?: string
  external_rate_id?: string
  effective_from: string
  effective_to?: string
  metadata: Record<string, unknown>
  created_at: string
}

export interface CreateExchangeRateInput {
  from_currency: string
  to_currency: string
  rate: number
  source?: string
  provider?: string
  external_rate_id?: string
  effective_from?: string
  effective_to?: string
  metadata?: Record<string, unknown>
}

export interface ConvertCostInput {
  amount: number
  from_currency: string
  to_currency: string
  at?: string
}

export interface ConversionResult {
  amount: number
  from_currency: string
  to_currency: string
  converted_amount: number
  rate: number
  exchange_rate_version_id?: string
}

export interface CostRateCard {
  id: string
  subject_type: string
  subject_id?: string
  scope_type?: string
  scope_id?: string
  rate_type: string
  amount: number
  currency: string
  base_amount: number
  base_currency: string
  exchange_rate_version_id?: string
  effective_from: string
  effective_to?: string
  status: string
  metadata: Record<string, unknown>
  created_at: string
}

export interface CreateCostRateCardInput {
  subject_type: string
  subject_id?: string
  scope_type?: string
  scope_id?: string
  rate_type?: string
  amount: number
  currency?: string
  effective_from?: string
  effective_to?: string
  status?: string
  metadata?: Record<string, unknown>
}

export interface CostBudget {
  id: string
  scope_type: string
  scope_id?: string
  amount: number
  currency: string
  base_amount: number
  base_currency: string
  exchange_rate_version_id?: string
  period_start?: string
  period_end?: string
  status: string
  metadata: Record<string, unknown>
  created_at: string
  updated_at: string
}

export interface CreateCostBudgetInput {
  scope_type: string
  scope_id?: string
  amount: number
  currency?: string
  period_start?: string
  period_end?: string
  status?: string
  metadata?: Record<string, unknown>
}

export interface CostLedgerEntry {
  id: string
  ledger_type: string
  cost_category: string
  source_type: string
  source_id?: string
  organization_id?: string
  department_id?: string
  requirement_id?: string
  project_id?: string
  workflow_id?: string
  task_id?: string
  capability_id?: string
  actor_id?: string
  actor_type?: string
  resource_type?: string
  amount: number
  currency: string
  base_amount: number
  base_currency: string
  exchange_rate_version_id?: string
  occurred_at: string
  status: string
  finance_export_line_id?: string
  description: string
  metadata: Record<string, unknown>
  created_at: string
}

export interface CreateCostLedgerEntryInput {
  ledger_type?: string
  cost_category?: string
  source_type?: string
  source_id?: string
  organization_id?: string
  department_id?: string
  requirement_id?: string
  project_id?: string
  workflow_id?: string
  task_id?: string
  capability_id?: string
  actor_id?: string
  actor_type?: string
  resource_type?: string
  amount: number
  currency?: string
  occurred_at?: string
  status?: string
  description?: string
  metadata?: Record<string, unknown>
}

export interface CostSummary {
  scope_type?: string
  scope_id?: string
  currency: string
  total_amount: number
  budget_amount: number
  budget_variance: number
  entry_count: number
  by_category: Record<string, number>
  by_source: Record<string, number>
  by_currency: Record<string, number>
  recent_entries: CostLedgerEntry[]
  metadata?: Record<string, unknown>
}
