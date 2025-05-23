# Phoenix-vNext Kubernetes GitOps Configuration

This directory contains Kubernetes manifests organized for GitOps deployment using Kustomize.

## Directory Structure

```
k8s/
├── base/                    # Base configuration shared across all environments
│   ├── kustomization.yaml
│   ├── namespace.yaml
│   ├── otel-collector-main.yaml
│   ├── otel-collector-observer.yaml
│   ├── control-actuator.yaml
│   ├── prometheus.yaml
│   ├── grafana.yaml
│   ├── analytics.yaml
│   └── ingress.yaml
└── overlays/               # Environment-specific configurations
    ├── dev/
    │   ├── kustomization.yaml
    │   ├── resource-limits.yaml
    │   └── ingress-patch.yaml
    ├── staging/
    │   └── kustomization.yaml
    └── production/
        ├── kustomization.yaml
        ├── resource-limits.yaml
        ├── security-patches.yaml
        └── monitoring-patches.yaml
```

## Deployment

### Prerequisites

1. Kubernetes cluster (1.24+)
2. kubectl configured
3. kustomize (or kubectl with built-in kustomize)
4. ArgoCD or Flux (for GitOps)

### Manual Deployment

```bash
# Deploy to development
kubectl apply -k k8s/overlays/dev/

# Deploy to production
kubectl apply -k k8s/overlays/production/

# Preview changes
kubectl kustomize k8s/overlays/production/
```

### GitOps with ArgoCD

1. Install ArgoCD:
```bash
kubectl create namespace argocd
kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
```

2. Create Application:
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: phoenix-vnext-prod
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/your-org/phoenix-vnext
    targetRevision: HEAD
    path: k8s/overlays/production
  destination:
    server: https://kubernetes.default.svc
    namespace: phoenix-vnext-prod
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
    - CreateNamespace=true
```

### GitOps with Flux

1. Install Flux:
```bash
flux bootstrap github \
  --owner=your-org \
  --repository=phoenix-vnext \
  --branch=main \
  --path=./k8s/flux \
  --personal
```

2. Create Kustomization:
```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
  name: phoenix-vnext-prod
  namespace: flux-system
spec:
  interval: 10m
  path: "./k8s/overlays/production"
  prune: true
  sourceRef:
    kind: GitRepository
    name: phoenix-vnext
```

## Environment Configuration

### Development
- Single replicas for all components
- Reduced resource limits
- No TLS enforcement
- Debug logging enabled
- Local hostnames

### Staging
- Similar to production but with:
  - Staging certificates
  - Lower resource allocations
  - Test data sources

### Production
- High availability (multiple replicas)
- Production resource limits
- Security hardening:
  - Non-root containers
  - Read-only root filesystems
  - Network policies
  - Pod disruption budgets
- TLS everywhere
- Production monitoring with ServiceMonitors
- Alerting rules

## Customization

### Adding New Components

1. Create component manifest in `base/`
2. Add to `base/kustomization.yaml` resources
3. Create environment-specific patches in overlays

### Modifying Resources

Use kustomize patches in overlays:

```yaml
# overlays/production/cpu-patch.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: otel-collector-main
spec:
  template:
    spec:
      containers:
      - name: otel-collector
        resources:
          limits:
            cpu: 4000m
```

### Secret Management

Secrets should be managed externally:

1. **Sealed Secrets**:
```bash
kubeseal --format=yaml < secret.yaml > sealed-secret.yaml
```

2. **External Secrets Operator**:
```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: newrelic-license
spec:
  secretStoreRef:
    name: vault-backend
    kind: SecretStore
  target:
    name: newrelic-license
  data:
  - secretKey: license-key
    remoteRef:
      key: phoenix/newrelic
      property: license-key
```

## Monitoring

Access services:
- Grafana: `https://grafana.phoenix.example.com`
- Prometheus: `https://prometheus.phoenix.example.com`
- Analytics: `https://analytics.phoenix.example.com`

## Troubleshooting

```bash
# Check pod status
kubectl get pods -n phoenix-vnext-prod

# View logs
kubectl logs -n phoenix-vnext-prod deployment/otel-collector-main

# Describe resources
kubectl describe -n phoenix-vnext-prod deployment/otel-collector-main

# Port forward for debugging
kubectl port-forward -n phoenix-vnext-prod svc/prometheus 9090:9090
```