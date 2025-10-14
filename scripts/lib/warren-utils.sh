#!/usr/bin/env bash
#
# Warren Utilities Library
#
# This library provides functions for Warren cluster operations including:
# - Binary download and installation
# - Manager initialization
# - Worker joining
# - Cluster validation
#

# Source this file from your main script:
#   source "$(dirname "$0")/lib/warren-utils.sh"

# ============================================================================
# WARREN BINARY MANAGEMENT
# ============================================================================

# Download Warren binary for specified OS and architecture
# Usage: warren_download_binary <version> <os> <arch> <output_dir>
warren_download_binary() {
  local version="$1"
  local os="$2"
  local arch="$3"
  local output_dir="$4"

  local binary_name="warren-${os}-${arch}"
  local download_url="https://github.com/cuemby/warren/releases/download/${version}/${binary_name}.tar.gz"
  local output_file="${output_dir}/${binary_name}"

  log_verbose "Downloading Warren ${version} for ${os}/${arch}"

  # Create output directory if it doesn't exist
  mkdir -p "${output_dir}"

  # Check if binary already exists
  if [[ -f "${output_file}" ]]; then
    log_info "Warren binary already exists: ${output_file}"
    return 0
  fi

  # Download binary
  progress_start "Downloading Warren ${version}"
  if execute "curl -sL ${download_url} -o ${output_dir}/${binary_name}.tar.gz"; then
    # Extract binary
    if execute "tar -xzf ${output_dir}/${binary_name}.tar.gz -C ${output_dir}"; then
      # Make executable
      execute "chmod +x ${output_file}"
      # Cleanup tarball
      rm -f "${output_dir}/${binary_name}.tar.gz"
      progress_done
      return 0
    fi
  fi

  progress_fail
  log_error "Failed to download Warren binary"
  return 1
}

# Install Warren binary on a VM
# Usage: warren_install_on_vm <vm_name> <local_binary_path>
warren_install_on_vm() {
  local vm_name="$1"
  local local_binary="$2"

  log_verbose "Installing Warren on ${vm_name}"

  # Copy binary to VM
  if ! lima_copy_to_vm "${vm_name}" "${local_binary}" "/tmp/warren"; then
    log_error "Failed to copy Warren binary to ${vm_name}"
    return 1
  fi

  # Install to /usr/local/bin and make executable
  progress_start "Installing Warren on ${vm_name}"
  if limactl shell "${vm_name}" sudo sh -c 'mv /tmp/warren /usr/local/bin/warren && chmod +x /usr/local/bin/warren' >> "${LOG_FILE}" 2>&1; then
    progress_done
    return 0
  else
    progress_fail
    return 1
  fi
}

# Verify Warren installation on VM
# Usage: warren_verify_installation <vm_name>
warren_verify_installation() {
  local vm_name="$1"

  log_verbose "Verifying Warren installation on ${vm_name}"

  # Check if warren binary exists and is executable
  if lima_exec "${vm_name}" "test -x /usr/local/bin/warren" &>/dev/null; then
    log_success "Warren installed on ${vm_name}: /usr/local/bin/warren"
    return 0
  else
    log_error "Warren not properly installed on ${vm_name}"
    return 1
  fi
}

# ============================================================================
# MANAGER OPERATIONS
# ============================================================================

# Initialize first manager (leader)
# Usage: warren_init_manager <vm_name> <api_host> <api_port>
warren_init_manager() {
  local vm_name="$1"
  local api_host="${2:-0.0.0.0}"
  local api_port="${3:-8080}"

  log_step "Initializing manager on ${vm_name}"

  # Initialize cluster in background (daemonize)
  progress_start "Initializing Warren cluster"
  if lima_exec_root "${vm_name}" "nohup warren cluster init --api-addr ${api_host}:${api_port} > /var/log/warren/manager.log 2>&1 &" &>/dev/null; then
    progress_done
  else
    progress_fail
    log_error "Failed to initialize cluster on ${vm_name}"
    return 1
  fi

  # Wait for cluster to be ready
  log_verbose "Waiting for Warren manager to start..."
  sleep 5

  # Verify cluster is running
  if warren_verify_manager_ready "${vm_name}"; then
    log_success "Manager initialized successfully on ${vm_name}"
    return 0
  else
    log_error "Manager failed to become ready on ${vm_name}"
    return 1
  fi
}

