import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { monitorsApi } from '@/api/client'
import MonitorCard from '@/components/MonitorCard'
import Modal from '@/components/Modal'
import ConfirmDialog from '@/components/ConfirmDialog'
import { Plus, Search, Loader2 } from 'lucide-react'
import type { MonitorFormData, MonitorType } from '@/api/types'

const MONITOR_TYPES: { value: MonitorType; label: string }[] = [
  { value: 'http', label: 'HTTP / HTTPS' },
  { value: 'keyword', label: 'Keyword' },
  { value: 'json_api', label: 'JSON API' },
  { value: 'port', label: 'TCP Port' },
  { value: 'ping', label: 'Ping' },
  { value: 'dns', label: 'DNS' },
  { value: 'ssl', label: 'SSL Certificate' },
  { value: 'domain_expiry', label: 'Domain Expiry' },
  { value: 'response_time', label: 'Response Time' },
  { value: 'udp', label: 'UDP' },
]

const defaultForm = (): MonitorFormData => ({
  Name: '',
  Type: 'http',
  IntervalSeconds: 60,
  TimeoutSeconds: 30,
  Retries: 3,
  URL: '',
  Method: 'GET',
  ExpectedStatus: 200,
})

function MonitorForm({
  initial,
  onSave,
  onCancel,
  saving,
}: {
  initial: MonitorFormData
  onSave: (data: MonitorFormData) => void
  onCancel: () => void
  saving: boolean
}) {
  const [form, setForm] = useState<MonitorFormData>(initial)
  const set = (k: keyof MonitorFormData, v: unknown) => setForm((f) => ({ ...f, [k]: v }))

  const needsURL = ['http', 'keyword', 'json_api', 'response_time'].includes(form.Type)
  const needsHost = ['port', 'ping', 'udp'].includes(form.Type)
  const needsDomain = ['ssl', 'domain_expiry'].includes(form.Type)
  const needsDNS = form.Type === 'dns'

  return (
    <form
      onSubmit={(e) => { e.preventDefault(); onSave(form) }}
      className="space-y-4 max-h-[70vh] overflow-y-auto pr-1"
    >
      {/* Name + Type */}
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="label">Name *</label>
          <input className="input" value={form.Name} onChange={(e) => set('Name', e.target.value)} required placeholder="My Website" />
        </div>
        <div>
          <label className="label">Type *</label>
          <select className="input" value={form.Type} onChange={(e) => set('Type', e.target.value as MonitorType)}>
            {MONITOR_TYPES.map((t) => <option key={t.value} value={t.value}>{t.label}</option>)}
          </select>
        </div>
      </div>

      {/* URL */}
      {needsURL && (
        <div>
          <label className="label">URL *</label>
          <input className="input" value={form.URL ?? ''} onChange={(e) => set('URL', e.target.value)} required placeholder="https://example.com" />
        </div>
      )}

      {/* HTTP extras */}
      {(form.Type === 'http' || form.Type === 'response_time') && (
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="label">Method</label>
            <select className="input" value={form.Method ?? 'GET'} onChange={(e) => set('Method', e.target.value)}>
              {['GET', 'POST', 'HEAD', 'PUT', 'DELETE'].map((m) => <option key={m}>{m}</option>)}
            </select>
          </div>
          <div>
            <label className="label">Expected Status</label>
            <input className="input" type="number" value={form.ExpectedStatus ?? 200} onChange={(e) => set('ExpectedStatus', +e.target.value)} />
          </div>
        </div>
      )}

      {/* Keyword */}
      {form.Type === 'keyword' && (
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="label">Keyword *</label>
            <input className="input" value={form.Keyword ?? ''} onChange={(e) => set('Keyword', e.target.value)} required />
          </div>
          <div>
            <label className="label">Mode</label>
            <select className="input" value={form.KeywordMode ?? 'contains'} onChange={(e) => set('KeywordMode', e.target.value)}>
              <option value="contains">Contains</option>
              <option value="not_contains">Not Contains</option>
            </select>
          </div>
        </div>
      )}

      {/* JSON API */}
      {form.Type === 'json_api' && (
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="label">JSON Path</label>
            <input className="input" value={form.JSONPath ?? ''} onChange={(e) => set('JSONPath', e.target.value)} placeholder="$.status" />
          </div>
          <div>
            <label className="label">Expected Value</label>
            <input className="input" value={form.JSONExpected ?? ''} onChange={(e) => set('JSONExpected', e.target.value)} placeholder="ok" />
          </div>
        </div>
      )}

      {/* Host + Port */}
      {needsHost && (
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="label">Host *</label>
            <input className="input" value={form.Host ?? ''} onChange={(e) => set('Host', e.target.value)} required placeholder="example.com" />
          </div>
          {form.Type !== 'ping' && (
            <div>
              <label className="label">Port *</label>
              <input className="input" type="number" value={form.Port ?? ''} onChange={(e) => set('Port', +e.target.value)} required placeholder="80" />
            </div>
          )}
        </div>
      )}

      {/* Domain */}
      {needsDomain && (
        <div>
          <label className="label">Domain *</label>
          <input className="input" value={form.Domain ?? ''} onChange={(e) => set('Domain', e.target.value)} required placeholder="example.com" />
        </div>
      )}

      {/* DNS */}
      {needsDNS && (
        <div className="grid grid-cols-3 gap-3">
          <div>
            <label className="label">DNS Host *</label>
            <input className="input" value={form.DNSHost ?? ''} onChange={(e) => set('DNSHost', e.target.value)} required placeholder="example.com" />
          </div>
          <div>
            <label className="label">Record Type</label>
            <select className="input" value={form.DNSRecordType ?? 'A'} onChange={(e) => set('DNSRecordType', e.target.value)}>
              {['A', 'AAAA', 'CNAME', 'MX', 'TXT', 'NS'].map((r) => <option key={r}>{r}</option>)}
            </select>
          </div>
          <div>
            <label className="label">Expected</label>
            <input className="input" value={form.DNSExpected ?? ''} onChange={(e) => set('DNSExpected', e.target.value)} placeholder="1.2.3.4" />
          </div>
        </div>
      )}

      {/* Response time threshold */}
      {form.Type === 'response_time' && (
        <div>
          <label className="label">Max Response Time (ms)</label>
          <input className="input" type="number" value={form.MaxResponseTimeMs ?? 1000} onChange={(e) => set('MaxResponseTimeMs', +e.target.value)} />
        </div>
      )}

      {/* Scheduling */}
      <div className="grid grid-cols-3 gap-3 pt-2 border-t border-gray-800">
        <div>
          <label className="label">Interval (s)</label>
          <input className="input" type="number" min={10} value={form.IntervalSeconds} onChange={(e) => set('IntervalSeconds', +e.target.value)} />
        </div>
        <div>
          <label className="label">Timeout (s)</label>
          <input className="input" type="number" min={1} value={form.TimeoutSeconds} onChange={(e) => set('TimeoutSeconds', +e.target.value)} />
        </div>
        <div>
          <label className="label">Retries</label>
          <input className="input" type="number" min={0} max={10} value={form.Retries} onChange={(e) => set('Retries', +e.target.value)} />
        </div>
      </div>

      {/* Actions */}
      <div className="flex justify-end gap-3 pt-2">
        <button type="button" className="btn-secondary" onClick={onCancel}>Cancel</button>
        <button type="submit" className="btn-primary" disabled={saving}>
          {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : 'Save'}
        </button>
      </div>
    </form>
  )
}

