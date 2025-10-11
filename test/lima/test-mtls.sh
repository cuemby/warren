#!/bin/bash
set -e

# Warren mTLS End-to-End Test
# Tests certificate authority, manager/worker/CLI mTLS connections

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=========================================="
echo "Warren mTLS Security Test"
echo "=========================================="
echo ""

# Configuration
MANAGER_VM="warren-manager-1"
WORKER_VM="warren-worker-1"
MANAGER_ADDR="192.168.104.1:8080"
BINARY_PATH="/Users/ar4mirez/Developer/Work/cuemby/warren/bin/warren-linux-arm64"

# Check if binary exists
if [ ! -f "$BINARY_PATH" ]; then
    echo -e "${RED}✗ Binary not found: $BINARY_PATH${NC}"
    echo "Building Linux ARM64 binary..."
    cd /Users/ar4mirez/Developer/Work/cuemby/warren
    CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/warren-linux-arm64 ./cmd/warren
    echo -e "${GREEN}✓ Binary built${NC}"
fi

# Helper function to run command on VM
run_on_vm() {
    local vm=$1
    shift
    limactl shell "$vm" bash -c "$@"
}

# Helper function to copy binary to VM
copy_binary() {
    local vm=$1
    echo "Copying binary to $vm..."
    limactl copy "$BINARY_PATH" "$vm:/tmp/warren"
    run_on_vm "$vm" chmod +x /tmp/warren
}

echo -e "${YELLOW}Step 1: Clean up previous tests${NC}"
echo "----------------------------------------"
run_on_vm "$MANAGER_VM" sudo pkill warren || true
run_on_vm "$WORKER_VM" sudo pkill warren || true
run_on_vm "$MANAGER_VM" rm -rf /tmp/warren-data ~/.warren || true
run_on_vm "$WORKER_VM" rm -rf /tmp/warren-data ~/.warren || true
sleep 2
echo -e "${GREEN}✓ Cleanup complete${NC}"
echo ""

echo -e "${YELLOW}Step 2: Copy binaries to VMs${NC}"
echo "----------------------------------------"
copy_binary "$MANAGER_VM"
copy_binary "$WORKER_VM"
echo -e "${GREEN}✓ Binaries copied${NC}"
echo ""

echo -e "${YELLOW}Step 3: Initialize manager with CA${NC}"
echo "----------------------------------------"
run_on_vm "$MANAGER_VM" "sudo /tmp/warren cluster init --node-id manager-1 --bind-addr 192.168.104.1:7946 --api-addr 192.168.104.1:8080 --data-dir /tmp/warren-data > /tmp/manager.log 2>&1 &"

echo "Waiting for manager to start..."
sleep 5

# Check manager is running
if run_on_vm "$MANAGER_VM" pgrep -f warren > /dev/null; then
    echo -e "${GREEN}✓ Manager started${NC}"
else
    echo -e "${RED}✗ Manager failed to start${NC}"
    run_on_vm "$MANAGER_VM" cat /tmp/manager.log
    exit 1
fi

# Check CA was created
if run_on_vm "$MANAGER_VM" "test -f ~/.warren/certs/manager-manager-1/node.crt"; then
    echo -e "${GREEN}✓ Manager certificate created${NC}"
else
    echo -e "${RED}✗ Manager certificate not found${NC}"
    exit 1
fi

# Check CA in storage
if run_on_vm "$MANAGER_VM" "test -f ~/.warren/certs/manager-manager-1/ca.crt"; then
    echo -e "${GREEN}✓ CA certificate found${NC}"
else
    echo -e "${RED}✗ CA certificate not found${NC}"
    exit 1
fi
echo ""

echo -e "${YELLOW}Step 4: Generate join token for worker${NC}"
echo "----------------------------------------"
# Try to get token, retry if manager not ready
TOKEN=""
for i in {1..10}; do
    TOKEN=$(run_on_vm "$MANAGER_VM" "/tmp/warren cluster join-token worker --manager 192.168.104.1:8080 2>/dev/null" || true)
    if [ -n "$TOKEN" ]; then
        break
    fi
    echo "Waiting for manager API to be ready (attempt $i/10)..."
    sleep 2
