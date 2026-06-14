'use client'

import { Activity, Bot, FileCode2, KeyRound, ListChecks, RefreshCw, ServerCog, Wrench } from 'lucide-react'
import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import {
  createModel,
  createModelProvider,
  getAICostSummary,
  listInvocations,
  listModelProviders,
  listModels,
  listToolExecutions,
  listTools,
  rotateModelProviderKey,
  testModelProvider,
  type AICostSummary,
  type AIInvocation,
  type ModelCatalogItem,
  type ModelProvider,
  type ToolDefinition,
  type ToolExecution,
} from '@/lib/api'
import { useI18n } from '@/lib/i18n'

interface DeveloperToolsWorkspaceProps {
  token: string
}

type TabID = 'providers' | 'models' | 'tools' | 'interfaces' | 'invocations' | 'cost'

const tabs: Array<{ id: TabID; label: string; icon: typeof ServerCog }> = [
  { id: 'providers', label: 'developer.providers', icon: ServerCog },
  { id: 'models', label: 'developer.models', icon: Bot },
  { id: 'tools', label: 'developer.tools', icon: Wrench },
  { id: 'interfaces', label: 'developer.interfaces', icon: FileCode2 },
  { id: 'invocations', label: 'developer.invocations', icon: Activity },
  { id: 'cost', label: 'developer.cost', icon: ListChecks },
]

function money(value: number | undefined, currency = 'CNY'): string {
  return `${currency} ${Number(value ?? 0).toFixed(4)}`
}

