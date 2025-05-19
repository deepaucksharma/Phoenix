#!/bin/bash
# welcome.sh - Display role-specific welcome instructions

source ~/.profile

# Detect current role from git config
ROLE=$(git config --get user.role 2>/dev/null || echo "unset")

# Set default terminal title
echo -e "\033]0;Phoenix - $ROLE\007"

echo "╔═════════════════════════════════════════════════════════╗"
echo "║                 Welcome to Phoenix                      ║"
echo "╚═════════════════════════════════════════════════════════╝"
echo ""

if [ "$ROLE" = "unset" ]; then
  echo "Please set your role to see role-specific instructions:"
  echo "  git config --global user.role <role>"
  echo ""
  echo "Available roles:"
  ls agents/ | sed 's/\.yaml//'
  echo ""
else
  echo "Current role: $ROLE"
  echo ""
  echo "Available commands for your role:"
  
  case "$ROLE" in
    architect)
      echo "  hack/new-adr.sh \"Title\"        - Create a new Architecture Decision Record"
      echo "  make architect-check            - Run validation checks for your role"
      ;;
    planner)
      echo "  hack/create-task.sh \"Title\"     - Create a new task"
      echo "  make planner-check              - Validate all tasks"
      ;;
    implementer)
      echo "  hack/create-branch.sh implementer <task-id> - Create a branch for your task"
      echo "  hack/new-component.sh <type> <name>         - Create a new component"
      echo "  make implementer-check                      - Run all checks for your code"
      ;;
    reviewer)
      echo "  make reviewer-check             - Run reviewer checks"
      ;;
    security-auditor)
      echo "  make security-auditor-check     - Run security checks"
      ;;
    doc-writer)
      echo "  make doc-writer-check           - Validate documentation"
      ;;
    devops)
      echo "  make devops-check               - Validate CI/CD configuration"
      ;;
    integrator)
      echo "  make integrator-check           - Run all checks before merging"
      ;;
    *)
      echo "  Unknown role: $ROLE"
      echo "  Please set a valid role with: git config --global user.role <role>"
      ;;
  esac
  
  echo ""
  echo "General commands:"
  echo "  make build                    - Build the collector"
  echo "  make test                     - Run tests"
  echo "  make lint                     - Run linting"
fi

echo ""
echo "For more information, see:"
echo "  AGENTS.md         - Role descriptions"
echo "  AGENT_RAILS.md    - Guidelines for agents"
echo "  CONTRIBUTING.md   - Contribution guidelines"
echo ""
