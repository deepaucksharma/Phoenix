# Phoenix Platform Makefile

.PHONY: all build test clean help

# Variables
DOCKER_REGISTRY ?= phoenix
VERSION ?= latest
GO_VERSION = 1.21
NODE_VERSION = 18
KUBEBUILDER_VERSION = 3.11.1

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOTEST = $(GOCMD) test
GOMOD = $(GOCMD) mod
GOFMT = gofmt
GOLINT = golangci-lint

# Directories
BUILD_DIR = build
DIST_DIR = dist
VENDOR_DIR = vendor

# All targets
all: build

## help: Show this help message
help:
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## deps: Download and install dependencies
deps:
	@echo "Installing dependencies..."
	@$(GOMOD) download
	@cd dashboard && npm ci
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
	@go install sigs.k8s.io/kustomize/kustomize/v4@latest

## fmt: Format code
fmt:
	@echo "Formatting Go code..."
	@$(GOFMT) -w -s .
	@echo "Formatting frontend code..."
	@cd dashboard && npm run format

## lint: Run linters
lint:
	@echo "Linting Go code..."
	@$(GOLINT) run ./...
	@echo "Linting frontend code..."
	@cd dashboard && npm run lint

## test: Run all tests
test: test-unit test-integration

## test-unit: Run unit tests
test-unit:
	@echo "Running unit tests..."
	@$(GOTEST) -v -race -coverprofile=coverage.out ./...
	@cd dashboard && npm test

## test-integration: Run integration tests
test-integration:
	@echo "Running integration tests..."
	@$(GOTEST) -v -tags=integration ./test/integration/...

## test-e2e: Run end-to-end tests
test-e2e:
	@echo "Running e2e tests..."
	@$(GOTEST) -v -tags=e2e -timeout=30m ./test/e2e/...

## build: Build all components
build: build-api build-controller build-generator build-operators build-simulator build-dashboard

## build-api: Build API service
build-api:
	@echo "Building API service..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 $(GOBUILD) -o $(BUILD_DIR)/phoenix-api ./cmd/api

## build-controller: Build experiment controller
build-controller:
	@echo "Building experiment controller..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 $(GOBUILD) -o $(BUILD_DIR)/experiment-controller ./cmd/controller

## build-generator: Build configuration generator
build-generator:
	@echo "Building configuration generator..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 $(GOBUILD) -o $(BUILD_DIR)/config-generator ./cmd/generator

## build-operators: Build Kubernetes operators
build-operators:
	@echo "Building pipeline operator..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 $(GOBUILD) -o $(BUILD_DIR)/pipeline-operator ./operators/pipeline/cmd
	@echo "Building loadsim operator..."
	@CGO_ENABLED=0 $(GOBUILD) -o $(BUILD_DIR)/loadsim-operator ./operators/loadsim/cmd

## build-simulator: Build process simulator
build-simulator:
	@echo "Building process simulator..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 $(GOBUILD) -o $(BUILD_DIR)/process-simulator ./cmd/simulator

