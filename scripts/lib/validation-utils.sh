#!/usr/bin/env bash
#
# Validation Utilities Library
#
# This library provides functions for E2E validation of Warren clusters.
# It automates the validation checklist from docs/e2e-validation.md
#

# Source this file from your main script:
#   source "$(dirname "$0")/lib/validation-utils.sh"

# ============================================================================
# PHASE 1: CLUSTER HEALTH VALIDATION
# ============================================================================

# Validate cluster health endpoints
# Usage: validate_cluster_health <leader_vm_name>
validate_cluster_health() {
  local leader_vm="$1"

  log_step "Phase 1: Cluster Health Validation"

  # 1.1 Health endpoint
  progress_start "Checking /health endpoint"
  if lima_exec "${leader_vm}" "curl -sf http://localhost:8080/health" &>/dev/null; then
    progress_done
  else
    progress_fail
    log_error "Health endpoint failed"
    return 1
  fi

  # 1.2 Ready endpoint
  progress_start "Checking /ready endpoint"
  if lima_exec "${leader_vm}" "curl -sf http://localhost:8080/ready" &>/dev/null; then
    progress_done
  else
    progress_fail
    log_error "Ready endpoint failed"
    return 1
  fi

  # 1.3 Live endpoint
  progress_start "Checking /live endpoint"
  if lima_exec "${leader_vm}" "curl -sf http://localhost:8080/live" &>/dev/null; then
    progress_done
  else
    progress_fail
    log_error "Live endpoint failed"
    return 1
  fi

  # 1.4 Verify Raft leadership
  progress_start "Verifying Raft leadership"
  local metrics=$(lima_exec "${leader_vm}" "curl -s http://localhost:9090/metrics" 2>/dev/null)
  if echo "$metrics" | grep -q "warren_raft_is_leader 1"; then
    progress_done
    log_success "Raft leader confirmed"
  else
    progress_fail
    log_error "No Raft leader detected"
    return 1
  fi

  # 1.5 Check node count
  progress_start "Verifying node count"
  local node_output=$(lima_exec "${leader_vm}" "warren node list --manager localhost:8080" 2>/dev/null)
  local node_count=$(echo "$node_output" | grep -c -E "(manager|worker)")

  if [[ $node_count -gt 0 ]]; then
    progress_done
    log_info "Found ${node_count} nodes in cluster"
  else
    progress_fail
    log_error "No nodes found in cluster"
    return 1
  fi

  log_success "Phase 1: Cluster Health - PASSED"
  return 0
}

# ============================================================================
# PHASE 2: SERVICE DEPLOYMENT VALIDATION
# ============================================================================

# Validate service deployment
# Usage: validate_service_deployment <leader_vm_name>
validate_service_deployment() {
  local leader_vm="$1"
  local test_service="validation-nginx"

  log_step "Phase 2: Service Deployment Validation"

  # 2.1 Create test service
  progress_start "Creating test service (${test_service})"
  if lima_exec "${leader_vm}" "warren service create ${test_service} --image nginx:alpine --replicas 2 --port 80 --manager localhost:8080" &>/dev/null; then
    progress_done
  else
    progress_fail
    log_error "Failed to create test service"
    return 1
  fi

  # 2.2 Wait for service to be ready
  progress_start "Waiting for service to be ready"
  local max_wait=60
  local elapsed=0
  local ready=false

  while [[ $elapsed -lt $max_wait ]]; do
    local service_info=$(lima_exec "${leader_vm}" "warren service inspect ${test_service} --manager localhost:8080" 2>/dev/null || echo "")

    # Check if service has 2 running replicas
    if echo "$service_info" | grep -q "Replicas.*2/2"; then
      ready=true
      break
    fi

    sleep 3
    elapsed=$((elapsed + 3))
  done

  if [[ "$ready" == "true" ]]; then
    progress_done
    log_success "Service deployed successfully (2/2 replicas)"
  else
    progress_fail
    log_error "Service failed to reach desired state within ${max_wait}s"
    # Cleanup
    lima_exec "${leader_vm}" "warren service delete ${test_service} --manager localhost:8080" &>/dev/null
    return 1
  fi

  # 2.3 Verify service is listed
  progress_start "Verifying service appears in list"
  if lima_exec "${leader_vm}" "warren service list --manager localhost:8080" 2>/dev/null | grep -q "${test_service}"; then
    progress_done
  else
    progress_fail
    log_error "Service not found in service list"
    # Cleanup
    lima_exec "${leader_vm}" "warren service delete ${test_service} --manager localhost:8080" &>/dev/null
    return 1
  fi

  # 2.4 Cleanup test service
  progress_start "Cleaning up test service"
  if lima_exec "${leader_vm}" "warren service delete ${test_service} --manager localhost:8080" &>/dev/null; then
    progress_done
  else
    progress_fail
    log_warning "Failed to delete test service (may need manual cleanup)"
  fi

  log_success "Phase 2: Service Deployment - PASSED"
  return 0
}

