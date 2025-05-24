# Phoenix Observability Platform – Process-Metrics MVP PRD (v1.4 - Expanded)

**Document ID:** PHX-MVP-PROC-1.4  
**Status:** Draft (For Final Review)  
**Date:** 2024-05-24  
**Primary Author:** [Your Name/Team]  
**Stakeholders:** Platform SRE, Observability Guild, Product Management, Infrastructure Team, New Relic Product Team (for NRDOT evolution)

| Section | Purpose |
|---------|---------|
| 1. Scope & Vision | Defines the specific problem this MVP solves for process metrics, its value, and what's deliberately excluded. |
| 2. Functional Goals & KPIs | Sets clear, measurable objectives for the MVP's success regarding process metric optimization. |
| 3. Architecture Snapshot | Provides a simplified diagram of the MVP components and their interactions for process metrics. |
| 4. Detailed Requirements | Exhaustive breakdown of functional needs for CLI, Web Console, Control Plane, OTel Pipelines, Load Sim, Back-end & Infra. |
| 5. Acceptance Test Matrix | Defines specific, measurable, and automatable tests to gate the MVP release. |
| 6. Delivery Plan (13 weeks) | Outlines sprint-level milestones, ownership, and success criteria for focused MVP delivery. |
| 7. Risks & Mitigations | Identifies potential challenges specific to the process metrics MVP and proposes countermeasures. |
| 8. Glossary (MVP Context) | Defines key terms as they apply to this process-metrics focused MVP. |

## 1. Scope & Vision

### 1.1 Vision (MVP)

Empower New Relic users to slash host-process metric spend by ≥40% through intelligent, pre-validated OpenTelemetry pipelines, while preserving 100% visibility of critical processes—deployable in under 10 minutes per host, with optimization impact observable in under 5 minutes, and easily reversible.

This MVP specifically targets the optimization of process metrics ingested into New Relic Infrastructure (via NRDOT OTLP endpoint), providing immediate cost relief and operational clarity without the complexity of a service mesh.

### 1.2 Problem Statement

**Uncontrolled Process Metrics Cardinality & Cost:** Standard `hostmetrics` receiver in OpenTelemetry often captures thousands of per-process time series from each host. This "collect everything" approach leads to:
- Rapidly escalating New Relic ingest costs, making comprehensive process monitoring prohibitively expensive.
- Performance degradation in observability backends due to high cardinality.
- Alert fatigue from noisy, irrelevant process fluctuations.

**Lack of Safe & Standardized Optimization:** Engineers lack:
- Proven, ready-to-use configurations for intelligently filtering or aggregating process metrics.
- A safe, isolated method to A/B test different optimization strategies on live hosts without impacting production monitoring or data integrity.
- Standardized benchmarks to quantify the impact (cost, visibility, performance) of these strategies.

**Opaque Spend & Value:** It's difficult to:
- Attribute process monitoring costs to specific hosts, services, or even types of processes.
- Demonstrate the value of optimizations in concrete dollar terms or retained critical visibility.

### 1.3 Out-of-Scope (MVP)

| Area | Defers to Post-MVP Phase | Rationale for MVP Exclusion |
|------|--------------------------|----------------------------|
| Traces, Logs, or Non-Process System Metrics | PHX-v2 (Full Telemetry Optimization) | Focus on high-pain, high-cardinality process metrics first. |
| Istio / Service-Mesh based Traffic Routing | PHX-v2 (Advanced Experimentation) | Simplifies MVP deployment; A/B on same host via dual collectors. |
| Global Multi-Region Control & Data Residency | PHX-v3 (Enterprise Scale) | MVP targets single-region EKS or standalone VMs. |
| ML-Powered PII Redaction/Filtering | PHX-v2 (Advanced Compliance) | Process metrics typically have less PII than traces/logs. |
| Automated Budget Enforcement by Cost Governor | PHX-v2 (Proactive Cost Control) | MVP focuses on visibility of cost & ingest for processes. |
| Advanced PID/ML Adaptive Sampling | PHX-v2 (Dynamic Optimization) | MVP uses simpler threshold/rule-based adaptive logic. |
| Full Phoenix Kubernetes Operators for CRDs | PHX-v1.5 (Operational Maturity) | MVP uses simplified operator/scripts for PhoenixProcessPipeline CRD. |
| Comprehensive Web Console UI for Configuration | PHX-v1.5 (Enhanced DX) | MVP Web Console is primarily for monitoring & results display. |
| Automated Layer Generation Framework | PHX-v2 (Platform Scalability) | Process pipelines are manually curated for MVP catalog. |
| Multi-Tenancy in Phoenix Control Plane | PHX-v3 (Enterprise Scale) | MVP assumes a single operational tenant/team. |
| Audit Service for Control Plane Actions | PHX-v2 (Security & Compliance) | Basic logging sufficient for MVP control actions. |

## 2. Functional Goals & KPIs

