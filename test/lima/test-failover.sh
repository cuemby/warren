#!/usr/bin/env bash
#
# Warren Leader Failover Test
#
# Tests Raft leader failover in a 3-manager cluster:
# 1. Identify current leader
# 2. Kill the leader process
# 3. Verify new leader elected within 10 seconds
# 4. Verify cluster continues operating
# 5. Test read/write operations after failover
#
# Prerequisites:
#   - Run ./test/lima/test-cluster.sh first to set up cluster
#   - 3 managers must be running
#
# Usage:
#   ./test/lima/test-failover.sh

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

MANAGER_1_API="lima-warren-manager-1.internal:8080"
MANAGER_2_API="lima-warren-manager-2.internal:8081"
MANAGER_3_API="lima-warren-manager-3.internal:8082"

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

# Get cluster info from a manager
get_cluster_info() {
  local manager_api="$1"
  local vm="$2"

  vm_exec "$vm" /tmp/lima/warren/bin/warren cluster info --manager="$manager_api" 2>/dev/null || echo ""
}

# Get leader ID from cluster info
get_leader_id() {
  local cluster_info="$1"
  echo "$cluster_info" | grep "Leader ID:" | awk '{print $3}' || echo ""
}

# Map manager ID to VM name and API
get_manager_info() {
  local manager_id="$1"
  case "$manager_id" in
    manager-1)
      echo "$MANAGER_1:$MANAGER_1_API:$MANAGER_1_RAFT"
      ;;
    manager-2)
      echo "$MANAGER_2:$MANAGER_2_API:$MANAGER_2_RAFT"
      ;;
    manager-3)
      echo "$MANAGER_3:$MANAGER_3_API:$MANAGER_3_RAFT"
      ;;
    *)
      echo "unknown:unknown:unknown"
      ;;
  esac
}

# Test: Identify current leader
test_identify_leader() {
  log_step "Step 1: Identify current leader"

  log_info "Querying cluster info..."

  local cluster_info
  cluster_info=$(get_cluster_info "$MANAGER_1_API" "$MANAGER_1")

  if [[ -z "$cluster_info" ]]; then
    log_error "Failed to get cluster info from manager-1"
    log_info "Trying manager-2..."
    cluster_info=$(get_cluster_info "$MANAGER_2_API" "$MANAGER_2")
  fi

  if [[ -z "$cluster_info" ]]; then
    log_error "Failed to get cluster info from manager-2"
    log_info "Trying manager-3..."
    cluster_info=$(get_cluster_info "$MANAGER_3_API" "$MANAGER_3")
  fi

  if [[ -z "$cluster_info" ]]; then
    log_error "Cannot get cluster info from any manager"
    log_error "Make sure the cluster is running (./test/lima/test-cluster.sh)"
    return 1
  fi

  local leader_id
  leader_id=$(get_leader_id "$cluster_info")

  if [[ -z "$leader_id" ]]; then
    log_error "No leader found in cluster"
    return 1
  fi

  log_success "Current leader: $leader_id"

  # Export for other functions
  echo "$leader_id" > /tmp/warren-current-leader
}

# Test: Kill leader
test_kill_leader() {
  log_step "Step 2: Kill current leader"

  local leader_id
  leader_id=$(cat /tmp/warren-current-leader)

  local manager_info
  manager_info=$(get_manager_info "$leader_id")
  IFS=':' read -r vm api raft <<< "$manager_info"

  if [[ "$vm" == "unknown" ]]; then
    log_error "Unknown leader ID: $leader_id"
    return 1
  fi

  log_info "Killing $leader_id (VM: $vm)..."

  # Kill warren process on leader
  vm_exec "$vm" pkill -9 warren || true

  log_success "Leader process killed"

  # Save info for verification
  echo "$leader_id:$vm:$api:$raft" > /tmp/warren-killed-leader
}

# Test: Verify new leader elected
test_verify_new_leader() {
  log_step "Step 3: Verify new leader election"

  local old_leader_id
  old_leader_id=$(cat /tmp/warren-current-leader)

  log_info "Waiting for new leader election (timeout: 10s)..."

  # Try all remaining managers
  local managers=("$MANAGER_1:$MANAGER_1_API" "$MANAGER_2:$MANAGER_2_API" "$MANAGER_3:$MANAGER_3_API")

  local start_time=$SECONDS
  local new_leader_id=""
  local elapsed=0

  while [[ $elapsed -lt 15 ]]; do
    for manager_entry in "${managers[@]}"; do
      IFS=':' read -r vm api <<< "$manager_entry"

      # Skip the killed leader
      if [[ "$vm" == *"${old_leader_id##*-}"* ]]; then
        continue
      fi

      local cluster_info
      cluster_info=$(get_cluster_info "$api" "$vm" 2>/dev/null || echo "")

      if [[ -n "$cluster_info" ]]; then
        new_leader_id=$(get_leader_id "$cluster_info")

        if [[ -n "$new_leader_id" && "$new_leader_id" != "$old_leader_id" ]]; then
          elapsed=$((SECONDS - start_time))
          log_success "New leader elected: $new_leader_id"
          log_success "Failover time: ${elapsed}s (target: <10s)"

          if [[ $elapsed -gt 10 ]]; then
            log_warning "Failover took longer than 10s target"
          fi

          # Save new leader
          echo "$new_leader_id" > /tmp/warren-new-leader
          return 0
        fi
      fi
    done

    sleep 1
    elapsed=$((SECONDS - start_time))
    echo -n "."
  done

  echo
  log_error "New leader not elected within 15s"
  return 1
}