# Generate manager join token
# Usage: warren_get_manager_token <leader_vm_name>
warren_get_manager_token() {
  local leader_vm="$1"

  log_verbose "Getting manager join token from ${leader_vm}"

  local token=$(lima_exec_root "${leader_vm}" "warren node token manager" 2>/dev/null | grep -v "^$" | tail -1)

  if [[ -z "$token" ]]; then
    log_error "Failed to get manager token from ${leader_vm}"
    return 1
  fi

  echo "$token"
  return 0
}

# Generate worker join token
# Usage: warren_get_worker_token <leader_vm_name>
warren_get_worker_token() {
  local leader_vm="$1"

  log_verbose "Getting worker join token from ${leader_vm}"

  local token=$(lima_exec_root "${leader_vm}" "warren node token worker" 2>/dev/null | grep -v "^$" | tail -1)

  if [[ -z "$token" ]]; then
    log_error "Failed to get worker token from ${leader_vm}"
    return 1
  fi

  echo "$token"
  return 0
}

# Join additional manager to cluster
# Usage: warren_join_manager <vm_name> <manager_address> <token>
warren_join_manager() {
  local vm_name="$1"
  local manager_addr="$2"
  local token="$3"

  log_step "Joining manager ${vm_name} to cluster"

  progress_start "Joining manager ${vm_name}"
  if lima_exec_root "${vm_name}" "warren cluster join --manager ${manager_addr} --token ${token}" &>/dev/null; then
    progress_done
  else
    progress_fail
    log_error "Failed to join manager ${vm_name}"
    return 1
  fi

  # Wait for manager to be ready
  sleep 3

  if warren_verify_manager_ready "${vm_name}"; then
    log_success "Manager ${vm_name} joined successfully"
    return 0
  else
    log_error "Manager ${vm_name} failed to become ready"
    return 1
  fi
}

# Verify manager is ready
# Usage: warren_verify_manager_ready <vm_name>
warren_verify_manager_ready() {
  local vm_name="$1"
  local max_attempts=15
  local attempt=0

  log_verbose "Verifying manager ${vm_name} is ready"

  while [[ $attempt -lt $max_attempts ]]; do
    # Check if manager process is running and has been up for at least 3 seconds
    # This ensures initialization completed
    if lima_exec_root "${vm_name}" "pgrep -f 'warren cluster init'" &>/dev/null; then
      # Wait a bit longer on first detection to ensure full startup
      if [[ $attempt -eq 0 ]]; then
        log_verbose "Warren process detected, waiting for full initialization..."
        sleep 3
      fi
      log_verbose "Manager ${vm_name} is ready (process running)"
      return 0
    fi

    sleep 2
    attempt=$((attempt + 1))
  done

  log_error "Manager ${vm_name} did not become ready within timeout"
  return 1
}

# ============================================================================
# WORKER OPERATIONS
# ============================================================================

# Start worker and join cluster
# Usage: warren_start_worker <vm_name> <manager_address> <token>
warren_start_worker() {
  local vm_name="$1"
  local manager_addr="$2"
  local token="$3"

  log_step "Starting worker ${vm_name}"

  # Start worker in background (daemonize)
  progress_start "Starting worker ${vm_name}"
  if lima_exec_root "${vm_name}" "nohup warren worker start --manager ${manager_addr} --token ${token} > /var/log/warren/worker.log 2>&1 &" &>/dev/null; then
    progress_done
  else
    progress_fail
    log_error "Failed to start worker ${vm_name}"
    return 1
  fi

  # Wait for worker to be ready
  log_verbose "Waiting for Warren worker to start..."
  sleep 5

  if warren_verify_worker_ready "${vm_name}"; then
    log_success "Worker ${vm_name} started successfully"
    return 0
  else
    log_error "Worker ${vm_name} failed to become ready"
    return 1
  fi
}

