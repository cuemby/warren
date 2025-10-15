#!/bin/bash
# Warren v1.6.0 Hybrid Mode Testing Script (for Lima VM)
# Uses pre-built binary

set -e

echo "=================================================="
echo "Warren v1.6.0 Hybrid Mode Test (Lima VM)"
echo "=================================================="
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counters
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
pass() {
    echo -e "${GREEN}✓ PASS:${NC} $1"
    TESTS_PASSED=$((TESTS_PASSED + 1))
}

fail() {
    echo -e "${RED}✗ FAIL:${NC} $1"
    TESTS_FAILED=$((TESTS_FAILED + 1))
}

info() {
    echo -e "${YELLOW}ℹ INFO:${NC} $1"
}

cleanup() {
    info "Cleaning up..."
    pkill -f "warren cluster init" || true
    pkill -f warren || true
    sleep 2
    rm -rf /tmp/warren-hybrid-test
    rm -f /tmp/warren-hybrid-output.log
    info "Cleanup complete"
}

# Trap cleanup on exit
trap cleanup EXIT

# Cleanup before starting
cleanup

echo "=== Test 1: Verify Warren Binary ==="
echo ""

if [ -f /tmp/warren ]; then
    pass "Warren binary found at /tmp/warren"
else
    fail "Warren binary NOT found"
    exit 1
fi

WARREN_BIN="/tmp/warren"

echo ""
echo "=== Test 2: Initialize Cluster in Hybrid Mode ==="
echo ""

# Start warren in background
$WARREN_BIN cluster init --data-dir /tmp/warren-hybrid-test > /tmp/warren-hybrid-output.log 2>&1 &
WARREN_PID=$!

info "Warren started with PID: $WARREN_PID"
info "Waiting 15 seconds for initialization..."
sleep 15

# Check if process is still running
if ps -p $WARREN_PID > /dev/null 2>&1; then
    pass "Warren process is running"
else
    fail "Warren process died"
    echo "=== Warren output ==="
    cat /tmp/warren-hybrid-output.log
    exit 1
fi

echo ""
echo "=== Test 3: Verify Hybrid Mode Output ==="
echo ""

echo "=== Warren output (first 50 lines) ==="
head -50 /tmp/warren-hybrid-output.log
echo "=== End of output sample ==="
echo ""

if grep -q "HYBRID mode" /tmp/warren-hybrid-output.log; then
    pass "Hybrid mode message found in output"
else
    fail "Hybrid mode message NOT found"
    info "Searching for alternative patterns..."
    if grep -i "worker" /tmp/warren-hybrid-output.log | head -5; then
        info "Found worker-related output"
    fi
fi

if grep -q "Workload execution: Ready" /tmp/warren-hybrid-output.log; then
    pass "Workload execution ready"
else
    fail "Workload execution status not found (might be ok if worker started differently)"
fi

echo ""
echo "=== Test 4: Verify Node List Shows Hybrid Role ==="
echo ""

sleep 3  # Give it a moment to settle

info "Running: $WARREN_BIN node list"
NODE_OUTPUT=$($WARREN_BIN node list 2>&1 || true)
echo "$NODE_OUTPUT"

if echo "$NODE_OUTPUT" | grep -q "hybrid"; then
    pass "Node registered with 'hybrid' role"
elif echo "$NODE_OUTPUT" | grep -q "manager"; then
    info "Node showing 'manager' role (expected if hybrid logic has issue)"
    fail "Expected 'hybrid' role but got 'manager'"
else
    fail "Unable to verify node role"
fi

echo ""
echo "=== Test 5: Deploy Service (Test Immediate Deployment) ==="
echo ""

info "Running: $WARREN_BIN service create test-nginx --image nginx:latest --replicas 2"
$WARREN_BIN service create test-nginx \
    --image nginx:latest \
    --replicas 2 \
    2>&1 || info "Service create command finished (check output)"

sleep 8

info "Running: $WARREN_BIN service list"
SERVICE_LIST=$($WARREN_BIN service list 2>&1 || true)
echo "$SERVICE_LIST"

if echo "$SERVICE_LIST" | grep -q "test-nginx"; then
    pass "Service created successfully"
else
    fail "Service NOT found in list"
fi

echo ""
echo "=== Test 6: Check Service Replicas ==="
echo ""

sleep 5  # Wait for scheduling

info "Running: $WARREN_BIN service inspect test-nginx"
SERVICE_INSPECT=$($WARREN_BIN service inspect test-nginx 2>&1 || true)
echo "$SERVICE_INSPECT"

if echo "$SERVICE_INSPECT" | grep -i "running\|pending\|replicas"; then
    pass "Service has task information"
else
    fail "No task information found"
fi

echo ""
echo "=== Test 7: Verify Unix Socket ==="
echo ""

if [ -S /var/run/warren.sock ]; then
    pass "Unix socket exists at /var/run/warren.sock"
    ls -l /var/run/warren.sock
else
    fail "Unix socket NOT found"
fi

echo ""
echo "=== Test 8: Clean Up Service ==="
echo ""

info "Running: $WARREN_BIN service delete test-nginx"
$WARREN_BIN service delete test-nginx 2>&1 || true
sleep 3

SERVICE_LIST_AFTER=$($WARREN_BIN service list 2>&1 || true)
if echo "$SERVICE_LIST_AFTER" | grep -q "test-nginx"; then
    fail "Service deletion failed (still exists)"
else
    pass "Service deleted successfully"
fi

echo ""
echo "=================================================="
echo "Test Summary"
echo "=================================================="
echo -e "${GREEN}Tests Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Tests Failed: $TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 ALL TESTS PASSED!${NC}"
    echo ""
    echo "Warren v1.6.0 Hybrid Mode is working correctly!"
    exit 0
else
    echo -e "${YELLOW}⚠️  SOME TESTS FAILED${NC}"
    echo ""
    echo "This might be expected if some features need refinement."
    echo "Review the output above for details."
    exit 1
fi
