import clsx from 'clsx'
import type { MonitorStatus } from '@/api/types'

interface Props {
  status: MonitorStatus
  size?: 'sm' | 'md'
}

const config: Record<MonitorStatus, { dot: string; pill: string; label: string; glow: string }> = {
  UP:          { dot: 'bg-emerald-400', glow: 'glow-green', pill: 'bg-emerald-500/8 text-emerald-400 border border-emerald-500/20', label: 'Up' },
  DOWN:        { dot: 'bg-red-400 animate-pulse', glow: 'glow-red', pill: 'bg-red-500/8 text-red-400 border border-red-500/20', label: 'Down' },
  PENDING:     { dot: 'bg-yellow-400', glow: '', pill: 'bg-yellow-500/8 text-yellow-400 border border-yellow-500/20', label: 'Pending' },
  MAINTENANCE: { dot: 'bg-blue-400', glow: '', pill: 'bg-blue-500/8 text-blue-400 border border-blue-500/20', label: 'Maint.' },
}

export default function StatusBadge({ status, size = 'md' }: Props) {
  const { dot, glow, pill, label } = config[status] ?? config.PENDING
  return (
    <span className={clsx(
      'inline-flex items-center gap-1.5 font-mono font-medium rounded-md',
      pill,
      size === 'sm' ? 'text-[10px] px-1.5 py-0.5' : 'text-xs px-2 py-1',
    )}>
      <span className={clsx('rounded-full flex-shrink-0 w-1.5 h-1.5', dot, glow)} />
      {label}
    </span>
  )
}
