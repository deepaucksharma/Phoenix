# Phoenix Agent Framework

This document provides a comprehensive guide to the agent roles, responsibilities, and workflows used in the Phoenix project. This framework enables efficient collaboration between both autonomous agents and human contributors.

## Agent Roles and Responsibilities

| Role ID | Purpose | Responsibilities | Outputs | Permissions | Boundaries |
|---------|---------|------------------|---------|------------|------------|
| **Architect** | Define system architecture | High-level design decisions, architecture planning | ADRs, architecture diagrams | Can modify architecture docs | Focus on system design, not implementation details |
| **Planner** | Break down features into tasks | Issue triage, task decomposition | `tasks/*.yaml` files, roadmap updates | Can modify task files | Does not write production code |
| **Implementer** | Write and test code | Code implementation, unit testing | Code, tests | Can modify code and tests | Works on assigned tasks only, cannot modify ADRs |
| **Reviewer** | Ensure code quality | Code review, validation | Review comments, approval/rejection | Cannot push commits (except minor fixes) | Focus on code quality assessment |
| **Security-Auditor** | Identify security issues | Security scanning, vulnerability assessment | Security reports | Cannot modify source code | Limited to security analysis |
| **Doc-Writer** | Create/update documentation | Technical writing, API documentation | Markdown files, diagrams | Limited to documentation files | Cannot modify source code |
| **DevOps** | CI/CD, deployment | Pipeline maintenance, deployment scripts | Workflow files, Dockerfiles | Limited to CI/CD files | Focused on operational aspects |
| **Integrator** | Merge PRs, resolve conflicts | Conflict resolution, release management | Merged PRs, release notes | Can merge PRs | Only merges approved PRs |
| **Tester** | Create and maintain tests | Test development, validation framework | Test files, quality reports | Limited to test files | Cannot modify core implementation |
| **Dependabot** | Dependency management | Update dependencies | PR for dependency updates | Limited to package files | Automated dependency updates only |

## File Permission Boundaries

Each agent role has specific file permissions defined in their respective YAML files:

- **can_touch**: Files that the agent is allowed to modify
- **blocked_files**: Files that the agent is not allowed to modify
- **must_touch**: Files that the agent must modify as part of their work

These permissions use glob patterns and apply project-wide. For example:

```yaml
# Implementer permissions
can_touch:
  - cmd/**
  - internal/**
  - pkg/**
  - test/**
must_touch:
  - test/**
blocked_files:
  - agents/**
  - .github/workflows/**
  - docs/adr/**
```

## Workflow and Collaboration

### Development Process

1. **Creation Phase**:
   - **Architect** defines the architecture or changes through ADRs
   - **Planner** breaks down features into tasks and updates the task list

2. **Implementation Phase**:
   - **Implementer** works on assigned tasks
   - **Tester** creates or updates related tests
   - **Reviewer** validates the implementation
   - **Security-Auditor** checks for vulnerabilities

3. **Documentation & Integration Phase**:
   - **Doc-Writer** updates documentation
   - **DevOps** ensures CI/CD works correctly
   - **Integrator** merges approved PRs

### PR Process

All contributors (human or agent) must:

1. Declare their role in PRs (using `ROLE: role_name`)
2. Reference tasks they're working on (using `TASKS: task-id` or `TASKS: N/A`)
3. Respect the file permissions defined for their role
4. Ensure all required checks pass before merging

### Branch Naming Conventions

Use consistent branch naming:

- `feature/description` for new features
- `fix/description` for bug fixes
- `refactor/description` for refactoring
- `docs/description` for documentation changes

### Commit Message Standards

Follow the Conventional Commits specification:

```
type(scope): description

[optional body]

[optional footer]
```

Where `type` is one of:
- feat: A new feature
- fix: A bug fix
- docs: Documentation changes
- style: Code style changes (formatting, etc.)
- refactor: Code refactoring
- test: Adding or modifying tests
- chore: Routine tasks, maintenance

## Agent Collaboration Workflows

The Phoenix project relies on well-defined collaboration workflows between different agent roles. This section describes key workflows and interaction patterns.

### PID Control Loop Workflow

The PID Control Loop is a central mechanism in the Phoenix architecture that enables self-regulation. Here's how agents collaborate throughout its lifecycle:

