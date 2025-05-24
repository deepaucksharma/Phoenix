# Phoenix Monorepo Build System

.PHONY: all install build test lint clean deploy help

# Colors
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m

# Default target
all: install build

## Install dependencies
install:
	@echo "$(BLUE)Installing dependencies...$(NC)"
	npm install
	@echo "$(GREEN)Dependencies installed!$(NC)"

## Build all projects (Node.js packages via Turborepo)
build:
	@echo "$(BLUE)Building all Node.js projects via Turborepo...$(NC)"
	@echo "$(YELLOW)Note: Go services are built within Docker containers$(NC)"
	npm run build
	@echo "$(GREEN)Node.js build complete!$(NC)"

## Build Docker images (includes Go services)
build-docker:
	@echo "$(BLUE)Building Docker images (includes all Go services)...$(NC)"
	docker-compose build
	@echo "$(GREEN)Docker images built!$(NC)"

## Build Go services locally
build-go:
	@echo "$(BLUE)Building Go services locally...$(NC)"
	cd apps/control-actuator-go && go build -o ../../bin/control-actuator
	cd apps/anomaly-detector && go build -o ../../bin/anomaly-detector
	cd services/benchmark && go build -o ../../bin/benchmark-controller
	@echo "$(GREEN)Go services built in bin/ directory!$(NC)"

## Run tests
test:
	@echo "$(BLUE)Running tests...$(NC)"
	npm run test

## Run integration tests
test-integration:
	@echo "$(BLUE)Running integration tests...$(NC)"
	npm run test:integration

## Lint code
lint:
	@echo "$(BLUE)Linting code...$(NC)"
	npm run lint

## Clean build artifacts
clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	npm run clean
	rm -rf node_modules
	rm -rf packages/*/node_modules
	rm -rf services/*/node_modules
	rm -rf tools/*/node_modules
	@echo "$(GREEN)Clean complete!$(NC)"

## Deploy to development
deploy-dev:
	@echo "$(BLUE)Deploying to development...$(NC)"
	npm run deploy:dev
	@echo "$(GREEN)Development deployment complete!$(NC)"

## Deploy to production
deploy-prod:
	@echo "$(YELLOW)Deploying to production...$(NC)"
	@read -p "Are you sure you want to deploy to production? [y/N] " -n 1 -r; \
	echo; \
	if [[ $$REPLY =~ ^[Yy]$$ ]]; then \
		npm run deploy:prod; \
		echo "$(GREEN)Production deployment complete!$(NC)"; \
	else \
		echo "$(RED)Production deployment cancelled.$(NC)"; \
	fi

## Show logs
logs:
	npm run logs

## Health check
health:
	@echo "$(BLUE)Running health checks...$(NC)"
	npm run health-check

## Development mode
dev:
	@echo "$(BLUE)Starting development mode...$(NC)"
	npm run dev

# Service-specific targets
collector-logs:
	docker-compose logs -f otelcol-main

observer-logs:
	docker-compose logs -f otelcol-observer

actuator-logs:
	docker-compose logs -f control-actuator-go

generator-logs:
	docker-compose logs -f synthetic-metrics-generator

anomaly-logs:
	docker-compose logs -f anomaly-detector

benchmark-logs:
	docker-compose logs -f benchmark-controller

# Utility targets
setup-env:
	@echo "$(BLUE)Setting up environment...$(NC)"
	./scripts/consolidated/core/initialize-environment.sh
	@echo "$(GREEN)Environment setup complete!$(NC)"

validate-config:
	@echo "$(BLUE)Validating configurations...$(NC)"
	@find configs -name "*.yaml" -o -name "*.yml" | xargs yamllint
	@echo "$(GREEN)Configuration validation complete!$(NC)"

# Documentation
docs-serve:
	@echo "$(BLUE)Serving documentation...$(NC)"
	cd docs && python -m http.server 8000

# Performance monitoring
monitor:
	@echo "$(BLUE)Opening monitoring dashboards...$(NC)"
	@echo "Grafana: http://localhost:3000 (admin/admin)"
	@echo "Prometheus: http://localhost:9090"
	@echo "Collector Metrics: http://localhost:8888/metrics"

## Show help
help:
	@echo "$(BLUE)Phoenix Monorepo Commands$(NC)"
	@echo ""
	@echo "$(YELLOW)Build Information:$(NC)"
	@echo "  - 'make build' builds Node.js packages via Turborepo"
	@echo "  - 'make build-docker' builds all Docker images (recommended for Go services)"
	@echo "  - 'make build-go' builds Go services locally (requires Go 1.21+)"
	@echo ""
	@echo "$(YELLOW)Main targets:$(NC)"
	@grep -E '^## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'
	@echo ""
	@echo "$(YELLOW)Service logs:$(NC)"
	@echo "  $(GREEN)collector-logs      $(NC) Show main collector logs"
	@echo "  $(GREEN)observer-logs       $(NC) Show observer collector logs"
	@echo "  $(GREEN)actuator-logs       $(NC) Show control actuator logs"
	@echo "  $(GREEN)generator-logs      $(NC) Show synthetic generator logs"
	@echo "  $(GREEN)anomaly-logs        $(NC) Show anomaly detector logs"
	@echo "  $(GREEN)benchmark-logs      $(NC) Show benchmark controller logs"
	@echo ""
	@echo "$(YELLOW)Utilities:$(NC)"
	@echo "  $(GREEN)setup-env           $(NC) Setup development environment"
	@echo "  $(GREEN)validate-config     $(NC) Validate YAML configurations"
	@echo "  $(GREEN)docs-serve          $(NC) Serve documentation locally"
	@echo "  $(GREEN)monitor             $(NC) Open monitoring dashboards"