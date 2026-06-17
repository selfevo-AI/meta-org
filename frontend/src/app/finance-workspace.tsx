'use client'

import { Banknote, FileWarning, RefreshCw, Send, ServerCog, TableProperties } from 'lucide-react'
import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import {
  allocateFinancePayment,
  createFinanceAdapter,
  createFinanceExportBatch,
  createFinancePayable,
  createFinancePayment,
  getFinanceExportBatch,
  importFinanceExpenseFile,
  importFinanceExpenses,
  listFinanceAdapters,
  listFinanceExportBatches,
  listFinanceImportBatches,
  listFinanceImportRecords,
  listFinancePayables,
  listFinancePayments,
  listFinanceReconciliation,
  pullFinanceExpenses,
  submitFinanceExportBatch,
  testFinanceAdapter,
  type FinanceAdapter,
  type FinanceExportBatch,
  type FinanceImportBatch,
  type FinanceImportRecord,
  type FinancePayable,
  type FinancePayment,
  type FinanceReconciliationItem,
} from '@/lib/api'
import { useI18n } from '@/lib/i18n'

interface FinanceWorkspaceProps {
  token: string
}

type TabID = 'ingestion' | 'imports' | 'payables' | 'payments' | 'adapters' | 'batches' | 'reconciliation' | 'failed'