| ID | Goal | Metric (How to Measure Success) | Target (MVP) |
|----|----|--------------------------------|--------------|
| G-1 | Rapid Deployment of Optimized Process Pipelines | Time from `phoenix pipeline deploy` to first optimized process metrics visible in New Relic. | ≤ 10 min/host |
| G-2 | Safe & Simple A/B Comparison of Process Strategies | Time from `phoenix experiment run` to first comparison dashboard view (cardinality, critical processes). | ≤ 60 min |
| G-3 | Significant Process Metric Cardinality Reduction | Percentage reduction in unique process metric time series count (vs. baseline-v1) for a given load profile. | ≥ 50% |
| G-4 | Guaranteed Critical Process Visibility | Percentage of pre-defined critical processes consistently reported to New Relic despite optimizations. | 100% |
| G-5 | Demonstrable Infrastructure Monitoring Cost Savings | Estimated percentage reduction in New Relic ingest cost attributed to process metrics (vs. baseline-v1). | ≥ 40% |
| G-6 | Low Collector Performance Overhead | Additional CPU & Memory utilization by the Phoenix OTel Collector on the monitored host. | < 5% per core/GB |
| G-7 | Effective Load Simulation for Pipeline Validation | Ability of the load simulator to generate defined process load profiles (Realistic, High-Cardinality, High-Churn). | Qualitative Pass |

## 3. Architecture Snapshot (Process-Metrics MVP)

```
+---------------------------------+      +---------------------------------+
|      Developer Interface        |      |      Phoenix Control Plane      |
|  [Phoenix CLI]                  |<----->|  [Experiment Controller (MVP)]  |
|  [Web Console (Basic Monitor)]  |      |  [Pipeline Deployer (MVP)]    |
+---------------------------------+      |  [Cost/Ingest Benchmarker]    |
                                         +-----------------|-----------------+
                                                           | (kubectl apply PhoenixProcessPipeline CRD,
                                                           |  K8s API for collector deployment/config)
                                         +-----------------V-----------------+
                                         |      Infrastructure Layer         |
                                         |  [AWS EKS (Single Region)] or   |
                                         |  [AWS VMs (Standalone)]         |
                                         +-----------------|-----------------+
                                                           | (Deploys/Configures)
+-----------------------------------------------------------------------------------------+
| Target Host (EKS Node / VM)                                                             |
|                                                                                         |
|  +-------------------------+     +--------------------------+     +-------------------+ |
|  | OTel Collector Variant A|<----|(Optional) Load Simulator |     | Host Processes    | |
|  | (Config: Pipeline A)    |     | (Generates process load) |---->| (nginx, java, etc)| |
|  |  - hostmetrics receiver |     +--------------------------+     |                   | |
|  |  - process_pipeline_A   |                                      +---------^---------+ |
|  |  - newrelic_exporter_A  |                                                |           |
|  |  - prom_exporter_A      |     +--------------------------+               |           |
|  +-------------------------+     | OTel Collector Variant B |               | (scrape)  |
|                                  | (Config: Pipeline B)     |<--------------+           |
|  (For A/B Experiments on        |  - hostmetrics receiver  |                           |
|   the same host, processing     |  - process_pipeline_B    |                           |
|   the same raw hostmetrics)     |  - newrelic_exporter_B   |                           |
|                                  |  - prom_exporter_B       |                           |
|                                  +------------^-------------+                           |
+----------------------------------------------|------------------------------------------+
                                               | (OTLP: Optimized Process Metrics)
                                               |
                         +---------------------V---------------------+     +---------------------+
                         |      Observability Backend (Phoenix)      |     | Observability Backend |
                         |  [Prometheus (for Phoenix metrics)]       |<--->|  [New Relic (NRDOT)]  |
                         |  [Grafana (for Phoenix dashboards)]     |     |  (Customer Data)    |
                         +-----------------------------------------+     +---------------------+
```

**Notes on MVP Architecture:**

- **No Istio/Service Mesh:** A/B experiments on the same host are achieved by running two OTel Collector instances (or a single collector capable of parallel internal processing paths, though two instances are simpler for MVP), each processing the same raw hostmetrics data but applying different processor chains (Pipeline A vs. Pipeline B). They export with distinct attributes to differentiate data in backends.

- **Control Plane (MVP):**
  - **Experiment Controller (MVP):** Manages the lifecycle of these dual collector deployments for A/B tests.
  - **Pipeline Deployer (MVP):** Handles deploying single collector configurations. Could be a simple K8s operator watching a new `PhoenixProcessPipeline` CRD or CLI-driven scripts for MVP.
  - **Cost/Ingest Benchmarker:** A component (or set of queries/scripts) that pulls data from Prometheus/New Relic to calculate KPIs.

- **OTel Collectors:** Deployed as DaemonSet on EKS or as a service on VMs.
- **Load Simulator:** Runs as a pod on EKS or a process on VMs, directly on the target host being benchmarked to generate realistic process activity.

## 4. Detailed Functional Requirements

### 4.1 Developer Interface Layer (DI)

#### 4.1.1 Phoenix CLI (`phoenix`)

**FR-DI-01: Process Pipeline Management Commands**

- `phoenix pipeline list --type process`: Lists available process metric optimization pipelines from the catalog.
  - Output includes: Name, Version, Brief Description, Expected Cardinality Reduction %.

- `phoenix pipeline show <catalog_pipeline_name>`: Displays the full YAML configuration of a specific catalog process pipeline.

- `phoenix pipeline deploy <catalog_pipeline_name> --target-host <host_identifier_or_k8s_node_selector> [--crd-name <custom_crd_name>]`: Deploys/Updates a process pipeline configuration to the OTel Collector(s) on the specified target(s).
  - `--target-host`: Can be a VM IP/hostname, or a K8s node label selector (e.g., `kubernetes.io/hostname=node1`).
  - `--crd-name`: (For K8s) Name of the PhoenixProcessPipeline CR to create/update. If not provided, a default name is generated.
  - Generates/Applies a `PhoenixProcessPipeline` CRD instance if on K8s, or directly updates collector config on VMs (via SSH/CM tool for MVP, manual for barebones MVP).

