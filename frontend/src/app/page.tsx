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
import type { DragEvent, FormEvent } from 'react'
import {
  getDashboardOverview,
  listRoles,
  login,
  registerUser,
} from '@/lib/api'
import type { DashboardOverview, Role } from '@/lib/api'
import { clearSession, getSessionUser, getToken, setSession } from '@/lib/auth'
import { useI18n } from '@/lib/i18n'
import { apiOperations, operationDomains } from '@/lib/operations'
import { ApiWorkbench } from './api-workbench'
import {
  CapabilityEvaluationWorkspace,
  GovernanceWorkspace,
  WeightWorkspace,
  WorkflowDesignerWorkspace,
  WorkflowMatchingWorkspace,
} from './control-workspaces'
import { OrganizationWorkspace } from './organization-workspace'
import { ProjectLifecycleWorkspace } from './project-lifecycle-workspace'

type AuthMode = 'login' | 'register'
type WorkspaceView = 'overview' | `domain:${string}`

const domainLabels: Record<string, string> = {
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
}

type MenuGroup = {
  id: string
  label: string
  domains: string[]
}

const lifecycleDomains = ['Requirement', 'Project', 'Delivery', 'Cost', 'Feedback']
const dedicatedDomains = new Set(['Organization', 'Governance', 'Evolution', 'Capability', 'Workflow', ...lifecycleDomains])
const menuStorageKey = 'harness.menu.groups.v1'
const expandedMenuStorageKey = 'harness.menu.expanded.v1'

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
    domains: ['Dashboard', 'Identity', 'Layer', 'Observability'],
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
    const raw = window.localStorage.getItem(menuStorageKey)
    if (!raw) return normalizeMenuGroups()
    return normalizeMenuGroups(JSON.parse(raw))
  } catch {
    return normalizeMenuGroups()
  }
}

