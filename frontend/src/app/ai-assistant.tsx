'use client'

import { Bot, CircleStop, Send, Sparkles, Wrench } from 'lucide-react'
import { useRef, useState } from 'react'
import { API_BASE, getAIInvocation } from '@/lib/api'
import { useI18n } from '@/lib/i18n'
import { streamSSE } from '@/lib/stream'

type AssistantState =
  | 'idle'
  | 'streaming'
  | 'approval_required'
  | 'completed'
  | 'provider_error'
  | 'governance_denied'
  | 'cancelled'

interface GatewayStreamData {
  invocation_id?: string
  delta?: string
  error?: string
  done?: boolean
  estimated_cost_amount?: number
  cost_amount?: number
  currency?: string
  tool_call?: {
    name?: string
    status?: string
  }
  usage?: {
    input_tokens?: number
    output_tokens?: number
  }
}

interface AssistantEvent {
  id: string
  event: string
  detail: string
}

interface AssistantUsage {
  input_tokens?: number
  output_tokens?: number
}

interface AssistantCost {
  estimated?: number
  final?: number
  currency: string
}

interface AIAssistantProps {
  token: string
  contextType: 'meta_org' | 'requirement' | 'project' | 'organization' | 'governance' | 'developer_tools'
  contextID?: string
}

const contextActions: Record<AIAssistantProps['contextType'], string[]> = {
  meta_org: ['assistant.action.summarizeHealth', 'assistant.action.reviewRisks'],
  requirement: ['assistant.action.analyzeRequirement', 'assistant.action.acceptanceCriteria'],
  project: ['assistant.action.recommendMembers', 'assistant.action.generateWorkflow', 'assistant.action.estimateCost'],
  organization: ['assistant.action.suggestPositions', 'assistant.action.capabilityGaps'],
  governance: ['assistant.action.explainDecision', 'assistant.action.proposeRule'],
  developer_tools: ['assistant.action.testProviders', 'assistant.action.inspectSchemas'],
}

function completedState(current: AssistantState): AssistantState {
  if (current === 'provider_error' || current === 'governance_denied' || current === 'cancelled') return current
  return 'completed'
}

function money(value: number | undefined, currency: string, empty: string): string {
  return typeof value === 'number' ? `${currency} ${value.toFixed(4)}` : empty
}

