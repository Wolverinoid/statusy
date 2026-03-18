import { useParams, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { statusPagesApi } from '@/api/client'
import { monitorsApi } from '@/api/client'
import UptimeBars from '@/components/UptimeBars'
import { CheckCircle2, XCircle, Clock, Wrench, AlertTriangle, ArrowLeft } from 'lucide-react'
import type { Monitor } from '@/api/types'

// ── Status helpers ────────────────────────────────────────────────────────────

function StatusIcon({ status }: { status: Monitor['Status'] }) {
  switch (status) {
    case 'UP':          return <CheckCircle2 className="w-5 h-5 text-emerald-400" />
    case 'DOWN':        return <XCircle className="w-5 h-5 text-red-400" />
    case 'MAINTENANCE': return <Wrench className="w-5 h-5 text-blue-400" />
    default:            return <Clock className="w-5 h-5 text-gray-500" />
  }
}

function statusLabel(status: Monitor['Status']) {
  switch (status) {
    case 'UP':          return 'Operational'
    case 'DOWN':        return 'Outage'
    case 'MAINTENANCE': return 'Maintenance'
    default:            return 'Pending'
  }
}

function statusTextColor(status: Monitor['Status']) {
  switch (status) {
    case 'UP':          return 'text-emerald-400'
    case 'DOWN':        return 'text-red-400'
    case 'MAINTENANCE': return 'text-blue-400'
    default:            return 'text-gray-500'
  }
}

// ── Overall banner ────────────────────────────────────────────────────────────

function OverallBanner({ monitors }: { monitors: Monitor[] }) {
  if (!monitors.length) {
    return (
      <div className="rounded-xl bg-gray-800 border border-gray-700 p-5 flex items-center gap-3">
        <AlertTriangle className="w-6 h-6 text-gray-500" />
        <span className="font-medium text-gray-400">No monitors configured</span>
      </div>
    )
  }

  const allUp = monitors.every((m) => m.Status === 'UP')
  const anyDown = monitors.some((m) => m.Status === 'DOWN')

  if (allUp) {
    return (
      <div className="rounded-xl bg-emerald-500/10 border border-emerald-500/30 p-5 flex items-center gap-3">
        <CheckCircle2 className="w-6 h-6 text-emerald-400 flex-shrink-0" />
        <div>
          <p className="font-semibold text-emerald-400">All systems operational</p>
          <p className="text-sm text-emerald-400/70 mt-0.5">Everything is running smoothly</p>
        </div>
      </div>
    )
  }

  if (anyDown) {
    return (
      <div className="rounded-xl bg-red-500/10 border border-red-500/30 p-5 flex items-center gap-3">
        <XCircle className="w-6 h-6 text-red-400 flex-shrink-0" />
        <div>
          <p className="font-semibold text-red-400">Incident in progress</p>
          <p className="text-sm text-red-400/70 mt-0.5">
            {monitors.filter((m) => m.Status === 'DOWN').length} service{monitors.filter((m) => m.Status === 'DOWN').length !== 1 ? 's' : ''} affected
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="rounded-xl bg-yellow-500/10 border border-yellow-500/30 p-5 flex items-center gap-3">
      <AlertTriangle className="w-6 h-6 text-yellow-400 flex-shrink-0" />
      <div>
        <p className="font-semibold text-yellow-400">Partial outage</p>
        <p className="text-sm text-yellow-400/70 mt-0.5">Some services are degraded</p>
      </div>
    </div>
  )
}

// ── Monitor row ───────────────────────────────────────────────────────────────

function MonitorRow({ monitor }: { monitor: Monitor }) {
  // Fetch latest check for response time
  const { data: history = [] } = useQuery({
    queryKey: ['monitor-history-bars', monitor.ID],
    queryFn: () => monitorsApi.history(monitor.ID),
    staleTime: 60_000,
  })

  const latest = history[0]

  return (
    <div className="card p-4">
      <div className="flex items-center gap-3">
        <StatusIcon status={monitor.Status} />
        <div className="flex-1 min-w-0">
          <p className="font-medium text-gray-100">{monitor.Name}</p>
          {(monitor.URL || monitor.Host || monitor.Domain) && (
            <p className="text-xs text-gray-500 truncate mt-0.5">
              {monitor.URL || monitor.Host || monitor.Domain}
            </p>
          )}
        </div>
        <div className="text-right flex-shrink-0">
          <p className={`text-sm font-medium ${statusTextColor(monitor.Status)}`}>
            {statusLabel(monitor.Status)}
          </p>
          {latest?.ResponseTimeMs > 0 && (
            <p className="text-xs text-gray-500 mt-0.5">{latest.ResponseTimeMs}ms</p>
          )}
        </div>
      </div>
      <UptimeBars monitorId={monitor.ID} />
    </div>
  )
}

// ── Main view ─────────────────────────────────────────────────────────────────

export default function StatusPageView() {
  const { slug } = useParams<{ slug: string }>()

  const { data: page, isLoading, isError, error } = useQuery({
    queryKey: ['status-page-public', slug],
    queryFn: () => statusPagesApi.getPublic(slug!),
    refetchInterval: 30_000,
    retry: false,
  })

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gray-950 flex items-center justify-center">
        <div className="w-8 h-8 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  if (isError) {
    const msg = (error as { response?: { data?: { error?: string } } })?.response?.data?.error
    const is401 = msg?.includes('private') || msg?.includes('unauthorized')
    const is403 = msg?.includes('access denied')

    return (
      <div className="min-h-screen bg-gray-950 flex items-center justify-center p-6">
        <div className="text-center max-w-sm">
          {is401 || is403
            ? <Lock className="w-12 h-12 text-gray-600 mx-auto mb-4" />
            : <AlertTriangle className="w-12 h-12 text-gray-600 mx-auto mb-4" />
          }
          <h1 className="text-xl font-bold text-gray-200 mb-2">
            {is401 ? 'Private Status Page' : is403 ? 'Access Denied' : 'Page Not Found'}
          </h1>
          <p className="text-gray-500 text-sm mb-6">
            {is401 ? 'This status page requires authentication.' :
             is403 ? "You don't have access to this status page." :
             "The status page you're looking for doesn't exist."}
          </p>
          <Link to="/login" className="btn-primary">Sign in</Link>
        </div>
      </div>
    )
  }

  if (!page) return null

  const monitors = page.Monitors ?? []
  const lastUpdated = new Date().toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' })

  return (
    <div className="min-h-screen bg-gray-950">
      {/* Header */}
      <div className="border-b border-gray-800 bg-gray-900/50 backdrop-blur-sm sticky top-0 z-10">
        <div className="max-w-3xl mx-auto px-6 py-4 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-2 h-2 rounded-full bg-indigo-500 animate-pulse" />
            <span className="font-semibold text-gray-100">{page.Name}</span>
          </div>
          <span className="text-xs text-gray-500">Updated {lastUpdated}</span>
        </div>
      </div>

      {/* Content */}
      <div className="max-w-3xl mx-auto px-6 py-10 space-y-6">
        {/* Description */}
        {page.Description && (
          <p className="text-gray-400 text-sm">{page.Description}</p>
        )}

        {/* Overall status banner */}
        <OverallBanner monitors={monitors} />

        {/* Monitor list */}
        {monitors.length > 0 && (
          <div className="space-y-3">
            <h2 className="text-sm font-medium text-gray-500 uppercase tracking-wider">Services</h2>
            {monitors.map((m) => (
              <MonitorRow key={m.ID} monitor={m} />
            ))}
          </div>
        )}

        {/* Footer */}
        <div className="pt-6 border-t border-gray-800 flex items-center justify-between">
          <p className="text-xs text-gray-600">
            Powered by <span className="text-gray-500">Statusy</span>
          </p>
          <Link to="/" className="flex items-center gap-1.5 text-xs text-gray-600 hover:text-gray-400 transition-colors">
            <ArrowLeft className="w-3 h-3" /> Dashboard
          </Link>
        </div>
      </div>
    </div>
  )
}

// Need this for the lock icon in error state
function Lock({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M16.5 10.5V6.75a4.5 4.5 0 10-9 0v3.75m-.75 11.25h10.5a2.25 2.25 0 002.25-2.25v-6.75a2.25 2.25 0 00-2.25-2.25H6.75a2.25 2.25 0 00-2.25 2.25v6.75a2.25 2.25 0 002.25 2.25z" />
    </svg>
  )
}
