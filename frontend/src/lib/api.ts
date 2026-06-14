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

export interface RecentEvent {
  id: string
  type: string
  title: string
  status?: string
  created_at: string
}
