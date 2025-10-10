#!/usr/bin/env bash
#
# Warren End-to-End Test Suite
#
# Runs the complete test suite for Warren multi-manager cluster:
# 1. Setup: Create Lima VMs
# 2. Test: Cluster formation (3 managers + 2 workers)
# 3. Test: Leader failover
# 4. Cleanup: Optionally delete VMs
#
# Usage:
#   ./test/lima/test-e2e.sh [--skip-setup] [--skip-cleanup] [--keep-vms]
#
# Options:
#   --skip-setup     Skip VM creation (assume VMs already exist)
#   --skip-cleanup   Don't run cleanup after tests
#   --keep-vms       Keep VMs after cleanup (just stop processes)
#   --help           Show this help message

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Configuration
SKIP_SETUP=false
SKIP_CLEANUP=false
KEEP_VMS=false

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --skip-setup)
      SKIP_SETUP=true
      shift
      ;;
    --skip-cleanup)
      SKIP_CLEANUP=true
      shift
      ;;
    --keep-vms)
      KEEP_VMS=true
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

# Helper functions
log_phase() {
  echo
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo -e "${CYAN}  $*${NC}"
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo
}

log_info() {
  echo -e "${BLUE}ℹ${NC} $*"
}

log_success() {
  echo -e "${GREEN}✓${NC} $*"
}

log_error() {
  echo -e "${RED}✗${NC} $*"
}

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0
START_TIME=$SECONDS

# Run a test script
run_test() {
  local test_name="$1"
  local test_script="$2"

  log_phase "Running: $test_name"

  if "$test_script"; then
    log_success "$test_name passed"
    TESTS_PASSED=$((TESTS_PASSED + 1))
    return 0
  else
    log_error "$test_name failed"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    return 1
  fi
}

# Main execution
main() {
  echo
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "  Warren End-to-End Test Suite"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo
  log_info "Test configuration:"
  log_info "  Skip setup: $SKIP_SETUP"
  log_info "  Skip cleanup: $SKIP_CLEANUP"
  log_info "  Keep VMs: $KEEP_VMS"
  echo

  # Phase 1: Setup
  if [[ "$SKIP_SETUP" == "false" ]]; then
    log_phase "Phase 1: Environment Setup"
    if "${SCRIPT_DIR}/setup.sh" --clean; then
      log_success "Environment setup complete"
    else
      log_error "Environment setup failed"
      exit 1
    fi
  else
    log_info "Skipping setup (--skip-setup)"
  fi

  # Phase 2: Cluster formation test
  if ! run_test "Cluster Formation Test" "${SCRIPT_DIR}/test-cluster.sh"; then
    log_error "Cluster formation failed, aborting remaining tests"
    exit 1
  fi

  # Phase 3: Leader failover test
  run_test "Leader Failover Test" "${SCRIPT_DIR}/test-failover.sh" || true

  # Phase 4: Cleanup
  if [[ "$SKIP_CLEANUP" == "false" ]]; then
    log_phase "Phase 4: Cleanup"

    local cleanup_args=()
    if [[ "$KEEP_VMS" == "true" ]]; then
      cleanup_args+=("--keep-vms")
    fi

    if "${SCRIPT_DIR}/cleanup.sh" "${cleanup_args[@]}" --force; then
      log_success "Cleanup complete"
    else
      log_error "Cleanup failed"
    fi
  else
    log_info "Skipping cleanup (--skip-cleanup)"
  fi

  # Summary
  local duration=$((SECONDS - START_TIME))
  local minutes=$((duration / 60))
  local seconds=$((duration % 60))

  echo
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "  Test Suite Summary"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo
  echo -e "Tests Passed:  ${GREEN}$TESTS_PASSED${NC}"
  echo -e "Tests Failed:  ${RED}$TESTS_FAILED${NC}"
  echo -e "Total Time:    ${minutes}m ${seconds}s"
  echo

  if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "${GREEN}✓ All tests passed!${NC}"
    echo
    exit 0
  else
    echo -e "${RED}✗ Some tests failed${NC}"
    echo
    exit 1
  fi
}

main
