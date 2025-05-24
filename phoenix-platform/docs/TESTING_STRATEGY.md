# Phoenix Platform Testing Strategy

This document outlines the testing strategy for the Phoenix Platform, covering component-level and overall project-level testing.

## 1. Current Testing Landscape

The Phoenix Platform currently has the following testing practices in place:

### 1.1. Go Backend Components (API, Simulator, Operators, etc.)

*   **Unit Tests:**
    *   Implemented using Go's standard `testing` package.
    *   Test files are co-located with the source code they test (e.g., `service_test.go` alongside `service.go`).
    *   Executed via `make test-unit` or `go test ./...`.
    *   The Makefile target includes options for race detection (`-race`) and code coverage reporting (`-coverprofile=coverage.out`).

### 1.2. React Dashboard

*   **Unit and Component Tests:**
    *   Implemented using `vitest` as the test runner and `@testing-library/react` for rendering components and simulating user interactions.
    *   Test files are typically located within the `src` directory, often alongside the components they test.
    *   Executed via `make test-unit` (which in turn runs `npm test` in the `dashboard` directory).
    *   The `package.json` file in `phoenix-platform/dashboard/` lists `vitest`, `@testing-library/react`, `@testing-library/jest-dom`, and `jsdom` as development dependencies for testing.

### 1.3. Integration and End-to-End (E2E) Tests

*   **Makefile Targets:** The main `Makefile` includes targets for integration and E2E tests:
    *   `test-integration`: Runs `$(GOTEST) -v -tags=integration ./test/integration/...`
    *   `test-e2e`: Runs `$(GOTEST) -v -tags=e2e -timeout=30m ./test/e2e/...`
*   **Current Status:**
    *   The `TECHNICAL_SPEC_MASTER.md` document refers to a `phoenix-platform/test/` directory containing `integration/` and `e2e/` subdirectories.
    *   However, as of the last review, these directories (`phoenix-platform/test/integration` and `phoenix-platform/test/e2e`) are either missing or empty.
    *   This indicates that while the infrastructure for invoking these tests is planned in the Makefile, the actual tests and their supporting directory structure have not yet been implemented.

This strategy document aims to build upon the existing unit testing practices and provide a clear path forward for implementing comprehensive integration and end-to-end testing.

## 2. Component-Level Testing Strategy

This section details the testing approach for each type of component within the Phoenix Platform.

### 2.1. Go Backend Services (API, Simulator, Config Generator)

These services form the core logic of the platform.

*   **Unit Tests:**
    *   **Framework:** Continue using Go's standard `testing` package.
    *   **Location:** Co-located `*_test.go` files.
    *   **Focus:** Test individual functions, methods, and logic units in isolation. Business logic, data transformations, and utility functions are key candidates.
    *   **Techniques:**
        *   Employ table-driven tests for comprehensive input/output validation.
        *   Utilize mocking for external dependencies (database connections, gRPC clients, other services). Standard library interfaces or tools like `gomock` can be used.
        *   Aim for high code coverage (e.g., >80%) for critical business logic.
        *   Ensure error paths are tested as thoroughly as success paths.

*   **Integration Tests:**
    *   **Purpose:** To verify interactions between a service and its direct dependencies (e.g., API service with PostgreSQL database, API service with Experiment Controller gRPC service).
    *   **Location:** Proposed under `test/integration/go/`. Subdirectories can be created per service (e.g., `test/integration/go/api`, `test/integration/go/simulator`).
    *   **Framework:** Use Go's standard `testing` package, potentially augmented with test setup/teardown utilities.
    *   **Techniques:**
        *   Spin up real dependencies using Docker containers (e.g., PostgreSQL, NATS). Libraries like `testcontainers-go` can manage the lifecycle of these containers.
        *   For gRPC services, test client-server communication.
        *   Focus on contract testing: ensuring that a service correctly interacts with its dependencies according to their defined APIs or schemas.
        *   Keep these tests separate from unit tests (e.g., using build tags like `//go:build integration`).

### 2.2. Kubernetes Operators (Pipeline Operator, LoadSim Operator)

