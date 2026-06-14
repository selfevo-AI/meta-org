'use client'

import { Banknote, FileWarning, RefreshCw, Send, ServerCog, TableProperties } from 'lucide-react'
import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import {
  createFinanceAdapter,
  createFinanceExportBatch,
  getFinanceExportBatch,
  listFinanceAdapters,
  listFinanceExportBatches,
  listFinanceReconciliation,
  submitFinanceExportBatch,
  testFinanceAdapter,
  type FinanceAdapter,
  type FinanceExportBatch,
  type FinanceReconciliationItem,
} from '@/lib/api'
import { useI18n } from '@/lib/i18n'

interface FinanceWorkspaceProps {
  token: string
}

type TabID = 'adapters' | 'batches' | 'reconciliation' | 'failed'

const tabs: Array<{ id: TabID; label: string; icon: typeof ServerCog }> = [
  { id: 'adapters', label: 'finance.adapters', icon: ServerCog },
  { id: 'batches', label: 'finance.exportBatches', icon: Banknote },
  { id: 'reconciliation', label: 'finance.reconciliation', icon: TableProperties },
  { id: 'failed', label: 'finance.failedWebhooks', icon: FileWarning },
]

function money(value: number | undefined, currency = 'CNY'): string {
  return `${currency} ${Number(value ?? 0).toFixed(4)}`
}

function dateOnly(value?: string): string {
  if (!value) return ''
  return new Date(value).toISOString().slice(0, 10)
}

