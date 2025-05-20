# Phoenix Troubleshooting Guide

This guide provides solutions for common issues you might encounter when using Phoenix (SA-OMF). It covers build problems, runtime issues, configuration challenges, and performance troubleshooting.

## Build Issues

### Unable to Build Project

**Symptoms**:
- Build fails with module errors
- "Package not found" errors
- Version conflicts

**Solutions**:

1. **Refresh Dependencies**:
   ```bash
   go mod tidy
   go mod vendor
   make build
   ```

2. **Check Go Version**:
   ```bash
   go version
   ```
   Ensure you're using Go 1.24 or higher.

3. **Clean and Rebuild**:
   ```bash
   make clean
   make build
   ```

4. **Try Docker Build**:
   ```bash
   make docker
   ```
   This isolates the build environment.

### Vendor Directory Issues

**Symptoms**:
- Errors about missing packages
- "Use -mod=vendor or -mod=mod" warnings

**Solutions**:

1. **Create/Update Vendor Directory**:
   ```bash
   go mod vendor
   ```

2. **Force Vendor Mode**:
   ```bash
   make build GO_BUILD_FLAGS="-mod=vendor"
   ```

3. **Check Vendor Directory**:
   ```bash
   make vendor-check
   ```

### Docker Build Issues

**Symptoms**:
- Docker build fails
- Container won't start

**Solutions**:

1. **Check Docker Daemon**:
   ```bash
   docker info
   ```

2. **Clean Docker Environment**:
   ```bash
   docker system prune -f
   make docker
   ```

3. **Check Dockerfile**:
   Verify paths in deploy/docker/Dockerfile.

## Runtime Issues

### Collector Fails to Start

**Symptoms**:
- Immediate exit after starting
- "Failed to create pipeline" errors

**Solutions**:

1. **Check Configuration**:
   ```bash
   make run CONFIG=configs/default/config.yaml
   ```
   Use a known-good configuration.

2. **Check Logs**:
   Run with verbose logging:
   ```bash
   make run CONFIG=configs/development/config.yaml
   ```

3. **Validate Configuration Files**:
   ```bash
   make schema-check
   ```

4. **Remove Control Pipeline References**:
   Ensure your config.yaml doesn't reference removed components like pic_control_ext or pic_connector.

### Processor Registration Issues

**Symptoms**:
- "Processor not registered" errors
- "Failed to construct" errors

**Solutions**:

1. **Check Factory Registration**:
   Verify processor is registered in cmd/sa-omf-otelcol/main.go

2. **Check Processor Configuration**:
   Ensure processor name matches registration name exactly

3. **Debug with Reduced Pipeline**:
   Remove processors one by one to isolate the issue

### No Metrics Being Processed

**Symptoms**:
- No output in logs
- Empty Prometheus metrics

**Solutions**:

1. **Verify Receivers**:
   Check that receivers are correctly configured and sources are available

2. **Check Pipeline Configuration**:
   Ensure all components in the pipeline exist and are connected properly

3. **Inspect with Direct Logging**:
   Add logging exporter for direct visibility:
   ```yaml
   exporters:
     logging:
       verbosity: detailed
   ```

4. **Check Server Ports**:
   Verify ports are accessible and not blocked by firewall:
   ```bash
   netstat -tulpn | grep sa-omf
   ```

## Configuration Issues

### Invalid Configuration

**Symptoms**:
- "Failed to load config" errors
- YAML parsing errors

**Solutions**:

1. **Validate YAML Syntax**:
   ```bash
   yamllint configs/development/config.yaml
   ```

2. **Check for Indentation**:
   YAML is sensitive to indentation

3. **Check for Required Fields**:
   Review the [Configuration Reference](../configuration-reference.md)

4. **Use Default Configs**:
   Start with configs/default/ and modify incrementally

### Adaptation Not Working

**Symptoms**:
- Parameters not changing
- No PID controller metrics

**Solutions**:

1. **Check Controller Configuration**:
   Verify controllers are enabled in policy.yaml

2. **Check KPI Metrics**:
   Ensure target metrics exist and are correctly named

3. **Review PID Parameters**:
   Very low gain values may not produce visible changes

4. **Increase Logging Verbosity**:
   ```yaml
   exporters:
     logging:
       verbosity: detailed
   ```

### Oscillating Parameters

**Symptoms**:
- Parameter values changing rapidly
- Unstable behavior

**Solutions**:

1. **Adjust PID Parameters**:
   - Decrease kp (proportional gain)
   - Increase hysteresis_percent
   ```yaml
   controller:
     kp: 0.3  # Lower value
     ki: 0.1
     kd: 0.0  # Often best to start with zero
     hysteresis_percent: 5  # Higher value
   ```

2. **Enable Circuit Breakers**:
   ```yaml
   oscillation_detection:
     enabled: true
     window_size: 10
     threshold: 0.7
   ```

