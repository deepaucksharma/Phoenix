# Dockerfile for the control-loop-actuator service
FROM alpine:3.19

# yq for YAML processing, jq for JSON (Prometheus output), bc for arithmetic, coreutils for GNU date
RUN apk add --no-cache bash curl jq yq bc coreutils

WORKDIR /app

# Scripts and templates will be copied via docker-compose volume mounts.
# Ensure execute permissions are set on the script via docker-compose command.
ENTRYPOINT ["/bin/bash"]