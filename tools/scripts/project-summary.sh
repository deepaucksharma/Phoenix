#!/bin/bash

# Phoenix Project Summary Script

# Colors
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                    Phoenix Monorepo Summary                      ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════════╝${NC}"
echo

# Project Structure
echo -e "${CYAN}📁 Project Structure:${NC}"
echo "   phoenix/"
echo "   ├── packages/          # Shared libraries"
echo "   ├── services/          # Microservices"
echo "   ├── infrastructure/    # Docker & deployment configs"
echo "   ├── monitoring/        # Prometheus & Grafana"
echo "   ├── tools/            # Scripts and utilities"
echo "   ├── docs/             # Documentation"
echo "   └── config/           # Environment configs"
echo

# Services
echo -e "${CYAN}🚀 Services:${NC}"
echo "   • Collector         - Core OTEL metrics pipeline"
echo "   • Observer          - Control plane monitoring"
echo "   • Actuator          - Adaptive control logic"
echo "   • Synthetic Gen     - Process metrics generator"
echo "   • Complex Gen       - High-cardinality generator"
echo "   • Validator         - Performance benchmarking"
echo

# Key Features
echo -e "${CYAN}✨ Key Features:${NC}"
echo "   • 3-Pipeline Architecture (Full, Optimized, Experimental)"
echo "   • Adaptive Cardinality Control"
echo "   • Real-time Monitoring with Grafana"
echo "   • Modular Monorepo Structure"
echo "   • Environment-based Configuration"
echo

# Quick Commands
echo -e "${CYAN}⚡ Quick Commands:${NC}"
echo "   ${GREEN}make install${NC}      - Install dependencies"
echo "   ${GREEN}make build${NC}        - Build all services"
echo "   ${GREEN}make deploy-dev${NC}   - Deploy to development"
echo "   ${GREEN}make logs${NC}         - View service logs"
echo "   ${GREEN}make health${NC}       - Check service health"
echo "   ${GREEN}make monitor${NC}      - Open monitoring dashboards"
echo

# Access Points
echo -e "${CYAN}🌐 Access Points:${NC}"
echo "   • Grafana:    http://localhost:3000 (admin/admin)"
echo "   • Prometheus: http://localhost:9090"
echo "   • OTLP:       localhost:4318"
echo "   • Control API: http://localhost:8080/api/v1"
echo

# File Count Summary
echo -e "${CYAN}📊 Project Stats:${NC}"
TOTAL_FILES=$(find . -type f -not -path "./node_modules/*" -not -path "./.git/*" -not -path "./data/*" -not -path "./backup-*/*" | wc -l)
YAML_FILES=$(find . -name "*.yaml" -o -name "*.yml" -not -path "./node_modules/*" | wc -l)
GO_FILES=$(find . -name "*.go" -not -path "./node_modules/*" | wc -l)
JSON_FILES=$(find . -name "*.json" -not -path "./node_modules/*" | wc -l)

echo "   • Total Files:    $TOTAL_FILES"
echo "   • YAML Configs:   $YAML_FILES"
echo "   • Go Files:       $GO_FILES"
echo "   • JSON Files:     $JSON_FILES"
echo

echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}✅ Phoenix monorepo is ready for deployment!${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"
