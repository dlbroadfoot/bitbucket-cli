#!/bin/sh
# Bitbucket CLI (bb) installer
# Usage: curl -fsSL https://raw.githubusercontent.com/dlbroadfoot/bitbucket-cli/main/install.sh | sh
#
# Works on macOS, Linux, and WSL.
# For Windows (native), use: irm https://raw.githubusercontent.com/dlbroadfoot/bitbucket-cli/main/install.ps1 | iex

set -e

REPO="dlbroadfoot/bitbucket-cli"
INSTALL_DIR="/usr/local/bin"

# Colors (if terminal supports them)
if [ -t 1 ]; then
  BOLD='\033[1m'
  GREEN='\033[0;32m'
  RED='\033[0;31m'
  RESET='\033[0m'
else
  BOLD=''
  GREEN=''
  RED=''
  RESET=''
fi

info() { printf "${BOLD}%s${RESET}\n" "$1"; }
success() { printf "${GREEN}%s${RESET}\n" "$1"; }
error() { printf "${RED}error: %s${RESET}\n" "$1" >&2; exit 1; }

# Detect OS
detect_os() {
  case "$(uname -s)" in
    Darwin)  echo "macOS" ;;
    Linux)   echo "linux" ;;
    MINGW*|MSYS*|CYGWIN*) error "Use the PowerShell installer on Windows: irm .../install.ps1 | iex" ;;
    *)       error "Unsupported operating system: $(uname -s)" ;;
  esac
}

# Detect architecture
detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64)  echo "amd64" ;;
    aarch64|arm64)  echo "arm64" ;;
    armv6l|armv7l)  echo "armv6" ;;
    i386|i686)      echo "386" ;;
    *)              error "Unsupported architecture: $(uname -m)" ;;
  esac
}

# Get latest release tag from GitHub
get_latest_version() {
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/'
  elif command -v wget >/dev/null 2>&1; then
    wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/'
  else
    error "curl or wget is required"
  fi
}

# Download a file
download() {
  url="$1"
  dest="$2"
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" -o "$dest"
  elif command -v wget >/dev/null 2>&1; then
    wget -q "$url" -O "$dest"
  fi
}

main() {
  OS=$(detect_os)
  ARCH=$(detect_arch)

  info "Installing Bitbucket CLI (bb)..."
  printf "  OS: %s  Arch: %s\n" "$OS" "$ARCH"

  # Get latest version
  info "Fetching latest release..."
  TAG=$(get_latest_version)
  VERSION="${TAG#v}"

  if [ -z "$VERSION" ]; then
    error "Could not determine latest version"
  fi

  printf "  Version: %s\n" "$VERSION"

  # Build download URL and determine format
  case "$OS" in
    macOS)
      # Use PKG installer for macOS (universal binary, installs to /usr/local/bin)
      FILENAME="bb_${VERSION}_macOS_universal.pkg"
      URL="https://github.com/${REPO}/releases/download/${TAG}/${FILENAME}"
      ;;
    linux)
      FILENAME="bb_${VERSION}_linux_${ARCH}.tar.gz"
      URL="https://github.com/${REPO}/releases/download/${TAG}/${FILENAME}"
      ;;
  esac

  # Create temp directory
  BB_TMPDIR=$(mktemp -d)
  trap 'rm -rf "$BB_TMPDIR"' EXIT

  info "Downloading ${FILENAME}..."
  download "$URL" "${BB_TMPDIR}/${FILENAME}"

  # Verify checksum
  info "Verifying checksum..."
  CHECKSUMS_URL="https://github.com/${REPO}/releases/download/${TAG}/checksums.txt"
  download "$CHECKSUMS_URL" "${BB_TMPDIR}/checksums.txt"

  EXPECTED_SHA=$(awk -v f="$FILENAME" '$2==f { print $1; exit }' "${BB_TMPDIR}/checksums.txt")
  [ -n "$EXPECTED_SHA" ] || error "Could not find checksum for ${FILENAME}"

  if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL_SHA=$(sha256sum "${BB_TMPDIR}/${FILENAME}" | awk '{print $1}')
  else
    ACTUAL_SHA=$(shasum -a 256 "${BB_TMPDIR}/${FILENAME}" | awk '{print $1}')
  fi

  [ "$ACTUAL_SHA" = "$EXPECTED_SHA" ] || error "Checksum mismatch for ${FILENAME}"

  # Install
  case "$OS" in
    macOS)
      info "Installing via macOS package installer..."
      sudo installer -pkg "${BB_TMPDIR}/${FILENAME}" -target /
      ;;
    linux)
      info "Extracting archive..."
      tar xzf "${BB_TMPDIR}/${FILENAME}" -C "$BB_TMPDIR"

      # Find the bb binary (it's inside a directory in the tarball)
      BB_BIN=$(find "$BB_TMPDIR" -name "bb" -type f | head -1)

      if [ -z "$BB_BIN" ]; then
        error "Could not find bb binary in archive"
      fi

      info "Installing to ${INSTALL_DIR}/bb..."
      if [ -w "$INSTALL_DIR" ]; then
        cp "$BB_BIN" "${INSTALL_DIR}/bb"
        chmod +x "${INSTALL_DIR}/bb"
      else
        sudo cp "$BB_BIN" "${INSTALL_DIR}/bb"
        sudo chmod +x "${INSTALL_DIR}/bb"
      fi
      ;;
  esac

  # Verify installation
  if command -v bb >/dev/null 2>&1; then
    success "Bitbucket CLI (bb) ${VERSION} installed successfully!"
    printf "\n"
    info "Get started:"
    printf "  bb auth login    # Authenticate with Bitbucket\n"
    printf "  bb --help        # See all commands\n"
  else
    success "Installed bb to ${INSTALL_DIR}/bb"
    printf "\n"
    printf "Make sure %s is in your PATH, then run:\n" "$INSTALL_DIR"
    printf "  bb auth login\n"
  fi
}

main
