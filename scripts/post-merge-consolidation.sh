#!/bin/bash
# post-merge-consolidation.sh - Script to perform final consolidation steps after all PRs are merged
# Usage: ./scripts/post-merge-consolidation.sh

set -e  # Exit on any error

echo "===== Running final consolidation steps after all PRs are merged ====="

# 1. Update local main branch

echo -e "\n>> Updating local main branch..."
git checkout main
git pull origin main

# 2. Verify all PRs are merged

echo -e "\n>> Verifying all PRs (38, 39, 40, 41) have been merged..."
if git log --oneline -n 20 | grep -q "PR #38" && \
   git log --oneline -n 20 | grep -q "PR #39" && \
   git log --oneline -n 20 | grep -q "PR #40" && \
   git log --oneline -n 20 | grep -q "PR #41"; then
  echo "All PRs confirmed merged"
else
  echo "ERROR: Not all PRs have been merged yet. Exiting."
  exit 1
fi

# 3. Create a fix branch for the TestSpaceSavingSkewedDistribution issue

echo -e "\n>> Creating a fix branch for the TestSpaceSavingSkewedDistribution issue..."
git checkout -b fix/topk-space-saving-test-failures

echo -e "\n>> You should now edit the test/unit/topk/space_saving_test.go file to fix the issue"
echo "Once the fix is complete, run the following commands:"
echo "git add ."
echo "git commit -m \"fix: resolve TestSpaceSavingSkewedDistribution failure in topk package\""
echo "git push origin fix/topk-space-saving-test-failures"
echo "gh pr create --title \"Fix TestSpaceSavingSkewedDistribution failure\" --body \"Resolves test failures in the topk package mentioned in PRs #38, #39, #40.\""

read -p "Press Enter when you have fixed the issue and are ready to continue..."

# 4. Run the tests to verify the fix

echo -e "\n>> Running the topk tests to verify the fix..."
go test -v ./pkg/util/topk/...

# 5. Go back to main for remaining steps

echo -e "\n>> Switching back to main branch for remaining steps..."
git checkout main

# 6. List actions to run full test suite

echo -e "\n>> To run the full test suite, execute:"
echo "make test-all"

# 7. Generate coverage report

echo -e "\n>> To generate a coverage report, execute:"
echo "make test-coverage"

# 8. Suggest version tagging

echo -e "\n>> If appropriate, to create a new version tag, execute:"
echo "make release VERSION=x.y.z"

echo -e "\n===== Final consolidation steps completed ====="
echo "Don't forget to create a PR for the test fix branch and merge it"
