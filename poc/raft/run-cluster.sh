#!/bin/bash

# Simple script to run 3-node Raft cluster for testing
# Usage: ./run-cluster.sh

set -e

echo "Starting 3-node Raft cluster..."
echo "Press Ctrl+C to stop all nodes"
echo ""

# Clean up old data
rm -rf /tmp/raft-node*

# Trap to kill all background processes on exit
trap 'kill $(jobs -p) 2>/dev/null' EXIT

# Start node1 (bootstrap)
echo "Starting node1 (leader)..."
go run . -id node1 -addr 127.0.0.1:8001 &
NODE1_PID=$!

# Wait for node1 to bootstrap
sleep 3

# Start node2
echo "Starting node2..."
go run . -id node2 -addr 127.0.0.1:8002 -join 127.0.0.1:8001 &
NODE2_PID=$!

# Start node3
echo "Starting node3..."
go run . -id node3 -addr 127.0.0.1:8003 -join 127.0.0.1:8001 &
NODE3_PID=$!

echo ""
echo "Cluster started!"
echo "Node 1: 127.0.0.1:8001 (PID: $NODE1_PID)"
echo "Node 2: 127.0.0.1:8002 (PID: $NODE2_PID)"
echo "Node 3: 127.0.0.1:8003 (PID: $NODE3_PID)"
echo ""
echo "To test failover, kill the leader node:"
echo "  kill $NODE1_PID"
echo ""
echo "Press Ctrl+C to stop all nodes"

# Wait for all background jobs
wait
