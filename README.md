# Statusy

> Free, open-source, self-hosted uptime monitoring dashboard.

Statusy monitors your services (HTTP, TCP, DNS, SSL, ping, and more), sends alerts on status changes, and exposes public status pages — all in a single Go binary with an embedded React frontend.

---

## Features

- **10 monitor types** — HTTP, Port, Ping, Keyword, JSON API, UDP, Response Time, DNS, SSL Certificate, Domain Expiry
- **Alerts** — Email (SMTP), Telegram, Slack
- **Public status pages** — shareable per-service status with uptime history
- **Prometheus metrics** — `/metrics` endpoint with monitor state, response time, SSL/domain expiry
- **Dark & light theme** — toggle in the sidebar, persisted in localStorage
- **Role-based access** — Admin and User roles
- **Auth** — Local (bcrypt) + LDAP/Active Directory
- **Multi-database** — SQLite (default), MySQL, PostgreSQL via GORM
- **Single binary** — frontend embedded via `go:embed`, zero external dependencies at runtime

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.21+, Chi router, GORM |
| Frontend | React 18, Vite 6, TailwindCSS 3, TypeScript |
| Database | SQLite / MySQL / PostgreSQL |
| Auth | JWT + Local / LDAP |
| Metrics | Prometheus |
| Config | YAML via Viper |

---

## Quick Start

### Docker (recommended)

```bash
docker compose up -d
```

App will be available at **http://localhost:8080**.

Data is persisted in the `statusy_data` Docker volume. To use a different port:

```bash
PORT=9090 docker compose up -d
```

### Binary (build from source)

**Prerequisites:** Go 1.21+, Node.js 18+, npm

```bash
# 1. Build frontend + Go binary
make build

# 2. Run
./statusy
```

### Development mode

Run the Go backend and Vite dev server separately (with HMR):

```bash
# Terminal 1 — backend (API only, no embedded frontend)
make dev

# Terminal 2 — frontend dev server with proxy to backend
make frontend-dev
```

Frontend dev server runs on **http://localhost:5173** and proxies `/api` to the Go backend.

---

## First Run & Setup

On first launch Statusy detects no admin account and redirects to `/setup` where you create the initial admin user. After that you're taken to the login page.

**Default dev credentials (if seeded):**
```
Username: admin
Password: 12345678
```

---

## Configuration

Statusy is configured via a YAML file (`config.yaml`) or environment variables (prefixed `STATUSY_`).

Key environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `STATUSY_SERVER_ADDRESS` | `:8080` | Listen address |
| `STATUSY_SERVER_SERVE_FRONTEND` | `true` | Serve embedded frontend |
| `STATUSY_DATABASE_DRIVER` | `sqlite` | `sqlite`, `mysql`, `postgres` |
| `STATUSY_DATABASE_DSN` | `data/statusy.db` | Database connection string |
| `STATUSY_AUTH_JWT_SECRET` | *(auto-generated)* | JWT signing secret |

---

## Makefile Reference

| Command | Description |
|---------|-------------|
| `make build` | Build frontend then compile Go binary |
| `make run` | Run Go binary directly (no rebuild) |
| `make dev` | Run backend only (frontend served by Vite) |
| `make frontend` | Build frontend (`npm ci && npm run build`) |
| `make frontend-dev` | Start Vite dev server |
| `make test` | Run Go tests with race detector |
| `make lint` | Run golangci-lint |
| `make docker` | Build Docker image `statusy:latest` |
| `make docker-up` | Start via docker compose |
| `make docker-down` | Stop docker compose stack |
| `make clean` | Remove binary and `frontend/dist` |

---

## Frontend Build Notes

The frontend lives in `frontend/` and is built with **Vite 6 + TailwindCSS 3**.

```bash
cd frontend
npm ci
npm run build   # outputs to frontend/dist (embedded into Go binary)
npm run dev     # dev server on :5173
```

### Theme system

- Dark mode is the default; toggled via the sun/moon button in the sidebar.
- Theme class (`dark`) is applied to `<html>` before first paint to prevent flash.
- `StatusPageView` (public, outside auth layout) also respects the stored theme.
- Light theme overrides are written as flat `html:not(.dark) .class` rules — **not nested** — to avoid a PostCSS `@apply` limitation with nested CSS.

### Vite upgrade (5 → 6)

Vite was upgraded from 5.x to **6.4.1** to resolve a moderate esbuild vulnerability ([GHSA-67mh-4wv8-2f99](https://github.com/advisories/GHSA-67mh-4wv8-2f99)) that allowed cross-origin requests to the dev server. No config changes were required.

---

## npm Audit

Last audit: **0 vulnerabilities** (after Vite 6 upgrade).

```
$ cd frontend && npm audit
found 0 vulnerabilities
```

Previously 2 moderate vulnerabilities in `esbuild ≤0.24.2` (via Vite 5) — resolved by upgrading to Vite 6.4.1.

---

## Monitor Types

| Type | What it checks |
|------|---------------|
| `http` | HTTP/HTTPS status code, optional keyword |
| `port` | TCP port open/closed |
| `ping` | ICMP reachability and latency |
| `keyword` | Presence/absence of text in HTTP response body |
| `json_api` | JSON key/value in API response |
| `udp` | UDP packet send/receive |
| `response_time` | HTTP response time vs threshold |
| `dns` | DNS record resolution and expected value |
| `ssl` | SSL certificate validity and expiry warning |
| `domain_expiry` | Domain WHOIS expiry warning |

---

## Prometheus Metrics

Exposed at `/metrics`:

| Metric | Description |
|--------|-------------|
| `statusy_monitor_up` | 1 = UP, 0 = DOWN |
| `statusy_monitor_response_time_ms` | Last response time (ms) |
| `statusy_monitor_ssl_days_remaining` | Days until SSL cert expiry |
| `statusy_monitor_domain_days_remaining` | Days until domain expiry |
| `statusy_check_total` | Total checks performed |
| `statusy_check_failures_total` | Total failed checks |

Labels: `monitor_id`, `monitor_name`, `monitor_type`.

---

## Contributing

1. Fork the repo and create a feature branch
2. Follow existing code style (Go: `gofmt`; TS: ESLint config in repo)
3. Add tests for new backend logic (`make test`)
4. Open a PR with a clear description

### Running tests

```bash
# Go tests
make test

# Frontend lint
cd frontend && npm run lint
```

---

## License

MIT
