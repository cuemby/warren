#!/bin/bash
# Test Warren v1.4.0 Unix Socket in Lima VM
# This script tests the Unix socket auto-detection feature in a real Linux environment

set -e

echo "=== Warren v1.4.0 Unix Socket Test (Lima) ==="
echo ""

VM_NAME="warren-unix-socket-test"
WARREN_BINARY="./bin/warren-linux-arm64"

# Check if binary exists
if [ ! -f "$WARREN_BINARY" ]; then
    echo "Error: Warren binary not found at $WARREN_BINARY"
    echo "Please run: make build-linux-arm64"
    exit 1
fi

# Clean up any existing test VM
echo "1. Cleaning up existing test VM..."
limactl delete -f "$VM_NAME" 2>/dev/null || true
sleep 2

# Create Lima VM with Warren
echo "2. Creating Lima VM..."
limactl create --name="$VM_NAME" template://default
limactl start "$VM_NAME"

# Copy Warren binary to VM
echo "3. Copying Warren binary to VM..."
limactl copy "$WARREN_BINARY" "$VM_NAME:/tmp/warren"
limactl shell "$VM_NAME" chmod +x /tmp/warren

# Start Warren cluster in VM (in background)
echo "4. Starting Warren cluster in VM..."
limactl shell "$VM_NAME" sudo nohup /tmp/warren cluster init --api-addr 0.0.0.0:8080 > /tmp/warren.log 2>&1 &
sleep 5

# Check if Unix socket was created
echo "5. Checking Unix socket creation..."
if limactl shell "$VM_NAME" sudo test -S /var/run/warren.sock; then
    echo "   ✓ Unix socket created: /var/run/warren.sock"
    limactl shell "$VM_NAME" sudo ls -la /var/run/warren.sock
else
    echo "   ✗ FAILED: Unix socket not found"
    echo "   Warren logs:"
    limactl shell "$VM_NAME" sudo cat /tmp/warren.log
    exit 1
fi

# Test read-only command WITHOUT warren init (this is the key test!)
echo "6. Testing 'warren node list' WITHOUT 'warren init'..."
echo "   This should work immediately via Unix socket!"
if limactl shell "$VM_NAME" sudo /tmp/warren node list; then
    echo "   ✓ SUCCESS: Node list worked without warren init!"
else
    echo "   ✗ FAILED: Node list failed"
    limactl shell "$VM_NAME" sudo cat /tmp/warren.log
    exit 1
fi

# Test cluster info
echo "7. Testing 'warren cluster info'..."
if limactl shell "$VM_NAME" sudo /tmp/warren cluster info; then
    echo "   ✓ SUCCESS: Cluster info worked!"
else
    echo "   ✗ FAILED: Cluster info failed"
    exit 1
fi

# Test service list (should be empty but work)
echo "8. Testing 'warren service list'..."
if limactl shell "$VM_NAME" sudo /tmp/warren service list; then
    echo "   ✓ SUCCESS: Service list worked!"
else
    echo "   ✗ FAILED: Service list failed"
    exit 1
fi

# Cleanup
echo "9. Cleaning up..."
limactl delete -f "$VM_NAME"

echo ""
echo "=== ✓ All Tests Passed! ==="
echo ""
echo "Warren v1.4.0 Unix Socket Support Verified:"
echo "  ✓ Unix socket created automatically at /var/run/warren.sock"
echo "  ✓ CLI commands work immediately without 'warren init'"
echo "  ✓ Read-only operations accessible via Unix socket"
echo ""
echo "Ready for v1.4.0 release! 🎉"
