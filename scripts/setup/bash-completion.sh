#!/bin/bash
# bash-completion.sh - Bash completion for Phoenix make targets and binaries

_phoenix_make_completion() {
  local cur prev opts
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  
  # Common make targets
  opts="build run test test-unit test-integration test-coverage clean lint verify"
  opts+=" fast-build fast-run benchmark docker docker-run release dev-setup help"
  
  # Role-specific targets
  opts+=" architect-check planner-check implementer-check reviewer-check"
  opts+=" security-auditor-check doc-writer-check devops-check integrator-check"
  
  # Handle CONFIG= syntax
  if [[ $prev == "CONFIG=" || $prev == "run" || $prev == "fast-run" || $prev == "run-bin" ]]; then
    # Complete config files
    local configs=$(find ./configs -type f -name "*.yaml" | sed 's|^\./||')
    COMPREPLY=( $(compgen -W "${configs}" -- ${cur}) )
    return 0
  fi
  
  # Handle VERSION= syntax
  if [[ $prev == "VERSION=" || $prev == "release" ]]; then
    # Suggest semantic versioning
    local versions="1.0.0 1.0.1 1.1.0 2.0.0"
    COMPREPLY=( $(compgen -W "${versions}" -- ${cur}) )
    return 0
  fi
  
  # Handle DOCKER_TAG= syntax
  if [[ $prev == "DOCKER_TAG=" || $prev == "docker" ]]; then
    local tags="latest dev staging production"
    COMPREPLY=( $(compgen -W "${tags}" -- ${cur}) )
    return 0
  fi
  
  # Complete make targets
  COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
  return 0
}

_phoenix_binary_completion() {
  local cur prev opts
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  
  # If previous word is the binary itself
  if [[ $prev == *"sa-omf-otelcol" ]]; then
    opts="--config --help --version"
    COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
    return 0
  fi
  
  # If previous word is --config
  if [[ $prev == "--config" ]]; then
    local configs=$(find ./configs -type f -name "*.yaml" | sed 's|^\./||')
    COMPREPLY=( $(compgen -W "${configs}" -- ${cur}) )
    return 0
  fi
  
  return 0
}

# Register completion functions
complete -F _phoenix_make_completion make
complete -F _phoenix_binary_completion ./bin/sa-omf-otelcol
complete -F _phoenix_binary_completion bin/sa-omf-otelcol

# Installation instructions - display when sourced
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  echo "This script should be sourced, not executed directly."
  echo "Add to your ~/.bashrc:"
  echo "source $(realpath ${BASH_SOURCE[0]})"
else
  echo "Phoenix bash completion loaded successfully."
fi