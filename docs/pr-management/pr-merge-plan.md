# Pull Request Merge Plan

This document outlines the plan for merging open PRs, including a prioritized merge order and post-merge actions to ensure codebase stability.

## PR Summary

We have 4 open PRs, all adding test coverage to different parts of the codebase:

1. **PR #38**: Add unit tests for policy and metrics
   - New unit tests for policy parser and self-metrics emitter
   - Files: `test/unit/metrics/selfmetrics_test.go`, `test/unit/policy/policy_test.go`

2. **PR #39**: Add others_rollup processor tests
   - New processor tests for others_rollup
   - Files: `test/processors/others_rollup/processor_test.go`

3. **PR #40**: Add cardinality_guardian processor tests
   - New tests for cardinality_guardian processor
   - Files: `test/processors/cardinality_guardian/processor_test.go`

4. **PR #41**: Add chaos suite scenarios
   - Implements chaos test scenarios
   - Files: `docs/testing/README.md`, `test/chaos/chaos_suite.go`

## Common Issue

All PRs mention a test failure in `TestSpaceSavingSkewedDistribution` in the `topk` package, which appears to be a pre-existing issue in the main branch.

## Merge Priority

Based on the scope and complexity, we recommend the following merge order:

1. **PR #38**: Unit tests for policy and metrics (foundation for other test coverage)
2. **PR #39**: Others_rollup processor tests
3. **PR #40**: Cardinality_guardian processor tests 
4. **PR #41**: Chaos suite scenarios (most complex and builds on other test coverage)

## Merge Process for Each PR

For each PR, follow these steps:

### Pre-Merge Actions

1. Verify the PR passes all required checks (except for the known `TestSpaceSavingSkewedDistribution` failure)
2. Ensure the PR has adequate review coverage
3. Check for any merge conflicts with the main branch and resolve if needed

### Merge Actions

1. Use GitHub's "Merge pull request" feature with "Squash and merge" option
2. Use the PR title as the commit title
3. Include the PR description in the commit message
4. Add a reference to the PR number in the commit message

### Post-Merge Actions for Each PR

After each PR is merged, perform these actions:

1. **Code Cleanup**:
   - Run `make lint` to check for any new linting issues
   - Run `make test-unit` to verify tests (ignoring the known failure)
   - Run `make drift-check` to ensure component registration is correct

2. **Documentation Updates**:
   - Update test coverage documentation if needed
   - Confirm any related documentation is consistent

3. **Fix Known Issues**:
   - For PR #38-40: Don't address the `TestSpaceSavingSkewedDistribution` issue yet
   - After PR #41: Create a separate fix for the `TestSpaceSavingSkewedDistribution` issue

## Final Consolidation Steps

After all PRs are merged, perform these steps:

1. **Fix Test Failures**:
   ```bash
   # Create a branch for fixing the test issue
   git checkout -b fix/topk-space-saving-test-failures
   
   # Investigate and fix the test
   # Edit the test/unit/topk/space_saving_test.go file
   
   # Commit and push the changes
   git add .
   git commit -m "fix: resolve TestSpaceSavingSkewedDistribution failure in topk package"
   git push origin fix/topk-space-saving-test-failures
   
   # Create a PR
   gh pr create --title "Fix TestSpaceSavingSkewedDistribution failure" --body "Resolves test failures in the topk package mentioned in PRs #38, #39, #40."
   ```

2. **Run Full Test Suite**:
   ```bash
   # On the main branch after all PRs are merged
   git checkout main
   git pull
   
   # Run all tests
   make test-all
   
   # Generate a coverage report
   make test-coverage
   ```

3. **Update Documentation**:
   ```bash
   # Update the test coverage section in documentation
   # Edit the relevant documentation files to reflect new test coverage
   ```

4. **Tag a New Version**:
   ```bash
   # If appropriate, create a new version tag
   make release VERSION=0.x.y
   ```

## Monitoring and Verification

After completing all merges and consolidation steps:

1. Verify CI pipeline is passing on the main branch
2. Confirm test coverage has increased
3. Run the chaos suite tests to ensure system stability
4. Update project board to reflect completed work

## Rollback Plan

If issues arise after merging, prepare to implement these rollback steps:

1. Identify which PR introduced the issue
2. If needed, temporarily revert the problematic PR
3. Fix the issue in a separate branch
4. Re-apply the changes with the fix

This controlled approach ensures systematic and stable integration of the new test coverage into the codebase.