# Verify worker is ready
# Usage: warren_verify_worker_ready <vm_name>
warren_verify_worker_ready() {
  local vm_name="$1"
  local max_attempts=30
  local attempt=0

  log_verbose "Verifying worker ${vm_name} is ready"

  while [[ $attempt -lt $max_attempts ]]; do
    # Check if worker process is running
    if lima_exec_root "${vm_name}" "pgrep -f 'warren worker'" &>/dev/null; then
      log_verbose "Worker ${vm_name} is ready"
      return 0
    fi

    sleep 2
    attempt=$((attempt + 1))
  done

  log_error "Worker ${vm_name} did not become ready within timeout"
  return 1
}

# ============================================================================
# CLUSTER OPERATIONS
# ============================================================================

# Get cluster status
# Usage: warren_cluster_status <leader_vm_name>
warren_cluster_status() {
  local leader_vm="$1"

  log_verbose "Getting cluster status from ${leader_vm}"

  lima_exec "${leader_vm}" "warren node list --manager localhost:8080"
}

# Verify cluster health
# Usage: warren_verify_cluster_health <leader_vm_name> <expected_managers> <expected_workers> <manager_token>
warren_verify_cluster_health() {
  local leader_vm="$1"
  local expected_managers="$2"
  local expected_workers="$3"
  local manager_token="$4"

  log_step "Verifying cluster health"

  # Check if Warren manager process is still running
  progress_start "Checking cluster health"
  if ! lima_exec_root "${leader_vm}" "pgrep -f 'warren cluster init'" &>/dev/null; then
    progress_fail
    log_error "Cluster health check failed: Warren manager not running"
    return 1
  fi
  progress_done

  # Initialize CLI certificates for node list command
  # Get the manager's actual bind address
  local manager_addr="0.0.0.0:8080"

  log_verbose "Initializing CLI certificates..."
  local init_output
  init_output=$(lima_exec "${leader_vm}" "warren init --manager ${manager_addr} --token ${manager_token}" 2>&1)
  local init_exit_code=$?

  if [[ $init_exit_code -ne 0 ]]; then
    log_warning "Failed to initialize CLI certificates (exit code: $init_exit_code)"
    if [[ "$VERBOSE" == "true" ]]; then
      log_verbose "CLI init output: $init_output"
    fi
  else
    log_verbose "CLI certificates initialized successfully"
  fi

  # Wait for nodes to register (they need time to join and sync)
  log_verbose "Waiting for nodes to register in cluster..."
  sleep 10

  # Count nodes by role with retry logic
  local max_attempts=10
  local attempt=0
  local manager_count=0
  local worker_count=0

  progress_start "Verifying node registration"
  while [[ $attempt -lt $max_attempts ]]; do
    # Get raw output from node list command
    local node_list_output
    node_list_output=$(lima_exec "${leader_vm}" "warren node list --manager localhost:8080 2>&1" || echo "")

    # Debug: show raw output on first attempt
    if [[ $attempt -eq 0 && "$VERBOSE" == "true" ]]; then
      log_verbose "Raw node list output:"
      log_verbose "$node_list_output"
    fi

    # Count manager and worker nodes by counting lines that contain the role
    # grep -c "manager" counts ALL lines containing "manager", including headers
    # We need to count only data rows, so we'll grep for lines that look like node entries
    if [[ -n "$node_list_output" ]]; then
      # Count lines that contain "manager" or "worker" in the ROLE column
      # Assuming format: ID NAME ROLE STATUS
      # We'll count lines where the role field (3rd column) matches
      manager_count=$(echo "$node_list_output" | awk '$3 == "manager" {count++} END {print count+0}')
      worker_count=$(echo "$node_list_output" | awk '$3 == "worker" {count++} END {print count+0}')
    else
      manager_count="0"
      worker_count="0"
    fi

    # Clean up any whitespace or newlines
    manager_count=$(echo "$manager_count" | tr -d ' \t\n\r')
    worker_count=$(echo "$worker_count" | tr -d ' \t\n\r')

    # Ensure we have valid numbers (fallback to 0 if not)
    if ! [[ "$manager_count" =~ ^[0-9]+$ ]]; then
      manager_count="0"
    fi
    if ! [[ "$worker_count" =~ ^[0-9]+$ ]]; then
      worker_count="0"
    fi

    log_verbose "Attempt $((attempt + 1))/$max_attempts: Managers: ${manager_count}/${expected_managers}, Workers: ${worker_count}/${expected_workers}"

    # Check if we have the expected counts
    if [[ "$manager_count" -eq "$expected_managers" ]] && [[ "$worker_count" -eq "$expected_workers" ]]; then
      progress_done
      log_success "Cluster health verification passed"
      log_info "Cluster status:"
      log_info "  Managers: ${manager_count}/${expected_managers}"
      log_info "  Workers: ${worker_count}/${expected_workers}"
      return 0
    fi

    sleep 3
    attempt=$((attempt + 1))
  done

  # Failed to reach expected counts
  progress_fail
  log_error "Cluster health verification failed: unexpected node counts after ${max_attempts} attempts"
  log_info "Final cluster status:"
  log_info "  Managers: ${manager_count}/${expected_managers}"
  log_info "  Workers: ${worker_count}/${expected_workers}"
  return 1
}

