import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { Plus, Trash2, Edit2, ExternalLink, Globe, Lock, Monitor, Users, Loader2 } from 'lucide-react'
import { statusPagesApi, monitorsApi, usersApi } from '@/api/client'
import Modal from '@/components/Modal'
import ConfirmDialog from '@/components/ConfirmDialog'
import type { StatusPage, StatusPageFormData } from '@/api/types'

// ── Form ──────────────────────────────────────────────────────────────────────

function slugify(s: string) {
  return s.toLowerCase().replace(/\s+/g, '-').replace(/[^a-z0-9-]/g, '')
}

function StatusPageForm({
  initial,
  onSave,
  onCancel,
  saving,
}: {
  initial: StatusPageFormData
  onSave: (data: StatusPageFormData) => void
  onCancel: () => void
  saving: boolean
}) {
  const [form, setForm] = useState<StatusPageFormData>(initial)
  const set = <K extends keyof StatusPageFormData>(k: K, v: StatusPageFormData[K]) =>
    setForm((f) => ({ ...f, [k]: v }))

  return (
    <form onSubmit={(e) => { e.preventDefault(); onSave(form) }} className="space-y-4">
      <div>
        <label className="label">Name *</label>
        <input
          className="input"
          value={form.name}
          onChange={(e) => {
            set('name', e.target.value)
            if (!initial.slug) set('slug', slugify(e.target.value))
          }}
          required
          placeholder="My Status Page"
        />
      </div>
      <div>
        <label className="label">Slug * <span className="text-gray-600 font-normal">(URL identifier)</span></label>
        <div className="flex items-center gap-2">
          <span className="text-sm text-gray-500 flex-shrink-0">/status/</span>
          <input
            className="input"
            value={form.slug}
            onChange={(e) => set('slug', slugify(e.target.value))}
            required
            placeholder="my-status-page"
          />
        </div>
      </div>
      <div>
        <label className="label">Description</label>
        <textarea
          className="input resize-none"
          rows={2}
          value={form.description}
          onChange={(e) => set('description', e.target.value)}
          placeholder="Optional description shown on the status page"
        />
      </div>
      <label className="flex items-center gap-3 cursor-pointer">
        <div className="relative">
          <input
            type="checkbox"
            className="sr-only"
            checked={form.public}
            onChange={(e) => set('public', e.target.checked)}
          />
          <div className={`w-10 h-6 rounded-full transition-colors ${form.public ? 'bg-indigo-600' : 'bg-gray-700'}`} />
          <div className={`absolute top-1 left-1 w-4 h-4 bg-white rounded-full transition-transform ${form.public ? 'translate-x-4' : ''}`} />
        </div>
        <span className="text-sm text-gray-300">Public page <span className="text-gray-500">(accessible without login)</span></span>
      </label>
      <div className="flex justify-end gap-3 pt-2">
        <button type="button" className="btn-secondary" onClick={onCancel}>Cancel</button>
        <button type="submit" className="btn-primary" disabled={saving}>
          {saving ? <Loader2 className="w-4 h-4 animate-spin" /> : 'Save'}
        </button>
      </div>
    </form>
  )
}

// ── Monitor selector ──────────────────────────────────────────────────────────

