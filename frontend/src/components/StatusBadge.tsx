import clsx from 'clsx'
import type { MonitorStatus } from '@/api/types'

interface Props {
  status: MonitorStatus
  size?: 'sm' | 'md'
}

const config: Record<MonitorStatus, { dot: string; pill: string; label: string; glow: string }> = {
  UP:          { dot: 'bg-emerald-400', glow: 'glow-green', pill: 'dark:bg-emerald-500/10 bg-emerald-50 dark:text-emerald-400 text-emerald-600 dark:border-emerald-500/20 border-emerald-200', label: 'Up' },
  DOWN:        { dot: 'bg-red-400 animate-pulse', glow: 'glow-red', pill: 'dark:bg-red-500/10 bg-red-50 dark:text-red-400 text-red-600 dark:border-red-500/20 border-red-200', label: 'Down' },
  PENDING:     { dot: 'bg-yellow-400', glow: '', pill: 'dark:bg-yellow-500/10 bg-yellow-50 dark:text-yellow-400 text-yellow-600 dark:border-yellow-500/20 border-yellow-200', label: 'Pending' },
  MAINTENANCE: { dot: 'bg-blue-400', glow: '', pill: 'dark:bg-blue-500/10 bg-blue-50 dark:text-blue-400 text-blue-600 dark:border-blue-500/20 border-blue-200', label: 'Maint.' },
}

export default function StatusBadge({ status, size = 'md' }: Props) {
  const { dot, glow, pill, label } = config[status] ?? config.PENDING
  return (
    <span className={clsx(
      'inline-flex items-center gap-1.5 font-mono font-medium rounded-md border',
      pill,
      size === 'sm' ? 'text-[10px] px-1.5 py-0.5' : 'text-xs px-2 py-1',
    )}>
      <span className={clsx('rounded-full flex-shrink-0 w-1.5 h-1.5', dot, glow)} />
      {label}
    </span>
  )
}
