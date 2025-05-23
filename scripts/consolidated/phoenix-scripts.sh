#!/bin/bash

# Phoenix Scripts - Master Index and Execution Script
# Provides easy access to all Phoenix scripts with categorization and help

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Script base directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Function to display header
show_header() {
    echo -e "${BLUE}🚀 Phoenix Scripts Manager${NC}"
    echo -e "${BLUE}=========================${NC}"
    echo ""
}

# Function to show category
show_category() {
    local category="$1"
    local description="$2"
    local icon="$3"
    
    echo -e "${PURPLE}$icon $category${NC} - $description"
}

# Function to show script
show_script() {
    local script="$1"
    local description="$2"
    local category="$3"
    
    echo "  ${script} - $description"
}

# Function to run script
run_script() {
    local category="$1"
    local script="$2"
    local script_path="$SCRIPT_DIR/$category/$script"
    
    if [ -f "$script_path" ]; then
        echo -e "${GREEN}🔄 Running: $category/$script${NC}"
        echo ""
        cd "$PROJECT_ROOT"
        "$script_path" "$@"
    else
        echo -e "${RED}❌ Script not found: $script_path${NC}"
        exit 1
    fi
}

# Function to list scripts in category
list_category() {
    local category="$1"
    local category_path="$SCRIPT_DIR/$category"
    
    if [ -d "$category_path" ]; then
        echo -e "${CYAN}Scripts in $category:${NC}"
        for script in "$category_path"/*.sh; do
            if [ -f "$script" ]; then
                basename "$script"
            fi
        done
    else
        echo -e "${RED}Category $category not found${NC}"
    fi
}

# Function to show help
show_help() {
    show_header
    
    echo -e "${YELLOW}Usage:${NC}"
    echo "  ./phoenix-scripts.sh [category] [script] [args...]"
    echo "  ./phoenix-scripts.sh list [category]"
    echo "  ./phoenix-scripts.sh help"
    echo ""
    
    echo -e "${YELLOW}Categories:${NC}"
    echo ""
    
    show_category "core" "Essential system operations" "📦"
    show_script "run-phoenix.sh" "Main system startup/shutdown script"
    show_script "initialize-environment.sh" "Environment setup and initialization"
    echo ""
    
    show_category "deployment" "System deployment and configuration" "🚀"
    show_script "deploy.sh" "Deployment orchestration script"
    show_script "generate_certs.sh" "TLS certificate generation"
    show_script "test-deployment.sh" "Deployment validation"
    echo ""
    
    show_category "testing" "Testing and validation scripts" "🧪"
    show_script "verify-services.sh" "Service availability verification"
    show_script "verify-apis.sh" "API endpoint testing"
    show_script "verify-configs.sh" "Configuration validation"
    show_script "full-verification.sh" "Complete system verification"
    show_script "test_core_functionality.sh" "Core functionality integration tests"
    show_script "functional-test.sh" "Functional testing suite"
    show_script "api-test.sh" "API testing utilities"
    echo ""
    
    show_category "monitoring" "System monitoring and health" "📊"
    show_script "health_check_aggregator.sh" "Health check aggregation"
    show_script "validate-system.sh" "System validation checks"
    echo ""
    
    show_category "maintenance" "System maintenance and operations" "🔧"
    show_script "cleanup.sh" "System cleanup operations"
    show_script "backup_phoenix_data.sh" "Data backup utilities"
    show_script "restore_phoenix_data.sh" "Data restoration utilities"
    echo ""
    
    show_category "utils" "General purpose utilities" "🛠️"
    show_script "show-docs.sh" "Documentation display utilities"
    show_script "project-summary.sh" "Project summary generation"
    show_script "phoenix-metric-generator.sh" "Metric generation utilities"
    show_script "newrelic-integration.sh" "New Relic integration utilities"
    echo ""
    
    show_category "legacy" "Legacy and migration scripts" "📁"
    show_script "control-loop-enhanced.sh" "Enhanced control loop (legacy)"
    show_script "update-control-file.sh" "Control file updates (legacy)"
    show_script "migrate-to-monorepo.sh" "Migration utilities (legacy)"
    echo ""
    
    echo -e "${YELLOW}Examples:${NC}"
    echo "  ./phoenix-scripts.sh core run-phoenix.sh"
    echo "  ./phoenix-scripts.sh testing full-verification.sh"
    echo "  ./phoenix-scripts.sh core initialize-environment.sh"
    echo "  ./phoenix-scripts.sh list testing"
    echo "  ./phoenix-scripts.sh maintenance cleanup.sh"
    echo ""
    
    echo -e "${YELLOW}Quick Commands:${NC}"
    echo "  ./phoenix-scripts.sh start     # Start Phoenix system"
    echo "  ./phoenix-scripts.sh test      # Run full verification"
    echo "  ./phoenix-scripts.sh init      # Initialize environment"
    echo "  ./phoenix-scripts.sh deploy    # Deploy system"
    echo "  ./phoenix-scripts.sh clean     # Clean system"
    echo "  ./phoenix-scripts.sh health    # Check system health"
}

# Function to handle quick commands
handle_quick_command() {
    case "$1" in
        "start")
            run_script "core" "run-phoenix.sh" "${@:2}"
            ;;
        "test")
            run_script "testing" "full-verification.sh" "${@:2}"
            ;;
        "init")
            run_script "core" "initialize-environment.sh" "${@:2}"
            ;;
        "deploy")
            run_script "deployment" "deploy.sh" "${@:2}"
            ;;
        "clean")
            run_script "maintenance" "cleanup.sh" "${@:2}"
            ;;
        "health")
            run_script "monitoring" "health_check_aggregator.sh" "${@:2}"
            ;;
        *)
            return 1
            ;;
    esac
}

# Main execution logic
main() {
    # Check if we're in the Phoenix project directory
    if [ ! -f "$PROJECT_ROOT/CLAUDE.md" ]; then
        echo -e "${RED}❌ Not in Phoenix project directory${NC}"
        echo "Please run this script from the Phoenix project root or scripts directory"
        exit 1
    fi
    
    case "$1" in
        "help"|"-h"|"--help"|"")
            show_help
            ;;
        "list")
            if [ -n "$2" ]; then
                list_category "$2"
            else
                echo -e "${YELLOW}Available categories:${NC}"
                ls -1 "$SCRIPT_DIR" | grep -v "phoenix-scripts.sh\|README.md"
            fi
            ;;
        *)
            # Try quick command first
            if handle_quick_command "$@"; then
                exit 0
            fi
            
            # Handle category/script execution
            if [ -n "$1" ] && [ -n "$2" ]; then
                run_script "$1" "$2" "${@:3}"
            else
                echo -e "${RED}❌ Invalid usage${NC}"
                echo "Use './phoenix-scripts.sh help' for usage information"
                exit 1
            fi
            ;;
    esac
}

# Execute main function with all arguments
main "$@"