- `phoenix pipeline status [--target-host <host_identifier>] [--crd-name <custom_crd_name>]`: Shows deployment status, running config version, and key ingest metrics (input process count, output series count, critical processes seen) for the pipeline on target(s).

- `phoenix pipeline validate <process_pipeline_config.yaml>`: Validates a local process pipeline YAML against the OTel spec and Phoenix best practices for process metrics.
  - Checks for `hostmetrics` receiver with process scraper.
  - Verifies known processors from the catalog are configured correctly.
  - Flags potentially high-cardinality configurations if not using optimization processors.

- `phoenix pipeline get-config <deployed_crd_name> --target-host <host_identifier>`: Fetches the currently active pipeline configuration YAML from a deployed collector.

**Acceptance Criteria (FR-DI-01):**
- CLI commands provide clear feedback (success, error messages, progress).
- `deploy` results in the target OTel Collector(s) actively using the new configuration within 5 minutes.
- `status` provides actionable insights into pipeline health and effectiveness.
- JSON output mode (`--output json`) available for scripting.

**FR-DI-02: Process Experiment Management Commands**

- `phoenix experiment create --scenario <process_experiment_scenario.yaml>`: Creates an experiment definition from a YAML file specifying two PhoenixProcessPipeline configurations (variants) and comparison metrics.
  - Scenario YAML includes: `experimentName`, `duration`, `targetHost`, `variantA_pipelineNameOrPath`, `variantB_pipelineNameOrPath`, `criticalProcessRegex` (for retention check), `comparisonMetrics` (e.g., cardinalityReduction, cpuOverhead).

- `phoenix experiment run <experiment_name>`: Starts the defined experiment on the specified `targetHost`. This involves deploying two OTel Collector instances (or configuring one for parallel processing) on that host, each with a variant's configuration, both reading local hostmetrics.

- `phoenix experiment status <experiment_name>`: Shows real-time (or near real-time via Prometheus queries) comparison of specified metrics between variants (e.g., input processes, output series, CPU/mem usage of collectors, critical processes retained).

- `phoenix experiment compare <experiment_name>`: Provides a summary report (CLI table, JSON) at the end of the experiment, including statistical significance if basic t-tests/p-values are implemented for MVP.

- `phoenix experiment promote <experiment_name> --variant <A|B>`: Makes the chosen variant's pipeline configuration the active one on the `targetHost` and cleans up the other variant's collector.

- `phoenix experiment stop <experiment_name>`: Stops the experiment and cleans up all experiment-specific collector instances/configurations, reverting to the pre-experiment pipeline if one was active.

**Acceptance Criteria (FR-DI-02):**
- A/B experiments for process pipelines can be fully managed via CLI.
- Comparison metrics clearly show the difference in cardinality reduction and critical process retention.
- Promotion or stop actions cleanly transition the host's monitoring configuration.

**FR-DI-03: Process Load Simulation Commands**

- `phoenix loadsim start --profile <realistic|high-cardinality|high-churn> --target-host <host_identifier> [--duration <time_string>] [--process-count <int>] [--churn-rate <float_ops/sec>]`: Starts a process load simulation profile on the target host.
  - `--process-count`: Overrides default process count for the profile.
  - `--churn-rate`: Overrides default churn rate for the profile.

- `phoenix loadsim stop [--target-host <host_identifier>]`: Stops all active load simulations on the target host.

- `phoenix loadsim list-profiles`: Displays details of available load simulation profiles.

**Acceptance Criteria (FR-DI-03):**
- Simulator accurately generates defined process loads (verifiable with `ps`, `top` on target host).
- Can be run concurrently with A/B experiments to test pipeline behavior under load.
- Starts and stops cleanly without leaving orphaned simulation processes.

#### 4.1.2 Web Console (Basic Monitoring & Results Display for MVP)

**FR-DI-04 (Web Console - Deployed Process Pipelines View):**

- **Display:** A sortable, filterable table listing all hosts where Phoenix-managed OTel collectors are deployed for process metrics.
- **Details per Host/Pipeline Instance:**
  - Hostname / K8s Node Name.
  - Active PhoenixProcessPipeline name and version.
  - Status (Running, Error, Stale Config).
  - Key Performance Indicators (sourced from Prometheus, updated ~1 min):
    - Input Raw Process Count (from hostmetrics before processing).
    - Output Process Metric Time Series Count (exported to New Relic).
    - Calculated Cardinality Reduction Percentage.
    - Critical Processes Retained (count / percentage based on a configurable list).
    - Collector CPU & Memory Usage.
  - **Actions:** Link to detailed Grafana dashboard for that host/pipeline.

**Acceptance Criteria (FR-DI-04):**
- Provides at-a-glance overview of process pipeline effectiveness across the fleet.
- Data is accurate and reflects current collector state and performance.

**FR-DI-05 (Web Console - Process Experiment Dashboard):**

