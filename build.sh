#!/bin/bash

# Script to check all requirements and build slsbench
# Exit code: 0 if build succeeds, 1 otherwise

set -e

ERRORS=0
WARNINGS=0
BINARY_NAME="slsbench"
BUILD_PATH="./cmd/slsbench"

echo "Checking prerequisites for slsbench..."
echo ""

# Check Go
echo -n "Checking Go... "
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    GO_MAJOR=$(echo $GO_VERSION | cut -d. -f1)
    GO_MINOR=$(echo $GO_VERSION | cut -d. -f2)
    
    if [ "$GO_MAJOR" -gt 1 ] || ([ "$GO_MAJOR" -eq 1 ] && [ "$GO_MINOR" -ge 24 ]); then
        echo "✓ Found Go $GO_VERSION"
    else
        echo "✗ Go version $GO_VERSION found, but Go 1.24+ is required"
        ERRORS=$((ERRORS + 1))
    fi
else
    echo "✗ Go is not installed"
    echo "  Install from: https://go.dev/doc/install"
    ERRORS=$((ERRORS + 1))
fi

# Check Git
echo -n "Checking Git... "
if command -v git &> /dev/null; then
    GIT_VERSION=$(git --version | awk '{print $3}')
    echo "✓ Found Git $GIT_VERSION"
else
    echo "✗ Git is not installed"
    echo "  Install from: https://git-scm.com/downloads"
    ERRORS=$((ERRORS + 1))
fi

# Check Docker
echo -n "Checking Docker... "
if command -v docker &> /dev/null; then
    if docker info &> /dev/null; then
        DOCKER_VERSION=$(docker --version | awk '{print $3}' | sed 's/,//')
        echo "✓ Found Docker $DOCKER_VERSION (running)"
    else
        echo "⚠ Docker is installed but not running"
        echo "  Please start Docker daemon"
        WARNINGS=$((WARNINGS + 1))
    fi
else
    echo "✗ Docker is not installed"
    echo "  Install from: https://docs.docker.com/get-docker/"
    ERRORS=$((ERRORS + 1))
fi

# Check Docker Compose
echo -n "Checking Docker Compose... "
if command -v docker-compose &> /dev/null; then
    COMPOSE_VERSION=$(docker-compose --version | awk '{print $3}' | sed 's/,//')
    echo "✓ Found Docker Compose $COMPOSE_VERSION"
elif docker compose version &> /dev/null; then
    COMPOSE_VERSION=$(docker compose version | awk '{print $4}')
    echo "✓ Found Docker Compose v2 $COMPOSE_VERSION (via 'docker compose')"
else
    echo "✗ Docker Compose is not installed"
    echo "  Docker Compose is usually included with Docker Desktop"
    echo "  Or install from: https://docs.docker.com/compose/install/"
    ERRORS=$((ERRORS + 1))
fi

echo ""

# Check if build path exists
if [ ! -d "$BUILD_PATH" ]; then
    echo "✗ Build path '$BUILD_PATH' does not exist"
    echo "  Make sure you're running this script from the repository root"
    ERRORS=$((ERRORS + 1))
fi

# Exit if there are errors
if [ $ERRORS -gt 0 ]; then
    echo "✗ Found $ERRORS error(s) and $WARNINGS warning(s). Please fix the errors before building."
    exit 1
fi

# Proceed with build if only warnings
if [ $WARNINGS -gt 0 ]; then
    echo "⚠ Some warnings detected, but proceeding with build..."
    echo ""
fi

# Build the binary
echo "Building $BINARY_NAME..."
if go build -o "$BINARY_NAME" "$BUILD_PATH"; then
    echo ""
    echo "✓ Build successful! Binary created: $BINARY_NAME"
    echo ""
    
    # Ask if user wants to move the binary to a bin directory
    read -p "Move $BINARY_NAME to a directory in your PATH? [y/N] " -n 1 -r
    echo ""
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        # Determine Go bin path
        GO_BIN_PATH="$HOME/go/bin"
        if [ -n "$GOPATH" ]; then
            GO_BIN_PATH="$GOPATH/bin"
        fi
        
        echo ""
        echo "Choose destination:"
        echo "  1) /usr/local/bin/ (system-wide, requires sudo)"
        echo "  2) $GO_BIN_PATH (user-specific)"
        echo "  3) Cancel"
        read -p "Enter choice [1-3]: " -n 1 -r
        echo ""
        
        case $REPLY in
            1)
                if sudo mv "$BINARY_NAME" /usr/local/bin/ 2>/dev/null; then
                    echo "✓ Moved $BINARY_NAME to /usr/local/bin/"
                else
                    echo "✗ Failed to move binary (may require sudo privileges)"
                    exit 1
                fi
                ;;
            2)
                # Create directory if it doesn't exist
                mkdir -p "$GO_BIN_PATH"
                if mv "$BINARY_NAME" "$GO_BIN_PATH/"; then
                    echo "✓ Moved $BINARY_NAME to $GO_BIN_PATH/"
                    echo ""
                    echo "Make sure $GO_BIN_PATH is in your PATH:"
                    echo "  export PATH=\$PATH:$GO_BIN_PATH"
                else
                    echo "✗ Failed to move binary"
                    exit 1
                fi
                ;;
            3)
                echo "Binary left in current directory: $BINARY_NAME"
                ;;
            *)
                echo "Invalid choice. Binary left in current directory: $BINARY_NAME"
                ;;
        esac
    else
        echo "Binary left in current directory: $BINARY_NAME"
    fi
    
    exit 0
else
    echo ""
    echo "✗ Build failed!"
    exit 1
fi