Operators manage custom resources and interact with the Kubernetes API.

*   **Unit Tests:**
    *   **Framework:** Utilize `controller-runtime`'s testing utilities or Kubebuilder's `envtest` for testing reconciliation logic.
    *   **Location:** Co-located `*_test.go` files within the operator's source code (e.g., `operators/pipeline/controllers/pipeline_controller_test.go`).
    *   **Focus:** Test the reconciliation loop (`Reconcile` function) of each controller. Mock Kubernetes API interactions and client calls.
    *   **Techniques:**
        *   Create mock `CustomResource` objects and verify the controller's response (e.g., creation/update of other K8s resources, status updates on the CR).
        *   Test different states and conditions of the custom resources.

*   **Integration Tests:**
    *   **Purpose:** To test the operator's behavior against a real or simulated Kubernetes API server.
    *   **Location:** Proposed under `test/integration/operators/`.
    *   **Framework:** `envtest` (from Kubebuilder/controller-runtime) is highly recommended as it sets up a temporary Kubernetes API server and etcd instance.
    *   **Techniques:**
        *   Deploy the operator's Custom Resource Definitions (CRDs) to the test API server.
        *   Create instances of the CRs and verify that the operator takes the expected actions (e.g., creating Deployments, Services, Jobs).
        *   Test status updates, finalizers, and lifecycle management of the CRs.

### 2.3. React Dashboard (Frontend)

The dashboard provides the user interface for the platform.

*   **Unit Tests:**
    *   **Framework:** Continue using `vitest` and `@testing-library/react`.
    *   **Location:** Co-located with components (e.g., `src/components/MyComponent/MyComponent.test.tsx`).
    *   **Focus:** Test individual React components in isolation.
    *   **Techniques:**
        *   Verify correct rendering based on props.
        *   Test basic user interactions (clicks, input changes) and their immediate effects within the component.
        *   Mock child components or external hooks/services that are not relevant to the component's specific logic.

*   **Component Integration Tests:**
    *   **Framework:** `vitest` and `@testing-library/react`.
    *   **Location:** Can be co-located or in a dedicated `tests` subdirectory within the dashboard source.
    *   **Focus:** Test the interaction between several components that form a piece of functionality (e.g., a form with multiple input fields and a submit button, a list view with filtering and sorting).
    *   **Techniques:**
        *   Render a tree of components and verify their collective behavior.
        *   Test data flow between parent and child components.

*   **Mocking API Calls:**
    *   **Strategy:** For unit and component integration tests that rely on API data, API calls should be mocked to ensure tests are fast, reliable, and independent of the backend.
    *   **Recommended Tool:** `msw` (Mock Service Worker) is a powerful library that allows intercepting network requests at the network level. This means components can make actual `fetch` or `axios` calls, and `msw` will provide mock responses.
    *   **Alternative:** For simpler cases, `vitest.mock()` can be used to mock specific API client modules or functions.

## 3. Overall Project-Level Testing Strategy

This section covers testing strategies that span multiple components and validate the platform as a whole.

### 3.1. End-to-End (E2E) Tests

*   **Purpose:** To verify complete user flows and interactions across multiple components of the Phoenix platform, from the UI to the backend services and Kubernetes resources.
*   **Location:** Proposed under `test/e2e/`.
*   **Framework Recommendations:**
    *   **UI-centric E2E Tests:** Frameworks like **Playwright** or **Cypress** are excellent for testing user interactions through the web dashboard. They offer robust features for browser automation, assertions, and debugging. Playwright also has strong capabilities for API testing, which can be useful for setting up state or verifying backend responses.
    *   **Platform/API-centric E2E Tests:** For scenarios that primarily involve API interactions, backend processes, and Kubernetes resource manipulations, Go's `testing` package can be used. This allows for writing E2E tests in the same language as the backend, potentially sharing some utility code.
    *   **Hybrid Approach:** A combination might be most effective: Playwright/Cypress for UI-driven flows and Go tests for complex backend orchestrations or scenarios not easily driven through the UI.
