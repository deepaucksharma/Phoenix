# Build stage
FROM golang:1.21 as builder

WORKDIR /app
COPY . .

# Build with version info
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

RUN GOVERSION=$(go version) && \
    make build LDFLAGS="-X github.com/deepaucksharma/Phoenix/cmd/sa-omf-otelcol/version.Version=${VERSION} \
                         -X github.com/deepaucksharma/Phoenix/cmd/sa-omf-otelcol/version.Commit=${COMMIT} \
                         -X github.com/deepaucksharma/Phoenix/cmd/sa-omf-otelcol/version.BuildDate=${BUILD_DATE} \
                         -X 'github.com/deepaucksharma/Phoenix/cmd/sa-omf-otelcol/version.GoVersion=${GOVERSION}'"

# Runtime stage
FROM alpine:3.19  # Aligning with a common recent alpine version

RUN apk --no-cache add ca-certificates tzdata libc6-compat

WORKDIR /
COPY --from=builder /app/bin/sa-omf-otelcol /usr/local/bin/

# Default config location
COPY configs/default/config.yaml /etc/sa-omf/config.yaml
COPY deploy/policy.yaml /etc/sa-omf/policy.yaml

USER nobody
EXPOSE 8888 13133
ENTRYPOINT ["/usr/local/bin/sa-omf-otelcol"]
CMD ["--config=/etc/sa-omf/config.yaml"]
