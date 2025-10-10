#!/usr/bin/env bash
#
# Warren Load Testing Script
#
# This script runs comprehensive load tests against a Warren cluster to validate:
# - API throughput and latency
# - Scheduler performance
# - Memory usage under load
# - Cluster stability with many services/tasks
#
# Usage:
#   ./test/lima/test-load.sh [options]
#
# Options:
#   --scale small      Small load test (1 manager, 2 workers, 50 services)
#   --scale medium     Medium load test (3 managers, 5 workers, 200 services)
#   --scale large      Large load test (3 managers, 10 workers, 1000 services)
#   --services N       Number of services to create (overrides scale)
#   --replicas N       Replicas per service (default: 3)
#   --profile          Enable pprof profiling
#   --help             Show this help

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Default configuration
SCALE="small"
NUM_SERVICES=""
REPLICAS_PER_SERVICE=3
ENABLE_PROFILE=false
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WARREN_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Test results
START_TIME=""
END_TIME=""
TOTAL_SERVICES=0
TOTAL_TASKS=0
FAILURES=0
TEST_RUNNING=false

# Trap handler for cleanup on interrupt
cleanup_on_exit() {
  if [[ "$TEST_RUNNING" == "true" ]]; then
    echo
    log_warning "Test interrupted! Starting cleanup..."
    cleanup_services
    log_info "Cleanup complete. Exiting."
  fi
  exit 1
}

# Set trap for SIGINT and SIGTERM
trap cleanup_on_exit INT TERM

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --scale)
      SCALE="$2"
      shift 2
      ;;
    --services)
      NUM_SERVICES="$2"
      shift 2
      ;;
    --replicas)
      REPLICAS_PER_SERVICE="$2"
      shift 2
      ;;
    --profile)
      ENABLE_PROFILE=true
      shift
      ;;
    --help)
      grep '^#' "$0" | sed 's/^# //' | sed 's/^#//'
      exit 0
      ;;
    *)
      echo -e "${RED}Error: Unknown option $1${NC}"
      exit 1
      ;;
  esac
done

# Set defaults based on scale if services not specified
if [[ -z "$NUM_SERVICES" ]]; then
  case "$SCALE" in
    small)
      NUM_SERVICES=50
      ;;
    medium)
      NUM_SERVICES=200
      ;;
    large)
      NUM_SERVICES=1000
      ;;
    *)
      echo -e "${RED}Error: Invalid scale: $SCALE (use small, medium, or large)${NC}"
      exit 1
      ;;
  esac
fi

TOTAL_TASKS=$((NUM_SERVICES * REPLICAS_PER_SERVICE))

# Helper functions
log_info() {
  echo -e "${BLUE}ℹ${NC} $*"
}

log_success() {
  echo -e "${GREEN}✓${NC} $*"
}

log_warning() {
  echo -e "${YELLOW}⚠${NC} $*"
}

log_error() {
  echo -e "${RED}✗${NC} $*"
}

log_step() {
  echo
  echo -e "${CYAN}━━━ $* ━━━${NC}"
}

# Check if manager is running
check_manager() {
  log_info "Checking manager availability..."

  if ! limactl list | grep -q "warren-manager-1.*Running"; then
    log_error "Manager VM not running. Please run ./test/lima/setup.sh first"
    exit 1
  fi

  # Check if Warren manager is running
  if ! limactl shell warren-manager-1 sudo pgrep -f "warren" &> /dev/null; then
    log_error "Warren manager process not running. Please start the cluster first"
    exit 1
  fi

  log_success "Manager is running"
}

# Check available workers
check_workers() {
  log_info "Checking available workers..."

  local worker_count
  worker_count=$(limactl list | grep -c "warren-worker-.*Running" || true)

  if [[ $worker_count -eq 0 ]]; then
    log_error "No worker VMs running. Please run ./test/lima/setup.sh first"
    exit 1
  fi

  log_success "Found $worker_count worker VM(s)"

  # Check how many are registered with the cluster
  local registered_workers
  registered_workers=$(limactl shell warren-manager-1 sudo /tmp/lima/warren/bin/warren node list 2>/dev/null | grep -c "worker" || true)

  if [[ $registered_workers -gt 0 ]]; then
    log_success "$registered_workers worker(s) registered with cluster"
  else
    log_warning "No workers registered yet. Start workers with: warren worker start --manager <addr>"
  fi
}