function MonitorSelector({
  pageId,
  currentIds,
  onClose,
}: {
  pageId: number
  currentIds: number[]
  onClose: () => void
}) {
  const qc = useQueryClient()
  const [selected, setSelected] = useState<Set<number>>(new Set(currentIds))

  const { data: monitors = [] } = useQuery({ queryKey: ['monitors'], queryFn: monitorsApi.list })

  const saveMut = useMutation({
    mutationFn: () => statusPagesApi.setMonitors(pageId, Array.from(selected)),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['status-pages'] }); onClose() },
  })

  const toggle = (id: number) =>
    setSelected((s) => { const n = new Set(s); n.has(id) ? n.delete(id) : n.add(id); return n })

  return (
    <div className="space-y-3">
      <p className="text-sm text-gray-400">Select monitors to show on this status page:</p>
      <div className="max-h-64 overflow-y-auto space-y-1">
        {monitors.length === 0 && <p className="text-sm text-gray-500 py-4 text-center">No monitors yet</p>}
        {monitors.map((m) => (
          <label key={m.ID} className="flex items-center gap-3 p-2 rounded-lg hover:bg-gray-800 cursor-pointer">
            <input
              type="checkbox"
              className="w-4 h-4 accent-indigo-500"
              checked={selected.has(m.ID)}
              onChange={() => toggle(m.ID)}
            />
            <span className="text-sm text-gray-200 flex-1">{m.Name}</span>
            <span className={`w-2 h-2 rounded-full flex-shrink-0 ${
              m.Status === 'UP' ? 'bg-emerald-500' :
              m.Status === 'DOWN' ? 'bg-red-500' : 'bg-gray-500'
            }`} />
          </label>
        ))}
      </div>
      <div className="flex justify-between items-center pt-2">
        <span className="text-xs text-gray-500">{selected.size} selected</span>
        <div className="flex gap-3">
          <button className="btn-secondary" onClick={onClose}>Cancel</button>
          <button className="btn-primary" onClick={() => saveMut.mutate()} disabled={saveMut.isPending}>
            {saveMut.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : 'Save'}
          </button>
        </div>
      </div>
    </div>
  )
}

// ── User selector ─────────────────────────────────────────────────────────────

function UserSelector({
  pageId,
  currentIds,
  onClose,
}: {
  pageId: number
  currentIds: number[]
  onClose: () => void
}) {
  const qc = useQueryClient()
  const [selected, setSelected] = useState<Set<number>>(new Set(currentIds))

  const { data: users = [] } = useQuery({ queryKey: ['users'], queryFn: usersApi.list })

  const saveMut = useMutation({
    mutationFn: () => statusPagesApi.setUsers(pageId, Array.from(selected)),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ['status-pages'] }); onClose() },
  })

  const toggle = (id: number) =>
    setSelected((s) => { const n = new Set(s); n.has(id) ? n.delete(id) : n.add(id); return n })

  return (
    <div className="space-y-3">
      <p className="text-sm text-gray-400">Select users who can view this private status page:</p>
      <div className="max-h-64 overflow-y-auto space-y-1">
        {users.length === 0 && <p className="text-sm text-gray-500 py-4 text-center">No users yet</p>}
        {users.map((u) => (
          <label key={u.id} className="flex items-center gap-3 p-2 rounded-lg hover:bg-gray-800 cursor-pointer">
            <input
              type="checkbox"
              className="w-4 h-4 accent-indigo-500"
              checked={selected.has(u.id)}
              onChange={() => toggle(u.id)}
            />
            <div className="flex-1 min-w-0">
              <p className="text-sm text-gray-200">{u.display_name || u.username}</p>
              <p className="text-xs text-gray-500">{u.email}</p>
            </div>
            <span className="text-xs text-gray-600 bg-gray-800 px-1.5 py-0.5 rounded">{u.role}</span>
          </label>
        ))}
      </div>
      <div className="flex justify-between items-center pt-2">
        <span className="text-xs text-gray-500">{selected.size} selected</span>
        <div className="flex gap-3">
          <button className="btn-secondary" onClick={onClose}>Cancel</button>
          <button className="btn-primary" onClick={() => saveMut.mutate()} disabled={saveMut.isPending}>
            {saveMut.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : 'Save'}
          </button>
        </div>
      </div>
    </div>
  )
}

// ── Status Page Card ──────────────────────────────────────────────────────────

