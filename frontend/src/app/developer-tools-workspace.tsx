'use client'

import { Activity, Bot, FileCode2, Gauge, KeyRound, ListChecks, Network, RefreshCw, Route, ServerCog, Wrench } from 'lucide-react'
import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import {
  createModel,
  createModelProvider,
  createProviderChannel,
  createRoutingRule,
  getAICostSummary,
  getAIUsageAnalysis,
  listInvocations,
  listModelProviders,
  listModels,
  listProviderChannels,
  listRoutingRules,
  listToolExecutions,
  listTools,
  rotateModelProviderKey,
  rotateProviderChannelKey,
  testModelProvider,
  testProviderChannel,
  type AICostSummary,
  type AIInvocation,
  type AIRoutingRule,
  type AIUsageAnalysis,
  type ModelCatalogItem,
  type ModelProvider,
  type ProviderChannel,
  type ToolDefinition,
  type ToolExecution,
} from '@/lib/api'
import { useI18n } from '@/lib/i18n'

interface DeveloperToolsWorkspaceProps {
  token: string
}

type TabID = 'providers' | 'channels' | 'models' | 'routing' | 'invocations' | 'analysis' | 'tools' | 'interfaces'

const tabs: Array<{ id: TabID; label: string; icon: typeof ServerCog }> = [
  { id: 'providers', label: 'developer.providers', icon: ServerCog },
  { id: 'channels', label: 'developer.channels', icon: Network },
  { id: 'models', label: 'developer.models', icon: Bot },
  { id: 'routing', label: 'developer.routing', icon: Route },
  { id: 'invocations', label: 'developer.invocations', icon: Activity },
  { id: 'analysis', label: 'developer.usageAnalysis', icon: Gauge },
  { id: 'tools', label: 'developer.tools', icon: Wrench },
  { id: 'interfaces', label: 'developer.interfaces', icon: FileCode2 },
]

function money(value: number | undefined, currency = 'CNY'): string {
  return `${currency} ${Number(value ?? 0).toFixed(4)}`
}