# ============================================================================
# PHASE 3: SCALING VALIDATION
# ============================================================================

# Validate service scaling
# Usage: validate_scaling <leader_vm_name>
validate_scaling() {
  local leader_vm="$1"
  local test_service="validation-scale"

  log_step "Phase 3: Scaling Validation"

  # 3.1 Create service with 1 replica
  progress_start "Creating service with 1 replica"
  if lima_exec "${leader_vm}" "warren service create ${test_service} --image nginx:alpine --replicas 1 --manager localhost:8080" &>/dev/null; then
    progress_done
  else
    progress_fail
    log_error "Failed to create service"
    return 1
  fi

  # Wait for initial replica
  sleep 5

  # 3.2 Scale up to 5 replicas
  progress_start "Scaling up to 5 replicas"
  if lima_exec "${leader_vm}" "warren service update ${test_service} --replicas 5 --manager localhost:8080" &>/dev/null; then
    progress_done
  else
    progress_fail
    log_error "Failed to scale service"
    # Cleanup
    lima_exec "${leader_vm}" "warren service delete ${test_service} --manager localhost:8080" &>/dev/null
    return 1
  fi

  # 3.3 Wait for scale-up to complete
  progress_start "Waiting for scale-up to complete"
  local max_wait=60
  local elapsed=0
  local scaled=false

  while [[ $elapsed -lt $max_wait ]]; do
    local service_info=$(lima_exec "${leader_vm}" "warren service inspect ${test_service} --manager localhost:8080" 2>/dev/null || echo "")

    if echo "$service_info" | grep -q "Replicas.*5/5"; then
      scaled=true
      break
    fi

    sleep 3
    elapsed=$((elapsed + 3))
  done

  if [[ "$scaled" == "true" ]]; then
    progress_done
    log_success "Scale-up completed (5/5 replicas)"
  else
    progress_fail
    log_error "Scale-up failed to complete within ${max_wait}s"
    # Cleanup
    lima_exec "${leader_vm}" "warren service delete ${test_service} --manager localhost:8080" &>/dev/null
    return 1
  fi

  # 3.4 Scale down to 2 replicas
  progress_start "Scaling down to 2 replicas"
  if lima_exec "${leader_vm}" "warren service update ${test_service} --replicas 2 --manager localhost:8080" &>/dev/null; then
    progress_done
  else
    progress_fail
    log_error "Failed to scale down"
    # Cleanup
    lima_exec "${leader_vm}" "warren service delete ${test_service} --manager localhost:8080" &>/dev/null
    return 1
  fi

  # Wait for scale-down
  sleep 5

  # 3.5 Verify scale-down
  progress_start "Verifying scale-down completed"
  local service_info=$(lima_exec "${leader_vm}" "warren service inspect ${test_service} --manager localhost:8080" 2>/dev/null || echo "")

  if echo "$service_info" | grep -q "Replicas.*2/2"; then
    progress_done
    log_success "Scale-down completed (2/2 replicas)"
  else
    progress_fail
    log_error "Scale-down verification failed"
    # Cleanup
    lima_exec "${leader_vm}" "warren service delete ${test_service} --manager localhost:8080" &>/dev/null
    return 1
  fi

  # 3.6 Cleanup
  progress_start "Cleaning up test service"
  lima_exec "${leader_vm}" "warren service delete ${test_service} --manager localhost:8080" &>/dev/null
  progress_done

  log_success "Phase 3: Scaling - PASSED"
  return 0
}