export function FinanceWorkspace({ token }: FinanceWorkspaceProps) {
  const { t } = useI18n()
  const [activeTab, setActiveTab] = useState<TabID>('adapters')
  const [adapters, setAdapters] = useState<FinanceAdapter[]>([])
  const [batches, setBatches] = useState<FinanceExportBatch[]>([])
  const [selectedBatch, setSelectedBatch] = useState<FinanceExportBatch | null>(null)
  const [reconciliation, setReconciliation] = useState<FinanceReconciliationItem[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [adapterForm, setAdapterForm] = useState({
    name: '',
    endpoint_url: '',
    auth_type: 'hmac' as 'hmac' | 'bearer',
    secret: '',
    timeout_ms: '30000',
    retry_count: '3',
  })
  const [batchForm, setBatchForm] = useState({
    adapter_id: '',
    period_start: new Date().toISOString().slice(0, 10),
    period_end: new Date().toISOString().slice(0, 10),
    currency: 'CNY',
  })

  const failedItems = useMemo(
    () => batches.filter((batch) => batch.status === 'failed' || batch.error_message),
    [batches],
  )

  const loadFinance = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [adapterData, batchData, reconciliationData] = await Promise.all([
        listFinanceAdapters(token),
        listFinanceExportBatches(token),
        listFinanceReconciliation(token),
      ])
      setAdapters(adapterData)
      setBatches(batchData)
      setReconciliation(reconciliationData)
      setBatchForm((current) => ({ ...current, adapter_id: current.adapter_id || adapterData[0]?.id || '' }))
    } catch (err) {
      setError(err instanceof Error ? err.message : t('finance.loadFailed'))
    } finally {
      setLoading(false)
    }
  }, [t, token])

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void loadFinance()
    }, 0)
    return () => window.clearTimeout(timer)
  }, [loadFinance])

  async function run(action: () => Promise<void>, success: string) {
    setLoading(true)
    setError('')
    setNotice('')
    try {
      await action()
      setNotice(t(success))
      await loadFinance()
    } catch (err) {
      setError(err instanceof Error ? err.message : t('common.operationFailed'))
    } finally {
      setLoading(false)
    }
  }

  async function submitAdapter(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    await run(
      () =>
        createFinanceAdapter(token, {
          name: adapterForm.name,
          endpoint_url: adapterForm.endpoint_url,
          auth_type: adapterForm.auth_type,
          secret: adapterForm.secret,
          timeout_ms: Number(adapterForm.timeout_ms || 30000),
          retry_count: Number(adapterForm.retry_count || 3),
          metadata: {},
        }).then(() => setAdapterForm((current) => ({ ...current, secret: '' }))),
      'finance.adapterCreated',
    )
  }

  async function submitBatch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    await run(
      () =>
        createFinanceExportBatch(token, {
          adapter_id: batchForm.adapter_id,
          period_start: batchForm.period_start,
          period_end: batchForm.period_end,
          currency: batchForm.currency,
          metadata: {},
        }).then((batch) => setSelectedBatch(batch)),
      'finance.batchCreated',
    )
  }

  async function openBatch(id: string) {
    await run(() => getFinanceExportBatch(token, id).then((batch) => setSelectedBatch(batch)), 'finance.batchLoaded')
  }

  async function submitBatchToAdapter(id: string) {
    await run(() => submitFinanceExportBatch(token, id).then((batch) => setSelectedBatch(batch)), 'finance.batchSubmitted')
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
          onClick={() => void loadFinance()}
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

      {activeTab === 'adapters' && (
        <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
          <Panel title="finance.adapters">
            <Table
              headers={['common.name', 'finance.endpoint', 'finance.authType', 'developer.status']}
              rows={adapters.map((adapter) => [adapter.name, adapter.endpoint_url, t(adapter.auth_type), t(adapter.status)])}
            />
          </Panel>
          <Panel title="finance.adapterSettings">
            <form className="space-y-3" onSubmit={submitAdapter}>
              <TextInput label="common.name" value={adapterForm.name} onChange={(value) => setAdapterForm({ ...adapterForm, name: value })} />
              <TextInput
                label="finance.endpoint"
                value={adapterForm.endpoint_url}
                onChange={(value) => setAdapterForm({ ...adapterForm, endpoint_url: value })}
              />
              <SelectInput
                label="finance.authType"
                value={adapterForm.auth_type}
                onChange={(value) => setAdapterForm({ ...adapterForm, auth_type: value as 'hmac' | 'bearer' })}
                options={['hmac', 'bearer']}
              />
              <TextInput
                label="finance.secret"
                type="password"
                value={adapterForm.secret}
                onChange={(value) => setAdapterForm({ ...adapterForm, secret: value })}
              />
              <div className="grid gap-3 sm:grid-cols-2">
                <TextInput
                  label="developer.timeout"
                  value={adapterForm.timeout_ms}
                  onChange={(value) => setAdapterForm({ ...adapterForm, timeout_ms: value })}
                />
                <TextInput
                  label="developer.retries"
                  value={adapterForm.retry_count}
                  onChange={(value) => setAdapterForm({ ...adapterForm, retry_count: value })}
                />
              </div>
              <SubmitButton loading={loading} label="finance.createAdapter" />
            </form>
            <div className="mt-5 space-y-2 border-t border-slate-100 pt-4">
              {adapters.map((adapter) => (
                <button
                  key={adapter.id}
                  type="button"
                  onClick={() => void run(() => testFinanceAdapter(token, adapter.id).then(() => undefined), 'finance.adapterTested')}
                  className="inline-flex h-9 w-full items-center justify-between rounded-md border border-slate-300 px-3 text-sm font-semibold text-slate-700 hover:bg-slate-100"
                >
                  <span className="truncate">{adapter.name}</span>
                  <span>{t('finance.testAdapter')}</span>
                </button>
              ))}
            </div>
          </Panel>
        </div>
      )}

      {activeTab === 'batches' && (
        <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
          <Panel title="finance.exportBatches">
            <Table
              headers={[
                'finance.period',
                'developer.status',
                'finance.amount',
                'finance.currency',
                'finance.externalBatch',
                'finance.failureReason',
              ]}
              rows={batches.map((batch) => [
                `${dateOnly(batch.period_start)} - ${dateOnly(batch.period_end)}`,
                t(batch.status),
                money(batch.total_amount, batch.currency),
                batch.currency,
                batch.external_batch_id || t('common.none'),
                batch.error_message || t('common.none'),
              ])}
              actions={batches.map((batch) => (
                <div key={batch.id} className="flex gap-2">
                  <button
                    type="button"
                    onClick={() => void openBatch(batch.id)}
                    className="h-8 rounded-md border border-slate-300 px-2 text-xs font-semibold text-slate-700 hover:bg-slate-100"
                  >
                    {t('finance.details')}
                  </button>
                  <button
                    type="button"
                    onClick={() => void submitBatchToAdapter(batch.id)}
                    className="inline-flex h-8 items-center gap-1 rounded-md bg-slate-950 px-2 text-xs font-semibold text-white hover:bg-slate-800"
                  >
                    <Send className="h-3.5 w-3.5" />
                    {t('finance.submit')}
                  </button>
                </div>
              ))}
            />
            {selectedBatch && (
              <div className="mt-5 rounded-lg border border-slate-200 bg-slate-50 p-3">
                <p className="text-sm font-semibold text-slate-950">{t('finance.batchDetails')}</p>
                <Table
                  headers={['finance.line', 'finance.amount', 'developer.status', 'finance.projectCostEntry']}
                  rows={(selectedBatch.lines ?? []).map((line) => [
                    line.id,
                    money(line.amount, line.currency),
                    t(line.status),
                    line.project_cost_entry_id || t('common.none'),
                  ])}
                />
              </div>
            )}
          </Panel>
          <Panel title="finance.createBatch">
            <form className="space-y-3" onSubmit={submitBatch}>
              <SelectInput
                label="finance.adapter"
                value={batchForm.adapter_id}
                onChange={(value) => setBatchForm({ ...batchForm, adapter_id: value })}
                options={adapters.map((adapter) => adapter.id)}
                labels={Object.fromEntries(adapters.map((adapter) => [adapter.id, adapter.name]))}
              />
              <TextInput
                label="finance.periodStart"
                type="date"
                value={batchForm.period_start}
                onChange={(value) => setBatchForm({ ...batchForm, period_start: value })}
              />
              <TextInput
                label="finance.periodEnd"
                type="date"
                value={batchForm.period_end}
                onChange={(value) => setBatchForm({ ...batchForm, period_end: value })}
              />
              <TextInput
                label="finance.currency"
                value={batchForm.currency}
                onChange={(value) => setBatchForm({ ...batchForm, currency: value })}
              />
              <SubmitButton loading={loading} label="finance.createBatch" />
            </form>
          </Panel>
        </div>
      )}

      {activeTab === 'reconciliation' && (
        <Panel title="finance.reconciliation">
          <Table
            headers={['finance.batch', 'developer.status', 'finance.amount', 'finance.difference']}
            rows={reconciliation.map((item) => [
              item.batch_id,
              t(item.status),
              money(item.total_amount, item.currency),
              money(item.difference_amount, item.currency),
            ])}
          />
        </Panel>
      )}

      {activeTab === 'failed' && (
        <Panel title="finance.failedWebhooks">
          <Table
            headers={['finance.batch', 'developer.status', 'finance.failureReason', 'finance.updatedAt']}
            rows={failedItems.map((batch) => [
              batch.id,
              t(batch.status),
              batch.error_message || t('finance.webhookFailurePending'),
              new Date(batch.updated_at).toLocaleString(),
            ])}
          />
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
            {labels[option] ?? t(option)}
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

function Table({
  headers,
  rows,
  actions = [],
}: {
  headers: string[]
  rows: string[][]
  actions?: ReactNode[]
}) {
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
            {actions.length > 0 && <th className="px-3 py-2 text-left text-xs font-semibold text-slate-500">{t('common.action')}</th>}
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
              {actions.length > 0 && <td className="px-3 py-2">{actions[index]}</td>}
            </tr>
          ))}
          {rows.length === 0 && (
            <tr>
              <td className="px-3 py-4 text-sm text-slate-500" colSpan={headers.length + (actions.length > 0 ? 1 : 0)}>
                {t('common.noData')}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  )
}