# Capture memory usage
capture_memory() {
  local vm_name="$1"
  local process_name="$2"

  # Get RSS memory in MB
  local mem_mb
  mem_mb=$(limactl shell "$vm_name" bash -c "ps aux | grep -v grep | grep '$process_name' | awk '{print \$6}' | head -1" 2>/dev/null || echo "0")
  mem_mb=$((mem_mb / 1024))

  echo "$mem_mb"
}

# Monitor cluster during load test
monitor_cluster() {
  local duration="$1"
  local output_file="$2"

  log_info "Monitoring cluster for ${duration}s..."

  local end_time=$((SECONDS + duration))

  echo "timestamp,manager_mem_mb,services,tasks,nodes" > "$output_file"

  while [[ $SECONDS -lt $end_time ]]; do
    local timestamp=$(date +%s)
    local manager_mem=$(capture_memory "warren-manager-1" "warren")

    # Get cluster stats from API
    local services=$(limactl shell warren-manager-1 sudo /tmp/lima/warren/bin/warren service list 2>/dev/null | tail -n +2 | wc -l || echo "0")
    local nodes=$(limactl shell warren-manager-1 sudo /tmp/lima/warren/bin/warren node list 2>/dev/null | tail -n +2 | wc -l || echo "0")

    # For tasks, we'd need to add a task list command or query the API
    local tasks=$((services * REPLICAS_PER_SERVICE))

    echo "$timestamp,$manager_mem,$services,$tasks,$nodes" >> "$output_file"

    sleep 5
  done

  log_success "Monitoring complete. Results in $output_file"
}

# Create services in batch
create_services_batch() {
  local start_idx="$1"
  local end_idx="$2"
  local batch_num="$3"

  log_info "  Batch $batch_num: Creating services $start_idx to $end_idx..."

  local batch_start=$SECONDS
  local failed=0

  for i in $(seq "$start_idx" "$end_idx"); do
    if ! limactl shell warren-manager-1 sudo /tmp/lima/warren/bin/warren service create \
      "load-test-$i" \
      --image nginx:latest \
      --replicas "$REPLICAS_PER_SERVICE" \
      --manager localhost:8080 &> /dev/null; then
      ((failed++))
    fi
  done

  local batch_duration=$((SECONDS - batch_start))
  local batch_size=$((end_idx - start_idx + 1))

  if [[ $failed -eq 0 ]]; then
    log_success "  Batch $batch_num: Created $batch_size services in ${batch_duration}s ($(echo "scale=2; $batch_size / $batch_duration" | bc) svc/s)"
  else
    log_warning "  Batch $batch_num: Created $((batch_size - failed)) services, $failed failed in ${batch_duration}s"
    FAILURES=$((FAILURES + failed))
  fi
}

# Create all test services
create_test_services() {
  log_step "Creating $NUM_SERVICES test services (${REPLICAS_PER_SERVICE} replicas each = $TOTAL_TASKS tasks)"

  START_TIME=$SECONDS

  # Create in batches of 50 to avoid overwhelming the API
  local batch_size=50
  local num_batches=$(( (NUM_SERVICES + batch_size - 1) / batch_size ))

  for batch in $(seq 1 "$num_batches"); do
    local start_idx=$(( (batch - 1) * batch_size + 1 ))
    local end_idx=$(( batch * batch_size ))
    if [[ $end_idx -gt $NUM_SERVICES ]]; then
      end_idx=$NUM_SERVICES
    fi

    create_services_batch "$start_idx" "$end_idx" "$batch"
  done

  local creation_duration=$((SECONDS - START_TIME))

  log_success "Service creation complete in ${creation_duration}s"
  log_info "  Total services: $NUM_SERVICES"
  log_info "  Total tasks: $TOTAL_TASKS"
  log_info "  Creation rate: $(echo "scale=2; $NUM_SERVICES / $creation_duration" | bc) services/s"

  if [[ $FAILURES -gt 0 ]]; then
    log_warning "  Failed creations: $FAILURES"
  fi
}

