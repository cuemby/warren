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
    if ! curl -L -o "$TEMP_DIR/$tarball" "$url"; then
        echo "    ⚠️  Failed to download $os/$arch (may not be available)"
        return 1
    fi

    # Extract just the containerd binary
    echo "    Extracting containerd binary"
    tar -xzf "$TEMP_DIR/$tarball" -C "$TEMP_DIR" bin/containerd

    # Move to binaries directory with platform-specific name
    mv "$TEMP_DIR/bin/containerd" "$BINARIES_DIR/$output_name"
    chmod +x "$BINARIES_DIR/$output_name"

    # Show size
    local size=$(du -h "$BINARIES_DIR/$output_name" | cut -f1)
    echo "    ✓ Extracted to: $output_name ($size)"

    # Cleanup
    rm -f "$TEMP_DIR/$tarball"
    rm -rf "$TEMP_DIR/bin"
}

# Download for all supported platforms
# Primary platforms
download_containerd "linux" "amd64" || true
download_containerd "linux" "arm64" || true
download_containerd "darwin" "amd64" || true  # Intel macOS
download_containerd "darwin" "arm64" || true  # Apple Silicon macOS

# Cleanup temp directory
rm -rf "$TEMP_DIR"

echo ""
echo "==> Summary of downloaded binaries:"
ls -lh "$BINARIES_DIR" | grep containerd || echo "   No binaries downloaded"

echo ""
echo "==> Done! Binaries are ready to be embedded in Warren."
echo "    Run 'make build' to create Warren binary with embedded containerd."