# Get Raft leader info
# Usage: warren_get_leader <leader_vm_name>
warren_get_leader() {
  local leader_vm="$1"

  log_verbose "Getting Raft leader info from ${leader_vm}"

  # Query Raft metrics for leader
  local metrics=$(lima_exec "${leader_vm}" "curl -s http://localhost:9090/metrics 2>/dev/null" || echo "")

  if [[ -n "$metrics" ]]; then
    local is_leader=$(echo "$metrics" | grep "^warren_raft_is_leader" | awk '{print $2}')
    if [[ "$is_leader" == "1" ]]; then
      echo "${leader_vm}"
      return 0
    fi
  fi

  echo "unknown"
  return 1
}

# Deploy test service
# Usage: warren_deploy_test_service <leader_vm_name> <service_name> <image> <replicas>
warren_deploy_test_service() {
  local leader_vm="$1"
  local service_name="$2"
  local image="$3"
  local replicas="${4:-1}"

  log_step "Deploying test service: ${service_name}"

  progress_start "Creating service ${service_name}"
  if lima_exec "${leader_vm}" "warren service create ${service_name} --image ${image} --replicas ${replicas} --manager localhost:8080" &>/dev/null; then
    progress_done
    log_success "Service ${service_name} deployed successfully"
    return 0
  else
    progress_fail
    log_error "Failed to deploy service ${service_name}"
    return 1
  fi
}

# Get service status
# Usage: warren_get_service_status <leader_vm_name> <service_name>
warren_get_service_status() {
  local leader_vm="$1"
  local service_name="$2"

  log_verbose "Getting status for service ${service_name}"

  lima_exec "${leader_vm}" "warren service inspect ${service_name} --manager localhost:8080"
}

# Wait for service to be ready
# Usage: warren_wait_service_ready <leader_vm_name> <service_name> <timeout_seconds>
warren_wait_service_ready() {
  local leader_vm="$1"
  local service_name="$2"
  local timeout="${3:-60}"
  local elapsed=0

  log_verbose "Waiting for service ${service_name} to be ready"

  progress_start "Waiting for service ${service_name}"

  while [[ $elapsed -lt $timeout ]]; do
    # Get service status
    local status=$(warren_get_service_status "${leader_vm}" "${service_name}" 2>/dev/null || echo "")

    # Check if service has expected replicas running
    if echo "$status" | grep -q "Status.*Running"; then
      progress_done
      log_success "Service ${service_name} is ready"
      return 0
    fi

    sleep 5
    elapsed=$((elapsed + 5))
  done

  progress_fail
  log_error "Service ${service_name} did not become ready within ${timeout} seconds"
  return 1
}