- **Display:** A list of active and recently completed process metric A/B experiments.
- **Details per Experiment:**
  - Experiment Name, Target Host, Duration, Status (Running, Completed, Failed).
  - Variant A Pipeline Name, Variant B Pipeline Name.
  - Side-by-side real-time charts (embedding Grafana panels or simple charts) for:
    - Input Raw Process Count (should be identical for both variants on the same host).
    - Output Process Metric Time Series Count (Variant A vs. Variant B).
    - Cardinality Reduction % (Variant A vs. Variant B).
    - Critical Processes Retained % (Variant A vs. Variant B).
    - Collector CPU Usage (Variant A vs. Variant B).
  - Summary table with final comparison metrics upon completion.
  - **Actions:** Link to detailed Grafana comparison dashboard. Option to trigger `phoenix experiment promote/stop` via backend API call (CLI is primary interaction for MVP).

**Acceptance Criteria (FR-DI-05):**
- Clearly visualizes the performance trade-offs between process pipeline variants.
- Helps users make informed decisions about promoting a variant.

### 4.2 Orchestration & Control Plane (CP - Simplified for Process Metrics MVP)

**FR-CP-01 (Experiment Controller - Process A/B Execution on Single Host):**

- **Input:** An experiment definition (from CLI) specifying `targetHost` and two process pipeline configurations (e.g., `variantA_pipeline: process-baseline-v1`, `variantB_pipeline: process-topk-v1`).
- **Action:**
  - If deploying on EKS, for the `targetHost` (node):
    - Ensures two distinct OTel Collector Deployments (or Pods if simpler for MVP) are created, labeled for Variant A and Variant B.
    - Each collector Deployment gets its respective PhoenixProcessPipeline configuration mounted as a ConfigMap.
    - Both collectors are configured to scrape local hostmetrics (i.e., from the node they are running on).
    - Each collector's exporter (to New Relic and Prometheus) adds a distinguishing attribute (e.g., `phoenix.variant: A`).
  - If deploying on a VM:
    - (Simplified for MVP) The controller might output two collector config files and instructions for the user to run two collector processes manually on the VM, or it uses SSH to manage two systemd services.
- **Output:** Two collectors running in parallel on the `targetHost`, processing the same raw process data stream with different optimization strategies.

**Acceptance Criteria (FR-CP-01):**
- Two collector instances are verifiably running on the target host with their respective configurations.
- Both collectors successfully scrape local process metrics.
- Exported metrics to New Relic and Prometheus contain the `phoenix.variant` attribute.
- Resources (collectors, ConfigMaps) are cleanly removed when the experiment is stopped or a variant is promoted.

**FR-CP-02 (Pipeline Deployment - PhoenixProcessPipeline CRD & Operator):**

**CRD Definition (PhoenixProcessPipeline - Simplified for MVP):**

```yaml
apiVersion: phoenix.newrelic.com/v1alpha1 # Example group
kind: PhoenixProcessPipeline
metadata:
  name: my-host-process-config
  namespace: phoenix-collectors # Or target application namespace
spec:
  # Selector for K8s nodes this pipeline applies to (if deployed via DaemonSet/Operator)
  # For VMs, this CRD might be conceptual and translated to config files by CLI/scripts.
  nodeSelector:
    kubernetes.io/hostname: "specific-node-1" 
    # or: environment: "production", role: "webserver"
  
  # Reference to a pipeline configuration from the catalog, or inline definition
  pipelineCatalogRef: "process-topk-v1" # Name of a predefined catalog pipeline
  # OR
  # inlineConfig: |
  #   receivers:
  #     hostmetrics: ...
  #   processors: ...
  #   exporters: ...
  #   service: ...

  # Variables to pass to the collector config (e.g., for exporter endpoints, API keys)
  # These would be sourced from K8s Secrets or a central Phoenix ConfigMap
  configVariables:
    NEW_RELIC_API_KEY_SECRET_NAME: "nr-api-key-secret"
    NEW_RELIC_OTLP_ENDPOINT: "https://otlp.nr-data.net:4317"
    PROMETHEUS_REMOTE_WRITE_ENDPOINT: "http://prometheus.phoenix-observability:9090/api/v1/write"

  # MVP: Basic resource requests for the collector pod(s)
  resources:
    requests:
      cpu: "100m"
      memory: "256Mi"
    limits:
      cpu: "500m"
      memory: "512Mi"
status: # Updated by the operator
  observedGeneration: 1
  conditions:
    - type: Deployed
      status: "True"
      lastTransitionTime: "2024-05-24T10:00:00Z"
      reason: "DeploymentSuccessful"
      message: "OTel Collector deployed and configured successfully."
  activeConfigVersion: "process-topk-v1" # Or a hash of inlineConfig
  observedProcessCountInput: 150 # Example
  observedTimeSeriesOutput: 30  # Example
  criticalProcessesRetained: "10/10" # Example (10 out of 10 defined criticals)
```

**Operator/Deployer Logic:**
- Watches `PhoenixProcessPipeline` CRs.
- For each CR, identifies target K8s nodes (via `spec.nodeSelector`) or VM targets (if applicable for MVP).
- Fetches the pipeline YAML from the catalog (if `pipelineCatalogRef` is used) or uses `inlineConfig`.
- Creates a ConfigMap containing the final OTel collector YAML, injecting `configVariables`.
- Deploys/Updates an OTel Collector DaemonSet (targeting selected nodes) or individual Deployments/Pods on EKS, mounting the ConfigMap and setting resources. For VMs, this step might involve SSH + config update + service restart (or manual execution guided by CLI output for a lean MVP).
- Updates the CR `status` field with deployment status and key performance indicators pulled from Prometheus.

