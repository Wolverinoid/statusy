import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { monitorsApi } from '@/api/client'
import type { CheckResult, MonitorStatus } from '@/api/types'

const BAR_COUNT = 30

interface TooltipState {
  x: number
  y: number
  result: CheckResult
}

function statusColor(status: MonitorStatus): string {
  switch (status) {
    case 'UP':          return 'bg-emerald-500'
    case 'DOWN':        return 'bg-red-500'
    case 'MAINTENANCE': return 'bg-blue-500'
    default:            return 'bg-gray-700'
  }
}

function statusColorHover(status: MonitorStatus): string {
  switch (status) {
    case 'UP':          return 'hover:bg-emerald-400'
    case 'DOWN':        return 'hover:bg-red-400'
    case 'MAINTENANCE': return 'hover:bg-blue-400'
    default:            return 'hover:bg-gray-600'
  }
}

function formatDate(iso: string) {
  return new Date(iso).toLocaleString(undefined, {
    month: 'short', day: 'numeric',
    hour: '2-digit', minute: '2-digit',
  })
}

interface Props {
  monitorId: number
}

export default function UptimeBars({ monitorId }: Props) {
  const navigate = useNavigate()
  const [tooltip, setTooltip] = useState<TooltipState | null>(null)

  const { data: history = [], isLoading } = useQuery({
    queryKey: ['monitor-history-bars', monitorId],
    queryFn: () => monitorsApi.history(monitorId),
    staleTime: 60_000,
    refetchInterval: 60_000,
  })

  // Most recent BAR_COUNT checks, oldest→newest left→right
  const bars: (CheckResult | null)[] = (() => {
    const recent = [...history].slice(0, BAR_COUNT).reverse()
    // Pad left with nulls if fewer than BAR_COUNT
    const padding = Array(Math.max(0, BAR_COUNT - recent.length)).fill(null)
    return [...padding, ...recent]
  })()

  const uptime = history.length
    ? Math.round((history.filter((r) => r.Status === 'UP').length / history.length) * 100)
    : null

  if (isLoading) {
    return (
      <div className="flex gap-0.5 mt-2">
        {Array(BAR_COUNT).fill(null).map((_, i) => (
          <div key={i} className="flex-1 h-5 rounded-sm bg-gray-800 animate-pulse" />
        ))}
      </div>
    )
  }

  return (
    <div className="mt-2 relative">
      {/* Bars row */}
      <div
        className="flex gap-0.5 cursor-pointer"
        onClick={() => navigate(`/monitors/${monitorId}`)}
        title="View details"
      >
        {bars.map((result, i) =>
          result === null ? (
            <div
              key={i}
              className="flex-1 h-5 rounded-sm bg-gray-800/50"
            />
          ) : (
            <div
              key={i}
              className={`flex-1 h-5 rounded-sm transition-all duration-100 ${statusColor(result.Status)} ${statusColorHover(result.Status)}`}
              onMouseEnter={(e) => {
                const rect = e.currentTarget.getBoundingClientRect()
                setTooltip({ x: rect.left + rect.width / 2, y: rect.top, result })
              }}
              onMouseLeave={() => setTooltip(null)}
            />
          )
        )}
      </div>

      {/* Footer: label + uptime % */}
      <div className="flex items-center justify-between mt-1">
        <span className="text-xs text-gray-600">Last {BAR_COUNT} checks</span>
        {uptime !== null && (
          <span className={`text-xs font-medium ${uptime >= 99 ? 'text-emerald-400' : uptime >= 90 ? 'text-yellow-400' : 'text-red-400'}`}>
            {uptime}% uptime
          </span>
        )}
      </div>

      {/* Tooltip — rendered via portal-like fixed positioning */}
      {tooltip && (
        <div
          className="fixed z-50 pointer-events-none"
          style={{ left: tooltip.x, top: tooltip.y - 8, transform: 'translate(-50%, -100%)' }}
        >
          <div className="bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-xs shadow-xl whitespace-nowrap">
            <div className="flex items-center gap-2 mb-1">
              <span className={`w-2 h-2 rounded-full flex-shrink-0 ${statusColor(tooltip.result.Status)}`} />
              <span className="font-semibold text-gray-100">{tooltip.result.Status}</span>
              {tooltip.result.ResponseTimeMs > 0 && (
                <span className="text-gray-400">{tooltip.result.ResponseTimeMs}ms</span>
              )}
            </div>
            {tooltip.result.Message && (
              <p className="text-gray-400 max-w-[200px] truncate">{tooltip.result.Message}</p>
            )}
            <p className="text-gray-600 mt-0.5">{formatDate(tooltip.result.CheckedAt)}</p>
          </div>
          {/* Arrow */}
          <div className="absolute left-1/2 -translate-x-1/2 bottom-0 translate-y-full w-0 h-0 border-l-4 border-r-4 border-t-4 border-l-transparent border-r-transparent border-t-gray-700" />
        </div>
      )}
    </div>
  )
}
