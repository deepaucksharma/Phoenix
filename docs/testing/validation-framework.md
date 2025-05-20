# SA-OMF (Phoenix) Testing & Validation Framework

This ultra-detailed testing framework provides a comprehensive approach to validate every component and integration point in the SA-OMF system. Following this framework after each implementation step ensures that issues are caught early and the system remains stable throughout development.

## Table of Contents

1. [Test Infrastructure Setup](#1-test-infrastructure-setup)
2. [Component Testing Framework](#2-component-testing-framework)
3. [Phase-by-Phase Validation Checklist](#3-phase-by-phase-validation-checklist)
4. [End-to-End Testing Strategy](#4-end-to-end-testing-strategy)
5. [Performance & Chaos Testing](#5-performance--chaos-testing)
6. [Continuous Validation Pipeline](#6-continuous-validation-pipeline)

## 1. Test Infrastructure Setup

### 1.1 Local Development Environment

```bash
# Create test environments with Docker Compose
# Example compose files are provided in `test-environments/*`.
# Run the following commands if you need to recreate them manually.
mkdir -p test-environments/{bare,prometheus,full}

# Bare Environment (just the collector)
cat > test-environments/bare/docker-compose.yaml <<EOL
version: '3'
services:
  sa-omf-collector:
    image: yourorg/sa-omf-collector:latest
    build: ../../
    volumes:
      - ./config.yaml:/etc/sa-omf/config.yaml
      - ./policy.yaml:/etc/sa-omf/policy.yaml
      - /proc:/proc:ro
      - /sys:/sys:ro
    ports:
      - "8888:8888"
      - "13133:13133"
EOL

# Prometheus Environment (collector + Prometheus)
cp test-environments/bare/docker-compose.yaml test-environments/prometheus/
cat >> test-environments/prometheus/docker-compose.yaml <<EOL
  prometheus:
    image: prom/prometheus:v2.48.0
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"
EOL

# Full Environment (collector + Prometheus + Grafana)
cp test-environments/prometheus/docker-compose.yaml test-environments/full/
cat >> test-environments/full/docker-compose.yaml <<EOL
  grafana:
    image: grafana/grafana:10.2.0
    volumes:
      - ./grafana/provisioning:/etc/grafana/provisioning
      - ../../dashboards:/var/lib/grafana/dashboards
    environment:
      - GF_SECURITY_ADMIN_USER=admin
      - GF_SECURITY_ADMIN_PASSWORD=admin
    ports:
      - "3000:3000"
EOL
```

### 1.2 Kubernetes Test Cluster

```bash
# Create a kind cluster for Kubernetes testing
cat > kind-config.yaml <<EOL
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
EOL

kind create cluster --name sa-omf-test --config kind-config.yaml

# Install Prometheus Operator
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring --create-namespace
```

### 1.3 Test Data Generator

Create a synthetic workload generator that can simulate various host/process metrics patterns:

```bash
cat > test/generator/workload.go <<EOL
package main

import (
    "flag"
    "log"
    "math/rand"
    "time"
    // other imports
)

func main() {
    var (
        processCount = flag.Int("processes", 1000, "Number of processes to simulate")
        spikeFrequency = flag.Float64("spike-freq", 0.05, "Frequency of load spikes")
        cardinality = flag.Int("cardinality", 5000, "Label cardinality to generate")
        duration = flag.Duration("duration", 10*time.Minute, "Test duration")
    )
    flag.Parse()
    
    // Simulation logic here
}
EOL
```

## 2. Component Testing Framework

### 2.1 Interface Testing

For each core interface, create test suites that verify implementation correctness:

```bash
# UpdateableProcessor Interface Test Suite
cat > test/interfaces/updateable_processor_test.go <<EOL
package interfaces

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    "github.com/yourorg/sa-omf/api/updateable"
)

// TestUpdateableProcessor provides a reusable test suite for any
// processor implementing the UpdateableProcessor interface
func TestUpdateableProcessor(t *testing.T, processor updateable.UpdateableProcessor) {
    ctx := context.Background()
    
    // Test GetConfigStatus
    status, err := processor.GetConfigStatus(ctx)
    require.NoError(t, err)
    require.NotNil(t, status.Parameters)
    
    // Test parameter patching
    // For each parameter supported by the processor...
    
    // Test invalid values
    invalidPatch := updateable.ConfigPatch{
        // Invalid configuration
    }
    err = processor.OnConfigPatch(ctx, invalidPatch)
    assert.Error(t, err)
    
    // Test enabling/disabling
    enablePatch := updateable.ConfigPatch{
        ParameterPath: "enabled",
        NewValue: true,
    }
    err = processor.OnConfigPatch(ctx, enablePatch)
    require.NoError(t, err)
    
    status, _ = processor.GetConfigStatus(ctx)
    assert.True(t, status.Enabled)
}
EOL
```

### 2.2 Algorithm Unit Tests

Each algorithm implementation must have comprehensive unit tests:

```bash
# Top-K Algorithm Testing
cat > test/alg/topk/space_saving_test.go <<EOL
package topk

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/yourorg/sa-omf/core/alg/topk"
)

func TestSpaceSaving(t *testing.T) {
    // Test basic functionality
    ss := topk.NewSpaceSaving(10)
    
    // Add items with known distribution
    for i := 0; i < 1000; i++ {
        ss.Add("item1", 1.0)
    }
    for i := 0; i < 500; i++ {
        ss.Add("item2", 1.0)
    }
    // Add many more items with smaller counts
    
    // Verify correct top items
    items := ss.GetTopK()
    assert.Equal(t, "item1", items[0].ID)
    assert.Equal(t, "item2", items[1].ID)
    
    // Test coverage calculation
    coverage := ss.GetCoverage()
    assert.InDelta(t, 0.95, coverage, 0.05)
    
    // Test K adjustment
    ss.SetK(5)
    assert.Equal(t, 5, len(ss.GetTopK()))
    
    // Test error bounds
    // ...
}
EOL
```

### 2.3 Processor Testing Template

Create a standardized test framework for all processors:

```bash
cat > test/processors/processor_test_template.go <<EOL
package processors

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go.opentelemetry.io/collector/component"
    "go.opentelemetry.io/collector/consumer/consumertest"
    "go.opentelemetry.io/collector/pdata/pmetric"
    
    "github.com/yourorg/sa-omf/api/updateable"
    "github.com/yourorg/sa-omf/test/interfaces"
)

// ProcessorTestCase defines a standardized test case for processors
type ProcessorTestCase struct {
    Name           string
    InputMetrics   pmetric.Metrics
    ExpectedOutput func(pmetric.Metrics) bool
    ConfigPatches  []updateable.ConfigPatch
}

// RunProcessorTests executes a standardized set of tests for any processor
func RunProcessorTests(t *testing.T, factory component.Factory, defaultConfig component.Config, testCases []ProcessorTestCase) {
    for _, tc := range testCases {
        t.Run(tc.Name, func(t *testing.T) {
            // Setup
            next := new(consumertest.MetricsSink)
            processor, err := factory.CreateMetricsProcessor(
                context.Background(),
                component.ProcessorCreateSettings{},
                defaultConfig,
                next,
            )
            require.NoError(t, err)
            
            // Start the processor
            err = processor.Start(context.Background(), nil)
            require.NoError(t, err)
            
            // Test updateable interface
            if upProc, ok := processor.(updateable.UpdateableProcessor); ok {
                interfaces.TestUpdateableProcessor(t, upProc)
                
                // Apply config patches
                for _, patch := range tc.ConfigPatches {
                    err = upProc.OnConfigPatch(context.Background(), patch)
                    require.NoError(t, err)
                }
            }
            
            // Process metrics
            err = processor.ConsumeMetrics(context.Background(), tc.InputMetrics)
            require.NoError(t, err)
            
            // Verify output
            allMetrics := next.AllMetrics()
            require.NotEmpty(t, allMetrics)
            assert.True(t, tc.ExpectedOutput(allMetrics[0]))
            
            // Shutdown
            err = processor.Shutdown(context.Background())
            require.NoError(t, err)
        })
    }
}
EOL
```

### 2.4 PIC Control Extension Testing

The pic_control extension requires specialized tests:

```bash
cat > test/extension/pic_control_test.go <<EOL
package extension

import (
    "context"
    "os"
    "path/filepath"
    "testing"
    "time"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "go.opentelemetry.io/collector/component"
    
    "github.com/yourorg/sa-omf/api/updateable"
    "github.com/yourorg/sa-omf/otel/ext_pic"
)

func TestPICControl(t *testing.T) {
    // Setup temp policy file
    dir, err := os.MkdirTemp("", "pic-test")
    require.NoError(t, err)
    defer os.RemoveAll(dir)
    
    policyPath := filepath.Join(dir, "policy.yaml")
    err = os.WriteFile(policyPath, []byte("global_settings:\n  autonomy_level: shadow"), 0644)
    require.NoError(t, err)
    
    // Create extension
    factory := ext_pic.NewFactory()
    config := factory.CreateDefaultConfig().(*ext_pic.Config)
    config.PolicyFilePath = policyPath
    
    ext, err := factory.CreateExtension(context.Background(), component.ExtensionCreateSettings{}, config)
    require.NoError(t, err)
    
    // Start with mock host
    mockHost := newMockHost()
    err = ext.Start(context.Background(), mockHost)
    require.NoError(t, err)
    
    // Test policy file watching
    err = os.WriteFile(policyPath, []byte("global_settings:\n  autonomy_level: active"), 0644)
    require.NoError(t, err)
    
    // Wait for policy reload
    time.Sleep(200 * time.Millisecond)
    
    // Test patch submission
    picControl, ok := ext.(ext_pic.PicControl)
    require.True(t, ok)
    
    patch := updateable.ConfigPatch{
        PatchID: "test-patch",
        TargetProcessorName: component.NewID("processor"),
        ParameterPath: "test",
        NewValue: 42,
    }
    
    // Test with no processors
    err = picControl.SubmitConfigPatch(context.Background(), patch)
    assert.Error(t, err)
    
    // Add mock processor
    mockProc := newMockProcessor()
    mockHost.AddProcessor(component.NewID("processor"), mockProc)
    
    // Test with processor
    err = picControl.SubmitConfigPatch(context.Background(), patch)
    require.NoError(t, err)
    
    // Verify patch was applied
    assert.True(t, mockProc.PatchApplied)
    
    // Test rate limiting
    for i := 0; i < 10; i++ {
        patch.PatchID = fmt.Sprintf("test-patch-%d", i)
        err = picControl.SubmitConfigPatch(context.Background(), patch)
        if i >= config.MaxPatchesPerMinute {
            assert.Error(t, err)
        }
    }
    
    // Test safe mode
    // ...
    
    // Shutdown
    err = ext.Shutdown(context.Background())
    require.NoError(t, err)
}

// Mock implementations
func newMockHost() *mockHost { ... }
func newMockProcessor() *mockProcessor { ... }
EOL
```

## 3. Phase-by-Phase Validation Checklist

### 3.1 Phase 1: Foundation - Validation Script

```bash
#!/bin/bash
# validate_phase1.sh

set -e

echo "=== Validating Phase 1: Foundation ==="

echo "Step 1: Building code"
make build

echo "Step 2: Running unit tests"
make test

echo "Step 3: Starting bare environment"
cd test-environments/bare
docker-compose up -d
sleep 5

echo "Step 4: Checking health endpoint"
curl -s http://localhost:13133/health | grep -q "status\":\"ready" || { echo "Health check failed"; exit 1; }

echo "Step 5: Validating metrics endpoint"
curl -s http://localhost:8888/metrics | grep -q "aemf_pic_control" || { echo "Self-metrics missing"; exit 1; }
curl -s http://localhost:8888/metrics | grep -q "aemf_priority_tagger" || { echo "Priority tagger metrics missing"; exit 1; }
curl -s http://localhost:8888/metrics | grep -q "aemf_adaptive_topk" || { echo "Adaptive topK metrics missing"; exit 1; }

echo "Step 6: Validating config patch functionality"
# Use test client to submit test patch
go run test/clients/patch_client.go --target adaptive_topk --param k_value --value 15
sleep 2
# Verify change was applied
curl -s http://localhost:8888/metrics | grep -q "aemf_adaptive_topk_current_k_value.*15" || { echo "Config patch failed"; exit 1; }

echo "Step 7: Checking policy.yaml hot reload"
sed -i 's/k_value: 25/k_value: 20/' policy.yaml
sleep 3
curl -s http://localhost:8888/metrics | grep -q "aemf_adaptive_topk_current_k_value.*20" || { echo "Policy hot reload failed"; exit 1; }

echo "Step 8: Cleanup"
docker-compose down
cd ../..

echo "=== Phase 1 validation PASSED ==="
```

### 3.2 Phase 2: Enhanced Processors - Validation Script

```bash
#!/bin/bash
# validate_phase2.sh

set -e

echo "=== Validating Phase 2: Enhanced Processors ==="

echo "Step 1: Building code with new processors"
make build

echo "Step 2: Running extended unit tests"
make test

echo "Step 3: Starting Prometheus environment"
cd test-environments/prometheus
docker-compose up -d
sleep 10

echo "Step 4: Generating test workload"
go run ../../test/generator/workload.go --processes 1000 --cardinality 8000 --duration 2m &
GENERATOR_PID=$!
sleep 30

echo "Step 5: Validating cardinality reduction"
# Query Prometheus for cardinality stats
cardinality_reduction=$(curl -s "http://localhost:9090/api/v1/query?query=aemf_impact_cardinality_reduction_ratio" | jq '.data.result[0].value[1]' | tr -d '"')
echo "Cardinality reduction: $cardinality_reduction"
if (( $(echo "$cardinality_reduction < 0.7" | bc -l) )); then
  echo "Cardinality reduction too low"
  exit 1
fi

echo "Step 6: Validating coverage score"
coverage=$(curl -s "http://localhost:9090/api/v1/query?query=aemf_impact_adaptive_topk_coverage_score" | jq '.data.result[0].value[1]' | tr -d '"')
echo "Coverage score: $coverage"
if (( $(echo "$coverage < 0.85" | bc -l) )); then
  echo "Coverage score too low"
  exit 1
fi

echo "Step 7: Validating PID control functionality"
# Modify workload to trigger adaptation
kill $GENERATOR_PID
go run ../../test/generator/workload.go --processes 5000 --cardinality 20000 --duration 2m &
GENERATOR_PID=$!
sleep 60

# Check for config changes in response
patch_count=$(curl -s "http://localhost:9090/api/v1/query?query=aemf_decision_patch_applied_total" | jq '.data.result[0].value[1]' | tr -d '"')
if [[ "$patch_count" == "0" ]]; then
  echo "No configuration patches detected"
  exit 1
fi

echo "Step 8: Validating multi-KPI control"
# Check that both coverage and cardinality targets are maintained
coverage=$(curl -s "http://localhost:9090/api/v1/query?query=aemf_impact_adaptive_topk_coverage_score" | jq '.data.result[0].value[1]' | tr -d '"')
cardinality_reduction=$(curl -s "http://localhost:9090/api/v1/query?query=aemf_impact_cardinality_reduction_ratio" | jq '.data.result[0].value[1]' | tr -d '"')

if (( $(echo "$coverage < 0.85" | bc -l) )) || (( $(echo "$cardinality_reduction < 0.7" | bc -l) )); then
  echo "Multi-KPI control failed"
  exit 1
fi

echo "Step 9: Cleanup"
kill $GENERATOR_PID
docker-compose down
cd ../..

echo "=== Phase 2 validation PASSED ==="
```

### 3.3 Phase 3: Advanced Intelligence - Validation Script

```bash
#!/bin/bash
# validate_phase3.sh

set -e

echo "=== Validating Phase 3: Advanced Intelligence ==="

# Similar structure to Phase 2, with additional tests for:
# - Process context learner functionality
# - OpAMP integration
# - Bayesian optimization fallbacks

echo "Step 5: Validating process importance learning"
# Generate stable workload with consistent patterns
go run ../../test/generator/workload.go --processes 500 --stable-patterns 20 --duration 5m &
GENERATOR_PID=$!
sleep 180

# Check that critical processes are identified
importance_metrics=$(curl -s "http://localhost:9090/api/v1/query?query=aemf_impact_process_importance_index" | jq '.data.result | length')
if [[ "$importance_metrics" -lt 10 ]]; then
  echo "Process importance learning not working"
  exit 1
fi

# ... additional tests for advanced features
```

### 3.4 Phase 4: Production Hardening - Validation Checklist

For the final phase, create a comprehensive checklist that must be manually verified:

```yaml
# production_readiness.yaml
---
performance:
  - CPU usage < 250 mCores under 5000-process workload
  - Memory usage < 256 MiB under 5000-process workload
  - Startup time < 5 seconds
  - Config reload time < 1 second
  - Patch application latency < 100ms

stability:
  - Survives 7-day continuous operation
  - No memory leaks (flat RSS over time)
  - Graceful recovery from OOM conditions
  - Clean shutdown and state persistence

security:
  - Policy file permission checks
  - No sensitive data in logs or metrics
  - TLS configuration for OpAMP

usability:
  - All metrics properly labeled
  - All dashboards functional
  - Alert thresholds properly calibrated
  - Documentation complete and accurate

correctness:
  - Safe mode activates under resource pressure
  - Policy hot-reload works reliably
  - All processors respect enabled flag
  - PID controllers maintain stability

compatibility:
  - Works with Prometheus v2.45+
  - Works with Grafana v9.5+
  - Compatible with OTel SDK v1.10+
  - Kubernetes 1.25+ compatible
```

## 4. End-to-End Testing Strategy

### 4.1 Kubernetes Integration Test

```bash
# install-and-test.sh

#!/bin/bash
set -e

echo "=== Running full Kubernetes integration test ==="

# Build and push Docker image
make docker
make docker-push

# Deploy to test cluster
kubectl apply -f deploy/kubernetes/

# Wait for deployment
kubectl rollout status daemonset/sa-omf-collector -n monitoring --timeout=2m

# Check that all pods are running
RUNNING_PODS=$(kubectl get pods -n monitoring -l app=sa-omf-collector -o jsonpath='{.items[*].status.phase}' | tr ' ' '\n' | grep -c "Running")
TOTAL_PODS=$(kubectl get nodes | grep -c "Ready")
if [[ "$RUNNING_PODS" -ne "$TOTAL_PODS" ]]; then
  echo "Not all pods are running: $RUNNING_PODS/$TOTAL_PODS"
  exit 1
fi

# Generate test workload on each node
kubectl create configmap workload-generator --from-file=test/generator/workload.go -n monitoring
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: workload-generator
  namespace: monitoring
spec:
  template:
    spec:
      containers:
      - name: generator
        image: golang:1.22
        command: ["go", "run", "/workload/workload.go", "--processes", "2000", "--duration", "10m"]
        volumeMounts:
        - name: workload
          mountPath: /workload
      volumes:
      - name: workload
        configMap:
          name: workload-generator
      restartPolicy: Never
  backoffLimit: 0
EOF

# Wait for metrics to be collected
sleep 120

# Check Prometheus metrics
kubectl port-forward svc/prometheus-kube-prometheus-prometheus -n monitoring 9090:9090 &
PROM_PID=$!
sleep 5

# Run a series of queries to validate functionality
# Similar to the phase 2-3 validation scripts, checking:
# - Coverage score
# - Cardinality reduction
# - Control loop functioning
# - Resource usage

kill $PROM_PID

# Clean up
kubectl delete job workload-generator -n monitoring
kubectl delete configmap workload-generator -n monitoring

echo "=== Kubernetes integration test PASSED ==="
```

### 4.2 Multi-Node Scenario Tests

Create multi-node test scenarios for Kubernetes:

```bash
# scenarios/high_cardinality.sh
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: high-cardinality-test
  namespace: monitoring
spec:
  template:
    spec:
      containers:
      - name: generator
        image: golang:1.22
        command: ["go", "run", "/workload/workload.go", "--processes", "3000", "--cardinality", "100000"]
        volumeMounts:
        - name: workload
          mountPath: /workload
      volumes:
      - name: workload
        configMap:
          name: workload-generator
      restartPolicy: Never
EOF

# scenarios/cpu_spike.sh
kubectl apply -f - <<EOF
apiVersion: batch/v1
kind: Job
metadata:
  name: cpu-spike-test
  namespace: monitoring
spec:
  template:
    spec:
      containers:
      - name: stress
        image: polinux/stress
        command: ["stress", "--cpu", "8", "--timeout", "300"]
      restartPolicy: Never
EOF

# scenarios/memory_pressure.sh
# Similar to CPU spike but with --vm and --vm-bytes flags
```

## 5. Performance & Chaos Testing

### 5.1 Performance Benchmarks

Create detailed benchmarks for each component:

```bash
# test/benchmark/processors_bench.go
package benchmark

import (
    "context"
    "testing"
    
    "github.com/yourorg/sa-omf/api/updateable"
    // other imports
)

func BenchmarkPriorityTagger(b *testing.B) {
    // Benchmark with varying numbers of processes
    for _, numProcesses := range []int{100, 1000, 10000} {
        b.Run(fmt.Sprintf("Processes-%d", numProcesses), func(b *testing.B) {
            metrics := generateBenchmarkMetrics(numProcesses)
            processor := setupPriorityTagger()
            
            b.ResetTimer()
            for i := 0; i < b.N; i++ {
                processor.ConsumeMetrics(context.Background(), metrics)
            }
        })
    }
}

func BenchmarkAdaptiveTopK(b *testing.B) {
    // Similar benchmark with varying K values and process counts
}

func BenchmarkPIDDecider(b *testing.B) {
    // Benchmark decision making with varying KPI deltas
}

func BenchmarkPICControl(b *testing.B) {
    // Benchmark patch application with varying rates
}
```

### 5.2 Resource Consumption Tests

```bash
#!/bin/bash
# resource_consumption.sh

echo "=== Testing resource consumption under load ==="

# Start collector with resource monitoring
cd test-environments/bare
docker-compose up -d

# Start resource monitoring
docker stats sa-omf-collector --format "{{.CPUPerc}},{{.MemUsage}}" > resource_stats.csv &
STATS_PID=$!

# Generate increasing load
for processes in 100 500 1000 2000 5000; do
  echo "Testing with $processes processes"
  go run ../../test/generator/workload.go --processes $processes --duration 2m
  echo "$processes" >> process_counts.csv
  sleep 10
done

# Stop monitoring
kill $STATS_PID

# Analyze results
echo "Plotting results..."
go run ../../test/analysis/plot_resources.go \
  --stats resource_stats.csv \
  --counts process_counts.csv \
  --output resource_scaling.png

echo "=== Resource consumption test complete ==="
```

### 5.3 Chaos Testing Suite

```bash
# test/chaos/chaos_suite.go
package main

import (
    "flag"
    "fmt"
    "log"
    "os"
    "time"
    // other imports
)

func main() {
    var (
        environment = flag.String("env", "docker", "Test environment: docker or k8s")
        duration = flag.Duration("duration", 30*time.Minute, "Test duration")
    )
    flag.Parse()
    
    scenarios := []scenario{
        {"config_oscillation", runConfigOscillation},
        {"process_explosion", runProcessExplosion},
        {"cardinality_bomb", runCardinalityBomb},
        {"resource_starvation", runResourceStarvation},
        {"network_partition", runNetworkPartition},
        {"out_of_memory", runOutOfMemory},
    }
    
    for _, s := range scenarios {
        log.Printf("Running chaos scenario: %s", s.name)
        err := s.run(*environment, *duration)
        if err != nil {
            log.Printf("Scenario failed: %v", err)
        } else {
            log.Printf("Scenario passed")
        }
    }
}

type scenario struct {
    name string
    run  func(string, time.Duration) error
}

func runConfigOscillation(env string, duration time.Duration) error {
    // Rapidly toggle between two configurations
    // Verify system stabilizes despite contradictory signals
    return nil
}

// Other scenario implementations
```

## 6. Continuous Validation Pipeline

### 6.1 CI/CD Pipeline Configuration

```yaml
# .github/workflows/validation.yml
name: Continuous Validation

on:
  push:
    branches: [ main ]
  pull_request:
    branches: [ main ]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      - name: Run unit tests
        run: go test -v ./...

  component-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.22'
      - name: Test Core Interfaces
        run: go test -v ./test/interfaces/...
      - name: Test Algorithms
        run: go test -v ./test/alg/...
      - name: Test Processors
        run: go test -v ./test/processors/...

  integration-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Start test environment
        run: |
          cd test-environments/prometheus
          docker-compose up -d
      - name: Run basic integration tests
        run: bash ./test/integration/basic_test.sh
      - name: Generate test workload
        run: |
          go run ./test/generator/workload.go --processes 500 --duration 2m
      - name: Validate metrics and adaptation
        run: bash ./test/integration/validate_metrics.sh

  performance-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run benchmarks
        run: go test -bench=. -benchmem ./test/benchmark/...
      - name: Check resource usage
        run: bash ./test/performance/resource_test.sh

  build-and-deploy:
    runs-on: ubuntu-latest
    needs: [unit-tests, component-tests, integration-test, performance-test]
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v3
      - name: Build and push Docker image
        uses: docker/build-push-action@v4
        with:
          push: true
          tags: yourorg/sa-omf-collector:latest
      - name: Trigger K8s deployment test
        run: curl -X POST https://your-deployment-webhook.com
```

### 6.2 Automated Test Report Generator

```go
// test/report/generate_report.go
package main

import (
    "flag"
    "fmt"
    "html/template"
    "os"
    "path/filepath"
    // other imports
)

func main() {
    var (
        outputDir = flag.String("output", "test-report", "Output directory")
        testResults = flag.String("results", "test-results.json", "Test results JSON file")
        benchmarks = flag.String("benchmarks", "benchmark-results.json", "Benchmark results JSON file")
        metrics = flag.String("metrics", "metrics-report.json", "Metrics report JSON file")
    )
    flag.Parse()
    
    // Create report structure
    report := Report{
        // Load data from input files
    }
    
    // Generate HTML report
    if err := generateHTMLReport(*outputDir, report); err != nil {
        fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
        os.Exit(1)
    }
}

type Report struct {
    TestResults TestResults
    Benchmarks BenchmarkResults
    Metrics MetricsReport
}

func generateHTMLReport(outputDir string, report Report) error {
    // Create directory
    if err := os.MkdirAll(outputDir, 0755); err != nil {
        return err
    }
    
    // Generate index.html
    tmpl, err := template.New("index").Parse(indexTemplate)
    if err != nil {
        return err
    }
    
    f, err := os.Create(filepath.Join(outputDir, "index.html"))
    if err != nil {
        return err
    }
    defer f.Close()
    
    return tmpl.Execute(f, report)
}

const indexTemplate = `
<!DOCTYPE html>
<html>
<head>
    <title>SA-OMF Test Report</title>
    <style>
        /* CSS styles */
    </style>
</head>
<body>
    <h1>SA-OMF Test Report</h1>
    <!-- Template content -->
</body>
</html>
`
```

### 6.3 Regression Test Automation

```bash
#!/bin/bash
# regression_test.sh

set -e

echo "=== Running regression tests ==="

# Store current version metrics as baseline
if [[ ! -d "baselines" ]]; then
  mkdir -p baselines
fi

VERSION=$(git describe --tags)
BASELINE_DIR="baselines/$VERSION"
mkdir -p "$BASELINE_DIR"

# Run standard test suite
echo "Collecting baseline metrics for $VERSION"
make test

# Run performance baselines
go test -bench=. -benchmem ./test/benchmark/... > "$BASELINE_DIR/benchmark.txt"

# Run resource usage tests
bash ./test/performance/resource_test.sh
cp test-environments/bare/resource_stats.csv "$BASELINE_DIR/"

# Compare with previous version if available
PREV_VERSION=$(ls baselines | sort -V | tail -n 2 | head -n 1)
if [[ -n "$PREV_VERSION" && "$PREV_VERSION" != "$VERSION" ]]; then
  echo "Comparing with previous version: $PREV_VERSION"
  
  # Compare benchmarks
  go run ./test/analysis/compare_benchmarks.go \
    --baseline "baselines/$PREV_VERSION/benchmark.txt" \
    --current "$BASELINE_DIR/benchmark.txt" \
    --threshold 10 # Allow 10% regression
  
  # Compare resource usage
  go run ./test/analysis/compare_resources.go \
    --baseline "baselines/$PREV_VERSION/resource_stats.csv" \
    --current "$BASELINE_DIR/resource_stats.csv" \
    --threshold 15 # Allow 15% regression
fi

echo "=== Regression test complete ==="
```

## End-to-End Validation Master Script

This script ties everything together and should be run after every significant change:

```bash
#!/bin/bash
# validate_all.sh

set -e

echo "=== SA-OMF Comprehensive Validation ==="
echo "Starting at $(date)"

# Phase 1: Core testing
echo "=== Phase 1: Core Testing ==="
go test -v ./...
go test -v ./test/interfaces/...
go test -v ./test/alg/...

# Phase 2: Component testing
echo "=== Phase 2: Component Testing ==="
go test -v ./test/processors/...
go test -v ./test/extension/...

# Phase 3: Integration testing
echo "=== Phase 3: Integration Testing ==="
bash test/integration/basic_test.sh
bash test/integration/control_loop_test.sh

# Phase 4: Performance testing
echo "=== Phase 4: Performance Testing ==="
go test -bench=. -benchmem ./test/benchmark/...
bash test/performance/resource_test.sh

# Phase 5: Chaos testing (if enabled)
if [[ "$ENABLE_CHAOS" == "true" ]]; then
  echo "=== Phase 5: Chaos Testing ==="
  go run test/chaos/chaos_suite.go --env docker --duration 10m
fi

# Phase 6: Deployment testing
echo "=== Phase 6: Deployment Testing ==="
bash test/deployment/install_and_test.sh

# Phase 7: Regression testing
echo "=== Phase 7: Regression Testing ==="
bash test/regression_test.sh

# Generate report
echo "=== Generating Test Report ==="
go run test/report/generate_report.go

echo "=== Validation Complete at $(date) ==="
echo "See test-report/index.html for details"
```

This comprehensive testing framework ensures that every aspect of the SA-OMF system is thoroughly validated at each step of implementation. By following this approach, issues can be caught early, and the system will maintain high quality throughout its development lifecycle.
