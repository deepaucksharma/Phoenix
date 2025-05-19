# Implementer Quickstart

This guide will help you get started as an Implementer in the Phoenix project.

## Day One Steps

1. **Set up your environment**
   ```bash
   git config --global user.role implementer
   ```

2. **Pick a task to work on**
   ```bash
   ls tasks/
   # Find a task with state: open
   ```

3. **Create a branch for your task**
   ```bash
   scripts/dev/create-branch.sh implementer PID-001 "Add feature"
   ```

4. **Write code and tests**
   - If creating a new component: `scripts/dev/new-component.sh processor my_processor`
   - Make sure to write tests: `test/interfaces/updateable_processor_test.go` has examples
   - Add appropriate logging and metrics

5. **Run checks before submitting**
   ```bash
   make implementer-check
   ```

## PR Submission Checklist

- [ ] Code follows style guidelines
- [ ] Tests cover your changes
- [ ] Documentation updated if needed
- [ ] PR template filled out correctly (ROLE and TASKS)
- [ ] Self-review completed

## Common Tasks

### Adding a new processor

```bash
scripts/dev/new-component.sh processor my_processor
```

### Implementing the UpdateableProcessor interface

See `internal/processor/prioritytagger/processor.go` for a complete example.

### Adding metrics

Use the MetricsEmitter:

```go
metrics := metrics.NewMetricsEmitter(meter, "processor_name", id)
counter, _ := metrics.RegisterCounter("count_name", "Count description")
gauge, _ := metrics.RegisterGauge("gauge_name", "Gauge description")
```
