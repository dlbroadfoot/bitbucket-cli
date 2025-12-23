#!/bin/bash
set -e

echo ""
echo "🚀 Setting up Bitbucket CLI workspace..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check Go
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed."
    echo "   Install via: brew install go"
    exit 1
fi
echo "✓ Go $(go version)"

# Check make
if ! command -v make &> /dev/null; then
    echo "❌ Error: make is not installed."
    exit 1
fi
echo "✓ make available"

# Download dependencies
echo ""
echo "📦 Downloading Go modules..."
cd bb && go mod download && cd ..

# Verify build
echo ""
echo "🔨 Verifying build..."
cd bb && make build || go build -o bb ./cmd/bb && cd ..

echo ""
echo "✅ Setup complete! Run 'cd bb && make build' to compile."
echo ""
