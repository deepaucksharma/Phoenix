# tools.mk - Role-specific Make targets
.PHONY: architect-check planner-check implementer-check reviewer-check security-auditor-check doc-writer-check devops-check integrator-check

# Architect checks
architect-check:
	@echo "Running checks for architect role..."
	@hack/validate-adr.sh

# Planner checks
planner-check:
	@echo "Running checks for planner role..."
	@find tasks -name "*.yaml" -exec hack/validate-task.sh {} \;

# Implementer checks
implementer-check: lint test drift-check
	@echo "Running checks for implementer role..."

# Reviewer checks
reviewer-check:
	@echo "Running checks for reviewer role..."
	@echo "Review complete" # Placeholder - no specific checks needed

# Security Auditor checks
security-auditor-check:
	@echo "Running checks for security-auditor role..."
	@go run golang.org/x/vuln/cmd/govulncheck@latest ./...

# Doc Writer checks
doc-writer-check:
	@echo "Running checks for doc-writer role..."
	@echo "Checking markdown files..."
	@find . -name "*.md" -not -path "./vendor/*" | xargs markdownlint

# DevOps checks
devops-check:
	@echo "Running checks for devops role..."
	@echo "Validating workflows..."
	@find .github/workflows -name "*.yml" -exec yamllint {} \;
	@echo "Checking Dockerfile..."
	@hadolint deploy/docker/Dockerfile

# Integrator checks
integrator-check: lint test drift-check
	@echo "Running checks for integrator role..."
	@scripts/validation/validate_policy_schema.sh
	@scripts/ci/check_component_registry.sh
