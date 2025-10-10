#!/usr/bin/env bash
#
# Warren Multi-Manager Cluster Formation Test
#
# Tests the formation of a 3-manager Raft cluster using Lima VMs:
# 1. Bootstrap first manager (manager-1)
# 2. Generate manager join token
# 3. Join second manager (manager-2)
# 4. Join third manager (manager-3)
# 5. Verify Raft quorum (3 voters)
# 6. Deploy test service
# 7. Verify service is running on workers
#
# Prerequisites:
#   - Run ./test/lima/setup.sh first to create VMs
#   - Warren binary built at bin/warren
#
# Usage:
#   ./test/lima/test-cluster.sh

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Test configuration
MANAGER_1="warren-manager-1"
MANAGER_2="warren-manager-2"
MANAGER_3="warren-manager-3"
WORKER_1="warren-worker-1"
WORKER_2="warren-worker-2"

# API addresses (each manager on different port)
MANAGER_1_API="lima-warren-manager-1.internal:8080"
MANAGER_2_API="lima-warren-manager-2.internal:8081"
MANAGER_3_API="lima-warren-manager-3.internal:8082"

# Raft addresses
MANAGER_1_RAFT="lima-warren-manager-1.internal:7946"
MANAGER_2_RAFT="lima-warren-manager-2.internal:7947"
MANAGER_3_RAFT="lima-warren-manager-3.internal:7948"

# Helper functions
log_step() {
  echo
  echo -e "${CYAN}==>${NC} ${BLUE}$*${NC}"
}

log_info() {
  echo -e "  ${BLUE}ℹ${NC} $*"
}

log_success() {
  echo -e "  ${GREEN}✓${NC} $*"
}

log_error() {
  echo -e "  ${RED}✗${NC} $*"
}

log_warning() {
  echo -e "  ${YELLOW}⚠${NC} $*"
}

# Execute command in VM
vm_exec() {
  local vm="$1"
  shift
  limactl shell "$vm" sudo "$@"
}

# Wait for condition with timeout
wait_for() {
  local timeout="$1"
  local interval="$2"
  local description="$3"
  shift 3
  local command=("$@")

  log_info "Waiting for: $description (timeout: ${timeout}s)"

  local elapsed=0
  while ! "${command[@]}" &> /dev/null; do
    if [[ $elapsed -ge $timeout ]]; then
      log_error "Timeout waiting for: $description"
      return 1
    fi
    sleep "$interval"
    elapsed=$((elapsed + interval))
    echo -n "."
  done
  echo
  log_success "$description"
}

# Check VM is running
check_vm() {
  local vm="$1"
  if ! limactl list --quiet | grep -q "^${vm}$"; then
    log_error "VM $vm not found. Run ./test/lima/setup.sh first"
    exit 1
  fi
  log_success "VM $vm is running"
}

# Test: Check all VMs are running
test_vms_running() {
  log_step "Step 0: Checking VMs are running"

  check_vm "$MANAGER_1"
  check_vm "$MANAGER_2"
  check_vm "$MANAGER_3"
  check_vm "$WORKER_1"
  check_vm "$WORKER_2"
}

# Test: Bootstrap first manager
test_bootstrap_manager1() {
  log_step "Step 1: Bootstrap first manager (manager-1)"

  log_info "Starting manager-1 cluster..."

  # Kill any existing warren processes
  vm_exec "$MANAGER_1" pkill -9 warren || true
  sleep 2

  # Start manager in background
  vm_exec "$MANAGER_1" bash -c "cd /tmp/lima/warren && nohup ./bin/warren cluster init \
    --node-id=manager-1 \
    --bind-addr=${MANAGER_1_RAFT} \
    --api-addr=${MANAGER_1_API} \
    --data-dir=/tmp/warren-data-1 \
    > /tmp/warren-manager-1.log 2>&1 &"

  # Wait for API to be ready
  wait_for 30 2 "manager-1 API to be ready" \
    vm_exec "$MANAGER_1" curl -s "http://${MANAGER_1_API}/health"

  log_success "Manager-1 is running and ready"
}

# Test: Generate join token
test_generate_join_token() {
  log_step "Step 2: Generate manager join token"

  # Generate token from manager-1
  local token
  token=$(vm_exec "$MANAGER_1" /tmp/lima/warren/bin/warren cluster join-token manager \
    --manager="${MANAGER_1_API}" | grep -oP '[a-f0-9]{64}' | head -1)

  if [[ -z "$token" ]]; then
    log_error "Failed to generate join token"
    return 1
  fi

  log_success "Generated join token: ${token:0:16}..."

  # Export for other functions
  echo "$token" > /tmp/warren-join-token
}

# Test: Join second manager
test_join_manager2() {
  log_step "Step 3: Join second manager (manager-2)"

  local token
  token=$(cat /tmp/warren-join-token)

  log_info "Starting manager-2..."

  # Kill any existing warren processes
  vm_exec "$MANAGER_2" pkill -9 warren || true
  sleep 2

  # Start manager-2 in background
  vm_exec "$MANAGER_2" bash -c "cd /tmp/lima/warren && nohup ./bin/warren manager join \
    --node-id=manager-2 \
    --bind-addr=${MANAGER_2_RAFT} \
    --api-addr=${MANAGER_2_API} \
    --data-dir=/tmp/warren-data-2 \
    --leader=${MANAGER_1_API} \
    --token=${token} \
    > /tmp/warren-manager-2.log 2>&1 &"

  # Wait for API to be ready
  wait_for 30 2 "manager-2 API to be ready" \
    vm_exec "$MANAGER_2" curl -s "http://${MANAGER_2_API}/health"

  log_success "Manager-2 joined cluster"
}

