# ── Build stage ───────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Cache module downloads separately from source for faster rebuilds.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /rate-limiter ./cmd/server

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM scratch

# Copy the static binary.
COPY --from=builder /rate-limiter /rate-limiter

# Copy TLS root certificates so HTTPS calls (e.g. to Redis Cloud) work.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8080

ENTRYPOINT ["/rate-limiter"]
