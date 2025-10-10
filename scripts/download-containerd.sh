#!/bin/bash
set -e

# Script to download and prepare containerd binaries for embedding in Warren
# This script downloads official containerd releases and places them in pkg/embedded/binaries/

CONTAINERD_VERSION="${CONTAINERD_VERSION:-1.7.24}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BINARIES_DIR="$PROJECT_ROOT/pkg/embedded/binaries"
TEMP_DIR="$(mktemp -d)"

echo "==> Downloading containerd binaries for embedding in Warren"
echo "    Version: $CONTAINERD_VERSION"
echo "    Target directory: $BINARIES_DIR"

# Detect current OS
CURRENT_OS=$(uname -s | tr '[:upper:]' '[:lower:]')
if [[ "$CURRENT_OS" == "darwin" ]]; then
    echo ""
    echo "⚠️  WARNING: Running on macOS"
    echo "    Containerd does NOT provide official macOS binaries."
    echo "    Only Linux binaries will be downloaded for cross-compilation."
    echo ""
    echo "    For macOS development:"
    echo "    1. Install containerd: brew install containerd"
    echo "    2. Start containerd: sudo containerd &"
    echo "    3. Build Warren: make build (without embedded containerd)"
    echo "    4. Run with: sudo warren cluster init --external-containerd"
    echo ""
fi

# Create binaries directory
mkdir -p "$BINARIES_DIR"

# Function to download and extract containerd for a specific platform
download_containerd() {
    local os=$1
    local arch=$2

    echo ""
    echo "==> Downloading containerd for $os/$arch"

    local tarball="containerd-${CONTAINERD_VERSION}-${os}-${arch}.tar.gz"
    local url="https://github.com/containerd/containerd/releases/download/v${CONTAINERD_VERSION}/${tarball}"
    local output_name="containerd-${os}-${arch}"

    # Download
    echo "    Downloading from: $url"
    if ! curl -f -L -o "$TEMP_DIR/$tarball" "$url" 2>/dev/null; then
        echo "    ⚠️  Not available: $os/$arch (this is expected for macOS)"
        return 1
    fi

    # Extract just the containerd binary
    echo "    Extracting containerd binary"
    if ! tar -xzf "$TEMP_DIR/$tarball" -C "$TEMP_DIR" bin/containerd 2>/dev/null; then
        echo "    ⚠️  Failed to extract (invalid archive format)"
        rm -f "$TEMP_DIR/$tarball"
        return 1
    fi

    # Move to binaries directory with platform-specific name
    mv "$TEMP_DIR/bin/containerd" "$BINARIES_DIR/$output_name"
    chmod +x "$BINARIES_DIR/$output_name"

    # Show size
    local size=$(du -h "$BINARIES_DIR/$output_name" | cut -f1)
    echo "    ✓ Extracted to: $output_name ($size)"

    # Cleanup
    rm -f "$TEMP_DIR/$tarball"
    rm -rf "$TEMP_DIR/bin"
    return 0
}

# Download for all supported platforms
# Linux binaries (always available)
echo ""
echo "==> Downloading Linux binaries (for embedded use)..."
LINUX_AMD64_OK=false
LINUX_ARM64_OK=false

if download_containerd "linux" "amd64"; then
    LINUX_AMD64_OK=true
fi

if download_containerd "linux" "arm64"; then
    LINUX_ARM64_OK=true
fi

# macOS binaries (NOT officially available from containerd)
# We attempt download but expect failure - suppress errors
if [[ "$CURRENT_OS" != "darwin" ]]; then
    echo ""
    echo "==> Attempting macOS binaries (not officially supported)..."
fi
download_containerd "darwin" "amd64" 2>/dev/null || true
download_containerd "darwin" "arm64" 2>/dev/null || true

# Cleanup temp directory
rm -rf "$TEMP_DIR"

echo ""
echo "==> Summary of downloaded binaries:"
if ls "$BINARIES_DIR"/containerd-linux-* 1> /dev/null 2>&1; then
    ls -lh "$BINARIES_DIR" | grep containerd-linux

    if [[ "$CURRENT_OS" == "darwin" ]]; then
        echo ""
        echo "    ℹ️  macOS binaries not available (containerd doesn't provide them)"
    fi
else
    echo "   ⚠️  No Linux binaries downloaded!"
    echo ""
    echo "   This may happen if:"
    echo "   - Network issues prevented download"
    echo "   - Containerd version $CONTAINERD_VERSION not found"
    echo ""
    exit 1
fi

echo ""
echo "==> Done! Binaries are ready to be embedded in Warren."
if [[ "$CURRENT_OS" == "darwin" ]]; then
    echo ""
    echo "    📝 macOS Usage:"
    echo "       make build                 # Dev build (use with --external-containerd)"
    echo "       make build-linux-amd64     # Cross-compile for Linux (with embedded containerd)"
    echo ""
    echo "    To test on macOS:"
    echo "       1. brew install containerd"
    echo "       2. sudo containerd &"
    echo "       3. make build"
    echo "       4. sudo ./bin/warren cluster init --external-containerd"
else
    echo "    Run 'make build-embedded' to create Warren binary with embedded containerd."
fi
