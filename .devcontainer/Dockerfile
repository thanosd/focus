# trunk-ignore-all(checkov/CKV_DOCKER_2)
# trunk-ignore-all(hadolint/DL3008): Dev container uses latest package versions
# trunk-ignore-all(checkov/CKV2_DOCKER_1): Dev container requires sudo for vscode user
FROM golang:1.26-bookworm

RUN apt-get update && rm -rf /var/lib/apt/lists/*

# Install Node.js 22 (LTS)
SHELL ["/bin/bash", "-o", "pipefail", "-c"]
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/*

# Install PostgreSQL client
RUN apt-get update && apt-get install -y --no-install-recommends \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*

# Set up workspace
WORKDIR /workspace

# Create vscode user
RUN useradd -m -s /bin/bash vscode

# Configure vscode user with sudo access
RUN apt-get update \
    && apt-get install -y --no-install-recommends sudo \
    && echo vscode ALL=\(root\) NOPASSWD:ALL > /etc/sudoers.d/vscode \
    && chmod 0440 /etc/sudoers.d/vscode \
    && rm -rf /var/lib/apt/lists/*

USER vscode