#### 1. Design Phase
- **Architect**:
  - Defines Key Performance Indicators (KPIs) that the system will regulate
  - Establishes target values for each KPI
  - Designs the feedback control topology
  - Determines sensor points for measuring actual values
  - Creates ADRs documenting the control architecture

- **Planner**:
  - Breaks down the control loop into implementable components
  - Creates detailed task specifications for each component
  - Establishes dependencies between tasks
  - Defines integration points between components

#### 2. Implementation Phase
- **Implementer**:
  - Builds the PID controller with proper anti-windup mechanisms
  - Implements sensor components for measuring KPIs
  - Creates the ConfigPatch generation mechanism
  - Ensures thread safety across all control components
  - Implements limits and constraints for safe operation

- **Integrator**:
  - Connects the PID controller with sensor inputs
  - Establishes pathways for ConfigPatch delivery
  - Ensures control feedback properly reaches target processors
  - Validates that the control pipeline operates independently of the data pipeline

#### 3. Testing Phase
- **Tester**:
  - Creates test scenarios for different control conditions
  - Verifies controller behavior under varying inputs
  - Tests stability under perturbation
  - Validates recovery from extreme conditions
  - Develops specialized testing tools for control systems

- **Security Auditor**:
  - Verifies that the control system cannot be exploited
  - Ensures limits are properly enforced
  - Validates that the control system fails safely
  - Checks for potential oscillation or runaway scenarios

#### 4. Operational Phase
- **DevOps**:
  - Monitors controller performance in production
  - Creates dashboards for observing control metrics
  - Sets up alerts for control loop issues
  - Establishes procedures for tuning controllers in production

- **Reviewer**:
  - Periodically reviews controller performance
  - Evaluates tuning parameters for optimality
  - Suggests adjustments to control parameters
  - Validates that control objectives are being met

### Task Propagation Chain

Tasks in the Phoenix project follow a structured propagation path from conception to completion, with clear handoffs between agent roles:

#### 1. Architecture Decision Records (ADRs)
- **Architect** creates ADRs that establish fundamental design decisions
- ADRs include rationale, alternatives considered, and consequences
- Each ADR identifies affected components and necessary changes

#### 2. Task Specification
- **Planner** translates ADRs into specific implementation tasks
- Each task includes acceptance criteria, dependencies, and estimated effort
- Tasks are assigned to specific agent roles based on expertise

#### 3. Implementation
- **Implementer** or appropriate role executes the task
- Implementation follows the specifications in the task
- Changes are made with careful attention to interfaces and dependencies
- Code reviews ensure implementation meets requirements

#### 4. Testing and Validation
- **Tester** verifies that implementation meets acceptance criteria
- **Reviewer** evaluates the implementation for quality and correctness
- **Security Auditor** checks for security implications
- Sign-offs from multiple roles may be required for critical components

#### 5. Documentation Update
- **Doc Writer** updates component documentation
- Implementation details are captured for future reference
- Examples and usage patterns are documented
- Any changes to interfaces or behavior are clearly noted

### Config Patch Lifecycle

The ConfigPatch mechanism enables dynamic reconfiguration of the system. Here's how these patches flow through the system:

#### 1. Patch Generation
- **Adaptive PID Processor** monitors KPIs and generates patches
- Patches include specific parameter changes to optimize performance
- Each patch has a unique ID, target processor, parameter path, and new value
- Patches also include metadata like reason, severity, and source

#### 2. Patch Validation
- **PIC Control Extension** receives patches and validates them
- Validation includes checking parameter bounds, rate limiting, and safety constraints
- Patches that violate policy constraints are rejected
- Validation errors are logged for analysis

#### 3. Patch Application
- **PIC Control Extension** routes validated patches to target processors
- Target processors implement the UpdateableProcessor interface
- Processors apply the patch if possible, or reject with error
- Successful application is confirmed back to the extension

#### 4. Patch Monitoring
- **Metrics subsystem** tracks patch application success/failure
- Dashboards show patch frequency, types, and outcomes
- Alerts trigger on excessive patch rates or high failure rates
- Historical patch data aids in system tuning

### Dual Pipeline Touchpoints

