#!/bin/bash

# Phoenix Documentation Overview Script

# Colors
BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                    Phoenix Documentation                         ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════════╝${NC}"
echo

echo -e "${CYAN}📚 Documentation Structure:${NC}"
echo "   docs/"
echo "   ├── INDEX.md              # Documentation hub"
echo "   ├── ARCHITECTURE.md       # System design"
echo "   ├── MONOREPO_STRUCTURE.md # Project organization"
echo "   ├── PIPELINE_ANALYSIS.md  # Pipeline details"
echo "   ├── TROUBLESHOOTING.md    # Problem solving"
echo "   └── MIGRATION_GUIDE.md    # Migration help"
echo

echo -e "${CYAN}📄 Root Documentation:${NC}"
echo "   ├── README.md             # Project overview"
echo "   └── CLAUDE.md             # AI assistant guide"
echo

echo -e "${CYAN}📊 Documentation Stats:${NC}"
MD_COUNT=$(find docs -name "*.md" | wc -l)
TOTAL_LINES=$(find . -name "*.md" -not -path "./node_modules/*" -not -path "./backup-*/*" -exec wc -l {} + | tail -1 | awk '{print $1}')
echo "   • Documentation files: $MD_COUNT"
echo "   • Total lines: $TOTAL_LINES"
echo

echo -e "${CYAN}🔗 Quick Links:${NC}"
echo "   • View docs online: https://github.com/deepaucksharma/Phoenix/tree/main/docs"
echo "   • Serve locally: make docs-serve"
echo "   • Edit docs: cd docs/"
echo

echo -e "${GREEN}✅ Documentation is organized and ready!${NC}"
