# Pull Request

## 1. Declared Agent Role
<!-- One of: architect | planner | implementer | reviewer | security-auditor | doc-writer | devops | integrator | tester -->
<!-- IMPORTANT: This must be in the format `ROLE: role_name` (include the backticks) -->
`ROLE: `

## 2. Linked Task(s)
<!-- List task IDs from tasks/*.yaml or put N/A -->
<!-- IMPORTANT: This must be in the format `TASKS: id1, id2` or `TASKS: N/A` (include the backticks) -->
`TASKS: `

## 3. Changes Summary
<!-- Provide a clear and concise description of the changes -->

## 4. Motivation and Context
<!-- Why is this change required? What problem does it solve? -->

## 5. Related Issue
<!-- Link to the related issue(s) if applicable -->

## 6. How Has This Been Tested?
<!-- Please describe how you tested your changes -->

- [ ] Unit tests
- [ ] Integration tests
- [ ] Benchmark tests
- [ ] Manual testing
- [ ] Other: <!-- Specify -->

## 7. Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Performance improvement
- [ ] Documentation update
- [ ] Code refactoring
- [ ] CI/CD improvement
- [ ] Test improvement
- [ ] Other: <!-- Specify -->

## 8. Checklist
<!-- Mark completed items with an [x] -->
- [ ] Code follows project style guidelines
- [ ] Changes respect role boundaries
- [ ] Documentation has been updated (if required)
- [ ] Tests have been added or updated
- [ ] All CI checks pass locally
- [ ] I have rebased my branch on the latest main
- [ ] I have verified my changes work with existing components

## 9. Agent-Specific Checks
<!-- Only complete if ROLE is specified above -->
- [ ] I've confirmed my agent role has permission to modify all changed files
- [ ] This PR only contains changes consistent with my assigned role
- [ ] If tester role: I've added or updated benchmarks as needed
- [ ] If architect role: I've updated ADRs as needed
- [ ] If doc-writer role: All documentation follows project standards
- [ ] If implementer role: Implementation aligns with architectural decisions
- [ ] If security-auditor role: I've validated security implications

## 10. Additional Notes
<!-- Any additional information that reviewers should know -->