const tabs: Array<{ id: TabID; label: string; icon: typeof ServerCog }> = [
  { id: 'ingestion', label: 'finance.ingestion', icon: TableProperties },
  { id: 'imports', label: 'finance.importRecords', icon: FileWarning },
  { id: 'payables', label: 'finance.payables', icon: Banknote },
  { id: 'payments', label: 'finance.payments', icon: Send },
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

function parseMapping(value: string): Record<string, string> {
  return Object.fromEntries(
    value
      .split('\n')
      .map((line) => line.trim())
      .filter(Boolean)
      .map((line) => {
        const [target, source] = line.split(':')
        return [target?.trim(), source?.trim() || target?.trim()]
      })
      .filter(([target]) => Boolean(target)),
  )
}

export function FinanceWorkspace({ token }: FinanceWorkspaceProps) {
  const { t } = useI18n()
  const [activeTab, setActiveTab] = useState<TabID>('adapters')
  const [adapters, setAdapters] = useState<FinanceAdapter[]>([])
  const [batches, setBatches] = useState<FinanceExportBatch[]>([])
  const [selectedBatch, setSelectedBatch] = useState<FinanceExportBatch | null>(null)
  const [reconciliation, setReconciliation] = useState<FinanceReconciliationItem[]>([])
  const [importBatches, setImportBatches] = useState<FinanceImportBatch[]>([])
  const [importRecords, setImportRecords] = useState<FinanceImportRecord[]>([])
  const [payables, setPayables] = useState<FinancePayable[]>([])
  const [payments, setPayments] = useState<FinancePayment[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [adapterForm, setAdapterForm] = useState({
    name: '',
    endpoint_url: '',
    auth_type: 'hmac' as 'hmac' | 'bearer',
    adapter_type: 'generic',
    direction: 'bidirectional',
    secret: '',
    timeout_ms: '30000',
    retry_count: '3',
    field_mapping: 'external_record_id:external_record_id\namount:amount\ncurrency:currency\noccurred_at:occurred_at\nexpense_type:expense_type',
  })
  const [batchForm, setBatchForm] = useState({
    adapter_id: '',
    period_start: new Date().toISOString().slice(0, 10),
    period_end: new Date().toISOString().slice(0, 10),
    currency: 'CNY',
  })
  const [expenseForm, setExpenseForm] = useState({
    adapter_id: '',
    external_record_id: '',
    expense_type: 'daily_expense',
    amount: '0',
    currency: 'CNY',
    occurred_at: new Date().toISOString().slice(0, 10),
    description: '',
    account_code: '',
    account_name: '',
    cost_center_code: '',
    vendor_name: '',
    employee_name: '',
    invoice_number: '',
    payment_status: 'unpaid',
  })
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [payableForm, setPayableForm] = useState({
    payable_type: 'expense',
    external_payable_id: '',
    invoice_number: '',
    vendor_name: '',
    employee_name: '',
    amount: '0',
    tax_amount: '0',
    currency: 'CNY',
    due_date: '',
  })
  const [paymentForm, setPaymentForm] = useState({
    payment_number: '',
    external_payment_id: '',
    payment_method: '',
    vendor_name: '',
    employee_name: '',
    amount: '0',
    currency: 'CNY',
    paid_at: new Date().toISOString().slice(0, 10),
  })
  const [allocationForm, setAllocationForm] = useState({
    payment_id: '',
    payable_id: '',
    amount: '0',
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
      const [importBatchData, importRecordData, payableData, paymentData] = await Promise.all([
        listFinanceImportBatches(token),
        listFinanceImportRecords(token),
        listFinancePayables(token),
        listFinancePayments(token),
      ])
      setAdapters(adapterData)
      setBatches(batchData)
      setReconciliation(reconciliationData)
      setImportBatches(importBatchData)
      setImportRecords(importRecordData)
      setPayables(payableData)
      setPayments(paymentData)
      setBatchForm((current) => ({ ...current, adapter_id: current.adapter_id || adapterData[0]?.id || '' }))
      setExpenseForm((current) => ({ ...current, adapter_id: current.adapter_id || adapterData[0]?.id || '' }))
      setAllocationForm((current) => ({ ...current, payment_id: current.payment_id || paymentData[0]?.id || '', payable_id: current.payable_id || payableData[0]?.id || '' }))
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
          adapter_type: adapterForm.adapter_type,
          direction: adapterForm.direction,
          secret: adapterForm.secret,
          timeout_ms: Number(adapterForm.timeout_ms || 30000),
          retry_count: Number(adapterForm.retry_count || 3),
          field_mapping: parseMapping(adapterForm.field_mapping),
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

  async function submitExpense(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    await run(
      () =>
        importFinanceExpenses(token, {
          adapter_id: expenseForm.adapter_id,
          source_type: 'api',
          records: [
            {
              external_record_id: expenseForm.external_record_id,
              expense_type: expenseForm.expense_type,
              amount: Number(expenseForm.amount || 0),
              currency: expenseForm.currency,
              occurred_at: expenseForm.occurred_at,
              description: expenseForm.description,
              account_code: expenseForm.account_code,
              account_name: expenseForm.account_name,
              cost_center_code: expenseForm.cost_center_code,
              vendor_name: expenseForm.vendor_name,
              employee_name: expenseForm.employee_name,
              invoice_number: expenseForm.invoice_number,
              payment_status: expenseForm.payment_status,
            },
          ],
        }).then(() => setExpenseForm((current) => ({ ...current, external_record_id: '', amount: '0', description: '' }))),
      'finance.expenseImported',
    )
  }

  async function submitFileImport(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (!selectedFile || !expenseForm.adapter_id) return
    await run(
      () => importFinanceExpenseFile(token, expenseForm.adapter_id, selectedFile).then(() => setSelectedFile(null)),
      'finance.fileImported',
    )
  }

  async function pullAdapter(adapterID: string) {
    await run(() => pullFinanceExpenses(token, adapterID).then(() => undefined), 'finance.pullCompleted')
  }

  async function submitPayable(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    await run(
      () =>
        createFinancePayable(token, {
          payable_type: payableForm.payable_type,
          external_payable_id: payableForm.external_payable_id,
          invoice_number: payableForm.invoice_number,
          vendor_name: payableForm.vendor_name,
          employee_name: payableForm.employee_name,
          amount: Number(payableForm.amount || 0),
          tax_amount: Number(payableForm.tax_amount || 0),
          currency: payableForm.currency,
          due_date: payableForm.due_date,
        }).then(() => undefined),
      'finance.payableCreated',
    )
  }

  async function submitPayment(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    await run(
      () =>
        createFinancePayment(token, {
          payment_number: paymentForm.payment_number,
          external_payment_id: paymentForm.external_payment_id,
          payment_method: paymentForm.payment_method,
          vendor_name: paymentForm.vendor_name,
          employee_name: paymentForm.employee_name,
          amount: Number(paymentForm.amount || 0),
          currency: paymentForm.currency,
          paid_at: paymentForm.paid_at,
          status: 'paid',
        }).then(() => undefined),
      'finance.paymentCreated',
    )
  }

  async function submitAllocation(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    await run(
      () =>
        allocateFinancePayment(token, allocationForm.payment_id, {
          payable_id: allocationForm.payable_id,
          amount: Number(allocationForm.amount || 0),
          currency: allocationForm.currency,
        }).then(() => undefined),
      'finance.paymentAllocated',
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

      {activeTab === 'ingestion' && (
        <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
          <Panel title="finance.ingestion">
            <form className="space-y-3" onSubmit={submitExpense}>
              <SelectInput
                label="finance.adapter"
                value={expenseForm.adapter_id}
                onChange={(value) => setExpenseForm({ ...expenseForm, adapter_id: value })}
                options={adapters.map((adapter) => adapter.id)}
                labels={Object.fromEntries(adapters.map((adapter) => [adapter.id, adapter.name]))}
              />
              <div className="grid gap-3 sm:grid-cols-2">
                <TextInput label="finance.externalRecordId" value={expenseForm.external_record_id} onChange={(value) => setExpenseForm({ ...expenseForm, external_record_id: value })} />
                <SelectInput label="finance.expenseType" value={expenseForm.expense_type} onChange={(value) => setExpenseForm({ ...expenseForm, expense_type: value })} options={['daily_expense', 'project_expense', 'salary', 'model_fee', 'agent_fee']} />
              </div>
              <div className="grid gap-3 sm:grid-cols-3">
                <TextInput label="finance.amount" value={expenseForm.amount} onChange={(value) => setExpenseForm({ ...expenseForm, amount: value })} />
                <TextInput label="finance.currency" value={expenseForm.currency} onChange={(value) => setExpenseForm({ ...expenseForm, currency: value })} />
                <TextInput label="finance.occurredAt" type="date" value={expenseForm.occurred_at} onChange={(value) => setExpenseForm({ ...expenseForm, occurred_at: value })} />
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                <TextInput label="finance.accountCode" value={expenseForm.account_code} onChange={(value) => setExpenseForm({ ...expenseForm, account_code: value })} />
                <TextInput label="finance.accountName" value={expenseForm.account_name} onChange={(value) => setExpenseForm({ ...expenseForm, account_name: value })} />
                <TextInput label="finance.costCenter" value={expenseForm.cost_center_code} onChange={(value) => setExpenseForm({ ...expenseForm, cost_center_code: value })} />
                <TextInput label="finance.vendor" value={expenseForm.vendor_name} onChange={(value) => setExpenseForm({ ...expenseForm, vendor_name: value })} />
                <TextInput label="finance.employee" value={expenseForm.employee_name} onChange={(value) => setExpenseForm({ ...expenseForm, employee_name: value })} />
                <TextInput label="finance.invoiceNumber" value={expenseForm.invoice_number} onChange={(value) => setExpenseForm({ ...expenseForm, invoice_number: value })} />
              </div>
              <TextInput label="costing.description" value={expenseForm.description} onChange={(value) => setExpenseForm({ ...expenseForm, description: value })} />
              <SubmitButton loading={loading || !expenseForm.adapter_id} label="finance.importExpense" />
            </form>
          </Panel>
          <Panel title="finance.fileAndPull">
            <form className="space-y-3" onSubmit={submitFileImport}>
              <SelectInput
                label="finance.adapter"
                value={expenseForm.adapter_id}
                onChange={(value) => setExpenseForm({ ...expenseForm, adapter_id: value })}
                options={adapters.map((adapter) => adapter.id)}
                labels={Object.fromEntries(adapters.map((adapter) => [adapter.id, adapter.name]))}
              />
              <label className="block">
                <span className="text-xs font-semibold text-slate-500">{t('finance.file')}</span>
                <input type="file" accept=".csv" onChange={(event) => setSelectedFile(event.target.files?.[0] ?? null)} className="mt-1 block w-full rounded-lg border border-slate-300 px-3 py-2 text-sm" />
              </label>
              <SubmitButton loading={loading || !selectedFile || !expenseForm.adapter_id} label="finance.importFile" />
            </form>
            <div className="mt-5 space-y-2 border-t border-slate-100 pt-4">
              {adapters.map((adapter) => (
                <button key={adapter.id} type="button" onClick={() => void pullAdapter(adapter.id)} className="inline-flex h-9 w-full items-center justify-between rounded-md border border-slate-300 px-3 text-sm font-semibold text-slate-700 hover:bg-slate-100">
                  <span className="truncate">{adapter.name}</span>
                  <span>{t('finance.pull')}</span>
                </button>
              ))}
            </div>
          </Panel>
        </div>
      )}

      {activeTab === 'imports' && (
        <div className="grid gap-5 xl:grid-cols-2">
          <Panel title="finance.importBatches">
            <Table
              headers={['finance.sourceType', 'developer.status', 'finance.totalRecords', 'finance.failedRecords']}
              rows={importBatches.map((batch) => [batch.source_type, t(batch.status), String(batch.total_records), String(batch.failed_records)])}
            />
          </Panel>
          <Panel title="finance.importRecords">
            <Table
              headers={['finance.externalRecordId', 'finance.expenseType', 'developer.status', 'finance.amount']}
              rows={importRecords.map((record) => [
                record.external_record_id,
                t(`finance.${record.expense_type}`),
                t(record.status),
                money(Number(record.normalized_payload.amount ?? 0), String(record.normalized_payload.currency ?? 'CNY')),
              ])}
            />
          </Panel>
        </div>
      )}

      {activeTab === 'payables' && (
        <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
          <Panel title="finance.payables">
            <Table
              headers={['finance.invoiceNumber', 'finance.vendor', 'finance.employee', 'finance.amount', 'developer.status']}
              rows={payables.map((payable) => [payable.invoice_number || payable.external_payable_id, payable.vendor_name, payable.employee_name, money(payable.amount, payable.currency), t(payable.status)])}
            />
          </Panel>
          <Panel title="finance.createPayable">
            <form className="space-y-3" onSubmit={submitPayable}>
              <SelectInput label="finance.payableType" value={payableForm.payable_type} onChange={(value) => setPayableForm({ ...payableForm, payable_type: value })} options={['expense', 'salary', 'project', 'model', 'agent', 'vendor']} />
              <TextInput label="finance.externalPayableId" value={payableForm.external_payable_id} onChange={(value) => setPayableForm({ ...payableForm, external_payable_id: value })} />
              <TextInput label="finance.invoiceNumber" value={payableForm.invoice_number} onChange={(value) => setPayableForm({ ...payableForm, invoice_number: value })} />
              <TextInput label="finance.vendor" value={payableForm.vendor_name} onChange={(value) => setPayableForm({ ...payableForm, vendor_name: value })} />
              <TextInput label="finance.employee" value={payableForm.employee_name} onChange={(value) => setPayableForm({ ...payableForm, employee_name: value })} />
              <div className="grid gap-3 sm:grid-cols-2">
                <TextInput label="finance.amount" value={payableForm.amount} onChange={(value) => setPayableForm({ ...payableForm, amount: value })} />
                <TextInput label="finance.taxAmount" value={payableForm.tax_amount} onChange={(value) => setPayableForm({ ...payableForm, tax_amount: value })} />
              </div>
              <TextInput label="finance.dueDate" type="date" value={payableForm.due_date} onChange={(value) => setPayableForm({ ...payableForm, due_date: value })} />
              <SubmitButton loading={loading} label="finance.createPayable" />
            </form>
          </Panel>
        </div>
      )}

      {activeTab === 'payments' && (
        <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
          <Panel title="finance.payments">
            <Table
              headers={['finance.paymentNumber', 'finance.vendor', 'finance.employee', 'finance.amount', 'developer.status']}
              rows={payments.map((payment) => [payment.payment_number || payment.external_payment_id, payment.vendor_name, payment.employee_name, money(payment.amount, payment.currency), t(payment.status)])}
            />
          </Panel>
          <Panel title="finance.createPayment">
            <form className="space-y-3" onSubmit={submitPayment}>
              <TextInput label="finance.paymentNumber" value={paymentForm.payment_number} onChange={(value) => setPaymentForm({ ...paymentForm, payment_number: value })} />
              <TextInput label="finance.externalPaymentId" value={paymentForm.external_payment_id} onChange={(value) => setPaymentForm({ ...paymentForm, external_payment_id: value })} />
              <TextInput label="finance.paymentMethod" value={paymentForm.payment_method} onChange={(value) => setPaymentForm({ ...paymentForm, payment_method: value })} />
              <TextInput label="finance.vendor" value={paymentForm.vendor_name} onChange={(value) => setPaymentForm({ ...paymentForm, vendor_name: value })} />
              <TextInput label="finance.employee" value={paymentForm.employee_name} onChange={(value) => setPaymentForm({ ...paymentForm, employee_name: value })} />
              <TextInput label="finance.amount" value={paymentForm.amount} onChange={(value) => setPaymentForm({ ...paymentForm, amount: value })} />
              <SubmitButton loading={loading} label="finance.createPayment" />
            </form>
            <form className="mt-5 space-y-3 border-t border-slate-100 pt-4" onSubmit={submitAllocation}>
              <SelectInput label="finance.payment" value={allocationForm.payment_id} onChange={(value) => setAllocationForm({ ...allocationForm, payment_id: value })} options={payments.map((payment) => payment.id)} labels={Object.fromEntries(payments.map((payment) => [payment.id, payment.payment_number || payment.id]))} />
              <SelectInput label="finance.payable" value={allocationForm.payable_id} onChange={(value) => setAllocationForm({ ...allocationForm, payable_id: value })} options={payables.map((payable) => payable.id)} labels={Object.fromEntries(payables.map((payable) => [payable.id, payable.invoice_number || payable.id]))} />
              <TextInput label="finance.amount" value={allocationForm.amount} onChange={(value) => setAllocationForm({ ...allocationForm, amount: value })} />
              <SubmitButton loading={loading || !allocationForm.payment_id || !allocationForm.payable_id} label="finance.allocatePayment" />
            </form>
          </Panel>
        </div>
      )}

      {activeTab === 'adapters' && (
        <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
          <Panel title="finance.adapters">
            <Table
              headers={['common.name', 'finance.adapterType', 'finance.direction', 'finance.authType', 'developer.status']}
              rows={adapters.map((adapter) => [
                adapter.name,
                t(`finance.${adapter.adapter_type}`),
                t(`finance.${adapter.direction}`),
                t(adapter.auth_type),
                t(adapter.status),
              ])}
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
              <div className="grid gap-3 sm:grid-cols-2">
                <SelectInput
                  label="finance.adapterType"
                  value={adapterForm.adapter_type}
                  onChange={(value) => setAdapterForm({ ...adapterForm, adapter_type: value })}
                  options={['generic', 'erp', 'accounting', 'hr', 'cloud_billing', 'agent_platform']}
                />
                <SelectInput
                  label="finance.direction"
                  value={adapterForm.direction}
                  onChange={(value) => setAdapterForm({ ...adapterForm, direction: value })}
                  options={['inbound', 'outbound', 'bidirectional']}
                />
              </div>
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
              <TextAreaInput
                label="finance.fieldMapping"
                value={adapterForm.field_mapping}
                onChange={(value) => setAdapterForm({ ...adapterForm, field_mapping: value })}
              />
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

function TextAreaInput({
  label,
  value,
  onChange,
}: {
  label: string
  value: string
  onChange: (value: string) => void
}) {
  const { t } = useI18n()
  return (
    <label className="block">
      <span className="text-xs font-semibold text-slate-500">{t(label)}</span>
      <textarea
        value={value}
        onChange={(event) => onChange(event.target.value)}
        rows={5}
        className="mt-1 w-full resize-y rounded-lg border border-slate-300 px-3 py-2 font-mono text-xs outline-none focus:border-slate-500 focus:ring-2 focus:ring-slate-200"
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
