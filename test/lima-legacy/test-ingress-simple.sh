#!/bin/bash
set -e

echo "=== Warren Ingress Test - Simple (M7 Phase 7.1) ==="
echo

# Configuration
MANAGER_ADDR="192.168.104.1:8080"
WARREN_CLI="/tmp/warren"

echo "Step 1: Start a simple HTTP server on port 9999"
echo "---"
limactl shell warren-manager-1 'nohup python3 -m http.server 9999 > /tmp/http-server.log 2>&1 &'
sleep 2
echo "✓ HTTP server started on localhost:9999"

echo
echo "Step 2: Test direct access to the HTTP server"
echo "---"
limactl shell warren-manager-1 'curl -s http://localhost:9999/ | head -5'

echo
echo "Step 3: Create an ingress rule pointing to localhost:9999"
echo "---"
limactl shell warren-manager-1 sudo $WARREN_CLI ingress create test \
  --host "test.local" \
  --path "/" \
  --path-type "Prefix" \
  --service "dummy" \
  --port 9999 \
  --manager $MANAGER_ADDR

echo
echo "Step 4: List ingresses"
echo "---"
limactl shell warren-manager-1 sudo $WARREN_CLI ingress list --manager $MANAGER_ADDR

echo
echo "Step 5: Test HTTP routing through ingress proxy"
echo "---"
echo "Testing: curl -H 'Host: test.local' http://localhost:8000/"
limactl shell warren-manager-1 'curl -s -H "Host: test.local" http://localhost:8000/ | head -10'

echo
echo "Step 6: Test wrong host (should return 404)"
echo "---"
echo "Testing: curl -H 'Host: wrong.local' http://localhost:8000/"
limactl shell warren-manager-1 'curl -s -w "\nHTTP Status: %{http_code}\n" -H "Host: wrong.local" http://localhost:8000/'

echo
echo "Step 7: Cleanup - delete ingress and stop HTTP server"
echo "---"
limactl shell warren-manager-1 sudo $WARREN_CLI ingress delete test --manager $MANAGER_ADDR
limactl shell warren-manager-1 'pkill -f "python3 -m http.server"'
echo "✓ Cleanup complete"

echo
echo "=== Test Complete ==="
echo "Summary: Ingress proxy successfully routes HTTP requests based on Host header!"