# ============================================================================
# PHASE 4: LEADER FAILOVER VALIDATION
# ============================================================================

# Validate leader failover
# Usage: validate_leader_failover <num_managers>
validate_leader_failover() {
  local num_managers="$1"

  log_step "Phase 4: Leader Failover Validation"

  # Skip if only 1 manager
  if [[ $num_managers -lt 2 ]]; then
    log_warning "Skipping failover test (requires 2+ managers)"
    return 0
  fi

  local leader_vm="${VM_NAME_PREFIX}-manager-1"

  # 4.1 Identify current leader
  progress_start "Identifying current leader"
  local current_leader=$(warren_get_leader "${leader_vm}")
  if [[ "$current_leader" != "unknown" ]]; then
    progress_done
    log_info "Current leader: ${current_leader}"
  else
    progress_fail
    log_error "Failed to identify leader"
    return 1
  fi

  # 4.2 Stop leader node
  progress_start "Stopping leader node (${current_leader})"
  if lima_stop_vm "${current_leader}"; then
    progress_done
  else
    progress_fail
    log_error "Failed to stop leader node"
    return 1
  fi

  # 4.3 Wait for new leader election
  progress_start "Waiting for leader election (max 30s)"
  sleep 5

  local max_wait=30
  local elapsed=0
  local new_leader_found=false

  # Check other managers for new leader
  for ((i=2; i<=num_managers; i++)); do
    local vm_name="${VM_NAME_PREFIX}-manager-${i}"

    if [[ "$vm_name" == "$current_leader" ]]; then
      continue
    fi

    while [[ $elapsed -lt $max_wait ]]; do
      local metrics=$(lima_exec "${vm_name}" "curl -s http://localhost:9090/metrics" 2>/dev/null || echo "")

      if echo "$metrics" | grep -q "warren_raft_is_leader 1"; then
        new_leader_found=true
        progress_done
        log_success "New leader elected: ${vm_name}"
        break 2
      fi

      sleep 2
      elapsed=$((elapsed + 2))
    done
  done

  if [[ "$new_leader_found" != "true" ]]; then
    progress_fail
    log_error "No new leader elected within ${max_wait}s"
    # Restart original leader
    lima_start_vm "${current_leader}"
    return 1
  fi

  # 4.4 Verify cluster is still operational
  progress_start "Verifying cluster is operational"
  local new_leader="${VM_NAME_PREFIX}-manager-2"
  if lima_exec "${new_leader}" "warren node list --manager localhost:8080" &>/dev/null; then
    progress_done
    log_success "Cluster operational with new leader"
  else
    progress_fail
    log_error "Cluster not operational after failover"
    # Restart original leader
    lima_start_vm "${current_leader}"
    return 1
  fi

  # 4.5 Restart original leader
  progress_start "Restarting original leader (${current_leader})"
  if lima_start_vm "${current_leader}"; then
    lima_wait_ready "${current_leader}"
    progress_done
    log_info "Original leader rejoined cluster"
  else
    progress_fail
    log_warning "Failed to restart original leader (manual intervention may be needed)"
  fi

  log_success "Phase 4: Leader Failover - PASSED"
  return 0
}

# ============================================================================
# PHASE 5: SECRETS VALIDATION
# ============================================================================

