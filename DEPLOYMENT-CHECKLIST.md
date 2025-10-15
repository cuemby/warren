# Warren v1.6.0 Production Deployment Checklist

**Quick Reference** for deploying Warren to production.
**Full Guide**: [docs/production-deployment-guide.md](docs/production-deployment-guide.md)

**New in v1.6.0**: **Hybrid Mode** - Manager + worker in single process! Single-node deployments work out-of-the-box without separate worker process. Perfect for edge deployments and development.

**Features**:
- ✨ Hybrid mode by default (manager + worker combined)
- 🔒 CLI with mTLS after `warren init` (write operations enabled)
- 📡 Unix socket for read-only local access
- ⚙️  Opt-out with `--manager-only` for dedicated control plane

---

## Quick Start (Single Node - Hybrid Mode)

**For edge deployments or testing** (5 minutes):

```bash
# 1. Initialize cluster (hybrid mode: manager + worker in one process)
sudo warren cluster init

# 2. Initialize CLI for write operations
warren init --manager 127.0.0.1:8080 --token <CLI_TOKEN>

# 3. Deploy service
warren service create test --image nginx:latest --replicas 2

# 4. Verify
warren node list  # Shows "hybrid" role
warren service list
```

✅ **Done!** Your node is running both control plane and workloads.

---

## Pre-Deployment (Production HA Cluster - 30 minutes)

### Infrastructure
- [ ] 3 manager nodes ready (2 CPU, 2GB RAM, 10GB disk each)
- [ ] 3+ worker nodes ready (4 CPU, 4GB RAM, 20GB disk each)
  - **OR** use hybrid nodes (manager + worker combined)
- [ ] Network connectivity verified between all nodes
- [ ] DNS configured (optional)

### Ports
- [ ] 8080 open (Warren API)
- [ ] 9090 open (Metrics/Health)
- [ ] 8000 open (HTTP Ingress)
- [ ] 8443 open (HTTPS Ingress)
- [ ] 7946 open (Raft)

### Software
- [ ] containerd v1.7+ installed on all nodes
- [ ] Warren v1.6.0 binary installed on all nodes: `/usr/local/bin/warren`
- [ ] curl and jq installed for testing

### Documentation
- [ ] [E2E Validation](docs/e2e-validation.md) reviewed
- [ ] [Monitoring Guide](docs/monitoring.md) reviewed
- [ ] [Operational Runbooks](docs/operational-runbooks.md) accessible

---

## Deployment (1-2 hours)

### Manager-1 (Leader)

**Option A: Manager-Only (Dedicated Control Plane)**
```bash
# For dedicated control plane (no workloads on managers)
warren cluster init --node-id manager-1 \
  --bind-addr <IP>:7946 \
  --api-addr 0.0.0.0:8080 \
  --manager-only
```

**Option B: Hybrid Mode (Manager + Worker)**
```bash
# Manager can also run workloads (v1.6.0+ default)
warren cluster init --node-id manager-1 \
  --bind-addr <IP>:7946 \
  --api-addr 0.0.0.0:8080
```

**Post-Init Steps:**
- [ ] Save manager token: `echo "MANAGER_TOKEN=<token>" >> /etc/warren/tokens`
- [ ] Save worker token: `echo "WORKER_TOKEN=<token>" >> /etc/warren/tokens`
- [ ] Save CLI token: `echo "CLI_TOKEN=<token>" >> /etc/warren/tokens`
- [ ] Verify health: `curl http://localhost:9090/health | jq .`
- [ ] Verify ready: `curl http://localhost:9090/ready | jq .`
- [ ] Check leadership: `curl http://localhost:9090/metrics | grep warren_raft_is_leader`
- [ ] Check role: `warren node list` (should show "manager" or "hybrid")

### Manager-2 & Manager-3
- [ ] Copy tokens to nodes: `scp /etc/warren/tokens manager-{2,3}:/etc/warren/`
- [ ] Run: `warren cluster join --node-id manager-{2,3} --role manager --leader <manager-1-ip>:8080 --token $MANAGER_TOKEN ...`
- [ ] Verify: `warren node list` (works via Unix socket! should show 3 managers)
- [ ] Check quorum: `curl http://localhost:9090/metrics | grep warren_raft_peers_total` (should be 3)

### Workers (All)
- [ ] Run: `warren cluster join --node-id worker-{1,2,3} --role worker --leader <manager-1-ip>:8080 --token $WORKER_TOKEN ...`
- [ ] Verify: `warren node list` (works via Unix socket! should show 3 managers + 3 workers, all "ready")

---

## Monitoring Setup (30 minutes)

### Prometheus
- [ ] Create prometheus.yml with Warren scrape configs
- [ ] Deploy Prometheus: `docker run -d -p 9091:9090 -v prometheus.yml:/etc/prometheus/prometheus.yml prom/prometheus`
- [ ] Verify scraping: Open http://<prometheus>:9091/targets
- [ ] All Warren targets should be "UP"

### Alerts
- [ ] Create alert_rules.yml from [monitoring.md](docs/monitoring.md)
- [ ] Configure 10 critical alerts:
  * WarrenNoLeader
  * WarrenQuorumLost
  * WarrenHighSchedulingLatency
  * WarrenHighContainerFailureRate
  * WarrenNoReadyWorkers
  * (5 more from guide)
- [ ] Test alerts: `promtool check rules alert_rules.yml`

### Health Checks
- [ ] Create health check script: `/usr/local/bin/warren-health-check.sh`
- [ ] Test: `/usr/local/bin/warren-health-check.sh`
- [ ] Schedule cron: `*/5 * * * * /usr/local/bin/warren-health-check.sh`

---

## Validation (30 minutes)

