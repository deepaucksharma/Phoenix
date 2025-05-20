# Phoenix Documentation

Welcome to the Phoenix (SA-OMF) documentation. This directory contains comprehensive guides and reference material to help you understand, configure, and use Phoenix effectively.

## Core Documentation

| Document | Description |
|----------|-------------|
| [Quick Start Guide](quick-start.md) | Get up and running quickly |
| [Architecture Overview](architecture.md) | Understanding Phoenix's design |
| [Adaptive Processing](adaptive-processing.md) | How Phoenix adapts automatically |
| [PID Controllers](pid-controllers.md) | Details on the PID control implementation |
| [Configuration Reference](configuration-reference.md) | Complete configuration options |

## Architecture

The Phoenix architecture has evolved to a streamlined design where adaptation happens directly within processors. For details, see:

- [Architecture Overview](architecture.md) - Current architecture explanation
- [Architecture Decision Records](architecture/adr/) - Key design decisions and rationale

## Components

Phoenix includes several specialized components:

### Processors

- **adaptive_topk**: Dynamically adjusts k parameter to maintain coverage
- **others_rollup**: Aggregates low-priority resources to reduce cardinality
- **priority_tagger**: Tags resources with priority levels
- **adaptive_pid**: Provides monitoring and insights on system KPIs
- **cardinality_guardian**: Controls metrics cardinality

### Control Components

- **PID Controller**: The core feedback control algorithm
- **Safety Monitor**: Provides safeguards against resource exhaustion

## Additional Resources

- [Development Guide](development-guide.md) - Guide for Phoenix developers
- [Improvements](improvements/stability-improvements.md) - Recent improvements and stability fixes
- [Configuration Examples](../configs/) - Example configurations for different environments

## Getting Started

If you're new to Phoenix, we recommend the following reading order:

1. [Quick Start Guide](quick-start.md) - Get a working system quickly
2. [Architecture Overview](architecture.md) - Understand the design
3. [Adaptive Processing](adaptive-processing.md) - Learn the core concepts
4. [PID Controllers](pid-controllers.md) - Understand the control mechanism
5. [Configuration Reference](configuration-reference.md) - Configure for your use case

## Contributing to Documentation

To contribute improvements to this documentation:

1. Fork the repository
2. Make your changes
3. Submit a pull request with a clear description of the improvements

We welcome corrections, clarifications, and additions that make the documentation more useful.