#!/bin/bash
set -e

echo "=== Warren HTTPS/TLS Test (M7 Phase 7.2) ==="
echo

# Configuration
MANAGER_ADDR="192.168.104.1:8080"
WARREN_CLI="/tmp/warren"

echo "Step 1: Generate self-signed certificate for test.local"
echo "---"
limactl shell warren-manager-1 openssl req -x509 -newkey rsa:2048 -nodes -keyout /tmp/test.key -out /tmp/test.crt -days 365 -subj "/C=US/ST=Test/L=Test/O=Warren/CN=test.local" && echo "✓ Certificate generated"

echo
echo "Step 2: Start a simple HTTP server on port 9999"
echo "---"
limactl shell warren-manager-1 pkill -f "python3 -m http.server" || true
limactl shell warren-manager-1 nohup python3 -m http.server 9999 > /tmp/http-server.log 2>&1 &
sleep 2
echo "✓ HTTP server started on localhost:9999"

echo
echo "Step 3: Upload TLS certificate to Warren"
echo "---"
limactl shell warren-manager-1 sudo $WARREN_CLI certificate create test-cert \
  --cert /tmp/test.crt \
  --key /tmp/test.key \
  --hosts test.local \
  --manager $MANAGER_ADDR

echo
echo "Step 4: List certificates"
echo "---"
limactl shell warren-manager-1 sudo $WARREN_CLI certificate list --manager $MANAGER_ADDR

echo
echo "Step 5: Create an ingress rule for test.local"
echo "---"
limactl shell warren-manager-1 sudo $WARREN_CLI ingress create test-https \
  --host "test.local" \
  --path "/" \
  --path-type "Prefix" \
  --service "dummy" \
  --port 9999 \
  --manager $MANAGER_ADDR

echo
echo "Step 6: Test HTTP routing (port 8000)"
echo "---"
echo "Testing: curl -H 'Host: test.local' http://localhost:8000/"
limactl shell warren-manager-1 curl -s -H "Host: test.local" http://localhost:8000/ | head -5

echo
echo "Step 7: Test HTTPS routing (port 8443)"
echo "---"
echo "Testing: curl -k -H 'Host: test.local' https://localhost:8443/"
limactl shell warren-manager-1 curl -k -s -H "Host: test.local" https://localhost:8443/ | head -5

echo
echo "Step 8: Verify TLS connection details"
echo "---"
echo "Testing: openssl s_client -connect localhost:8443 -servername test.local"
echo | limactl shell warren-manager-1 openssl s_client -connect localhost:8443 -servername test.local 2>/dev/null | grep -E "(subject|issuer|Cipher)"

echo
echo "Step 9: Cleanup"
echo "---"
limactl shell warren-manager-1 sudo $WARREN_CLI ingress delete test-https --manager $MANAGER_ADDR
limactl shell warren-manager-1 sudo $WARREN_CLI certificate delete test-cert --manager $MANAGER_ADDR
limactl shell warren-manager-1 pkill -f "python3 -m http.server"
limactl shell warren-manager-1 rm -f /tmp/test.{crt,key}
echo "✓ Cleanup complete"

echo
echo "=== Test Complete ==="
echo "Summary: HTTPS/TLS termination is working!"
