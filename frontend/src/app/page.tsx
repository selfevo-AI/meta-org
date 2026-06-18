'use client'

import {
  Activity,
  ArrowDown,
  ArrowUp,
  Bot,
  Boxes,
  BrainCircuit,
  BriefcaseBusiness,
  ChevronDown,
  ChevronRight,
  CheckCircle2,
  CircleDollarSign,
  Code2,
  FolderKanban,
  Gauge,
  GitBranch,
  Github,
  Home as HomeIcon,
  KeyRound,
  LogOut,
  Menu,
  Moon,
  MoreHorizontal,
  RefreshCw,
  ShieldCheck,
  SlidersHorizontal,
  Sparkles,
  Sun,
  Users,
  WalletCards,
  Workflow,
  X,
} from 'lucide-react'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import type { DragEvent, FormEvent } from 'react'
import {
  activateAssistantSkill,
  approveToolApproval,
  confirmAssistantProposal,
  createAssistantSkill,
  getMetaOrgInbox,
  getMetaOrgOverview,
  listAssistantContextTargets,
  listAssistantProposals,
  listAssistantSkills,
  listRoles,
  login,
  registerUser,
  rejectAssistantProposal,
  rejectToolApproval,
  runAssistantSkill,
} from '@/lib/api'
import type { AssistantBusinessSkill, AssistantContextTarget, AssistantProposal, InboxItem, MetaOrgOverview, Role } from '@/lib/api'
import { clearSession, getSessionUser, getToken, setSession } from '@/lib/auth'
import { useI18n } from '@/lib/i18n'
import { apiOperations, getOperationProfile, operationDomains } from '@/lib/operations'
import type { ApiOperation } from '@/lib/operations'
import { AIAssistant } from './ai-assistant'
import {
  CapabilityEvaluationWorkspace,
  GovernanceWorkspace,
  WeightWorkspace,
  WorkflowDesignerWorkspace,
  WorkflowMatchingWorkspace,
} from './control-workspaces'
import { CostingWorkspace } from './costing-workspace'
import { DeveloperToolsWorkspace } from './developer-tools-workspace'
import { FinanceWorkspace } from './finance-workspace'
import { MetaResourceWorkspace } from './meta-resource-workspace'
import { OrganizationWorkspace } from './organization-workspace'
import { ProjectLifecycleWorkspace } from './project-lifecycle-workspace'

type AuthMode = 'login' | 'register'
type WorkspaceView = 'overview' | `domain:${string}`
type ThemeMode = 'dark' | 'light'

const domainLabels: Record<string, string> = {
  MetaOrg: 'Meta-Org',
  Dashboard: '系统',
  Identity: '身份',
  Organization: '组织',
  Layer: '分层',
  Capability: '能力',
  Workflow: '工作流',
  Observability: '可观测',
  Verification: '验证',
  Governance: '治理',
  Evolution: '自进化',
  Requirement: '需求',
  Project: '项目',
  Delivery: '交付',
  Cost: '成本',
  Feedback: '反馈评估',
  DeveloperTools: '模型设置',
  Finance: '财务核算',
  Costing: '成本核算',
  FinanceAccounting: '财务核算',
  FinanceReceivables: '应收',
  FinancePayables: '应付',
  FinanceCostAccounting: '成本核算',
  MetaResource: 'Meta 资源',
}

const domainIcons: Record<string, typeof Gauge> = {
  MetaOrg: Sparkles,
  Dashboard: Gauge,
  Identity: KeyRound,
  Organization: Users,
  Layer: Boxes,
  Capability: BrainCircuit,
  Workflow,
  Observability: Activity,
  Verification: CheckCircle2,
  Governance: ShieldCheck,
  Evolution: GitBranch,
  Requirement: BriefcaseBusiness,
  Project: FolderKanban,
  Delivery: ArrowUp,
  Cost: CircleDollarSign,
  Feedback: Activity,
  DeveloperTools: Code2,
  Finance: WalletCards,
  Costing: CircleDollarSign,
  FinanceAccounting: WalletCards,
  FinanceReceivables: ArrowDown,
  FinancePayables: ArrowUp,
  FinanceCostAccounting: CircleDollarSign,
  MetaResource: Boxes,
}

type MenuGroup = {
  id: string
  label: string
  domains: string[]
}

const lifecycleDomains = ['Requirement', 'Project', 'Delivery', 'Cost', 'Feedback']
const virtualDomains = ['Costing', 'MetaResource']
const dedicatedDomains = new Set([
  'MetaResource',
  'Organization',
  'Governance',
  'Evolution',
  'Capability',
  'Workflow',
  'DeveloperTools',
  'Finance',
  'Costing',
  'FinanceAccounting',
  'FinanceReceivables',
  'FinancePayables',
  'FinanceCostAccounting',
  ...lifecycleDomains,
])
const menuStorageKey = 'meta_org.menu.groups.v2'
const expandedMenuStorageKey = 'meta_org.menu.expanded.v2'
const themeStorageKey = 'meta_org.theme.v1'
const legacyMenuStorageKey = 'harness.menu.groups.v1'
const legacyExpandedMenuStorageKey = 'harness.menu.expanded.v1'
const projectGithubURL = 'https://github.com/selfevo-AI/meta-org'

const defaultMenuGroups: MenuGroup[] = [
  {
    id: 'business',
    label: '业务闭环',
    domains: ['Requirement', 'Project', 'Delivery', 'Cost', 'Feedback'],
  },
  {
    id: 'baseData',
    label: '基础数据',
    domains: [
      'MetaResource',
      'Organization',
      'Workflow',
      'Capability',
      'Governance',
      'Evolution',
      'Verification',
      'MetaOrg',
      'Dashboard',
      'DeveloperTools',
      'Identity',
      'Layer',
      'Observability',
    ],
  },
  {
    id: 'finance',
    label: '财务',
    domains: ['FinanceAccounting', 'FinanceReceivables', 'FinancePayables', 'FinanceCostAccounting'],
  },
]

const numberFormatter = new Intl.NumberFormat('zh-CN')
const compactFormatter = new Intl.NumberFormat('zh-CN', { notation: 'compact' })
const percentFormatter = new Intl.NumberFormat('zh-CN', {
  maximumFractionDigits: 1,
  minimumFractionDigits: 0,
})

function formatNumber(value: number): string {
  return numberFormatter.format(value)
}

function formatCompact(value: number): string {
  return compactFormatter.format(value)
}

function formatPercent(value: number): string {
  return `${percentFormatter.format(value * 100)}%`
}

function formatMoney(value: number, currency = 'CNY'): string {
  return `${currency} ${Number(value || 0).toFixed(2)}`
}