export default function Monitors() {
  const qc = useQueryClient()
  const [search, setSearch] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [editId, setEditId] = useState<number | null>(null)
  const [deleteId, setDeleteId] = useState<number | null>(null)

  const { data: monitors = [], isLoading } = useQuery({
    queryKey: ['monitors'],
    queryFn: monitorsApi.list,
    refetchInterval: 30_000,
  })

  const invalidate = () => qc.invalidateQueries({ queryKey: ['monitors'] })

  const createMut = useMutation({ mutationFn: monitorsApi.create, onSuccess: () => { invalidate(); setShowForm(false) } })
  const updateMut = useMutation({ mutationFn: ({ id, data }: { id: number; data: MonitorFormData }) => monitorsApi.update(id, data), onSuccess: () => { invalidate(); setEditId(null) } })
  const deleteMut = useMutation({ mutationFn: monitorsApi.delete, onSuccess: () => { invalidate(); setDeleteId(null) } })
  const pauseMut  = useMutation({ mutationFn: monitorsApi.pause,  onSuccess: invalidate })
  const resumeMut = useMutation({ mutationFn: monitorsApi.resume, onSuccess: invalidate })

  const filtered = monitors.filter((m) =>
    m.Name.toLowerCase().includes(search.toLowerCase()) ||
    (m.URL || m.Host || m.Domain || '').toLowerCase().includes(search.toLowerCase()),
  )

  const editMonitor = monitors.find((m) => m.ID === editId)

  return (
    <div className="p-6 max-w-4xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-bold text-gray-100">Monitors</h1>
          <p className="text-sm text-gray-500 mt-0.5">{monitors.length} monitor{monitors.length !== 1 ? 's' : ''}</p>
        </div>
        <button className="btn-primary" onClick={() => setShowForm(true)}>
          <Plus className="w-4 h-4" /> Add monitor
        </button>
      </div>

      {/* Search */}
      {monitors.length > 0 && (
        <div className="relative mb-4">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500" />
          <input
            className="input pl-9"
            placeholder="Search monitors…"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
      )}

      {/* List */}
      {isLoading ? (
        <div className="flex justify-center py-16">
          <div className="w-6 h-6 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
        </div>
      ) : filtered.length === 0 ? (
        <div className="card p-12 flex flex-col items-center text-center">
          <p className="text-gray-400 font-medium">{search ? 'No matches' : 'No monitors yet'}</p>
          {!search && <button className="btn-primary mt-4" onClick={() => setShowForm(true)}>Add your first monitor</button>}
        </div>
      ) : (
        <div className="space-y-2">
          {filtered.map((m) => (
            <MonitorCard
              key={m.ID}
              monitor={m}
              onPause={(id) => pauseMut.mutate(id)}
              onResume={(id) => resumeMut.mutate(id)}
              onDelete={(id) => setDeleteId(id)}
            />
          ))}
        </div>
      )}

      {/* Create modal */}
      {showForm && (
        <Modal title="Add Monitor" onClose={() => setShowForm(false)} size="lg">
          <MonitorForm
            initial={defaultForm()}
            onSave={(data) => createMut.mutate(data)}
            onCancel={() => setShowForm(false)}
            saving={createMut.isPending}
          />
        </Modal>
      )}

      {/* Edit modal */}
      {editId !== null && editMonitor && (
        <Modal title="Edit Monitor" onClose={() => setEditId(null)} size="lg">
          <MonitorForm
            initial={editMonitor as unknown as MonitorFormData}
            onSave={(data) => updateMut.mutate({ id: editId, data })}
            onCancel={() => setEditId(null)}
            saving={updateMut.isPending}
          />
        </Modal>
      )}

      {/* Delete confirm */}
      {deleteId !== null && (
        <ConfirmDialog
          title="Delete Monitor"
          message="This will permanently delete the monitor and all its history. Are you sure?"
          confirmLabel="Delete"
          danger
          onConfirm={() => deleteMut.mutate(deleteId)}
          onCancel={() => setDeleteId(null)}
        />
      )}
    </div>
  )
}