export function DeveloperToolsWorkspace({ token }: DeveloperToolsWorkspaceProps) {
  const { t } = useI18n()
  const [activeTab, setActiveTab] = useState<TabID>('providers')
  const [providers, setProviders] = useState<ModelProvider[]>([])
  const [models, setModels] = useState<ModelCatalogItem[]>([])
  const [tools, setTools] = useState<ToolDefinition[]>([])
  const [executions, setExecutions] = useState<ToolExecution[]>([])
  const [invocations, setInvocations] = useState<AIInvocation[]>([])
  const [cost, setCost] = useState<AICostSummary | null>(null)
  const [selectedProviderID, setSelectedProviderID] = useState('')
  const [notice, setNotice] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [providerForm, setProviderForm] = useState({
    name: '',
    provider_type: 'openai' as 'openai' | 'anthropic' | 'gemini',
    base_url: '',
    api_key: '',
    risk_level: 'medium',
    timeout_ms: '60000',
    retry_count: '1',
    tags: '',
  })
  const [modelForm, setModelForm] = useState({
    provider_id: '',
    model_key: '',
    display_name: '',
    context_window: '',
    max_output_tokens: '',
    input_price_per_1k: '',
    output_price_per_1k: '',
    currency: 'CNY',
    capabilities: '',
  })
  const [secretInput, setSecretInput] = useState('')
  const [testModel, setTestModel] = useState('')

  const selectedProvider = useMemo(
    () => providers.find((provider) => provider.id === selectedProviderID) ?? providers[0],
    [providers, selectedProviderID],
  )

  const loadAll = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [providerData, modelData, toolData, executionData, invocationData, costData] = await Promise.all([
        listModelProviders(token),
        listModels(token),
        listTools(token),
        listToolExecutions(token),
        listInvocations(token),
        getAICostSummary(token),
      ])
      setProviders(providerData)
      setModels(modelData)
      setTools(toolData)
      setExecutions(executionData)
      setInvocations(invocationData)
      setCost(costData)
      setSelectedProviderID((current) => current || providerData[0]?.id || '')
      setModelForm((current) => ({ ...current, provider_id: current.provider_id || providerData[0]?.id || '' }))
    } catch (err) {
      setError(err instanceof Error ? err.message : t('developer.loadFailed'))
    } finally {
      setLoading(false)
    }
  }, [t, token])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadAll()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadAll])

  async function run(action: () => Promise<void>, success: string) {
    setLoading(true)
    setError('')
    setNotice('')
    try {
      await action()
      setNotice(t(success))
      await loadAll()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('common.operationFailed'))
    } finally {
      setLoading(false)
    }
  }

  async function submitProvider(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    await run(
      () =>
        createModelProvider(token, {
          name: providerForm.name,
          provider_type: providerForm.provider_type,
          base_url: providerForm.base_url || undefined,
          api_key: providerForm.api_key,
          risk_level: providerForm.risk_level,
          timeout_ms: Number(providerForm.timeout_ms || 60000),
          retry_count: Number(providerForm.retry_count || 1),
          tags: splitCsv(providerForm.tags),
          metadata: {},
        }).then(() => setProviderForm((current) => ({ ...current, api_key: '' }))),
      'developer.providerCreated',
    )
  }

  async function submitModel(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    await run(
      () =>
        createModel(token, {
          provider_id: modelForm.provider_id,
          model_key: modelForm.model_key,
          display_name: modelForm.display_name,
          context_window: Number(modelForm.context_window || 0),
          max_output_tokens: Number(modelForm.max_output_tokens || 0),
          input_price_per_1k: Number(modelForm.input_price_per_1k || 0),
          output_price_per_1k: Number(modelForm.output_price_per_1k || 0),
          currency: modelForm.currency,
          capabilities: splitCsv(modelForm.capabilities),
          metadata: {},
        }).then(() => setModelForm((current) => ({ ...current, model_key: '', display_name: '' }))),
      'developer.modelCreated',
    )
  }

  async function rotateKey() {
    if (!selectedProvider || !secretInput) return
    await run(
      () => rotateModelProviderKey(token, selectedProvider.id, secretInput).then(() => setSecretInput('')),
      'developer.keyRotated',
    )
  }

  async function testProvider() {
    if (!selectedProvider) return
    await run(() => testModelProvider(token, selectedProvider.id, testModel || undefined).then(() => undefined), 'developer.providerTested')
  }

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap gap-2 rounded-lg border border-slate-200 bg-white p-2 shadow-sm">
        {tabs.map((tab) => {
          const Icon = tab.icon
          return (
            <button
              key={tab.id}
              type="button"
              onClick={() => setActiveTab(tab.id)}
              className={`inline-flex h-10 items-center gap-2 rounded-md px-3 text-sm font-semibold transition ${
                activeTab === tab.id ? 'bg-slate-950 text-white' : 'text-slate-600 hover:bg-slate-100'
              }`}
            >
              <Icon className="h-4 w-4" />
              {t(tab.label)}
            </button>
          )
        })}
        <button
          type="button"
          onClick={() => void loadAll()}
          disabled={loading}
          className="ml-auto inline-flex h-10 items-center gap-2 rounded-md border border-slate-300 px-3 text-sm font-semibold text-slate-700 hover:bg-slate-100 disabled:opacity-50"
        >
          <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
          {t('common.refresh')}
        </button>
      </div>

      {(error || notice) && (
        <div
          className={`rounded-lg border px-3 py-2 text-sm ${
            error ? 'border-red-200 bg-red-50 text-red-700' : 'border-emerald-200 bg-emerald-50 text-emerald-700'
          }`}
        >
          {error || notice}
        </div>
      )}

      {activeTab === 'providers' && (
        <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
          <Panel title="developer.modelProviders">
            <div className="divide-y divide-slate-100">
              {providers.map((provider) => (
                <button
                  key={provider.id}
                  type="button"
                  onClick={() => setSelectedProviderID(provider.id)}
                  className={`grid w-full gap-2 py-3 text-left md:grid-cols-[1fr_auto] ${
                    selectedProvider?.id === provider.id ? 'text-slate-950' : 'text-slate-700'
                  }`}
                >
                  <div className="min-w-0">
                    <p className="truncate text-sm font-semibold">{provider.name}</p>
                    <p className="mt-1 truncate text-xs text-slate-500">
                      {t(`provider.${provider.provider_type}`)} · {provider.masked_api_key || t('common.none')}
                    </p>
                  </div>
                  <StatusBadge label={provider.last_test_status || provider.status} />
                </button>
              ))}
              {providers.length === 0 && <EmptyText>{t('developer.noProviders')}</EmptyText>}
            </div>
          </Panel>

          <Panel title="developer.providerSettings">
            <form className="space-y-3" onSubmit={submitProvider}>
              <TextInput label="common.name" value={providerForm.name} onChange={(value) => setProviderForm({ ...providerForm, name: value })} />
              <SelectInput
                label="developer.providerType"
                value={providerForm.provider_type}
                onChange={(value) => setProviderForm({ ...providerForm, provider_type: value as 'openai' | 'anthropic' | 'gemini' })}
                options={['openai', 'anthropic', 'gemini']}
              />
              <TextInput
                label="developer.baseUrl"
                value={providerForm.base_url}
                onChange={(value) => setProviderForm({ ...providerForm, base_url: value })}
              />
              <TextInput
                label="developer.apiKey"
                type="password"
                value={providerForm.api_key}
                onChange={(value) => setProviderForm({ ...providerForm, api_key: value })}
              />
              <div className="grid gap-3 sm:grid-cols-2">
                <TextInput
                  label="developer.timeout"
                  value={providerForm.timeout_ms}
                  onChange={(value) => setProviderForm({ ...providerForm, timeout_ms: value })}
                />
                <TextInput
                  label="developer.retries"
                  value={providerForm.retry_count}
                  onChange={(value) => setProviderForm({ ...providerForm, retry_count: value })}
                />
              </div>
              <TextInput label="developer.tags" value={providerForm.tags} onChange={(value) => setProviderForm({ ...providerForm, tags: value })} />
              <SubmitButton loading={loading} label="developer.createProvider" />
            </form>

            <div className="mt-5 border-t border-slate-100 pt-4">
              <p className="text-sm font-semibold text-slate-950">{t('developer.keyRotation')}</p>
              <TextInput label="developer.newKey" type="password" value={secretInput} onChange={setSecretInput} />
              <button
                type="button"
                onClick={() => void rotateKey()}
                disabled={!selectedProvider || !secretInput || loading}
                className="mt-3 inline-flex h-10 w-full items-center justify-center gap-2 rounded-lg border border-slate-300 px-3 text-sm font-semibold text-slate-700 hover:bg-slate-100 disabled:opacity-50"
              >
                <KeyRound className="h-4 w-4" />
                {t('developer.rotateKey')}
              </button>
              <TextInput label="developer.testModel" value={testModel} onChange={setTestModel} />
              <button
                type="button"
                onClick={() => void testProvider()}
                disabled={!selectedProvider || loading}
                className="mt-3 inline-flex h-10 w-full items-center justify-center rounded-lg bg-slate-950 px-3 text-sm font-semibold text-white hover:bg-slate-800 disabled:opacity-50"
              >
                {t('developer.testProvider')}
              </button>
            </div>
          </Panel>
        </div>
      )}

      {activeTab === 'models' && (
        <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
          <Panel title="developer.modelCatalog">
            <Table
              headers={['developer.model', 'developer.provider', 'developer.status', 'developer.context']}
              rows={models.map((model) => [
                model.display_name || model.model_key,
                providers.find((provider) => provider.id === model.provider_id)?.name || model.provider_id,
                t(model.status),
                String(model.context_window || 0),
              ])}
            />
          </Panel>
          <Panel title="developer.createModel">
            <form className="space-y-3" onSubmit={submitModel}>
              <SelectInput
                label="developer.provider"
                value={modelForm.provider_id}
                onChange={(value) => setModelForm({ ...modelForm, provider_id: value })}
                options={providers.map((provider) => provider.id)}
                labels={Object.fromEntries(providers.map((provider) => [provider.id, provider.name]))}
              />
              <TextInput label="developer.modelKey" value={modelForm.model_key} onChange={(value) => setModelForm({ ...modelForm, model_key: value })} />
              <TextInput
                label="developer.displayName"
                value={modelForm.display_name}
                onChange={(value) => setModelForm({ ...modelForm, display_name: value })}
              />
              <div className="grid gap-3 sm:grid-cols-2">
                <TextInput
                  label="developer.context"
                  value={modelForm.context_window}
                  onChange={(value) => setModelForm({ ...modelForm, context_window: value })}
                />
                <TextInput
                  label="developer.maxOutput"
                  value={modelForm.max_output_tokens}
                  onChange={(value) => setModelForm({ ...modelForm, max_output_tokens: value })}
                />
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                <TextInput
                  label="developer.inputPrice"
                  value={modelForm.input_price_per_1k}
                  onChange={(value) => setModelForm({ ...modelForm, input_price_per_1k: value })}
                />
                <TextInput
                  label="developer.outputPrice"
                  value={modelForm.output_price_per_1k}
                  onChange={(value) => setModelForm({ ...modelForm, output_price_per_1k: value })}
                />
              </div>
              <TextInput
                label="developer.capabilities"
                value={modelForm.capabilities}
                onChange={(value) => setModelForm({ ...modelForm, capabilities: value })}
              />
              <SubmitButton loading={loading} label="developer.createModel" />
            </form>
          </Panel>
        </div>
      )}

      {activeTab === 'tools' && (
        <Panel title="developer.toolRegistry">
          <Table
            headers={['developer.tool', 'developer.policy', 'developer.risk', 'developer.status']}
            rows={tools.map((tool) => [tool.name, t(tool.default_policy), t(tool.risk_level), t(tool.status)])}
          />
          <div className="mt-5">
            <p className="text-sm font-semibold text-slate-950">{t('developer.recentExecutions')}</p>
            <Table
              headers={['developer.tool', 'developer.actor', 'developer.status', 'developer.error']}
              rows={executions.map((execution) => [
                execution.tool_name || execution.tool_id,
                `${execution.actor_type}:${execution.actor_id}`,
                t(execution.status),
                execution.error_message || t('common.none'),
              ])}
            />
          </div>
        </Panel>
      )}

      {activeTab === 'interfaces' && (
        <Panel title="developer.interfaceFiles">
          <div className="grid gap-4 lg:grid-cols-2">
            <JsonTemplate title="developer.providerContract" value={providerContract()} />
            <JsonTemplate title="developer.toolContract" value={toolContract()} />
          </div>
        </Panel>
      )}

      {activeTab === 'invocations' && (
        <Panel title="developer.invocationLogs">
          <Table
            headers={['developer.invocation', 'developer.mode', 'developer.status', 'developer.cost']}
            rows={invocations.map((invocation) => [
              invocation.id,
              t(invocation.mode),
              t(invocation.status),
              money(invocation.cost_amount, invocation.currency),
            ])}
          />
        </Panel>
      )}

      {activeTab === 'cost' && (
        <Panel title="developer.costSummary">
          <div className="grid gap-3 sm:grid-cols-3">
            <Metric label="developer.totalCost" value={money(cost?.total, cost?.currency)} />
            <Metric label="developer.unexportedCost" value={money(cost?.unexported, cost?.currency)} />
            <Metric label="developer.providerCount" value={String(Object.keys(cost?.by_provider ?? {}).length)} />
          </div>
          <div className="mt-5 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {Object.entries(cost?.by_provider ?? {}).map(([provider, amount]) => (
              <Metric key={provider} label={`provider.${provider}`} value={money(amount, cost?.currency)} />
            ))}
          </div>
        </Panel>
      )}
    </div>
  )
}

