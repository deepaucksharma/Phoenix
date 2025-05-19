# Custom .zshrc for Phoenix SA-OMF development container

# Enable Powerlevel10k instant prompt
if [[ -r "${XDG_CACHE_HOME:-$HOME/.cache}/p10k-instant-prompt-${(%):-%n}.zsh" ]]; then
  source "${XDG_CACHE_HOME:-$HOME/.cache}/p10k-instant-prompt-${(%):-%n}.zsh"
fi

# Path configuration
export PATH=$HOME/.local/bin:$HOME/go/bin:$PATH

# Set environment variables
export GOEXPERIMENT=loopvar
export EDITOR=code

# Aliases for project commands
alias mb="make build"
alias mt="make test"
alias mtu="make test-unit"
alias mti="make test-integration"
alias mr="make run"
alias ml="make lint"
alias mc="make clean"
alias mbm="make benchmark"
alias gcb="git checkout -b"
alias gst="git status"
alias gd="git diff"
alias gp="git pull"

# Function to run specific component tests
function testcomp() {
  go test -v ./test/processors/$1/...
}

# Function to create a new processor
function newproc() {
  scripts/dev/new-component.sh processor $1
}

# Function to create a new ADR
function newadr() {
  scripts/dev/new-adr.sh "$1"
}

# Function to create a new task
function newtask() {
  scripts/dev/create-task.sh "$1"
}

# Function to create a new branch
function newbranch() {
  scripts/dev/create-branch.sh "$1"
}

# Setup auto-completion
autoload -Uz compinit && compinit

# Source git-prompt if available
if [ -f ~/.git-prompt.sh ]; then
  source ~/.git-prompt.sh
  setopt PROMPT_SUBST
  export PS1='%F{cyan}%~%f%F{green}$(__git_ps1 " (%s)")%f %F{yellow}λ%f '
else
  export PS1='%F{cyan}%~%f %F{yellow}λ%f '
fi

# Welcome message with helpful commands
cat << 'EOF'
🔥 Phoenix SA-OMF Development Environment 🔥

Useful commands:
  mb      - make build
  mt      - make test
  mtu     - make test-unit
  mti     - make test-integration
  mr      - make run
  ml      - make lint
  mbm     - make benchmark
  
Helper functions:
  testcomp <name>      - Test specific component (e.g., testcomp priority_tagger)
  newproc <name>       - Create new processor
  newadr "title"       - Create new ADR
  newtask "desc"       - Create new task
  newbranch <name>     - Create new branch
  
Run `make help` for more commands.
EOF