# Delete service
# Usage: warren_delete_service <leader_vm_name> <service_name>
warren_delete_service() {
  local leader_vm="$1"
  local service_name="$2"

  log_verbose "Deleting service ${service_name}"

  if lima_exec "${leader_vm}" "warren service delete ${service_name} --manager localhost:8080" &>/dev/null; then
    log_success "Service ${service_name} deleted"
    return 0
  else
    log_error "Failed to delete service ${service_name}"
    return 1
  fi
}

# ============================================================================
# CLUSTER INITIALIZATION ORCHESTRATION
# ============================================================================

# Initialize complete Warren cluster
# Usage: warren_initialize_cluster <num_managers> <num_workers> <cpus> <memory>
warren_initialize_cluster() {
  local num_managers="$1"
  local num_workers="$2"

  log_step "Initializing Warren cluster"

  # Get leader VM name
  local leader_vm="${VM_NAME_PREFIX}-manager-1"
  local leader_ip=$(lima_get_ip "${leader_vm}")

  if [[ -z "$leader_ip" ]]; then
    log_error "Failed to get leader IP address"
    return 1
  fi

  log_info "Leader VM: ${leader_vm} (${leader_ip})"

  # Initialize first manager
  if ! warren_init_manager "${leader_vm}"; then
    return 1
  fi

  # Get manager token
  log_info "Generating manager join token..."
  local manager_token=$(warren_get_manager_token "${leader_vm}")
  if [[ -z "$manager_token" ]]; then
    return 1
  fi

  # Join additional managers
  if [[ $num_managers -gt 1 ]]; then
    log_info "Joining additional managers..."
    for ((i=2; i<=num_managers; i++)); do
      local vm_name="${VM_NAME_PREFIX}-manager-${i}"
      if ! warren_join_manager "${vm_name}" "${leader_ip}:8080" "${manager_token}"; then
        return 1
      fi
    done
  fi

  # Get worker token
  log_info "Generating worker join token..."
  local worker_token=$(warren_get_worker_token "${leader_vm}")
  if [[ -z "$worker_token" ]]; then
    return 1
  fi

  # Start workers
  log_info "Starting workers..."
  for ((i=1; i<=num_workers; i++)); do
    local vm_name="${VM_NAME_PREFIX}-worker-${i}"
    if ! warren_start_worker "${vm_name}" "${leader_ip}:8080" "${worker_token}"; then
      return 1
    fi
  done

  # Verify cluster health
  if ! warren_verify_cluster_health "${leader_vm}" "$num_managers" "$num_workers" "${manager_token}"; then
    return 1
  fi

  log_success "Warren cluster initialized successfully"
  log_info "Cluster details:"
  log_info "  Leader: ${leader_vm} (${leader_ip}:8080)"
  log_info "  Managers: ${num_managers}"
  log_info "  Workers: ${num_workers}"
  log_info ""
  log_info "Access cluster:"
  log_info "  limactl shell ${leader_vm}"
  log_info "  warren node list --manager localhost:8080"

  return 0
}

# ============================================================================
# EXPORT FUNCTIONS
# ============================================================================

# Make functions available to sourcing scripts
export -f warren_download_binary
export -f warren_install_on_vm
export -f warren_verify_installation
export -f warren_init_manager
export -f warren_get_manager_token
export -f warren_get_worker_token
export -f warren_join_manager
export -f warren_verify_manager_ready
export -f warren_start_worker
export -f warren_verify_worker_ready
export -f warren_cluster_status
export -f warren_verify_cluster_health
export -f warren_get_leader
export -f warren_deploy_test_service
export -f warren_get_service_status
export -f warren_wait_service_ready
export -f warren_delete_service
export -f warren_initialize_cluster

log_verbose "Warren utilities library loaded"