function Panel({ title, children }: { title: string; children: ReactNode }) {
  const { t } = useI18n()
  return (
    <section className="rounded-lg border border-slate-200 bg-white p-5 shadow-sm">
      <h2 className="text-base font-semibold text-slate-950">{t(title)}</h2>
      <div className="mt-4">{children}</div>
    </section>
  )
}

function TextInput({
  label,
  value,
  onChange,
  type = 'text',
}: {
  label: string
  value: string
  onChange: (value: string) => void
  type?: string
}) {
  const { t } = useI18n()
  return (
    <label className="block">
      <span className="text-xs font-semibold text-slate-500">{t(label)}</span>
      <input
        type={type}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="mt-1 h-10 w-full rounded-lg border border-slate-300 px-3 text-sm outline-none focus:border-slate-500 focus:ring-2 focus:ring-slate-200"
      />
    </label>
  )
}

function SelectInput({
  label,
  value,
  onChange,
  options,
  labels = {},
}: {
  label: string
  value: string
  onChange: (value: string) => void
  options: string[]
  labels?: Record<string, string>
}) {
  const { t } = useI18n()
  return (
    <label className="block">
      <span className="text-xs font-semibold text-slate-500">{t(label)}</span>
      <select
        value={value}
        onChange={(event) => onChange(event.target.value)}
        className="mt-1 h-10 w-full rounded-lg border border-slate-300 bg-white px-3 text-sm outline-none focus:border-slate-500 focus:ring-2 focus:ring-slate-200"
      >
        {options.map((option) => (
          <option key={option} value={option}>
            {labels[option] ?? t(`provider.${option}`)}
          </option>
        ))}
      </select>
    </label>
  )
}

