import { Link } from 'react-router-dom'
import { Pause, Play, Trash2, ExternalLink, ShieldCheck, ShieldAlert } from 'lucide-react'
import StatusBadge from './StatusBadge'
import UptimeBars from './UptimeBars'
import type { Monitor } from '@/api/types'

/** Parses "TLS cert valid until 2025-06-01 (14 days)" from LastMessage.
 *  Returns { date, days } or null if not an HTTPS monitor / no cert info. */
function parseTlsExpiry(monitor: Monitor): { date: string; days: number } | null {
  if (monitor.Type !== 'http') return null
  const msg = monitor.LastMessage ?? ''
  const match = msg.match(/TLS cert valid until (\d{4}-\d{2}-\d{2}) \((\d+) days\)/)
  if (match) return { date: match[1], days: parseInt(match[2], 10) }
  // Expired case
  const expiredMatch = msg.match(/TLS certificate expired on (\d{4}-\d{2}-\d{2})/)
  if (expiredMatch) return { date: expiredMatch[1], days: 0 }
  return null
}

interface Props {
  monitor: Monitor
  onPause: (id: number) => void
  onResume: (id: number) => void
  onDelete: (id: number) => void
}

const typeLabel: Record<string, string> = {
  http: 'HTTP',
  port: 'Port',
  ping: 'Ping',
  keyword: 'Keyword',
  json_api: 'JSON API',
  udp: 'UDP',
  response_time: 'Response Time',
  dns: 'DNS',
  ssl: 'SSL',
  domain_expiry: 'Domain Expiry',
}

export default function MonitorCard({ monitor, onPause, onResume, onDelete }: Props) {
  const tls = parseTlsExpiry(monitor)

  return (
    <div className="card p-4 hover:border-gray-700 transition-colors group">
      {/* Single row: status | name+badges | bars | interval | actions */}
      <div className="flex items-center gap-3">
        {/* Status dot */}
        <div className="flex-shrink-0">
          <StatusBadge status={monitor.Status} />
        </div>

        {/* Name + type badge — fixed width on larger screens, shrinks on small */}
        <div className="flex items-center gap-2 min-w-0 flex-shrink-0 w-40 sm:w-52">
          <Link
            to={`/monitors/${monitor.ID}`}
            className="font-medium text-gray-100 hover:text-indigo-400 transition-colors truncate"
            title={monitor.Name}
          >
            {monitor.Name}
          </Link>
          <span className="text-xs text-gray-600 bg-gray-800 px-1.5 py-0.5 rounded flex-shrink-0 hidden sm:inline">
            {typeLabel[monitor.Type] ?? monitor.Type}
          </span>
          {!monitor.Active && (
            <span className="text-xs text-gray-500 bg-gray-800 px-1.5 py-0.5 rounded flex-shrink-0 hidden sm:inline">
              Paused
            </span>
          )}
        </div>

        {/* Uptime bars — fills remaining space */}
        <div className="flex-1 min-w-0">
          <UptimeBars monitorId={monitor.ID} compact />
        </div>

        {/* TLS cert expiry */}
        {tls && (
          <div className="relative flex-shrink-0 hidden md:flex group/tls">
            <span
              className={`flex items-center gap-1 text-xs cursor-default ${
                tls.days === 0
                  ? 'text-red-400'
                  : tls.days <= 7
                  ? 'text-yellow-400'
                  : 'text-gray-500'
              }`}
            >
              {tls.days <= 7 ? (
                <ShieldAlert className="w-3 h-3" />
              ) : (
                <ShieldCheck className="w-3 h-3" />
              )}
              {tls.days === 0 ? 'Cert expired' : `TLS ${tls.date}`}
            </span>
            {/* Hover tooltip */}
            <div className="absolute bottom-full right-0 mb-2 w-52 bg-gray-900 border border-gray-700 rounded-lg p-3 shadow-xl opacity-0 group-hover/tls:opacity-100 pointer-events-none transition-opacity z-50">
              <p className="text-xs font-semibold text-gray-200 mb-2">TLS Certificate</p>
              <div className="space-y-1">
                <div className="flex justify-between text-xs">
                  <span className="text-gray-500">Expires</span>
                  <span className="text-gray-200">{tls.date}</span>
                </div>
                <div className="flex justify-between text-xs">
                  <span className="text-gray-500">Days left</span>
                  <span className={
                    tls.days === 0 ? 'text-red-400 font-semibold' :
                    tls.days <= 7 ? 'text-yellow-400 font-semibold' :
                    tls.days <= 30 ? 'text-yellow-600' : 'text-emerald-400'
                  }>
                    {tls.days === 0 ? 'Expired' : `${tls.days} days`}
                  </span>
                </div>
              </div>
              {/* Arrow */}
              <div className="absolute right-3 bottom-0 translate-y-full w-0 h-0 border-l-4 border-r-4 border-t-4 border-l-transparent border-r-transparent border-t-gray-700" />
            </div>
          </div>
        )}

        {/* Interval */}
        <span className="text-xs text-gray-600 flex-shrink-0 hidden md:block">
          every {monitor.IntervalSeconds}s
        </span>

        {/* Actions */}
        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0">
          <Link
            to={`/monitors/${monitor.ID}`}
            className="p-1.5 rounded-lg text-gray-500 hover:text-gray-300 hover:bg-gray-800 transition-colors"
            title="View details"
          >
            <ExternalLink className="w-3.5 h-3.5" />
          </Link>
          {monitor.Active ? (
            <button
              onClick={() => onPause(monitor.ID)}
              className="p-1.5 rounded-lg text-gray-500 hover:text-yellow-400 hover:bg-gray-800 transition-colors"
              title="Pause"
            >
              <Pause className="w-3.5 h-3.5" />
            </button>
          ) : (
            <button
              onClick={() => onResume(monitor.ID)}
              className="p-1.5 rounded-lg text-gray-500 hover:text-emerald-400 hover:bg-gray-800 transition-colors"
              title="Resume"
            >
              <Play className="w-3.5 h-3.5" />
            </button>
          )}
          <button
            onClick={() => onDelete(monitor.ID)}
            className="p-1.5 rounded-lg text-gray-500 hover:text-red-400 hover:bg-gray-800 transition-colors"
            title="Delete"
          >
            <Trash2 className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>
    </div>
  )
}
