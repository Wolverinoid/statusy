import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { notificationsApi } from '@/api/client'
import Modal from '@/components/Modal'
import ConfirmDialog from '@/components/ConfirmDialog'
import { Plus, Trash2, TestTube2, Mail, Send, Hash, Loader2 } from 'lucide-react'
import type { NotificationFormData, NotificationType } from '@/api/types'

const TYPE_ICONS: Record<NotificationType, React.ReactNode> = {
  email:    <Mail className="w-4 h-4" />,
  telegram: <Send className="w-4 h-4" />,
  slack:    <Hash className="w-4 h-4" />,
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

  return (
    <form onSubmit={(e) => { e.preventDefault(); onSave(form) }} className="space-y-4">
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="label">Name *</label>
          <input className="input" value={form.Name} onChange={(e) => set('Name', e.target.value)} required placeholder="My Email Alert" />
        </div>
        <div>
          <label className="label">Type *</label>
          <select className="input" value={form.Type} onChange={(e) => set('Type', e.target.value as NotificationType)}>
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
        <div>
          <label className="label">Chat ID *</label>
          <input className="input" value={configObj.chat_id ?? ''} onChange={(e) => setConfig('chat_id', e.target.value)} required placeholder="-100123456789" />
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

export default function Notifications() {
  const qc = useQueryClient()
  const [showForm, setShowForm] = useState(false)
  const [editId, setEditId] = useState<number | null>(null)
  const [deleteId, setDeleteId] = useState<number | null>(null)
  const [testingId, setTestingId] = useState<number | null>(null)

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
  const testMut = useMutation({
    mutationFn: notificationsApi.test,
    onSettled: () => setTestingId(null),
  })

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
            <div key={n.ID} className="card p-4 flex items-center gap-4 group hover:border-gray-700 transition-colors">
              <div className={`w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0 ${n.Active ? 'bg-indigo-600/20 text-indigo-400' : 'bg-gray-800 text-gray-600'}`}>
                {TYPE_ICONS[n.Type]}
              </div>
              <div className="flex-1 min-w-0">
                <p className="font-medium text-gray-200">{n.Name}</p>
                <p className="text-xs text-gray-500 capitalize">{n.Type} · {n.Active ? 'Active' : 'Disabled'}</p>
              </div>
              <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                <button
                  onClick={() => { setTestingId(n.ID); testMut.mutate(n.ID) }}
                  disabled={testingId === n.ID}
                  className="p-1.5 rounded-lg text-gray-500 hover:text-indigo-400 hover:bg-gray-800 transition-colors"
                  title="Send test"
                >
                  {testingId === n.ID ? <Loader2 className="w-3.5 h-3.5 animate-spin" /> : <TestTube2 className="w-3.5 h-3.5" />}
                </button>
                <button
                  onClick={() => setDeleteId(n.ID)}
                  className="p-1.5 rounded-lg text-gray-500 hover:text-red-400 hover:bg-gray-800 transition-colors"
                  title="Delete"
                >
                  <Trash2 className="w-3.5 h-3.5" />
                </button>
              </div>
            </div>
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
