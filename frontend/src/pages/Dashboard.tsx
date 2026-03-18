import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { monitorsApi } from '@/api/client'
import StatusBadge from '@/components/StatusBadge'
import { Activity, ArrowRight, XCircle } from 'lucide-react'
import type { MonitorStatus } from '@/api/types'

function StatCard({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div className="card p-5">
      <p className="text-sm text-gray-500 mb-1">{label}</p>
      <p className={`text-3xl font-bold ${color}`}>{value}</p>
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

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="w-6 h-6 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  return (
    <div className="p-6 max-w-6xl mx-auto">
      {/* Header */}
      <div className="mb-6">
        <h1 className="text-xl font-bold text-gray-100">Dashboard</h1>
        <p className="text-sm text-gray-500 mt-0.5">Overview of all your monitors</p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
        <StatCard label="Up" value={counts.UP} color="text-emerald-400" />
        <StatCard label="Down" value={counts.DOWN} color="text-red-400" />
        <StatCard label="Pending" value={counts.PENDING} color="text-yellow-400" />
        <StatCard label="Maintenance" value={counts.MAINTENANCE} color="text-blue-400" />
      </div>

      {/* Incidents */}
      {downMonitors.length > 0 && (
        <div className="mb-6">
          <h2 className="text-sm font-semibold text-gray-400 uppercase tracking-wider mb-3">
            🔴 Active Incidents
          </h2>
          <div className="space-y-2">
            {downMonitors.map((m) => (
              <Link
                key={m.ID}
                to={`/monitors/${m.ID}`}
                className="card p-4 flex items-center gap-3 hover:border-red-800/50 transition-colors border-red-900/30"
              >
                <XCircle className="w-4 h-4 text-red-400 flex-shrink-0" />
                <span className="font-medium text-gray-200 flex-1">{m.Name}</span>
                <span className="text-xs text-gray-500">{m.URL || m.Host || m.Domain}</span>
                <ArrowRight className="w-3.5 h-3.5 text-gray-600" />
              </Link>
            ))}
          </div>
        </div>
      )}

      {/* All monitors summary */}
      <div>
        <div className="flex items-center justify-between mb-3">
          <h2 className="text-sm font-semibold text-gray-400 uppercase tracking-wider">
            Monitors
          </h2>
          <Link to="/monitors" className="text-xs text-indigo-400 hover:text-indigo-300 flex items-center gap-1">
            View all <ArrowRight className="w-3 h-3" />
          </Link>
        </div>

        {monitors.length === 0 ? (
          <div className="card p-12 flex flex-col items-center text-center">
            <Activity className="w-10 h-10 text-gray-700 mb-3" />
            <p className="text-gray-400 font-medium">No monitors yet</p>
            <p className="text-sm text-gray-600 mt-1">Add your first monitor to start tracking uptime</p>
            <Link to="/monitors" className="btn-primary mt-4">Add monitor</Link>
          </div>
        ) : (
          <div className="space-y-2">
            {recentMonitors.map((m) => (
              <Link
                key={m.ID}
                to={`/monitors/${m.ID}`}
                className="card p-4 flex items-center gap-4 hover:border-gray-700 transition-colors"
              >
                <StatusBadge status={m.Status} size="sm" />
                <span className="flex-1 font-medium text-gray-200 truncate">{m.Name}</span>
                <span className="text-xs text-gray-600 hidden sm:block">
                  {m.URL || m.Host || m.Domain || '—'}
                </span>
                <ArrowRight className="w-3.5 h-3.5 text-gray-700 flex-shrink-0" />
              </Link>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
