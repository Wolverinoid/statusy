import axios from 'axios'
import type {
  LoginRequest, TokenPair, Me,
  Monitor, MonitorFormData, CheckResult,
  Notification, NotificationFormData,
  User, UserFormData,
  StatusPage, StatusPageFormData,
  PrometheusIntegration,
} from './types'

// ── Axios instance ────────────────────────────────────────────────────────────

const api = axios.create({
  baseURL: '/api',
  headers: { 'Content-Type': 'application/json' },
})

// Attach JWT from localStorage on every request
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('access_token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

// On 401 → clear tokens and redirect to login
api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401) {
      localStorage.removeItem('access_token')
      localStorage.removeItem('refresh_token')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  },
)

// ── Auth ──────────────────────────────────────────────────────────────────────

export const authApi = {
  login: (data: LoginRequest) =>
    api.post<TokenPair>('/auth/login', data).then((r) => r.data),

  logout: () =>
    api.post('/auth/logout').then((r) => r.data),

  me: () =>
    api.get<Me>('/auth/me').then((r) => r.data),

  refresh: (refreshToken: string) =>
    api.post<TokenPair>('/auth/refresh', { refresh_token: refreshToken }).then((r) => r.data),
}

// ── Monitors ──────────────────────────────────────────────────────────────────

export const monitorsApi = {
  list: () =>
    api.get<Monitor[]>('/monitors').then((r) => r.data),

  get: (id: number) =>
    api.get<Monitor>(`/monitors/${id}`).then((r) => r.data),

  create: (data: MonitorFormData) =>
    api.post<Monitor>('/monitors', data).then((r) => r.data),

  update: (id: number, data: MonitorFormData) =>
    api.put<Monitor>(`/monitors/${id}`, data).then((r) => r.data),

  delete: (id: number) =>
    api.delete(`/monitors/${id}`).then((r) => r.data),

  pause: (id: number) =>
    api.post(`/monitors/${id}/pause`).then((r) => r.data),

  resume: (id: number) =>
    api.post(`/monitors/${id}/resume`).then((r) => r.data),

  history: (id: number) =>
    api.get<CheckResult[]>(`/monitors/${id}/history`).then((r) => r.data),
}

// ── Notifications ─────────────────────────────────────────────────────────────

export const notificationsApi = {
  list: () =>
    api.get<Notification[]>('/notifications').then((r) => r.data),

  create: (data: NotificationFormData) =>
    api.post<Notification>('/notifications', data).then((r) => r.data),

  update: (id: number, data: NotificationFormData) =>
    api.put<Notification>(`/notifications/${id}`, data).then((r) => r.data),

  delete: (id: number) =>
    api.delete(`/notifications/${id}`).then((r) => r.data),

  test: (id: number) =>
    api.post(`/notifications/${id}/test`).then((r) => r.data),
}

// ── Users ─────────────────────────────────────────────────────────────────────

export const usersApi = {
  list: () =>
    api.get<User[]>('/users').then((r) => r.data),

  create: (data: UserFormData) =>
    api.post<User>('/users', data).then((r) => r.data),

  update: (id: number, data: Partial<UserFormData>) =>
    api.put<User>(`/users/${id}`, data).then((r) => r.data),

  delete: (id: number) =>
    api.delete(`/users/${id}`).then((r) => r.data),
}

// ── Status Pages ──────────────────────────────────────────────────────────────

export const statusPagesApi = {
  list: () =>
    api.get<StatusPage[]>('/status-pages').then((r) => r.data),

  get: (id: number) =>
    api.get<StatusPage>(`/status-pages/${id}`).then((r) => r.data),

  create: (data: StatusPageFormData) =>
    api.post<StatusPage>('/status-pages', data).then((r) => r.data),

  update: (id: number, data: StatusPageFormData) =>
    api.put<StatusPage>(`/status-pages/${id}`, data).then((r) => r.data),

  delete: (id: number) =>
    api.delete(`/status-pages/${id}`).then((r) => r.data),

  setMonitors: (id: number, monitorIds: number[]) =>
    api.put(`/status-pages/${id}/monitors`, { monitor_ids: monitorIds }).then((r) => r.data),

  setUsers: (id: number, userIds: number[]) =>
    api.put(`/status-pages/${id}/users`, { user_ids: userIds }).then((r) => r.data),

  getPublic: (slug: string) =>
    api.get<StatusPage>(`/status/${slug}`).then((r) => r.data),
}

// ── Integrations ─────────────────────────────────────────────────────────────

export const integrationsApi = {
  getPrometheus: () =>
    api.get<PrometheusIntegration>('/admin/integrations/prometheus').then((r) => r.data),

  savePrometheus: (data: PrometheusIntegration) =>
    api.put<PrometheusIntegration>('/admin/integrations/prometheus', data).then((r) => r.data),
}

export default api
