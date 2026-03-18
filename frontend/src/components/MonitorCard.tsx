import { Link } from 'react-router-dom'
import { Pause, Play, Trash2, ExternalLink } from 'lucide-react'
import StatusBadge from './StatusBadge'
import UptimeBars from './UptimeBars'
import type { Monitor } from '@/api/types'

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
