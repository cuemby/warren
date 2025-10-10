#!/usr/bin/env bash
#
# Warren Lima Test Environment Cleanup
#
# This script cleans up the Lima-based test environment:
# - Stops all Warren processes in VMs
# - Deletes all warren-* Lima VMs
# - Removes temporary test files
#
# Usage:
#   ./test/lima/cleanup.sh [--keep-vms] [--force]
#
# Options:
#   --keep-vms    Stop Warren processes but keep VMs running
#   --force       Delete VMs without confirmation
#   --help        Show this help message

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
KEEP_VMS=false
FORCE=false

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --keep-vms)
      KEEP_VMS=true
      shift
      ;;
    --force)
      FORCE=true
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

# Stop Warren processes in VMs
stop_warren_processes() {
  log_info "Stopping Warren processes in all VMs..."

  local vms=()
  while IFS= read -r vm; do
    if [[ "$vm" == warren-* ]]; then
      vms+=("$vm")
    fi
  done < <(limactl list --quiet 2>/dev/null || echo "")

  if [[ ${#vms[@]} -eq 0 ]]; then
    log_info "No Warren VMs found"
    return
  fi

  for vm in "${vms[@]}"; do
    log_info "  Stopping Warren in $vm..."
    limactl shell "$vm" sudo pkill -9 warren 2>/dev/null || true
  done

  log_success "Warren processes stopped"
}

# Delete Lima VMs
delete_vms() {
  log_info "Finding Warren test VMs..."

  local vms=()
  while IFS= read -r vm; do
    if [[ "$vm" == warren-* ]]; then
      vms+=("$vm")
    fi
  done < <(limactl list --quiet 2>/dev/null || echo "")

  if [[ ${#vms[@]} -eq 0 ]]; then
    log_info "No Warren VMs to delete"
    return
  fi

  log_info "Found ${#vms[@]} VM(s) to delete:"
  for vm in "${vms[@]}"; do
    echo "  - $vm"
  done

  # Confirm unless --force
  if [[ "$FORCE" != "true" ]]; then
    echo
    read -p "Delete these VMs? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      log_warning "Cleanup cancelled"
      exit 0
    fi
  fi

  log_info "Deleting VMs..."
  for vm in "${vms[@]}"; do
    log_info "  Deleting $vm..."
    limactl delete -f "$vm" 2>/dev/null || true
  done

  log_success "All VMs deleted"
}

# Clean temporary files
clean_temp_files() {
  log_info "Cleaning temporary test files..."

  local temp_files=(
    /tmp/warren-join-token
    /tmp/warren-current-leader
    /tmp/warren-killed-leader
    /tmp/warren-new-leader
  )

  for file in "${temp_files[@]}"; do
    if [[ -f "$file" ]]; then
      rm -f "$file"
      log_info "  Removed $file"
    fi
  done

  log_success "Temporary files cleaned"
}

# Main execution
main() {
  echo
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "  Warren Lima Test Environment Cleanup"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo

  # Stop Warren processes
  stop_warren_processes
  echo

  # Delete VMs unless --keep-vms
  if [[ "$KEEP_VMS" == "true" ]]; then
    log_info "Keeping VMs (--keep-vms specified)"
  else
    delete_vms
    echo
  fi

  # Clean temp files
  clean_temp_files

  echo
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  log_success "Cleanup complete!"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo

  if [[ "$KEEP_VMS" == "true" ]]; then
    echo "VMs are still running. To delete them, run:"
    echo "  ./test/lima/cleanup.sh"
  else
    echo "All Warren test VMs have been deleted."
    echo
    echo "To recreate the test environment:"
    echo "  ./test/lima/setup.sh"
  fi
  echo
}

main