function SubmitButton({ loading, label }: { loading: boolean; label: string }) {
  const { t } = useI18n()
  return (
    <button
      type="submit"
      disabled={loading}
      className="inline-flex h-10 w-full items-center justify-center rounded-lg bg-slate-950 px-3 text-sm font-semibold text-white hover:bg-slate-800 disabled:opacity-50"
    >
      {t(label)}
    </button>
  )
}

function Table({ headers, rows }: { headers: string[]; rows: string[][] }) {
  const { t } = useI18n()
  return (
    <div className="overflow-x-auto rounded-lg border border-slate-200">
      <table className="min-w-full divide-y divide-slate-200 text-sm">
        <thead className="bg-slate-50">
          <tr>
            {headers.map((header) => (
              <th key={header} className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-normal text-slate-500">
                {t(header)}
              </th>
            ))}
          </tr>
        </thead>
        <tbody className="divide-y divide-slate-100 bg-white">
          {rows.map((row, index) => (
            <tr key={`${row[0]}-${index}`}>
              {row.map((cell, cellIndex) => (
                <td key={`${cell}-${cellIndex}`} className="max-w-[260px] truncate px-3 py-2 text-slate-700">
                  {cell}
                </td>
              ))}
            </tr>
          ))}
          {rows.length === 0 && (
            <tr>
              <td className="px-3 py-4 text-sm text-slate-500" colSpan={headers.length}>
                {t('common.noData')}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  )
}

function JsonTemplate({ title, value }: { title: string; value: unknown }) {
  const { t } = useI18n()
  return (
    <div>
      <p className="text-sm font-semibold text-slate-950">{t(title)}</p>
      <pre className="mt-2 min-h-[220px] overflow-auto rounded-lg border border-slate-200 bg-slate-950 p-3 text-xs text-slate-50">
        {JSON.stringify(value, null, 2)}
      </pre>
    </div>
  )
}

function Metric({ label, value }: { label: string; value: string }) {
  const { t } = useI18n()
  return (
    <div className="rounded-lg border border-slate-200 bg-slate-50 p-4">
      <p className="text-xs font-semibold text-slate-500">{t(label)}</p>
      <p className="mt-2 text-xl font-semibold text-slate-950">{value}</p>
    </div>
  )
}

function StatusBadge({ label }: { label: string }) {
  const { t } = useI18n()
  return (
    <span className="inline-flex h-7 items-center rounded-md border border-slate-200 bg-slate-50 px-2 text-xs font-semibold text-slate-600">
      {t(label)}
    </span>
  )
}

function EmptyText({ children }: { children: ReactNode }) {
  return <p className="py-4 text-sm text-slate-500">{children}</p>
}

function splitCsv(value: string): string[] {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter(Boolean)
}

function providerContract() {
  return {
    format_version: 'meta-org.provider.v1',
    provider_type: 'openai | anthropic | gemini',
    auth: { type: 'api_key', encrypted_at_rest: true },
    streaming: { protocol: 'sse', events: ['lifecycle', 'delta', 'usage_update', 'error', 'done'] },
  }
}

function toolContract() {
  return {
    format_version: 'meta-org.tool.v1',
    policy: 'auto | notify | approve | deny',
    governance: { required_level: 'L1-L4', risk_level: 'low | medium | high | critical' },
    execution: { idempotency_key: 'required for mutating tools', audit: true },
  }
}
