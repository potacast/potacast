#!/bin/sh
# Potacast install script for Linux and macOS.
# Usage: curl -fsSL https://raw.githubusercontent.com/.../install.sh | sh

set -eu

red="$( (/usr/bin/tput bold 2>/dev/null || true; /usr/bin/tput setaf 1 2>/dev/null || true) 2>/dev/null)"
plain="$( (/usr/bin/tput sgr0 2>/dev/null || true) 2>/dev/null)"

status() { echo ">>> $*" >&2; }
error() { echo "${red}ERROR:${plain} $*" >&2; exit 1; }

POTACAST_VERSION="${POTACAST_VERSION:-latest}"
GITHUB_REPO="${GITHUB_REPO:-potacast/potacast}"

OS="$(uname -s)"
ARCH="$(uname -m)"

case "$ARCH" in
	x86_64|amd64) ARCH="amd64" ;;
	aarch64|arm64) ARCH="arm64" ;;
	*) error "Unsupported architecture: $ARCH" ;;
esac

case "$OS" in
	Linux)  OS="linux" ;;
	Darwin) OS="darwin" ;;
	*) error "Unsupported OS: $OS. This script supports Linux and macOS only." ;;
esac

PLATFORM="${OS}-${ARCH}"
TEMP_DIR="$(mktemp -d)"
cleanup() { rm -rf "$TEMP_DIR"; }
trap cleanup EXIT

if [ "$POTACAST_VERSION" = "latest" ]; then
	DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/latest/download/potacast-${PLATFORM}.tar.gz"
else
	DOWNLOAD_URL="https://github.com/${GITHUB_REPO}/releases/download/${POTACAST_VERSION}/potacast-${PLATFORM}.tar.gz"
fi

status "Downloading Potacast for ${PLATFORM}..."
if ! curl -fsSL -o "$TEMP_DIR/potacast.tar.gz" "$DOWNLOAD_URL"; then
	error "Download failed. Check that the release exists: $DOWNLOAD_URL"
fi

status "Extracting..."
tar -xzf "$TEMP_DIR/potacast.tar.gz" -C "$TEMP_DIR"

# Determine install location
if [ "$(id -u)" -eq 0 ]; then
	BIN_DIR="/usr/local/bin"
else
	BIN_DIR="${HOME}/.local/bin"
	mkdir -p "$BIN_DIR"
fi

status "Installing to $BIN_DIR..."
cp "$TEMP_DIR/potacast" "$BIN_DIR/potacast"
chmod +x "$BIN_DIR/potacast"

if ! echo "$PATH" | grep -q "$BIN_DIR"; then
	status "Add $BIN_DIR to your PATH:"
	echo "  export PATH=\"\$PATH:$BIN_DIR\""
fi

status "Install complete. Run 'potacast --help' to get started."
