'use client'

import {
  Activity,
  Bot,
  BrainCircuit,
  ChevronDown,
  ChevronRight,
  CheckCircle2,
  FolderKanban,
  Gauge,
  GitBranch,
  GripVertical,
  KeyRound,
  LogOut,
  PanelLeft,
  RefreshCw,
  ShieldCheck,
  SlidersHorizontal,
  Users,
  Workflow,
} from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import type { DragEvent, FormEvent, ReactNode } from 'react'
import { getMetaOrgInbox, getMetaOrgOverview, listRoles, login, registerUser } from '@/lib/api'
import type { InboxItem, MetaOrgOverview, Role } from '@/lib/api'
import { clearSession, getSessionUser, getToken, setSession } from '@/lib/auth'
import { useI18n } from '@/lib/i18n'
import { apiOperations, operationDomains } from '@/lib/operations'
import { ApiWorkbench } from './api-workbench'
import { AIAssistant } from './ai-assistant'
import {
  CapabilityEvaluationWorkspace,
  GovernanceWorkspace,
  WeightWorkspace,
  WorkflowDesignerWorkspace,
  WorkflowMatchingWorkspace,
} from './control-workspaces'
import { DeveloperToolsWorkspace } from './developer-tools-workspace'
import { FinanceWorkspace } from './finance-workspace'
import { OrganizationWorkspace } from './organization-workspace'
import { ProjectLifecycleWorkspace } from './project-lifecycle-workspace'

type AuthMode = 'login' | 'register'
type WorkspaceView = 'overview' | `domain:${string}`

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
  DeveloperTools: '开发者工具',
  Finance: '财务导出',
}

type MenuGroup = {
  id: string
  label: string
  domains: string[]
}

const lifecycleDomains = ['Requirement', 'Project', 'Delivery', 'Cost', 'Feedback']
const dedicatedDomains = new Set([
  'Organization',
  'Governance',
  'Evolution',
  'Capability',
  'Workflow',
  'DeveloperTools',
  'Finance',
  ...lifecycleDomains,
])
const menuStorageKey = 'meta_org.menu.groups.v1'
const expandedMenuStorageKey = 'meta_org.menu.expanded.v1'
const legacyMenuStorageKey = 'harness.menu.groups.v1'
const legacyExpandedMenuStorageKey = 'harness.menu.expanded.v1'