*   **Scope:**
    *   Cover key user workflows, such as:
        *   Creating a new experiment through the dashboard.
        *   Visual pipeline builder interactions leading to OTel configuration generation.
        *   Deployment of baseline and candidate OTel collectors.
        *   Verification of metrics collection and data appearing in Prometheus/New Relic (or mock equivalents).
        *   Viewing experiment results and cost analysis.
        *   User authentication and authorization flows.
*   **Environment:**
    *   E2E tests should run against a fully deployed instance of the Phoenix platform.
    *   This could be a local deployment using `kind` (a `test/kind-config.yaml` is mentioned in `TECHNICAL_SPEC_MASTER.md`).
    *   Alternatively, a dedicated, ephemeral test environment spun up in a CI/CD pipeline.
*   **Data Management:** E2E tests require careful consideration of test data setup and teardown to ensure they are repeatable and independent.

### 3.2. Performance Tests

*   **Purpose:** To ensure the platform meets its performance requirements under various load conditions, identify bottlenecks, and validate scalability. (Refer to `TECHNICAL_SPEC_MASTER.md` for specific SLOs and scalability targets).
*   **Location:** Could be organized under `test/performance/` or integrated within E2E test suites if appropriate.
*   **Tool Recommendations:**
    *   **API Load Testing:** Tools like **k6** (JavaScript/Go), **Locust** (Python), or **JMeter** are suitable for generating load on the API Gateway and other backend services.
    *   **Go Benchmarking:** Go's built-in `testing` package provides benchmarking capabilities (`testing.B`) which can be used for micro-benchmarks of critical code paths and specific Go services.
*   **Scope:**
    *   **API Server:** Test request latency, throughput, and error rates under concurrent user load for key API endpoints.
    *   **OTel Collectors:** Measure resource usage (CPU, memory), processing latency, and data throughput under various metric loads.
    *   **Experiment Controller & Operators:** Test reconciliation times and resource impact under a high number of custom resources.
    *   **Dashboard:** Frontend performance (load time, responsiveness) can be assessed using tools like Lighthouse, integrated into E2E test frameworks.
*   **Strategy:**
    *   Establish baseline performance metrics.
    *   Run tests regularly (e.g., nightly, before releases) to detect performance regressions.
    *   Include different load profiles (e.g., steady load, burst load).

### 3.3. Security Testing

*   **Purpose:** To identify and mitigate security vulnerabilities within the platform. (Refer to `TECHNICAL_SPEC_MASTER.md` Section 8 for Security Architecture).
*   **Strategy (High-Level):**
    *   **Static Application Security Testing (SAST):** Integrate tools like `gosec` for Go, and `eslint-plugin-security` for TypeScript/JavaScript into the CI pipeline to scan code for potential vulnerabilities.
    *   **Dynamic Application Security Testing (DAST):** Periodically run DAST tools against a deployed test environment to find runtime vulnerabilities.
    *   **Dependency Scanning:**
        *   Utilize GitHub's Dependabot (as suggested by `dependabot.yml` in the master spec) or tools like `npm audit` / `go list -m -json all | go-vulncheck-parser` to identify and alert on known vulnerabilities in third-party libraries.
        *   Regularly update dependencies.
    *   **Container Image Scanning:** Scan Docker images for known vulnerabilities using tools integrated with the container registry or CI (e.g., Trivy, Clair).
    *   **Penetration Testing:** Consider periodic penetration testing by a third party for critical releases or significant changes.

While detailed implementation of all security tests is a broad effort, acknowledging these categories is crucial for a comprehensive testing strategy.

## 4. Proposed Test Directory Structure

To house the integration and end-to-end tests, the following directory structure is proposed under `phoenix-platform/test/`. This aligns with the structure mentioned in `TECHNICAL_SPEC_MASTER.md` and the Makefile targets.

**Action Item:** These directories currently do not exist or are empty and need to be created.