export function AIAssistant({ token, contextType, contextID }: AIAssistantProps) {
  const { t } = useI18n()
  const [providerType, setProviderType] = useState<'openai' | 'anthropic' | 'gemini'>('openai')
  const [model, setModel] = useState('gpt-4o-mini')
  const [prompt, setPrompt] = useState('')
  const [state, setState] = useState<AssistantState>('idle')
  const [output, setOutput] = useState('')
  const [events, setEvents] = useState<AssistantEvent[]>([])
  const [invocationID, setInvocationID] = useState('')
  const [usage, setUsage] = useState<AssistantUsage>({})
  const [cost, setCost] = useState<AssistantCost>({ currency: 'CNY' })
  const abortRef = useRef<AbortController | null>(null)

  async function send(nextPrompt = prompt) {
    const trimmed = nextPrompt.trim()
    if (!trimmed || state === 'streaming') return
    const controller = new AbortController()
    abortRef.current = controller
    setState('streaming')
    setOutput('')
    setEvents([])
    setInvocationID('')
    setUsage({})
    setCost({ currency: 'CNY' })
    let currentInvocationID = ''

    const message = [
      t('assistant.contextLabel'),
      contextType,
      contextID ? `${t('assistant.contextId')}: ${contextID}` : '',
      trimmed,
    ]
      .filter(Boolean)
      .join('\n')

    const params = new URLSearchParams({
      provider_type: providerType,
      model,
      message,
      source_surface: contextType,
    })

    try {
      await streamSSE<GatewayStreamData>(
        `${API_BASE}/ai-gateway/stream?${params.toString()}`,
        token,
        ({ event, data }) => {
          if (data.invocation_id) {
            currentInvocationID = data.invocation_id
            setInvocationID(data.invocation_id)
          }
          if (data.delta) {
            setOutput((current) => current + data.delta)
          }
          if (data.usage) {
            setUsage({
              input_tokens: data.usage.input_tokens,
              output_tokens: data.usage.output_tokens,
            })
          }
          if (typeof data.estimated_cost_amount === 'number' || typeof data.cost_amount === 'number' || data.currency) {
            setCost((current) => ({
              estimated: typeof data.estimated_cost_amount === 'number' ? data.estimated_cost_amount : current.estimated,
              final: typeof data.cost_amount === 'number' ? data.cost_amount : current.final,
              currency: data.currency || current.currency,
            }))
          }
          if (data.tool_call || event.startsWith('tool_call')) {
            const toolStatus = data.tool_call?.status ?? ''
            if (event.includes('approval') || toolStatus === 'approval_required') {
              setState('approval_required')
            }
            setEvents((current) => [
              ...current,
              {
                id: `${event}-${current.length}`,
                event,
                detail: data.tool_call?.name || data.tool_call?.status || t('assistant.toolCall'),
              },
            ])
          }
          if (data.error) {
            const denied = data.error.toLowerCase().includes('denied') || data.error.toLowerCase().includes('forbidden')
            setState(denied ? 'governance_denied' : 'provider_error')
            setEvents((current) => [...current, { id: `${event}-${current.length}`, event, detail: data.error || '' }])
          }
          if (data.done) {
            setState(completedState)
          }
        },
        controller.signal,
      )
      if (currentInvocationID) {
        try {
          const invocation = await getAIInvocation(token, currentInvocationID)
          setUsage({
            input_tokens: invocation.input_tokens,
            output_tokens: invocation.output_tokens,
          })
          setCost({ final: invocation.cost_amount, currency: invocation.currency })
        } catch (err) {
          setEvents((current) => [
            ...current,
            {
              id: `cost-${current.length}`,
              event: 'cost',
              detail: err instanceof Error ? err.message : t('assistant.costUnavailable'),
            },
          ])
        }
      }
      setState(completedState)
    } catch (err) {
      if (controller.signal.aborted) {
        setState('cancelled')
        return
      }
      setState('provider_error')
      setEvents((current) => [
        ...current,
        { id: `error-${current.length}`, event: 'error', detail: err instanceof Error ? err.message : t('assistant.error') },
      ])
    } finally {
      abortRef.current = null
    }
  }

  function cancel() {
    abortRef.current?.abort()
    setState('cancelled')
  }

  function runAction(key: string) {
    const nextPrompt = t(key)
    setPrompt(nextPrompt)
    void send(nextPrompt)
  }

  return (
    <aside className="h-fit rounded-lg border border-slate-200 bg-white p-4 shadow-sm">
      <div className="flex items-center justify-between gap-3">
        <div className="flex min-w-0 items-center gap-2">
          <Bot className="h-5 w-5 shrink-0 text-slate-500" />
          <h2 className="truncate text-base font-semibold text-slate-950">{t('assistant.title')}</h2>
        </div>
        <StateBadge state={state} />
      </div>

      <div className="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-1">
        <label className="block">
          <span className="text-xs font-semibold text-slate-500">{t('assistant.provider')}</span>
          <select
            value={providerType}
            onChange={(event) => setProviderType(event.target.value as 'openai' | 'anthropic' | 'gemini')}
            className="mt-1 h-10 w-full rounded-lg border border-slate-300 bg-white px-3 text-sm outline-none focus:border-slate-500 focus:ring-2 focus:ring-slate-200"
          >
            <option value="openai">{t('provider.openai')}</option>
            <option value="anthropic">{t('provider.anthropic')}</option>
            <option value="gemini">{t('provider.gemini')}</option>
          </select>
        </label>
        <label className="block">
          <span className="text-xs font-semibold text-slate-500">{t('assistant.model')}</span>
          <input
            value={model}
            onChange={(event) => setModel(event.target.value)}
            className="mt-1 h-10 w-full rounded-lg border border-slate-300 px-3 text-sm outline-none focus:border-slate-500 focus:ring-2 focus:ring-slate-200"
          />
        </label>
      </div>

      <div className="mt-4 flex flex-wrap gap-2">
        {contextActions[contextType].map((key) => (
          <button
            key={key}
            type="button"
            onClick={() => runAction(key)}
            disabled={state === 'streaming'}
            className="inline-flex h-8 items-center gap-1.5 rounded-md border border-slate-300 px-2 text-xs font-semibold text-slate-600 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <Sparkles className="h-3.5 w-3.5" />
            {t(key)}
          </button>
        ))}
      </div>

      <label className="mt-4 block">
        <span className="text-xs font-semibold text-slate-500">{t('assistant.prompt')}</span>
        <textarea
          value={prompt}
          onChange={(event) => setPrompt(event.target.value)}
          className="mt-1 h-28 w-full resize-y rounded-lg border border-slate-300 p-3 text-sm outline-none focus:border-slate-500 focus:ring-2 focus:ring-slate-200"
        />
      </label>

      <div className="mt-3 flex gap-2">
        <button
          type="button"
          onClick={() => void send()}
          disabled={state === 'streaming' || !prompt.trim()}
          className="inline-flex h-10 flex-1 items-center justify-center gap-2 rounded-lg bg-slate-950 px-3 text-sm font-semibold text-white hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-50"
        >
          <Send className="h-4 w-4" />
          {t('assistant.send')}
        </button>
        {state === 'streaming' && (
          <button
            type="button"
            onClick={cancel}
            className="inline-flex h-10 items-center justify-center rounded-lg border border-slate-300 px-3 text-sm font-semibold text-slate-700 hover:bg-slate-100"
            aria-label={t('assistant.cancel')}
          >
            <CircleStop className="h-4 w-4" />
          </button>
        )}
      </div>

      <div className="mt-4 min-h-[180px] rounded-lg border border-slate-200 bg-slate-50 p-3 text-sm text-slate-800">
        {output || <span className="text-slate-400">{t('assistant.empty')}</span>}
      </div>

      <div className="mt-3 grid gap-2 sm:grid-cols-2 xl:grid-cols-1">
        <AssistantFact label="assistant.invocation" value={invocationID || t('common.none')} />
        <AssistantFact
          label="assistant.tokens"
          value={
            typeof usage.input_tokens === 'number' || typeof usage.output_tokens === 'number'
              ? `${usage.input_tokens ?? 0} / ${usage.output_tokens ?? 0}`
              : t('common.none')
          }
        />
        <AssistantFact label="assistant.estimatedCost" value={money(cost.estimated, cost.currency, t('common.none'))} />
        <AssistantFact label="assistant.finalCost" value={money(cost.final, cost.currency, t('common.none'))} />
      </div>

      {events.length > 0 && (
        <div className="mt-4 space-y-2">
          {events.map((item) => (
            <div key={item.id} className="flex items-start gap-2 rounded-lg border border-slate-200 bg-white p-2 text-xs">
              <Wrench className="mt-0.5 h-3.5 w-3.5 shrink-0 text-slate-500" />
              <div className="min-w-0">
                <p className="font-semibold text-slate-700">{item.event}</p>
                <p className="mt-1 break-words text-slate-500">{item.detail}</p>
              </div>
            </div>
          ))}
        </div>
      )}
    </aside>
  )
}

function AssistantFact({ label, value }: { label: string; value: string }) {
  const { t } = useI18n()
  return (
    <div className="min-w-0 rounded-md border border-slate-200 bg-white px-2.5 py-2">
      <p className="text-xs font-semibold text-slate-500">{t(label)}</p>
      <p className="mt-1 truncate text-xs font-medium text-slate-800" title={value}>
        {value}
      </p>
    </div>
  )
}

function StateBadge({ state }: { state: AssistantState }) {
  const { t } = useI18n()
  const tone =
    state === 'completed'
      ? 'border-emerald-200 bg-emerald-50 text-emerald-700'
      : state === 'provider_error' || state === 'governance_denied'
        ? 'border-red-200 bg-red-50 text-red-700'
        : state === 'approval_required'
          ? 'border-amber-200 bg-amber-50 text-amber-700'
          : 'border-slate-200 bg-slate-50 text-slate-600'
  return (
    <span className={`inline-flex h-7 max-w-[150px] items-center truncate rounded-md border px-2 text-xs font-semibold ${tone}`}>
      {t(`assistant.state.${state}`)}
    </span>
  )
}