const defaultMenuGroups: MenuGroup[] = [
  {
    id: 'business',
    label: '业务闭环',
    domains: ['Requirement', 'Project', 'Delivery', 'Cost', 'Feedback'],
  },
  {
    id: 'organization',
    label: '组织能力',
    domains: ['Organization', 'Workflow', 'Capability'],
  },
  {
    id: 'governance',
    label: '治理演进',
    domains: ['Governance', 'Evolution', 'Verification'],
  },
  {
    id: 'system',
    label: '系统工具',
    domains: ['MetaOrg', 'Dashboard', 'DeveloperTools', 'Finance', 'Identity', 'Layer', 'Observability'],
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
  const knownDomains = new Set(operationDomains)
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

  operationDomains.forEach((domain) => {
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

export default function Home() {
  const { locale, setLocale, t } = useI18n()
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
  const [menuReady, setMenuReady] = useState(false)
  const [draggedDomain, setDraggedDomain] = useState<string | null>(null)

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
      setMenuReady(true)
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
    if (!domain || !operationDomains.includes(domain)) return
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

  const activeDomain = workspaceView === 'overview' ? 'MetaOrg' : workspaceView.replace('domain:', '')
  const activeGroup = menuGroups.find((group) => group.domains.includes(activeDomain))
  const activeOperationCount =
    workspaceView === 'overview'
      ? apiOperations.filter((operation) => operation.domain === 'MetaOrg').length
      : apiOperations.filter((operation) => operation.domain === activeDomain).length

  return (
    <main className="min-h-screen bg-slate-50">
      <header className="border-b border-slate-200 bg-white">
        <div className="mx-auto flex max-w-7xl flex-col gap-4 px-4 py-5 sm:px-6 lg:flex-row lg:items-center lg:justify-between lg:px-8">
          <div>
            <p className="text-sm font-medium text-slate-500">{t('app.product')}</p>
            <h1 className="mt-1 text-2xl font-semibold text-slate-950">{t('app.title')}</h1>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            {userType && <StatusPill label={userType === 'ai' ? 'AI Agent' : 'Human'} tone="blue" />}
            {overview && (
              <StatusPill
                label={`${t('common.refresh')} ${formatDate(overview.generated_at)}`}
                tone={overviewLoading ? 'amber' : 'green'}
              />
            )}
            <div className="inline-flex h-10 items-center rounded-lg border border-slate-300 bg-white p-1">
              <button
                type="button"
                onClick={() => setLocale('zh')}
                className={`h-8 rounded-md px-2.5 text-sm font-semibold transition ${
                  locale === 'zh' ? 'bg-slate-950 text-white' : 'text-slate-600 hover:bg-slate-100'
                }`}
              >
                {t('language.zh')}
              </button>
              <button
                type="button"
                onClick={() => setLocale('en')}
                className={`h-8 rounded-md px-2.5 text-sm font-semibold transition ${
                  locale === 'en' ? 'bg-slate-950 text-white' : 'text-slate-600 hover:bg-slate-100'
                }`}
              >
                {t('language.en')}
              </button>
            </div>
            {token && (
              <>
                <button
                  type="button"
                  onClick={() => loadOverview()}
                  className="inline-flex h-10 items-center gap-2 rounded-lg border border-slate-300 bg-white px-3 text-sm font-medium text-slate-700 transition hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-60"
                  disabled={overviewLoading}
                >
                  <RefreshCw className={`h-4 w-4 ${overviewLoading ? 'animate-spin' : ''}`} />
                  {t('common.refresh')}
                </button>
                <button
                  type="button"
                  onClick={handleSignOut}
                  className="inline-flex h-10 items-center gap-2 rounded-lg bg-slate-950 px-3 text-sm font-medium text-white transition hover:bg-slate-800"
                >
                  <LogOut className="h-4 w-4" />
                  {t('common.signOut')}
                </button>
              </>
            )}
          </div>
        </div>
      </header>

      <div className="mx-auto grid max-w-7xl gap-5 px-4 py-6 sm:px-6 lg:grid-cols-[320px_1fr] lg:px-8">
        {!token && (
          <section className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
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
                className="inline-flex h-11 w-full items-center justify-center gap-2 rounded-lg bg-slate-950 px-4 text-sm font-semibold text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-60"
              >
                <KeyRound className="h-4 w-4" />
                {loading ? t('auth.processing') : mode === 'login' ? t('auth.signIn') : t('auth.createAccount')}
              </button>
            </form>
          </section>
        )}

        {!token && <RoleDirectory roles={roles} />}

        {token && (
          <section className="grid gap-5 lg:col-span-2 lg:grid-cols-[280px_1fr]">
            <NavigationSidebar
              workspaceView={workspaceView}
              groups={menuGroups}
              expandedGroups={expandedGroups}
              onViewChange={setWorkspaceView}
              onToggleGroup={toggleMenuGroup}
              onDragStart={handleDomainDragStart}
              onDropDomain={handleDomainDrop}
              onReset={resetMenuLayout}
            />

            <div className="min-w-0 space-y-5">
              <WorkspaceHeader
                title={workspaceView === 'overview' ? '总览' : domainLabels[activeDomain] ?? activeDomain}
                domain={workspaceView === 'overview' ? 'Overview' : activeDomain}
                groupLabel={workspaceView === 'overview' ? '工作台' : activeGroup?.label ?? '功能台'}
                operationCount={activeOperationCount}
                dedicated={workspaceView === 'overview' || dedicatedDomains.has(activeDomain)}
              />
              {error && (
                <div className="mb-5 rounded-lg border border-red-200 bg-red-50 px-4 py-3 text-sm text-red-700">
                  {error}
                </div>
              )}
              {workspaceView === 'overview' ? (
                overview ? (
                  <Dashboard overview={overview} inbox={inbox} healthRatio={healthRatio} token={token} />
                ) : (
                  <div className="flex min-h-[420px] items-center justify-center rounded-lg border border-slate-200 bg-white">
                    <RefreshCw className="h-5 w-5 animate-spin text-slate-500" />
                  </div>
                )
              ) : workspaceView === 'domain:Organization' ? (
                <WorkspaceWithAssistant token={token} contextType="organization">
                  <OrganizationWorkspace token={token} currentUserId={userId} />
                </WorkspaceWithAssistant>
              ) : workspaceView === 'domain:Governance' ? (
                <WorkspaceWithAssistant token={token} contextType="governance">
                  <GovernanceWorkspace token={token} currentUserId={userId} />
                </WorkspaceWithAssistant>
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
                <WorkspaceWithAssistant
                  token={token}
                  contextType={workspaceView === 'domain:Requirement' ? 'requirement' : 'project'}
                >
                  <ProjectLifecycleWorkspace
                    token={token}
                    currentUserId={userId}
                    mode={workspaceView.replace('domain:', '') as 'Requirement' | 'Project' | 'Delivery' | 'Cost' | 'Feedback'}
                  />
                </WorkspaceWithAssistant>
              ) : workspaceView === 'domain:DeveloperTools' ? (
                <WorkspaceWithAssistant token={token} contextType="developer_tools">
                  <DeveloperToolsWorkspace token={token} />
                </WorkspaceWithAssistant>
              ) : workspaceView === 'domain:Finance' ? (
                <FinanceWorkspace token={token} />
              ) : (
                <ApiWorkbench
                  key={workspaceView}
                  token={token}
                  domain={workspaceView.replace('domain:', '')}
                  showDomainMenu={false}
                />
              )}
            </div>
          </section>
        )}
      </div>
    </main>
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
    <aside className="h-fit rounded-lg border border-slate-200 bg-white p-3 shadow-sm">
      <div className="flex items-center justify-between gap-2 px-1">
        <div className="flex items-center gap-2">
          <PanelLeft className="h-4 w-4 text-slate-500" />
          <p className="text-sm font-semibold text-slate-900">{t('nav.menu')}</p>
        </div>
        <button
          type="button"
          onClick={onReset}
          className="inline-flex h-8 items-center gap-1.5 rounded-md border border-slate-300 px-2 text-xs font-semibold text-slate-600 hover:bg-slate-100"
        >
          <SlidersHorizontal className="h-3.5 w-3.5" />
          {t('nav.reset')}
        </button>
      </div>

      <button
        type="button"
        onClick={() => onViewChange('overview')}
        className={`mt-3 flex h-11 w-full items-center justify-between rounded-lg px-3 text-left text-sm font-semibold transition ${
          workspaceView === 'overview'
            ? 'bg-slate-950 text-white'
            : 'text-slate-600 hover:bg-slate-100 hover:text-slate-950'
        }`}
      >
        <span className="inline-flex items-center gap-2">
          <Gauge className="h-4 w-4" />
          {t('nav.overview')}
        </span>
        <span className="text-xs opacity-70">Overview</span>
      </button>

      <div className="mt-4 space-y-2 border-t border-slate-100 pt-4">
        {groups.map((group) => {
          const expanded = expandedGroups[group.id] ?? true
          const groupOperations = group.domains.reduce(
            (sum, domain) => sum + apiOperations.filter((operation) => operation.domain === domain).length,
            0,
          )

          return (
            <div
              key={group.id}
              className={`rounded-lg border transition ${
                expanded ? 'border-slate-200 bg-white' : 'border-slate-100 bg-slate-50'
              }`}
              onDragOver={(event) => event.preventDefault()}
              onDrop={(event) => onDropDomain(event, group.id)}
            >
              <button
                type="button"
                onClick={() => onToggleGroup(group.id)}
                className="flex h-10 w-full items-center justify-between px-3 text-left text-sm font-semibold text-slate-700 hover:text-slate-950"
              >
                <span className="inline-flex min-w-0 items-center gap-2">
                  {expanded ? <ChevronDown className="h-4 w-4 shrink-0" /> : <ChevronRight className="h-4 w-4 shrink-0" />}
                  <FolderKanban className="h-4 w-4 shrink-0 text-slate-500" />
                  <span className="truncate">{t(group.label)}</span>
                </span>
                <span className="text-xs font-medium text-slate-400">{groupOperations}</span>
              </button>

              {expanded && (
                <div className="space-y-1 px-2 pb-2">
                  {group.domains.map((domain) => {
                    const menuKey = `domain:${domain}` as const
                    const count = apiOperations.filter((operation) => operation.domain === domain).length

                    return (
                      <button
                        key={domain}
                        type="button"
                        draggable
                        onDragStart={(event) => onDragStart(event, domain)}
                        onClick={() => onViewChange(menuKey)}
                        className={`group flex h-10 w-full items-center justify-between gap-2 rounded-lg px-2.5 text-left text-sm font-medium transition ${
                          workspaceView === menuKey
                            ? 'bg-slate-950 text-white'
                            : 'text-slate-600 hover:bg-slate-100 hover:text-slate-950'
                        }`}
                      >
                        <span className="inline-flex min-w-0 items-center gap-2">
                          <GripVertical className="h-4 w-4 shrink-0 opacity-50" />
                          <span className="truncate">{t(domainLabels[domain] ?? domain)}</span>
                        </span>
                        <span className="shrink-0 text-xs opacity-70">{count}</span>
                      </button>
                    )
                  })}
                  {group.domains.length === 0 && <p className="px-3 py-2 text-sm text-slate-400">{t('nav.empty')}</p>}
                </div>
              )}
            </div>
          )
        })}
      </div>
    </aside>
  )
}

function WorkspaceHeader({
  title,
  domain,
  groupLabel,
  operationCount,
  dedicated,
}: {
  title: string
  domain: string
  groupLabel: string
  operationCount: number
  dedicated: boolean
}) {
  const { t } = useI18n()
  return (
    <section className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0">
          <p className="text-xs font-semibold uppercase tracking-normal text-slate-400">{t(groupLabel)}</p>
          <h2 className="mt-1 truncate text-xl font-semibold text-slate-950">{t(title)}</h2>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span className="inline-flex h-8 items-center rounded-md border border-slate-200 bg-slate-50 px-2.5 text-xs font-semibold text-slate-600">
            {domain}
          </span>
          <span className="inline-flex h-8 items-center rounded-md border border-slate-200 bg-white px-2.5 text-xs font-semibold text-slate-600">
            {operationCount} API
          </span>
          <span
            className={`inline-flex h-8 items-center rounded-md border px-2.5 text-xs font-semibold ${
              dedicated ? 'border-emerald-200 bg-emerald-50 text-emerald-700' : 'border-blue-200 bg-blue-50 text-blue-700'
            }`}
          >
            {dedicated ? t('workspace.workspace') : t('workspace.api')}
          </span>
        </div>
      </div>
    </section>
  )
}

function WorkspaceWithAssistant({
  token,
  contextType,
  children,
}: {
  token: string
  contextType: 'requirement' | 'project' | 'organization' | 'governance' | 'developer_tools'
  children: ReactNode
}) {
  return (
    <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
      <div className="min-w-0">{children}</div>
      <AIAssistant token={token} contextType={contextType} />
    </div>
  )
}

function Dashboard({
  overview,
  inbox,
  healthRatio,
  token,
}: {
  overview: MetaOrgOverview
  inbox: InboxItem[]
  healthRatio: number
  token: string
}) {
  const { t } = useI18n()
  const agentCoverage = overview.agents.total > 0 ? overview.agents.active / overview.agents.total : 0

  return (
    <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
      <div className="min-w-0 space-y-5">
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
                  <div className="flex gap-2">
                    <StatusPill label={item.priority} tone={item.priority === 'high' || item.priority === 'critical' ? 'amber' : 'blue'} />
                    <StatusPill label={item.status} tone="green" />
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

      <AIAssistant token={token} contextType="meta_org" />
    </div>
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
