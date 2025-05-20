# Architecture Documentation

This directory contains architectural documentation for the Phoenix project.

## Contents

- **adr/** - Architecture Decision Records
  - Contains formal records of significant architectural decisions made during the project
  - Each ADR explains the context, decision, and consequences of a key architectural choice

## Architecture Overview

Phoenix (SA-OMF) is built around a dual-pipeline architecture:

1. **Data Pipeline**:
   - Collects metrics from hostmetrics receiver
   - Processes through various adaptive processors
   - Exports metrics to configured destinations

2. **Control Pipeline**:
   - Monitors self-metrics
   - Evaluates KPIs against targets
   - Generates and applies configuration patches
   - Ensures system stays within operational bounds

See the ADRs for more detailed information on key architectural decisions.
