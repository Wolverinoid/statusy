---
SECTION_ID: docs.statusy-specification
TYPE: note
---

# Statusy — Preliminary Project Specification

## Overview

**Statusy** is a free, open-source, self-hosted monitoring tool. It is designed to be simple to deploy and use, inspired by tools like Uptime Kuma. Users can create monitors that run in the background, check various endpoints/services, and send alerts when status changes occur.

---

## 1. Technical Stack

| Component | Technology |
|-----------|------------|
| **Backend** | Go (1.21+) |
| **Frontend** | React (Vite + TailwindCSS), embedded via `go:embed` |
| **Database** | SQLite (default), MySQL, PostgreSQL (via GORM) |
| **Auth** | JWT (JSON Web Tokens) with Local/LDAP providers |
| **Config** | YAML (via Viper) |
| **Metrics** | Prometheus (`/metrics` endpoint) |
| **API** | RESTful (Chi router) |

---

## 2. User Roles & Permissions

### Roles

| Role  | Description |
|-------|-------------|
| **Admin** | Full access: manage users, monitors, notifications, system settings |
| **User** | Limited access: can only view/manage monitors and pages explicitly assigned by admin |

### Permissions Model

- Admin can create/edit/delete any user.
- Admin assigns users access to specific **Status Pages**.
- Users can only see monitors and status pages they have been granted access to.
- Only admins can manage notification integrations and system-level settings.

---

## 3. Database Support

Statusy supports three database backends, selectable at installation/configuration time:

| Database       | Notes |
|----------------|-------|
| **SQLite**     | Default, zero-config, suitable for small deployments |
| **MySQL**      | Recommended for medium/large teams |
| **PostgreSQL** | Recommended for production, advanced use cases |

Database selection is done via a configuration file or environment variable. Schema migrations are handled automatically on startup.

---

## 4. Monitors

Monitors are background workers that periodically check a target and update its status. Each monitor has:

- **Name** — human-readable label
- **Type** — see below
- **Check interval** — how often to run (e.g., every 30s, 1m, 5m)
- **Status** — `UP`, `DOWN`, `PENDING`, `MAINTENANCE`
- **Retries** — number of retries before marking as DOWN
- **Timeout** — max wait time per check

### Monitor Types

#### 4.1 HTTP Monitor
- Sends HTTP/HTTPS request to a URL.
- Checks for expected status code (e.g., 200).
- Supports GET, POST, HEAD methods.
- Optional: custom headers, request body.

#### 4.2 Port Monitor
- Checks if a TCP port is open on a given host.
- Parameters: host, port.

#### 4.3 Ping Monitor
- Sends ICMP ping to a host.
- Checks for response and measures latency.

#### 4.4 Keyword Monitor
- Fetches a URL and checks if a specified keyword/phrase is present (or absent) in the response body.
- Parameters: URL, keyword, match mode (`contains` / `not contains`).

#### 4.5 JSON API Monitor
- Fetches a JSON endpoint and validates the response structure.
- Checks for presence of specific keys/values using a configurable JSON path or schema.
- Parameters: URL, expected JSON path/key, expected value (optional).

#### 4.6 UDP Monitor
- Sends a UDP packet to a host:port and checks for a response.
- Parameters: host, port, payload (optional), expected response (optional).

#### 4.7 Response Time Monitor
- Measures HTTP response time and alerts if it exceeds a defined threshold.
- Parameters: URL, max acceptable response time (ms).

#### 4.8 DNS Monitor
- Resolves a DNS record and validates the result.
- Supported record types: A, AAAA, CNAME, MX, TXT, NS.
- Parameters: hostname, record type, expected value.

#### 4.9 SSL Certificate Monitor
- Checks SSL certificate validity for a given domain.
- Alerts when certificate is expired or will expire within a configurable number of days.
- Parameters: domain, warning threshold (days before expiry).

#### 4.10 Domain Expiry Monitor
- Checks domain WHOIS data for expiration date.
- Alerts when domain will expire within a configurable number of days.
- Parameters: domain, warning threshold (days before expiry).

---

## 5. Background Check Engine

- All monitors run as background jobs on a configurable schedule.
- Each monitor runs independently; failures in one do not affect others.
- Check results are stored in the database (history, response time, status).
- On every status change (`UP → DOWN`, `DOWN → UP`, etc.), the alert pipeline is triggered.
- Maintenance mode: monitors can be paused individually or globally.

