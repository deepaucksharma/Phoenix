FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o analytics cmd/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/analytics .

# Create static directory for web assets
RUN mkdir -p /root/static

EXPOSE 8080

CMD ["./analytics"]