**Acceptance Criteria (FR-CP-02):**
- Applying a `PhoenixProcessPipeline` CR results in OTel collectors running the specified process pipeline configuration on the targeted nodes/VMs.
- Changes to the CR (e.g., switching `pipelineCatalogRef`) trigger a rolling update of the collectors.
- `status` field is accurately updated by the operator/deployer.

**FR-CP-03 (Cost & Ingest Benchmarking Component):**

A scheduled job or persistent service that:
- Periodically queries Prometheus for OTel collector metrics:
  - `otelcol_receiver_accepted_metric_points{receiver="hostmetrics",หน่วย="process"}` (or similar for input process data points).
  - `otelcol_exporter_sent_metric_points{exporter="otlphttp/newrelic", pipeline_variant="X"}` (output series count).
  - Metrics indicating presence of critical processes (e.g., a custom metric `phoenix_critical_process_seen_count{process_name="nginx"}`).
  - `container_cpu_usage_seconds_total`, `container_memory_working_set_bytes` for collector pods.
- Calculates: Cardinality Reduction %, Critical Process Retention %, Collector Resource Overhead.
- Estimates cost impact based on New Relic ingest pricing for metric data points (DPMs) or custom metrics (a simplified model for MVP, e.g., $X per million DPMs).
- Writes these calculated benchmarks back to Prometheus as new time series (e.g., `phoenix_pipeline_cardinality_reduction_percent`, `phoenix_pipeline_estimated_cost_per_hour`).

**Acceptance Criteria (FR-CP-03):**
- Benchmark metrics are available in Prometheus and can be visualized in Grafana and the Web Console.
- Calculations accurately reflect the difference between input process metrics and optimized output.
- Cost estimation provides a reasonable indicator of ingest savings.

### 4.3 OTel Stack & Ecosystem - Process Metrics Pipelines (PL)

**FR-PL-01 (OTel Collector - Core Configuration for Process Metrics):**

- **Receiver:** `hostmetrics` with process scraper enabled.
  - `collection_interval`: Configurable, default 10s.
  - `root_path`: `/hostfs` for containerized collectors on EKS.
  - Metrics enabled: `process.cpu.time`, `process.cpu.utilization`, `process.memory.physical_usage` (aliased from rss), `process.memory.virtual_usage` (aliased from vms), `process.disk.io.read_bytes`, `process.disk.io.write_bytes`, `process.threads`, `process.open_file_descriptors`.
  - `process.command`, `process.command_line`, `process.executable.name`, `process.executable.path`, `process.owner`, `process.parent_pid`, `process.pid` attributes are collected.

- **Processors (Common to all pipelines):**
  - `memory_limiter`: Default limits (e.g., 80% limit, 20% spike).
  - `batch`: Configured for efficiency (e.g., `send_batch_size: 1000`, `timeout: 5s`).
  - `resourcedetection`: Detects `host.name`, `os.type`, K8s attributes (`k8s.pod.name`, `k8s.node.name` for collector pod itself), EC2 instance ID, etc.
  - `cumulativetodelta`: For `process.cpu.time` and `process.disk.io.*`.

- **Exporters:**
  - `otlphttp/newrelic`: Target NRDOT endpoint, API key from secret/env.
  - `prometheus`: For collector self-metrics and pipeline benchmark metrics (port 8888).
  - (Optional for experiments) `logging` exporter for debugging specific variant outputs.

**Acceptance Criteria (FR-PL-01):**
- Collectors start and run stably with the base configuration.
- All specified process metrics and attributes are collected from the host.
- Resource attributes are correctly added to exported metrics.

**FR-PL-02 (Validated Process Metrics Pipeline Catalog - MVP Set):**

- **Stored Location:** `/pipelines/catalog/process/` as `PhoenixProcessPipeline` CRD YAML files or raw OTel Collector YAMLs that the `PhoenixProcessPipeline` CR can reference.

- **Pipelines:**

1. **process-baseline-v1.yaml:**
   - **Strategy:** No optimization, collect all process metrics.
   - **Processors:** `memory_limiter`, `cumulativetodelta`, `resourcedetection`, `batch` (small batch size, e.g., 100, for near real-time).
   - **Purpose:** Control group for comparison; full fidelity but high cost.

2. **process-priority-based-v1.yaml:**
   - **Strategy:** Filter based on process importance (critical, high, low).
   - **Processors:** Adds `transform/process_classifier` (sets `attributes["process.priority"]` based on regex match of `process.executable.name` - e.g., `nginx`, `java`, `postgres` = critical; `systemd`, `sshd`, `kubelet` = high; others = low) and `filter/priority_filter` (drops low priority metrics, or heavily samples them).
   - **Critical Process Regex List:** Part of the pipeline config, e.g., `^(nginx|java|postgres|mysql|redis|kubelet|docker[d]?)$`.

3. **process-topk-v1.yaml:**
   - **Strategy:** Keep detailed metrics only for Top-K CPU/Memory consumers.
   - **Processors:** Adds `metricsgeneration` (to create `process.cpu.utilization_avg_1m` from `process.cpu.time`), `groupbyattrs` (to calculate per-process CPU/Mem), then `filter/top_k_filter` (keeps top 20 by CPU util, others dropped or aggregated).
   - **Note:** True Top-K is stateful and complex. MVP might use a simpler "high consumer filter": keep if CPU util > X% or Mem > Y MB.

