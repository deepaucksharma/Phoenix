#!/bin/bash
# standardize_yaml.sh - Standardize formatting in YAML files

set -e

# Create scripts/cleanup directory if it doesn't exist

mkdir -p "$(dirname "$0")"

# Function to standardize a YAML file

standardize_yaml() {
  local file=$1
  echo "Standardizing YAML formatting in $file"
  
  # Make a backup
  cp "$file" "${file}.bak"
  
  # Fix spacing after hash comments
  sed -i 's/\(#\)\([^ ]\)/\1 \2/g' "$file"
  
  # Ensure consistent newlines at end of file (single newline)
  # First remove all trailing newlines, then add one
  sed -i -e :a -e '/^\n*$/{$d;N;ba' -e '}' "$file"
  echo "" >> "$file"
  
  echo "✓ Completed standardizing $file"
}

# Process all YAML files

find_yaml_files() {
  local dirs="$1"
  find $dirs -name "*.yaml" -o -name "*.yml" | sort
}

if [ $# -eq 0 ]; then
  # Default directories to process if none specified
  dirs="configs agents tasks deploy"
else
  dirs="$@"
fi

# Get list of files to process

files=$(find_yaml_files "$dirs")

# Process each file

for file in $files; do
  standardize_yaml "$file"
done

echo "All YAML files have been standardized."
