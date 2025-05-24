# Build stage
FROM golang:1.21-alpine AS builder

# Install dependencies
RUN apk add --no-cache git make ca-certificates

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o phoenix-api cmd/api/main.go

# Runtime stage
FROM alpine:3.19

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /

# Copy binary from builder
COPY --from=builder /app/phoenix-api .

# Create non-root user
RUN addgroup -g 1000 phoenix && \
    adduser -D -u 1000 -G phoenix phoenix

USER phoenix

EXPOSE 5050 8080

ENTRYPOINT ["/phoenix-api"]