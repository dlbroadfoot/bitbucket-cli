#!/bin/sh
# Bitbucket CLI installer
# Usage: curl -fsSL https://raw.githubusercontent.com/dlbroadfoot/bitbucket-cli/main/bb/script/install.sh | sh
set -e

REPO="dlbroadfoot/bitbucket-cli"
BINARY="bb"
INSTALL_DIR="/usr/local/bin"

# Detect OS and architecture
OS="$(uname -s)"
ARCH="$(uname -m)"

case "$OS" in
  Linux)   OS_NAME="linux" ;;
  Darwin)  OS_NAME="macOS" ;;
  MINGW*|MSYS*|CYGWIN*) OS_NAME="windows" ;;
  *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

case "$ARCH" in
  x86_64|amd64)  ARCH_NAME="amd64" ;;
  aarch64|arm64)  ARCH_NAME="arm64" ;;
  armv6l)         ARCH_NAME="armv6" ;;
  i386|i686)      ARCH_NAME="386" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Get latest version
VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/')
if [ -z "$VERSION" ]; then
  echo "Failed to determine latest version" >&2
  exit 1
fi

echo "Installing ${BINARY} v${VERSION} (${OS_NAME}/${ARCH_NAME})..."

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

if [ "$OS_NAME" = "macOS" ]; then
  # macOS uses .zip
  ASSET="${BINARY}_${VERSION}_${OS_NAME}_${ARCH_NAME}.zip"
  curl -fsSL "https://github.com/${REPO}/releases/download/v${VERSION}/${ASSET}" -o "${TMPDIR}/${ASSET}"
  unzip -q "${TMPDIR}/${ASSET}" -d "${TMPDIR}"
elif [ "$OS_NAME" = "windows" ]; then
  ASSET="${BINARY}_${VERSION}_${OS_NAME}_${ARCH_NAME}.zip"
  curl -fsSL "https://github.com/${REPO}/releases/download/v${VERSION}/${ASSET}" -o "${TMPDIR}/${ASSET}"
  unzip -q "${TMPDIR}/${ASSET}" -d "${TMPDIR}"
else
  # Linux uses .tar.gz
  ASSET="${BINARY}_${VERSION}_${OS_NAME}_${ARCH_NAME}.tar.gz"
  curl -fsSL "https://github.com/${REPO}/releases/download/v${VERSION}/${ASSET}" -o "${TMPDIR}/${ASSET}"
  tar -xzf "${TMPDIR}/${ASSET}" -C "${TMPDIR}"
fi

# Find the binary (goreleaser wraps in a subdirectory)
BINARY_PATH=$(find "$TMPDIR" -name "$BINARY" -type f -perm +111 | head -1)
if [ -z "$BINARY_PATH" ]; then
  echo "Failed to find ${BINARY} binary in archive" >&2
  exit 1
fi

# Install binary
if [ -w "$INSTALL_DIR" ]; then
  cp "$BINARY_PATH" "${INSTALL_DIR}/${BINARY}"
else
  echo "Need sudo to install to ${INSTALL_DIR}"
  sudo cp "$BINARY_PATH" "${INSTALL_DIR}/${BINARY}"
fi

chmod +x "${INSTALL_DIR}/${BINARY}"
echo "Installed ${BINARY} v${VERSION} to ${INSTALL_DIR}/${BINARY}"
