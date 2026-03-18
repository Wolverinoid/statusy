import clsx from 'clsx'
import type { MonitorStatus } from '@/api/types'

interface Props {
  status: MonitorStatus
  size?: 'sm' | 'md'
}

const config: Record<MonitorStatus, { dot: string; text: string; label: string }> = {
  UP:          { dot: 'bg-emerald-400', text: 'text-emerald-400', label: 'Up' },
  DOWN:        { dot: 'bg-red-400 animate-pulse', text: 'text-red-400', label: 'Down' },
  PENDING:     { dot: 'bg-yellow-400', text: 'text-yellow-400', label: 'Pending' },
  MAINTENANCE: { dot: 'bg-blue-400', text: 'text-blue-400', label: 'Maintenance' },
}

export default function StatusBadge({ status, size = 'md' }: Props) {
  const { dot, text, label } = config[status] ?? config.PENDING
  return (
    <span className={clsx('inline-flex items-center gap-1.5 font-medium', text, size === 'sm' ? 'text-xs' : 'text-sm')}>
      <span className={clsx('rounded-full flex-shrink-0', dot, size === 'sm' ? 'w-1.5 h-1.5' : 'w-2 h-2')} />
      {label}
    </span>
  )
}