4. **process-aggregated-v1.yaml:**
   - **Strategy:** Roll up metrics from common, low-priority user applications or numerous identical background processes.
   - **Processors:** Adds `transform/process_aggregator` (renames `process.executable.name` for specified patterns like `chrome`, `firefox`, `slack`, `code` to `common_user_app` or `developer_tool`) then `groupbyattrs` (sums metrics for these aggregated names).

5. **process-adaptive-filter-v1.yaml:**
   - **Strategy:** Simple adaptive filtering based on overall system state.
   - **Processors (Conceptual for MVP):**
   ```yaml
   # In processors list:
   filter/adaptive_system_load:
     metrics: # This applies to process metrics
       datapoint: # Drop datapoints if conditions met
         # If total host CPU > 80%, only keep critical/high priority processes
         - 'resource.attributes["host.cpu.utilization.total"] > 0.80 and attributes["process.priority"] == "low"'
         # If total process count > 500, only keep critical/high for low CPU users
         - 'resource.attributes["host.process.count"] > 500 and attributes["process.priority"] == "low" and attributes["process.cpu.utilization"] < 0.01'
   ```
   - This requires another processor (e.g., `metricsextract` or a custom one) to put `host.cpu.utilization.total` and `host.process.count` into resource attributes accessible by the filter processor. Simpler MVP might be a static filter that is manually switched based on perceived load.

**Acceptance Criteria (FR-PL-02):**
- All 5 catalog pipelines are syntactically valid and deployable.
- Each pipeline implements its described optimization logic.
- `process-priority-based-v1` correctly retains all metrics for processes matching its critical list.
- Performance and cardinality reduction characteristics are documented for each.

**FR-PL-03 (Built-in Process Load Simulator):**

- **Tool:** A Go application packaged as a container, deployable via `phoenix loadsim start`.
- **Functionality:** When run on a host, it generates specified process loads.
  - `realistic`: Spawns 50-200 child processes. Some are steady (e.g., mock web servers, databases), some have moderate CPU/memory spikes, and a few start/stop every minute.
  - `high-cardinality`: Spawns 1000-2000 mostly idle child processes, each with a unique command line argument to ensure distinct PIDs and `process.command_line` attributes. A few active processes generate baseline metrics.
  - `high-churn`: Continuously spawns and terminates 20-30 short-lived child processes per second (e.g., `sleep 0.1`, `echo test`).
- **Configuration:** Profiles define process names, resource usage patterns (CPU sinewave, memory ramps), and churn rates.
- **Metrics:** The simulator itself may expose metrics about generated load (e.g., `phoenix_loadsim_active_processes`, `phoenix_loadsim_churn_rate_ops_sec`).

**Acceptance Criteria (FR-PL-03):**
- Simulator can be deployed and controlled via Phoenix CLI on EKS nodes and VMs.
- Each profile generates observable and distinct process activity on the target host, verifiable with standard OS tools (`ps`, `top`, `htop`).
- The generated load is sufficient to test the different behaviors of the catalog pipelines.

### 4.4 Observability Backend (OB)

**FR-OB-01 (New Relic NRDOT OTLP Export - Process Metrics):**

- OTel Collectors configured with catalog pipelines successfully export optimized process metrics to the customer's New Relic account via OTLP endpoint.
- Metrics must map correctly to New Relic Infrastructure entities and attributes for processes.
  - Essential attributes: `processDisplayName`, `commandLine`, `hostname`, `entityGuid`, `processId`, `parentProcessId`.
  - Metrics: `cpuPercent`, `memoryResidentSizeBytes`, `ioReadBytesPerSecond`, `ioWriteBytesPerSecond`, etc.

**Acceptance Criteria (FR-OB-01):**
- Optimized process data is visible in New Relic Infrastructure UI, associated with the correct hosts.
- Critical process metrics are present and accurate.
- Aggregated process metrics (e.g., for `common_user_app`) appear as distinct entities or summarized under the host.
- Variant attribute (`phoenix.variant: A/B`) is visible if an experiment is running.

**FR-OB-02 (Prometheus - Collector & Pipeline Performance Metrics):**

