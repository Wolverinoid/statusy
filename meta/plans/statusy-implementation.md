---
SECTION_ID: plans.statusy-implementation
TYPE: plan
STATUS: in_progress
PRIORITY: high
---

# Statusy — Initial Implementation

GOAL: Bootstrap the full project skeleton: Go backend + React frontend + all core subsystems
TIMELINE: Phase 1 (foundation)

## Task Checklist

### Phase 1: Project Scaffold
- [x] Create plan
- [*] go.mod + main.go entry point
- [ ] config/config.go (Viper YAML)
- [ ] config.example.yaml

### Phase 2: Database & Models
- [ ] internal/models/models.go (GORM models: User, Monitor, CheckResult, Notification, StatusPage)
- [ ] internal/database/database.go (SQLite/MySQL/Postgres selector + AutoMigrate)

### Phase 3: Auth
- [ ] internal/auth/jwt.go (issue/validate JWT)
- [ ] internal/auth/ldap.go (LDAP bind flow)
- [ ] internal/auth/handlers.go (login, logout, refresh endpoints)
- [ ] internal/api/middleware.go (JWT middleware, role check)

### Phase 4: API Router
- [ ] internal/api/router.go (Chi router, all route groups)
- [ ] internal/monitors/handlers.go (CRUD for monitors)

### Phase 5: Check Engine
- [ ] internal/monitors/engine.go (scheduler, goroutine per monitor)
- [ ] internal/monitors/checkers.go (HTTP, Port, Ping, Keyword, JSON, UDP, DNS, SSL, Domain)

### Phase 6: Notifications
- [ ] internal/notifications/dispatcher.go (channel + worker)
- [ ] internal/notifications/smtp.go
- [ ] internal/notifications/telegram.go
- [ ] internal/notifications/slack.go

### Phase 7: Prometheus
- [ ] internal/metrics/prometheus.go (gauges, counters, /metrics handler)

### Phase 8: Frontend Scaffold
- [ ] frontend/package.json + vite config + tailwind
- [ ] frontend/src/main.tsx + App.tsx
- [ ] frontend/src/api/client.ts (axios wrapper)

### Phase 9: Docker & Build
- [ ] Dockerfile (multi-stage: build frontend → embed → build Go binary)
- [ ] docker-compose.yml
- [ ] Makefile

## Success Criteria
- [ ] `go build ./...` succeeds
- [ ] `npm install && npm run build` in frontend succeeds
- [ ] App starts, serves embedded frontend, exposes /api and /metrics
- [ ] SQLite DB auto-created on first run
