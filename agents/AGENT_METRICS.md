# Agent Metrics

This document describes the metrics emitted by the Phoenix agent system, their interpretation, and how they can be used for monitoring and debugging.

## Core Agent Metrics

### Agent Activity Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `agent.tasks_scheduled` | Counter | Total number of tasks scheduled for agents | `agent_type` |
| `agent.tasks_completed` | Counter | Total number of tasks successfully completed | `agent_type`, `status` |
| `agent.execution_time` | Histogram | Time taken to complete agent tasks | `agent_type`, `task_type` |
| `agent.memory_usage` | Gauge | Memory consumed by agent processes | `agent_type` |
| `agent.error_count` | Counter | Number of errors encountered during agent execution | `agent_type`, `error_type` |

### Agent Communication Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `agent.messages_sent` | Counter | Number of messages sent between agents | `source_agent`, `target_agent` |
| `agent.message_size` | Histogram | Size of messages exchanged between agents | `source_agent`, `target_agent` |
| `agent.response_time` | Histogram | Time taken for an agent to respond to messages | `agent_type` |

## Agent-Specific Metrics

### Planner Agent

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `planner.plans_created` | Counter | Number of plans created | `complexity` |
| `planner.planning_time` | Histogram | Time taken to create plans | `complexity` |
| `planner.plan_revisions` | Counter | Number of times plans were revised | |

### Implementer Agent

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `implementer.code_lines_written` | Counter | Lines of code written | `language` |
| `implementer.implementation_time` | Histogram | Time taken to implement features | `complexity` |
| `implementer.build_success_rate` | Gauge | Percentage of successful builds | |

### Tester Agent

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `tester.tests_created` | Counter | Number of tests created | `test_type` |
| `tester.test_coverage` | Gauge | Code coverage percentage | `component` |
| `tester.tests_passed` | Counter | Number of tests that passed | |
| `tester.tests_failed` | Counter | Number of tests that failed | |

### Reviewer Agent

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `reviewer.reviews_completed` | Counter | Number of code reviews completed | |
| `reviewer.issues_found` | Counter | Number of issues identified during review | `severity` |
| `reviewer.review_time` | Histogram | Time taken to complete reviews | `code_size` |

### Architect Agent

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `architect.designs_created` | Counter | Number of architectural designs created | |
| `architect.design_complexity` | Gauge | Complexity score of designs | |
| `architect.design_time` | Histogram | Time taken to create architectural designs | `complexity` |

### Security Auditor Agent

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `security_auditor.vulnerabilities_found` | Counter | Number of security vulnerabilities found | `severity` |
| `security_auditor.audit_time` | Histogram | Time taken to complete security audits | `codebase_size` |
| `security_auditor.remediation_rate` | Gauge | Percentage of vulnerabilities remediated | |

## Integration Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `integration.agent_collaboration_time` | Histogram | Time taken for multiple agents to collaborate on tasks | `task_type` |
| `integration.handoff_success_rate` | Gauge | Percentage of successful task handoffs between agents | `source_agent`, `target_agent` |
| `integration.context_sharing_size` | Histogram | Size of context information shared between agents | `source_agent`, `target_agent` |

## System Performance Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `system.agent_cpu_usage` | Gauge | CPU usage by agent processes | `agent_type` |
| `system.agent_memory_usage` | Gauge | Memory usage by agent processes | `agent_type` |
| `system.agent_throughput` | Gauge | Number of tasks processed per time unit | `agent_type` |
| `system.queue_depth` | Gauge | Number of tasks waiting in the queue | `priority` |

## Adaptive Control Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `control.pid_proportional_term` | Gauge | Current value of the proportional term in PID controllers | `controller_name` |
| `control.pid_integral_term` | Gauge | Current value of the integral term in PID controllers | `controller_name` |
| `control.pid_derivative_term` | Gauge | Current value of the derivative term in PID controllers | `controller_name` |
| `control.adaptation_rate` | Gauge | Rate of adaptation in self-tuning components | `component` |
| `control.stability_score` | Gauge | Measure of system stability during adaptation | |

## Using Agent Metrics

### Dashboards

Agent metrics can be visualized using Grafana dashboards. Pre-configured dashboards are available in the `/dashboards` directory.

### Alerts

Set up alerts based on these metrics to detect:
- Agent failures or excessive error rates
- Performance degradation
- Resource constraints
- Security vulnerabilities

### Performance Tuning

Use metrics to identify:
- Bottlenecks in agent processing
- Communication inefficiencies
- Resource optimization opportunities

### SLOs (Service Level Objectives)

Define SLOs based on:
- Agent response times
- Task completion rates
- Error rates
- Resource utilization thresholds