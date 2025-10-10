#!/usr/bin/env bash
#
# Warren Lima Test Environment Setup
#
# This script sets up the Lima-based testing environment for Warren:
# - Installs Lima if not present
# - Creates 3 manager VMs and 2 worker VMs
# - Configures user-v2 networking for VM-to-VM communication
#
# Usage:
#   ./test/lima/setup.sh [--managers N] [--workers N]
#
# Options:
#   --managers N    Number of manager VMs to create (default: 3)
#   --workers N     Number of worker VMs to create (default: 2)
#   --clean         Delete existing VMs before creating new ones
#   --help          Show this help message

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
NUM_MANAGERS=3
NUM_WORKERS=2
CLEAN=false
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WARREN_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
TEMPLATE="${SCRIPT_DIR}/warren.yaml"

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --managers)
      NUM_MANAGERS="$2"
      shift 2
      ;;
    --workers)
      NUM_WORKERS="$2"
      shift 2
      ;;
    --clean)
      CLEAN=true
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

# Check if Lima is installed
check_lima() {
  if ! command -v limactl &> /dev/null; then
    log_error "Lima is not installed!"
    echo
    echo "Please install Lima:"
    echo
    if [[ "$OSTYPE" == "darwin"* ]]; then
      echo "  macOS:"
      echo "    brew install lima"
    else
      echo "  Linux:"
      echo "    See https://lima-vm.io/docs/installation/"
    fi
    echo
    exit 1
  fi

  log_success "Lima is installed: $(limactl --version)"
}

# Check if Warren binary exists
check_warren_binary() {
  if [[ ! -f "${WARREN_ROOT}/bin/warren" ]]; then
    log_warning "Warren binary not found at ${WARREN_ROOT}/bin/warren"
    log_info "Building Warren..."
    cd "${WARREN_ROOT}"
    make build
    if [[ ! -f "${WARREN_ROOT}/bin/warren" ]]; then
      log_error "Failed to build Warren binary"
      exit 1
    fi
    log_success "Warren binary built successfully"
  else
    log_success "Warren binary found"
  fi
}

# Clean up existing VMs
cleanup_vms() {
  log_info "Cleaning up existing Warren test VMs..."

  local vms_to_delete=()

  # Find all warren-* VMs
  while IFS= read -r vm; do
    if [[ "$vm" == warren-* ]]; then
      vms_to_delete+=("$vm")
    fi
  done < <(limactl list --quiet)

  if [[ ${#vms_to_delete[@]} -eq 0 ]]; then
    log_info "No existing Warren VMs to clean up"
    return
  fi

  log_info "Deleting ${#vms_to_delete[@]} VM(s)..."
  for vm in "${vms_to_delete[@]}"; do
    log_info "  Deleting $vm..."
    limactl delete -f "$vm" || true
  done

  log_success "Cleanup complete"
}

# Create a VM
create_vm() {
  local vm_name="$1"
  local vm_type="$2"  # manager or worker

  log_info "Creating VM: $vm_name ($vm_type)"

  # Create VM with template
  limactl create \
    --name="$vm_name" \
    --tty=false \
    "$TEMPLATE"

  # Start VM
  log_info "  Starting $vm_name..."
  limactl start "$vm_name"

  log_success "  $vm_name is ready"
}

# Create all VMs
create_all_vms() {
  log_info "Creating $NUM_MANAGERS manager VMs and $NUM_WORKERS worker VMs..."
  echo

  # Create manager VMs
  for i in $(seq 1 "$NUM_MANAGERS"); do
    create_vm "warren-manager-$i" "manager"
  done

  # Create worker VMs
  for i in $(seq 1 "$NUM_WORKERS"); do
    create_vm "warren-worker-$i" "worker"
  done

  echo
  log_success "All VMs created successfully!"
}

# Show VM status and access information
show_status() {
  echo
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "  Warren Test Environment Status"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo

  # List all warren VMs
  limactl list | grep -E "NAME|warren-" || true

  echo
  echo "VM Networking (user-v2):"
  echo "  Subnet: 192.168.104.0/24"
  echo "  Gateway: 192.168.104.1"
  echo
  echo "VM Hostnames:"
  for i in $(seq 1 "$NUM_MANAGERS"); do
    echo "  manager-$i: lima-warren-manager-$i.internal"
  done
  for i in $(seq 1 "$NUM_WORKERS"); do
    echo "  worker-$i: lima-warren-worker-$i.internal"
  done

  echo
  echo "Access VMs:"
  echo "  limactl shell warren-manager-1"
  echo "  limactl shell warren-worker-1"
  echo
  echo "Run Tests:"
  echo "  ./test/lima/test-cluster.sh     # Test 3-manager cluster"
  echo "  ./test/lima/test-failover.sh    # Test leader failover"
  echo "  ./test/lima/test-e2e.sh         # End-to-end workflow"
  echo
  echo "Cleanup:"
  echo "  ./test/lima/cleanup.sh"
  echo
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# Main execution
main() {
  echo
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "  Warren Lima Test Environment Setup"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo

  log_info "Configuration:"
  log_info "  Managers: $NUM_MANAGERS"
  log_info "  Workers: $NUM_WORKERS"
  log_info "  Template: $TEMPLATE"
  log_info "  Warren Root: $WARREN_ROOT"
  echo

  # Pre-flight checks
  check_lima
  check_warren_binary

  # Clean up if requested
  if [[ "$CLEAN" == "true" ]]; then
    cleanup_vms
    echo
  fi

  # Create VMs
  create_all_vms

  # Show status
  show_status

  log_success "Setup complete! Warren test environment is ready."
}

main
