FROM otel/opentelemetry-collector-contrib:0.91.0

# Copy observer configuration
COPY config/observer.yaml /etc/phoenix/observer.yaml

# Set environment variables
ENV CONFIG_PATH=/etc/phoenix/observer.yaml

# Health check endpoint
EXPOSE 13134

# KPI metrics endpoint
EXPOSE 9888

# Run the collector
CMD ["--config", "${CONFIG_PATH}"]