```
phoenix-platform/
├── test/
│   ├── e2e/
│   │   ├── dashboard/  # UI E2E tests (e.g., Playwright/Cypress)
│   │   │   ├── specs/              # Test spec files
│   │   │   ├── pages/              # Page Object Models
│   │   │   └── fixtures/           # Test data fixtures
│   │   └── platform/   # Platform/API E2E tests (e.g., Go tests)
│   │       ├── experiments_test.go # Example E2E test for experiments
│   │       └── pipelines_test.go   # Example E2E test for pipelines
│   ├── integration/
│   │   ├── go/         # Go backend services integration tests
│   │   │   ├── api/                # Integration tests for the API service
│   │   │   │   └── api_integration_test.go
│   │   │   ├── simulator/          # Integration tests for the Simulator
│   │   │   └── ...                 # Other services
│   │   ├── operators/  # Kubernetes Operator integration tests
│   │   │   ├── pipeline_operator_integration_test.go
│   │   │   └── loadsim_operator_integration_test.go
│   │   └── testdata/   # Common test data, fixtures, SQL schemas for tests
│   │       ├── sample_experiment.json
│   │       └── db_schema.sql
│   └── kind-config.yaml # Example KinD cluster configuration for local E2E testing
```

**Explanation of Directories:**

*   `test/e2e/dashboard/`: For UI-focused end-to-end tests.
    *   `specs/`: Contains the actual test scripts or spec files.
    *   `pages/`: Page Object Models representing pages or major components of the dashboard, promoting reusable and maintainable test code.
    *   `fixtures/`: Test data specific to UI E2E tests.
*   `test/e2e/platform/`: For end-to-end tests focusing on backend APIs, platform workflows, and overall system behavior without necessarily driving through the UI.
*   `test/integration/go/`: For integration tests of individual Go backend services with their direct dependencies (databases, other services via gRPC, etc.).
*   `test/integration/operators/`: For integration tests of Kubernetes operators, likely using tools like `envtest`.
*   `test/integration/testdata/`: For shared test data, configuration files, database schemas, or any fixtures needed by multiple integration or E2E tests.
*   `test/kind-config.yaml`: A configuration file for creating a local Kubernetes cluster using KinD (Kubernetes in Docker), suitable for running E2E and integration tests that require a Kubernetes environment.

This structure provides a clear separation of concerns for different types of tests and aligns with common Go and testing best practices.

## 5. Tooling and Infrastructure

Consistent tooling and a robust infrastructure are key to an effective testing strategy.

### 5.1. Test Execution

*   **Makefile Targets:** Continue using and expanding `make` targets for executing different types of tests. This provides a consistent interface for developers and CI/CD systems. Example targets from the current `Makefile`:
    *   `make test-unit`
    *   `make test-integration` (currently points to a non-existent location)
    *   `make test-e2e` (currently points to a non-existent location)
    *   Consider adding `make test-all` to run all checks (lint, unit, integration).
    *   Targets for running performance tests could also be added: `make test-performance-api`.
*   **Build Tags:** Utilize Go build tags (e.g., `//go:build integration`) to segregate tests that require specific environments or have longer run times (like integration tests) from standard unit tests.

### 5.2. Test Environment Dependencies

*   **Docker:** Docker should be extensively used for managing dependencies for integration and end-to-end tests. This includes:
    *   Databases (e.g., PostgreSQL)
    *   Messaging queues (e.g., NATS, if used)
    *   Mock external services
    *   `testcontainers-go` is recommended for managing the lifecycle of these Dockerized dependencies within Go-based integration tests.
*   **Kubernetes:**
    *   `kind` (Kubernetes in Docker) is recommended for creating local Kubernetes clusters for E2E testing and operator integration testing (as suggested by `test/kind-config.yaml`).
    *   `envtest` for operator testing provides a lightweight Kubernetes API server.

### 5.3. Code Quality and Coverage

*   **Linters:** Continue using `golangci-lint` for Go and `ESLint` for TypeScript/JavaScript, as configured in the `Makefile` and `package.json`.
*   **Code Formatters:** `gofmt -s .` for Go and `prettier` for the dashboard are already in use and should be enforced, ideally via pre-commit hooks.
*   **Code Coverage:**
    *   For Go unit tests, continue generating coverage reports (`-coverprofile=coverage.out`).
    *   Aim for a meaningful coverage target (e.g., 80% for critical modules) and consider tools to track coverage trends over time (e.g., Codecov, Coveralls).
    *   `vitest` also provides coverage capabilities for the frontend code.

