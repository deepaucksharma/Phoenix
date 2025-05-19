#!/bin/bash
# format_dashboards.sh - Format and validate JSON dashboard files
#
# This script formats JSON dashboard files for consistent formatting
# and adds metadata to track version and modification dates.

set -e

# Create scripts/cleanup directory if it doesn't exist
mkdir -p "$(dirname "$0")"

# Function to format a JSON dashboard file
format_dashboard() {
  local file=$1
  echo "Formatting dashboard file: $file"
  
  # Make a backup
  cp "$file" "${file}.bak"
  
  # Use jq to format the JSON
  if command -v jq &> /dev/null; then
    # First, read the entire file as JSON
    local json=$(cat "$file")
    
    # Add or update metadata fields
    json=$(echo "$json" | jq \
      --arg version "$(date +%Y%m%d.%H%M%S)" \
      --arg lastModified "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
      '. + {
        "meta": {
          "formatVersion": 1,
          "lastModified": $lastModified,
          "version": $version
        }
      }')
    
    # Write the formatted JSON back to the file
    echo "$json" | jq . > "$file"
    echo "✓ Formatted and added metadata to $file"
  else
    echo "Warning: jq is not installed. Cannot format JSON."
    echo "Please install jq for JSON formatting."
  fi
}

# Process dashboard files
find_dashboard_files() {
  find dashboards -name "*.json" | sort
}

# Get list of files to process
files=$(find_dashboard_files)

# Process each file
if [ -z "$files" ]; then
  echo "No dashboard files found in the dashboards directory."
else
  for file in $files; do
    format_dashboard "$file"
  done
  echo "All dashboard files have been formatted."
fi

# Create a README.md for the dashboards directory if it doesn't exist
if [ ! -f "dashboards/README.md" ]; then
  echo "Creating dashboards/README.md..."
  cat > "dashboards/README.md" << EOF
# Phoenix Dashboards

This directory contains Grafana dashboard definitions for monitoring the Phoenix (SA-OMF) system.

## Available Dashboards

| Dashboard | Description | Tags |
|-----------|-------------|------|
| [autonomy-pulse.json](./autonomy-pulse.json) | Monitor autonomy level and self-adaptive behavior | sa-omf, phoenix |

## Usage

These dashboards can be imported into Grafana using the JSON file or by referencing the file URL.

### Local Development

1. Start Grafana locally using the provided docker-compose file:
   \`\`\`
   docker-compose -f deploy/compose/full/docker-compose.yaml up -d
   \`\`\`

2. Access Grafana at http://localhost:3000
   - Default credentials: admin/admin

3. Import the dashboard(s) from the JSON files in this directory

### Production Deployment

When deploying to production, consider using the Grafana provisioning feature to automatically 
load these dashboards. See the [Grafana documentation](https://grafana.com/docs/grafana/latest/administration/provisioning/#dashboards) for details.
EOF
  echo "✓ Created dashboards/README.md"
fi