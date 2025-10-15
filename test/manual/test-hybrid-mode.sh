#!/bin/bash
# Warren v1.6.0 Hybrid Mode Testing Script
# Tests single-node hybrid mode deployment

set -e

echo "=================================================="
echo "Warren v1.6.0 Hybrid Mode Test"
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
    sudo pkill -f "warren cluster init" || true
    sudo pkill -f warren || true
    sleep 2
    sudo rm -rf /tmp/warren-hybrid-test
    info "Cleanup complete"
}

# Trap cleanup on exit
trap cleanup EXIT

echo "=== Test 1: Build Warren v1.6.0 ==="
echo ""

if go build -o /tmp/warren-hybrid-test-binary ./cmd/warren 2>&1; then
    pass "Warren built successfully"
else
    fail "Warren build failed"
    exit 1
fi

echo ""
echo "=== Test 2: Initialize Cluster in Hybrid Mode ==="
echo ""

# Start warren in background
sudo /tmp/warren-hybrid-test-binary cluster init --data-dir /tmp/warren-hybrid-test > /tmp/warren-hybrid-output.log 2>&1 &
WARREN_PID=$!

info "Warren started with PID: $WARREN_PID"
info "Waiting 10 seconds for initialization..."
sleep 10

# Check if process is still running
if ps -p $WARREN_PID > /dev/null; then
    pass "Warren process is running"
else
    fail "Warren process died"
    cat /tmp/warren-hybrid-output.log
    exit 1
fi

echo ""
echo "=== Test 3: Verify Hybrid Mode Output ==="
echo ""

if grep -q "Node running in HYBRID mode" /tmp/warren-hybrid-output.log; then
    pass "Hybrid mode message found in output"
else
    fail "Hybrid mode message NOT found in output"
    info "Log contents:"
    cat /tmp/warren-hybrid-output.log
fi

if grep -q "Workload execution: Ready" /tmp/warren-hybrid-output.log; then
    pass "Workload execution ready"
else
    fail "Workload execution NOT ready"
fi

echo ""
echo "=== Test 4: Verify Node List Shows Hybrid Role ==="
echo ""

sleep 2  # Give it a moment to settle

NODE_OUTPUT=$(/tmp/warren-hybrid-test-binary node list 2>&1 || true)
echo "$NODE_OUTPUT"

if echo "$NODE_OUTPUT" | grep -q "hybrid"; then
    pass "Node registered with 'hybrid' role"
else
    fail "Node NOT showing 'hybrid' role"
    info "Node list output: $NODE_OUTPUT"
fi

echo ""
echo "=== Test 5: Deploy Service (Should Work Immediately) ==="
echo ""

/tmp/warren-hybrid-test-binary service create test-nginx \
    --image nginx:latest \
    --replicas 2 \
    2>&1 || true

sleep 5

SERVICE_LIST=$(/tmp/warren-hybrid-test-binary service list 2>&1 || true)
echo "$SERVICE_LIST"

if echo "$SERVICE_LIST" | grep -q "test-nginx"; then
    pass "Service created successfully"
else
    fail "Service creation failed"
    info "Service list output: $SERVICE_LIST"
fi

echo ""
echo "=== Test 6: Verify Service Scheduled on Hybrid Node ==="
echo ""

sleep 5  # Wait for scheduling

SERVICE_INSPECT=$(/tmp/warren-hybrid-test-binary service inspect test-nginx 2>&1 || true)
echo "$SERVICE_INSPECT"

if echo "$SERVICE_INSPECT" | grep -q "replicas"; then
    pass "Service inspection works"
else
    fail "Service inspection failed"
fi

echo ""
echo "=== Test 7: Verify Unix Socket Works ==="
echo ""

if [ -S /var/run/warren.sock ]; then
    pass "Unix socket exists at /var/run/warren.sock"
else
    fail "Unix socket NOT found"
fi

echo ""
echo "=== Test 8: Clean Up Service ==="
echo ""

/tmp/warren-hybrid-test-binary service delete test-nginx 2>&1 || true
sleep 2

SERVICE_LIST_AFTER=$(/tmp/warren-hybrid-test-binary service list 2>&1 || true)
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
    echo -e "${RED}❌ SOME TESTS FAILED${NC}"
    echo ""
    echo "Review the output above for details."
    exit 1
fi
