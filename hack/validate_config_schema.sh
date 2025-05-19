#!/bin/bash
# validate_config_schema.sh - Generate and validate JSON schemas from Config structs

set -e

# Create temporary directory for generated schemas
mkdir -p .tmp/config-schemas

# Generate JSON schemas from Config structs
echo "Generating JSON schemas from Config structs..."
go run hack/generate_config_schemas.go .tmp/config-schemas/

# Validate config files against schemas
echo "Validating config files against schemas..."
for config_file in $(find ./config -name "*.yaml"); do
  component_name=$(basename "$config_file" .yaml)
  schema_file=".tmp/config-schemas/${component_name}.json"
  
  if [ -f "$schema_file" ]; then
    echo "Checking $config_file against schema..."
    go run hack/validate_config.go "$config_file" "$schema_file"
    if [ $? -ne 0 ]; then
      echo "Error: Config file $config_file failed validation"
      exit 1
    fi
  else
    echo "Warning: No schema found for $component_name"
  fi
done

echo "All config files valid!"
exit 0