function StatusPageCard({
  page,
  onEdit,
  onDelete,
}: {
  page: StatusPage
  onEdit: () => void
  onDelete: () => void
}) {
  const [showMonitors, setShowMonitors] = useState(false)
  const [showUsers, setShowUsers] = useState(false)

  const overallStatus = page.Monitors?.length
    ? page.Monitors.every((m) => m.Status === 'UP') ? 'all-up'
    : page.Monitors.some((m) => m.Status === 'DOWN') ? 'some-down'
    : 'partial'
    : 'no-monitors'

  return (
    <div className="card p-5 hover:border-gray-700 transition-colors">
      <div className="flex items-start justify-between gap-4">
        <div className="flex-1 min-w-0">
          {/* Title row */}
          <div className="flex items-center gap-2 flex-wrap">
            <h3 className="font-semibold text-gray-100">{page.Name}</h3>
            {page.Public
              ? <span className="flex items-center gap-1 text-xs text-emerald-400 bg-emerald-400/10 px-2 py-0.5 rounded-full"><Globe className="w-3 h-3" />Public</span>
              : <span className="flex items-center gap-1 text-xs text-gray-400 bg-gray-800 px-2 py-0.5 rounded-full"><Lock className="w-3 h-3" />Private</span>
            }
            {overallStatus === 'all-up' && <span className="text-xs text-emerald-400 bg-emerald-400/10 px-2 py-0.5 rounded-full">All systems operational</span>}
            {overallStatus === 'some-down' && <span className="text-xs text-red-400 bg-red-400/10 px-2 py-0.5 rounded-full">Incident detected</span>}
          </div>

          {/* Slug */}
          <p className="text-xs text-gray-500 mt-0.5">
            /status/<span className="text-indigo-400">{page.Slug}</span>
          </p>

          {/* Description */}
          {page.Description && <p className="text-sm text-gray-400 mt-1">{page.Description}</p>}

          {/* Stats */}
          <div className="flex items-center gap-4 mt-3">
            <button
              onClick={() => setShowMonitors(true)}
              className="flex items-center gap-1.5 text-xs text-gray-400 hover:text-indigo-400 transition-colors"
            >
              <Monitor className="w-3.5 h-3.5" />
              {page.Monitors?.length ?? 0} monitor{(page.Monitors?.length ?? 0) !== 1 ? 's' : ''}
            </button>
            <button
              onClick={() => setShowUsers(true)}
              className="flex items-center gap-1.5 text-xs text-gray-400 hover:text-indigo-400 transition-colors"
            >
              <Users className="w-3.5 h-3.5" />
              {page.Users?.length ?? 0} user{(page.Users?.length ?? 0) !== 1 ? 's' : ''} with access
            </button>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-1 flex-shrink-0">
          <Link
            to={`/status/${page.Slug}`}
            target="_blank"
            className="p-1.5 rounded-lg text-gray-500 hover:text-gray-300 hover:bg-gray-800 transition-colors"
            title="View public page"
          >
            <ExternalLink className="w-4 h-4" />
          </Link>
          <button
            onClick={onEdit}
            className="p-1.5 rounded-lg text-gray-500 hover:text-indigo-400 hover:bg-gray-800 transition-colors"
            title="Edit"
          >
            <Edit2 className="w-4 h-4" />
          </button>
          <button
            onClick={onDelete}
            className="p-1.5 rounded-lg text-gray-500 hover:text-red-400 hover:bg-gray-800 transition-colors"
            title="Delete"
          >
            <Trash2 className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* Monitor selector modal */}
      {showMonitors && (
        <Modal title="Assign Monitors" onClose={() => setShowMonitors(false)}>
          <MonitorSelector
            pageId={page.ID}
            currentIds={page.Monitors?.map((m) => m.ID) ?? []}
            onClose={() => setShowMonitors(false)}
          />
        </Modal>
      )}

      {/* User selector modal */}
      {showUsers && (
        <Modal title="Manage Access" onClose={() => setShowUsers(false)}>
          <UserSelector
            pageId={page.ID}
            currentIds={page.Users?.map((u) => u.id) ?? []}
            onClose={() => setShowUsers(false)}
          />
        </Modal>
      )}
    </div>
  )
}