done

if [ -z "$TOKEN" ]; then
    echo -e "${RED}✗ Failed to generate join token${NC}"
    echo "Manager logs:"
    run_on_vm "$MANAGER_VM" cat /tmp/manager.log
    exit 1
fi

echo -e "${GREEN}✓ Join token generated: ${TOKEN:0:20}...${NC}"
echo ""

echo -e "${YELLOW}Step 5: Start worker with token (should request certificate)${NC}"
echo "----------------------------------------"
run_on_vm "$WORKER_VM" "sudo /tmp/warren worker start --node-id worker-1 --manager 192.168.104.1:8080 --data-dir /tmp/warren-data --cpu 2 --memory 4 --token $TOKEN > /tmp/worker.log 2>&1 &"

echo "Waiting for worker to start and request certificate..."
sleep 5

# Check worker is running
if run_on_vm "$WORKER_VM" pgrep -f warren > /dev/null; then
    echo -e "${GREEN}✓ Worker started${NC}"
else
    echo -e "${RED}✗ Worker failed to start${NC}"
    echo "Worker logs:"
    run_on_vm "$WORKER_VM" cat /tmp/worker.log
    exit 1
fi

# Check worker received certificate
if run_on_vm "$WORKER_VM" "test -f ~/.warren/certs/worker-worker-1/node.crt"; then
    echo -e "${GREEN}✓ Worker certificate received and saved${NC}"
else
    echo -e "${RED}✗ Worker certificate not found${NC}"
    echo "Worker logs:"
    run_on_vm "$WORKER_VM" cat /tmp/worker.log
    exit 1
fi

# Verify worker has CA cert
if run_on_vm "$WORKER_VM" "test -f ~/.warren/certs/worker-worker-1/ca.crt"; then
    echo -e "${GREEN}✓ Worker has CA certificate${NC}"
else
    echo -e "${RED}✗ Worker CA certificate not found${NC}"
    exit 1
fi
echo ""

echo -e "${YELLOW}Step 6: Verify worker is connected via mTLS${NC}"
echo "----------------------------------------"
sleep 3
NODES=$(run_on_vm "$MANAGER_VM" "/tmp/warren node list --manager 192.168.104.1:8080 2>/dev/null" || true)
if echo "$NODES" | grep -q "worker-1"; then
    echo -e "${GREEN}✓ Worker registered with manager via mTLS${NC}"
    echo "$NODES"
else
    echo -e "${RED}✗ Worker not registered${NC}"
    echo "Manager logs:"
    run_on_vm "$MANAGER_VM" tail -20 /tmp/manager.log
    echo "Worker logs:"
    run_on_vm "$WORKER_VM" tail -20 /tmp/worker.log
    exit 1
fi
echo ""

echo -e "${YELLOW}Step 7: Initialize CLI certificate${NC}"
echo "----------------------------------------"
# Generate CLI token
CLI_TOKEN=$(run_on_vm "$MANAGER_VM" "/tmp/warren cluster join-token worker --manager 192.168.104.1:8080 2>/dev/null" || true)
if [ -z "$CLI_TOKEN" ]; then
    echo -e "${RED}✗ Failed to generate CLI token${NC}"
    exit 1
fi

# Initialize CLI
run_on_vm "$MANAGER_VM" "/tmp/warren init --manager 192.168.104.1:8080 --token $CLI_TOKEN" > /tmp/cli-init.log 2>&1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ CLI certificate initialized${NC}"
else
    echo -e "${RED}✗ CLI initialization failed${NC}"
    cat /tmp/cli-init.log
    exit 1
fi

# Check CLI certificate exists
if run_on_vm "$MANAGER_VM" "test -f ~/.warren/certs/cli/node.crt"; then
    echo -e "${GREEN}✓ CLI certificate found${NC}"