# Wait for tasks to be scheduled
wait_for_scheduling() {
  log_step "Waiting for scheduler to process tasks..."

  local wait_start=$SECONDS
  local max_wait=300  # 5 minutes max

  log_info "Waiting for tasks to be scheduled (max ${max_wait}s)..."

  # Just wait a bit for the scheduler to run
  # In a real test, we'd query the API for task states
  sleep 30

  local wait_duration=$((SECONDS - wait_start))
  log_success "Scheduling wait complete (${wait_duration}s)"

  # Show current cluster state
  log_info "Cluster state:"
  limactl shell warren-manager-1 sudo /tmp/lima/warren/bin/warren service list 2>/dev/null | tail -5 || true
}

# Measure API latency
measure_api_latency() {
  log_step "Measuring API latency"

  local num_requests=100
  local latencies=()

  log_info "Sending $num_requests API requests..."

  for i in $(seq 1 "$num_requests"); do
    local start=$(date +%s%N)
    limactl shell warren-manager-1 sudo /tmp/lima/warren/bin/warren service list --manager localhost:8080 &> /dev/null || true
    local end=$(date +%s%N)
    local latency_ms=$(( (end - start) / 1000000 ))
    latencies+=("$latency_ms")
  done

  # Calculate statistics
  local sum=0
  local min=999999
  local max=0

  for lat in "${latencies[@]}"; do
    sum=$((sum + lat))
    if [[ $lat -lt $min ]]; then min=$lat; fi
    if [[ $lat -gt $max ]]; then max=$lat; fi
  done

  local avg=$((sum / num_requests))

  log_success "API latency (service list):"
  log_info "  Requests: $num_requests"
  log_info "  Average: ${avg}ms"
  log_info "  Min: ${min}ms"
  log_info "  Max: ${max}ms"
}

# Capture memory profiles
capture_profiles() {
  if [[ "$ENABLE_PROFILE" != "true" ]]; then
    return
  fi

  log_step "Capturing memory profiles"

  local profile_dir="${WARREN_ROOT}/test/load-profiles-$(date +%Y%m%d-%H%M%S)"
  mkdir -p "$profile_dir"

  log_info "Profile directory: $profile_dir"

  # Capture manager heap profile
  log_info "Capturing manager heap profile..."
  if limactl shell warren-manager-1 curl -s http://127.0.0.1:9090/debug/pprof/heap > "${profile_dir}/manager_heap.prof"; then
    log_success "Manager heap profile saved"
  else
    log_warning "Failed to capture manager heap profile (is --enable-pprof set?)"
  fi

  # Capture manager CPU profile
  log_info "Capturing manager CPU profile (30s)..."
  if limactl shell warren-manager-1 curl -s "http://127.0.0.1:9090/debug/pprof/profile?seconds=30" > "${profile_dir}/manager_cpu.prof"; then
    log_success "Manager CPU profile saved"
  else
    log_warning "Failed to capture manager CPU profile"
  fi

  # Capture worker profiles if workers exist
  local worker_count
  worker_count=$(limactl list | grep -c "warren-worker-.*Running" || true)

  if [[ $worker_count -gt 0 ]]; then
    log_info "Capturing worker-1 heap profile..."
    if limactl shell warren-worker-1 curl -s http://127.0.0.1:6060/debug/pprof/heap > "${profile_dir}/worker1_heap.prof" 2>/dev/null; then
      log_success "Worker-1 heap profile saved"
    else
      log_warning "Failed to capture worker heap profile (is --enable-pprof set?)"
    fi
  fi

  log_success "Profiles saved to $profile_dir"
  echo
  log_info "Analyze profiles with:"
  log_info "  go tool pprof ${profile_dir}/manager_heap.prof"
  log_info "  go tool pprof ${profile_dir}/manager_cpu.prof"
}

