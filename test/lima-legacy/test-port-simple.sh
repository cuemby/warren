#!/bin/bash
# Simple port publishing test - assumes cluster is already running

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "=========================================="
echo "Warren Port Publishing - Simple Test"
echo "=========================================="
echo ""

MANAGER_VM="warren-manager-1"
WORKER_VM="warren-worker-1"
MANAGER_ADDR="192.168.104.1:8080"

echo -e "${YELLOW}Prerequisites:${NC}"
echo "1. Manager must be running"
echo "2. Worker must be running and connected"
echo "3. CLI certificate must be initialized"
echo ""

# Check if manager is accessible
echo -e "${YELLOW}Step 1: Verify manager is accessible${NC}"
if limactl shell "$MANAGER_VM" bash -c '/tmp/warren cluster info --manager '"$MANAGER_ADDR"' 2>/dev/null' > /dev/null; then
    echo -e "${GREEN}✓ Manager is accessible${NC}"
else
    echo -e "${RED}✗ Manager not accessible. Please start the cluster first.${NC}"
    exit 1
fi
echo ""

# Create service with published port
echo -e "${YELLOW}Step 2: Deploy nginx with published port 8080:80${NC}"
limactl shell "$MANAGER_VM" bash -c '/tmp/warren service create nginx-port-test --image nginx:alpine --replicas 1 --publish 8080:80 --manager '"$MANAGER_ADDR"
echo -e "${GREEN}✓ Service creation command sent${NC}"
echo ""

# Wait for scheduling
echo "Waiting for service to be scheduled and started..."
sleep 15
echo ""

# Check service status
echo -e "${YELLOW}Step 3: Verify service is running${NC}"
SERVICES=$(limactl shell "$MANAGER_VM" bash -c '/tmp/warren service list --manager '"$MANAGER_ADDR" || true)
echo "$SERVICES"
if echo "$SERVICES" | grep -q "nginx-port-test"; then
    echo -e "${GREEN}✓ Service found in service list${NC}"
else
    echo -e "${RED}✗ Service not found${NC}"
    exit 1
fi
echo ""

# Check iptables on worker
echo -e "${YELLOW}Step 4: Check iptables rules on worker${NC}"
echo "Checking PREROUTING chain..."
PREROUTING=$(limactl shell "$WORKER_VM" bash -c 'sudo iptables -t nat -L PREROUTING -n -v' || true)
if echo "$PREROUTING" | grep -q "8080"; then
    echo -e "${GREEN}✓ Found iptables PREROUTING rule for port 8080${NC}"
    echo "$PREROUTING" | grep "8080"
else
    echo -e "${YELLOW}⚠ No PREROUTING rule found for port 8080${NC}"
    echo "Full PREROUTING chain:"
    echo "$PREROUTING"
    echo ""
    echo "Worker logs (last 30 lines):"
    limactl shell "$WORKER_VM" bash -c 'tail -30 /tmp/worker.log' || true
fi
echo ""

# Try to access the service
echo -e "${YELLOW}Step 5: Test HTTP access to published port${NC}"
WORKER_IP="192.168.104.2"
echo "Trying to access http://$WORKER_IP:8080..."
HTTP_CODE=$(limactl shell "$MANAGER_VM" bash -c "curl -s -o /dev/null -w '%{http_code}' http://$WORKER_IP:8080 --max-time 5" || echo "failed")

if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}✓ HTTP request successful (HTTP $HTTP_CODE)${NC}"
    echo "Getting full response:"
    limactl shell "$MANAGER_VM" bash -c "curl -s http://$WORKER_IP:8080 | head -5"
else
    echo -e "${RED}✗ HTTP request failed (got: $HTTP_CODE)${NC}"
    echo ""
    echo "Debugging information:"
    echo "1. Container status:"
    limactl shell "$WORKER_VM" bash -c 'sudo ctr -n warren containers ls' || true
    echo ""
    echo "2. Container tasks:"
    limactl shell "$WORKER_VM" bash -c 'sudo ctr -n warren tasks ls' || true
    echo ""
    echo "3. Full iptables NAT table:"
    limactl shell "$WORKER_VM" bash -c 'sudo iptables -t nat -L -n -v' || true
fi
echo ""

echo "=========================================="
echo "Test complete!"
echo "=========================================="
