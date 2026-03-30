#!/bin/sh
# Install script for Squire
# Usage: curl -sSL https://raw.githubusercontent.com/dan-strohschein/squire/main/install.sh | sh
set -e

REPO="dan-strohschein/squire"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY="squire"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Error: unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  linux|darwin) ;;
  mingw*|msys*|cygwin*) OS="windows" ;;
  *) echo "Error: unsupported OS: $OS"; exit 1 ;;
esac

echo "Squire — AI code assistant toolkit"
echo ""

# Get latest release tag
TAG=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name"' | cut -d'"' -f4)
if [ -z "$TAG" ]; then
  echo "Error: could not determine latest release."
  echo "Check https://github.com/${REPO}/releases for available versions."
  exit 1
fi

echo "Installing squire ${TAG} for ${OS}/${ARCH}..."

SUFFIX="${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
  ARCHIVE="${BINARY}-${SUFFIX}.zip"
else
  ARCHIVE="${BINARY}-${SUFFIX}.tar.gz"
fi

URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE}"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading ${URL}..."
curl -sSL -o "${TMPDIR}/${ARCHIVE}" "$URL"

if [ $? -ne 0 ]; then
  echo "Error: download failed."
  exit 1
fi

cd "$TMPDIR"
if [ "$OS" = "windows" ]; then
  unzip -q "$ARCHIVE"
else
  tar xzf "$ARCHIVE"
fi

# Find the binary (may have OS/arch suffix)
BIN=$(find . -name "${BINARY}*" -type f -not -name "*.tar.gz" -not -name "*.zip" | head -1)
if [ -z "$BIN" ]; then
  echo "Error: binary not found in archive."
  exit 1
fi

chmod +x "$BIN"

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv "$BIN" "${INSTALL_DIR}/${BINARY}"
else
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv "$BIN" "${INSTALL_DIR}/${BINARY}"
fi

echo ""
echo "✓ Installed squire to ${INSTALL_DIR}/${BINARY}"
echo ""
echo "Get started:"
echo "  cd /path/to/your/project"
echo "  squire init"
echo ""
echo "squire embeds cartograph (semantic graph) and chisel (refactoring)."
echo "Language generators are installed on demand by 'squire init'."