# Cleanup test services
cleanup_services() {
  log_step "Cleaning up test services"

  log_info "Deleting $NUM_SERVICES test services..."

  local cleanup_start=$SECONDS
  local deleted=0
  local failed=0

  for i in $(seq 1 "$NUM_SERVICES"); do
    if limactl shell warren-manager-1 sudo /tmp/lima/warren/bin/warren service delete "load-test-$i" --manager localhost:8080 &> /dev/null; then
      ((deleted++))
    else
      ((failed++))
    fi

    # Show progress every 50 services
    if [[ $((i % 50)) -eq 0 ]]; then
      log_info "  Deleted $i / $NUM_SERVICES services..."
    fi
  done

  local cleanup_duration=$((SECONDS - cleanup_start))

  log_success "Cleanup complete in ${cleanup_duration}s"
  log_info "  Services deleted: $deleted / $NUM_SERVICES"

  if [[ $failed -gt 0 ]]; then
    log_warning "  Failed deletions: $failed"
  fi

  # Verify cleanup - check that no load-test services remain
  log_info "Verifying cleanup..."
  local remaining
  remaining=$(limactl shell warren-manager-1 sudo /tmp/lima/warren/bin/warren service list 2>/dev/null | grep -c "load-test-" || true)

  if [[ $remaining -eq 0 ]]; then
    log_success "All test services removed successfully"
  else
    log_warning "$remaining load-test services still present"
    log_info "Attempting forced cleanup..."

    # Try to delete remaining services one more time
    limactl shell warren-manager-1 sudo /tmp/lima/warren/bin/warren service list 2>/dev/null | grep "load-test-" | awk '{print $1}' | while read -r svc; do
      limactl shell warren-manager-1 sudo /tmp/lima/warren/bin/warren service delete "$svc" --manager localhost:8080 &> /dev/null || true
    done

    # Final check
    remaining=$(limactl shell warren-manager-1 sudo /tmp/lima/warren/bin/warren service list 2>/dev/null | grep -c "load-test-" || true)
    if [[ $remaining -eq 0 ]]; then
      log_success "Forced cleanup successful"
    else
      log_error "Unable to remove $remaining services. Manual cleanup required."
    fi
  fi
}

# Generate test report
generate_report() {
  END_TIME=$SECONDS
  local total_duration=$((END_TIME - START_TIME))

  log_step "Load Test Report"

  echo
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "  Warren Load Test Results"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo
  echo "Test Configuration:"
  echo "  Scale: $SCALE"
  echo "  Services Created: $NUM_SERVICES"
  echo "  Replicas per Service: $REPLICAS_PER_SERVICE"
  echo "  Total Tasks: $TOTAL_TASKS"
  echo "  Profiling: $ENABLE_PROFILE"
  echo
  echo "Results:"
  echo "  Total Duration: ${total_duration}s"
  echo "  Service Creation Rate: $(echo "scale=2; $NUM_SERVICES / $total_duration" | bc) services/s"
  echo "  Failed Creations: $FAILURES"
  echo

  # Memory usage
  local manager_mem=$(capture_memory "warren-manager-1" "warren")
  echo "Memory Usage:"
  echo "  Manager: ${manager_mem}MB"

  if [[ $manager_mem -gt 256 ]]; then
    log_warning "  Manager memory exceeds target (256MB)"
  else
    log_success "  Manager memory within target"
  fi

  echo
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo
}

# Main test execution
main() {
  echo
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "  Warren Load Test"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo
  log_info "Configuration:"
  log_info "  Scale: $SCALE"
  log_info "  Services: $NUM_SERVICES"
  log_info "  Replicas per service: $REPLICAS_PER_SERVICE"
  log_info "  Total tasks: $TOTAL_TASKS"
  log_info "  Profiling: $ENABLE_PROFILE"
  echo

  # Pre-flight checks
  check_manager
  check_workers

  # Set flag so trap knows to cleanup
  TEST_RUNNING=true

  # Run load test
  create_test_services
  wait_for_scheduling

  # Measurements
  measure_api_latency

  # Profiling
  if [[ "$ENABLE_PROFILE" == "true" ]]; then
    capture_profiles
  fi

  # Cleanup
  cleanup_services

  # Clear flag - cleanup complete
  TEST_RUNNING=false

  # Report
  generate_report

  log_success "Load test complete!"
}

# Run main
main
