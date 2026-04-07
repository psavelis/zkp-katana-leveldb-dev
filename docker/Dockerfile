# Multi-stage build for ZKP Demo
# Stage 1: Build
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binaries
RUN CGO_ENABLED=1 GOOS=linux go build -o /demo ./cmd/demo
RUN CGO_ENABLED=1 GOOS=linux go build -o /benchmark ./cmd/benchmark

# Stage 2: Runtime
FROM alpine:3.19

RUN apk add --no-cache ca-certificates curl

WORKDIR /app

# Copy binaries from builder
COPY --from=builder /demo /app/demo
COPY --from=builder /benchmark /app/benchmark

# Create data directory
RUN mkdir -p /app/data

# Set environment variables
ENV LEVELDB_PATH=/app/data/proofs.db
ENV KATANA_RPC_URL=http://katana:5050

# Expose no ports (CLI application)
# Health check via benchmark
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD /app/benchmark || exit 1

ENTRYPOINT ["/app/demo"]
