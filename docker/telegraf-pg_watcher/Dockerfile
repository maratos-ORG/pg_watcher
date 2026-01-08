# ---- Stage 1: Build pg_watcher ----
FROM golang:1.25-alpine AS builder

# Build argument for version
ARG VERSION=docker

WORKDIR /build

# Copy go mod files and source code
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# Build pg_watcher binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X main.build=${VERSION}" \
    -o pg_watcher \
    ./cmd/pg_watcher/main.go

# ---- Stage 2: Final image with Telegraf ----
# FROM telegraf:latest
FROM telegraf:alpine

# Copy pg_watcher binary from builder and make it executable
COPY --from=builder /build/pg_watcher /usr/local/bin/pg_watcher
RUN chmod +x /usr/local/bin/pg_watcher

