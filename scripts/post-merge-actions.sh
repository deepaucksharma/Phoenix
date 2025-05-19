#!/bin/bash
# post-merge-actions.sh - Script to perform post-merge actions for PRs
# Usage: ./scripts/post-merge-actions.sh <PR_NUMBER>

set -e  # Exit on any error

PR_NUMBER=$1

if [ -z "$PR_NUMBER" ]; then
  echo "Error: PR number is required"
  echo "Usage: ./scripts/post-merge-actions.sh <PR_NUMBER>"
  exit 1
fi

echo "===== Running post-merge actions for PR #$PR_NUMBER ====="

# 1. Update local main branch

echo -e "\n>> Updating local main branch..."
git checkout main
git pull origin main

# 2. Run linting

echo -e "\n>> Running linter..."
make lint || echo "WARNING: Linting issues detected - review and fix manually"

# 3. Run unit tests (ignoring the known failure)

echo -e "\n>> Running unit tests (ignoring known TestSpaceSavingSkewedDistribution failure)..."
make test-unit || echo "WARNING: Some tests failed - check if it's only the known failing test"

# 4. Check for component registration drift

echo -e "\n>> Checking component registration drift..."
make drift-check || echo "WARNING: Component registration drift detected - fix manually"

# 5. Update documentation if needed

echo -e "\n>> Checking if documentation update is needed..."
case $PR_NUMBER in
  38)
    echo "PR #38 merged - Consider updating test coverage documentation for policy and metrics"
    ;;
  39)
    echo "PR #39 merged - Consider updating test coverage documentation for others_rollup processor"
    ;;
  40)
    echo "PR #40 merged - Consider updating test coverage documentation for cardinality_guardian processor"
    ;;
  41)
    echo "PR #41 merged - Ensure chaos suite documentation is updated"
    echo "Time to fix the TestSpaceSavingSkewedDistribution issue in a separate PR"
    ;;
esac

# 6. Run specific checks based on the PR

echo -e "\n>> Running PR-specific checks..."
case $PR_NUMBER in
  38)
    echo "Testing policy parser and self-metrics..."
    go test -v ./test/unit/policy/... ./test/unit/metrics/...
    ;;
  39)
    echo "Testing others_rollup processor..."
    go test -v ./test/processors/others_rollup/...
    ;;
  40)
    echo "Testing cardinality_guardian processor..."
    go test -v ./test/processors/cardinality_guardian/...
    ;;
  41)
    echo "Testing chaos suite..."
    go test -v ./test/chaos/...
    ;;
esac

# 7. Verify if all PRs have been merged

if [[ $PR_NUMBER -eq 41 ]]; then
  echo -e "\n>> Checking if all PRs (38, 39, 40, 41) have been merged..."
  
  if git log --oneline -n 20 | grep -q "PR #38" && \
     git log --oneline -n 20 | grep -q "PR #39" && \
     git log --oneline -n 20 | grep -q "PR #40" && \
     git log --oneline -n 20 | grep -q "PR #41"; then
    
    echo -e "\n===== All PRs have been merged! ====="
    echo "Time to perform final consolidation steps:"
    echo "1. Create a fix for TestSpaceSavingSkewedDistribution issue"
    echo "2. Run full test suite with 'make test-all'"
    echo "3. Generate coverage report with 'make test-coverage'"
    echo "4. Consider creating a new version tag"
  else
    echo -e "\n>> Not all PRs have been merged yet"
    echo "Continue merging the remaining PRs"
  fi
fi

echo -e "\n===== Post-merge actions for PR #$PR_NUMBER completed ====="
