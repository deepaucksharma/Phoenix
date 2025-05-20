# Architecture Documentation

This directory contains architectural documentation for the Phoenix project.

## Contents

- **adr/** - Architecture Decision Records
  - Contains formal records of significant architectural decisions made during the project
  - Each ADR explains the context, decision, and consequences of a key architectural choice

## Current Architecture Overview

Phoenix features a streamlined architecture with self-adaptive processors:

1. **Data Pipeline**:
   - Collects metrics from hostmetrics receiver
   - Processes through various adaptive processors
   - Exports metrics to configured destinations

2. **Self-Adaptive Components**:
   - Each processor implements internal self-adaptation
   - PID controllers are embedded within processors
   - Each processor monitors its own metrics and adjusts parameters

For a detailed explanation of the current architecture and its evolution, see [CURRENT_STATE.md](./CURRENT_STATE.md).

## Key Architectural Principles

1. **Self-Adaptation**: Components automatically adjust to changing conditions
2. **Feedback Control**: PID controllers provide stable parameter adjustments
3. **Safety Limits**: All adaptive behavior is constrained by configurable limits
4. **Observable Decisions**: All adaptation decisions are exposed as metrics

## Historical Context

The project has evolved from an earlier dual-pipeline design (see [ADR-001](./adr/001-dual-pipeline-architecture.md)) to a more streamlined approach with self-contained adaptive processors. This evolution has simplified the architecture while maintaining the core value proposition of adaptive processing.