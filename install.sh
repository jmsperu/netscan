#!/usr/bin/env bash
#
# netscan installer — one-liner install for Mac and Linux
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/jmsperu/netscan/main/install.sh | sudo bash
#   curl -fsSL https://raw.githubusercontent.com/jmsperu/netscan/main/install.sh | sudo bash -s -- --version v0.1.1
#
set -euo pipefail

REPO="jmsperu/netscan"
BIN_NAME="netscan"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
VERSION="${VERSION:-latest}"

# Parse flags
while [[ $# -gt 0 ]]; do
    case "$1" in
        --version) VERSION="$2"; shift 2 ;;
        --dir)     INSTALL_DIR="$2"; shift 2 ;;
        *) echo "Unknown flag: $1"; exit 1 ;;
    esac
done

# Colors
if [[ -t 1 ]]; then
    BOLD=$'\033[1m'; GREEN=$'\033[32m'; RED=$'\033[31m'; YELLOW=$'\033[33m'; RESET=$'\033[0m'
else
    BOLD=""; GREEN=""; RED=""; YELLOW=""; RESET=""
fi

log()  { echo "${BOLD}${GREEN}==>${RESET} $*"; }
warn() { echo "${BOLD}${YELLOW}==>${RESET} $*"; }
err()  { echo "${BOLD}${RED}==>${RESET} $*" >&2; exit 1; }

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
    darwin) OS="darwin" ;;
    linux)  OS="linux" ;;
    *) err "Unsupported OS: $OS (netscan supports Linux and macOS)" ;;
esac

# Detect arch
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64)  ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) err "Unsupported architecture: $ARCH (netscan supports amd64 and arm64)" ;;
esac

ASSET="${BIN_NAME}-${OS}-${ARCH}"

log "Detected platform: ${OS}/${ARCH}"

# Determine download URL
if [[ "$VERSION" == "latest" ]]; then
    URL="https://github.com/${REPO}/releases/latest/download/${ASSET}"
else
    URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET}"
fi

log "Downloading from: $URL"

# Check write permission
if [[ ! -w "$INSTALL_DIR" ]] && [[ "$(id -u)" != "0" ]]; then
    err "Cannot write to $INSTALL_DIR. Run with sudo or set INSTALL_DIR (e.g. INSTALL_DIR=~/.local/bin)"
fi

# Download
TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT

if command -v curl >/dev/null 2>&1; then
    curl -fL --progress-bar "$URL" -o "$TMP"
elif command -v wget >/dev/null 2>&1; then
    wget -q --show-progress "$URL" -O "$TMP"
else
    err "Neither curl nor wget is installed. Please install one."
fi

# Verify it's a real binary (not a GitHub 404 page)
SIZE=$(wc -c < "$TMP")
if [[ "$SIZE" -lt 100000 ]]; then
    err "Downloaded file is only $SIZE bytes — release asset may not exist for this platform. Check https://github.com/${REPO}/releases"
fi

# Install
TARGET="${INSTALL_DIR}/${BIN_NAME}"
log "Installing to: $TARGET"

install -m 0755 "$TMP" "$TARGET" 2>/dev/null || {
    cp "$TMP" "$TARGET"
    chmod +x "$TARGET"
}

# macOS: strip quarantine attribute
if [[ "$OS" == "darwin" ]] && command -v xattr >/dev/null 2>&1; then
    xattr -d com.apple.quarantine "$TARGET" 2>/dev/null || true
fi

# Verify
if command -v "$BIN_NAME" >/dev/null 2>&1 && [[ "$(command -v $BIN_NAME)" == "$TARGET" ]]; then
    log "Installed successfully:"
    "$TARGET" --version || true
else
    warn "Installed to $TARGET but it's not in your PATH."
    warn "Add this to your shell rc file:"
    echo "    export PATH=\"$INSTALL_DIR:\$PATH\""
fi

log "Done. Try: ${BOLD}netscan${RESET}"