function loadExpandedGroups(): Record<string, boolean> {
  if (typeof window === 'undefined') return defaultExpandedGroups()
  try {
    const raw = window.localStorage.getItem(expandedMenuStorageKey)
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
  const [overview, setOverview] = useState<DashboardOverview | null>(null)
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

    getDashboardOverview(token)
      .then((data) => {
        if (!cancelled) setOverview(data)
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
    const active = overview.workflow.instances_by_status.active ?? 0
    const total = Math.max(overview.workflow.instances, 1)
    return active / total
  }, [overview])

  async function loadOverview(activeToken = token) {
    if (!activeToken) return
    setOverviewLoading(true)
    setError(null)
    try {
      const data = await getDashboardOverview(activeToken)
      setOverview(data)
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

  const activeDomain = workspaceView === 'overview' ? 'Dashboard' : workspaceView.replace('domain:', '')
  const activeGroup = menuGroups.find((group) => group.domains.includes(activeDomain))
  const activeOperationCount =
    workspaceView === 'overview'
      ? apiOperations.filter((operation) => operation.domain === 'Dashboard').length
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
                  <Dashboard overview={overview} healthRatio={healthRatio} />
                ) : (
                  <div className="flex min-h-[420px] items-center justify-center rounded-lg border border-slate-200 bg-white">
                    <RefreshCw className="h-5 w-5 animate-spin text-slate-500" />
                  </div>
                )
              ) : workspaceView === 'domain:Organization' ? (
                <OrganizationWorkspace token={token} currentUserId={userId} />
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
                />
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

function Dashboard({ overview, healthRatio }: { overview: DashboardOverview; healthRatio: number }) {
  const { t } = useI18n()
  const agentCoverage =
    overview.identity.total_agents > 0 ? overview.identity.active_agents / overview.identity.total_agents : 0

  return (
    <div className="space-y-5">
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <MetricCard
          icon={Users}
          label={t('组织成员')}
          value={formatNumber(overview.identity.users + overview.identity.total_agents)}
          detail={`${formatNumber(overview.identity.active_agents)} ${t('个活跃 Agent')}`}
          tone="blue"
        />
        <MetricCard
          icon={GitBranch}
          label={t('MVRU 单元')}
          value={formatNumber(overview.organization.mvrus)}
          detail={`${formatNumber(overview.organization.relationships)} ${t('条关系')}`}
          tone="emerald"
        />
        <MetricCard
          icon={Workflow}
          label={t('工作流实例')}
          value={formatNumber(overview.workflow.instances)}
          detail={`${formatPercent(healthRatio)} active`}
          tone="amber"
        />
        <MetricCard
          icon={ShieldCheck}
          label={t('治理覆盖')}
          value={formatNumber(overview.governance.active_principles)}
          detail={`${formatNumber(overview.governance.active_control_rules)} ${t('条规则启用')}`}
          tone="violet"
        />
      </div>

      <div className="grid gap-4 xl:grid-cols-3">
        <StatusBars
          title={t('MVRU 状态')}
          icon={GitBranch}
          data={overview.organization.mvrus_by_status}
          labels={{
            designing: '设计中',
            active: '运行中',
            evaluating: '评估中',
            evolving: '演进中',
            dissolved: '已解散',
          }}
        />
        <StatusBars
          title={t('任务队列')}
          icon={Activity}
          data={overview.workflow.tasks_by_status}
          labels={{
            pending: '待处理',
            assigned: '已分配',
            in_progress: '执行中',
            completed: '已完成',
            rejected: '已拒绝',
          }}
        />
        <StatusBars
          title={t('实验状态')}
          icon={BrainCircuit}
          data={overview.evolution.experiments_by_status}
          labels={{
            proposed: '提议',
            running: '运行',
            completed: '完成',
            failed: '失败',
          }}
        />
      </div>

      <div className="grid gap-4 xl:grid-cols-[1.2fr_0.8fr]">
        <section className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
          <div className="flex items-center justify-between gap-3">
            <div className="flex items-center gap-2">
              <Gauge className="h-5 w-5 text-slate-500" />
              <h2 className="text-base font-semibold text-slate-950">{t('运行信号')}</h2>
            </div>
            <StatusPill
              label={overview.evolution.high_priority_signals > 0 ? t('高优先级') : t('稳定')}
              tone={overview.evolution.high_priority_signals > 0 ? 'amber' : 'green'}
            />
          </div>

          <div className="mt-5 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            <SignalStat label={t('能力调用 24h')} value={formatCompact(overview.capability.invocations_24h)} />
            <SignalStat label={t('失败调用 24h')} value={formatCompact(overview.capability.failed_invocations_24h)} />
            <SignalStat label={t('平均耗时')} value={`${Math.round(overview.capability.average_duration_ms)}ms`} />
            <SignalStat label={t('观测 Span 24h')} value={formatCompact(overview.observability.spans_24h)} />
            <SignalStat label={t('验证均分')} value={overview.verification.average_score.toFixed(2)} />
            <SignalStat label={t('未确认信号')} value={formatCompact(overview.evolution.unacknowledged_signals)} />
          </div>
        </section>

        <section className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
          <div className="flex items-center gap-2">
            <Bot className="h-5 w-5 text-slate-500" />
            <h2 className="text-base font-semibold text-slate-950">{t('Agent 覆盖')}</h2>
          </div>
          <div className="mt-5">
            <div className="flex items-end justify-between">
              <span className="text-3xl font-semibold text-slate-950">{formatPercent(agentCoverage)}</span>
              <span className="text-sm text-slate-500">
                {formatNumber(overview.identity.active_agents)} / {formatNumber(overview.identity.total_agents)}
              </span>
            </div>
            <div className="mt-3 h-2 rounded-full bg-slate-100">
              <div
                className="h-2 rounded-full bg-emerald-500"
                style={{ width: `${Math.min(agentCoverage * 100, 100)}%` }}
              />
            </div>
          </div>
          <div className="mt-5 grid grid-cols-2 gap-3">
            <SignalStat label={t('能力库')} value={formatNumber(overview.capability.active_capabilities)} />
            <SignalStat label={t('能力绑定')} value={formatNumber(overview.capability.bindings)} />
          </div>
        </section>
      </div>

      <RecentEvents events={overview.recent_events} />
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

function RecentEvents({ events }: { events: DashboardOverview['recent_events'] }) {
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
