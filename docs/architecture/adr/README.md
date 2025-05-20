# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for the SA-OMF project.

## What are ADRs?

Architecture Decision Records are documents that capture important architectural decisions made along with their context and consequences.

## ADR Index

- [ADR-001: Dual Pipeline Architecture](001-dual-pipeline-architecture.md)
- [ADR-002: Self-Regulating PID Control for Adaptive Processing](20250519-use-self-regulating-pid-control-for-adaptive-processing.md)

## Creating a New ADR

Use the script in the repository to create a new ADR:

```bash
scripts/dev/new-adr.sh "Title of the ADR"
```

This will create a new ADR with the correct format and naming convention. The
generated file will be placed in `docs/architecture/adr/`.

