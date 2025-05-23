#!/bin/bash

# Phoenix Monorepo Migration Script
# This script helps migrate from the old structure to the new monorepo structure

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Phoenix Monorepo Migration Script${NC}"
echo -e "${BLUE}===================================${NC}"
echo

# Check if we're in the right directory
if [ ! -f "docker-compose.yaml" ] || [ ! -d "apps" ]; then
    echo -e "${RED}Error: This doesn't appear to be a Phoenix project root directory${NC}"
    echo -e "${RED}Please run this script from the Phoenix project root${NC}"
    exit 1
fi

# Backup current state
echo -e "${YELLOW}Creating backup of current state...${NC}"
BACKUP_DIR="backup-$(date +%Y%m%d-%H%M%S)"
mkdir -p "$BACKUP_DIR"
cp -r apps configs scripts docker-compose.yaml "$BACKUP_DIR/" 2>/dev/null || true
echo -e "${GREEN}Backup created in $BACKUP_DIR${NC}"

# Check if monorepo structure already exists
if [ -d "packages" ] && [ -d "services" ] && [ -f "package.json" ]; then
    echo -e "${GREEN}Monorepo structure already exists!${NC}"
    echo -e "${YELLOW}Cleaning up old directories...${NC}"
    
    # Clean up old structure
    rm -rf apps modules 2>/dev/null || true
    rm -f docker-compose.yaml docker-compose.modular.yaml 2>/dev/null || true
    
    # Use new files
    mv -f README.monorepo.md README.md 2>/dev/null || true
    mv -f Makefile.monorepo Makefile 2>/dev/null || true
    mv -f .gitignore.monorepo .gitignore 2>/dev/null || true
    
    echo -e "${GREEN}Migration complete!${NC}"
else
    echo -e "${RED}Monorepo structure not found. Please run the refactoring first.${NC}"
    exit 1
fi

# Initialize git for new structure
echo -e "${YELLOW}Updating git...${NC}"
git add -A
git status

echo
echo -e "${GREEN}Migration completed successfully!${NC}"
echo
echo -e "${BLUE}Next steps:${NC}"
echo "1. Review the changes: git status"
echo "2. Commit the changes: git commit -m 'Migrate to monorepo structure'"
echo "3. Install dependencies: make install"
echo "4. Build services: make build"
echo "5. Deploy: make deploy-dev"
echo
echo -e "${YELLOW}Old structure backed up in: $BACKUP_DIR${NC}"
echo -e "${YELLOW}Documentation available at: docs/architecture/MONOREPO_STRUCTURE.md${NC}"

# Make scripts executable
chmod +x tools/scripts/*.sh 2>/dev/null || true