function formatDate(value: string): string {
  return new Intl.DateTimeFormat('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}

function normalizeMenuGroups(input?: MenuGroup[]): MenuGroup[] {
  const allMenuDomains = [...operationDomains, ...virtualDomains]
  const knownDomains = new Set(allMenuDomains)
  const defaultByID = new Map(defaultMenuGroups.map((group) => [group.id, group]))
  const defaultTargetByDomain = new Map(
    defaultMenuGroups.flatMap((group) => group.domains.map((domain) => [domain, group.id] as const)),
  )
  const sourceGroups = Array.isArray(input) && input.length > 0 ? input : defaultMenuGroups
  const nextGroups = defaultMenuGroups.map((group) => ({
    ...group,
    label: defaultByID.get(group.id)?.label ?? group.label,
    domains: [] as string[],
  }))
  const groupByID = new Map(nextGroups.map((group) => [group.id, group]))
  const assigned = new Set<string>()

  sourceGroups.forEach((sourceGroup) => {
    const target = groupByID.get(sourceGroup.id)
    if (!target) return
    sourceGroup.domains.forEach((domain) => {
      if (!knownDomains.has(domain) || assigned.has(domain)) return
      target.domains.push(domain)
      assigned.add(domain)
    })
  })

  allMenuDomains.forEach((domain) => {
    if (assigned.has(domain)) return
    const targetID = defaultTargetByDomain.get(domain) ?? 'system'
    const target = groupByID.get(targetID) ?? nextGroups[nextGroups.length - 1]
    target.domains.push(domain)
    assigned.add(domain)
  })

  return nextGroups
}

function defaultExpandedGroups(): Record<string, boolean> {
  return Object.fromEntries(defaultMenuGroups.map((group) => [group.id, true]))
}

function loadMenuGroups(): MenuGroup[] {
  if (typeof window === 'undefined') return normalizeMenuGroups()
  try {
    const legacyRaw = window.localStorage.getItem(legacyMenuStorageKey)
    const raw = window.localStorage.getItem(menuStorageKey) || legacyRaw
    if (legacyRaw && raw) {
      window.localStorage.setItem(menuStorageKey, raw)
      window.localStorage.removeItem(legacyMenuStorageKey)
    }
    if (!raw) return normalizeMenuGroups()
    return normalizeMenuGroups(JSON.parse(raw))
  } catch {
    return normalizeMenuGroups()
  }
}

function loadExpandedGroups(): Record<string, boolean> {
  if (typeof window === 'undefined') return defaultExpandedGroups()
  try {
    const legacyRaw = window.localStorage.getItem(legacyExpandedMenuStorageKey)
    const raw = window.localStorage.getItem(expandedMenuStorageKey) || legacyRaw
    if (legacyRaw && raw) {
      window.localStorage.setItem(expandedMenuStorageKey, raw)
      window.localStorage.removeItem(legacyExpandedMenuStorageKey)
    }
    if (!raw) return defaultExpandedGroups()
    return { ...defaultExpandedGroups(), ...JSON.parse(raw) }
  } catch {
    return defaultExpandedGroups()
  }
}

function loadThemeMode(): ThemeMode {
  if (typeof window === 'undefined') return 'dark'
  return window.localStorage.getItem(themeStorageKey) === 'light' ? 'light' : 'dark'
}

function assistantModuleForDomain(domain: string): string {
  const modules: Record<string, string> = {
    Overview: 'meta_org',
    MetaOrg: 'meta_org',
    Dashboard: 'meta_org',
    MetaResource: 'meta_resource',
    Requirement: 'requirement',
    Project: 'project',
    Delivery: 'delivery',
    Cost: 'project_cost',
    Feedback: 'feedback',
    Organization: 'organization',
    Workflow: 'workflow',
    Capability: 'capability',
    Governance: 'governance',
    Evolution: 'self_evolution',
    Verification: 'verification',
    DeveloperTools: 'model_settings',
    Costing: 'costing',
    Finance: 'finance',
    FinanceAccounting: 'finance',
    FinanceReceivables: 'finance',
    FinancePayables: 'finance',
    FinanceCostAccounting: 'costing',
  }
  return modules[domain] ?? domain.toLowerCase()
}

const globalAssistantModules = [
  { id: 'meta_org', key: 'meta_org', targetType: '', label: 'MetaOrg' },
  { id: 'requirement', key: 'requirement', targetType: 'requirement', label: 'Requirement' },
  { id: 'project', key: 'project', targetType: 'project', label: 'Project' },
  { id: 'delivery', key: 'delivery', targetType: 'deliverable', label: 'Delivery' },
  { id: 'project_cost', key: 'project_cost', targetType: 'project_cost', label: 'Cost' },
  { id: 'feedback', key: 'feedback', targetType: 'project_evaluation', label: 'Feedback' },
  { id: 'workflow', key: 'workflow', targetType: 'workflow_instance', label: 'Workflow' },
  { id: 'finance_accounting', key: 'finance', targetType: 'finance_settlement', label: 'FinanceAccounting' },
  { id: 'finance_receivables', key: 'finance', targetType: 'finance_receivable', label: 'FinanceReceivables' },
  { id: 'finance_payables', key: 'finance', targetType: 'finance_payable', label: 'FinancePayables' },
  { id: 'finance_costing', key: 'costing', targetType: 'cost_ledger_entry', label: 'FinanceCostAccounting' },
]

function agentIntentForOperation(operation: ApiOperation, context: Record<string, string>): string {
  const contextLines = Object.entries(context)
    .filter(([, value]) => value)
    .map(([key, value]) => `${key}: ${value}`)
    .join('\n')
  const profile = getOperationProfile(operation)

  return [
    `请作为当前功能工作台的执行 Agent 处理这个操作：${operation.title}`,
    `操作 ID: ${operation.id}`,
    `业务域: ${operation.domain}`,
    `目标接口语义: ${operation.method} ${operation.path}`,
    `风险级别: ${profile.dangerLevel}`,
    '执行要求: 这是人类在当前工作台主动调用 Agent 的操作。请先读取当前工作记录和历史数据，必要时调用工具执行；不要要求人类手动调用 API。涉及写入、财务、治理、模型配置或高风险动作时进入审批。',
    contextLines ? `当前已选业务上下文:\n${contextLines}` : '当前没有选中记录，请先根据数据库工作记录识别可操作对象，无法确定时向人类提出审核问题。',
  ].join('\n')
}

export default function Home() {
  const { locale, setLocale, t } = useI18n()
  const [ready, setReady] = useState(false)
  const [mode, setMode] = useState<AuthMode>('login')
  const [email, setEmail] = useState('')
  const [name, setName] = useState('')
  const [password, setPassword] = useState('')
  const [token, setToken] = useState<string | null>(null)
  const [userId, setUserId] = useState<string | null>(null)
  const [userType, setUserType] = useState<string | null>(null)
  const [overview, setOverview] = useState<MetaOrgOverview | null>(null)
  const [inbox, setInbox] = useState<InboxItem[]>([])
  const [roles, setRoles] = useState<Role[]>([])
  const [loading, setLoading] = useState(false)
  const [overviewLoading, setOverviewLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [workspaceView, setWorkspaceView] = useState<WorkspaceView>('overview')
  const [menuGroups, setMenuGroups] = useState<MenuGroup[]>(() => normalizeMenuGroups())
  const [expandedGroups, setExpandedGroups] = useState<Record<string, boolean>>(() => defaultExpandedGroups())
  const [themeMode, setThemeMode] = useState<ThemeMode>('dark')
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const [menuReady, setMenuReady] = useState(false)
  const [draggedDomain, setDraggedDomain] = useState<string | null>(null)
  const [assistantOpen, setAssistantOpen] = useState(false)
  const [assistantIntent, setAssistantIntent] = useState('')
  const [assistantIntentKey, setAssistantIntentKey] = useState('')
  const [operationContext, setOperationContext] = useState<Record<string, string>>({})

  useEffect(() => {
    let cancelled = false

    Promise.resolve().then(() => {
      if (cancelled) return
      const existingToken = getToken()
      const sessionUser = getSessionUser()
      setToken(existingToken)
      setUserId(sessionUser?.id ?? null)
      setUserType(sessionUser?.type ?? null)
      setMenuGroups(loadMenuGroups())
      setExpandedGroups(loadExpandedGroups())
      setThemeMode(loadThemeMode())
      setMenuReady(true)
      setReady(true)
    })

    listRoles()
      .then((data) => {
        if (!cancelled) setRoles(data)
      })
      .catch(() => {
        if (!cancelled) setRoles([])
      })

    return () => {
      cancelled = true
    }
  }, [])

  useEffect(() => {
    if (!menuReady || typeof window === 'undefined') return
    window.localStorage.setItem(menuStorageKey, JSON.stringify(menuGroups))
  }, [menuGroups, menuReady])

  useEffect(() => {
    if (!menuReady || typeof window === 'undefined') return
    window.localStorage.setItem(expandedMenuStorageKey, JSON.stringify(expandedGroups))
  }, [expandedGroups, menuReady])

  useEffect(() => {
    if (!menuReady || typeof window === 'undefined') return
    window.localStorage.setItem(themeStorageKey, themeMode)
  }, [menuReady, themeMode])

  useEffect(() => {
    if (!token) return
    let cancelled = false

    Promise.all([getMetaOrgOverview(token), getMetaOrgInbox(token)])
      .then(([overviewData, inboxData]) => {
        if (!cancelled) {
          setOverview(overviewData)
          setInbox(inboxData)
        }
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : t('加载概览失败'))
      })

    return () => {
      cancelled = true
    }
  }, [t, token])

  const healthRatio = useMemo(() => {
    if (!overview) return 0
    const active = overview.health.active_projects
    const total = Math.max(overview.health.active_projects + overview.health.open_requirements, 1)
    return active / total
  }, [overview])

  const handleOperationContextChange = useCallback((context: Record<string, string>) => {
    setOperationContext((current) => ({ ...current, ...context }))
  }, [])

  if (!ready) {
    return (
      <main className="app-dark">
        <div className="flex min-h-screen items-center justify-center">
          <RefreshCw className="h-8 w-8 animate-spin text-[#DF6A24]" />
        </div>
      </main>
    )
  }

  async function loadOverview(activeToken = token) {
    if (!activeToken) return
    setOverviewLoading(true)
    setError(null)
    try {
      const [overviewData, inboxData] = await Promise.all([getMetaOrgOverview(activeToken), getMetaOrgInbox(activeToken)])
      setOverview(overviewData)
      setInbox(inboxData)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('加载概览失败'))
    } finally {
      setOverviewLoading(false)
    }
  }

  async function handleAuth(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setLoading(true)
    setError(null)
    setNotice(null)

    try {
      if (mode === 'register') {
        await registerUser({ name, email, password })
        setMode('login')
        setNotice(t('auth.accountCreated'))
        setPassword('')
        return
      }

      const response = await login(email, password)
      setSession(response.token, response.user_id, response.user_type)
      setOverview(null)
      setToken(response.token)
      setUserId(response.user_id)
      setUserType(response.user_type)
      setPassword('')
    } catch (err) {
      setError(err instanceof Error ? err.message : t('auth.failed'))
    } finally {
      setLoading(false)
    }
  }

  function handleSignOut() {
    clearSession()
    setToken(null)
    setUserId(null)
    setUserType(null)
    setOverview(null)
    setInbox([])
    setError(null)
    setWorkspaceView('overview')
  }

  function toggleMenuGroup(groupID: string) {
    setExpandedGroups((current) => ({
      ...current,
      [groupID]: !current[groupID],
    }))
  }

  function handleDomainDragStart(event: DragEvent<HTMLButtonElement>, domain: string) {
    setDraggedDomain(domain)
    event.dataTransfer.effectAllowed = 'move'
    event.dataTransfer.setData('text/plain', domain)
  }

  function handleDomainDrop(event: DragEvent<HTMLElement>, groupID: string) {
    event.preventDefault()
    const domain = event.dataTransfer.getData('text/plain') || draggedDomain
    if (!domain || ![...operationDomains, ...virtualDomains].includes(domain)) return
    setMenuGroups((current) =>
      current.map((group) => {
        const domains = group.domains.filter((item) => item !== domain)
        if (group.id !== groupID) return { ...group, domains }
        return { ...group, domains: [...domains, domain] }
      }),
    )
    setExpandedGroups((current) => ({ ...current, [groupID]: true }))
    setDraggedDomain(null)
  }

  function resetMenuLayout() {
    setMenuGroups(normalizeMenuGroups())
    setExpandedGroups(defaultExpandedGroups())
  }

  function handleViewChange(view: WorkspaceView) {
    setWorkspaceView(view)
    setMobileMenuOpen(false)
  }

  function delegateOperationToAgent(operation: ApiOperation) {
    setAssistantIntent(agentIntentForOperation(operation, operationContext))
    setAssistantIntentKey(`${operation.id}-${Date.now()}`)
    setAssistantOpen(true)
  }

  async function handleToolApproval(id: string, decision: 'approve' | 'reject') {
    if (!token) return
    setOverviewLoading(true)
    setError(null)
    try {
      if (decision === 'approve') {
        await approveToolApproval(token, id)
        setNotice(t('agent.approvalApproved'))
      } else {
        await rejectToolApproval(token, id)
        setNotice(t('agent.approvalRejected'))
      }
      await loadOverview(token)
    } catch (err) {
      setError(err instanceof Error ? err.message : t('common.operationFailed'))
    } finally {
      setOverviewLoading(false)
    }
  }

  const activeDomain = workspaceView === 'overview' ? 'MetaOrg' : workspaceView.replace('domain:', '')
  const activeGroup = menuGroups.find((group) => group.domains.includes(activeDomain))
  const activeOperationCount =
    workspaceView === 'overview'
      ? apiOperations.filter((operation) => operation.domain === 'MetaOrg').length
      : apiOperations.filter((operation) => operation.domain === activeDomain).length
  const activeOperations = apiOperations.filter((operation) => operation.domain === (workspaceView === 'overview' ? 'MetaOrg' : activeDomain))
  const assistantModule = assistantModuleForDomain(workspaceView === 'overview' ? 'MetaOrg' : activeDomain)

  return (
    <main className={`app-dark ${themeMode === 'light' ? 'theme-light' : ''}`}>
      {!token ? (
        <div className="mx-auto grid min-h-screen max-w-6xl gap-5 px-4 py-8 sm:px-6 lg:grid-cols-[360px_1fr] lg:items-center lg:px-8">
          <section className="studio-panel rounded-lg p-5">
            <div className="mb-8 flex items-center gap-3">
              <div className="flex h-12 w-12 items-center justify-center rounded-lg border border-[#DF6A24]/25 bg-[#DF6A24]/10">
                <Sparkles className="h-6 w-6 text-[#F6A66A]" />
              </div>
              <div>
                <p className="text-xs font-bold uppercase text-[#F6A66A]">{t('shell.breadcrumbRoot')}</p>
                <h1 className="text-2xl font-semibold text-white">{t('app.title')}</h1>
              </div>
            </div>
            <div className="flex rounded-lg bg-slate-100 p-1">
              <button
                type="button"
                onClick={() => setMode('login')}
                className={`h-9 flex-1 rounded-md text-sm font-medium transition ${
                  mode === 'login' ? 'bg-white text-slate-950 shadow-sm' : 'text-slate-500 hover:text-slate-800'
                }`}
              >
                {t('auth.login')}
              </button>
              <button
                type="button"
                onClick={() => setMode('register')}
                className={`h-9 flex-1 rounded-md text-sm font-medium transition ${
                  mode === 'register' ? 'bg-white text-slate-950 shadow-sm' : 'text-slate-500 hover:text-slate-800'
                }`}
              >
                {t('auth.register')}
              </button>
            </div>

            <form className="mt-5 space-y-4" onSubmit={handleAuth}>
              {mode === 'register' && (
                <label className="block">
                  <span className="text-sm font-medium text-slate-700">{t('auth.name')}</span>
                  <input
                    value={name}
                    onChange={(event) => setName(event.target.value)}
                    className="mt-1 h-11 w-full rounded-lg border border-slate-300 px-3 text-sm outline-none transition focus:border-slate-500 focus:ring-2 focus:ring-slate-200"
                    autoComplete="name"
                    required
                  />
                </label>
              )}
              <label className="block">
                <span className="text-sm font-medium text-slate-700">{t('auth.email')}</span>
                <input
                  value={email}
                  onChange={(event) => setEmail(event.target.value)}
                  className="mt-1 h-11 w-full rounded-lg border border-slate-300 px-3 text-sm outline-none transition focus:border-slate-500 focus:ring-2 focus:ring-slate-200"
                  autoComplete="email"
                  type="email"
                  required
                />
              </label>
              <label className="block">
                <span className="text-sm font-medium text-slate-700">{t('auth.password')}</span>
                <input
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                  className="mt-1 h-11 w-full rounded-lg border border-slate-300 px-3 text-sm outline-none transition focus:border-slate-500 focus:ring-2 focus:ring-slate-200"
                  autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
                  type="password"
                  required
                />
              </label>

              {error && (
                <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
                  {error}
                </div>
              )}
              {notice && (
                <div className="rounded-lg border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-700">
                  {notice}
                </div>
              )}

              <button
                type="submit"
                disabled={loading}
                className="inline-flex h-11 w-full items-center justify-center gap-2 rounded-lg bg-[#AD4714] px-4 text-sm font-semibold text-[#fffaf5] transition hover:bg-[#B84F18] disabled:cursor-not-allowed disabled:opacity-60"
              >
                <KeyRound className="h-4 w-4" />
                {loading ? t('auth.processing') : mode === 'login' ? t('auth.signIn') : t('auth.createAccount')}
              </button>
            </form>
          </section>

          <RoleDirectory roles={roles} />
        </div>
      ) : (
        <div className="grid min-h-screen lg:grid-cols-[248px_1fr]">
          <div
            className={`fixed inset-y-0 left-0 z-40 w-[248px] transform transition lg:static lg:translate-x-0 ${
              mobileMenuOpen ? 'translate-x-0' : '-translate-x-full'
            }`}
          >
            <NavigationSidebar
              workspaceView={workspaceView}
              groups={menuGroups}
              expandedGroups={expandedGroups}
              onViewChange={handleViewChange}
              onToggleGroup={toggleMenuGroup}
              onDragStart={handleDomainDragStart}
              onDropDomain={handleDomainDrop}
              onReset={resetMenuLayout}
            />
          </div>
          {mobileMenuOpen && (
            <button
              type="button"
              aria-label="Close menu"
              className="fixed inset-0 z-30 bg-black/60 lg:hidden"
              onClick={() => setMobileMenuOpen(false)}
            />
          )}

          <section className="min-w-0">
            <Topbar
              activeTitle={workspaceView === 'overview' ? t('nav.overview') : t(domainLabels[activeDomain] ?? activeDomain)}
              activeDomain={workspaceView === 'overview' ? 'SuperClaw' : activeDomain}
              locale={locale}
              setLocale={setLocale}
              themeMode={themeMode}
              setThemeMode={setThemeMode}
              userType={userType}
              overview={overview}
              overviewLoading={overviewLoading}
              onRefresh={() => loadOverview()}
              onSignOut={handleSignOut}
              onOpenMenu={() => setMobileMenuOpen(true)}
            />
            <div className="mx-auto max-w-7xl space-y-5 px-4 py-6 sm:px-6 lg:px-8">
              <WorkspaceHeader
                title={workspaceView === 'overview' ? '总览' : domainLabels[activeDomain] ?? activeDomain}
                domain={workspaceView === 'overview' ? 'Overview' : activeDomain}
                groupLabel={workspaceView === 'overview' ? '工作台' : activeGroup?.label ?? '功能台'}
                operationCount={activeOperationCount}
                operations={activeOperations}
                dedicated={workspaceView === 'overview' || dedicatedDomains.has(activeDomain)}
                onOperationSelect={delegateOperationToAgent}
                onAssistantOpen={() => {
                  setAssistantIntent('')
                  setAssistantIntentKey('')
                  setAssistantOpen(true)
                }}
              />
              {error && (
                <div className="mb-5 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
                  {error}
                </div>
              )}
              {workspaceView === 'overview' ? (
                overview ? (
                  <Dashboard
                    token={token}
                    overview={overview}
                    inbox={inbox}
                    healthRatio={healthRatio}
                    onReviewApproval={(id, decision) => void handleToolApproval(id, decision)}
                  />
                ) : (
                  <div className="flex min-h-[420px] items-center justify-center rounded-lg border border-slate-200 bg-white">
                    <RefreshCw className="h-5 w-5 animate-spin text-slate-500" />
                  </div>
                )
              ) : workspaceView === 'domain:Organization' ? (
                <OrganizationWorkspace token={token} currentUserId={userId} />
              ) : workspaceView === 'domain:MetaResource' ? (
                <MetaResourceWorkspace token={token} />
              ) : workspaceView === 'domain:Governance' ? (
                <GovernanceWorkspace token={token} currentUserId={userId} />
              ) : workspaceView === 'domain:Evolution' ? (
                <WeightWorkspace token={token} currentUserId={userId} />
              ) : workspaceView === 'domain:Capability' ? (
                <CapabilityEvaluationWorkspace token={token} currentUserId={userId} />
              ) : workspaceView === 'domain:Workflow' ? (
                <div className="space-y-5">
                  <WorkflowDesignerWorkspace token={token} currentUserId={userId} />
                  <WorkflowMatchingWorkspace token={token} currentUserId={userId} />
                </div>
              ) : ['domain:Requirement', 'domain:Project', 'domain:Delivery', 'domain:Cost', 'domain:Feedback'].includes(
                  workspaceView,
                ) ? (
                <ProjectLifecycleWorkspace
                  token={token}
                  currentUserId={userId}
                  mode={workspaceView.replace('domain:', '') as 'Requirement' | 'Project' | 'Delivery' | 'Cost' | 'Feedback'}
                  onOperationContextChange={handleOperationContextChange}
                />
              ) : workspaceView === 'domain:DeveloperTools' ? (
                <DeveloperToolsWorkspace token={token} />
              ) : workspaceView === 'domain:Costing' || workspaceView === 'domain:FinanceCostAccounting' ? (
                <CostingWorkspace token={token} />
              ) : workspaceView === 'domain:Finance' || workspaceView === 'domain:FinanceAccounting' ? (
                <FinanceWorkspace token={token} mode="accounting" />
              ) : workspaceView === 'domain:FinanceReceivables' ? (
                <FinanceWorkspace token={token} mode="receivables" />
              ) : workspaceView === 'domain:FinancePayables' ? (
                <FinanceWorkspace token={token} mode="payables" />
              ) : (
                <AgentOnlyWorkspace domain={workspaceView.replace('domain:', '')} onAssistantOpen={() => setAssistantOpen(true)} />
              )}
            </div>
          </section>
          {assistantOpen && (
            <div className="fixed inset-0 z-50">
              <button
                type="button"
                className="absolute inset-0 bg-black/55"
                aria-label={t('common.close')}
                onClick={() => {
                  setAssistantOpen(false)
                  setAssistantIntent('')
                  setAssistantIntentKey('')
                }}
              />
              <aside className="absolute right-0 top-0 h-full w-full max-w-xl shadow-2xl">
                <button
                  type="button"
                  onClick={() => {
                    setAssistantOpen(false)
                    setAssistantIntent('')
                    setAssistantIntentKey('')
                  }}
                  className="absolute right-3 top-3 z-10 inline-flex h-9 w-9 items-center justify-center rounded-lg border border-slate-200 bg-white text-slate-500 transition hover:bg-slate-100 hover:text-slate-900"
                  aria-label={t('common.close')}
                >
                  <X className="h-4 w-4" />
                </button>
                <AIAssistant
                  token={token}
                  contextType={assistantModule}
                  initialIntent={assistantIntent}
                  initialIntentKey={assistantIntentKey}
                  autoRunInitialIntent={Boolean(assistantIntent)}
                />
              </aside>
            </div>
          )}
        </div>
      )}
    </main>
  )
}

function Topbar({
  activeTitle,
  activeDomain,
  locale,
  setLocale,
  themeMode,
  setThemeMode,
  userType,
  overview,
  overviewLoading,
  onRefresh,
  onSignOut,
  onOpenMenu,
}: {
  activeTitle: string
  activeDomain: string
  locale: 'zh' | 'en'
  setLocale: (locale: 'zh' | 'en') => void
  themeMode: ThemeMode
  setThemeMode: (mode: ThemeMode) => void
  userType: string | null
  overview: MetaOrgOverview | null
  overviewLoading: boolean
  onRefresh: () => void
  onSignOut: () => void
  onOpenMenu: () => void
}) {
  const { t } = useI18n()
  const activeWork = overview?.health.active_projects ?? 0
  const unexportedCost = overview ? formatMoney(overview.health.unexported_cost, overview.health.currency) : 'CNY 0.00'

  return (
    <header className="sticky top-0 z-20 border-b border-slate-800/80 bg-[#121317]/88 backdrop-blur-xl">
      <div className="flex h-[60px] items-center justify-between gap-3 px-4 sm:px-6 lg:px-8">
        <div className="flex min-w-0 items-center gap-3">
          <button
            type="button"
            onClick={onOpenMenu}
            className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-slate-700 text-slate-300 lg:hidden"
          >
            <Menu className="h-4 w-4" />
          </button>
          <div className="min-w-0 text-xs font-semibold">
            <span className="text-[#F6A66A]">{t('shell.breadcrumbRoot')}</span>
            <span className="px-2 text-slate-500">/</span>
            <span className="truncate text-slate-400">{activeTitle}</span>
          </div>
        </div>

        <div className="flex min-w-0 items-center justify-end gap-2">
          <div className="hidden items-center gap-2 text-xs font-semibold text-slate-400 xl:flex">
            <span>{t('shell.liveBounties', { count: activeWork })}</span>
            <span className="text-slate-600">·</span>
            <span>{t('shell.escrow', { amount: unexportedCost })}</span>
            <span className="text-slate-600">·</span>
            <span>{activeDomain}</span>
          </div>
          {userType && <StatusPill label={userType === 'ai' ? 'AI Agent' : 'Human'} tone="blue" />}
          {overview && (
            <button
              type="button"
              onClick={onRefresh}
              disabled={overviewLoading}
              className="hidden h-9 items-center gap-2 rounded-lg border border-slate-700 px-3 text-xs font-semibold text-slate-300 transition hover:border-blue-400/60 hover:text-blue-200 disabled:opacity-60 sm:inline-flex"
            >
              <RefreshCw className={`h-3.5 w-3.5 ${overviewLoading ? 'animate-spin' : ''}`} />
              {formatDate(overview.generated_at)}
            </button>
          )}
          <a
            href={projectGithubURL}
            target="_blank"
            rel="noreferrer"
            aria-label={t('shell.githubLink')}
            className="inline-flex h-9 items-center gap-2 rounded-lg border border-slate-700 px-3 text-xs font-bold text-slate-300 transition hover:border-blue-400/60 hover:text-blue-200"
          >
            <Github className="h-3.5 w-3.5" />
            <span className="hidden sm:inline">GitHub</span>
          </a>
          <div className="inline-flex h-9 items-center rounded-lg border border-slate-700 bg-slate-950/40 p-1">
            <button
              type="button"
              onClick={() => setLocale('zh')}
              className={`h-7 rounded-md px-2 text-xs font-bold transition ${
                locale === 'zh' ? 'bg-slate-100 text-slate-950' : 'text-slate-400 hover:text-white'
              }`}
            >
              中文
            </button>
            <button
              type="button"
              onClick={() => setLocale('en')}
              className={`h-7 rounded-md px-2 text-xs font-bold transition ${
                locale === 'en' ? 'bg-slate-100 text-slate-950' : 'text-slate-400 hover:text-white'
              }`}
            >
              EN
            </button>
          </div>
          <button
            type="button"
            onClick={() => setThemeMode(themeMode === 'dark' ? 'light' : 'dark')}
            className="inline-flex h-9 items-center gap-2 rounded-lg border border-slate-700 px-3 text-xs font-bold uppercase text-slate-300 transition hover:border-blue-400/60 hover:text-blue-200"
          >
            {themeMode === 'dark' ? <Moon className="h-3.5 w-3.5" /> : <Sun className="h-3.5 w-3.5" />}
            <span className="hidden sm:inline">{themeMode === 'dark' ? t('shell.theme.dark') : t('shell.theme.light')}</span>
          </button>
          <button
            type="button"
            onClick={onSignOut}
            className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-slate-700 text-slate-300 transition hover:border-blue-400/60 hover:text-blue-200"
          >
            <LogOut className="h-4 w-4" />
          </button>
        </div>
      </div>
    </header>
  )
}

function NavigationSidebar({
  workspaceView,
  groups,
  expandedGroups,
  onViewChange,
  onToggleGroup,
  onDragStart,
  onDropDomain,
  onReset,
}: {
  workspaceView: WorkspaceView
  groups: MenuGroup[]
  expandedGroups: Record<string, boolean>
  onViewChange: (view: WorkspaceView) => void
  onToggleGroup: (groupID: string) => void
  onDragStart: (event: DragEvent<HTMLButtonElement>, domain: string) => void
  onDropDomain: (event: DragEvent<HTMLElement>, groupID: string) => void
  onReset: () => void
}) {
  const { t } = useI18n()
  return (
    <aside className="studio-sidebar flex h-full min-h-screen flex-col px-3 py-4">
      <div className="flex items-center gap-3 px-2 pb-6">
        <div className="flex h-11 w-11 items-center justify-center rounded-lg border border-[#DF6A24]/25 bg-[#DF6A24]/10">
          <Sparkles className="h-6 w-6 text-[#F6A66A]" />
        </div>
        <div className="min-w-0">
          <p className="text-xl font-extrabold leading-none tracking-normal text-white">META-ORG</p>
          <p className="mt-1 text-[9px] font-bold uppercase tracking-[0.18em] text-[#F6A66A]">AI Operating System</p>
        </div>
      </div>

      <div className="space-y-1">
        <SidebarButton
          active={workspaceView === 'overview'}
          icon={HomeIcon}
          label={t('nav.item.home')}
          onClick={() => onViewChange('overview')}
        />
      </div>

      <div className="mt-6 flex-1 space-y-5 overflow-y-auto pr-1">
        {groups.map((group) => {
          const expanded = expandedGroups[group.id] ?? true
          const groupOperations = group.domains.reduce(
            (sum, domain) => sum + apiOperations.filter((operation) => operation.domain === domain).length,
            0,
          )

          return (
            <div
              key={group.id}
              className="space-y-1"
              onDragOver={(event) => event.preventDefault()}
              onDrop={(event) => onDropDomain(event, group.id)}
            >
              <button
                type="button"
                onClick={() => onToggleGroup(group.id)}
                className="flex h-9 w-full items-center justify-between px-2 text-left text-base font-semibold tracking-normal text-slate-300"
              >
                <span className="inline-flex min-w-0 items-center gap-2">
                  {expanded ? <ChevronDown className="h-4 w-4 shrink-0" /> : <ChevronRight className="h-4 w-4 shrink-0" />}
                  <span className="truncate">{t(`nav.group.${group.id}`)}</span>
                </span>
                <span className="text-xs font-bold text-slate-500">{groupOperations}</span>
              </button>

              {expanded && (
                <div className="space-y-1">
                  {group.domains.map((domain) => {
                    const menuKey = `domain:${domain}` as const
                    const count = apiOperations.filter((operation) => operation.domain === domain).length
                    const Icon = domainIcons[domain] ?? FolderKanban

                    return (
                      <SidebarButton
                        key={domain}
                        active={workspaceView === menuKey}
                        icon={Icon}
                        label={t(domainLabels[domain] ?? domain)}
                        count={count}
                        onClick={() => onViewChange(menuKey)}
                        draggable
                        onDragStart={(event) => onDragStart(event, domain)}
                      />
                    )
                  })}
                  {group.domains.length === 0 && <p className="px-3 py-2 text-sm text-slate-500">{t('nav.empty')}</p>}
                </div>
              )}
            </div>
          )
        })}
      </div>

      <div className="mt-5 border-t border-slate-800 pt-3">
        <button
          type="button"
          onClick={onReset}
          className="inline-flex h-9 w-full items-center justify-center gap-2 rounded-lg border border-slate-800 text-xs font-bold text-slate-400 transition hover:border-blue-400/50 hover:text-blue-200"
        >
          <SlidersHorizontal className="h-3.5 w-3.5" />
          {t('nav.reset')}
        </button>
      </div>
    </aside>
  )
}

function SidebarButton({
  active,
  icon: Icon,
  label,
  badge,
  count,
  onClick,
  draggable,
  onDragStart,
}: {
  active: boolean
  icon: typeof Gauge
  label: string
  badge?: string
  count?: number
  onClick: () => void
  draggable?: boolean
  onDragStart?: (event: DragEvent<HTMLButtonElement>) => void
}) {
  return (
    <button
      type="button"
      draggable={draggable}
      onDragStart={onDragStart}
      onClick={onClick}
      className={`studio-nav-item flex h-10 w-full items-center justify-between gap-2 rounded-lg border border-transparent px-3 text-left text-sm font-semibold transition ${
        active ? 'studio-nav-item-active' : ''
      }`}
    >
      <span className="inline-flex min-w-0 items-center gap-3">
        <Icon className={`h-4 w-4 shrink-0 ${active ? 'text-[#F6A66A]' : 'text-slate-500'}`} />
        <span className="truncate">{label}</span>
      </span>
      {badge ? (
        <span className="rounded-full border border-emerald-400/40 bg-emerald-500/15 px-2 py-0.5 text-[10px] font-bold text-emerald-300">
          {badge}
        </span>
      ) : count !== undefined ? (
        <span className="text-[11px] text-slate-500">{count}</span>
      ) : null}
    </button>
  )
}

function WorkspaceHeader({
  title,
  domain,
  groupLabel,
  operationCount,
  operations,
  dedicated,
  onOperationSelect,
  onAssistantOpen,
}: {
  title: string
  domain: string
  groupLabel: string
  operationCount: number
  operations: ApiOperation[]
  dedicated: boolean
  onOperationSelect: (operation: ApiOperation) => void
  onAssistantOpen: () => void
}) {
  const { t } = useI18n()
  const [moreOpen, setMoreOpen] = useState(false)
  const prioritizedOperations = [...operations].sort((left, right) => {
    const order = { direct: 0, contextual: 1, agent_assisted: 2, admin: 3 }
    return order[getOperationProfile(left).kind] - order[getOperationProfile(right).kind]
  })
  const primaryOperations = prioritizedOperations.slice(0, 4)
  const overflowOperations = prioritizedOperations.slice(4)
  return (
    <section className="studio-panel rounded-lg p-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0">
          <p className="text-xs font-bold uppercase tracking-[0.16em] text-slate-500">{t(groupLabel)}</p>
          <h2 className="mt-1 truncate text-xl font-semibold text-white">{t(title)}</h2>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span className="studio-badge inline-flex h-8 items-center rounded-md px-2.5 text-xs font-semibold">
            {domain}
          </span>
          {operationCount > 0 && (
            <span className="inline-flex h-8 items-center rounded-md border border-slate-700 bg-slate-950/30 px-2.5 text-xs font-semibold text-slate-300">
              {operationCount} {t('agent.actions')}
            </span>
          )}
          <span
            className={`inline-flex h-8 items-center rounded-md border px-2.5 text-xs font-semibold ${
              dedicated
                ? 'border-emerald-400/40 bg-emerald-500/15 text-emerald-300'
                : 'border-[#DF6A24]/25 bg-[#DF6A24]/10 text-[#F6A66A]'
            }`}
          >
            {dedicated ? t('workspace.workspace') : t('workspace.api')}
          </span>
          {primaryOperations.map((operation) => (
            <OperationButton key={operation.id} operation={operation} onClick={() => onOperationSelect(operation)} />
          ))}
          {overflowOperations.length > 0 && (
            <div className="relative">
              <button
                type="button"
                onClick={() => setMoreOpen((current) => !current)}
                className="inline-flex h-8 items-center gap-1.5 rounded-md border border-slate-700 bg-slate-950/30 px-2.5 text-xs font-semibold text-slate-200 transition hover:border-[#DF6A24]/50 hover:text-[#F6A66A]"
              >
                <MoreHorizontal className="h-3.5 w-3.5" />
                {t('operation.more')}
              </button>
              {moreOpen && (
                <div className="absolute right-0 z-20 mt-2 w-64 rounded-lg border border-slate-700 bg-slate-950 p-2 shadow-xl">
                  {overflowOperations.map((operation) => (
                    <button
                      key={operation.id}
                      type="button"
                      onClick={() => {
                        setMoreOpen(false)
                        onOperationSelect(operation)
                      }}
                      className="flex h-10 w-full items-center justify-between gap-2 rounded-md px-2 text-left text-xs font-semibold text-slate-300 transition hover:bg-[#DF6A24]/10 hover:text-[#F6A66A]"
                    >
                      <span className="min-w-0">
                        <span className="block truncate">{t(operation.title)}</span>
                        <span className="block truncate text-[10px] font-medium text-slate-500">{t(`operation.kind.${getOperationProfile(operation).kind}`)}</span>
                      </span>
                      <span className="text-[10px] text-slate-500">{t('agent.delegate')}</span>
                    </button>
                  ))}
                </div>
              )}
            </div>
          )}
          <button
            type="button"
            onClick={onAssistantOpen}
            className="inline-flex h-8 items-center gap-1.5 rounded-md bg-[#AD4714] px-2.5 text-xs font-bold text-[#fffaf5] transition hover:bg-[#B84F18]"
          >
            <Bot className="h-3.5 w-3.5" />
            {t('assistant.title')}
          </button>
        </div>
      </div>
    </section>
  )
}

function OperationButton({ operation, onClick }: { operation: ApiOperation; onClick: () => void }) {
  const { t } = useI18n()
  const profile = getOperationProfile(operation)
  const toneClass = {
    direct: 'border-blue-400/35 bg-blue-500/10 text-blue-100 hover:border-blue-300/60',
    contextual: 'border-amber-400/35 bg-amber-500/10 text-amber-100 hover:border-amber-300/60',
    agent_assisted: 'border-violet-400/35 bg-violet-500/10 text-violet-100 hover:border-violet-300/60',
    admin: 'border-slate-700 bg-slate-950/30 text-slate-200 hover:border-[#DF6A24]/50 hover:text-[#F6A66A]',
  }[profile.kind]

  return (
    <button
      type="button"
      onClick={onClick}
      className={`inline-flex h-8 max-w-[190px] items-center gap-1.5 rounded-md border px-2.5 text-xs font-semibold transition ${toneClass}`}
      title={`${t('agent.delegate')} · ${t(operation.title)} · ${t(`operation.kind.${profile.kind}`)}`}
    >
      <Bot className="h-3.5 w-3.5 shrink-0" />
      <span className="truncate">{t(operation.title)}</span>
      {profile.requiresEntityContext && <span className="text-[10px] opacity-70">*</span>}
    </button>
  )
}

function AgentOnlyWorkspace({ domain, onAssistantOpen }: { domain: string; onAssistantOpen: () => void }) {
  const { t } = useI18n()
  return (
    <section className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <p className="text-sm font-semibold text-slate-500">{domain}</p>
          <h2 className="mt-1 text-base font-semibold text-slate-950">{t('agent.consoleTitle')}</h2>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-slate-600">{t('agent.consoleHint')}</p>
        </div>
        <button
          type="button"
          onClick={onAssistantOpen}
          className="inline-flex h-10 items-center justify-center gap-2 rounded-lg bg-[#AD4714] px-3 text-sm font-semibold text-[#fffaf5] transition hover:bg-[#B84F18]"
        >
          <Bot className="h-4 w-4" />
          {t('assistant.title')}
        </button>
      </div>
    </section>
  )
}

function Dashboard({
  token,
  overview,
  inbox,
  healthRatio,
  onReviewApproval,
}: {
  token: string
  overview: MetaOrgOverview
  inbox: InboxItem[]
  healthRatio: number
  onReviewApproval: (id: string, decision: 'approve' | 'reject') => void
}) {
  const { t } = useI18n()
  const agentCoverage = overview.agents.total > 0 ? overview.agents.active / overview.agents.total : 0
  const filters = [
    ['shell.filter.all', Sparkles],
    ['shell.filter.business', BriefcaseBusiness],
    ['shell.filter.organization', Users],
    ['shell.filter.governance', ShieldCheck],
    ['shell.filter.aiOps', Bot],
    ['shell.filter.finance', WalletCards],
  ] as const

  return (
    <div className="space-y-5">
        <section className="px-2 py-6 text-center sm:py-10">
          <div className="mx-auto inline-flex items-center gap-2 text-xs font-bold uppercase tracking-[0.16em] text-slate-500">
            <span className="h-2 w-2 rounded-full bg-emerald-400 shadow-[0_0_14px_rgba(52,211,153,0.75)]" />
            {t('nav.badge.live')} - Meta-Org
          </div>
          <h1 className="mx-auto mt-5 max-w-4xl text-balance text-4xl font-extrabold tracking-normal text-white sm:text-5xl">
            {t('shell.commandTitle').replace('?', '')}
            <span className="text-[#DF6A24]">?</span>
          </h1>
          <p className="mx-auto mt-4 max-w-2xl text-base font-medium text-slate-400">{t('shell.commandSubtitle')}</p>

          <div className="mt-8 flex flex-wrap justify-center gap-2">
            {filters.map(([label, Icon], index) => (
              <button
                key={label}
                type="button"
                className={`inline-flex h-9 items-center gap-2 rounded-full border px-4 text-sm font-bold transition ${
                  index === 0
                    ? 'border-[#DF6A24] bg-[#DF6A24]/10 text-[#F6A66A]'
                    : 'border-slate-700 bg-slate-950/30 text-slate-300 hover:border-[#DF6A24]/35'
                }`}
              >
                <Icon className="h-4 w-4" />
                {t(label)}
              </button>
            ))}
          </div>
        </section>

        <GlobalBusinessInteraction token={token} />

        <div className="flex items-center justify-between gap-3 px-1">
          <h2 className="text-sm font-bold uppercase tracking-normal text-slate-400">{t('shell.openWork')}</h2>
          <span className="text-xs font-bold text-slate-500">
            {t('shell.results', { count: overview.activity.length + inbox.length })}
          </span>
        </div>

        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <MetricCard
            icon={Users}
            label={t('meta.openRequirements')}
            value={formatNumber(overview.health.open_requirements)}
            detail={t('meta.pendingWork')}
            tone="blue"
          />
          <MetricCard
            icon={GitBranch}
            label={t('meta.activeProjects')}
            value={formatNumber(overview.health.active_projects)}
            detail={`${formatPercent(healthRatio)} ${t('active')}`}
            tone="emerald"
          />
          <MetricCard
            icon={Bot}
            label={t('meta.activeAgents')}
            value={formatNumber(overview.health.active_agents)}
            detail={`${formatNumber(overview.agents.total)} ${t('meta.totalAgents')}`}
            tone="amber"
          />
          <MetricCard
            icon={ShieldCheck}
            label={t('meta.pendingApprovals')}
            value={formatNumber(overview.health.pending_approvals)}
            detail={formatMoney(overview.health.unexported_cost, overview.health.currency)}
            tone="violet"
          />
        </div>

        <div className="grid gap-4 xl:grid-cols-3">
          <StatusBars
            title={t('meta.projectStatus')}
            icon={Workflow}
            data={overview.projects.by_status}
            labels={{
              planning: 'planning',
              active: 'active',
              paused: 'paused',
              delivering: 'delivering',
              completed: 'completed',
            }}
          />
          <StatusBars
            title={t('meta.agentRisk')}
            icon={BrainCircuit}
            data={overview.agents.by_risk_level}
            labels={{
              low: 'low',
              medium: 'medium',
              high: 'high',
              critical: 'critical',
            }}
          />
          <section className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
            <div className="flex items-center gap-2">
              <Bot className="h-5 w-5 text-slate-500" />
              <h2 className="text-base font-semibold text-slate-950">{t('meta.agentCoverage')}</h2>
            </div>
            <div className="mt-5">
              <div className="flex items-end justify-between">
                <span className="text-3xl font-semibold text-slate-950">{formatPercent(agentCoverage)}</span>
                <span className="text-sm text-slate-500">
                  {formatNumber(overview.agents.active)} / {formatNumber(overview.agents.total)}
                </span>
              </div>
              <div className="mt-3 h-2 rounded-full bg-slate-100">
                <div
                  className="h-2 rounded-full bg-emerald-500"
                  style={{ width: `${Math.min(agentCoverage * 100, 100)}%` }}
                />
              </div>
            </div>
          </section>
        </div>

        <div className="grid gap-4 xl:grid-cols-[1fr_0.8fr]">
          <section className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
            <div className="flex items-center justify-between gap-3">
              <div className="flex items-center gap-2">
                <Gauge className="h-5 w-5 text-slate-500" />
                <h2 className="text-base font-semibold text-slate-950">{t('meta.costPanel')}</h2>
              </div>
              <StatusPill label={overview.cost.currency || 'CNY'} tone="blue" />
            </div>
            <div className="mt-5 grid gap-3 sm:grid-cols-3">
              <SignalStat label={t('meta.todayCost')} value={formatMoney(overview.cost.today, overview.cost.currency)} />
              <SignalStat
                label={t('meta.monthCost')}
                value={formatMoney(overview.cost.month_to_date, overview.cost.currency)}
              />
              <SignalStat
                label={t('meta.unexportedCost')}
                value={formatMoney(overview.cost.unexported, overview.cost.currency)}
              />
            </div>
            <div className="mt-5 grid gap-3 sm:grid-cols-2">
              {Object.entries(overview.cost.by_provider).map(([provider, amount]) => (
                <SignalStat key={provider} label={t(`provider.${provider}`)} value={formatMoney(amount, overview.cost.currency)} />
              ))}
            </div>
          </section>

          <section className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
            <div className="flex items-center gap-2">
              <ShieldCheck className="h-5 w-5 text-slate-500" />
              <h2 className="text-base font-semibold text-slate-950">{t('meta.riskPanel')}</h2>
            </div>
            <div className="mt-4 divide-y divide-slate-100">
              {overview.risks.length > 0 ? (
                overview.risks.map((risk) => (
                  <div key={`${risk.source}-${risk.id}`} className="grid gap-2 py-3">
                    <div className="flex items-start justify-between gap-3">
                      <p className="text-sm font-semibold text-slate-900">{risk.title}</p>
                      <StatusPill label={risk.severity} tone={risk.severity === 'critical' || risk.severity === 'high' ? 'amber' : 'blue'} />
                    </div>
                    <p className="text-xs text-slate-500">{risk.source}</p>
                  </div>
                ))
              ) : (
                <p className="py-3 text-sm text-slate-500">{t('meta.noRisks')}</p>
              )}
            </div>
          </section>
        </div>

        <section className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
          <div className="flex items-center gap-2">
            <Activity className="h-5 w-5 text-slate-500" />
            <h2 className="text-base font-semibold text-slate-950">{t('meta.inbox')}</h2>
          </div>
          <div className="mt-5 divide-y divide-slate-100">
            {inbox.length > 0 ? (
              inbox.map((item) => (
                <div key={`${item.type}-${item.id}`} className="grid gap-2 py-3 sm:grid-cols-[1fr_auto]">
                  <div className="min-w-0">
                    <p className="truncate text-sm font-semibold text-slate-950">{item.title}</p>
                    <p className="mt-1 text-xs text-slate-500">
                      {item.type} · {item.source || t('common.none')} · {formatDate(item.created_at)}
                    </p>
                  </div>
                  <div className="flex flex-wrap justify-end gap-2">
                    <StatusPill label={item.priority} tone={item.priority === 'high' || item.priority === 'critical' ? 'amber' : 'blue'} />
                    <StatusPill label={item.status} tone="green" />
                    {item.type === 'tool_approval' && item.status === 'pending' && (
                      <>
                        <button
                          type="button"
                          onClick={() => onReviewApproval(item.id, 'approve')}
                          className="inline-flex h-7 items-center rounded-md bg-emerald-600 px-2.5 text-xs font-semibold text-white transition hover:bg-emerald-700"
                        >
                          {t('agent.approve')}
                        </button>
                        <button
                          type="button"
                          onClick={() => onReviewApproval(item.id, 'reject')}
                          className="inline-flex h-7 items-center rounded-md border border-red-200 bg-red-50 px-2.5 text-xs font-semibold text-red-700 transition hover:bg-red-100"
                        >
                          {t('agent.reject')}
                        </button>
                      </>
                    )}
                  </div>
                </div>
              ))
            ) : (
              <p className="py-3 text-sm text-slate-500">{t('meta.noInbox')}</p>
            )}
          </div>
        </section>

        <RecentEvents events={overview.activity} />
    </div>
  )
}

function GlobalBusinessInteraction({ token }: { token: string }) {
  const { t } = useI18n()
  const [moduleID, setModuleID] = useState('meta_org')
  const [targets, setTargets] = useState<AssistantContextTarget[]>([])
  const [targetValue, setTargetValue] = useState('')
  const [skills, setSkills] = useState<AssistantBusinessSkill[]>([])
  const [selectedSkillID, setSelectedSkillID] = useState('')
  const [sessionID, setSessionID] = useState('')
  const [proposals, setProposals] = useState<AssistantProposal[]>([])
  const [skillName, setSkillName] = useState('')
  const [skillPrompt, setSkillPrompt] = useState('')
  const [skillIntent, setSkillIntent] = useState('')
  const [skillIntentKey, setSkillIntentKey] = useState('')
  const [busy, setBusy] = useState(false)
  const [notice, setNotice] = useState('')
  const [error, setError] = useState('')
  const pendingSkillRunRef = useRef<{
    skillID: string
    moduleKey: string
    targetType: string
    targetID: string
  } | null>(null)
  const activeModule = globalAssistantModules.find((item) => item.id === moduleID) ?? globalAssistantModules[0]
  const moduleKey = activeModule.key
  const moduleTargetType = activeModule.targetType
  const selectedSkill = skills.find((skill) => skill.id === selectedSkillID)

  const selectedTarget = useMemo(() => {
    if (!targetValue) return undefined
    return targets.find((target) => `${target.type}:${target.id}` === targetValue)
  }, [targetValue, targets])
  const activeTargetType = selectedTarget?.type || moduleTargetType
  const assistantContextKey = `${moduleKey}:${activeTargetType}:${selectedTarget?.id || ''}`

  function resetAssistantContext() {
    setSessionID('')
    setProposals([])
    setSkillIntent('')
    setSkillIntentKey('')
    pendingSkillRunRef.current = null
  }

  function handleModuleChange(nextModuleID: string) {
    setModuleID(nextModuleID)
    setTargetValue('')
    resetAssistantContext()
  }

  function handleTargetChange(nextTargetValue: string) {
    setTargetValue(nextTargetValue)
    resetAssistantContext()
  }

  function handleSessionCreated(nextSessionID: string) {
    setSessionID(nextSessionID)
    setProposals([])
    const pending = pendingSkillRunRef.current
    if (!pending) return
    pendingSkillRunRef.current = null
    setBusy(true)
    runAssistantSkill(token, pending.skillID, {
      module_key: pending.moduleKey,
      target_type: pending.targetType,
      target_id: pending.targetID,
      session_id: nextSessionID,
    })
      .then(() => setNotice(t('assistant.global.skillRunCreated')))
      .catch((err) => setError(err instanceof Error ? err.message : t('common.operationFailed')))
      .finally(() => setBusy(false))
  }

  useEffect(() => {
    let cancelled = false
    Promise.all([listAssistantContextTargets(token, moduleKey, moduleTargetType), listAssistantSkills(token, moduleKey, moduleTargetType)])
      .then(([targetItems, skillItems]) => {
        if (cancelled) return
        setTargets(targetItems)
        setSkills(skillItems)
        setSelectedSkillID((current) => (skillItems.some((skill) => skill.id === current) ? current : skillItems[0]?.id || ''))
      })
      .catch((err) => {
        if (!cancelled) setError(err instanceof Error ? err.message : t('assistant.global.loadFailed'))
      })
    return () => {
      cancelled = true
    }
  }, [moduleKey, moduleTargetType, t, token])

  useEffect(() => {
    if (!sessionID) return
    let cancelled = false
    const timers = [0, 2200, 5200, 8200].map((delay) =>
      window.setTimeout(() => {
        listAssistantProposals(token, sessionID)
          .then((items) => {
            if (!cancelled) setProposals(items)
          })
          .catch(() => {
            if (!cancelled) setProposals([])
          })
      }, delay),
    )
    return () => {
      cancelled = true
      timers.forEach((timer) => window.clearTimeout(timer))
    }
  }, [sessionID, token])

  async function refreshProposals() {
    if (!sessionID) return
    setBusy(true)
    setError('')
    try {
      setProposals(await listAssistantProposals(token, sessionID))
    } catch (err) {
      setError(err instanceof Error ? err.message : t('assistant.global.loadFailed'))
    } finally {
      setBusy(false)
    }
  }

  async function handleProposal(id: string, decision: 'confirm' | 'reject') {
    setBusy(true)
    setError('')
    setNotice('')
    try {
      if (decision === 'confirm') {
        await confirmAssistantProposal(token, id)
        setNotice(t('assistant.global.proposalConfirmed'))
      } else {
        await rejectAssistantProposal(token, id, 'rejected from global business interaction')
        setNotice(t('assistant.global.proposalRejected'))
      }
      await refreshProposals()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('common.operationFailed'))
    } finally {
      setBusy(false)
    }
  }

  async function handleCreateSkill() {
    if (!skillName.trim() || !skillPrompt.trim()) return
    setBusy(true)
    setError('')
    setNotice('')
    try {
      const skill = await createAssistantSkill(token, {
        module_key: moduleKey,
        target_type: activeTargetType,
        name: skillName.trim(),
        prompt_template: skillPrompt.trim(),
        source_session_id: sessionID || undefined,
        metadata: {
          source: 'global_business_interaction',
          target_id: selectedTarget?.id || '',
        },
      })
      setSkills((current) => [skill, ...current])
      setSelectedSkillID(skill.id)
      setSkillName('')
      setSkillPrompt('')
      setNotice(t('assistant.global.skillSaved'))
    } catch (err) {
      setError(err instanceof Error ? err.message : t('common.operationFailed'))
    } finally {
      setBusy(false)
    }
  }

  async function handleActivateSkill(id: string) {
    setBusy(true)
    setError('')
    try {
      const updated = await activateAssistantSkill(token, id)
      setSkills((current) => current.map((skill) => (skill.id === id ? updated : skill)))
      setNotice(t('assistant.global.skillActivated'))
    } catch (err) {
      setError(err instanceof Error ? err.message : t('common.operationFailed'))
    } finally {
      setBusy(false)
    }
  }

  async function handleRunSkill() {
    if (!selectedSkillID) return
    const skill = selectedSkill
    if (!skill || skill.status !== 'active') return
    setBusy(true)
    setError('')
    setNotice('')
    pendingSkillRunRef.current = {
      skillID: selectedSkillID,
      moduleKey,
      targetType: activeTargetType,
      targetID: selectedTarget?.id || '',
    }
    setSkillIntent(
      [
        skill.prompt_template,
        '',
        `${t('assistant.global.module')}: ${t(activeModule.label)}`,
        selectedTarget
          ? `${t('assistant.global.target')}: ${selectedTarget.type} ${selectedTarget.id} ${selectedTarget.title}`
          : `${t('assistant.global.target')}: ${t('assistant.global.noTarget')}`,
      ].join('\n'),
    )
    setSkillIntentKey(`${skill.id}-${Date.now()}`)
    setBusy(false)
  }

  return (
    <section className="rounded-lg border border-slate-200 bg-white shadow-sm">
      <div className="grid gap-0 xl:grid-cols-[1fr_360px]">
        <div className="min-h-[560px]">
          <AIAssistant
            key={assistantContextKey}
            token={token}
            contextType={moduleKey}
            targetType={activeTargetType}
            targetID={selectedTarget?.id}
            autoModel
            hideModelSelector
            initialIntent={skillIntent}
            initialIntentKey={skillIntentKey}
            autoRunInitialIntent
            className="min-h-[560px] rounded-l-lg"
            onSessionCreated={handleSessionCreated}
          />
        </div>
        <aside className="border-t border-slate-200 p-4 xl:border-l xl:border-t-0">
          <div className="space-y-3">
            <div>
              <label className="text-xs font-semibold text-slate-500">{t('assistant.global.module')}</label>
              <select
                value={moduleID}
                onChange={(event) => handleModuleChange(event.target.value)}
                className="mt-1 h-10 w-full rounded-lg border border-slate-300 bg-white px-3 text-sm text-slate-900 outline-none focus:border-[#AD4714] focus:ring-2 focus:ring-[#DF6A24]/20"
              >
                {globalAssistantModules.map((item) => (
                  <option key={item.id} value={item.id}>
                    {t(item.label)}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="text-xs font-semibold text-slate-500">{t('assistant.global.target')}</label>
              <select
                value={targetValue}
                onChange={(event) => handleTargetChange(event.target.value)}
                className="mt-1 h-10 w-full rounded-lg border border-slate-300 bg-white px-3 text-sm text-slate-900 outline-none focus:border-[#AD4714] focus:ring-2 focus:ring-[#DF6A24]/20"
              >
                <option value="">{t('assistant.global.noTarget')}</option>
                {targets.map((target) => (
                  <option key={`${target.type}:${target.id}`} value={`${target.type}:${target.id}`}>
                    {target.type} · {target.title || target.id}
                  </option>
                ))}
              </select>
            </div>

            <div className="rounded-lg border border-slate-200 bg-slate-50 p-3">
              <div className="flex items-center justify-between gap-2">
                <h3 className="text-sm font-semibold text-slate-950">{t('assistant.global.proposals')}</h3>
                <button
                  type="button"
                  onClick={() => void refreshProposals()}
                  disabled={!sessionID || busy}
                  className="inline-flex h-8 items-center rounded-md border border-slate-300 px-2 text-xs font-semibold text-slate-700 transition hover:bg-white disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {t('common.refresh')}
                </button>
              </div>
              <div className="mt-3 space-y-2">
                {proposals.length > 0 ? (
                  proposals.map((proposal) => (
                    <div key={proposal.id} className="rounded-md border border-slate-200 bg-white p-3">
                      <div className="flex items-start justify-between gap-2">
                        <p className="line-clamp-2 text-sm font-semibold text-slate-900">{proposal.title || proposal.summary}</p>
                        <StatusPill label={proposal.status} tone={proposal.status === 'applied' ? 'green' : 'blue'} />
                      </div>
                      <p className="mt-2 line-clamp-3 text-xs text-slate-500">{proposal.summary}</p>
                      {proposal.status === 'pending' && (
                        <div className="mt-3 flex flex-wrap gap-2">
                          <button
                            type="button"
                            onClick={() => void handleProposal(proposal.id, 'confirm')}
                            disabled={busy}
                            className="inline-flex h-8 items-center rounded-md bg-emerald-600 px-2.5 text-xs font-semibold text-white transition hover:bg-emerald-700 disabled:opacity-50"
                          >
                            {t('assistant.global.confirmWriteback')}
                          </button>
                          <button
                            type="button"
                            onClick={() => void handleProposal(proposal.id, 'reject')}
                            disabled={busy}
                            className="inline-flex h-8 items-center rounded-md border border-red-200 bg-red-50 px-2.5 text-xs font-semibold text-red-700 transition hover:bg-red-100 disabled:opacity-50"
                          >
                            {t('assistant.global.rejectProposal')}
                          </button>
                        </div>
                      )}
                    </div>
                  ))
                ) : (
                  <p className="text-sm text-slate-500">{t('assistant.global.noProposals')}</p>
                )}
              </div>
            </div>

            <div className="rounded-lg border border-slate-200 bg-slate-50 p-3">
              <h3 className="text-sm font-semibold text-slate-950">{t('assistant.global.skills')}</h3>
              <div className="mt-3 space-y-2">
                <select
                  value={selectedSkillID}
                  onChange={(event) => setSelectedSkillID(event.target.value)}
                  className="h-10 w-full rounded-lg border border-slate-300 bg-white px-3 text-sm text-slate-900 outline-none focus:border-[#AD4714] focus:ring-2 focus:ring-[#DF6A24]/20"
                >
                  <option value="">{t('assistant.global.noSkill')}</option>
                  {skills.map((skill) => (
                    <option key={skill.id} value={skill.id}>
                      {skill.name} · {t(skill.status)}
                    </option>
                  ))}
                </select>
                <div className="flex flex-wrap gap-2">
                  <button
                    type="button"
                    onClick={() => void handleRunSkill()}
                    disabled={!selectedSkillID || selectedSkill?.status !== 'active' || busy}
                    className="inline-flex h-8 items-center rounded-md bg-[#AD4714] px-2.5 text-xs font-semibold text-[#fffaf5] transition hover:bg-[#B84F18] disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    {t('assistant.global.runSkill')}
                  </button>
                  {selectedSkillID && selectedSkill?.status !== 'active' && (
                    <button
                      type="button"
                      onClick={() => void handleActivateSkill(selectedSkillID)}
                      disabled={busy}
                      className="inline-flex h-8 items-center rounded-md border border-slate-300 px-2.5 text-xs font-semibold text-slate-700 transition hover:bg-white disabled:opacity-50"
                    >
                      {t('assistant.global.activateSkill')}
                    </button>
                  )}
                </div>
              </div>

              <div className="mt-4 space-y-2">
                <input
                  value={skillName}
                  onChange={(event) => setSkillName(event.target.value)}
                  placeholder={t('assistant.global.skillName')}
                  className="h-10 w-full rounded-lg border border-slate-300 px-3 text-sm text-slate-900 outline-none focus:border-[#AD4714] focus:ring-2 focus:ring-[#DF6A24]/20"
                />
                <textarea
                  value={skillPrompt}
                  onChange={(event) => setSkillPrompt(event.target.value)}
                  placeholder={t('assistant.global.skillPrompt')}
                  className="min-h-[84px] w-full resize-none rounded-lg border border-slate-300 p-3 text-sm text-slate-900 outline-none focus:border-[#AD4714] focus:ring-2 focus:ring-[#DF6A24]/20"
                />
                <button
                  type="button"
                  onClick={() => void handleCreateSkill()}
                  disabled={!skillName.trim() || !skillPrompt.trim() || busy}
                  className="inline-flex h-9 w-full items-center justify-center rounded-md border border-slate-300 bg-white px-3 text-sm font-semibold text-slate-800 transition hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {t('assistant.global.saveSkill')}
                </button>
              </div>
            </div>

            {(notice || error) && (
              <p className={`rounded-md px-3 py-2 text-sm ${error ? 'bg-red-50 text-red-700' : 'bg-emerald-50 text-emerald-700'}`}>
                {error || notice}
              </p>
            )}
          </div>
        </aside>
      </div>
    </section>
  )
}

function RoleDirectory({ roles }: { roles: Role[] }) {
  const { t } = useI18n()
  return (
    <section className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
      <div className="flex items-center gap-2">
        <CheckCircle2 className="h-5 w-5 text-slate-500" />
        <h2 className="text-base font-semibold text-slate-950">{t('角色目录')}</h2>
      </div>
      <div className="mt-5 space-y-3">
        {roles.length > 0 ? (
          roles.map((role) => (
            <div key={role.id} className="border-t border-slate-100 py-3 first:border-t-0 first:pt-0">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="text-sm font-semibold text-slate-900">{role.name}</p>
                  {role.description && <p className="mt-1 text-sm text-slate-500">{role.description}</p>}
                </div>
                <StatusPill label={role.role_type} tone="blue" />
              </div>
            </div>
          ))
        ) : (
          <p className="text-sm text-slate-500">{t('角色目录暂不可用')}</p>
        )}
      </div>
    </section>
  )
}

function MetricCard({
  icon: Icon,
  label,
  value,
  detail,
  tone,
}: {
  icon: typeof Users
  label: string
  value: string
  detail: string
  tone: 'blue' | 'emerald' | 'amber' | 'violet'
}) {
  const toneClass = {
    blue: 'bg-blue-50 text-blue-700',
    emerald: 'bg-emerald-50 text-emerald-700',
    amber: 'bg-amber-50 text-amber-700',
    violet: 'bg-violet-50 text-violet-700',
  }[tone]

  return (
    <article className="min-h-[142px] rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
      <div className={`flex h-10 w-10 items-center justify-center rounded-lg ${toneClass}`}>
        <Icon className="h-5 w-5" />
      </div>
      <p className="mt-4 text-sm font-medium text-slate-500">{label}</p>
      <div className="mt-1 flex items-end justify-between gap-3">
        <p className="text-3xl font-semibold text-slate-950">{value}</p>
        <p className="pb-1 text-right text-sm text-slate-500">{detail}</p>
      </div>
    </article>
  )
}

function StatusBars({
  title,
  icon: Icon,
  data,
  labels,
}: {
  title: string
  icon: typeof Users
  data: Record<string, number>
  labels: Record<string, string>
}) {
  const { t } = useI18n()
  const entries = Object.entries(labels)
  const total = Math.max(
    entries.reduce((sum, [key]) => sum + (data[key] ?? 0), 0),
    1,
  )

  return (
    <section className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
      <div className="flex items-center gap-2">
        <Icon className="h-5 w-5 text-slate-500" />
        <h2 className="text-base font-semibold text-slate-950">{title}</h2>
      </div>
      <div className="mt-5 space-y-4">
        {entries.map(([key, label]) => {
          const value = data[key] ?? 0
          const width = `${Math.max((value / total) * 100, value > 0 ? 3 : 0)}%`

          return (
            <div key={key}>
              <div className="mb-1 flex items-center justify-between text-sm">
                <span className="font-medium text-slate-700">{t(label)}</span>
                <span className="text-slate-500">{formatNumber(value)}</span>
              </div>
              <div className="h-2 rounded-full bg-slate-100">
                <div className="h-2 rounded-full bg-slate-700" style={{ width }} />
              </div>
            </div>
          )
        })}
      </div>
    </section>
  )
}

function SignalStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="border-l-2 border-slate-200 py-1 pl-3">
      <p className="text-xs font-medium text-slate-500">{label}</p>
      <p className="mt-2 text-xl font-semibold text-slate-950">{value}</p>
    </div>
  )
}

function RecentEvents({ events }: { events: MetaOrgOverview['activity'] }) {
  const { t } = useI18n()
  return (
    <section className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
      <div className="flex items-center gap-2">
        <Activity className="h-5 w-5 text-slate-500" />
        <h2 className="text-base font-semibold text-slate-950">{t('近期事件')}</h2>
      </div>
      <div className="mt-5 divide-y divide-slate-100">
        {events.length > 0 ? (
          events.map((event) => (
            <div key={`${event.type}-${event.id}`} className="grid gap-2 py-3 sm:grid-cols-[140px_1fr_auto]">
              <span className="text-sm text-slate-500">{formatDate(event.created_at)}</span>
              <span className="min-w-0 truncate text-sm font-medium text-slate-900">{event.title}</span>
              <StatusPill label={event.status || event.type} tone="blue" />
            </div>
          ))
        ) : (
          <p className="py-3 text-sm text-slate-500">{t('暂无事件')}</p>
        )}
      </div>
    </section>
  )
}

function StatusPill({ label, tone }: { label: string; tone: 'blue' | 'green' | 'amber' }) {
  const { t } = useI18n()
  const toneClass = {
    blue: 'border-blue-200 bg-blue-50 text-blue-700',
    green: 'border-emerald-200 bg-emerald-50 text-emerald-700',
    amber: 'border-amber-200 bg-amber-50 text-amber-700',
  }[tone]

  return (
    <span
      className={`inline-flex h-7 max-w-[180px] items-center truncate rounded-full border px-2.5 text-xs font-semibold ${toneClass}`}
    >
      {t(label)}
    </span>
  )
}