3. **Increase Adaptation Interval**:
   ```yaml
   adaptation_interval: 60s  # Longer time between adjustments
   ```

## Performance Issues

### High Memory Usage

**Symptoms**:
- Increasing memory consumption
- OOM killer termination

**Solutions**:

1. **Check Metric Cardinality**:
   High cardinality is the most common cause of memory issues

2. **Configure Cardinality Limits**:
   ```yaml
   cardinality_guardian:
     enabled: true
     max_cardinality: 5000
   ```

3. **Tune Others Rollup**:
   Aggregate more low-priority metrics:
   ```yaml
   others_rollup:
     low_priority_values: ["low", "medium"]
   ```

4. **Review Resource Limits**:
   ```yaml
   safety:
     resource_limits:
       max_memory_percent: 80
   ```

### High CPU Usage

**Symptoms**:
- Sustained high CPU utilization
- Slow processing

**Solutions**:

1. **Check Processor Patterns**:
   Complex regex patterns can be CPU-intensive

2. **Optimize Batch Sizes**:
   ```yaml
   processors:
     batch:
       send_batch_size: 8192
       timeout: 5s
   ```

3. **Adjust Adaptation Frequency**:
   Lengthen adaptation intervals to reduce CPU overhead

4. **Profile the Application**:
   ```bash
   go tool pprof http://localhost:8888/debug/pprof/profile
   ```

### Slow Adaptation

**Symptoms**:
- Parameters change very slowly
- Target KPIs not reached

**Solutions**:

1. **Tune PID Parameters**:
   - Increase kp for faster response
   - Add small ki for sustained correction
   ```yaml
   controller:
     kp: 0.5  # Higher value
     ki: 0.1  # Small positive value
   ```

2. **Check Target Values**:
   Ensure target values are realistic

3. **Decrease Adaptation Interval**:
   ```yaml
   adaptation_interval: 15s  # Shorter time for development
   ```

4. **Review Logs for Constraints**:
   Parameter may be hitting min/max limits

## Debugging Techniques

### Enabling Debug Logging

Set higher verbosity in your config.yaml:

```yaml
exporters:
  logging:
    verbosity: detailed  # or "debug" for even more detail
```

### Using Hot Reload for Quick Testing

Hot reload allows rapid testing of configuration changes:

```bash
make hot-reload
```

Edit configs while running to see immediate effects.

### Using Docker Exec for Container Debugging

If running in Docker:

```bash
# Find container ID
docker ps

# Connect to container
docker exec -it <container_id> /bin/sh

# View logs
docker logs <container_id>
```

### Inspecting Metrics for Troubleshooting

Access the metrics endpoint for direct inspection:

```bash
curl http://localhost:8889/metrics | grep aemf
```

Look for specific processor metrics to diagnose issues.

## Common Error Messages

### "Failed to create pipeline: processor not found"

**Cause**: Processor referenced in pipeline doesn't exist or isn't registered.

**Solution**: Check processor registration in main.go and verify pipeline configuration.

### "Processor 'X' failed to construct: missing required field"

**Cause**: Configuration missing required parameters.

**Solution**: Check processor documentation for required fields.

### "Error creating adapter metrics: metric not found"

**Cause**: Referenced KPI metric doesn't exist.

**Solution**: 
1. Verify metric name is correct
2. Ensure metric source is properly configured
3. Check for typos in metric_name

### "Setting k value outside allowed range"

**Cause**: Adaptation attempting to set parameter beyond min/max limits.

**Solution**:
1. Adjust min_value and max_value in configuration
2. Review PID parameters for potential instability
3. Check if target KPI is realistic

## When to Get Help

If you've tried the solutions in this guide without success, consider:

1. **Check GitHub Issues**:
   Search existing issues for similar problems

2. **Collect Diagnostics**:
   - Logs with detailed verbosity
   - Configuration files (redact sensitive data)
   - Environment information (Go version, OS)
   - Steps to reproduce

3. **Create a Detailed Issue**:
   Include all collected diagnostics and exact error messages

## Preventative Measures

### Regular Maintenance

1. **Stay Updated**:
   ```bash
   git pull
   go mod tidy
   go mod vendor
   ```

2. **Run Verification Regularly**:
   ```bash
   make verify
   ```

3. **Test Configuration Changes**:
   ```bash
   make schema-check
   ```

### Monitoring for Early Detection

Set up monitoring dashboards to detect issues early:

1. Set alerts for critical thresholds
2. Monitor adaptation behavior for instability
3. Watch resource usage trends

## Conclusion

Most Phoenix issues can be resolved through proper configuration, understanding the system's design, and applying the troubleshooting techniques in this guide. Remember that adaptive systems need time to stabilize, so allow Phoenix sufficient time to adjust parameters after changes.

If you encounter persistent issues not covered here, consult the more detailed component-specific documentation or reach out to the community for assistance.