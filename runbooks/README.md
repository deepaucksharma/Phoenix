# Phoenix-vNext Operational Runbooks

This directory contains operational runbooks, procedures, and troubleshooting guides for the Phoenix-vNext platform.

## Directory Structure

```
runbooks/
├── incident-response/       # Emergency response procedures
│   ├── high-cardinality-explosion.md
│   └── collector-oom.md
├── operational-procedures/  # Standard operating procedures  
│   └── metric-standards.md
└── troubleshooting/        # Common issues and solutions
    └── common-issues.md
```

## Quick Links

### 🚨 Emergency Response
- [High Cardinality Explosion](incident-response/high-cardinality-explosion.md) - When metrics cardinality exceeds limits
- [Collector OOM](incident-response/collector-oom.md) - When OpenTelemetry Collector runs out of memory

### 📋 Operational Procedures
- [Metric Standards](operational-procedures/metric-standards.md) - Naming conventions and cardinality guidelines

### 🔧 Troubleshooting
- [Common Issues](troubleshooting/common-issues.md) - Frequently encountered problems and solutions

## On-Call Quick Reference

### Critical Alerts Response Times
| Alert | Severity | Response Time | Runbook |
|-------|----------|--------------|---------|
| CollectorDown | Critical | 5 minutes | [Collector OOM](incident-response/collector-oom.md) |
| HighCardinalityExplosion | Critical | 15 minutes | [High Cardinality](incident-response/high-cardinality-explosion.md) |
| PrometheusDown | Critical | 5 minutes | [Common Issues](troubleshooting/common-issues.md#prometheus-issues) |
| HighMemoryUsage | Warning | 30 minutes | [Common Issues](troubleshooting/common-issues.md#performance-issues) |

### Emergency Contacts
- **On-Call**: Check PagerDuty schedule
- **Slack**: #phoenix-incidents
- **Escalation**: Platform team lead

## Common Commands

### Health Checks
```bash
# Quick system health check
kubectl -n phoenix-vnext get pods
kubectl -n phoenix-vnext top pods
curl http://prometheus:9090/-/healthy
```

### Force Optimization Mode
```bash
# Emergency aggressive mode
kubectl -n phoenix-vnext create configmap emergency-control \
  --from-literal=optimization_mode=aggressive \
  --dry-run=client -o yaml | kubectl apply -f -
```

### Scale Up Collectors
```bash
# Emergency scale
kubectl -n phoenix-vnext scale deployment otel-collector-main --replicas=5
```

## Runbook Standards

### Structure
Each runbook should contain:
1. **Alert Name** - Prometheus alert that triggers this runbook
2. **Description** - What the issue is
3. **Severity** - Critical/Warning/Info
4. **Impact** - Business/technical impact
5. **Detection** - How to verify the issue
6. **Immediate Actions** - Steps to mitigate
7. **Root Cause Analysis** - How to investigate
8. **Long-term Fixes** - Permanent solutions
9. **Prevention** - How to avoid recurrence
10. **Communication** - Who to notify

### Updates
- Review quarterly
- Update after incidents
- Test procedures in staging
- Version control all changes

## Training Resources

### New On-Call Engineers
1. Read [Metric Standards](operational-procedures/metric-standards.md)
2. Review [Common Issues](troubleshooting/common-issues.md)
3. Practice emergency procedures in staging
4. Shadow experienced on-call engineer

### Drills
- Monthly cardinality explosion drill
- Quarterly disaster recovery test
- Annual full system failure simulation

## Contributing

### Adding New Runbooks
1. Use the template below
2. Test all commands
3. Get review from senior engineer
4. Update this index

### Template
```markdown
# Runbook: [Issue Name]

## Alert Name
`AlertName`

## Description
Brief description of the issue

## Severity
Critical/Warning/Info

## Impact
- Business impact
- Technical impact

## Detection
How to detect and verify the issue

## Immediate Actions
1. First mitigation step
2. Second step
...

## Root Cause Analysis
How to investigate the root cause

## Long-term Fixes
Permanent solutions

## Prevention
How to prevent recurrence

## Communication
Who to notify and how

## References
- Related documentation
- External resources
```

## Metrics and Monitoring

### Runbook Usage Tracking
Track which runbooks are used most frequently to identify:
- Common issues needing automation
- Training gaps
- System improvements needed

### Effectiveness Metrics
- Mean Time to Detect (MTTD)
- Mean Time to Resolve (MTTR)
- Runbook completion rate
- False positive rate

## Related Documentation
- [Architecture Documentation](../docs/ARCHITECTURE.md)
- [Troubleshooting Guide](../docs/TROUBLESHOOTING.md)
- [Pipeline Analysis](../docs/PIPELINE_ANALYSIS.md)