export function DeveloperToolsWorkspace({ token }: DeveloperToolsWorkspaceProps) {
  const { t } = useI18n()
  const [activeTab, setActiveTab] = useState<TabID>('providers')
  const [providers, setProviders] = useState<ModelProvider[]>([])
  const [channels, setChannels] = useState<ProviderChannel[]>([])
  const [models, setModels] = useState<ModelCatalogItem[]>([])
  const [rules, setRules] = useState<AIRoutingRule[]>([])
  const [tools, setTools] = useState<ToolDefinition[]>([])
  const [executions, setExecutions] = useState<ToolExecution[]>([])
  const [invocations, setInvocations] = useState<AIInvocation[]>([])
  const [cost, setCost] = useState<AICostSummary | null>(null)
  const [usageAnalysis, setUsageAnalysis] = useState<AIUsageAnalysis | null>(null)
  const [selectedProviderID, setSelectedProviderID] = useState('')
  const [selectedChannelID, setSelectedChannelID] = useState('')
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
  const [channelForm, setChannelForm] = useState({
    provider_id: '',
    name: '',
    base_url: '',
    api_key: '',
    owner_type: '',
    priority: '50',
    concurrency_limit: '0',
    load_factor: '1',
    rate_multiplier: '1',
    quota_amount: '0',
    quota_currency: 'CNY',
    supported_model_patterns: '*',
    model_mapping: '',
  })
  const [modelForm, setModelForm] = useState({
    provider_id: '',
    model_key: '',
    display_name: '',
    context_window: '',
    max_output_tokens: '',
    input_price_per_1k: '',
    output_price_per_1k: '',
    cache_creation_price_per_1k: '',
    cache_read_price_per_1k: '',
    image_output_price_per_1k: '',
    priority_input_price_per_1k: '',
    priority_output_price_per_1k: '',
    long_context_threshold: '',
    long_context_input_multiplier: '',
    long_context_output_multiplier: '',
    currency: 'CNY',
    capabilities: '',
  })
  const [routingForm, setRoutingForm] = useState({
    name: '',
    provider_id: '',
    channel_id: '',
    match_scope: 'global',
    match_value: '',
    model_pattern: '*',
    priority: '100',
  })
  const [secretInput, setSecretInput] = useState('')
  const [channelSecretInput, setChannelSecretInput] = useState('')
  const [testModel, setTestModel] = useState('')
  const [channelTestModel, setChannelTestModel] = useState('')

  const providerLabels = useMemo(() => Object.fromEntries(providers.map((provider) => [provider.id, provider.name])), [providers])
  const channelLabels = useMemo(() => Object.fromEntries(channels.map((channel) => [channel.id, channel.name])), [channels])
  const selectedProvider = useMemo(
    () => providers.find((provider) => provider.id === selectedProviderID) ?? providers[0],
    [providers, selectedProviderID],
  )
  const selectedChannel = useMemo(
    () => channels.find((channel) => channel.id === selectedChannelID) ?? channels[0],
    [channels, selectedChannelID],
  )

  const loadAll = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [providerData, channelData, modelData, ruleData, toolData, executionData, invocationData, costData, analysisData] = await Promise.all([
        listModelProviders(token),
        listProviderChannels(token),
        listModels(token),
        listRoutingRules(token),
        listTools(token),
        listToolExecutions(token),
        listInvocations(token),
        getAICostSummary(token),
        getAIUsageAnalysis(token),
      ])
      setProviders(providerData)
      setChannels(channelData)
      setModels(modelData)
      setRules(ruleData)
      setTools(toolData)
      setExecutions(executionData)
      setInvocations(invocationData)
      setCost(costData)
      setUsageAnalysis(analysisData)
      setSelectedProviderID((current) => current || providerData[0]?.id || '')
      setSelectedChannelID((current) => current || channelData[0]?.id || '')
      setModelForm((current) => ({ ...current, provider_id: current.provider_id || providerData[0]?.id || '' }))
      setChannelForm((current) => ({ ...current, provider_id: current.provider_id || providerData[0]?.id || '' }))
      setRoutingForm((current) => ({ ...current, provider_id: current.provider_id || providerData[0]?.id || '' }))
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

  async function submitChannel(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const providerID = channelForm.provider_id || selectedProvider?.id
    if (!providerID) return
    await run(
      () =>
        createProviderChannel(token, providerID, {
          provider_id: providerID,
          name: channelForm.name,
          base_url: channelForm.base_url || undefined,
          api_key: channelForm.api_key,
          owner_type: channelForm.owner_type || undefined,
          priority: numberOrUndefined(channelForm.priority),
          concurrency_limit: numberOrUndefined(channelForm.concurrency_limit),
          load_factor: numberOrUndefined(channelForm.load_factor),
          rate_multiplier: numberOrUndefined(channelForm.rate_multiplier),
          quota_amount: numberOrUndefined(channelForm.quota_amount),
          quota_currency: channelForm.quota_currency || undefined,
          supported_model_patterns: splitCsv(channelForm.supported_model_patterns),
          model_mapping: parseMapping(channelForm.model_mapping),
          metadata: {},
        }).then(() => setChannelForm((current) => ({ ...current, name: '', api_key: '' }))),
      'developer.channelCreated',
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
          cache_creation_price_per_1k: numberOrUndefined(modelForm.cache_creation_price_per_1k),
          cache_read_price_per_1k: numberOrUndefined(modelForm.cache_read_price_per_1k),
          image_output_price_per_1k: numberOrUndefined(modelForm.image_output_price_per_1k),
          priority_input_price_per_1k: numberOrUndefined(modelForm.priority_input_price_per_1k),
          priority_output_price_per_1k: numberOrUndefined(modelForm.priority_output_price_per_1k),
          long_context_threshold: numberOrUndefined(modelForm.long_context_threshold),
          long_context_input_multiplier: numberOrUndefined(modelForm.long_context_input_multiplier),
          long_context_output_multiplier: numberOrUndefined(modelForm.long_context_output_multiplier),
          currency: modelForm.currency,
          capabilities: splitCsv(modelForm.capabilities),
          metadata: {},
        }).then(() => setModelForm((current) => ({ ...current, model_key: '', display_name: '' }))),
      'developer.modelCreated',
    )
  }

  async function submitRoutingRule(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    await run(
      () =>
        createRoutingRule(token, {
          name: routingForm.name,
          provider_id: routingForm.provider_id || undefined,
          channel_id: routingForm.channel_id || undefined,
          match_scope: routingForm.match_scope,
          match_value: routingForm.match_value || undefined,
          model_pattern: routingForm.model_pattern || '*',
          priority: Number(routingForm.priority || 100),
          status: 'active',
          metadata: {},
        }).then(() => setRoutingForm((current) => ({ ...current, name: '', match_value: '' }))),
      'developer.routingRuleCreated',
    )
  }

  async function rotateKey() {
    if (!selectedProvider || !secretInput) return
    await run(
      () => rotateModelProviderKey(token, selectedProvider.id, secretInput).then(() => setSecretInput('')),
      'developer.keyRotated',
    )
  }

  async function rotateChannelKey() {
    if (!selectedChannel || !channelSecretInput) return
    await run(
      () => rotateProviderChannelKey(token, selectedChannel.id, channelSecretInput).then(() => setChannelSecretInput('')),
      'developer.channelKeyRotated',
    )
  }

  async function testProvider() {
    if (!selectedProvider) return
    await run(() => testModelProvider(token, selectedProvider.id, testModel || undefined).then(() => undefined), 'developer.providerTested')
  }

  async function testChannel() {
    if (!selectedChannel) return
    await run(() => testProviderChannel(token, selectedChannel.id, channelTestModel || undefined).then(() => undefined), 'developer.channelTested')
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
        <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_380px]">
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
              <TextInput label="developer.baseUrl" value={providerForm.base_url} onChange={(value) => setProviderForm({ ...providerForm, base_url: value })} />
              <TextInput label="developer.apiKey" type="password" value={providerForm.api_key} onChange={(value) => setProviderForm({ ...providerForm, api_key: value })} />
              <div className="grid gap-3 sm:grid-cols-2">
                <TextInput label="developer.timeout" value={providerForm.timeout_ms} onChange={(value) => setProviderForm({ ...providerForm, timeout_ms: value })} />
                <TextInput label="developer.retries" value={providerForm.retry_count} onChange={(value) => setProviderForm({ ...providerForm, retry_count: value })} />
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

      {activeTab === 'channels' && (
        <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_420px]">
          <Panel title="developer.channelPool">
            <Table
              headers={['developer.channel', 'developer.provider', 'developer.health', 'developer.load', 'developer.quota', 'developer.rateMultiplier']}
              rows={channels.map((channel) => [
                channel.name,
                providerLabels[channel.provider_id] || channel.provider_id,
                t(channel.health_status || channel.status),
                `${channel.inflight_requests}/${channel.concurrency_limit || t('common.none')}`,
                `${money(channel.quota_used, channel.quota_currency)} / ${channel.quota_amount || t('common.none')}`,
                String(channel.rate_multiplier),
              ])}
            />
          </Panel>

          <Panel title="developer.channelSettings">
            <form className="space-y-3" onSubmit={submitChannel}>
              <SelectInput label="developer.provider" value={channelForm.provider_id} onChange={(value) => setChannelForm({ ...channelForm, provider_id: value })} options={providers.map((provider) => provider.id)} labels={providerLabels} />
              <TextInput label="developer.channel" value={channelForm.name} onChange={(value) => setChannelForm({ ...channelForm, name: value })} />
              <TextInput label="developer.baseUrl" value={channelForm.base_url} onChange={(value) => setChannelForm({ ...channelForm, base_url: value })} />
              <TextInput label="developer.apiKey" type="password" value={channelForm.api_key} onChange={(value) => setChannelForm({ ...channelForm, api_key: value })} />
              <div className="grid gap-3 sm:grid-cols-2">
                <TextInput label="developer.priority" value={channelForm.priority} onChange={(value) => setChannelForm({ ...channelForm, priority: value })} />
                <TextInput label="developer.rateMultiplier" value={channelForm.rate_multiplier} onChange={(value) => setChannelForm({ ...channelForm, rate_multiplier: value })} />
                <TextInput label="developer.concurrency" value={channelForm.concurrency_limit} onChange={(value) => setChannelForm({ ...channelForm, concurrency_limit: value })} />
                <TextInput label="developer.loadFactor" value={channelForm.load_factor} onChange={(value) => setChannelForm({ ...channelForm, load_factor: value })} />
                <TextInput label="developer.quota" value={channelForm.quota_amount} onChange={(value) => setChannelForm({ ...channelForm, quota_amount: value })} />
                <TextInput label="finance.currency" value={channelForm.quota_currency} onChange={(value) => setChannelForm({ ...channelForm, quota_currency: value })} />
              </div>
              <TextInput label="developer.modelPatterns" value={channelForm.supported_model_patterns} onChange={(value) => setChannelForm({ ...channelForm, supported_model_patterns: value })} />
              <TextInput label="developer.modelMapping" value={channelForm.model_mapping} onChange={(value) => setChannelForm({ ...channelForm, model_mapping: value })} />
              <SubmitButton loading={loading} label="developer.createChannel" />
            </form>

            <div className="mt-5 border-t border-slate-100 pt-4">
              <SelectInput label="developer.channel" value={selectedChannel?.id || ''} onChange={setSelectedChannelID} options={channels.map((channel) => channel.id)} labels={channelLabels} />
              <TextInput label="developer.newKey" type="password" value={channelSecretInput} onChange={setChannelSecretInput} />
              <button
                type="button"
                onClick={() => void rotateChannelKey()}
                disabled={!selectedChannel || !channelSecretInput || loading}
                className="mt-3 inline-flex h-10 w-full items-center justify-center gap-2 rounded-lg border border-slate-300 px-3 text-sm font-semibold text-slate-700 hover:bg-slate-100 disabled:opacity-50"
              >
                <KeyRound className="h-4 w-4" />
                {t('developer.rotateChannelKey')}
              </button>
              <TextInput label="developer.testModel" value={channelTestModel} onChange={setChannelTestModel} />
              <button
                type="button"
                onClick={() => void testChannel()}
                disabled={!selectedChannel || loading}
                className="mt-3 inline-flex h-10 w-full items-center justify-center rounded-lg bg-slate-950 px-3 text-sm font-semibold text-white hover:bg-slate-800 disabled:opacity-50"
              >
                {t('developer.testChannel')}
              </button>
            </div>
          </Panel>
        </div>
      )}

      {activeTab === 'models' && (
        <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_420px]">
          <Panel title="developer.modelCatalog">
            <Table
              headers={['developer.model', 'developer.provider', 'developer.status', 'developer.context']}
              rows={models.map((model) => [
                model.display_name || model.model_key,
                providerLabels[model.provider_id] || model.provider_id,
                t(model.status),
                String(model.context_window || 0),
              ])}
            />
          </Panel>
          <Panel title="developer.createModel">
            <form className="space-y-3" onSubmit={submitModel}>
              <SelectInput label="developer.provider" value={modelForm.provider_id} onChange={(value) => setModelForm({ ...modelForm, provider_id: value })} options={providers.map((provider) => provider.id)} labels={providerLabels} />
              <TextInput label="developer.modelKey" value={modelForm.model_key} onChange={(value) => setModelForm({ ...modelForm, model_key: value })} />
              <TextInput label="developer.displayName" value={modelForm.display_name} onChange={(value) => setModelForm({ ...modelForm, display_name: value })} />
              <div className="grid gap-3 sm:grid-cols-2">
                <TextInput label="developer.context" value={modelForm.context_window} onChange={(value) => setModelForm({ ...modelForm, context_window: value })} />
                <TextInput label="developer.maxOutput" value={modelForm.max_output_tokens} onChange={(value) => setModelForm({ ...modelForm, max_output_tokens: value })} />
                <TextInput label="developer.inputPrice" value={modelForm.input_price_per_1k} onChange={(value) => setModelForm({ ...modelForm, input_price_per_1k: value })} />
                <TextInput label="developer.outputPrice" value={modelForm.output_price_per_1k} onChange={(value) => setModelForm({ ...modelForm, output_price_per_1k: value })} />
                <TextInput label="developer.cacheCreationPrice" value={modelForm.cache_creation_price_per_1k} onChange={(value) => setModelForm({ ...modelForm, cache_creation_price_per_1k: value })} />
                <TextInput label="developer.cacheReadPrice" value={modelForm.cache_read_price_per_1k} onChange={(value) => setModelForm({ ...modelForm, cache_read_price_per_1k: value })} />
                <TextInput label="developer.imageOutputPrice" value={modelForm.image_output_price_per_1k} onChange={(value) => setModelForm({ ...modelForm, image_output_price_per_1k: value })} />
                <TextInput label="developer.priorityInputPrice" value={modelForm.priority_input_price_per_1k} onChange={(value) => setModelForm({ ...modelForm, priority_input_price_per_1k: value })} />
                <TextInput label="developer.priorityOutputPrice" value={modelForm.priority_output_price_per_1k} onChange={(value) => setModelForm({ ...modelForm, priority_output_price_per_1k: value })} />
                <TextInput label="developer.longContextThreshold" value={modelForm.long_context_threshold} onChange={(value) => setModelForm({ ...modelForm, long_context_threshold: value })} />
              </div>
              <TextInput label="developer.capabilities" value={modelForm.capabilities} onChange={(value) => setModelForm({ ...modelForm, capabilities: value })} />
              <SubmitButton loading={loading} label="developer.createModel" />
            </form>
          </Panel>
        </div>
      )}

      {activeTab === 'routing' && (
        <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_420px]">
          <Panel title="developer.routingRules">
            <Table
              headers={['common.name', 'developer.modelPattern', 'developer.matchScope', 'developer.channel', 'developer.priority']}
              rows={rules.map((rule) => [
                rule.name,
                rule.model_pattern || '*',
                `${rule.match_scope}:${rule.match_value || t('common.none')}`,
                rule.channel_id ? channelLabels[rule.channel_id] || rule.channel_id : providerLabels[rule.provider_id || ''] || t('common.none'),
                String(rule.priority),
              ])}
            />
          </Panel>
          <Panel title="developer.createRoutingRule">
            <form className="space-y-3" onSubmit={submitRoutingRule}>
              <TextInput label="common.name" value={routingForm.name} onChange={(value) => setRoutingForm({ ...routingForm, name: value })} />
              <SelectInput label="developer.provider" value={routingForm.provider_id} onChange={(value) => setRoutingForm({ ...routingForm, provider_id: value })} options={['', ...providers.map((provider) => provider.id)]} labels={{ '': t('common.none'), ...providerLabels }} />
              <SelectInput label="developer.channel" value={routingForm.channel_id} onChange={(value) => setRoutingForm({ ...routingForm, channel_id: value })} options={['', ...channels.map((channel) => channel.id)]} labels={{ '': t('common.none'), ...channelLabels }} />
              <SelectInput label="developer.matchScope" value={routingForm.match_scope} onChange={(value) => setRoutingForm({ ...routingForm, match_scope: value })} options={['global', 'organization', 'department', 'project', 'requirement', 'workflow', 'task', 'agent', 'user', 'source_surface']} labels={scopeLabels(t)} />
              <TextInput label="developer.matchValue" value={routingForm.match_value} onChange={(value) => setRoutingForm({ ...routingForm, match_value: value })} />
              <div className="grid gap-3 sm:grid-cols-2">
                <TextInput label="developer.modelPattern" value={routingForm.model_pattern} onChange={(value) => setRoutingForm({ ...routingForm, model_pattern: value })} />
                <TextInput label="developer.priority" value={routingForm.priority} onChange={(value) => setRoutingForm({ ...routingForm, priority: value })} />
              </div>
              <SubmitButton loading={loading} label="developer.createRoutingRule" />
            </form>
          </Panel>
        </div>
      )}

      {activeTab === 'invocations' && (
        <Panel title="developer.invocationLogs">
          <Table
            headers={['developer.invocation', 'developer.modelRoute', 'developer.channel', 'developer.tokens', 'developer.serviceTier', 'developer.cost']}
            rows={invocations.map((invocation) => [
              invocation.id,
              `${invocation.requested_model || invocation.model_id} -> ${invocation.upstream_model || invocation.model_id}`,
              invocation.channel_id ? channelLabels[invocation.channel_id] || invocation.channel_id : t('developer.providerDefault'),
              `${invocation.input_tokens || 0}/${invocation.output_tokens || 0}`,
              invocation.service_tier || t('common.none'),
              money(invocation.cost_amount, invocation.currency),
            ])}
          />
        </Panel>
      )}

      {activeTab === 'analysis' && (
        <Panel title="developer.usageAnalysis">
          <div className="grid gap-3 sm:grid-cols-3">
            <Metric label="developer.totalCost" value={money(usageAnalysis?.total_cost ?? cost?.total, usageAnalysis?.currency ?? cost?.currency)} />
            <Metric label="developer.invocationCount" value={String(usageAnalysis?.invocation_count ?? invocations.length)} />
            <Metric label="developer.unexportedCost" value={money(cost?.unexported, cost?.currency)} />
          </div>
          <div className="mt-5 grid gap-4 lg:grid-cols-2">
            <Breakdown title="developer.byProvider" values={usageAnalysis?.by_provider ?? cost?.by_provider ?? {}} currency={usageAnalysis?.currency ?? cost?.currency} />
            <Breakdown title="developer.byChannel" values={usageAnalysis?.by_channel ?? cost?.by_channel ?? {}} currency={usageAnalysis?.currency ?? cost?.currency} />
            <Breakdown title="developer.byModel" values={usageAnalysis?.by_model ?? {}} currency={usageAnalysis?.currency ?? cost?.currency} />
            <Breakdown title="developer.byActor" values={usageAnalysis?.by_actor ?? {}} currency={usageAnalysis?.currency ?? cost?.currency} />
          </div>
        </Panel>
      )}

      {activeTab === 'tools' && (
        <Panel title="developer.toolRuntime">
          <Table
            headers={['developer.tool', 'developer.category', 'developer.approvalTier', 'developer.policy', 'developer.risk']}
            rows={tools.map((tool) => [
              tool.name,
              t(tool.tool_category || 'execution_operation'),
              t(tool.approval_tier_required || 'executor'),
              t(tool.default_policy),
              t(tool.risk_level),
            ])}
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
            <JsonTemplate title="developer.channelContract" value={channelContract()} />
            <JsonTemplate title="developer.routingContract" value={routingContract()} />
            <JsonTemplate title="developer.toolContract" value={toolContract()} />
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

function TextInput({ label, value, onChange, type = 'text' }: { label: string; value: string; onChange: (value: string) => void; type?: string }) {
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
          <option key={option || 'none'} value={option}>
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
                <td key={`${cell}-${cellIndex}`} className="max-w-[280px] truncate px-3 py-2 text-slate-700" title={cell}>
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

function Breakdown({ title, values, currency = 'CNY' }: { title: string; values: Record<string, number>; currency?: string }) {
  const { t } = useI18n()
  const entries = Object.entries(values)
  return (
    <div className="rounded-lg border border-slate-200 p-4">
      <p className="text-sm font-semibold text-slate-950">{t(title)}</p>
      <div className="mt-3 space-y-2">
        {entries.map(([key, value]) => (
          <div key={key} className="flex items-center justify-between gap-3 text-sm">
            <span className="min-w-0 truncate text-slate-600">{key}</span>
            <span className="shrink-0 font-semibold text-slate-900">{money(value, currency)}</span>
          </div>
        ))}
        {entries.length === 0 && <p className="text-sm text-slate-500">{t('common.noData')}</p>}
      </div>
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
      <p className="mt-2 truncate text-xl font-semibold text-slate-950" title={value}>
        {value}
      </p>
    </div>
  )
}

function StatusBadge({ label }: { label: string }) {
  const { t } = useI18n()
  return <span className="inline-flex h-7 items-center rounded-md border border-slate-200 bg-slate-50 px-2 text-xs font-semibold text-slate-600">{t(label)}</span>
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

function parseMapping(value: string): Record<string, string> {
  const trimmed = value.trim()
  if (!trimmed) return {}
  try {
    const parsed = JSON.parse(trimmed)
    return typeof parsed === 'object' && parsed ? (parsed as Record<string, string>) : {}
  } catch {
    return Object.fromEntries(
      trimmed
        .split(/[\n,]/)
        .map((line) => line.trim())
        .filter(Boolean)
        .map((line) => {
          const [from, to] = line.split('=').map((part) => part.trim())
          return [from, to]
        })
        .filter(([from, to]) => from && to),
    )
  }
}

function numberOrUndefined(value: string): number | undefined {
  if (value.trim() === '') return undefined
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : undefined
}

function scopeLabels(t: (key: string) => string): Record<string, string> {
  return {
    global: t('developer.scope.global'),
    organization: t('developer.scope.organization'),
    department: t('developer.scope.department'),
    project: t('developer.scope.project'),
    requirement: t('developer.scope.requirement'),
    workflow: t('developer.scope.workflow'),
    task: t('developer.scope.task'),
    agent: t('developer.scope.agent'),
    user: t('developer.scope.user'),
    source_surface: t('developer.scope.sourceSurface'),
  }
}

function providerContract() {
  return {
    format_version: 'meta-org.provider.v2',
    provider_type: 'openai | anthropic | gemini',
    auth: { type: 'api_key', encrypted_at_rest: true, provider_default_key: true },
    streaming: { protocol: 'sse', events: ['lifecycle', 'delta', 'usage_update', 'error', 'done'] },
  }
}

function channelContract() {
  return {
    format_version: 'meta-org.channel.v1',
    scheduling: { priority: 'ascending', load_factor: 'weighted', last_used_at: 'least recently used' },
    routing: { supported_model_patterns: ['gpt-*'], model_mapping: { 'logical-model': 'upstream-model' } },
    accounting: { rate_multiplier: '0 allowed', quota_currency: 'CNY' },
  }
}

function routingContract() {
  return {
    format_version: 'meta-org.routing.v1',
    match_scope: 'global | organization | project | agent | user | source_surface',
    model_pattern: '* wildcard supported',
    target: { provider_id: 'optional', channel_id: 'optional' },
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
