import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router-dom'
import { Radio, Loader2, Database, User, ChevronRight } from 'lucide-react'

type DBType = 'sqlite' | 'mysql' | 'postgres'

interface SetupForm {
  db_type: DBType
  db_path: string
  db_host: string
  db_port: string
  db_user: string
  db_pass: string
  db_name: string
  ssl_mode: string
  admin_username: string
  admin_email: string
  admin_password: string
  admin_password_confirm: string
}

const defaults: SetupForm = {
  db_type: 'sqlite',
  db_path: 'data/statusy.db',
  db_host: 'localhost',
  db_port: '5432',
  db_user: 'statusy',
  db_pass: '',
  db_name: 'statusy',
  ssl_mode: 'disable',
  admin_username: 'admin',
  admin_email: 'admin@example.com',
  admin_password: '',
  admin_password_confirm: '',
}

export default function Setup() {
  const navigate = useNavigate()
  const [form, setForm] = useState<SetupForm>(defaults)
  const [step, setStep] = useState<1 | 2>(1)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const set = (k: keyof SetupForm, v: string) => setForm((f) => ({ ...f, [k]: v }))

  const handleNext = (e: FormEvent) => {
    e.preventDefault()
    setStep(2)
  }

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setError('')

    if (form.admin_password.length < 8) {
      setError('Password must be at least 8 characters')
      return
    }
    if (form.admin_password !== form.admin_password_confirm) {
      setError('Passwords do not match')
      return
    }

    setLoading(true)
    try {
      const res = await fetch('/api/setup/complete', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          db_type: form.db_type,
          db_path: form.db_path,
          db_host: form.db_host,
          db_port: form.db_port,
          db_user: form.db_user,
          db_pass: form.db_pass,
          db_name: form.db_name,
          ssl_mode: form.ssl_mode,
          admin_username: form.admin_username,
          admin_email: form.admin_email,
          admin_password: form.admin_password,
        }),
      })
      const data = await res.json()
      if (!res.ok) {
        setError(data.error || 'Setup failed')
        return
      }
      // Server will restart — wait a moment then redirect to login
      await new Promise((r) => setTimeout(r, 2000))
      navigate('/login')
    } catch (err) {
      setError('Network error — server may be restarting, please refresh in a moment')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-screen bg-gray-950 flex items-center justify-center p-4">
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-1/3 left-1/2 -translate-x-1/2 -translate-y-1/2 w-96 h-96 bg-indigo-600/10 rounded-full blur-3xl" />
      </div>

      <div className="relative w-full max-w-lg">
        {/* Logo */}
        <div className="flex flex-col items-center mb-8">
          <div className="w-12 h-12 bg-indigo-600 rounded-2xl flex items-center justify-center mb-4 shadow-lg shadow-indigo-600/30">
            <Radio className="w-6 h-6 text-white" />
          </div>
          <h1 className="text-2xl font-bold text-white">Statusy Setup</h1>
          <p className="text-sm text-gray-500 mt-1">First-run configuration wizard</p>
        </div>

        {/* Steps indicator */}
        <div className="flex items-center justify-center gap-3 mb-6">
          <div className={`flex items-center gap-2 text-sm font-medium ${step === 1 ? 'text-indigo-400' : 'text-gray-500'}`}>
            <Database className="w-4 h-4" /> Database
          </div>
          <ChevronRight className="w-4 h-4 text-gray-700" />
          <div className={`flex items-center gap-2 text-sm font-medium ${step === 2 ? 'text-indigo-400' : 'text-gray-500'}`}>
            <User className="w-4 h-4" /> Admin Account
          </div>
        </div>

        <div className="card p-6">
          {/* Step 1: Database */}
          {step === 1 && (
            <form onSubmit={handleNext} className="space-y-4">
              <div>
                <label className="label">Database Type</label>
                <select className="input" value={form.db_type} onChange={(e) => set('db_type', e.target.value as DBType)}>
                  <option value="sqlite">SQLite (single file, zero config)</option>
                  <option value="mysql">MySQL / MariaDB</option>
                  <option value="postgres">PostgreSQL</option>
                </select>
              </div>

              {form.db_type === 'sqlite' && (
                <div>
                  <label className="label">Database File Path</label>
                  <input className="input" value={form.db_path} onChange={(e) => set('db_path', e.target.value)} placeholder="data/statusy.db" />
                </div>
              )}

              {(form.db_type === 'mysql' || form.db_type === 'postgres') && (
                <>
                  <div className="grid grid-cols-3 gap-3">
                    <div className="col-span-2">
                      <label className="label">Host</label>
                      <input className="input" value={form.db_host} onChange={(e) => set('db_host', e.target.value)} placeholder="localhost" required />
                    </div>
                    <div>
                      <label className="label">Port</label>
                      <input className="input" value={form.db_port}
                        onChange={(e) => set('db_port', e.target.value)}
                        placeholder={form.db_type === 'mysql' ? '3306' : '5432'}
                        required
                      />
                    </div>
                  </div>
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="label">Username</label>
                      <input className="input" value={form.db_user} onChange={(e) => set('db_user', e.target.value)} placeholder="statusy" required />
                    </div>
                    <div>
                      <label className="label">Password</label>
                      <input className="input" type="password" value={form.db_pass} onChange={(e) => set('db_pass', e.target.value)} placeholder="••••••••" />
                    </div>
                  </div>
                  <div>
                    <label className="label">Database Name</label>
                    <input className="input" value={form.db_name} onChange={(e) => set('db_name', e.target.value)} placeholder="statusy" required />
                  </div>
                  {form.db_type === 'postgres' && (
                    <div>
                      <label className="label">SSL Mode</label>
                      <select className="input" value={form.ssl_mode} onChange={(e) => set('ssl_mode', e.target.value)}>
                        <option value="disable">disable</option>
                        <option value="require">require</option>
                        <option value="verify-full">verify-full</option>
                      </select>
                    </div>
                  )}
                </>
              )}

              <button type="submit" className="btn-primary w-full justify-center py-2.5 mt-2">
                Next: Admin Account <ChevronRight className="w-4 h-4" />
              </button>
            </form>
          )}

          {/* Step 2: Admin account */}
          {step === 2 && (
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="label">Username</label>
                  <input className="input" value={form.admin_username} onChange={(e) => set('admin_username', e.target.value)} placeholder="admin" required />
                </div>
                <div>
                  <label className="label">Email</label>
                  <input className="input" type="email" value={form.admin_email} onChange={(e) => set('admin_email', e.target.value)} placeholder="admin@example.com" required />
                </div>
              </div>
              <div>
                <label className="label">Password (min 8 characters)</label>
                <input className="input" type="password" value={form.admin_password} onChange={(e) => set('admin_password', e.target.value)} placeholder="••••••••" required />
              </div>
              <div>
                <label className="label">Confirm Password</label>
                <input className="input" type="password" value={form.admin_password_confirm} onChange={(e) => set('admin_password_confirm', e.target.value)} placeholder="••••••••" required />
              </div>

              {error && (
                <p className="text-sm text-red-400 bg-red-500/10 border border-red-500/20 rounded-lg px-3 py-2">
                  {error}
                </p>
              )}

              <div className="flex gap-3 pt-1">
                <button type="button" className="btn-secondary flex-1 justify-center" onClick={() => setStep(1)}>
                  Back
                </button>
                <button type="submit" disabled={loading} className="btn-primary flex-1 justify-center py-2.5">
                  {loading ? <Loader2 className="w-4 h-4 animate-spin" /> : 'Complete Setup'}
                </button>
              </div>
            </form>
          )}
        </div>
      </div>
    </div>
  )
}
