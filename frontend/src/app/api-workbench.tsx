'use client'

import { useMemo, useState } from 'react'
import { useI18n } from '@/lib/i18n'
import { ApiOperation, apiOperations, operationDomains } from '@/lib/operations'
import { MethodBadge, OperationRunner } from './operation-runner'

interface ApiWorkbenchProps {
  token: string
  domain?: string
  showDomainMenu?: boolean
}

export function ApiWorkbench({ token, domain, showDomainMenu = true }: ApiWorkbenchProps) {
  const { t } = useI18n()
  const firstOperation = domain
    ? apiOperations.find((operation) => operation.domain === domain) ?? apiOperations[0]
    : apiOperations[0]
  const [activeDomain, setActiveDomain] = useState(firstOperation.domain)
  const [selectedOperation, setSelectedOperation] = useState<ApiOperation>(firstOperation)

  const domainOperations = useMemo(
    () => apiOperations.filter((operation) => operation.domain === activeDomain),
    [activeDomain],
  )

  function selectDomain(domain: string) {
    const nextOperation = apiOperations.find((operation) => operation.domain === domain) ?? firstOperation
    setActiveDomain(domain)
    setSelectedOperation(nextOperation)
  }

  function selectOperation(operation: ApiOperation) {
    setSelectedOperation(operation)
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

      <OperationRunner key={selectedOperation.id} token={token} operation={selectedOperation} />
    </div>
  )
}