- Phoenix OTel Collectors expose detailed operational and processing metrics via their `/metrics` endpoint (port 8888).
- **Key Prometheus Metrics to be Scraped:**
  - `otelcol_receiver_accepted_metric_points` (input from hostmetrics)
  - `otelcol_processor_refused_metric_points` (by optimization processors)
  - `otelcol_exporter_sent_metric_points` (output to New Relic)
  - `otelcol_process_cpu_seconds`, `otelcol_process_memory_rss` (collector's own resource usage)
  - Custom metrics from Benchmarking Component: `phoenix_pipeline_cardinality_reduction_percent`, `phoenix_pipeline_critical_processes_retained_percent`, `phoenix_pipeline_estimated_hourly_cost_usd`.
- A Prometheus instance in the EKS cluster (or accessible by VMs) scrapes these metrics. Basic federation setup if multiple Prometheus instances are needed (e.g., per experiment namespace, though MVP aims for simpler same-host A/B).

**Acceptance Criteria (FR-OB-02):**
- All listed metrics are available in Prometheus.
- Metrics are correctly labeled to identify the host, pipeline configuration, and experiment variant (if applicable).

**FR-OB-03 (Grafana Stack - Process Optimization Dashboards for MVP):**

A Grafana instance with Prometheus as a data source provides dashboards for:

- **Per-Host Process Pipeline Performance:**
  - Input raw process count vs. Output exported process metric series.
  - Cardinality Reduction %.
  - Critical Process Retention % (vs. a configurable list of critical process names/regex).
  - OTel Collector CPU/Memory usage.
  - Key metrics from the active catalog pipeline (e.g., `topk_kept_processes_count`, `aggregated_process_groups_count`).

- **A/B Experiment Comparison (Process Metrics):**
  - Side-by-side view of the above metrics for Variant A vs. Variant B.
  - Delta calculations for key comparison points.

**Acceptance Criteria (FR-OB-03):**
- Dashboards are provisioned automatically.
- Visualizations clearly show the effectiveness of different process pipeline configurations.
- Users can filter dashboards by host and/or experiment name.

### 4.5 Infrastructure Layer (IF - Simplified for MVP)

**FR-IF-01 (AWS EKS for Control Plane & Collector DaemonSet):**

- **Control Plane (MVP):** Experiment Controller, Pipeline Deployer (simplified operator or scripts), Cost/Ingest Benchmarker run as Deployments in a dedicated `phoenix-system` namespace on EKS.
- **OTel Collectors:** Deployed as a K8s DaemonSet in a `phoenix-collectors` namespace, ensuring one collector pod per node to monitor host-level process metrics. The active configuration for each collector pod is managed by the `PhoenixProcessPipeline` CR that targets that node (or all nodes if no specific selector).
- For A/B tests on a specific node, the Experiment Controller might temporarily deploy additional, specifically configured collector Pods on that node, or (more advanced) signal the DaemonSet pod on that node to run a second internal pipeline path. For MVP, deploying two distinct, labeled collector pods on the target node for the experiment duration is simpler.

**Acceptance Criteria (FR-IF-01):**
- All Phoenix MVP components deploy and run successfully on a single-region EKS cluster.
- Collectors in the DaemonSet correctly scrape process metrics from their respective nodes.
- Experiment-specific collector pods deploy to the correct target node for A/B testing.
- Basic K8s RBAC allows Phoenix components to manage their required resources (ConfigMaps, Pods, Deployments for collectors).

**FR-IF-02 (AWS VMs - Standalone OTel Collector Deployment):**

- Provide clear documentation and setup scripts (`install-phoenix-otel-vm.sh`) to install and configure the Phoenix-flavored OTel Collector on a standalone EC2 VM.
- The script should:
  - Install the OTel Collector binary.
  - Set up a systemd service for the collector.
  - Allow placement of a PhoenixProcessPipeline YAML (or raw OTel config YAML derived from it) to `/etc/otelcol/config.yaml`.
  - Configure the New Relic exporter API key via an environment file.

**Acceptance Criteria (FR-IF-02):**
- A user can successfully set up and run an optimized process metrics pipeline on an EC2 VM using the provided scripts and a catalog pipeline configuration.
- The VM-based collector exports data to New Relic and can be scraped by a central Prometheus (if network connectivity is configured).

## 5. Acceptance Test Matrix (MVP)

| Test ID | Scenario | Metric / Check | Pass Threshold | Automation Target |
|---------|----------|----------------|----------------|-------------------|
| AT-P01 | Deploy process-baseline-v1 to EKS node | Collector Ready; hostmetrics receiver active; metrics flowing to NR & Prom. | ≤ 10 min | GHA on KIND |
| AT-P02 | Deploy process-priority-based-v1 | Critical processes (nginx, java) metrics 100% retained in NR. | 100% | GHA on KIND |
| AT-P03 | Run LoadSim (High-Cardinality) on process-topk-v1 | Cardinality reduction in Prom (output vs input series) for that host. | ≥ 50% | GHA on KIND |
| AT-P04 | Run LoadSim (High-Churn) on process-adaptive-filter-v1 | Output series count varies inversely with total process count/CPU load. | Observable change | GHA on KIND |
| AT-P05 | A/B Test: baseline-v1 vs aggregated-v1 | `phoenix experiment compare` shows lower output series for aggregated-v1. | Output_Agg < Output_Base | GHA on KIND |
| AT-P06 | Critical Process Retention in A/B Test | Both variants in AT-P05 retain 100% of critical processes from target host. | 100% | GHA on KIND |
| AT-P07 | CLI: pipeline deploy with invalid YAML | CLI exits non-zero with clear validation error. | Yes | GHA CLI Test |
| AT-P08 | Web Console: View Deployed Pipelines | Table lists deployed pipelines from AT-P01 with correct status & reduction %. | UI displays data | Manual (MVP) |
| AT-P09 | Web Console: View A/B Experiment Results | Dashboard shows side-by-side comparison for AT-P05. | UI displays data | Manual (MVP) |
| AT-P10 | Collector Overhead Test | Collector CPU < 5% of 1 core, Mem < 100MiB on idle host with baseline-v1. | Pass | GHA Resource Chk |

## 6. Delivery Plan – 3 Sprints (Approx. 13 Weeks Including Sprint 0)

| Sprint | Duration | Key Epics & Focus | Primary Owner(s) | Key Exit Criteria / Deliverables |
|--------|----------|-------------------|------------------|----------------------------------|
| S-0 (Prep) | 1 week | Monorepo setup; EKS cluster provisioned; CI/CD basics; PhoenixProcessPipeline CRD definition; Contrib OTel base image. | Platform SRE/DevOps | Working EKS; Base OTel collector image buildable; CRD schema drafted. |
| S-1 (Core Pipe) | 2 weeks | Implement process-baseline-v1 & process-priority-based-v1; Basic CLI pipeline deploy/status for EKS (DaemonSet); Basic hostmetrics receiver setup; NR & Prom export. | Obs Guild | AT-P01, AT-P02 pass; Collectors deployable to EKS nodes. |
| S-2 (Adv Pipe) | 2 weeks | Implement process-topk-v1 & process-aggregated-v1; Load Simulator (Profiles: Realistic, High-Cardinality). | Obs Guild / DevTools | AT-P03 pass; Load simulator functional. |
| S-3 (Adaptive) | 2 weeks | Implement process-adaptive-filter-v1 (threshold-based); Load Simulator (Profile: High-Churn); Basic CLI loadsim commands. | Obs Guild / DevTools | AT-P04 pass; Adaptive logic demonstrable. |
| S-4 (Experiment) | 2 weeks | Experiment Controller (MVP for same-host A/B); CLI experiment create/run/status/compare/promote/stop; Benchmarking data collection to Prom. | ProdEng PM / CtrlPlane | AT-P05, AT-P06 pass; A/B experiments run end-to-end via CLI. |
| S-5 (UI & Docs) | 2 weeks | Basic Web Console views (Deployed Pipelines, Experiment Dashboard - Grafana embeds okay); Grafana dashboards for process metrics; CLI pipeline validate/get-config. | Platform Tools / Docs | AT-P07, AT-P08, AT-P09 pass (manual UI check); Basic user documentation for MVP. |
| S-6 (Polish & Test) | 2 weeks | Bug fixing; Full Acceptance Test Matrix automation; VM deployment scripts; Performance overhead testing; Finalize docs. | QE / All Teams | All ATs pass consistently; AT-P10 pass; Release candidate build. |

**GA Cut Criteria (MVP):** All Acceptance Tests (AT-P01 to AT-P10) passing in a staging-like environment for 7 consecutive days. Documentation reviewed and published. Key stakeholders sign off.

## 7. Risk Register (MVP Context)

| ID | Risk | Impact (L/M/H) | Likelihood (L/M/H) | Mitigation Strategy | Owner |
|----|------|----------------|--------------------|--------------------|-------|
| R-P1 | hostmetrics receiver PID namespace access issues in containers. | M | M | Use `hostPID: true` for collector DaemonSet pods on EKS. Document security implications. For VMs, ensure collector runs with sufficient privileges. | DevOps |
| R-P2 | Adaptive/Top-K filter logic too aggressive, drops critical processes. | H | M | Rigorous testing with `criticalProcessRegex` in Experiment CRD; default to conservative settings; clear per-pipeline documentation on guarantees. | Obs Guild |
| R-P3 | Prometheus load from many collectors or high-cardinality metrics. | M | L (for MVP scope) | MVP single Prometheus. Monitor Prom load closely. For future, consider Thanos or sharded Prometheus. Optimize metric labels. | Platform SRE |
| R-P4 | NRDOT OTLP endpoint/API key misconfiguration by users. | M | M | Clear CLI feedback; `phoenix pipeline validate` checks for presence of env vars/secrets; good default exporter config in catalog. | DevTools |
| R-P5 | Load Simulator doesn't accurately reflect real-world process churn. | M | M | Calibrate simulator profiles against actual production host samples. Allow users to define custom simulation process patterns post-MVP. | DevTools |
| R-P6 | Performance overhead of dual collectors for A/B on same host too high. | M | L | Monitor collector resource usage closely during experiments. Optimize collector configs for minimal footprint. Consider single-collector variant routing post-MVP. | Obs Guild |

## 8. Glossary (MVP Context - Process Metrics)

- **Process Metrics:** Telemetry data collected by the OpenTelemetry hostmetrics receiver's process scraper (e.g., CPU, memory, disk I/O per process).

- **Cardinality (Process Metrics):** The number of unique time series generated by process metrics. Primarily driven by (number of processes) x (number of metrics per process).

- **Critical Process:** A process deemed essential for system or application functionality, whose metrics must be retained with full fidelity. Defined by a configurable list of names or regex patterns.

- **NRDOT:** New Relic Distributed Tracing, used here as a shorthand for the New Relic OTLP ingest endpoint for infrastructure and telemetry data.

- **Pipeline (Process Metrics):** An OpenTelemetry Collector configuration specifically designed to receive, process (filter, aggregate, sample), and export host process metrics. Defined by a `PhoenixProcessPipeline` CRD in K8s or a corresponding YAML file.

- **Pipeline Catalog (Process Metrics):** A curated set of pre-validated `PhoenixProcessPipeline` configurations optimized for different process metric use cases and cost-reduction strategies.

- **Experiment (Process Metrics A/B):** A side-by-side comparison of two different `PhoenixProcessPipeline` configurations (variants) running on the same host, processing the same raw hostmetrics data, to evaluate their impact on cardinality, critical process retention, and collector performance.

- **Load Simulator (Process):** A tool that generates synthetic process activity (creation, termination, resource consumption) on a host to test pipeline behavior under various conditions.

---

*This expanded v1.4 PRD provides a much richer, actionable foundation for the Phoenix Process-Metrics MVP. It clarifies the "how" for many of the requirements and sets a clear path for development and validation.*