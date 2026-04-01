# syntax=docker/dockerfile:1

# ── Stage 1: Build ────────────────────────────────────────────────────────────
FROM golang:1.26.1-alpine AS builder

WORKDIR /app

# Download dependencies first — cached unless go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build. CGO disabled for a static binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w" \
    -o /bin/server \
    ./cmd/server

# ── Stage 2: Final ────────────────────────────────────────────────────────────
# distroless/static has no shell, no package manager — just CA certs.
FROM gcr.io/distroless/static:nonroot

# Copy the binary from builder.
COPY --from=builder /bin/server /server

# nonroot UID/GID 65532 — never run as root.
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/server"]