### Quick Smoke Test
- [ ] Cluster: `warren node list` (Unix socket - no setup needed!) → 3 managers + 3 workers, all ready ✅
- [ ] Health: `curl http://manager-1:9090/health` → 200 OK ✅
- [ ] Ready: `curl http://manager-1:9090/ready` → 200 OK ✅
- [ ] Metrics: `curl http://manager-1:9090/metrics | head -20` → Valid Prometheus format ✅

### Service Test
- [ ] Deploy: `warren service create nginx --image nginx:latest --replicas 3 --ports 80:8080`
- [ ] Wait 30s, check: `warren task ls --service nginx` → 3 running ✅
- [ ] Test: `curl http://worker-1:8080` → nginx welcome ✅
- [ ] Scale: `warren service update nginx --replicas 6` → 6 running ✅
- [ ] Cleanup: `warren service delete nginx`

### Full E2E (Follow [e2e-validation.md](docs/e2e-validation.md))
- [ ] Phase 1: Cluster Health ✅
- [ ] Phase 2: Service Deployment ✅
- [ ] Phase 3: Scaling Operations ✅
- [ ] Phase 4: Leader Failover ✅
- [ ] Phase 5: Secrets Management ✅
- [ ] Phase 6: Volumes ✅
- [ ] Phase 7: Built-in Ingress ✅
- [ ] Phase 8: Health Monitoring ✅

---

## Post-Deployment (1 hour)

### Systemd Services
- [ ] Create `/etc/systemd/system/warren.service` on all nodes
- [ ] Run: `sudo systemctl enable warren && sudo systemctl start warren`
- [ ] Verify: `sudo systemctl status warren` → active (running) ✅

### Log Rotation
- [ ] Create `/etc/logrotate.d/warren`
- [ ] Test: `sudo logrotate -d /etc/logrotate.d/warren`

### Documentation
- [ ] Fill out `/var/lib/warren/DEPLOYMENT.txt` with:
  * Deployment date
  * Deployed by
  * Cluster configuration
  * Monitoring URLs
  * Any notes or issues

### Baseline Metrics
- [ ] Record normal metric values:
  * `warren_nodes_total{status="ready"}` = 6
  * `warren_raft_peers_total` = 3
  * `warren_raft_is_leader` = 1 (on one manager)
  * API latency p95 < 100ms
  * Scheduling latency p95 < 5s

---

## Success Criteria

Deployment successful when ALL true:
- [x] All nodes status = "ready"
- [x] Health endpoints return 200 OK
- [x] Prometheus scraping all targets (6 targets UP)
- [x] Test service deploys successfully
- [x] Service scales up and down
- [x] Leader failover works (tested)
- [x] All 8 E2E validation phases pass
- [x] Monitoring dashboard shows healthy metrics
- [x] No critical alerts firing

---

## First 24 Hours

### Hour 1
- [ ] Monitor all metrics dashboards
- [ ] Watch for any alerts
- [ ] Verify no errors in logs: `warren logs manager-1 | grep -i error`

### Hour 8
- [ ] Check metric trends (should be stable)
- [ ] Verify no unexpected restarts: `systemctl status warren`
- [ ] Review any warnings in logs

### Hour 24
- [ ] Document baseline performance
- [ ] Fine-tune alert thresholds if needed
- [ ] Deploy first non-critical production service
- [ ] Schedule operational review with team

---

## Rollback Plan

If ANY issue occurs:

1. **Stop immediately**: `warren service delete <service-name>`
2. **Check health**: `curl http://manager-1:9090/health`
3. **Collect logs**: `warren logs manager-1 > /tmp/issue.log`
4. **Capture metrics**: `curl http://manager-1:9090/metrics > /tmp/metrics.txt`
5. **Document**: Add to `/var/lib/warren/DEPLOYMENT.txt`
6. **Escalate**: Contact team, review [troubleshooting.md](docs/troubleshooting.md)

---

## Quick Commands

```bash
# Check cluster (✨ works via Unix socket - no setup needed!)
warren node list

# Check services (✨ works via Unix socket!)
warren service list

# Check tasks (✨ works via Unix socket!)
warren task list

# Cluster info (✨ works via Unix socket!)
warren cluster info

# View logs
warren logs <node-id>

# Health check
curl http://<manager>:9090/health | jq .

# Readiness check
curl http://<manager>:9090/ready | jq .

# Metrics
curl http://<manager>:9090/metrics | head -50
```

**Note**:
- **v1.6.0+**: Hybrid mode by default (manager + worker in one process)
- **Read operations** (list, inspect, info) work via Unix socket without setup
- **Write operations** (create, update, delete) require `warren init` with mTLS
- See [docs/concepts/security.md](docs/concepts/security.md) and [README.md](README.md) Quick Start

---

## References

- **Getting Started**: [docs/getting-started.md](docs/getting-started.md) - Hybrid mode guide
- **Full Deployment Guide**: [docs/production-deployment-guide.md](docs/production-deployment-guide.md)
- **E2E Validation**: [docs/e2e-validation.md](docs/e2e-validation.md)
- **Monitoring**: [docs/monitoring.md](docs/monitoring.md)
- **Troubleshooting**: [docs/troubleshooting.md](docs/troubleshooting.md)
- **Runbooks**: [docs/operational-runbooks.md](docs/operational-runbooks.md)

---

**Version**: v1.6.0
**Last Updated**: 2025-10-15
**Status**: ✅ Production Ready

---

**Deployment Sign-Off**:

- [ ] **Pre-deployment complete**: _________________ (Name/Date)
- [ ] **Deployment complete**: _________________ (Name/Date)
- [ ] **Validation complete**: _________________ (Name/Date)
- [ ] **Production approved**: _________________ (Name/Date)

🎉 **Warren is LIVE!**
