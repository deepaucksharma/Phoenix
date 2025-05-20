#!/bin/bash
# validate_processors.sh - Validates processor configurations in the SA-OMF system

set -e

echo "Validating processor configurations..."

# Define the processor directories to validate
PROCESSOR_DIRS=(
  "internal/processor/adaptive_pid"
  "internal/processor/adaptive_topk"
  "internal/processor/others_rollup"
  "internal/processor/priority_tagger"
)

# Validate each processor directory exists
for dir in "${PROCESSOR_DIRS[@]}"; do
  if [ ! -d "$dir" ]; then
    echo "Error: Processor directory $dir does not exist!"
    exit 1
  fi
  
  # Verify required files exist in each processor directory
  for file in "config.go" "factory.go" "processor.go"; do
    if [ ! -f "$dir/$file" ]; then
      echo "Error: Required file $file missing from processor $dir!"
      exit 1
    fi
  done
  
  echo "✅ Processor $dir is valid"
done

# Verify that removed processors do not exist
REMOVED_PROCESSORS=(
  "internal/processor/semantic_correlator"
  "internal/processor/reservoir_sampler"
)

for dir in "${REMOVED_PROCESSORS[@]}"; do
  if [ -d "$dir" ]; then
    echo "Error: Removed processor $dir still exists!"
    exit 1
  fi
  
  echo "✅ Removed processor $dir is properly removed"
done

# Validate policy files reference only existing processors
echo "Validating policy files..."
CONFIG_DIRS=(
  "configs/default"
  "configs/development"
  "configs/production"
  "configs/testing"
)

for dir in "${CONFIG_DIRS[@]}"; do
  policy_file="$dir/policy.yaml"
  if [ -f "$policy_file" ]; then
    # Check if removed processors are referenced
    for removed in "semantic_correlator" "reservoir_sampler" "cardinality_guardian"; do
      if grep -q "$removed" "$policy_file"; then
        echo "Warning: Policy file $policy_file references removed processor $removed"
      fi
    done
    
    # Check if pid_decider is still referenced (should be replaced by adaptive_pid)
    if grep -q "pid_decider" "$policy_file"; then
      echo "Warning: Policy file $policy_file still references pid_decider (should use adaptive_pid)"
    fi
  fi
done

echo "✅ Processor validation completed successfully"
exit 0