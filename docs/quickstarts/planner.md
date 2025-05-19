# Planner Quickstart

This guide will help you get started as a Planner in the Phoenix project.

## Day One Steps

1. **Set up your environment**
   ```bash
   git config --global user.role planner
   ```

2. **Understand the roadmap**
   - Review `docs/architecture/adr/` for architectural decisions
   - Check existing tasks in `tasks/` directory

3. **Create a new task**
   ```bash
   hack/create-task.sh "Implement feature X"
   ```

4. **Edit the task file to add details**
   - Set the appropriate area
   - Add clear acceptance criteria
   - Expand the description with context
   - Set appropriate priority
   - Specify any task dependencies

5. **Validate tasks**
   ```bash
   make planner-check
   ```

## Task Creation Checklist

- [ ] Task ID is unique
- [ ] Title clearly describes the work
- [ ] Area is specified correctly
- [ ] Acceptance criteria are clear and testable
- [ ] Description provides enough context
- [ ] Dependencies are correctly identified

## Task Format

```yaml
id: PID-001
title: "Add feature X"
state: open
priority: high|medium|low
created_at: "YYYY-MM-DD"
assigned_to: ""
area: "internal/processor/example"
depends_on: ["PID-000"]
acceptance:
  - "Feature X passes tests"
  - "Performance doesn't degrade"
description: |
  Detailed description of the task.
  
  Include context, rationale, and any other
  information that will help implementers.
```

## Best Practices

- Break down large features into smaller, manageable tasks
- Ensure tasks have clear acceptance criteria
- Specify area to help with routing to the right implementer
- Set realistic priorities
