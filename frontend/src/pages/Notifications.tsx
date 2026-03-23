import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { notificationsApi } from '@/api/client'
import Modal from '@/components/Modal'
import ConfirmDialog from '@/components/ConfirmDialog'
import { Plus, Trash2, TestTube2, Mail, Send, Hash, Loader2, CheckCircle2, XCircle, Pencil } from 'lucide-react'
import type { Notification, NotificationFormData, NotificationType } from '@/api/types'

const TYPE_ICONS: Record<NotificationType, React.ReactNode> = {
  email:    <Mail className="w-4 h-4" />,
  telegram: <Send className="w-4 h-4" />,
  slack:    <Hash className="w-4 h-4" />,
}

/** Returns a human-readable summary of the config JSON for display in the card */
function configSummary(type: NotificationType, config: string): string {
  try {
    const c = JSON.parse(config)
    if (type === 'email') return c.to ?? '—'
    if (type === 'telegram') return c.chat_id ? `Chat: ${c.chat_id}` : '—'
    if (type === 'slack') return c.webhook_url ? 'Webhook configured' : '—'
  } catch { /* ignore */ }
  return '—'
}

const defaultForm = (): NotificationFormData => ({
  Name: '',
  Type: 'email',
  Config: '{}',
  Active: true,
  NotifyOnDown: true,
  NotifyOnUp: true,
  NotifyAfterFail: 1,
})

function NotifForm({
  initial, onSave, onCancel, saving,
}: {
  initial: NotificationFormData
  onSave: (d: NotificationFormData) => void
  onCancel: () => void
  saving: boolean
}) {
  const [form, setForm] = useState(initial)
  const [configObj, setConfigObj] = useState<Record<string, string>>(() => {
    try { return JSON.parse(form.Config) } catch { return {} }
  })

  const set = (k: keyof NotificationFormData, v: unknown) => setForm((f) => ({ ...f, [k]: v }))
  const setConfig = (k: string, v: string) => {
    const next = { ...configObj, [k]: v }
    setConfigObj(next)
    setForm((f) => ({ ...f, Config: JSON.stringify(next) }))
  }

  // Reset config when type changes
  const setType = (t: NotificationType) => {
    setConfigObj({})
    setForm((f) => ({ ...f, Type: t, Config: '{}' }))
  }

  return (
    <form onSubmit={(e) => { e.preventDefault(); onSave(form) }} className="space-y-4">
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="label">Name *</label>
          <input className="input" value={form.Name} onChange={(e) => set('Name', e.target.value)} required placeholder="My Telegram Alert" />
        </div>
        <div>
          <label className="label">Type *</label>
          <select className="input" value={form.Type} onChange={(e) => setType(e.target.value as NotificationType)}>
            <option value="email">Email (SMTP)</option>
            <option value="telegram">Telegram</option>
            <option value="slack">Slack</option>
          </select>
        </div>
      </div>

      {/* Type-specific config */}
      {form.Type === 'email' && (
        <div>
          <label className="label">Recipient Email *</label>
          <input className="input" type="email" value={configObj.to ?? ''} onChange={(e) => setConfig('to', e.target.value)} required placeholder="alerts@example.com" />
        </div>
      )}

      {form.Type === 'telegram' && (
        <div className="space-y-3">
          <div>
            <label className="label">Bot Token *</label>
            <input
              className="input font-mono text-sm"
              value={configObj.bot_token ?? ''}
              onChange={(e) => setConfig('bot_token', e.target.value)}
              required
              placeholder="123456789:AABBccDDeeFFggHH..."
            />
            <p className="text-xs text-gray-600 mt-1">
              Get it from <a href="https://t.me/BotFather" target="_blank" rel="noreferrer" className="text-indigo-400 hover:underline">@BotFather</a>. Overrides the global bot token from config.yaml.
            </p>
          </div>
          <div>
            <label className="label">Chat ID *</label>
            <input
              className="input"
              value={configObj.chat_id ?? ''}
              onChange={(e) => setConfig('chat_id', e.target.value)}
              required
              placeholder="-100123456789"
            />
            <p className="text-xs text-gray-600 mt-1">
              Use a negative ID for groups/channels. Get it via <span className="text-gray-400">@userinfobot</span>.
            </p>
          </div>
        </div>
      )}

      {form.Type === 'slack' && (
        <div>
          <label className="label">Webhook URL *</label>
          <input className="input" value={configObj.webhook_url ?? ''} onChange={(e) => setConfig('webhook_url', e.target.value)} required placeholder="https://hooks.slack.com/…" />
        </div>
      )}

      {/* Alert rules */}
      <div className="grid grid-cols-3 gap-3 pt-2 border-t border-gray-800">
        <div>
          <label className="label">Notify after fails</label>
          <input className="input" type="number" min={1} value={form.NotifyAfterFail} onChange={(e) => set('NotifyAfterFail', +e.target.value)} />
        </div>
        <div className="flex flex-col gap-2 pt-5">
          <label className="flex items-center gap-2 text-sm text-gray-400 cursor-pointer">
            <input type="checkbox" checked={form.NotifyOnDown} onChange={(e) => set('NotifyOnDown', e.target.checked)} className="accent-indigo-500" />
            Notify on Down
          </label>
          <label className="flex items-center gap-2 text-sm text-gray-400 cursor-pointer">
            <input type="checkbox" checked={form.NotifyOnUp} onChange={(e) => set('NotifyOnUp', e.target.checked)} className="accent-indigo-500" />
            Notify on Recovery
          </label>
        </div>
        <div className="flex flex-col gap-2 pt-5">
          <label className="flex items-center gap-2 text-sm text-gray-400 cursor-pointer">
            <input type="checkbox" checked={form.Active} onChange={(e) => set('Active', e.target.checked)} className="accent-indigo-500" />
            Active
          </label>
        </div>
      </div>

      <div className="flex justify-end gap-3 pt-2">
        <button type="button" className="btn-secondary" onClick={onCancel}>Cancel</button>
        <button type="submit" className="btn-primary" disabled={saving}>
          {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : 'Save'}
        </button>
      </div>
    </form>
  )
}