---

## 6. Alerts & Notifications

Alerts are triggered on status changes. Each monitor can have one or more notification channels attached.

### Notification Channels

#### 6.1 Email
- SMTP configuration (host, port, user, password, TLS).
- Supports multiple recipients.
- Customizable subject and body template.

#### 6.2 Telegram
- Bot token + Chat ID configuration.
- Sends message on status change.
- Supports Markdown formatting in messages.

#### 6.3 Slack
- Webhook URL configuration.
- Sends formatted message to a Slack channel on status change.
- Supports custom channel and username override.

### Alert Rules
- Notify on: `DOWN`, `UP` (recovery), `SSL expiry warning`, `Domain expiry warning`.
- Configurable: notify only after N consecutive failures (to reduce noise).

---

## 7. Status Pages

- Public or private status pages showing monitor statuses.
- Admin can create multiple status pages and assign monitors to each.
- Users can be granted access to specific status pages.
- Each page shows: current status, uptime percentage, response time chart, incident history.

---

## 8. Prometheus Export

- Statusy exposes a `/metrics` endpoint in Prometheus format.
- Exported metrics include:

| Metric | Description |
|--------|-------------|
| `statusy_monitor_up` | 1 if monitor is UP, 0 if DOWN |
| `statusy_monitor_response_time_ms` | Last response time in milliseconds |
| `statusy_monitor_ssl_days_remaining` | Days until SSL cert expiry |
| `statusy_monitor_domain_days_remaining` | Days until domain expiry |
| `statusy_check_total` | Total number of checks performed |
| `statusy_check_failures_total` | Total number of failed checks |

- The endpoint can be secured with a bearer token.
- Labels include: `monitor_id`, `monitor_name`, `monitor_type`.

---

## 9. Authentication Providers

Statusy supports multiple authentication backends. The active provider is configured via the config file or environment variables. Providers can be mixed (e.g., local accounts + LDAP).

### 9.1 Local Authentication (default)
- Username + password stored in the database.
- Passwords hashed with bcrypt.
- Built-in password reset flow (via email).

### 9.2 LDAP / Active Directory
- Connects to an LDAP/AD server to authenticate users.
- On successful LDAP login, a local user record is auto-provisioned (or synced).
- Configuration parameters:

| Parameter | Description |
|-----------|-------------|
| `ldap.url` | LDAP server URL (e.g., `ldap://ldap.example.com:389`) |
| `ldap.bind_dn` | Service account DN for binding |
| `ldap.bind_password` | Service account password |
| `ldap.base_dn` | Base DN for user search |
| `ldap.user_filter` | LDAP filter to find user (e.g., `(uid={username})`) |
| `ldap.tls` | Enable LDAPS / StartTLS (`none`, `tls`, `starttls`) |
| `ldap.attr.email` | Attribute mapped to user email |
| `ldap.attr.display_name` | Attribute mapped to display name |
| `ldap.default_role` | Role assigned to new LDAP users (`admin` / `user`) |

- Group-based role mapping: optionally map LDAP groups to Statusy roles.
- Admin can disable local login and enforce LDAP-only auth.

### 9.3 OAuth 2.0 / OIDC (future)
- Generic OAuth2/OIDC provider support (GitHub, Google, Keycloak, etc.).
- Planned for v2, noted here for architecture awareness.

### Auth Provider UI (Admin Settings)
- Admin panel has an **Authentication** section.
- Shows currently active provider(s).
- LDAP: form to enter/test connection settings with a **Test Connection** button.
- Toggle to allow/disallow local login when LDAP is active.
- User list shows auth source per user (`local` / `ldap`).

### Security Considerations
- LDAP bind password stored encrypted at rest.
- LDAP connections use TLS by default; plaintext must be explicitly enabled.
- Failed login attempts are rate-limited and logged regardless of provider.
- Sessions are invalidated on password change (local) or on LDAP account disable (checked at next login).
- Audit log records login events with provider type and IP address.

---

## 10. Out of Scope (v1)

- Mobile native apps
- On-call scheduling / escalation policies
- Multi-region distributed checks
- OAuth 2.0 / OIDC login (planned for v2)
