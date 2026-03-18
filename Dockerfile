# ─── Stage 1: Build React frontend ───────────────────────────────────────────
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ─── Stage 2: Build Go binary ─────────────────────────────────────────────────
FROM golang:1.21-alpine AS go-builder
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Copy built frontend into the embed path
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

RUN CGO_ENABLED=1 GOOS=linux CGO_CFLAGS="-D_LARGEFILE64_SOURCE" \
    go build -ldflags="-s -w" -o statusy .

# ─── Stage 3: Minimal runtime image ──────────────────────────────────────────
FROM alpine:3.19
WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY --from=go-builder /app/statusy .

EXPOSE 8080

ENTRYPOINT ["./statusy"]
