#!/bin/bash
# Setup Phoenix development environment

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Check Docker
check_docker() {
    log_info "Checking Docker..."
    if ! command -v docker &> /dev/null; then
        log_warn "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        log_warn "Docker daemon is not running. Please start Docker."
        exit 1
    fi
}

# Check Go
check_go() {
    log_info "Checking Go..."
    if ! command -v go &> /dev/null; then
        log_warn "Go is not installed. Please install Go 1.21+"
        exit 1
    fi
    
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    log_info "Go version: $GO_VERSION"
}

# Check Node.js
check_node() {
    log_info "Checking Node.js..."
    if ! command -v node &> /dev/null; then
        log_warn "Node.js is not installed. Please install Node.js 18+"
        exit 1
    fi
    
    NODE_VERSION=$(node --version)
    log_info "Node.js version: $NODE_VERSION"
}

# Install Go dependencies
install_go_deps() {
    log_info "Installing Go dependencies..."
    go mod download
    
    log_info "Installing Go tools..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
}

# Install Node dependencies
install_node_deps() {
    log_info "Installing Node.js dependencies..."
    cd dashboard
    npm ci
    cd ..
}

# Create local environment file
create_env_file() {
    log_info "Creating .env file..."
    if [ -f .env ]; then
        log_warn ".env file already exists, skipping..."
        return
    fi
    
    cat > .env <<EOF
# Phoenix Development Environment

# Database
DATABASE_URL=postgres://phoenix:phoenix@localhost:5432/phoenix?sslmode=disable

# JWT Secret (change in production!)
JWT_SECRET=development-secret-change-me-in-production

# Git Configuration
GIT_REPO_URL=https://github.com/phoenix/configs
GIT_TOKEN=

# New Relic
NEW_RELIC_API_KEY=
NEW_RELIC_ACCOUNT_ID=
NEW_RELIC_OTLP_ENDPOINT=https://otlp.nr-data.net

# API Configuration
GRPC_PORT=5050
HTTP_PORT=8080

# Dashboard
VITE_API_URL=http://localhost:8080/api/v1
EOF
    
    log_info "Created .env file - please update with your actual values"
}

# Setup pre-commit hooks
setup_git_hooks() {
    log_info "Setting up Git hooks..."
    
    cat > .git/hooks/pre-commit <<'EOF'
#!/bin/bash
# Pre-commit hook for Phoenix

# Run Go linters
echo "Running Go linters..."
golangci-lint run ./...

# Run Go tests
echo "Running Go tests..."
go test -short ./...

# Check dashboard
echo "Checking dashboard..."
cd dashboard
npm run lint
cd ..

echo "Pre-commit checks passed!"
EOF
    
    chmod +x .git/hooks/pre-commit
    log_info "Git hooks installed"
}

# Start development services
start_services() {
    log_info "Starting development services..."
    
    # Start only essential services
    docker-compose up -d postgres prometheus grafana
    
    log_info "Waiting for services to be ready..."
    sleep 10
    
    # Check services
    docker-compose ps
}

# Initialize database
init_database() {
    log_info "Initializing database..."
    
    # Wait for PostgreSQL to be ready
    until docker-compose exec -T postgres pg_isready -U phoenix; do
        log_info "Waiting for PostgreSQL..."
        sleep 2
    done
    
    # Run migrations (placeholder - implement actual migrations)
    log_info "Database ready"
}

# Build binaries
build_binaries() {
    log_info "Building binaries..."
    make build
}

# Print next steps
print_next_steps() {
    cat <<EOF

========================================
Development Environment Setup Complete!
========================================

Services running:
- PostgreSQL: localhost:5432
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3001 (admin/admin)

Next steps:

1. Update .env file with your configuration
2. Start the API server:
   go run cmd/api/main.go

3. Start the dashboard:
   cd dashboard && npm run dev

4. Access the dashboard:
   http://localhost:3000

5. To run tests:
   make test

6. To build Docker images:
   make docker

For more information, see docs/DEVELOPMENT.md

EOF
}

# Main
main() {
    log_info "Setting up Phoenix development environment..."
    
    check_docker
    check_go
    check_node
    
    install_go_deps
    install_node_deps
    
    create_env_file
    setup_git_hooks
    
    start_services
    init_database
    build_binaries
    
    print_next_steps
}

# Run main
main