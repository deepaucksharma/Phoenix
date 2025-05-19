# SA-OMF Scripts

This directory contains scripts that support development, CI/CD, and validation for the Self-Aware OpenTelemetry Metrics Fabric.

## Directory Structure

- **ci/**: Continuous Integration scripts
  - **check_component_registry.sh**: Verifies that all components are properly registered in main.go
- **dev/**: Development workflow scripts
  - **create-branch.sh**: Creates a new git branch with proper naming
  - **create-task.sh**: Creates a new task file
  - **new-component.sh**: Scaffolds a new component (processor, extension, etc.)
  - **new-adr.sh**: Creates a new Architecture Decision Record
  - **validate-adr.sh**: Validates ADR formatting
- **validation/**: Configuration validation scripts
  - **validate_policy_schema.sh**: Validates policy.yaml files against schema
  - **validate_config_schema.sh**: Validates config.yaml files against schema
  - **validate_policy.go**: CLI tool to validate a policy file
  - **validate_config.go**: CLI tool to validate a config file against a schema
  - **generate_config_schemas.go**: Generates JSON schemas for component configs

## Using the Scripts

Most scripts are self-documenting and will show usage instructions when run without arguments:

```bash
# Example usage
./scripts/dev/new-component.sh processor my_processor
./scripts/dev/new-adr.sh "Use PID Controllers for Adaptive Processing"
./scripts/validation/validate_policy_schema.sh configs/default/policy.yaml
# Validate a single policy file
go run scripts/validation/validate_policy.go configs/default/policy.yaml
# Generate schemas and validate configs
go run scripts/validation/generate_config_schemas.go .tmp/schemas
go run scripts/validation/validate_config.go configs/default/config.yaml .tmp/schemas/default.json
```

## CI Integration

The CI workflow uses these scripts to validate the code and configuration. See the `.github/workflows/ci.yml` file for details on how these scripts are integrated into the CI pipeline.