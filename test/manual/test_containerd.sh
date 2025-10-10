#!/bin/bash
# Manual test script for containerd integration
# This script tests the containerd runtime integration

set -e

echo "=== Warren Containerd Integration Test ==="
echo ""

# Check if containerd is available
if ! command -v containerd &> /dev/null; then
    echo "⚠️  containerd not found. Installing via Docker..."
    # On macOS with Docker Desktop, we'll test differently
    echo "Note: On macOS, containerd is inside Docker Desktop VM"
    echo "We'll use a simulated test instead."
fi

echo "Step 1: Building Warren..."
cd /Users/ar4mirez/Developer/Work/cuemby/warren
make build
echo "✓ Warren built successfully"
echo ""

echo "Step 2: Starting Warren manager..."
# Kill any existing warren processes
pkill -f "bin/warren" || true
sleep 1

# Start manager in background
./bin/warren manager start --data-dir /tmp/warren-test-manager &
MANAGER_PID=$!
echo "✓ Manager started (PID: $MANAGER_PID)"
sleep 3
echo ""

echo "Step 3: Starting Warren worker..."
./bin/warren worker start --manager localhost:2377 --data-dir /tmp/warren-test-worker &
WORKER_PID=$!
echo "✓ Worker started (PID: $WORKER_PID)"
sleep 3
echo ""

echo "Step 4: Creating a service..."
./bin/warren service create nginx-test --image nginx:alpine --replicas 1
echo "✓ Service created"
sleep 2
echo ""

echo "Step 5: Checking service status..."
./bin/warren service list
echo ""

echo "Step 6: Checking tasks..."
./bin/warren task list
echo ""

echo "Step 7: Waiting for container to start..."
sleep 5

echo "Step 8: Checking task status again..."
./bin/warren task list
echo ""

echo "=== Test Summary ==="
echo "Manager PID: $MANAGER_PID"
echo "Worker PID: $WORKER_PID"
echo ""
echo "To view logs:"
echo "  tail -f /tmp/warren-test-manager/warren.log"
echo "  tail -f /tmp/warren-test-worker/warren.log"
echo ""
echo "To cleanup:"
echo "  kill $MANAGER_PID $WORKER_PID"
echo "  rm -rf /tmp/warren-test-*"
echo ""
echo "Note: Container execution will fail on macOS without proper containerd setup."
echo "This test validates the API and workflow. For full containerd testing, use Linux."
