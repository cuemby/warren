#!/bin/bash
set -e

# Warren Port Publishing Test
# Tests host mode port publishing with iptables

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "=========================================="
echo "Warren Port Publishing Test"
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
    exit 1
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
run_on_vm "$MANAGER_VM" "sudo pkill warren || true"
run_on_vm "$WORKER_VM" "sudo pkill warren || true"
run_on_vm "$MANAGER_VM" "rm -rf /tmp/warren-data ~/.warren /tmp/warren || true"
run_on_vm "$WORKER_VM" "rm -rf /tmp/warren-data ~/.warren /tmp/warren || true"
sleep 2
echo -e "${GREEN}✓ Cleanup complete${NC}"
echo ""

echo -e "${YELLOW}Step 2: Copy binaries to VMs${NC}"
echo "----------------------------------------"
copy_binary "$MANAGER_VM"
copy_binary "$WORKER_VM"
echo -e "${GREEN}✓ Binaries copied${NC}"
echo ""

echo -e "${YELLOW}Step 3: Initialize manager${NC}"
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
echo ""

echo -e "${YELLOW}Step 4: Generate join token for worker${NC}"
echo "----------------------------------------"
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
    exit 1
fi

echo -e "${GREEN}✓ Join token generated: ${TOKEN:0:20}...${NC}"
echo ""

echo -e "${YELLOW}Step 5: Start worker${NC}"
echo "----------------------------------------"
run_on_vm "$WORKER_VM" "sudo /tmp/warren worker start --node-id worker-1 --manager 192.168.104.1:8080 --data-dir /tmp/warren-data --cpu 2 --memory 4 --token $TOKEN > /tmp/worker.log 2>&1 &"

echo "Waiting for worker to start..."
sleep 5

# Check worker is running
if run_on_vm "$WORKER_VM" pgrep -f warren > /dev/null; then
    echo -e "${GREEN}✓ Worker started${NC}"
else
    echo -e "${RED}✗ Worker failed to start${NC}"
    exit 1
fi
echo ""

echo -e "${YELLOW}Step 6: Initialize CLI certificate${NC}"
echo "----------------------------------------"
CLI_TOKEN=$(run_on_vm "$MANAGER_VM" "/tmp/warren cluster join-token worker --manager 192.168.104.1:8080 2>/dev/null" || true)
run_on_vm "$MANAGER_VM" "/tmp/warren init --manager 192.168.104.1:8080 --token $CLI_TOKEN" > /tmp/cli-init.log 2>&1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ CLI certificate initialized${NC}"
else
    echo -e "${RED}✗ CLI initialization failed${NC}"
    cat /tmp/cli-init.log
    exit 1
fi
echo ""

echo -e "${YELLOW}Step 7: Deploy nginx with published port 8080:80${NC}"
echo "----------------------------------------"
run_on_vm "$MANAGER_VM" "/tmp/warren service create nginx-test --image nginx:alpine --replicas 1 --publish 8080:80 --manager 192.168.104.1:8080" > /tmp/service-create.log 2>&1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Service created with published port${NC}"
else
    echo -e "${RED}✗ Service creation failed${NC}"
    cat /tmp/service-create.log
    exit 1
fi

echo "Waiting for service to be scheduled and started..."
sleep 10
echo ""

echo -e "${YELLOW}Step 8: Verify iptables rules created${NC}"
echo "----------------------------------------"
# Check iptables rules on worker (where container is running)
IPTABLES=$(run_on_vm "$WORKER_VM" "sudo iptables -t nat -L PREROUTING -n" || true)
if echo "$IPTABLES" | grep -q "8080"; then
    echo -e "${GREEN}✓ iptables PREROUTING rule found for port 8080${NC}"
    echo "$IPTABLES" | grep "8080"
else
    echo -e "${RED}✗ No iptables rule found for port 8080${NC}"
    echo "Full iptables output:"
    echo "$IPTABLES"
    echo ""
    echo "Worker logs:"
    run_on_vm "$WORKER_VM" tail -30 /tmp/worker.log
    exit 1
fi
echo ""

echo -e "${YELLOW}Step 9: Test HTTP access to published port${NC}"
echo "----------------------------------------"
# Get worker IP
WORKER_IP="192.168.104.2"

# Try to access nginx on published port
HTTP_RESPONSE=$(run_on_vm "$MANAGER_VM" "curl -s -o /dev/null -w '%{http_code}' http://$WORKER_IP:8080 --max-time 5" || echo "failed")
if [ "$HTTP_RESPONSE" = "200" ]; then
    echo -e "${GREEN}✓ HTTP request to port 8080 successful (HTTP 200)${NC}"
else
    echo -e "${RED}✗ HTTP request failed (got: $HTTP_RESPONSE)${NC}"
    echo "Checking if container is running..."
    run_on_vm "$WORKER_VM" "sudo ctr -n warren containers ls"
    echo ""
    echo "Checking iptables rules:"
    run_on_vm "$WORKER_VM" "sudo iptables -t nat -L -n -v"
    exit 1
fi
echo ""

echo -e "${YELLOW}Step 10: Test port cleanup on service deletion${NC}"
echo "----------------------------------------"
run_on_vm "$MANAGER_VM" "/tmp/warren service delete nginx-test --manager 192.168.104.1:8080" > /tmp/service-delete.log 2>&1
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Service deleted${NC}"
else
    echo -e "${RED}✗ Service deletion failed${NC}"
    cat /tmp/service-delete.log
    exit 1
fi

sleep 5

# Check iptables rules are removed
IPTABLES_AFTER=$(run_on_vm "$WORKER_VM" "sudo iptables -t nat -L PREROUTING -n" || true)
if echo "$IPTABLES_AFTER" | grep -q "8080"; then
    echo -e "${RED}✗ iptables rule still exists after service deletion${NC}"
    echo "$IPTABLES_AFTER"
    exit 1
else
    echo -e "${GREEN}✓ iptables rule cleaned up${NC}"
fi
echo ""

echo "=========================================="
echo -e "${GREEN}Port Publishing Test: PASSED ✓${NC}"
echo "=========================================="
echo ""
echo "Summary:"
echo "  ✓ Manager and worker started"
echo "  ✓ Service created with published port"
echo "  ✓ iptables PREROUTING rule created"
echo "  ✓ HTTP access to published port works"
echo "  ✓ iptables rule cleaned up on service deletion"
echo ""
echo "Port publishing feature working correctly!"
