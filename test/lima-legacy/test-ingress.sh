#!/bin/bash
set -e

echo "=== Warren Ingress Test (M7 Phase 7.1) ==="
echo

# Configuration
MANAGER_ADDR="192.168.104.1:8080"
WARREN_CLI="/tmp/warren"

echo "Step 1: Deploy a simple HTTP service (nginx)"
echo "---"
limactl shell warren-manager-1 sudo $WARREN_CLI service create nginx \
  --image nginx:alpine \
  --replicas 2 \
  --publish 8080:80 \
  --manager $MANAGER_ADDR

echo
echo "Waiting 10 seconds for nginx tasks to start..."
sleep 10

echo
echo "Step 2: List services"
echo "---"
limactl shell warren-manager-1 sudo $WARREN_CLI service list --manager $MANAGER_ADDR

echo
echo "Step 3: Create an ingress rule"
echo "---"
limactl shell warren-manager-1 sudo $WARREN_CLI ingress create web \
  --host "example.local" \
  --path "/" \
  --path-type "Prefix" \
  --service "nginx" \
  --port 8080 \
  --manager $MANAGER_ADDR

echo
echo "Step 4: List ingresses"
echo "---"
limactl shell warren-manager-1 sudo $WARREN_CLI ingress list --manager $MANAGER_ADDR

echo
echo "Step 5: Inspect the ingress"
echo "---"
limactl shell warren-manager-1 sudo $WARREN_CLI ingress inspect web --manager $MANAGER_ADDR

echo
echo "Step 6: Test HTTP routing (should route to nginx)"
echo "---"
echo "Testing: curl -H 'Host: example.local' http://localhost:8000/"
limactl shell warren-manager-1 curl -H "Host: example.local" http://localhost:8000/ 2>/dev/null | head -10

echo
echo "Step 7: Delete the ingress"
echo "---"
limactl shell warren-manager-1 sudo $WARREN_CLI ingress delete web --manager $MANAGER_ADDR

echo
echo "Step 8: Verify ingress was deleted"
echo "---"
limactl shell warren-manager-1 sudo $WARREN_CLI ingress list --manager $MANAGER_ADDR

echo
echo "Step 9: Cleanup - delete nginx service"
echo "---"
limactl shell warren-manager-1 sudo $WARREN_CLI service delete nginx --manager $MANAGER_ADDR

echo
echo "=== Test Complete ==="