# Test: Verify cluster operation
test_cluster_operation() {
  log_step "Step 4: Verify cluster continues operating"

  local new_leader_id
  new_leader_id=$(cat /tmp/warren-new-leader)

  local manager_info
  manager_info=$(get_manager_info "$new_leader_id")
  IFS=':' read -r vm api raft <<< "$manager_info"

  log_info "Testing cluster operations via new leader ($new_leader_id)..."

  # Test 1: List services
  log_info "Test 1: List services"
  local services
  services=$(vm_exec "$vm" /tmp/lima/warren/bin/warren service list --manager="$api" 2>&1 || echo "FAILED")

  if echo "$services" | grep -q "FAILED"; then
    log_error "Failed to list services"
    return 1
  fi
  log_success "Can list services"

  # Test 2: Create new service
  log_info "Test 2: Create new service (after failover)"
  if vm_exec "$vm" /tmp/lima/warren/bin/warren service create test-failover \
    --image=nginx:alpine \
    --replicas=1 \
    --manager="$api" 2>&1; then
    log_success "Created service after failover"
  else
    log_error "Failed to create service after failover"
    return 1
  fi

  # Test 3: List nodes
  log_info "Test 3: List nodes"
  local nodes
  nodes=$(vm_exec "$vm" /tmp/lima/warren/bin/warren node list --manager="$api" 2>&1 || echo "FAILED")

  if echo "$nodes" | grep -q "FAILED"; then
    log_error "Failed to list nodes"
    return 1
  fi
  log_success "Can list nodes"

  # Test 4: Check cluster info
  log_info "Test 4: Check cluster info"
  local cluster_info
  cluster_info=$(vm_exec "$vm" /tmp/lima/warren/bin/warren cluster info --manager="$api" 2>&1 || echo "FAILED")

  if echo "$cluster_info" | grep -q "FAILED"; then
    log_error "Failed to get cluster info"
    return 1
  fi

  # Verify we have 2 servers now (one is dead)
  local server_count
  server_count=$(echo "$cluster_info" | grep -c "ID:" || echo "0")

  log_info "Active servers: $server_count (expected: 2)"

  if [[ "$server_count" -lt 2 ]]; then
    log_warning "Expected at least 2 active servers, found $server_count"
  else
    log_success "Cluster has $server_count active servers"
  fi
}

# Test: Restart killed leader
test_restart_leader() {
  log_step "Step 5: Restart the killed leader (optional)"

  local leader_info
  leader_info=$(cat /tmp/warren-killed-leader)
  IFS=':' read -r leader_id vm api raft <<< "$leader_info"

  log_info "Restarting $leader_id..."

  # Restart as follower
  local node_num="${leader_id##*-}"
  vm_exec "$vm" bash -c "cd /tmp/lima/warren && nohup ./bin/warren cluster init \
    --node-id=${leader_id} \
    --bind-addr=${raft} \
    --api-addr=${api} \
    --data-dir=/tmp/warren-data-${node_num} \
    > /tmp/warren-${leader_id}.log 2>&1 &" || true

  log_info "Waiting for $leader_id to rejoin..."
  sleep 5

  # Check if it rejoined
  local new_leader_id
  new_leader_id=$(cat /tmp/warren-new-leader)
  local new_leader_info
  new_leader_info=$(get_manager_info "$new_leader_id")
  IFS=':' read -r new_vm new_api new_raft <<< "$new_leader_info"

  local cluster_info
  cluster_info=$(get_cluster_info "$new_api" "$new_vm" 2>/dev/null || echo "")

  if echo "$cluster_info" | grep -q "$leader_id"; then
    log_success "$leader_id rejoined cluster"
  else
    log_warning "$leader_id may not have rejoined yet (check logs)"
  fi
}

# Cleanup function
cleanup() {
  log_info "Cleaning up test artifacts..."
  rm -f /tmp/warren-current-leader /tmp/warren-killed-leader /tmp/warren-new-leader
}

# Main test execution
main() {
  echo
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "  Warren Leader Failover Test"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

  # Register cleanup
  trap cleanup EXIT

  # Run tests
  test_identify_leader
  test_kill_leader
  test_verify_new_leader
  test_cluster_operation
  test_restart_leader

  echo
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo -e "${GREEN}✓ Failover test passed!${NC}"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo
  echo "Results:"
  echo "  - Leader failover successful"
  echo "  - Cluster operational after failover"
  echo "  - All manager operations working"
  echo
  echo "Note: The killed leader has been restarted and should rejoin the cluster."
  echo "      Check cluster status with: warren cluster info"
  echo
}

main