The Phoenix architecture uses a dual-pipeline approach (data and control). Here are the key touchpoints where these pipelines interact and require agent collaboration:

#### 1. Processor Registration
- **PIC Control Extension** registers UpdateableProcessor instances
- Processors must register to receive configuration updates
- Registration establishes the connection between control and data pipelines
- Agents must ensure proper registration during initialization

#### 2. Metrics Exchange
- **Data Pipeline** produces metrics about its operation
- **Control Pipeline** consumes these metrics to make decisions
- Metrics must be properly exposed and collected
- Agents ensure metrics are meaningful and accurate

#### 3. Configuration Updates
- **Control Pipeline** generates ConfigPatch objects
- These patches are applied to processors in the Data Pipeline
- Proper serialization and deserialization is crucial
- Agents verify correct patch propagation

#### 4. Safety Interactions
- **Safety Monitor** observes both pipelines
- Safety conditions can affect both pipelines
- Safe mode activation changes behavior in both pipelines
- Agents ensure consistent safety responses

## Decision Framework

This section establishes the decision-making framework for different agent roles in the Phoenix project. It provides guidelines for making critical decisions about system parameters, tuning, safety, testing, and performance tradeoffs.

### Parameter Boundary Decisions

Parameter boundaries are crucial for ensuring system stability and preventing harmful configurations. The following guidelines apply to setting min/max constraints:

#### Decision Authority
- **Architect**: Establishes boundary guidelines and approves exceptions
- **Implementer**: Proposes specific boundaries based on implementation knowledge
- **Security Auditor**: Reviews boundaries for security implications
- **Performance Engineer**: Validates boundaries against performance goals

#### Boundary Setting Methodology

##### 1. Default Parameter Ranges
Default parameter ranges should be established based on the following principles:

| Parameter Type | Lower Bound | Upper Bound | Example |
|----------------|-------------|-------------|---------|
| Count/Size | Minimum functional value | 2-5x expected maximum | KValue: 10-60 |
| Time Intervals | Response time + buffer | Max acceptable latency | PatchCooldownSeconds: 10-60 |
| Ratios/Percentages | Minimum effective value | 100% or theoretical max | CoverageTarget: 0.7-0.99 |
| Control Gains | 0 or minimum stable value | Maximum stable value | Kp: 0-100 |

##### 2. Boundary Validation Requirements
Each parameter boundary must be:
- **Tested**: Verified at both extremes and boundary edges
- **Documented**: Clearly specified in component documentation
- **Justified**: Based on theoretical analysis or empirical data
- **Reviewed**: Approved by appropriate roles (Architect, Security Auditor)

##### 3. Boundary Adjustment Process
To change established parameter boundaries:
1. Present evidence that current boundaries are inadequate
2. Provide testing data for proposed new boundaries
3. Assess impact on system stability and performance
4. Obtain approval from Architect and relevant experts
5. Update code, tests, and documentation

### PID Controller Tuning Authority

PID controllers are sensitive to tuning parameters. This section defines who can adjust these parameters and under what circumstances.

#### Tuning Parameter Authority Matrix

| Parameter | Architect | Implementer | Performance Engineer | DevOps | Planner |
|-----------|-----------|-------------|----------------------|--------|---------|
| Kp (Proportional) | Approve | Recommend | Test | Monitor | Document |
| Ki (Integral) | Approve | Recommend | Test | Monitor | Document |
| Kd (Derivative) | Approve | Recommend | Test | Monitor | Document |
| Setpoint | Define | Implement | Validate | Adjust* | Document |
| Integral Limit | Review | Define | Validate | Monitor | Document |
| Anti-windup Settings | Review | Define | Validate | Monitor | Document |

*DevOps may adjust setpoints within pre-approved ranges in production

#### PID Controller Tuning Process

1. **Initial Tuning**:
   - Performance Engineer proposes initial values based on system modeling
   - Implementer implements the controller with these values
   - Tester verifies stability and performance
   - Architect approves the initial tuning

2. **Tuning Adjustments**:
   - Performance Engineer analyzes controller behavior and recommends changes
   - Implementer implements the changes
   - Tester verifies improvement
   - Architect approves significant changes

