FROM otel/opentelemetry-collector-contrib:0.91.0

# Copy configuration files
COPY configs/ /etc/phoenix/

# Set environment variables
ENV CONFIG_PATH=/etc/phoenix/main.yaml

# Health check endpoint
EXPOSE 13133

# Metrics endpoints
EXPOSE 8888 8889 8890 8891

# OTLP receiver port
EXPOSE 4318

# Run the collector
CMD ["--config", "${CONFIG_PATH}"]