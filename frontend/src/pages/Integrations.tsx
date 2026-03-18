import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { integrationsApi } from '@/api/client'
import { Activity, Eye, EyeOff, Loader2, CheckCircle2, ExternalLink } from 'lucide-react'
import type { PrometheusIntegration } from '@/api/types'

const defaultForm = (): PrometheusIntegration => ({
  enabled: false,
  url: '',
  basic_auth_user: '',
  basic_auth_pass: '',
})

export default function Integrations() {
  const qc = useQueryClient()
  const [form, setForm] = useState<PrometheusIntegration>(defaultForm())
  const [showPass, setShowPass] = useState(false)
  const [saved, setSaved] = useState(false)

  const { data, isLoading } = useQuery({
    queryKey: ['integrations', 'prometheus'],
    queryFn: integrationsApi.getPrometheus,
  })

  useEffect(() => {
    if (data) setForm(data)
  }, [data])

  const set = <K extends keyof PrometheusIntegration>(k: K, v: PrometheusIntegration[K]) =>
    setForm((f) => ({ ...f, [k]: v }))

  const saveMut = useMutation({
    mutationFn: integrationsApi.savePrometheus,
    onSuccess: (updated) => {
      qc.setQueryData(['integrations', 'prometheus'], updated)
      setSaved(true)
      setTimeout(() => setSaved(false), 2500)
    },
  })

  const metricsUrl = `${window.location.origin}/metrics`

  return (
    <div className="p-6 max-w-2xl mx-auto">
      <div className="mb-6">
        <h1 className="text-xl font-bold text-gray-100">Integrations</h1>
        <p className="text-sm text-gray-500 mt-0.5">Connect Statusy with external services</p>
      </div>

      {/* Prometheus card */}
      <div className="card p-6">
        <div className="flex items-start gap-4 mb-5">
          <div className="w-10 h-10 rounded-lg bg-orange-500/10 flex items-center justify-center flex-shrink-0">
            <Activity className="w-5 h-5 text-orange-400" />
          </div>
          <div className="flex-1">
            <div className="flex items-center gap-2">
              <h2 className="font-semibold text-gray-100">Prometheus</h2>
              {form.enabled && (
                <span className="text-xs text-emerald-400 bg-emerald-400/10 px-2 py-0.5 rounded-full">Enabled</span>
              )}
            </div>
            <p className="text-sm text-gray-400 mt-0.5">
              Statusy exposes a <code className="text-indigo-400 text-xs bg-gray-800 px-1 py-0.5 rounded">/metrics</code> endpoint
              in Prometheus format. Configure your Prometheus server to scrape it.
            </p>
          </div>
        </div>

        {isLoading ? (
          <div className="flex justify-center py-8">
            <Loader2 className="w-5 h-5 animate-spin text-gray-500" />
          </div>
        ) : (
          <form
            onSubmit={(e) => { e.preventDefault(); saveMut.mutate(form) }}
            className="space-y-4"
          >
            {/* Enable toggle */}
            <label className="flex items-center gap-3 cursor-pointer">
              <div className="relative">
                <input
                  type="checkbox"
                  className="sr-only"
                  checked={form.enabled}
                  onChange={(e) => set('enabled', e.target.checked)}
                />
                <div className={`w-10 h-6 rounded-full transition-colors ${form.enabled ? 'bg-indigo-600' : 'bg-gray-700'}`} />
                <div className={`absolute top-1 left-1 w-4 h-4 bg-white rounded-full transition-transform ${form.enabled ? 'translate-x-4' : ''}`} />
              </div>
              <span className="text-sm text-gray-300">Enable Prometheus integration</span>
            </label>

            {/* Metrics endpoint info */}
            <div className="bg-gray-800/50 rounded-lg p-3 flex items-center gap-3">
              <div className="flex-1 min-w-0">
                <p className="text-xs text-gray-500 mb-0.5">Metrics endpoint (scrape this URL)</p>
                <p className="text-sm text-indigo-400 font-mono truncate">{metricsUrl}</p>
              </div>
              <a
                href="/metrics"
                target="_blank"
                rel="noopener noreferrer"
                className="p-1.5 rounded-lg text-gray-500 hover:text-gray-300 hover:bg-gray-700 transition-colors flex-shrink-0"
                title="Open metrics endpoint"
              >
                <ExternalLink className="w-4 h-4" />
              </a>
            </div>

            {/* Prometheus server URL */}
            <div>
              <label className="label">Prometheus Server URL</label>
              <input
                className="input"
                type="url"
                value={form.url}
                onChange={(e) => set('url', e.target.value)}
                placeholder="http://prometheus:9090"
              />
              <p className="text-xs text-gray-600 mt-1">
                URL of your Prometheus server (used for reference / future push integrations)
              </p>
            </div>

            {/* Basic auth */}
            <div className="border border-gray-800 rounded-lg p-4 space-y-3">
              <p className="text-xs font-medium text-gray-400 uppercase tracking-wide">
                HTTP Basic Auth <span className="text-gray-600 normal-case font-normal">(optional — protects /metrics endpoint)</span>
              </p>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="label">Username</label>
                  <input
                    className="input"
                    value={form.basic_auth_user}
                    onChange={(e) => set('basic_auth_user', e.target.value)}
                    placeholder="prometheus"
                    autoComplete="off"
                  />
                </div>
                <div>
                  <label className="label">Password</label>
                  <div className="relative">
                    <input
                      className="input pr-9"
                      type={showPass ? 'text' : 'password'}
                      value={form.basic_auth_pass}
                      onChange={(e) => set('basic_auth_pass', e.target.value)}
                      placeholder="••••••••"
                      autoComplete="new-password"
                    />
                    <button
                      type="button"
                      onClick={() => setShowPass((v) => !v)}
                      className="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-300"
                    >
                      {showPass ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                    </button>
                  </div>
                </div>
              </div>
            </div>

            <div className="flex items-center justify-end gap-3 pt-1">
              {saved && (
                <span className="flex items-center gap-1.5 text-sm text-emerald-400">
                  <CheckCircle2 className="w-4 h-4" /> Saved
                </span>
              )}
              <button type="submit" className="btn-primary" disabled={saveMut.isPending}>
                {saveMut.isPending ? <Loader2 className="w-4 h-4 animate-spin" /> : 'Save'}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  )
}