type TestState = 'idle' | 'loading' | 'ok' | 'error'

function NotifCard({
  n,
  onEdit,
  onDelete,
}: {
  n: Notification
  onEdit: () => void
  onDelete: () => void
}) {
  const [testState, setTestState] = useState<TestState>('idle')

  const handleTest = async () => {
    setTestState('loading')
    try {
      await notificationsApi.test(n.ID)
      setTestState('ok')
      setTimeout(() => setTestState('idle'), 3000)
    } catch {
      setTestState('error')
      setTimeout(() => setTestState('idle'), 3000)
    }
  }

  return (
    <div className="card p-4 flex items-center gap-4 hover:border-gray-700 transition-colors group">
      <div className={`w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0 ${n.Active ? 'bg-indigo-600/20 text-indigo-400' : 'bg-gray-800 text-gray-600'}`}>
        {TYPE_ICONS[n.Type]}
      </div>

      <div className="flex-1 min-w-0">
        <p className="font-medium text-gray-200">{n.Name}</p>
        <p className="text-xs text-gray-500">
          <span className="capitalize">{n.Type}</span>
          <span className="mx-1">·</span>
          <span>{configSummary(n.Type, n.Config)}</span>
          <span className="mx-1">·</span>
          <span className={n.Active ? 'text-emerald-500' : 'text-gray-600'}>{n.Active ? 'Active' : 'Disabled'}</span>
        </p>
      </div>

      {/* Test button — always visible */}
      <button
        onClick={handleTest}
        disabled={testState === 'loading'}
        className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors ${
          testState === 'ok'
            ? 'bg-emerald-600/20 text-emerald-400'
            : testState === 'error'
            ? 'bg-red-600/20 text-red-400'
            : 'bg-gray-800 text-gray-400 hover:text-indigo-400 hover:bg-indigo-600/10'
        }`}
        title="Send test notification"
      >
        {testState === 'loading' && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
        {testState === 'ok' && <CheckCircle2 className="w-3.5 h-3.5" />}
        {testState === 'error' && <XCircle className="w-3.5 h-3.5" />}
        {testState === 'idle' && <TestTube2 className="w-3.5 h-3.5" />}
        {testState === 'loading' ? 'Sending…' : testState === 'ok' ? 'Sent!' : testState === 'error' ? 'Failed' : 'Test'}
      </button>

      {/* Edit + Delete — visible on hover */}
      <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
        <button
          onClick={onEdit}
          className="p-1.5 rounded-lg text-gray-500 hover:text-gray-300 hover:bg-gray-800 transition-colors"
          title="Edit"
        >
          <Pencil className="w-3.5 h-3.5" />
        </button>
        <button
          onClick={onDelete}
          className="p-1.5 rounded-lg text-gray-500 hover:text-red-400 hover:bg-gray-800 transition-colors"
          title="Delete"
        >
          <Trash2 className="w-3.5 h-3.5" />
        </button>
      </div>
    </div>
  )
}

export default function Notifications() {
  const qc = useQueryClient()
  const [showForm, setShowForm] = useState(false)
  const [editId, setEditId] = useState<number | null>(null)
  const [deleteId, setDeleteId] = useState<number | null>(null)

  const { data: notifs = [], isLoading } = useQuery({
    queryKey: ['notifications'],
    queryFn: notificationsApi.list,
  })

  const invalidate = () => qc.invalidateQueries({ queryKey: ['notifications'] })

  const createMut = useMutation({ mutationFn: notificationsApi.create, onSuccess: () => { invalidate(); setShowForm(false) } })
  const updateMut = useMutation({
    mutationFn: ({ id, data }: { id: number; data: NotificationFormData }) => notificationsApi.update(id, data),
    onSuccess: () => { invalidate(); setEditId(null) },
  })
  const deleteMut = useMutation({ mutationFn: notificationsApi.delete, onSuccess: () => { invalidate(); setDeleteId(null) } })

  const editNotif = notifs.find((n) => n.ID === editId)

  return (
    <div className="p-6 max-w-3xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-bold text-gray-100">Notifications</h1>
          <p className="text-sm text-gray-500 mt-0.5">Alert channels for monitor events</p>
        </div>
        <button className="btn-primary" onClick={() => setShowForm(true)}>
          <Plus className="w-4 h-4" /> Add channel
        </button>
      </div>

      {isLoading ? (
        <div className="flex justify-center py-16">
          <div className="w-6 h-6 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
        </div>
      ) : notifs.length === 0 ? (
        <div className="card p-12 flex flex-col items-center text-center">
          <p className="text-gray-400 font-medium">No notification channels yet</p>
          <p className="text-sm text-gray-600 mt-1">Add a channel to receive alerts when monitors go down</p>
          <button className="btn-primary mt-4" onClick={() => setShowForm(true)}>Add channel</button>
        </div>
      ) : (
        <div className="space-y-2">
          {notifs.map((n) => (
            <NotifCard
              key={n.ID}
              n={n}
              onEdit={() => setEditId(n.ID)}
              onDelete={() => setDeleteId(n.ID)}
            />
          ))}
        </div>
      )}

      {showForm && (
        <Modal title="Add Notification Channel" onClose={() => setShowForm(false)}>
          <NotifForm initial={defaultForm()} onSave={(d) => createMut.mutate(d)} onCancel={() => setShowForm(false)} saving={createMut.isPending} />
        </Modal>
      )}

      {editId !== null && editNotif && (
        <Modal title="Edit Notification Channel" onClose={() => setEditId(null)}>
          <NotifForm
            initial={editNotif as unknown as NotificationFormData}
            onSave={(d) => updateMut.mutate({ id: editId, data: d })}
            onCancel={() => setEditId(null)}
            saving={updateMut.isPending}
          />
        </Modal>
      )}

      {deleteId !== null && (
        <ConfirmDialog
          title="Delete Channel"
          message="This notification channel will be removed from all monitors. Continue?"
          confirmLabel="Delete"
          danger
          onConfirm={() => deleteMut.mutate(deleteId)}
          onCancel={() => setDeleteId(null)}
        />
      )}
    </div>
  )
}
