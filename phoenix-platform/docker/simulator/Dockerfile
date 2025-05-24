# Build stage
FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o process-simulator cmd/simulator/main.go

# Runtime stage
FROM ubuntu:22.04

# Install stress-ng and other utilities
RUN apt-get update && apt-get install -y \
    stress-ng \
    procps \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Copy binary from builder
COPY --from=builder /app/process-simulator /usr/local/bin/

# Copy default profiles (if any)
RUN mkdir -p /etc/phoenix/profiles

# Health check
HEALTHCHECK --interval=30s --timeout=3s \
  CMD ps aux | grep -v grep | grep process-simulator || exit 1

# Run as non-root user
RUN groupadd -g 1000 phoenix && \
    useradd -r -u 1000 -g phoenix phoenix

USER phoenix

ENTRYPOINT ["/usr/local/bin/process-simulator"]