// ── Main page ─────────────────────────────────────────────────────────────────

const defaultForm = (): StatusPageFormData => ({
  name: '',
  slug: '',
  description: '',
  public: true,
})

export default function StatusPages() {
  const qc = useQueryClient()
  const [showCreate, setShowCreate] = useState(false)
  const [editPage, setEditPage] = useState<StatusPage | null>(null)
  const [deleteId, setDeleteId] = useState<number | null>(null)

  const { data: pages = [], isLoading } = useQuery({
    queryKey: ['status-pages'],
    queryFn: statusPagesApi.list,
  })

  const invalidate = () => qc.invalidateQueries({ queryKey: ['status-pages'] })

  const createMut = useMutation({
    mutationFn: statusPagesApi.create,
    onSuccess: () => { invalidate(); setShowCreate(false) },
  })

  const updateMut = useMutation({
    mutationFn: ({ id, data }: { id: number; data: StatusPageFormData }) =>
      statusPagesApi.update(id, data),
    onSuccess: () => { invalidate(); setEditPage(null) },
  })

  const deleteMut = useMutation({
    mutationFn: statusPagesApi.delete,
    onSuccess: () => { invalidate(); setDeleteId(null) },
  })

  return (
    <div className="p-6 max-w-4xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-bold text-gray-100">Status Pages</h1>
          <p className="text-sm text-gray-500 mt-0.5">{pages.length} page{pages.length !== 1 ? 's' : ''}</p>
        </div>
        <button className="btn-primary" onClick={() => setShowCreate(true)}>
          <Plus className="w-4 h-4" /> New page
        </button>
      </div>

      {/* List */}
      {isLoading ? (
        <div className="flex justify-center py-16">
          <div className="w-6 h-6 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
        </div>
      ) : pages.length === 0 ? (
        <div className="card p-12 flex flex-col items-center text-center">
          <Globe className="w-10 h-10 text-gray-700 mb-3" />
          <p className="text-gray-400 font-medium">No status pages yet</p>
          <p className="text-sm text-gray-600 mt-1">Create a page to share monitor statuses publicly or with specific users</p>
          <button className="btn-primary mt-4" onClick={() => setShowCreate(true)}>Create your first page</button>
        </div>
      ) : (
        <div className="space-y-3">
          {pages.map((page) => (
            <StatusPageCard
              key={page.ID}
              page={page}
              onEdit={() => setEditPage(page)}
              onDelete={() => setDeleteId(page.ID)}
            />
          ))}
        </div>
      )}

      {/* Create modal */}
      {showCreate && (
        <Modal title="New Status Page" onClose={() => setShowCreate(false)}>
          <StatusPageForm
            initial={defaultForm()}
            onSave={(data) => createMut.mutate(data)}
            onCancel={() => setShowCreate(false)}
            saving={createMut.isPending}
          />
        </Modal>
      )}

      {/* Edit modal */}
      {editPage && (
        <Modal title="Edit Status Page" onClose={() => setEditPage(null)}>
          <StatusPageForm
            initial={{ name: editPage.Name, slug: editPage.Slug, description: editPage.Description, public: editPage.Public }}
            onSave={(data) => updateMut.mutate({ id: editPage.ID, data })}
            onCancel={() => setEditPage(null)}
            saving={updateMut.isPending}
          />
        </Modal>
      )}

      {/* Delete confirm */}
      {deleteId !== null && (
        <ConfirmDialog
          title="Delete Status Page"
          message="This will permanently delete the status page. Monitors won't be affected. Are you sure?"
          confirmLabel="Delete"
          danger
          onConfirm={() => deleteMut.mutate(deleteId)}
          onCancel={() => setDeleteId(null)}
        />
      )}
    </div>
  )
}