3. **Emergency Tuning**:
   - DevOps may make emergency adjustments within pre-approved ranges
   - Changes must be documented and reviewed afterward
   - Permanent changes require normal approval process

#### Tuning Guidelines

- **Proportional Gain (Kp)**: 
  - Start low and increase until responsive but not oscillating
  - Should be high enough to provide meaningful corrections
  - Too high: causes oscillation; Too low: sluggish response

- **Integral Gain (Ki)**:
  - Start very low (0.1 × Kp) and increase gradually
  - Should eliminate steady-state error over time
  - Too high: causes overshoot and oscillation; Too low: slow correction

- **Derivative Gain (Kd)**:
  - Start at zero and increase cautiously
  - Should dampen oscillations and improve stability
  - Too high: amplifies noise; Too low: insufficient dampening

- **Anti-windup Parameters**:
  - Integral limits should be based on expected output range
  - Anti-windup gain typically 0.5-1.0 × Ki
  - Always enable anti-windup for production controllers

### Safety Threshold Determinations

Safety thresholds protect the system from excessive resource usage and ensure operational stability. This section provides a framework for determining these thresholds.

#### Threshold Determination Process

1. **Resource Profiling**:
   - Measure baseline resource usage under normal conditions
   - Determine peak resource usage under expected load
   - Identify resource usage patterns over time

2. **Threshold Setting**:
   - Set warning threshold at 80% of maximum acceptable usage
   - Set critical threshold at 90% of maximum acceptable usage
   - Set automatic intervention threshold at 95% of maximum acceptable usage

3. **Validation**:
   - Test system behavior at and above thresholds
   - Verify monitoring and alerting functions
   - Confirm intervention mechanisms work correctly

4. **Review and Approval**:
   - Security Auditor reviews thresholds for security implications
   - Performance Engineer validates performance impact
   - Architect approves final thresholds

#### Safety Threshold Categories

##### Resource Usage Thresholds

| Resource | Warning | Critical | Intervention | Responsible |
|----------|---------|----------|--------------|-------------|
| CPU | 80% | 90% | 95% | Performance Engineer |
| Memory | 80% | 90% | 95% | Performance Engineer |
| Disk Space | 75% | 85% | 90% | DevOps |
| Network Bandwidth | 70% | 85% | 95% | DevOps |
| File Descriptors | 70% | 85% | 95% | Implementer |

##### Operational Thresholds

| Metric | Warning | Critical | Intervention | Responsible |
|--------|---------|----------|--------------|-------------|
| Config Patch Rate | 5/min | 10/min | 15/min | Security Auditor |
| Error Rate | 1% | 5% | 10% | Implementer |
| Latency | 2x baseline | 5x baseline | 10x baseline | Performance Engineer |
| Pipeline Backpressure | 50% capacity | 75% capacity | 90% capacity | Implementer |
| Cardinality | 80% limit | 90% limit | 95% limit | Performance Engineer |

### Testing Depth Decisions

Different components require different levels of testing depth based on their criticality, complexity, and impact. This section provides criteria for determining required testing thoroughness.

#### Component Criticality Assessment

Components are categorized into four criticality levels:

##### Level 1: Critical Infrastructure
- Components that affect system stability
- Components with security implications
- Components that cannot be easily bypassed
- Examples: PIC Control Extension, Safety Monitor, ConfigPatch Validator

##### Level 2: Core Functionality
- Components that implement key features
- Components with complex algorithms
- Components with significant performance impact
- Examples: Adaptive PID, Adaptive TopK, PID Controller

##### Level 3: Supporting Components
- Components that enhance functionality
- Components that can be bypassed if necessary
- Components with moderate complexity
- Examples: Priority Tagger, Others Rollup, Reservoir Sampler

##### Level 4: Utility Components
- Helper functions and algorithms
- Components with well-defined interfaces
- Components with limited scope
- Examples: Utility algorithms, formatting helpers

#### Testing Requirements by Criticality Level

