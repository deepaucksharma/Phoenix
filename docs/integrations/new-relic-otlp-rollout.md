# New Relic OTLP Export Rollout Plan

This document outlines the rollout plan for deploying the process-metrics-only OTLP export configuration to New Relic.

## Phased Rollout Approach

### Phase 1: Development Testing (1 week)

1. **Setup Test Environment**
   - Deploy Phoenix SA-OMF with New Relic OTLP configuration in a development environment
   - Configure with lower collection frequency (30s intervals)

2. **Initial Validation**
   - Verify metrics are appearing in New Relic
   - Check cardinality levels
   - Validate PID controller behavior
   - Ensure process priorities are being respected

3. **Stress Testing**
   - Run with artificially high process count to test adaptive behavior
   - Validate rollup functionality under high cardinality conditions
   - Test safety mode activation and recovery

### Phase 2: Limited Production Deployment (1 week)

1. **Deploy to Limited Production Hosts**
   - Select 2-3 production hosts for initial deployment
   - Configure with shadow mode enabled (adaptive features active but conservative)

2. **Monitoring**
   - Closely monitor metric rates and cardinality
   - Track New Relic billing impact
   - Evaluate PID controller stability

3. **Tuning**
   - Adjust PID parameters based on production patterns
   - Fine-tune priority rules based on actual workloads
   - Optimize histogram boundaries for production metrics

### Phase 3: Full Production Rollout (2 weeks)

1. **Gradual Expansion**
   - Roll out to 25% of hosts
   - After 3 days, expand to 50% of hosts
   - After 3 more days, expand to 75% of hosts
   - Complete rollout by end of phase

2. **Continuous Monitoring**
   - Set up alerts for abnormal metric patterns
   - Monitor New Relic billing daily
   - Check for any performance impact on hosts

3. **Documentation Update**
   - Update runbooks with observed best practices
   - Document common patterns and solutions
   - Finalize tuning recommendations

## Testing Checklist

### Functional Testing

- [ ] Verify all critical processes appear as individual metrics
- [ ] Confirm less important processes are rolled up
- [ ] Validate histogram bucketing is optimal for visualization
- [ ] Check that cardinality stays below target levels
- [ ] Verify OTLP authentication to New Relic works

### Performance Testing

- [ ] Measure CPU overhead of collection (should be under 1%)
- [ ] Measure memory usage (should be under 200MB)
- [ ] Calculate network bandwidth consumption
- [ ] Test collection stability over 72-hour period
- [ ] Measure recovery time after host resource pressure

### Safety Testing

- [ ] Artificially trigger safety mode and verify behavior
- [ ] Simulate network outages to test retry behavior
- [ ] Test with abnormal process patterns (100+ short-lived processes)
- [ ] Verify behavior when New Relic endpoint is unavailable
- [ ] Test cardinality limit enforcement

## Rollback Plan

If serious issues are encountered, follow these rollback steps:

1. **Immediate Mitigation**
   - Switch to shadow mode (monitoring only)
   - Reduce collection frequency
   - Apply more aggressive cardinality limits

2. **Full Rollback If Needed**
   - Stop OTLP exporter
   - Revert to previous configuration
   - Document specific issues encountered

3. **Analysis and Correction**
   - Analyze logs and metrics to identify root cause
   - Fix issues in a development environment
   - Re-test thoroughly before attempting rollout again

## Success Criteria

The rollout will be considered successful when:

1. Process metrics are reliably visible in New Relic
2. Cardinality remains under 1000 unique metric names
3. CPU overhead stays below 1% on average
4. Memory usage remains stable (no leaks)
5. Adaptive PID control maintains target coverage
6. Important processes are consistently tracked individually
7. Histograms display with appropriate bucketing

## Post-Implementation Review

Schedule a review 2 weeks after full deployment to assess:

1. Metric quality and coverage
2. Performance impact
3. New Relic integration effectiveness
4. Any further tuning needs
5. Lessons learned for future integrations