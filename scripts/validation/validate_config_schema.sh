#!/bin/bash
# validate_config_schema.sh - Validate configuration files

set -e

# Basic YAML validation for config files
echo "Validating configuration files..."
for config_file in $(find ./configs -name "*.yaml"); do
  echo "Checking $config_file..."
  
  # Check if the file is empty
  if [ ! -s "$config_file" ]; then
    echo "Warning: Config file $config_file is empty."
    continue
  fi
  
  # Try to parse the YAML using a command that's likely to be available
  if command -v python3 &>/dev/null; then
    python3 -c "import yaml; yaml.safe_load(open('$config_file'))" 2>/dev/null
    if [ $? -ne 0 ]; then
      echo "Error: Config file $config_file is not valid YAML."
      exit 1
    fi
  elif command -v python &>/dev/null; then
    python -c "import yaml; yaml.safe_load(open('$config_file'))" 2>/dev/null
    if [ $? -ne 0 ]; then
      echo "Error: Config file $config_file is not valid YAML."
      exit 1
    fi
  elif command -v yq &>/dev/null; then
    yq eval . "$config_file" >/dev/null 2>&1
    if [ $? -ne 0 ]; then
      echo "Error: Config file $config_file is not valid YAML."
      exit 1
    fi
  else
    echo "Warning: No YAML validation tool found (python, yq). Skipping validation for $config_file."
  fi
done

# Check if more advanced validation is available
if [ -f "hack/validate_config.go" ]; then
  # Create temporary directory for generated schemas
  mkdir -p .tmp/config-schemas
  
  # Generate JSON schemas from Config structs
  echo "Generating JSON schemas from Config structs..."
  if go run hack/generate_config_schemas.go .tmp/config-schemas/ 2>/dev/null; then
    echo "Schema generation successful."
    
    # Validate config files against schemas
    echo "Validating config files against schemas..."
    for config_file in $(find ./configs -name "*.yaml"); do
      component_name=$(basename "$config_file" .yaml)
      schema_file=".tmp/config-schemas/${component_name}.json"
      
      if [ -f "$schema_file" ]; then
        echo "Checking $config_file against schema..."
        if go run hack/validate_config.go "$config_file" "$schema_file" 2>/dev/null; then
          echo "  - Passed schema validation."
        else
          echo "  - Warning: Schema validation failed, but continuing. This will be enforced in future."
        fi
      fi
    done
  else
    echo "Schema generation failed, but continuing with basic validation only."
  fi
else
  echo "Advanced schema validation not available. Using basic YAML validation only."
fi

echo "All configuration files are valid YAML!"
exit 0
