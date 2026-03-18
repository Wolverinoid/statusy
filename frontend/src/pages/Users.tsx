import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { usersApi } from '@/api/client'
import Modal from '@/components/Modal'
import ConfirmDialog from '@/components/ConfirmDialog'
import { Plus, Trash2, Shield, User, Loader2 } from 'lucide-react'
import type { UserFormData } from '@/api/types'

const defaultForm = (): UserFormData => ({
  username: '',
  email: '',
  display_name: '',
  password: '',
  role: 'user',
  active: true,
})

function UserForm({
  initial, onSave, onCancel, saving, isEdit,
}: {
  initial: UserFormData
  onSave: (d: UserFormData) => void
  onCancel: () => void
  saving: boolean
  isEdit?: boolean
}) {
  const [form, setForm] = useState(initial)
  const set = (k: keyof UserFormData, v: unknown) => setForm((f) => ({ ...f, [k]: v }))

  return (
    <form onSubmit={(e) => { e.preventDefault(); onSave(form) }} className="space-y-4">
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="label">Username *</label>
          <input className="input" value={form.username} onChange={(e) => set('username', e.target.value)} required placeholder="john" />
        </div>
        <div>
          <label className="label">Display Name</label>
          <input className="input" value={form.display_name} onChange={(e) => set('display_name', e.target.value)} placeholder="John Doe" />
        </div>
      </div>
      <div>
        <label className="label">Email</label>
        <input className="input" type="email" value={form.email} onChange={(e) => set('email', e.target.value)} placeholder="user@example.com" />
      </div>
      <div>
        <label className="label">{isEdit ? 'New Password (leave blank to keep)' : 'Password *'}</label>
        <input
          className="input"
          type="password"
          value={form.password ?? ''}
          onChange={(e) => set('password', e.target.value)}
          required={!isEdit}
          placeholder="••••••••"
        />
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="label">Role</label>
          <select className="input" value={form.role} onChange={(e) => set('role', e.target.value as 'admin' | 'user')}>
            <option value="user">User</option>
            <option value="admin">Admin</option>
          </select>
        </div>
        <div className="flex items-end pb-1">
          <label className="flex items-center gap-2 text-sm text-gray-400 cursor-pointer">
            <input type="checkbox" checked={form.active} onChange={(e) => set('active', e.target.checked)} className="accent-indigo-500" />
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

export default function Users() {
  const qc = useQueryClient()
  const [showForm, setShowForm] = useState(false)
  const [editId, setEditId] = useState<number | null>(null)
  const [deleteId, setDeleteId] = useState<number | null>(null)

  const { data: users = [], isLoading } = useQuery({
    queryKey: ['users'],
    queryFn: usersApi.list,
  })

  const invalidate = () => qc.invalidateQueries({ queryKey: ['users'] })

  const createMut = useMutation({ mutationFn: usersApi.create, onSuccess: () => { invalidate(); setShowForm(false) } })
  const updateMut = useMutation({
    mutationFn: ({ id, data }: { id: number; data: UserFormData }) => usersApi.update(id, data),
    onSuccess: () => { invalidate(); setEditId(null) },
  })
  const deleteMut = useMutation({ mutationFn: usersApi.delete, onSuccess: () => { invalidate(); setDeleteId(null) } })

  const editUser = users.find((u) => u.id === editId)

  return (
    <div className="p-6 max-w-3xl mx-auto">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-xl font-bold text-gray-100">Users</h1>
          <p className="text-sm text-gray-500 mt-0.5">Manage access to Statusy</p>
        </div>
        <button className="btn-primary" onClick={() => setShowForm(true)}>
          <Plus className="w-4 h-4" /> Add user
        </button>
      </div>

      {isLoading ? (
        <div className="flex justify-center py-16">
          <div className="w-6 h-6 border-2 border-indigo-500 border-t-transparent rounded-full animate-spin" />
        </div>
      ) : (
        <div className="space-y-2">
          {users.map((u) => (
            <div key={u.id} className="card p-4 flex items-center gap-4 group hover:border-gray-700 transition-colors">
              <div className={`w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0 ${u.role === 'admin' ? 'bg-indigo-600/20 text-indigo-400' : 'bg-gray-800 text-gray-500'}`}>
                {u.role === 'admin' ? <Shield className="w-4 h-4" /> : <User className="w-4 h-4" />}
              </div>
              <div className="flex-1 min-w-0">
                <p className="font-medium text-gray-200">{u.display_name || u.username}</p>
                <p className="text-xs text-gray-500">@{u.username} · {u.role} · {u.active ? 'Active' : 'Disabled'}</p>
              </div>
              <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                <button
                  onClick={() => setDeleteId(u.id)}
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
        <Modal title="Add User" onClose={() => setShowForm(false)}>
          <UserForm initial={defaultForm()} onSave={(d) => createMut.mutate(d)} onCancel={() => setShowForm(false)} saving={createMut.isPending} />
        </Modal>
      )}

      {editId !== null && editUser && (
        <Modal title="Edit User" onClose={() => setEditId(null)}>
          <UserForm
            initial={editUser as unknown as UserFormData}
            onSave={(d) => updateMut.mutate({ id: editId, data: d })}
            onCancel={() => setEditId(null)}
            saving={updateMut.isPending}
            isEdit
          />
        </Modal>
      )}

      {deleteId !== null && (
        <ConfirmDialog
          title="Delete User"
          message="This user will be permanently deleted. Are you sure?"
          confirmLabel="Delete"
          danger
          onConfirm={() => deleteMut.mutate(deleteId)}
          onCancel={() => setDeleteId(null)}
        />
      )}
    </div>
  )
}