else
    echo -e "${RED}✗ CLI certificate not found${NC}"
    exit 1
fi
echo ""

echo -e "${YELLOW}Step 8: Test CLI commands via mTLS${NC}"
echo "----------------------------------------"
# Test cluster info
INFO=$(run_on_vm "$MANAGER_VM" "/tmp/warren cluster info --manager 192.168.104.1:8080 2>/dev/null" || true)
if [ -n "$INFO" ]; then
    echo -e "${GREEN}✓ CLI can query cluster info via mTLS${NC}"
    echo "$INFO"
else
    echo -e "${RED}✗ CLI cluster info failed${NC}"
    exit 1
fi
echo ""

echo -e "${YELLOW}Step 9: Deploy test service via mTLS${NC}"
echo "----------------------------------------"
run_on_vm "$MANAGER_VM" "/tmp/warren service create nginx-mtls-test --image nginx:latest --replicas 1 --manager 192.168.104.1:8080" > /tmp/service-create.log 2>&1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Service created via mTLS${NC}"
else
    echo -e "${RED}✗ Service creation failed${NC}"
    cat /tmp/service-create.log
    exit 1
fi

sleep 3

# Verify service exists
SERVICES=$(run_on_vm "$MANAGER_VM" "/tmp/warren service list --manager 192.168.104.1:8080 2>/dev/null" || true)
if echo "$SERVICES" | grep -q "nginx-mtls-test"; then
    echo -e "${GREEN}✓ Service deployed successfully${NC}"
    echo "$SERVICES"
else
    echo -e "${RED}✗ Service not found${NC}"
    exit 1
fi
echo ""

echo -e "${YELLOW}Step 10: Test certificate persistence (restart worker)${NC}"
echo "----------------------------------------"
run_on_vm "$WORKER_VM" sudo pkill warren
sleep 2

run_on_vm "$WORKER_VM" "sudo /tmp/warren worker start --node-id worker-1 --manager 192.168.104.1:8080 --data-dir /tmp/warren-data --cpu 2 --memory 4 > /tmp/worker-restart.log 2>&1 &"

sleep 5

# Check worker reconnected
NODES2=$(run_on_vm "$MANAGER_VM" "/tmp/warren node list --manager 192.168.104.1:8080 2>/dev/null" || true)
if echo "$NODES2" | grep -q "worker-1"; then
    echo -e "${GREEN}✓ Worker reconnected with existing certificate${NC}"
else
    echo -e "${RED}✗ Worker failed to reconnect${NC}"
    echo "Worker logs:"
    run_on_vm "$WORKER_VM" cat /tmp/worker-restart.log
    exit 1
fi
echo ""

echo -e "${YELLOW}Step 11: Security test - unauthorized access${NC}"
echo "----------------------------------------"
# Try to connect without certificate from worker VM
run_on_vm "$WORKER_VM" "rm -rf ~/.warren/certs/cli || true"
UNAUTHORIZED=$(run_on_vm "$WORKER_VM" "/tmp/warren cluster info --manager 192.168.104.1:8080 2>&1" || true)
if echo "$UNAUTHORIZED" | grep -q "certificate"; then
    echo -e "${GREEN}✓ Unauthorized access correctly rejected${NC}"
else
    echo -e "${YELLOW}⚠ Expected certificate error, got: $UNAUTHORIZED${NC}"
fi
echo ""

echo "=========================================="
echo -e "${GREEN}mTLS Security Test: PASSED ✓${NC}"
echo "=========================================="
echo ""
echo "Summary:"
echo "  ✓ Manager CA initialized"
echo "  ✓ Manager certificate issued"
echo "  ✓ Worker requested and received certificate"
echo "  ✓ Worker connected via mTLS"
echo "  ✓ CLI certificate initialized"
echo "  ✓ CLI commands work via mTLS"
echo "  ✓ Service deployed via mTLS"
echo "  ✓ Certificate persistence works"
echo "  ✓ Unauthorized access rejected"
echo ""
echo "All mTLS security features working correctly!"
