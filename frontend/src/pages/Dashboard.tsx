import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { monitorsApi } from '@/api/client'
import StatusBadge from '@/components/StatusBadge'
import { Activity, ArrowRight, XCircle, TrendingUp } from 'lucide-react'
import type { MonitorStatus } from '@/api/types'

interface StatCardProps {
  label: string
  value: number
  color: string
  borderColor: string
  glowClass?: string
  dotColor: string
}

function StatCard({ label, value, color, borderColor, glowClass, dotColor }: StatCardProps) {
  return (
    <div className={`stat-card border-l-2 ${borderColor}`}>
      {/* Corner decoration */}
      <div className="absolute top-0 right-0 w-16 h-16 opacity-5">
        <div className={`w-full h-full rounded-bl-full ${dotColor.replace('bg-', 'bg-')}`} />
      </div>
      <div className="flex items-start justify-between">
        <div>
          <p className="text-[10px] font-mono text-gray-600 uppercase tracking-widest mb-2">{label}</p>
          <p className={`text-3xl font-bold font-mono tabular-nums ${color}`}>{value}</p>
        </div>
        <div className={`w-2 h-2 rounded-full mt-1 ${dotColor} ${glowClass ?? ''}`} />
      </div>
    </div>
  )
}

export default function Dashboard() {
  const { data: monitors = [], isLoading } = useQuery({
    queryKey: ['monitors'],
    queryFn: monitorsApi.list,
    refetchInterval: 30_000,
  })

  const counts: Record<MonitorStatus, number> = {
    UP: 0, DOWN: 0, PENDING: 0, MAINTENANCE: 0,
  }
  monitors.forEach((m) => { counts[m.Status] = (counts[m.Status] ?? 0) + 1 })

  const downMonitors = monitors.filter((m) => m.Status === 'DOWN')
  const recentMonitors = [...monitors].slice(0, 8)
  const uptimePct = monitors.length
    ? Math.round((counts.UP / monitors.length) * 100)
    : null

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="w-5 h-5 border-2 border-cyan-500 border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  return (
    <div className="p-6 max-w-5xl mx-auto">
      {/* Header */}
      <div className="mb-7 flex items-start justify-between">
        <div>
          <h1 className="text-xl font-bold dark:text-gray-100 text-gray-900 tracking-tight">Dashboard</h1>
          <p className="text-xs dark:text-gray-600 text-slate-400 mt-1 font-mono">
            {monitors.length} monitor{monitors.length !== 1 ? 's' : ''} tracked
            {uptimePct !== null && (
              <span className="ml-2 text-cyan-600">· {uptimePct}% up</span>
            )}
          </p>
        </div>
        {uptimePct !== null && (
          <div className="flex items-center gap-1.5 text-xs dark:text-gray-600 text-slate-400">
            <TrendingUp className="w-3.5 h-3.5 text-cyan-600" />
            <span className="font-mono text-cyan-600 font-semibold">{uptimePct}%</span>
            <span className="dark:text-gray-700 text-slate-400">overall uptime</span>
          </div>
        )}
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 mb-7">
        <StatCard label="Operational" value={counts.UP} color="text-emerald-400" borderColor="border-l-emerald-500/50" dotColor="bg-emerald-400" glowClass="glow-green" />
        <StatCard label="Down" value={counts.DOWN} color="text-red-400" borderColor="border-l-red-500/50" dotColor="bg-red-400" glowClass={counts.DOWN > 0 ? 'glow-red animate-pulse' : ''} />
        <StatCard label="Pending" value={counts.PENDING} color="text-yellow-400" borderColor="border-l-yellow-500/50" dotColor="bg-yellow-400" />
        <StatCard label="Maintenance" value={counts.MAINTENANCE} color="text-blue-400" borderColor="border-l-blue-500/50" dotColor="bg-blue-400" />
      </div>

      {/* Active Incidents */}
      {downMonitors.length > 0 && (
        <div className="mb-7">
          <div className="flex items-center gap-2 mb-3">
            <span className="w-1.5 h-1.5 rounded-full bg-red-400 animate-pulse" />
            <h2 className="section-header text-red-600">Active Incidents</h2>
            <span className="tag text-red-500 border-red-800/40">{downMonitors.length}</span>
          </div>
          <div className="space-y-1.5">
            {downMonitors.map((m) => (
              <Link
                key={m.ID}
                to={`/monitors/${m.ID}`}
                className="card p-3.5 flex items-center gap-3 hover:border-red-800/40 transition-all duration-200 border-l-2 border-l-red-500/50 dark:bg-red-950/10 bg-red-50/60 group"
              >
                <XCircle className="w-4 h-4 text-red-400 flex-shrink-0" />
                <span className="font-medium dark:text-gray-200 text-gray-700 flex-1 text-sm">{m.Name}</span>
                <span className="text-xs dark:text-gray-600 text-slate-400 truncate max-w-xs hidden sm:block font-mono">{m.URL || m.Host || m.Domain}</span>
                <ArrowRight className="w-3.5 h-3.5 text-gray-700 group-hover:text-red-500 transition-colors flex-shrink-0" />
              </Link>
            ))}
          </div>
        </div>
      )}

      {/* Monitor list */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h2 className="section-header">Monitors</h2>
          <Link to="/monitors" className="text-xs text-cyan-600 hover:text-cyan-400 flex items-center gap-1 transition-colors font-mono">
            view all <ArrowRight className="w-3 h-3" />
          </Link>
        </div>

        {monitors.length === 0 ? (
          <div className="card p-12 flex flex-col items-center text-center">
            <div className="w-14 h-14 rounded-xl bg-gray-800/60 border border-gray-700/40 flex items-center justify-center mb-4">
              <Activity className="w-7 h-7 text-gray-700" />
            </div>
            <p className="text-gray-400 font-medium text-sm">No monitors yet</p>
            <p className="text-xs text-gray-700 mt-1 mb-5 font-mono">Add your first monitor to start tracking uptime</p>
            <Link to="/monitors" className="btn-primary">Add monitor</Link>
          </div>
        ) : (
          <div className="space-y-1.5">
            {recentMonitors.map((m) => (
              <Link
                key={m.ID}
                to={`/monitors/${m.ID}`}
                className="card p-3.5 flex items-center gap-3 hover:border-gray-700/60 hover:bg-gray-900/80 transition-all duration-200 group"
              >
                <StatusBadge status={m.Status} size="sm" />
                <span className="flex-1 font-medium dark:text-gray-300 text-gray-700 truncate text-sm dark:group-hover:text-gray-100 group-hover:text-gray-900 transition-colors">{m.Name}</span>
                <span className="text-xs dark:text-gray-700 text-slate-400 hidden sm:block truncate max-w-xs font-mono">
                  {m.URL || m.Host || m.Domain || '—'}
                </span>
                <ArrowRight className="w-3.5 h-3.5 text-gray-700 group-hover:text-gray-500 transition-colors flex-shrink-0" />
              </Link>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
