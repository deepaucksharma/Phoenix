#!/bin/bash
# standardize_scripts.sh - Standardize shell script headers and comments
#
# This script ensures all shell scripts in the repository follow consistent
# format with proper headers, comments, and error handling.

set -e

# Create scripts/cleanup directory if it doesn't exist

mkdir -p "$(dirname "$0")"

# Function to standardize a shell script

standardize_script() {
  local file=$1
  echo "Standardizing shell script: $file"
  
  # Make a backup
  cp "$file" "${file}.bak"
  
  # Check if the file has a proper shebang
  if ! grep -q "^#!/bin/bash" "$file"; then
    # Add shebang if missing
    sed -i '1s/^/#!/bin/bash\n/' "$file"
    echo "  Added shebang to $file"
  fi
  
  # Check if the file has a description comment after shebang
  if ! grep -q "^#.*-.*" "$file" && ! grep -q "^# *[A-Z]" "$file" ; then
    # Extract filename without path and extension
    local filename=$(basename "$file" .sh)
    # Add description comment if missing
    sed -i "2s/^/# $filename - Script description\n/" "$file"
    echo "  Added description comment to $file"
  fi
  
  # Check if script has set -e for error handling
  if ! grep -q "set -e" "$file"; then
    # Add set -e if missing, after the comments section
    line_num=$(grep -n "^#" "$file" | tail -1 | cut -d: -f1)
    sed -i "$((line_num+1))i\\
set -e\\
" "$file"
    echo "  Added error handling to $file"
  fi
  
  # Ensure script has proper spacing
  # Add empty line after comment block if missing
  sed -i '/^# /!b;:a;n;/^$/b;/^[^#]/i\\' "$file"
  
  # Ensure file ends with a newline
  sed -i -e '$a\' "$file"
  
  echo "✓ Completed standardizing $file"
}

# Process all shell scripts

find_scripts() {
  local dirs="$1"
  find $dirs -name "*.sh" | sort
}

if [ $# -eq 0 ]; then
  # Default directories to process if none specified
  dirs="scripts"
else
  dirs="$@"
fi

# Get list of files to process

files=$(find_scripts "$dirs")

# Process each file

for file in $files; do
  # Skip the current script to avoid modifying itself
  if [[ "$file" != "$0" ]]; then
    standardize_script "$file"
  fi
done

echo "All shell scripts have been standardized."
