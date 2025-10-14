# Warren Production Deployment Guide

**Version**: v1.3.1
**Date**: 2025-10-14
**Status**: Production Ready ✅
**Audience**: DevOps/Platform Engineers

---

## Overview

This guide walks you through deploying Warren v1.3.1 to production with monitoring, health checks, and operational readiness validation.

**Deployment Architecture**: 3 managers + 3+ workers (High Availability)

**Estimated Time**: 2-3 hours (first-time deployment)

---

## Table of Contents

- [Pre-Deployment Checklist](#pre-deployment-checklist)
- [Infrastructure Setup](#infrastructure-setup)
- [Warren Deployment](#warren-deployment)
- [Monitoring Setup](#monitoring-setup)
- [Validation](#validation)
- [Post-Deployment](#post-deployment)
- [Rollback Plan](#rollback-plan)

---

## Pre-Deployment Checklist

### ✅ Requirements Verification

**Infrastructure Ready**:
- [ ] 3+ manager nodes provisioned (2 CPU, 2GB RAM, 10GB disk)
- [ ] 3+ worker nodes provisioned (4 CPU, 4GB RAM, 20GB disk)
- [ ] Network connectivity between all nodes
- [ ] DNS resolution configured (optional but recommended)

**Ports Open**:
- [ ] 8080 (Warren API)
- [ ] 9090 (Metrics/Health)
- [ ] 8000 (HTTP Ingress)
- [ ] 8443 (HTTPS Ingress)
- [ ] 7946 (Raft cluster communication)

**Software Installed**:
- [ ] containerd v1.7+ on all nodes
- [ ] Warren binary v1.3.1 on all nodes
- [ ] curl/jq for testing

**Documentation Reviewed**:
- [ ] [E2E Validation Guide](./e2e-validation.md)
- [ ] [Monitoring Guide](./monitoring.md)
- [ ] [Operational Runbooks](./operational-runbooks.md)

---

## Infrastructure Setup

### Step 1: Prepare Nodes

**On ALL nodes** (managers + workers):

```bash
# Update system
sudo apt-get update && sudo apt-get upgrade -y

# Install containerd
sudo apt-get install -y containerd

# Verify containerd
systemctl status containerd

# Create Warren directories
sudo mkdir -p /var/lib/warren
sudo mkdir -p /etc/warren
sudo mkdir -p /var/log/warren

# Set permissions
sudo chown -R $USER:$USER /var/lib/warren
```

### Step 2: Install Warren Binary

**On ALL nodes**:

```bash
# Download Warren v1.3.1
wget https://github.com/cuemby/warren/releases/download/v1.3.1/warren-linux-amd64
chmod +x warren-linux-amd64
sudo mv warren-linux-amd64 /usr/local/bin/warren

# Verify installation
warren version
# Expected: Warren v1.3.1
```

### Step 3: Configure Firewall

**On ALL nodes**:

```bash
# Allow Warren ports
sudo ufw allow 8080/tcp   # API
sudo ufw allow 9090/tcp   # Metrics/Health
sudo ufw allow 8000/tcp   # HTTP Ingress
sudo ufw allow 8443/tcp   # HTTPS Ingress
sudo ufw allow 7946/tcp   # Raft

# Verify rules
sudo ufw status
```

---

## Warren Deployment

### Phase 1: Initialize First Manager

**On manager-1** (192.168.1.10):

```bash
# Initialize cluster
warren cluster init \
  --node-id manager-1 \
  --bind-addr 192.168.1.10:7946 \
  --data-dir /var/lib/warren \
  --api-addr 0.0.0.0:8080 \
  --metrics-addr 0.0.0.0:9090

# Output will show:
# ✅ Cluster initialized
# Manager Join Token: <MANAGER_TOKEN>
# Worker Join Token: <WORKER_TOKEN>

# IMPORTANT: Save these tokens securely!
echo "MANAGER_TOKEN=<token>" >> /etc/warren/tokens
echo "WORKER_TOKEN=<token>" >> /etc/warren/tokens
chmod 600 /etc/warren/tokens
```

**Verify Manager-1**:

```bash
# Check cluster status
warren node ls

# Check health
curl http://localhost:9090/health | jq .
# Expected: {"status":"healthy","timestamp":"...","version":"1.3.1"}

# Check readiness
curl http://localhost:9090/ready | jq .
# Expected: {"status":"ready","checks":{"raft":"leader","storage":"ok"}}

# Check metrics
curl http://localhost:9090/metrics | grep warren_raft_is_leader
# Expected: warren_raft_is_leader 1
```

### Phase 2: Join Additional Managers

**On manager-2** (192.168.1.11):

```bash
# Load tokens
source /etc/warren/tokens

# Join cluster as manager
warren cluster join \
  --node-id manager-2 \
  --bind-addr 192.168.1.11:7946 \
  --data-dir /var/lib/warren \
  --api-addr 0.0.0.0:8080 \
  --metrics-addr 0.0.0.0:9090 \
  --leader 192.168.1.10:8080 \
  --token $MANAGER_TOKEN \
  --role manager

# Verify
warren node ls
curl http://localhost:9090/health | jq .
```

**On manager-3** (192.168.1.12):

```bash
# Same as manager-2, but with different node-id and bind-addr
warren cluster join \
  --node-id manager-3 \
  --bind-addr 192.168.1.12:7946 \
  --data-dir /var/lib/warren \
  --api-addr 0.0.0.0:8080 \
  --metrics-addr 0.0.0.0:9090 \
  --leader 192.168.1.10:8080 \
  --token $MANAGER_TOKEN \
  --role manager

# Verify
warren node ls
curl http://localhost:9090/health | jq .
```

**Verify Raft Quorum**:

```bash
# On any manager
curl http://localhost:9090/metrics | grep warren_raft_peers_total
# Expected: warren_raft_peers_total 3

# Check leadership
warren node ls
# Expected: One manager marked as "leader"
```

### Phase 3: Join Worker Nodes

**On worker-1** (192.168.1.20):

```bash
# Load tokens
source /etc/warren/tokens

# Join as worker
warren cluster join \
  --node-id worker-1 \
  --bind-addr 192.168.1.20:7946 \
  --data-dir /var/lib/warren \
  --leader 192.168.1.10:8080 \
  --token $WORKER_TOKEN \
  --role worker

# Verify
warren node ls
```

**On worker-2 and worker-3**:

```bash
# Repeat with appropriate node-id and bind-addr
warren cluster join \
  --node-id worker-{2,3} \
  --bind-addr 192.168.1.{21,22}:7946 \
  --data-dir /var/lib/warren \
  --leader 192.168.1.10:8080 \
  --token $WORKER_TOKEN \
  --role worker
```

**Verify Full Cluster**:

```bash
# On any manager
warren node ls

# Expected output:
# NODE ID      ROLE     STATUS  ADDRESS
# manager-1    manager  ready   192.168.1.10:7946  (leader)
# manager-2    manager  ready   192.168.1.11:7946
# manager-3    manager  ready   192.168.1.12:7946
# worker-1     worker   ready   192.168.1.20:7946
# worker-2     worker   ready   192.168.1.21:7946
# worker-3     worker   ready   192.168.1.22:7946
```

---

## Monitoring Setup

### Step 1: Deploy Prometheus

**Create prometheus.yml**:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  # Warren Managers
  - job_name: 'warren-managers'
    static_configs:
      - targets:
        - 'manager-1:9090'
        - 'manager-2:9090'
        - 'manager-3:9090'
    metrics_path: '/metrics'

  # Warren Workers
  - job_name: 'warren-workers'
    static_configs:
      - targets:
        - 'worker-1:9090'
        - 'worker-2:9090'
        - 'worker-3:9090'
    metrics_path: '/metrics'
```

**Deploy Prometheus** (on separate monitoring host):

```bash
# Using Docker
docker run -d \
  --name prometheus \
  -p 9091:9090 \
  -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
  prom/prometheus

# Verify Prometheus
curl http://localhost:9091/-/healthy
```

### Step 2: Configure Alerting

**Create alert_rules.yml** (from monitoring.md):

```yaml
groups:
- name: warren
  rules:
  - alert: WarrenNoLeader
    expr: sum(warren_raft_is_leader) == 0
    for: 1m
    labels:
      severity: critical
    annotations:
      summary: "No Raft leader elected"

  - alert: WarrenQuorumLost
    expr: warren_raft_peers_total < 2
    for: 30s
    labels:
      severity: critical
    annotations:
      summary: "Raft quorum lost"

  # Add more alerts from docs/monitoring.md
```

**Update prometheus.yml**:

```yaml
rule_files:
  - 'alert_rules.yml'

alerting:
  alertmanagers:
  - static_configs:
    - targets:
      - 'alertmanager:9093'
```

### Step 3: Setup Health Check Monitoring

**Configure health check scraper**:

```bash
# Create health check script
cat > /usr/local/bin/warren-health-check.sh << 'EOF'
#!/bin/bash
# Health check for Warren managers

for manager in manager-1 manager-2 manager-3; do
  echo "Checking $manager..."

  # Liveness
  curl -f http://$manager:9090/health || echo "❌ $manager health FAILED"

  # Readiness
  curl -f http://$manager:9090/ready || echo "⚠️  $manager not ready"
done
EOF

chmod +x /usr/local/bin/warren-health-check.sh

# Test
/usr/local/bin/warren-health-check.sh
```

---

## Validation

### Run E2E Validation Checklist

Follow the complete [E2E Validation Guide](./e2e-validation.md):

**Quick Validation** (5 minutes):

```bash
# 1. Cluster Health
warren node ls
# Expected: 3 managers + 3 workers, all "ready"

# 2. Deploy Test Service
warren service create nginx \
  --image nginx:latest \
  --replicas 3 \
  --ports 80:8080

# Wait 30 seconds
sleep 30

# 3. Check Service
warren service ls
warren task ls --service nginx
# Expected: 3 tasks "running"

# 4. Test Service
curl http://worker-1:8080
# Expected: nginx welcome page

# 5. Check Metrics
curl http://manager-1:9090/metrics | grep warren_services_total
# Expected: warren_services_total 1

# 6. Scale Test
warren service update nginx --replicas 6
sleep 30
warren task ls --service nginx
# Expected: 6 tasks "running"

# 7. Cleanup
warren service delete nginx
```

**Full Validation** (30 minutes):

Follow all 8 phases in [docs/e2e-validation.md](./e2e-validation.md):
1. ✅ Cluster Health
2. ✅ Service Deployment
3. ✅ Scaling Operations
4. ✅ Leader Failover
5. ✅ Secrets Management
6. ✅ Volumes
7. ✅ Built-in Ingress
8. ✅ Health Monitoring

---

## Post-Deployment

### Configure Systemd Services

**On each manager and worker**, create systemd service:

```bash
# Create service file
sudo cat > /etc/systemd/system/warren.service << 'EOF'
[Unit]
Description=Warren Cluster Manager
After=network.target containerd.service
Requires=containerd.service

[Service]
Type=simple
User=warren
Group=warren
ExecStart=/usr/local/bin/warren start --config /etc/warren/config.yaml
Restart=always
RestartSec=10
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

# Enable and start
sudo systemctl daemon-reload
sudo systemctl enable warren
sudo systemctl start warren

# Check status
sudo systemctl status warren
```

### Set Up Log Rotation

```bash
# Create logrotate config
sudo cat > /etc/logrotate.d/warren << 'EOF'
/var/log/warren/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0640 warren warren
    sharedscripts
    postrotate
        systemctl reload warren > /dev/null 2>&1 || true
    endscript
}
EOF
```

### Document Deployment

```bash
# Create deployment record
cat > /var/lib/warren/DEPLOYMENT.txt << EOF
Warren Production Deployment
============================
Version: v1.3.1
Deployed: $(date)
Deployed by: $USER

Cluster Configuration:
- Managers: 3 (manager-1, manager-2, manager-3)
- Workers: 3 (worker-1, worker-2, worker-3)
- Leader: $(warren node ls | grep leader | awk '{print $1}')

Monitoring:
- Prometheus: http://prometheus-host:9091
- Grafana: http://grafana-host:3000

Documentation:
- E2E Validation: docs/e2e-validation.md
- Monitoring: docs/monitoring.md
- Runbooks: docs/operational-runbooks.md
- Troubleshooting: docs/troubleshooting.md

Notes:
- All health checks passing
- Monitoring configured
- Alerts configured
EOF
```

---

## Rollback Plan

If issues arise during deployment:

### Rollback Step 1: Stop New Services

```bash
# On manager leader
warren service ls
warren service delete <problematic-service>
```

### Rollback Step 2: Verify Cluster Health

```bash
# Check health
curl http://manager-1:9090/health

# Check metrics
curl http://manager-1:9090/metrics | grep warren_raft_is_leader
```

### Rollback Step 3: Document Issues

```bash
# Collect logs
warren logs manager-1 > /tmp/warren-manager-1.log
warren logs worker-1 > /tmp/warren-worker-1.log

# Capture metrics
curl http://manager-1:9090/metrics > /tmp/warren-metrics.txt

# Document issue
echo "Issue encountered: <description>" >> /var/lib/warren/DEPLOYMENT.txt
```

### Rollback Step 4: Revert (if needed)

```bash
# If complete rollback needed
# Stop Warren on all nodes
sudo systemctl stop warren

# Remove data (CAUTION!)
# sudo rm -rf /var/lib/warren/*

# Re-deploy previous version
```

---

## Success Criteria

Deployment is successful when:

- [x] All nodes show "ready" status
- [x] Health endpoints return 200 OK
- [x] Prometheus scraping metrics successfully
- [x] Test service deploys and runs
- [x] Service can scale up and down
- [x] Leader failover works (kill leader, new leader elected)
- [x] All validation phases pass

---

## Next Steps

1. **Monitor for 24 hours**
   - Watch Prometheus dashboards
   - Check for any alerts
   - Monitor metrics trends

2. **Deploy First Production Service**
   - Start with non-critical service
   - Monitor closely
   - Verify end-to-end functionality

3. **Establish Baselines**
   - Document normal metric ranges
   - Set alert thresholds appropriately
   - Document any operational learnings

4. **Schedule Operational Review**
   - Review with team after 1 week
   - Document any issues encountered
   - Update runbooks as needed

---

## Support

**Documentation**:
- E2E Validation: [docs/e2e-validation.md](./e2e-validation.md)
- Monitoring: [docs/monitoring.md](./monitoring.md)
- Troubleshooting: [docs/troubleshooting.md](./troubleshooting.md)
- Runbooks: [docs/operational-runbooks.md](./operational-runbooks.md)

**Issues**: Report at https://github.com/cuemby/warren/issues

---

**Deployment Date**: _______________
**Deployed By**: _______________
**Sign-off**: _______________

✅ Warren v1.3.1 Production Deployment Complete!
