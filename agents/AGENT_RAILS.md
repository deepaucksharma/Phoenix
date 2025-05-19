# AGENT_RAILS.md - Guidelines for LLM Agents

This document defines the guidelines and constraints for autonomous agents working on the Phoenix project. By following these guidelines, agents can collaborate effectively without stepping on each other's toes.

## Agent Roles

Each agent must work within a specific role as defined in `/agents/*.yaml`. The available roles are:

- **architect**: Define high-level architecture (can only modify `docs/architecture/adr/**`)
- **planner**: Break down features into tasks (can only modify `tasks/**`)
- **implementer**: Write and test code (cannot modify `docs/architecture/adr/**` or `.github/**`)
- **reviewer**: Review code changes (cannot push commits)
- **security-auditor**: Perform security reviews (cannot modify source code)
- **doc-writer**: Update documentation (cannot modify source code)
- **devops**: Maintain CI/CD (can only modify `.github/**` and `deploy/**`)
- **integrator**: Merge PRs (cannot modify source code)

All agents must:
1. Declare their role in PRs (using `ROLE: role_name`)
2. Reference tasks they're working on (using `TASKS: task-id` or `TASKS: N/A`)
3. Respect the file permissions defined for their role. These permissions (`can_touch`, `blocked_files`, `must_touch` in `agents/*.yaml`) apply to the **entire repository** and use **glob patterns** for path matching.

## Directory Structure

Agents should understand and respect the following directory structure:

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

## Core Technical Concepts

### UpdateableProcessor Interface

The UpdateableProcessor interface is the foundation of Phoenix's adaptive capabilities:

```go
// UpdateableProcessor defines the interface for processors that can be dynamically reconfigured
type UpdateableProcessor interface {
    component.Component // Embed standard component interface

    // OnConfigPatch applies a configuration change.
    // Returns error if patch cannot be applied.
    OnConfigPatch(ctx context.Context, patch ConfigPatch) error

    // GetConfigStatus returns the current effective configuration.
    GetConfigStatus(ctx context.Context) (ConfigStatus, error)
}
```

Key implementation requirements:
- Validate incoming parameter values
- Apply changes safely, using appropriate locks
- Update internal state correctly
- Return appropriate errors for invalid patches
- Report status accurately
- Maintain thread safety

### ConfigPatch Mechanism

The ConfigPatch mechanism enables dynamic reconfiguration:

```go
// ConfigPatch defines a proposed change to a processor's configuration
type ConfigPatch struct {
    PatchID             string       // Unique ID for this patch attempt
    TargetProcessorName component.ID  // Name of the processor to update
    ParameterPath       string       // Dot-separated path to the parameter
    NewValue            any          // The new value for the parameter
    PrevValue           any          // Previous value (for rollback)
    Reason              string       // Why this patch is proposed
    Severity            string       // normal|urgent|safety
    Source              string       // pid_decider|opamp|manual
    Timestamp           int64        // When this patch was created
    TTLSeconds          int          // Time-to-live for this patch
}
```

The patch lifecycle includes:
1. Patch Creation (by adaptive_pid processor or manually)
2. Patch Validation (by pic_control extension)
3. Patch Application (via target processor's OnConfigPatch method)
4. Patch Monitoring (success/failure metrics and logging)

### PID Controller Guidelines

PID controllers should be tuned with care:

- **Proportional Term (P)**:
  - Determines immediate response strength to error
  - Higher Kp values give stronger response
  - Too high: causes oscillation; Too low: sluggish response
   
- **Integral Term (I)**:
  - Eliminates steady-state error over time
  - Higher Ki values correct steady-state error faster
  - Too high: overshoot and oscillation; Too low: slow correction
   
- **Derivative Term (D)**:
  - Dampens oscillations and improves stability
  - Higher Kd values give stronger response to error changes
  - Too high: noise sensitivity; Too low: inadequate dampening

All PID controllers should include anti-windup protection to prevent integral term accumulation when output is saturated.

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

## Component Implementation Standards

When implementing new components:

1. **Processors Must**:
   - Implement the UpdateableProcessor interface for adaptive processors
   - Extend BaseProcessor for common functionality
   - Include proper telemetry with metrics and traces
   - Implement thread-safe configuration changes
   - Handle edge cases and failure modes gracefully

2. **Processors Should Be**:
   - Stateless where possible (or handle state carefully)
   - Designed for performance and minimal resource usage
   - Well-documented with clear configuration options
   - Thoroughly tested with unit and benchmark tests

3. **Control Components Must**:
   - Validate inputs thoroughly
   - Implement safety boundaries
   - Log actions and decisions
   - Support monitoring and observability
   - Handle failure modes gracefully

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
- Use the processor_test_template.go for consistent testing structure
- Test UpdateableProcessor compliance with interfaces/updateable_processor_test.go
- For control components, use testutils/pid_helper.go to test PID behavior

## Configuration Standards

Follow these standards for configuration:

- Define configuration in a struct with yaml/json tags
- Validate configuration values during component creation
- Provide sensible defaults for all configuration parameters
- Include documentation for each configuration field
- Follow the same naming style as existing configurations

Example:
```go
// Config defines the configuration for the adaptive_topk processor
type Config struct {
    KValue         int     `mapstructure:"k_value"`
    KMin           int     `mapstructure:"k_min"`
    KMax           int     `mapstructure:"k_max"`
    CoverageTarget float64 `mapstructure:"coverage_target"`
    Enabled        bool    `mapstructure:"enabled"`
}
```

## Documentation Requirements

- All public APIs must be documented
- Major design decisions must be recorded in ADRs
- Configuration options must be documented
- Update component documentation in /docs/components/ for new features
- Include examples of configuration in documentation

## Reviewing Code

Reviewer agents should check:

1. Code meets style guidelines
2. Tests are comprehensive
3. Documentation is updated
4. Performance is acceptable
5. Security best practices are followed
6. Code matches the task specification

## Generated Files

Never edit generated files. These include:

- Go generated code (files with `// Code generated ... DO NOT EDIT.`)
- Generated protobuf code
- Build artifacts

## Branch Naming

Use consistent branch naming:

- `feature/description` for new features
- `fix/description` for bug fixes
- `refactor/description` for refactoring
- `docs/description` for documentation changes

## Commit Messages

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

## Getting Help

If you're unsure about something:
- Refer to existing code as examples
- Check the ADRs
- Read the CONSOLIDATED_AGENTS.md file
- Look at implementation notes in component READMEs
- Ask for clarification in the PR description