### 5.4. CI/CD Integration

*   **GitHub Actions:** The `TECHNICAL_SPEC_MASTER.md` indicates GitHub Actions are used for CI/CD. All automated tests should be integrated into the GitHub Actions workflows.
    *   Run linters and formatters on every commit/PR.
    *   Run unit tests on every commit/PR.
    *   Run integration tests on every PR, targeting the main branch.
    *   E2E tests might be run on PRs if they are fast enough, or nightly/on merges to the main branch due to their longer execution time and resource requirements.
    *   Performance tests could be run nightly or on demand.
    *   Security scans (SAST, dependency checks) should also be part of the CI pipeline.

## 6. Testing in CI/CD

Integrating automated testing into the Continuous Integration/Continuous Deployment (CI/CD) pipeline is crucial for maintaining code quality and enabling rapid, reliable releases. The `TECHNICAL_SPEC_MASTER.md` indicates GitHub Actions as the CI/CD platform.

### 6.1. Workflow Triggers and Test Execution

The following outlines when different types of tests should be executed within the GitHub Actions workflows:

*   **On Every Push to Any Branch / Pull Request Creation & Updates:**
    *   **Static Analysis:**
        *   Code linting (Go, TypeScript/JavaScript, YAML, etc.).
        *   Code formatting checks.
        *   SAST (Static Application Security Testing) scans.
        *   Dependency vulnerability scans (e.g., Dependabot alerts, `npm audit`).
    *   **Unit Tests:**
        *   Go unit tests (`make test-unit` or `go test ./...`).
        *   Dashboard unit and component tests (`cd dashboard && npm test`).
    *   **Build Verification:**
        *   Build all Go binaries (`make build`).
        *   Build dashboard (`make build-dashboard`).
        *   Build Docker images (`make docker`). This also helps catch Dockerfile issues early.

*   **On Pull Requests Targeting `main` (or other protected release branches):**
    *   All tests from the "Every Push" stage.
    *   **Integration Tests:**
        *   Go backend services integration tests (`make test-integration`). These may require setting up service dependencies (databases, etc.) using Docker within the CI environment.
        *   Kubernetes Operator integration tests (also via `make test-integration`, potentially with a different tag or target, using `envtest`).
    *   **Smoke Tests (Optional Subset of E2E):** If a small, critical subset of E2E tests can run quickly, they can be included here.

*   **On Merge to `main` Branch / Nightly Schedules:**
    *   All tests from the "Pull Requests Targeting `main`" stage.
    *   **End-to-End (E2E) Tests:**
        *   Full E2E test suite (`make test-e2e`). These tests will typically require deploying the entire platform to a Kubernetes environment (e.g., `kind` within CI, or a dedicated test environment).
    *   **Performance Tests:**
        *   Run performance benchmarks and load tests against key components (e.g., `make test-performance-api`). Results should be tracked to identify regressions.
    *   **Container Image Scans:** After images are built and pushed to a staging registry.

### 6.2. Reporting and Artifacts

*   **Test Results:** CI jobs should clearly indicate success or failure. Test runners often produce XML reports (e.g., JUnit format) that can be parsed by GitHub Actions for better display of test failures.
*   **Coverage Reports:** Upload code coverage reports (e.g., `coverage.out` from Go, LCOV from Vitest) as build artifacts. Consider integrating with services like Codecov or Coveralls to visualize coverage and track changes.
*   **Build Artifacts:** Store compiled binaries and Docker images (pushed to a staging/internal registry) for successful builds on the `main` branch, to be used for deployment.

### 6.3. Optimizations

*   **Caching:** Cache dependencies (Go modules, npm packages, Docker layers) to speed up CI runs.
*   **Parallelization:** Run independent test suites or jobs in parallel where possible.
*   **Selective Execution:** For very large projects, explore options to run tests only for the changed parts of the codebase (though this adds complexity).

A robust CI/CD testing pipeline provides rapid feedback to developers, prevents regressions, and builds confidence in the stability of the platform.
