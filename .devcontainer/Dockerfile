FROM mcr.microsoft.com/devcontainers/go:1.24

ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID

# Install additional packages
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
    jq \
    zsh \
    sudo \
    curl \
    htop \
    python3-pip \
    gnupg \
    lsb-release

# Install Docker CLI
RUN curl -fsSL https://download.docker.com/linux/debian/gpg | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg \
    && echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/debian $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null \
    && apt-get update \
    && apt-get -y install --no-install-recommends docker-ce-cli

# Install Go tools
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest \
    && go install github.com/golang/mock/mockgen@latest \
    && go install golang.org/x/vuln/cmd/govulncheck@latest \
    && go install github.com/cosmtrek/air@latest

# Install Python tools
RUN pip3 install pre-commit yamllint markdownlint-cli

# Configure shell for user
RUN usermod -s /bin/zsh $USERNAME \
    && echo "$USERNAME ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/$USERNAME \
    && chmod 0440 /etc/sudoers.d/$USERNAME

# Setup git-prompt for user
RUN curl -o /home/$USERNAME/.git-prompt.sh \
    https://raw.githubusercontent.com/git/git/master/contrib/completion/git-prompt.sh \
    && chown $USERNAME:$USERNAME /home/$USERNAME/.git-prompt.sh

# Clean up
RUN apt-get autoremove -y \
    && apt-get clean -y \
    && rm -rf /var/lib/apt/lists/*

ENV GOEXPERIMENT=loopvar

# Switch back to user vscode
USER $USERNAME

WORKDIR /workspace