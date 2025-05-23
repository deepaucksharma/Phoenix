FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o benchmark-controller .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/benchmark-controller .

# Expose port for health checks
EXPOSE 8080

CMD ["./benchmark-controller"]