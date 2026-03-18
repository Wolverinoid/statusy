.PHONY: all build run dev frontend tidy lint test docker clean

# ─── Config ───────────────────────────────────────────────────────────────────
BINARY   := statusy
GO       := go
NODE     := npm

# ─── Default ──────────────────────────────────────────────────────────────────
all: build

# ─── Go ───────────────────────────────────────────────────────────────────────
build: frontend
	CGO_ENABLED=1 $(GO) build -ldflags="-s -w" -o $(BINARY) .

run:
	$(GO) run .

tidy:
	$(GO) mod tidy

lint:
	golangci-lint run ./...

test:
	$(GO) test ./... -v -race

# ─── Frontend ─────────────────────────────────────────────────────────────────
frontend:
	cd frontend && $(NODE) ci && $(NODE) run build

frontend-dev:
	cd frontend && $(NODE) run dev

# ─── Dev (backend only, frontend served by Vite proxy) ───────────────────────
dev:
	STATUSY_SERVER_SERVE_FRONTEND=false $(GO) run .

# ─── Docker ───────────────────────────────────────────────────────────────────
docker:
	docker build -t statusy:latest .

docker-up:
	docker compose up -d

docker-down:
	docker compose down

# ─── Clean ────────────────────────────────────────────────────────────────────
clean:
	rm -f $(BINARY)
	rm -rf frontend/dist
