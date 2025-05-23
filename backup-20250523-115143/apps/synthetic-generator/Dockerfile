# Dockerfile for the Go-based synthetic metrics generator
FROM golang:1.22.3-alpine3.19 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /synthetic-generator .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=builder /synthetic-generator /synthetic-generator
ENTRYPOINT ["/synthetic-generator"]