# Warren Raft Configuration Tuning

This document explains Warren's Raft consensus configuration and tuning for edge deployments.

## Overview

Warren uses [HashiCorp Raft](https://github.com/hashicorp/raft) for distributed consensus across manager nodes. The Raft configuration has been tuned for edge and LAN deployments where low latency and fast failover are prioritized over WAN tolerance.

## Configuration

### Timeout Values

| Parameter | Default | Warren | Reason |
|-----------|---------|---------|--------|
| HeartbeatTimeout | 1000ms | 500ms | Faster failure detection |
| ElectionTimeout | 1000ms | 500ms | Faster leader elections |
| CommitTimeout | 50ms | 50ms | Not critical for failover |
| LeaderLeaseTimeout | 500ms | 250ms | Faster lease expiration |

### Failover Timeline

With Warren's tuned configuration:

1. **Leader Failure** (t=0s)
   - Leader stops sending heartbeats

2. **Failure Detection** (t=500ms)
   - Followers detect missing heartbeats after 500ms (HeartbeatTimeout)
   - Followers wait for LeaderLeaseTimeout (250ms) to expire

3. **Election Start** (t=750ms)
   - Follower transitions to candidate state
   - Requests votes from other managers

4. **Election Complete** (t=1.0-1.5s)
   - Majority of managers vote
   - New leader elected
   - ElectionTimeout (500ms) + network latency

5. **Leader Established** (t=1.5-2.0s)
   - New leader sends first heartbeat
   - Cluster operational again

**Total Failover Time: ~2-3 seconds** (well under 10s target)

## Trade-offs

### Benefits

✅ **Fast Failover**: 2-3s instead of 10-15s
✅ **Better Availability**: Shorter downtime during leader failures
✅ **Edge-Optimized**: Suitable for LAN and low-latency edge networks
✅ **Responsive**: Quick detection of unhealthy managers

### Costs

⚠️ **More Heartbeats**: Increased network traffic (leader sends heartbeats every 250ms)
⚠️ **False Positives**: Network hiccups may trigger unnecessary elections
⚠️ **Not WAN-Safe**: Higher latency networks (>100ms RTT) may cause instability

## When to Adjust

### Faster Failover (<1s)

For even faster failover in extremely low-latency environments:

```go
config.HeartbeatTimeout = 250 * time.Millisecond
config.ElectionTimeout = 250 * time.Millisecond
config.LeaderLeaseTimeout = 125 * time.Millisecond
```

**Failover time: ~500ms-1s**

**Warning**: Only use in very stable, low-latency networks. Risk of split-brain increases.

### Slower Failover (WAN)

For WAN deployments with higher latency:

```go
config.HeartbeatTimeout = 2000 * time.Millisecond
config.ElectionTimeout = 2000 * time.Millisecond
config.LeaderLeaseTimeout = 1000 * time.Millisecond
```

**Failover time: ~5-8s**

**Better stability** in high-latency or unreliable networks.

### Default (Conservative)

To revert to Raft's conservative defaults:

```go
config.HeartbeatTimeout = 1000 * time.Millisecond
config.ElectionTimeout = 1000 * time.Millisecond
config.LeaderLeaseTimeout = 500 * time.Millisecond
```

**Failover time: ~3-5s**

## Network Requirements

Warren's tuned configuration assumes:

- **RTT < 50ms** between managers (LAN or edge)
- **Packet loss < 1%** (reliable network)
- **Stable connectivity** (no frequent network partitions)

If your environment doesn't meet these requirements, consider using slower timeouts.

## Testing Failover

### Manual Test

1. **Start 3-manager cluster:**
```bash
# Manager 1 (leader)
warren cluster init

# Manager 2
warren manager join --token <token> --leader <addr>

# Manager 3
warren manager join --token <token> --leader <addr>
```

2. **Identify leader:**
```bash
warren cluster info | grep Leader
```

3. **Kill leader:**
```bash
# On leader manager VM
sudo pkill warren
```

4. **Measure failover time:**
```bash
# Start timer
time=$(date +%s)

# Wait for new leader
while true; do
  if warren cluster info 2>/dev/null | grep -q "Leader"; then
    elapsed=$(($(date +%s) - time))
    echo "Failover completed in ${elapsed}s"
    break
  fi
  sleep 0.5
done
```

5. **Expected result:**
   - Failover completes in 2-5s
   - New leader elected
   - Cluster operational

### Automated Test

Use the Lima failover test:

```bash
./test/lima/test-failover.sh
```

This test:
- Creates 3-manager cluster
- Identifies and kills leader
- Measures time until new leader elected
- Validates cluster still operational

**Target:** < 10s failover time
**Tuned:** ~2-3s failover time

## Monitoring

### Raft Metrics

Warren exposes Raft metrics via Prometheus:

```
# Leader status (1 = leader, 0 = follower)
warren_raft_leader

# Number of peers
warren_raft_peers

# Last contact with leader (ms)
warren_raft_last_contact_ms

# Commit index
warren_raft_commit_index

# Applied index
warren_raft_applied_index
```

### Key Metrics to Watch

1. **warren_raft_last_contact_ms**
   - Should be < HeartbeatTimeout (500ms)
   - If consistently > 500ms, network issues or overloaded leader

2. **warren_raft_leader flapping**
   - Frequent changes indicate election storms
   - May need to increase ElectionTimeout

3. **warren_raft_commit_index vs warren_raft_applied_index**
   - Large gap indicates slow FSM application
   - Check manager CPU and disk I/O

## Troubleshooting

### Issue: Frequent Elections

**Symptoms:**
- Leader changes multiple times per minute
- Logs show repeated elections
- API requests fail with "no leader" errors

**Causes:**
- Network latency too high for current timeouts
- Overloaded manager (CPU/disk saturation)
- Network partitions

**Solutions:**
1. Increase timeouts (double HeartbeatTimeout and ElectionTimeout)
2. Check network latency between managers (`ping`)
3. Profile manager CPU and memory
4. Verify no firewall blocking Raft port (7946)

### Issue: Slow Failover (>10s)

**Symptoms:**
- Failover takes longer than expected
- Cluster unavailable for extended period

**Causes:**
- Timeouts not applied (check binary version)
- Only 2 managers (no quorum after leader fails)
- Network partition isolating managers

**Solutions:**
1. Verify Warren version includes tuned timeouts
2. Use 3 or 5 managers (odd number for quorum)
3. Check network connectivity between all managers
4. Review Raft logs for election failures

### Issue: Split Brain

**Symptoms:**
- Two leaders simultaneously
- Cluster state inconsistent
- Services behaving erratically

**Causes:**
- Network partition separating managers
- Timeouts too aggressive for network
- Clock skew between managers

**Solutions:**
1. Immediately stop one leader manually
2. Increase timeouts to reduce false positive elections
3. Ensure NTP synchronized across all managers
4. Fix network partition (firewall rules, routing)

## References

- [HashiCorp Raft Library](https://github.com/hashicorp/raft)
- [Raft Consensus Algorithm](https://raft.github.io/)
- [Raft Paper](https://raft.github.io/raft.pdf)
- [Warren Performance Targets](../specs/prd.md#performance-targets)

## Configuration Location

Raft configuration is set in:
- `pkg/manager/manager.go` - Bootstrap() function
- `pkg/manager/manager.go` - Join() function

Both methods use identical timeout configuration to ensure consistency across the cluster.

---

**Last Updated**: 2025-10-10
**Warren Version**: dev
**Raft Library**: hashicorp/raft v1.x