## build-dashboard: Build web dashboard
build-dashboard:
	@echo "Building dashboard..."
	@cd dashboard && npm run build
	@mkdir -p $(DIST_DIR)/dashboard
	@cp -r dashboard/dist/* $(DIST_DIR)/dashboard/

## docker: Build all Docker images
docker: docker-api docker-controller docker-generator docker-operators docker-simulator docker-dashboard

## docker-api: Build API Docker image
docker-api:
	@echo "Building API Docker image..."
	@docker build -f docker/api/Dockerfile -t $(DOCKER_REGISTRY)/phoenix-api:$(VERSION) .

## docker-controller: Build controller Docker image
docker-controller:
	@echo "Building controller Docker image..."
	@docker build -f docker/controller/Dockerfile -t $(DOCKER_REGISTRY)/experiment-controller:$(VERSION) .

## docker-generator: Build generator Docker image
docker-generator:
	@echo "Building generator Docker image..."
	@docker build -f docker/generator/Dockerfile -t $(DOCKER_REGISTRY)/config-generator:$(VERSION) .

## docker-operators: Build operator Docker images
docker-operators:
	@echo "Building operator Docker images..."
	@docker build -f docker/pipeline-operator/Dockerfile -t $(DOCKER_REGISTRY)/pipeline-operator:$(VERSION) .
	@docker build -f docker/loadsim-operator/Dockerfile -t $(DOCKER_REGISTRY)/loadsim-operator:$(VERSION) .

## docker-simulator: Build simulator Docker image
docker-simulator:
	@echo "Building simulator Docker image..."
	@docker build -f docker/simulator/Dockerfile -t $(DOCKER_REGISTRY)/process-simulator:$(VERSION) .

## docker-dashboard: Build dashboard Docker image
docker-dashboard: build-dashboard
	@echo "Building dashboard Docker image..."
	@docker build -f docker/dashboard/Dockerfile -t $(DOCKER_REGISTRY)/phoenix-dashboard:$(VERSION) .

## push: Push all Docker images
push:
	@echo "Pushing Docker images..."
	@docker push $(DOCKER_REGISTRY)/phoenix-api:$(VERSION)
	@docker push $(DOCKER_REGISTRY)/experiment-controller:$(VERSION)
	@docker push $(DOCKER_REGISTRY)/config-generator:$(VERSION)
	@docker push $(DOCKER_REGISTRY)/pipeline-operator:$(VERSION)
	@docker push $(DOCKER_REGISTRY)/loadsim-operator:$(VERSION)
	@docker push $(DOCKER_REGISTRY)/process-simulator:$(VERSION)
	@docker push $(DOCKER_REGISTRY)/phoenix-dashboard:$(VERSION)

## generate: Generate code and manifests
generate:
	@echo "Generating CRDs..."
	@controller-gen crd paths="./operators/..." output:crd:artifacts:config=k8s/crds
	@echo "Generating Go code..."
	@controller-gen object paths="./operators/..."
	@echo "Generating protobuf code..."
	@protoc --go_out=. --go-grpc_out=. proto/*.proto

## manifests: Generate Kubernetes manifests
manifests: generate
	@echo "Building Kubernetes manifests..."
	@kustomize build k8s/base > k8s/phoenix-all.yaml

## install-crds: Install CRDs in cluster
install-crds:
	@echo "Installing CRDs..."
	@kubectl apply -f k8s/crds/

## deploy: Deploy Phoenix to Kubernetes
deploy: install-crds
	@echo "Deploying Phoenix..."
	@kubectl apply -k k8s/overlays/development

## undeploy: Remove Phoenix from Kubernetes
undeploy:
	@echo "Removing Phoenix..."
	@kubectl delete -k k8s/overlays/development
	@kubectl delete -f k8s/crds/

## cluster-up: Start local Kubernetes cluster
cluster-up:
	@echo "Starting kind cluster..."
	@kind create cluster --name phoenix-dev --config test/kind-config.yaml
	@kubectl cluster-info --context kind-phoenix-dev

## cluster-down: Stop local Kubernetes cluster
cluster-down:
	@echo "Deleting kind cluster..."
	@kind delete cluster --name phoenix-dev

## port-forward: Forward ports for local development
port-forward:
	@echo "Port forwarding services..."
	@kubectl port-forward -n phoenix-system svc/phoenix-api-gateway 8443:443 &
	@kubectl port-forward -n phoenix-system svc/prometheus-server 9090:80 &
	@kubectl port-forward -n phoenix-system svc/grafana 3000:80 &

## clean: Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR) $(VENDOR_DIR)
	@rm -f coverage.out coverage.html
	@cd dashboard && rm -rf dist node_modules

## setup-hooks: Setup git hooks
setup-hooks:
	@echo "Setting up git hooks..."
	@cp scripts/pre-commit .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit

.DEFAULT_GOAL := help