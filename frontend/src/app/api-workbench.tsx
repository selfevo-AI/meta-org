'use client'

import { AlertCircle, CheckCircle2, Play, RefreshCw } from 'lucide-react'
import { FormEvent, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import { apiRequest } from '@/lib/api'
import { useI18n } from '@/lib/i18n'
import { ApiOperation, apiOperations, operationDomains } from '@/lib/operations'

interface OperationFormState {
  path: Record<string, string>
  query: Record<string, string>
  body: string
}

interface ApiWorkbenchProps {
  token: string
  domain?: string
  showDomainMenu?: boolean
}

const methodTone: Record<ApiOperation['method'], string> = {
  GET: 'border-emerald-200 bg-emerald-50 text-emerald-700',
  POST: 'border-blue-200 bg-blue-50 text-blue-700',
  PUT: 'border-amber-200 bg-amber-50 text-amber-700',
  PATCH: 'border-violet-200 bg-violet-50 text-violet-700',
  DELETE: 'border-red-200 bg-red-50 text-red-700',
}

function createFormState(operation: ApiOperation): OperationFormState {
  return {
    path: Object.fromEntries((operation.pathParams ?? []).map((field) => [field.name, ''])),
    query: Object.fromEntries((operation.queryParams ?? []).map((field) => [field.name, ''])),
    body:
      operation.bodyTemplate === undefined
        ? ''
        : JSON.stringify(operation.bodyTemplate, null, 2),
  }
}

function formatJSON(value: unknown): string {
  return JSON.stringify(value, null, 2)
}

export function ApiWorkbench({ token, domain, showDomainMenu = true }: ApiWorkbenchProps) {
  const { t } = useI18n()
  const firstOperation = domain
    ? apiOperations.find((operation) => operation.domain === domain) ?? apiOperations[0]
    : apiOperations[0]
  const [activeDomain, setActiveDomain] = useState(firstOperation.domain)
  const [selectedOperation, setSelectedOperation] = useState<ApiOperation>(firstOperation)
  const [formState, setFormState] = useState<OperationFormState>(() => createFormState(firstOperation))
  const [response, setResponse] = useState<string>('')
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)

  const domainOperations = useMemo(
    () => apiOperations.filter((operation) => operation.domain === activeDomain),
    [activeDomain],
  )

  const requestPath = useMemo(
    () => buildRequestPath(selectedOperation, formState),
    [selectedOperation, formState],
  )

  function selectDomain(domain: string) {
    const nextOperation = apiOperations.find((operation) => operation.domain === domain) ?? firstOperation
    setActiveDomain(domain)
    setSelectedOperation(nextOperation)
    setFormState(createFormState(nextOperation))
    setResponse('')
    setError(null)
  }

  function selectOperation(operation: ApiOperation) {
    setSelectedOperation(operation)
    setFormState(createFormState(operation))
    setResponse('')
    setError(null)
  }

  function updatePathValue(name: string, value: string) {
    setFormState((current) => ({
      ...current,
      path: { ...current.path, [name]: value },
    }))
  }

  function updateQueryValue(name: string, value: string) {
    setFormState((current) => ({
      ...current,
      query: { ...current.query, [name]: value },
    }))
  }

  async function submitOperation(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setLoading(true)
    setError(null)
    setResponse('')

    try {
      const body = parseBody(selectedOperation, formState.body)
      const result = await apiRequest<unknown>(requestPath, {
        method: selectedOperation.method,
        token: selectedOperation.auth === false ? undefined : token,
        body,
      })
      setResponse(formatJSON(result))
    } catch (err) {
      setError(err instanceof Error ? err.message : t('请求失败'))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className={`grid gap-5 ${showDomainMenu ? 'lg:grid-cols-[220px_280px_1fr]' : 'lg:grid-cols-[280px_1fr]'}`}>
      {showDomainMenu && (
        <aside className="rounded-lg border border-slate-200 bg-white p-3 shadow-sm">
          <div className="space-y-1">
            {operationDomains.map((domain) => (
              <button
                key={domain}
                type="button"
                onClick={() => selectDomain(domain)}
                className={`flex h-10 w-full items-center justify-between rounded-lg px-3 text-left text-sm font-medium transition ${
                  activeDomain === domain
                    ? 'bg-slate-950 text-white'
                    : 'text-slate-600 hover:bg-slate-100 hover:text-slate-950'
                }`}
              >
                <span>{t(domain)}</span>
                <span className="text-xs opacity-70">
                  {apiOperations.filter((operation) => operation.domain === domain).length}
                </span>
              </button>
            ))}
          </div>
        </aside>
      )}

      <section className="rounded-lg border border-slate-200 bg-white p-3 shadow-sm">
        <div className="space-y-2">
          {domainOperations.map((operation) => (
            <button
              key={operation.id}
              type="button"
              onClick={() => selectOperation(operation)}
              className={`w-full rounded-lg border p-3 text-left transition ${
                selectedOperation.id === operation.id
                  ? 'border-slate-950 bg-slate-50'
                  : 'border-slate-200 hover:border-slate-300 hover:bg-slate-50'
              }`}
            >
              <div className="flex items-center justify-between gap-2">
                <span className="min-w-0 truncate text-sm font-semibold text-slate-950">{t(operation.title)}</span>
                <MethodBadge method={operation.method} />
              </div>
              <p className="mt-2 truncate text-xs text-slate-500">{operation.path}</p>
            </button>
          ))}
        </div>
      </section>

      <section className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
        <div className="flex flex-col gap-3 border-b border-slate-100 pb-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <div className="flex items-center gap-2">
              <MethodBadge method={selectedOperation.method} />
              <h2 className="text-base font-semibold text-slate-950">{t(selectedOperation.title)}</h2>
            </div>
            <p className="mt-2 break-all text-sm text-slate-500">{requestPath}</p>
          </div>
          {selectedOperation.auth === false ? (
            <StatusBadge label={t('common.public')} tone="blue" />
          ) : (
            <StatusBadge label={t('common.jwt')} tone="green" />
          )}
        </div>

        <form className="mt-5 space-y-5" onSubmit={submitOperation}>
          {selectedOperation.pathParams && selectedOperation.pathParams.length > 0 && (
            <FieldGroup title="路径参数">
              {selectedOperation.pathParams.map((field) => (
                <TextInput
                  key={field.name}
                  field={field}
                  value={formState.path[field.name] ?? ''}
                  onChange={(value) => updatePathValue(field.name, value)}
                />
              ))}
            </FieldGroup>
          )}

          {selectedOperation.queryParams && selectedOperation.queryParams.length > 0 && (
            <FieldGroup title="查询参数">
              {selectedOperation.queryParams.map((field) => (
                <TextInput
                  key={field.name}
                  field={field}
                  value={formState.query[field.name] ?? ''}
                  onChange={(value) => updateQueryValue(field.name, value)}
                />
              ))}
            </FieldGroup>
          )}

          {selectedOperation.bodyTemplate !== undefined && (
            <div>
              <label className="text-sm font-medium text-slate-700" htmlFor="operation-body">
                JSON Body
              </label>
              <textarea
                id="operation-body"
                value={formState.body}
                onChange={(event) => setFormState((current) => ({ ...current, body: event.target.value }))}
                className="mt-2 h-64 w-full resize-y rounded-lg border border-slate-300 bg-slate-950 p-3 font-mono text-sm text-slate-50 outline-none transition focus:border-slate-500 focus:ring-2 focus:ring-slate-200"
                spellCheck={false}
              />
            </div>
          )}

          {error && (
            <div className="flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
              <AlertCircle className="mt-0.5 h-4 w-4 flex-none" />
              <span>{error}</span>
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="inline-flex h-11 items-center justify-center gap-2 rounded-lg bg-slate-950 px-4 text-sm font-semibold text-white transition hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-60"
          >
            {loading ? <RefreshCw className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
            {t('执行')}
          </button>
        </form>

        <div className="mt-6">
          <div className="mb-2 flex items-center gap-2">
            <CheckCircle2 className="h-4 w-4 text-slate-500" />
            <h3 className="text-sm font-semibold text-slate-950">{t('响应')}</h3>
          </div>
          <pre className="min-h-[180px] overflow-auto rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm text-slate-800">
            {response || t('common.emptyResponse')}
          </pre>
        </div>
      </section>
    </div>
  )
}

function buildRequestPath(operation: ApiOperation, state: OperationFormState): string {
  let path = operation.path
  for (const [name, value] of Object.entries(state.path)) {
    path = path.replace(`{${name}}`, encodeURIComponent(value.trim()))
  }

  const query = new URLSearchParams()
  for (const [name, value] of Object.entries(state.query)) {
    const trimmed = value.trim()
    if (trimmed !== '') {
      query.set(name, trimmed)
    }
  }

  const queryString = query.toString()
  return queryString ? `${path}?${queryString}` : path
}

function parseBody(operation: ApiOperation, bodyText: string): unknown {
  if (operation.bodyTemplate === undefined) {
    return undefined
  }
  if (bodyText.trim() === '') {
    return {}
  }
  return JSON.parse(bodyText)
}

function FieldGroup({ title, children }: { title: string; children: ReactNode }) {
  const { t } = useI18n()
  return (
    <div>
      <h3 className="text-sm font-semibold text-slate-950">{t(title)}</h3>
      <div className="mt-3 grid gap-3 sm:grid-cols-2">{children}</div>
    </div>
  )
}

function TextInput({
  field,
  value,
  onChange,
}: {
  field: { name: string; label: string; placeholder?: string }
  value: string
  onChange: (value: string) => void
}) {
  const { t } = useI18n()
  return (
    <label className="block">
      <span className="text-sm font-medium text-slate-700">{t(field.label)}</span>
      <input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder={field.placeholder ? t(field.placeholder) : undefined}
        className="mt-1 h-10 w-full rounded-lg border border-slate-300 px-3 text-sm outline-none transition focus:border-slate-500 focus:ring-2 focus:ring-slate-200"
      />
    </label>
  )
}

function MethodBadge({ method }: { method: ApiOperation['method'] }) {
  return (
    <span className={`inline-flex h-6 items-center rounded-md border px-2 text-xs font-semibold ${methodTone[method]}`}>
      {method}
    </span>
  )
}

function StatusBadge({ label, tone }: { label: string; tone: 'blue' | 'green' }) {
  const toneClass = {
    blue: 'border-blue-200 bg-blue-50 text-blue-700',
    green: 'border-emerald-200 bg-emerald-50 text-emerald-700',
  }[tone]

  return (
    <span className={`inline-flex h-7 items-center rounded-md border px-2.5 text-xs font-semibold ${toneClass}`}>
      {label}
    </span>
  )
}
