FROM alpine:3.18

# Install dependencies
RUN apk add --no-cache bash curl jq

# Create app directory
WORKDIR /app

# Copy generator script
COPY src/phoenix-metric-generator.sh /app/

# Make script executable
RUN chmod +x /app/phoenix-metric-generator.sh

# Set default environment variables
ENV PROMETHEUS_PUSHGATEWAY_URL=http://pushgateway:9091
ENV METRIC_COUNT=1000
ENV DIMENSION_COUNT=5

# Run the generator
CMD ["/app/phoenix-metric-generator.sh"]