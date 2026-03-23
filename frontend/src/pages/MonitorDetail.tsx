import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { monitorsApi, notificationsApi } from '@/api/client'
import StatusBadge from '@/components/StatusBadge'
import { ArrowLeft, AlertCircle, Bell, BellOff } from 'lucide-react'
import { LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid } from 'recharts'

function formatDate(iso: string) {
  return new Date(iso).toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

export default function MonitorDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const monitorId = Number(id)
  const qc = useQueryClient()

  const { data: monitor, isLoading: loadingMonitor } = useQuery({
    queryKey: ['monitor', monitorId],
    queryFn: () => monitorsApi.get(monitorId),
    refetchInterval: 30_000,
  })

  const { data: history = [] } = useQuery({
    queryKey: ['monitor-history', monitorId],
    queryFn: () => monitorsApi.history(monitorId),
    refetchInterval: 30_000,
  })

  const { data: allNotifs = [] } = useQuery({
    queryKey: ['notifications'],
    queryFn: notificationsApi.list,
  })

  const setNotifsMut = useMutation({
    mutationFn: (ids: number[]) => monitorsApi.setNotifications(monitorId, ids),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['monitor', monitorId] }),
  })

  const linkedIds = new Set((monitor?.Notifications ?? []).map((n) => n.ID))

  const toggleNotif = (notifId: number) => {
    const next = new Set(linkedIds)
    if (next.has(notifId)) next.delete(notifId)
    else next.add(notifId)
    setNotifsMut.mutate([...next])
  }

  if (loadingMonitor) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="w-6 h-6 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
      </div>
    )
  }

  if (!monitor) {
    return (
      <div className="p-6 flex flex-col items-center gap-3 text-gray-500">
        <AlertCircle className="w-8 h-8" />
        <p>Monitor not found</p>
      </div>
    )
  }

  const chartData = [...history]
    .reverse()
    .slice(-50)
    .map((r) => ({
      time: formatDate(r.CheckedAt),
      ms: r.ResponseTimeMs,
      status: r.Status,
    }))

  const uptime = history.length
    ? Math.round((history.filter((r) => r.Status === 'UP').length / history.length) * 100)
    : null

  const avgMs = history.length
    ? Math.round(history.reduce((s, r) => s + r.ResponseTimeMs, 0) / history.length)
    : null

  return (
    <div className="p-6 max-w-4xl mx-auto">
      {/* Back */}
      <button
        onClick={() => navigate('/monitors')}
        className="flex items-center gap-2 text-sm text-gray-500 hover:text-gray-300 mb-5 transition-colors"
      >
        <ArrowLeft className="w-4 h-4" /> Back to monitors
      </button>

      {/* Header */}
      <div className="flex items-start justify-between mb-6">
        <div>
          <div className="flex items-center gap-3 mb-1">
            <h1 className="text-xl font-bold text-gray-100">{monitor.Name}</h1>
            <StatusBadge status={monitor.Status} />
          </div>
          <p className="text-sm text-gray-500">{monitor.URL || monitor.Host || monitor.Domain || monitor.DNSHost}</p>
        </div>
      </div>

      {/* Stats row */}
      <div className="grid grid-cols-3 gap-4 mb-6">
        <div className="card p-4">
          <p className="text-xs text-gray-500 mb-1">Uptime (last 100)</p>
          <p className="text-2xl font-bold text-emerald-400">{uptime !== null ? `${uptime}%` : '—'}</p>
        </div>
        <div className="card p-4">
          <p className="text-xs text-gray-500 mb-1">Avg Response</p>
          <p className="text-2xl font-bold text-indigo-400">{avgMs !== null ? `${avgMs}ms` : '—'}</p>
        </div>
        <div className="card p-4">
          <p className="text-xs text-gray-500 mb-1">Check Interval</p>
          <p className="text-2xl font-bold text-gray-300">{monitor.IntervalSeconds}s</p>
        </div>
      </div>

      {/* Response time chart */}
      {chartData.length > 0 && (
        <div className="card p-5 mb-6">
          <h2 className="text-sm font-semibold text-gray-400 mb-4">Response Time</h2>
          <ResponsiveContainer width="100%" height={180}>
            <LineChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#1f2937" />
              <XAxis dataKey="time" tick={{ fontSize: 10, fill: '#6b7280' }} tickLine={false} axisLine={false} interval="preserveStartEnd" />
              <YAxis tick={{ fontSize: 10, fill: '#6b7280' }} tickLine={false} axisLine={false} unit="ms" width={45} />
              <Tooltip
                contentStyle={{ background: '#111827', border: '1px solid #1f2937', borderRadius: 8, fontSize: 12 }}
                labelStyle={{ color: '#9ca3af' }}
                itemStyle={{ color: '#818cf8' }}
              />
              <Line type="monotone" dataKey="ms" stroke="#818cf8" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}

      {/* Notification channels */}
      <div className="card p-5 mb-6">
        <h2 className="text-sm font-semibold text-gray-400 mb-3 flex items-center gap-2">
          <Bell className="w-4 h-4" /> Notification Channels
        </h2>
        {allNotifs.length === 0 ? (
          <p className="text-sm text-gray-600">No channels configured. <a href="/notifications" className="text-indigo-400 hover:underline">Add one</a>.</p>
        ) : (
          <div className="flex flex-wrap gap-2">
            {allNotifs.map((n) => {
              const active = linkedIds.has(n.ID)
              return (
                <button
                  key={n.ID}
                  onClick={() => toggleNotif(n.ID)}
                  disabled={setNotifsMut.isPending}
                  className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium border transition-colors ${
                    active
                      ? 'bg-indigo-600/20 border-indigo-500/50 text-indigo-300'
                      : 'bg-gray-800 border-gray-700 text-gray-500 hover:text-gray-300'
                  }`}
                >
                  {active ? <Bell className="w-3 h-3" /> : <BellOff className="w-3 h-3" />}
                  {n.Name}
                </button>
              )
            })}
          </div>
        )}
      </div>

      {/* History table */}
      <div className="card overflow-hidden">
        <div className="px-5 py-4 border-b border-gray-800">
          <h2 className="text-sm font-semibold text-gray-400">Check History</h2>
        </div>
        {history.length === 0 ? (
          <div className="p-8 text-center text-gray-600 text-sm">No checks yet</div>
        ) : (
          <div className="divide-y divide-gray-800/50">
            {history.slice(0, 50).map((r) => (
              <div key={r.ID} className="flex items-center gap-4 px-5 py-3 text-sm">
                <StatusBadge status={r.Status} size="sm" />
                <span className="text-gray-500 flex-1 truncate">{r.Message || '—'}</span>
                <span className="text-gray-600 flex-shrink-0">{r.ResponseTimeMs}ms</span>
                <span className="text-gray-600 flex-shrink-0 text-xs">{formatDate(r.CheckedAt)}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