| Aspect | Level 1 (Critical) | Level 2 (Core) | Level 3 (Supporting) | Level 4 (Utility) |
|--------|-------------------|----------------|----------------------|-------------------|
| Unit Test Coverage | 95%+ | 90%+ | 80%+ | 70%+ |
| Integration Tests | Comprehensive | Multiple scenarios | Key interactions | Basic verification |
| Performance Tests | Thorough benchmarks | Key operations | Hot paths | If performance-critical |
| Security Tests | Penetration testing | Threat modeling | Basic security review | - |
| Chaos Testing | Required | Recommended | - | - |
| Documentation | Comprehensive | Detailed | Standard | Basic |
| Code Review | 2+ reviewers | 2 reviewers | 1 reviewer | 1 reviewer |

### Cardinality vs. Coverage Tradeoffs

The Phoenix system must balance data completeness (coverage) against performance (cardinality management). This section provides a decision framework for making these tradeoffs.

#### Key Concepts

- **Cardinality**: The number of unique combinations of labels/dimensions
- **Coverage**: The percentage of the total data represented after filtering
- **Completeness**: How well the data represents the full dataset
- **Performance**: System resource usage and processing speed

#### Tradeoff Decision Matrix

| Scenario | Coverage Target | Cardinality Limit | Decision Authority |
|----------|----------------|-------------------|-------------------|
| Default Production | 90% | Based on resources | Architect |
| High-Reliability Mode | 95%+ | Higher resource allocation | Performance Engineer |
| Resource-Constrained | 80-85% | Strict limits | DevOps |
| Debugging/Analysis | 99%+ | Temporary high limits | Implementer/DevOps |

## Technical Standards

- Go version: 1.22+
- Code style: Follow `golangci-lint` rules
- Test coverage: Maintain above 60%
- Documentation: All public APIs must be documented
- PR process: Create branch → Implement → CI checks → Review → Merge

## Repository Structure

Agents should understand and respect the following repository structure:

```
phoenix/
├── cmd/                 # Entry points for binaries
│   └── sa-omf-otelcol/  # Main collector binary
├── internal/            # Private implementation code
│   ├── processor/       # Data processors
│   ├── extension/       # Collector extensions
│   ├── connector/       # Connectors/exporters
│   ├── control/         # Control algorithms
│   └── interfaces/      # Core interfaces
├── pkg/                 # Reusable packages
│   ├── metrics/         # Metrics definitions
│   ├── util/            # Utility algorithms
│   └── policy/          # Policy schema
├── test/                # Test framework
├── docs/                # Documentation
│   └── architecture/    # Architecture documentation
│     └── adr/           # Architecture Decision Records
├── scripts/             # Development scripts
├── deploy/              # Deployment files
├── tasks/               # Task definitions
└── agents/              # Agent role definitions
```

## Adding New Components

To add a new processor, extension, or connector:

1. Call the appropriate script: 
   ```
   scripts/dev/new-component.sh processor example_processor
   ```

2. This will:
   - Create appropriate files with boilerplate
   - Register the factory in main.go
   - Create test files

3. Implement the required functionality in the generated files.

## Naming Conventions

- Use snake_case for package names and file names
- Use CamelCase for types and functions
- Use ALL_CAPS for constants
- Prefix metrics with `aemf_`
- Prefix component names with their category (e.g., `pic_control`)

## Testing Requirements

- All code must have unit tests
- Processors must have benchmarks
- Integration tests must be added for new features
- Coverage should be maintained above 60%

## Documentation Requirements

- All public APIs must be documented
- Major design decisions must be recorded in ADRs
- Configuration options must be documented

## Metrics and KPIs

The team's progress is tracked using these metrics:
- PR cycle time (creation to merge)
- Test coverage percentage
- Open issues count
- Documentation completeness

## Tooling

Use provided scripts in the `scripts` directory:
- `scripts/dev/new-component.sh` - Create a new component with proper boilerplate
- `scripts/dev/new-adr.sh` - Create a new ADR
- `scripts/dev/create-branch.sh` - Create a new branch with proper naming
- `scripts/dev/create-task.sh` - Create a new task
- `scripts/dev/validate-task.sh` - Validate a task specification

## Generated Files

Never edit generated files. These include:

- Go generated code (files with `// Code generated ... DO NOT EDIT.`)
- Generated protobuf code
- Build artifacts

## Getting Help

If you're unsure about something:
- Refer to existing code as examples
- Check the ADRs
- Consult the Phoenix project documentation
- Ask for clarification in PR descriptions