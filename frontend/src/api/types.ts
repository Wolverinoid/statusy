// ── Auth ──────────────────────────────────────────────────────────────────────

export interface LoginRequest {
  username: string
  password: string
}

export interface TokenPair {
  access_token: string
  refresh_token: string
}

export interface Me {
  id: number
  username: string
  email: string
  display_name: string
  role: 'admin' | 'user'
  auth_source: 'local' | 'ldap'
}

// ── Monitor ───────────────────────────────────────────────────────────────────

export type MonitorType =
  | 'http'
  | 'port'
  | 'ping'
  | 'keyword'
  | 'json_api'
  | 'udp'
  | 'response_time'
  | 'dns'
  | 'ssl'
  | 'domain_expiry'

export type MonitorStatus = 'UP' | 'DOWN' | 'PENDING' | 'MAINTENANCE'

export interface Monitor {
  ID: number
  CreatedAt: string
  UpdatedAt: string
  Name: string
  Type: MonitorType
  Status: MonitorStatus
  Active: boolean
  IntervalSeconds: number
  TimeoutSeconds: number
  Retries: number
  URL: string
  Method: string
  Headers: string
  Body: string
  ExpectedStatus: number
  Keyword: string
  KeywordMode: string
  JSONPath: string
  JSONExpected: string
  Host: string
  Port: number
  DNSHost: string
  DNSRecordType: string
  DNSExpected: string
  Domain: string
  WarnDaysExpiry: number
  MaxResponseTimeMs: number
  LastMessage: string
  Notifications: Notification[]
}

export interface MonitorFormData {
  Name: string
  Type: MonitorType
  IntervalSeconds: number
  TimeoutSeconds: number
  Retries: number
  URL?: string
  Method?: string
  ExpectedStatus?: number
  Keyword?: string
  KeywordMode?: string
  JSONPath?: string
  JSONExpected?: string
  Host?: string
  Port?: number
  DNSHost?: string
  DNSRecordType?: string
  DNSExpected?: string
  Domain?: string
  WarnDaysExpiry?: number
  MaxResponseTimeMs?: number
}

export interface CheckResult {
  ID: number
  MonitorID: number
  Status: MonitorStatus
  ResponseTimeMs: number
  Message: string
  CheckedAt: string
}

// ── Notification ──────────────────────────────────────────────────────────────

export type NotificationType = 'email' | 'telegram' | 'slack'

export interface Notification {
  ID: number
  Name: string
  Type: NotificationType
  Config: string
  Active: boolean
  NotifyOnDown: boolean
  NotifyOnUp: boolean
  NotifyAfterFail: number
}

export interface NotificationFormData {
  Name: string
  Type: NotificationType
  Config: string
  Active: boolean
  NotifyOnDown: boolean
  NotifyOnUp: boolean
  NotifyAfterFail: number
}

// ── User ──────────────────────────────────────────────────────────────────────

export interface User {
  id: number
  username: string
  email: string
  display_name: string
  role: 'admin' | 'user'
  auth_source: 'local' | 'ldap'
  active: boolean
}

export interface UserFormData {
  username: string
  email: string
  display_name: string
  password?: string
  role: 'admin' | 'user'
  active: boolean
}

// ── Integrations ─────────────────────────────────────────────────────────────

export interface PrometheusIntegration {
  enabled: boolean
  url: string
  basic_auth_user: string
  basic_auth_pass: string
}

// ── Status Page ───────────────────────────────────────────────────────────────

export interface StatusPage {
  ID: number
  CreatedAt: string
  Name: string
  Slug: string
  Description: string
  Public: boolean
  Monitors: Monitor[]
  Users: User[]
}

export interface StatusPageFormData {
  name: string
  slug: string
  description: string
  public: boolean
}