# Validate secrets management
# Usage: validate_secrets <leader_vm_name>
validate_secrets() {
  local leader_vm="$1"
  local test_secret="validation-secret"
  local secret_value="test-password-123"

  log_step "Phase 5: Secrets Validation"

  # 5.1 Create secret
  progress_start "Creating test secret"
  if echo "${secret_value}" | lima_exec "${leader_vm}" "warren secret create ${test_secret} --manager localhost:8080" &>/dev/null; then
    progress_done
  else
    progress_fail
    log_error "Failed to create secret"
    return 1
  fi

  # 5.2 List secrets
  progress_start "Verifying secret appears in list"
  if lima_exec "${leader_vm}" "warren secret list --manager localhost:8080" 2>/dev/null | grep -q "${test_secret}"; then
    progress_done
  else
    progress_fail
    log_error "Secret not found in list"
    # Cleanup
    lima_exec "${leader_vm}" "warren secret delete ${test_secret} --manager localhost:8080" &>/dev/null
    return 1
  fi

  # 5.3 Delete secret
  progress_start "Deleting test secret"
  if lima_exec "${leader_vm}" "warren secret delete ${test_secret} --manager localhost:8080" &>/dev/null; then
    progress_done
  else
    progress_fail
    log_error "Failed to delete secret"
    return 1
  fi

  # 5.4 Verify secret is deleted
  progress_start "Verifying secret is deleted"
  if ! lima_exec "${leader_vm}" "warren secret list --manager localhost:8080" 2>/dev/null | grep -q "${test_secret}"; then
    progress_done
  else
    progress_fail
    log_error "Secret still exists after deletion"
    return 1
  fi

  log_success "Phase 5: Secrets - PASSED"
  return 0
}

# ============================================================================
# PHASE 6: VOLUMES VALIDATION
# ============================================================================

# Validate volume management
# Usage: validate_volumes <leader_vm_name>
validate_volumes() {
  local leader_vm="$1"
  local test_volume="validation-volume"

  log_step "Phase 6: Volumes Validation"

  # 6.1 Create volume
  progress_start "Creating test volume"
  if lima_exec "${leader_vm}" "warren volume create ${test_volume} --manager localhost:8080" &>/dev/null; then
    progress_done
  else
    progress_fail
    log_error "Failed to create volume"
    return 1
  fi

  # 6.2 List volumes
  progress_start "Verifying volume appears in list"
  if lima_exec "${leader_vm}" "warren volume list --manager localhost:8080" 2>/dev/null | grep -q "${test_volume}"; then
    progress_done
  else
    progress_fail
    log_error "Volume not found in list"
    # Cleanup
    lima_exec "${leader_vm}" "warren volume delete ${test_volume} --manager localhost:8080" &>/dev/null
    return 1
  fi

  # 6.3 Delete volume
  progress_start "Deleting test volume"
  if lima_exec "${leader_vm}" "warren volume delete ${test_volume} --manager localhost:8080" &>/dev/null; then
    progress_done
  else
    progress_fail
    log_error "Failed to delete volume"
    return 1
  fi

  # 6.4 Verify volume is deleted
  progress_start "Verifying volume is deleted"
  if ! lima_exec "${leader_vm}" "warren volume list --manager localhost:8080" 2>/dev/null | grep -q "${test_volume}"; then
    progress_done
  else
    progress_fail
    log_error "Volume still exists after deletion"
    return 1
  fi

  log_success "Phase 6: Volumes - PASSED"
  return 0
}

# ============================================================================
# PHASE 7: METRICS VALIDATION
# ============================================================================

