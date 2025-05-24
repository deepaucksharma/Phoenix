#!/bin/bash
# Phoenix Complete System Runner
# Starts all services with proper configuration for core functionality

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║              Phoenix Adaptive Cardinality System                 ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════════╝${NC}"
echo

# Check prerequisites
check_prerequisites() {
    echo -e "${YELLOW}Checking prerequisites...${NC}"
    
    local missing=false
    
    # Check for Docker
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}✗ Docker not found${NC}"
        missing=true
    else
        echo -e "${GREEN}✓ Docker${NC}"
    fi
    
    # Check for docker-compose or docker compose
    if command -v docker-compose &> /dev/null; then
        COMPOSE_CMD="docker-compose"
        echo -e "${GREEN}✓ Docker Compose${NC}"
    elif docker compose version &> /dev/null; then
        COMPOSE_CMD="docker compose"
        echo -e "${GREEN}✓ Docker Compose (plugin)${NC}"
    else
        echo -e "${RED}✗ Docker Compose not found${NC}"
        missing=true
    fi
    
    # Check for Make
    if ! command -v make &> /dev/null; then
        echo -e "${RED}✗ Make not found${NC}"
        missing=true
    else
        echo -e "${GREEN}✓ Make${NC}"
    fi
    
    if [ "$missing" = true ]; then
        echo -e "\n${RED}Please install missing prerequisites before continuing.${NC}"
        exit 1
    fi
}

# Initialize environment
init_environment() {
    echo -e "\n${YELLOW}Initializing environment...${NC}"
    
    # Create necessary directories
    mkdir -p data/control data/prometheus data/grafana data/benchmark
    
    # Copy environment file if not exists
    if [ ! -f .env ]; then
        if [ -f configs/environments/dev/.env ]; then
            cp configs/environments/dev/.env .env
            echo -e "${GREEN}✓ Created .env from dev template${NC}"
        else
            echo -e "${RED}✗ No .env file found${NC}"
            exit 1
        fi
    else
        echo -e "${GREEN}✓ .env file exists${NC}"
    fi
    
    # Create control file template if not exists
    if [ ! -f configs/templates/control/optimization_mode_template.yaml ]; then
        cat > configs/templates/control/optimization_mode_template.yaml << 'EOF'
# Phoenix Control File Template
optimization_profile: "balanced"
config_version: 0
correlation_id: "init"
last_updated: "2025-01-01T00:00:00Z"
trigger_reason: "Initial configuration"
current_metrics:
  full_ts: 0
  optimized_ts: 0
  experimental_ts: 0
  cost_reduction_ratio: 0.0
  cardinality_explosion_alerts: 0
  cardinality_risk_processes: 0
thresholds:
  conservative_max_ts: 15000
  aggressive_min_ts: 25000
pipelines:
  experimental_enabled: false
last_profile_change_timestamp: "2025-01-01T00:00:00Z"
EOF
        echo -e "${GREEN}✓ Created control file template${NC}"
    fi
}

# Update configurations
update_configs() {
    echo -e "\n${YELLOW}Updating configurations...${NC}"
    
    # Update Prometheus config to include new rules
    if [ -f configs/monitoring/prometheus/rules/phoenix_core_rules.yml ]; then
        # Ensure Prometheus loads the new rules
        if ! grep -q "phoenix_core_rules.yml" configs/monitoring/prometheus/prometheus.yaml; then
            echo "    - '/etc/prometheus/rules/phoenix_core_rules.yml'" >> configs/monitoring/prometheus/prometheus.yaml
            echo -e "${GREEN}✓ Added core rules to Prometheus config${NC}"
        fi
    fi
    
    # Use enhanced observer if available
    if [ -f services/control-plane/observer/config/observer-enhanced.yaml ]; then
        cp services/control-plane/observer/config/observer-enhanced.yaml services/control-plane/observer/config/observer.yaml
        echo -e "${GREEN}✓ Using enhanced observer configuration${NC}"
    fi
}

# Build services
build_services() {
    echo -e "\n${YELLOW}Building services...${NC}"
    
    # Build updated services
    echo -e "${BLUE}Building services...${NC}"
    make build-docker || true
}

# Start services
start_services() {
    echo -e "\n${YELLOW}Starting Phoenix services...${NC}"
    
    # Use the modular docker-compose
    $COMPOSE_CMD -f infrastructure/docker/compose/base.yaml \
                 -f infrastructure/docker/compose/dev.yaml \
                 up -d
    
    echo -e "${GREEN}✓ Services started${NC}"
}

# Wait for services
wait_for_services() {
    echo -e "\n${YELLOW}Waiting for services to be ready...${NC}"
    
    local max_wait=60
    local waited=0
    
    while [ $waited -lt $max_wait ]; do
        if curl -s http://localhost:13133 > /dev/null 2>&1 && \
           curl -s http://localhost:13134 > /dev/null 2>&1; then
            echo -e "${GREEN}✓ Services are ready${NC}"
            return 0
        fi
        
        echo -n "."
        sleep 2
        ((waited+=2))
    done
    
    echo -e "\n${RED}✗ Services did not become ready in time${NC}"
    return 1
}

# Show status
show_status() {
    echo -e "\n${BLUE}═══════════════════════════════════════════════════════════════════${NC}"
    echo -e "${GREEN}Phoenix is running!${NC}"
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════════${NC}"
    echo
    echo -e "${YELLOW}Access Points:${NC}"
    echo -e "  • Grafana:      http://localhost:3000 (admin/admin)"
    echo -e "  • Prometheus:   http://localhost:9090"
    echo -e "  • OTLP Ingest:  localhost:4318"
    echo -e "  • Control API:  http://localhost:8080/api/v1"
    echo
    echo -e "${YELLOW}Useful Commands:${NC}"
    echo -e "  • View logs:    make logs"
    echo -e "  • Run tests:    ./tests/integration/test_core_functionality.sh"
    echo -e "  • Stop system:  make down"
    echo -e "  • Clean all:    make clean"
    echo
    echo -e "${BLUE}═══════════════════════════════════════════════════════════════════${NC}"
}

# Main execution
main() {
    check_prerequisites
    init_environment
    update_configs
    build_services
    start_services
    
    if wait_for_services; then
        show_status
        
        # Optional: Run tests
        read -p "Run integration tests? (y/N) " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            echo -e "\n${YELLOW}Running integration tests...${NC}"
            ./tests/integration/test_core_functionality.sh
        fi
    else
        echo -e "\n${RED}Failed to start Phoenix properly${NC}"
        echo "Check logs with: make logs"
        exit 1
    fi
}

# Handle arguments
case "${1:-}" in
    stop)
        echo -e "${YELLOW}Stopping Phoenix...${NC}"
        make down
        ;;
    clean)
        echo -e "${YELLOW}Cleaning Phoenix...${NC}"
        make clean
        ;;
    test)
        echo -e "${YELLOW}Running tests...${NC}"
        ./tests/integration/test_core_functionality.sh
        ;;
    *)
        main
        ;;
esac