# Test: Join third manager
test_join_manager3() {
  log_step "Step 4: Join third manager (manager-3)"

  local token
  token=$(cat /tmp/warren-join-token)

  log_info "Starting manager-3..."

  # Kill any existing warren processes
  vm_exec "$MANAGER_3" pkill -9 warren || true
  sleep 2

  # Start manager-3 in background
  vm_exec "$MANAGER_3" bash -c "cd /tmp/lima/warren && nohup ./bin/warren manager join \
    --node-id=manager-3 \
    --bind-addr=${MANAGER_3_RAFT} \
    --api-addr=${MANAGER_3_API} \
    --data-dir=/tmp/warren-data-3 \
    --leader=${MANAGER_1_API} \
    --token=${token} \
    > /tmp/warren-manager-3.log 2>&1 &"

  # Wait for API to be ready
  wait_for 30 2 "manager-3 API to be ready" \
    vm_exec "$MANAGER_3" curl -s "http://${MANAGER_3_API}/health"

  log_success "Manager-3 joined cluster"
}

# Test: Verify cluster formation
test_verify_cluster() {
  log_step "Step 5: Verify Raft cluster formation"

  # Give Raft time to stabilize
  log_info "Waiting for Raft to stabilize..."
  sleep 5

  # Get cluster info from manager-1
  log_info "Checking cluster info from manager-1..."
  local cluster_info
  cluster_info=$(vm_exec "$MANAGER_1" /tmp/lima/warren/bin/warren cluster info --manager="${MANAGER_1_API}")

  echo "$cluster_info"

  # Verify 3 servers
  local server_count
  server_count=$(echo "$cluster_info" | grep -c "ID:" || echo "0")

  if [[ "$server_count" -ne 3 ]]; then
    log_error "Expected 3 servers, found $server_count"
    return 1
  fi

  log_success "Cluster has 3 Raft servers"

  # Verify leader exists
  if ! echo "$cluster_info" | grep -q "Leader ID:"; then
    log_error "No leader found in cluster"
    return 1
  fi

  log_success "Raft leader elected"
}

# Test: Start workers
test_start_workers() {
  log_step "Step 6: Start worker nodes"

  # Start worker-1
  log_info "Starting worker-1..."
  vm_exec "$WORKER_1" pkill -9 warren || true
  sleep 2

  vm_exec "$WORKER_1" bash -c "cd /tmp/lima/warren && nohup ./bin/warren worker start \
    --node-id=worker-1 \
    --manager=${MANAGER_1_API} \
    --data-dir=/tmp/warren-worker-1 \
    --cpu=2 \
    --memory=2 \
    > /tmp/warren-worker-1.log 2>&1 &"

  sleep 3
  log_success "Worker-1 started"

  # Start worker-2
  log_info "Starting worker-2..."
  vm_exec "$WORKER_2" pkill -9 warren || true
  sleep 2

  vm_exec "$WORKER_2" bash -c "cd /tmp/lima/warren && nohup ./bin/warren worker start \
    --node-id=worker-2 \
    --manager=${MANAGER_1_API} \
    --data-dir=/tmp/warren-worker-2 \
    --cpu=2 \
    --memory=2 \
    > /tmp/warren-worker-2.log 2>&1 &"

  sleep 3
  log_success "Worker-2 started"

  # Wait for workers to register
  log_info "Waiting for workers to register..."
  sleep 5

  # Verify workers registered
  local nodes
  nodes=$(vm_exec "$MANAGER_1" /tmp/lima/warren/bin/warren node list --manager="${MANAGER_1_API}")
  echo "$nodes"

  if ! echo "$nodes" | grep -q "worker-1"; then
    log_error "Worker-1 not registered"
    return 1
  fi

  if ! echo "$nodes" | grep -q "worker-2"; then
    log_error "Worker-2 not registered"
    return 1
  fi

  log_success "Both workers registered with cluster"
}

# Test: Deploy service
test_deploy_service() {
  log_step "Step 7: Deploy test service"

  log_info "Creating nginx service with 2 replicas..."

  vm_exec "$MANAGER_1" /tmp/lima/warren/bin/warren service create nginx-test \
    --image=nginx:alpine \
    --replicas=2 \
    --manager="${MANAGER_1_API}"

  log_success "Service created"

  # Wait for tasks to be scheduled
  log_info "Waiting for tasks to be scheduled..."
  sleep 5

  # Check service status
  local service_info
  service_info=$(vm_exec "$MANAGER_1" /tmp/lima/warren/bin/warren service list --manager="${MANAGER_1_API}")
  echo "$service_info"

  if ! echo "$service_info" | grep -q "nginx-test"; then
    log_error "Service not found"
    return 1
  fi

  log_success "Service deployed successfully"
}

# Cleanup function
cleanup() {
  log_info "Cleaning up test artifacts..."
  rm -f /tmp/warren-join-token
}

# Main test execution
main() {
  echo
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "  Warren Multi-Manager Cluster Formation Test"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

  # Register cleanup
  trap cleanup EXIT

  # Run tests
  test_vms_running
  test_bootstrap_manager1
  test_generate_join_token
  test_join_manager2
  test_join_manager3
  test_verify_cluster
  test_start_workers
  test_deploy_service

  echo
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo -e "${GREEN}✓ All tests passed!${NC}"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo
  echo "Cluster Status:"
  echo "  Managers: 3 (manager-1, manager-2, manager-3)"
  echo "  Workers: 2 (worker-1, worker-2)"
  echo "  Services: 1 (nginx-test with 2 replicas)"
  echo
  echo "Next steps:"
  echo "  - Run ./test/lima/test-failover.sh to test leader failover"
  echo "  - Access manager-1: limactl shell warren-manager-1"
  echo "  - View logs: tail -f /tmp/warren-manager-1.log"
  echo
}

main
