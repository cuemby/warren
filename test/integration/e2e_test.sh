#!/bin/bash
#
# Warren End-to-End Integration Test
# Tests the complete workflow: manager + worker + service creation
#

set -e

WARREN_BIN="${WARREN_BIN:-./bin/warren}"
MANAGER_DATA="./test-manager-data"
WORKER_DATA="./test-worker-data"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

function log() {
    echo -e "${GREEN}[TEST]${NC} $1"
}

function error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

function warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

function cleanup() {
    log "Cleaning up..."

    # Kill processes
    if [ ! -z "$MANAGER_PID" ]; then
        kill $MANAGER_PID 2>/dev/null || true
        wait $MANAGER_PID 2>/dev/null || true
    fi

    if [ ! -z "$WORKER_PID" ]; then
        kill $WORKER_PID 2>/dev/null || true
        wait $WORKER_PID 2>/dev/null || true
    fi

    # Clean up data directories
    rm -rf "$MANAGER_DATA" "$WORKER_DATA"

    log "Cleanup complete"
}

# Set up trap to cleanup on exit
trap cleanup EXIT INT TERM

log "Warren End-to-End Integration Test"
log "===================================="

# Build if needed
if [ ! -f "$WARREN_BIN" ]; then
    log "Building Warren..."
    make build
fi

# Clean up any existing data
rm -rf "$MANAGER_DATA" "$WORKER_DATA"

# Step 1: Start Manager
log "Step 1: Starting manager..."
$WARREN_BIN cluster init \
    --node-id manager-1 \
    --bind-addr 127.0.0.1:7946 \
    --api-addr 127.0.0.1:8080 \
    --data-dir "$MANAGER_DATA" > manager.log 2>&1 &
MANAGER_PID=$!

log "Manager PID: $MANAGER_PID"
sleep 3

# Check if manager is running
if ! kill -0 $MANAGER_PID 2>/dev/null; then
    error "Manager failed to start"
    cat manager.log
    exit 1
fi

log "✓ Manager started"

# Step 2: Start Worker
log "Step 2: Starting worker..."
$WARREN_BIN worker start \
    --node-id worker-1 \
    --manager 127.0.0.1:8080 \
    --data-dir "$WORKER_DATA" \
    --cpu 4 \
    --memory 8 > worker.log 2>&1 &
WORKER_PID=$!

log "Worker PID: $WORKER_PID"
sleep 3

# Check if worker is running
if ! kill -0 $WORKER_PID 2>/dev/null; then
    error "Worker failed to start"
    cat worker.log
    exit 1
fi

log "✓ Worker started"

# Step 3: Verify Node Registration
log "Step 3: Verifying node registration..."
sleep 2

NODES=$($WARREN_BIN node list --manager 127.0.0.1:8080)
echo "$NODES"

if echo "$NODES" | grep -q "worker-1"; then
    log "✓ Worker registered successfully"
else
    error "Worker not found in node list"
    exit 1
fi

# Step 4: Create Service
log "Step 4: Creating service..."
$WARREN_BIN service create test-nginx \
    --image nginx:latest \
    --replicas 3 \
    --env "ENV=test" \
    --manager 127.0.0.1:8080

log "✓ Service created"

# Step 5: Wait for Scheduler
log "Step 5: Waiting for scheduler to create tasks..."
sleep 10

# Step 6: List Services
log "Step 6: Listing services..."
SERVICES=$($WARREN_BIN service list --manager 127.0.0.1:8080)
echo "$SERVICES"

if echo "$SERVICES" | grep -q "test-nginx"; then
    log "✓ Service found in list"
else
    error "Service not found"
    exit 1
fi

# Step 7: Inspect Service
log "Step 7: Inspecting service..."
$WARREN_BIN service inspect test-nginx --manager 127.0.0.1:8080

# Step 8: Scale Service
log "Step 8: Scaling service to 5 replicas..."
$WARREN_BIN service scale test-nginx --replicas 5 --manager 127.0.0.1:8080

log "✓ Service scaled"

# Wait for scheduler
sleep 10

# Step 9: Scale Down
log "Step 9: Scaling service down to 2 replicas..."
$WARREN_BIN service scale test-nginx --replicas 2 --manager 127.0.0.1:8080

log "✓ Service scaled down"

# Step 10: Delete Service
log "Step 10: Deleting service..."
$WARREN_BIN service delete test-nginx --manager 127.0.0.1:8080

log "✓ Service deleted"

# Step 11: Verify Deletion
log "Step 11: Verifying service deletion..."
sleep 3

SERVICES=$($WARREN_BIN service list --manager 127.0.0.1:8080)
if echo "$SERVICES" | grep -q "test-nginx"; then
    error "Service still exists after deletion"
    exit 1
else
    log "✓ Service successfully deleted"
fi

# Success
log ""
log "===================================="
log "✓ All tests passed!"
log "===================================="

exit 0
