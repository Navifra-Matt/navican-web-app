#!/bin/bash

# Install protoc in devcontainer
# This script installs protoc and required Go plugins
# Run WITHOUT sudo: bash scripts/install-protoc.sh

set -e

echo "Installing protoc and Go plugins..."

# Check if running in container
if [ ! -f /.dockerenv ] && [ ! -f /run/.containerenv ]; then
    echo "Warning: Not running in a container"
fi

# Install protoc if not already installed or if broken
if ! command -v protoc &> /dev/null || ! protoc --version &> /dev/null; then
    echo "Installing protoc..."

    # Detect architecture
    ARCH=$(uname -m)
    if [ "$ARCH" = "aarch64" ]; then
        PB_ARCH="aarch_64"
    elif [ "$ARCH" = "x86_64" ]; then
        PB_ARCH="x86_64"
    else
        echo "Unsupported architecture: $ARCH"
        exit 1
    fi

    # Download and install protoc
    PB_VERSION="28.3"
    PB_URL="https://github.com/protocolbuffers/protobuf/releases/download/v${PB_VERSION}/protoc-${PB_VERSION}-linux-${PB_ARCH}.zip"
    DOWNLOAD_DIR="${HOME}"

    echo "Downloading protoc ${PB_VERSION} for ${PB_ARCH}..."
    cd "${DOWNLOAD_DIR}"
    curl -LO "$PB_URL"

    echo "Installing to /usr/local... (requires sudo)"
    sudo unzip -o "protoc-${PB_VERSION}-linux-${PB_ARCH}.zip" -d /usr/local

    # Make sure it's executable
    sudo chmod +x /usr/local/bin/protoc

    # Clean up
    rm "protoc-${PB_VERSION}-linux-${PB_ARCH}.zip"

    echo "protoc installed successfully"
else
    echo "protoc is already installed: $(protoc --version)"
fi

# Install Go plugins (run as current user to get correct Go environment)
echo ""
echo "Installing Go protobuf plugins..."

# Check if Go is available
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed or not in PATH"
    exit 1
fi

go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Verify installation
echo ""
echo "Verifying installation..."
echo "protoc version: $(protoc --version)"
echo "protoc-gen-go: $(which protoc-gen-go || echo 'not found in PATH')"
echo "protoc-gen-go-grpc: $(which protoc-gen-go-grpc || echo 'not found in PATH')"

# Check if Go bin is in PATH
GOBIN=$(go env GOPATH)/bin
if [[ ":$PATH:" != *":$GOBIN:"* ]]; then
    echo ""
    echo "WARNING: $GOBIN is not in PATH"
    echo "Run this command to add it to your current session:"
    echo "  export PATH=\$PATH:$GOBIN"
    echo ""
    echo "To make it permanent, add this to your ~/.zshrc or ~/.bashrc:"
    echo "  export PATH=\$PATH:\$(go env GOPATH)/bin"
else
    echo ""
    echo "Go bin directory is in PATH ✓"
fi

echo ""
echo "Installation complete!"