# Validate metrics endpoint
# Usage: validate_metrics <leader_vm_name>
validate_metrics() {
  local leader_vm="$1"

  log_step "Phase 7: Metrics Validation"

  # 7.1 Check metrics endpoint
  progress_start "Checking /metrics endpoint"
  local metrics=$(lima_exec "${leader_vm}" "curl -s http://localhost:9090/metrics" 2>/dev/null || echo "")

  if [[ -n "$metrics" ]]; then
    progress_done
  else
    progress_fail
    log_error "Metrics endpoint not responding"
    return 1
  fi

  # 7.2 Verify critical metrics exist
  local critical_metrics=(
    "warren_nodes_total"
    "warren_services_total"
    "warren_raft_is_leader"
    "warren_raft_peers_total"
    "warren_api_requests_total"
  )

  progress_start "Verifying critical metrics"
  local missing_metrics=()

  for metric in "${critical_metrics[@]}"; do
    if ! echo "$metrics" | grep -q "^${metric}"; then
      missing_metrics+=("$metric")
    fi
  done

  if [[ ${#missing_metrics[@]} -eq 0 ]]; then
    progress_done
    log_success "All critical metrics present"
  else
    progress_fail
    log_error "Missing metrics: ${missing_metrics[*]}"
    return 1
  fi

  log_success "Phase 7: Metrics - PASSED"
  return 0
}

# ============================================================================
# PHASE 8: PERFORMANCE VALIDATION
# ============================================================================

# Validate performance benchmarks
# Usage: validate_performance <leader_vm_name>
validate_performance() {
  local leader_vm="$1"

  log_step "Phase 8: Performance Validation"

  # 8.1 Service creation latency
  progress_start "Testing service creation latency"
  local start_time=$(date +%s)

  if lima_exec "${leader_vm}" "warren service create perf-test --image nginx:alpine --replicas 1 --manager localhost:8080" &>/dev/null; then
    local end_time=$(date +%s)
    local latency=$((end_time - start_time))

    if [[ $latency -le 5 ]]; then
      progress_done
      log_success "Service creation latency: ${latency}s (target: <5s)"
    else
      progress_fail
      log_warning "Service creation latency: ${latency}s (exceeds target of 5s)"
    fi

    # Cleanup
    lima_exec "${leader_vm}" "warren service delete perf-test --manager localhost:8080" &>/dev/null
  else
    progress_fail
    log_error "Service creation failed"
    return 1
  fi

  # 8.2 API response time
  progress_start "Testing API response time"
  start_time=$(date +%s%N)
  lima_exec "${leader_vm}" "warren node list --manager localhost:8080" &>/dev/null
  end_time=$(date +%s%N)
  local response_time=$(( (end_time - start_time) / 1000000 ))  # Convert to ms

  if [[ $response_time -le 100 ]]; then
    progress_done
    log_success "API response time: ${response_time}ms (target: <100ms)"
  else
    progress_fail
    log_warning "API response time: ${response_time}ms (exceeds target of 100ms)"
  fi

  log_success "Phase 8: Performance - PASSED"
  return 0
}

# ============================================================================
# COMPREHENSIVE VALIDATION
# ============================================================================

# Run all validation phases
# Usage: validate_all <leader_vm_name> <num_managers> <num_workers>
validate_all() {
  local leader_vm="$1"
  local num_managers="$2"
  local num_workers="$3"

  log_step "Starting Comprehensive E2E Validation"
  log_info "Cluster: ${num_managers} managers, ${num_workers} workers"
  echo ""

  local failed_phases=()

  # Phase 1: Cluster Health
  if ! validate_cluster_health "${leader_vm}"; then
    failed_phases+=("Cluster Health")
  fi
  echo ""

  # Phase 2: Service Deployment
  if ! validate_service_deployment "${leader_vm}"; then
    failed_phases+=("Service Deployment")
  fi
  echo ""

  # Phase 3: Scaling
  if ! validate_scaling "${leader_vm}"; then
    failed_phases+=("Scaling")
  fi
  echo ""

  # Phase 4: Leader Failover (if HA cluster)
  if ! validate_leader_failover "${num_managers}"; then
    failed_phases+=("Leader Failover")
  fi
  echo ""

  # Phase 5: Secrets
  if ! validate_secrets "${leader_vm}"; then
    failed_phases+=("Secrets")
  fi
  echo ""

  # Phase 6: Volumes
  if ! validate_volumes "${leader_vm}"; then
    failed_phases+=("Volumes")
  fi
  echo ""

  # Phase 7: Metrics
  if ! validate_metrics "${leader_vm}"; then
    failed_phases+=("Metrics")
  fi
  echo ""

  # Phase 8: Performance
  if ! validate_performance "${leader_vm}"; then
    failed_phases+=("Performance")
  fi
  echo ""

  # Summary
  log_step "E2E Validation Summary"

  if [[ ${#failed_phases[@]} -eq 0 ]]; then
    log_success "✅ ALL VALIDATION PHASES PASSED"
    log_info "Warren cluster is production-ready!"
    return 0
  else
    log_error "❌ VALIDATION FAILURES"
    log_error "Failed phases: ${failed_phases[*]}"
    return 1
  fi
}

# ============================================================================
# EXPORT FUNCTIONS
# ============================================================================

# Make functions available to sourcing scripts
export -f validate_cluster_health
export -f validate_service_deployment
export -f validate_scaling
export -f validate_leader_failover
export -f validate_secrets
export -f validate_volumes
export -f validate_metrics
export -f validate_performance
export -f validate_all

log_verbose "Validation utilities library loaded"
