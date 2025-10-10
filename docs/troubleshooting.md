# Troubleshooting Warren

This guide helps you diagnose and fix common issues with Warren.

## Table of Contents

- [Cluster Issues](#cluster-issues)
- [Service Issues](#service-issues)
- [Network Issues](#network-issues)
- [Storage Issues](#storage-issues)
- [Performance Issues](#performance-issues)
- [Debugging Tools](#debugging-tools)
- [Getting Help](#getting-help)

---

## Cluster Issues

### Issue: Cluster Init Fails

**Symptoms:**
```
Error: failed to initialize Raft: permission denied
```

**Causes:**
- Insufficient permissions
- Data directory doesn't exist or not writable

**Solutions:**

```bash
# Run with sudo
sudo warren cluster init

# Check data directory permissions
sudo mkdir -p /var/lib/warren/data
sudo chown $(whoami):$(whoami) /var/lib/warren/data

# Or specify custom data directory
warren cluster init --data-dir $HOME/warren-data
```

---

### Issue: Manager Cannot Join Cluster

**Symptoms:**
```
Error: failed to join cluster: connection refused
```

**Causes:**
- Leader manager not reachable
- Firewall blocking ports
- Wrong manager address

**Solutions:**

```bash
# 1. Verify leader is running
warren cluster info --manager <leader-ip>:8080

# 2. Check connectivity
ping <leader-ip>
telnet <leader-ip> 8080

# 3. Check firewall
sudo iptables -L -n | grep 8080
sudo ufw status

# 4. Allow required ports
sudo ufw allow 8080/tcp    # API
sudo ufw allow 51820/udp   # WireGuard

# 5. Use correct address
warren manager join \
  --token <token> \
  --manager <correct-leader-ip>:8080 \
  --advertise-addr <this-node-ip>:8080
```

---

### Issue: No Raft Leader Elected

**Symptoms:**
```
Error: no leader available
Cluster status: no quorum
```

**Causes:**
- Less than majority of managers online
- Network partition
- Raft corruption

**Solutions:**

```bash
# 1. Check cluster info
warren cluster info --manager <any-manager>:8080

# 2. Verify quorum (3 managers = need 2)
# If only 1 manager online out of 3, cluster is unavailable

# 3. Start additional managers
sudo systemctl start warren-manager

# 4. Check manager logs
sudo journalctl -u warren-manager -n 100

# 5. Look for election logs
sudo journalctl -u warren-manager | grep "election"

# 6. If corrupted, restore from backup
sudo systemctl stop warren-manager
sudo rm -rf /var/lib/warren/data/
sudo tar xzf warren-backup.tar.gz -C /
sudo systemctl start warren-manager
```

---

### Issue: Worker Cannot Connect to Manager

**Symptoms:**
```
Error: failed to connect to manager: connection refused
Worker heartbeat failed
```

**Causes:**
- Manager not running
- Wrong manager address
- Firewall blocking connection

**Solutions:**

```bash
# 1. Verify manager is running
warren cluster info --manager <manager-ip>:8080

# 2. Test connectivity
curl http://<manager-ip>:8080/metrics

# 3. Check worker logs
sudo journalctl -u warren-worker -f

# 4. Restart worker with correct address
sudo warren worker start --manager <correct-manager-ip>:8080

# 5. Check firewall rules
sudo iptables -L -n | grep 8080
```

---

## Service Issues

### Issue: Service Creation Fails

**Symptoms:**
```
Error: failed to create service: image not found
Error: invalid image format
```

**Causes:**
- Image doesn't exist
- Image name typo
- No available workers

**Solutions:**

```bash
# 1. Verify image name
docker pull nginx:latest  # Test image exists

# 2. Check workers available
warren node list --manager 127.0.0.1:8080

# 3. Use full image name
warren service create nginx --image docker.io/library/nginx:latest

# 4. Check manager logs
sudo journalctl -u warren-manager | grep "service create"
```

---

### Issue: Tasks Not Scheduling

**Symptoms:**
```
Service status: 0/3 replicas
Tasks: pending (no tasks scheduled)
```

**Causes:**
- No workers available
- Workers not healthy
- Scheduler error

**Solutions:**

```bash
# 1. Check workers
warren node list --manager 127.0.0.1:8080
# Should show workers with status "ready"

# 2. Start workers if none
sudo warren worker start --manager 127.0.0.1:8080

# 3. Check worker connectivity
# On worker node:
sudo journalctl -u warren-worker -f

# 4. Check scheduler logs (manager)
sudo journalctl -u warren-manager | grep "scheduler"

# 5. Check for errors
sudo journalctl -u warren-manager | grep ERROR
```

---

### Issue: Tasks Stuck in Pending

**Symptoms:**
```
task-nginx-1  (pending)  pending  nginx:latest  -  5m ago
```

**Causes:**
- Worker offline
- Image pull failing
- Volume doesn't exist (for volume-bound tasks)

**Solutions:**

```bash
# 1. Check task details
warren service inspect nginx --manager 127.0.0.1:8080

# 2. Check worker logs (on assigned worker)
sudo journalctl -u warren-worker | grep "task-nginx-1"

# 3. Try pulling image manually on worker
sudo ctr -n warren image pull docker.io/library/nginx:latest

# 4. If volume issue, create volume
warren volume create <volume-name> --manager 127.0.0.1:8080

# 5. Delete and recreate service
warren service delete nginx
warren service create nginx --image nginx:latest --replicas 3
```

---

### Issue: Tasks Keep Failing

**Symptoms:**
```
task-app-1  worker-1  failed  app:v1.0  -  2m ago
task-app-2  worker-2  failed  app:v1.0  -  1m ago
```

**Causes:**
- Container crashes immediately
- Application error
- Missing dependencies (secrets, volumes)

**Solutions:**

```bash
# 1. Check container logs (future feature)
# Currently: Check worker logs
sudo journalctl -u warren-worker | grep "task-app"

# 2. Test image locally
sudo ctr -n warren run --rm docker.io/myapp:v1.0 test-container

# 3. Check if secrets exist
warren secret list --manager 127.0.0.1:8080

# 4. Check if volumes exist
warren volume list --manager 127.0.0.1:8080

# 5. Exec into container (if running)
sudo ctr -n warren tasks list
sudo ctr -n warren tasks exec -t --exec-id debug <task-id> sh

# 6. Check container exit code
sudo ctr -n warren tasks list
# Look for exit code (non-zero = error)
```

---

### Issue: Service Not Accessible

**Symptoms:**
```
curl: (7) Failed to connect to 10.1.0.5:80: No route to host
```

**Causes:**
- Service VIP not configured
- iptables rules missing
- No running tasks

**Solutions:**

```bash
# 1. Check service has running tasks
warren service inspect myapp --manager 127.0.0.1:8080
# Should show tasks in "running" state

# 2. Get service VIP
warren service inspect myapp | grep "VIP"

# 3. Check iptables NAT rules
sudo iptables -t nat -L -n -v | grep <vip>

# 4. Try accessing from manager node
# (VIP routing may not work from outside cluster)

# 5. Check WireGuard connectivity
ping <task-ip>
```

---

## Network Issues

### Issue: WireGuard Interface Not Created

**Symptoms:**
```
ip link show warren0
Device "warren0" does not exist
```

**Causes:**
- WireGuard kernel module not loaded
- Permissions issue
- Warren not managing WireGuard yet

**Solutions:**

```bash
# 1. Check if WireGuard kernel module exists
lsmod | grep wireguard

# 2. Install WireGuard if missing
# Ubuntu/Debian
sudo apt install wireguard

# 3. Load kernel module
sudo modprobe wireguard

# 4. Verify Warren created interface
sudo wg show warren0

# 5. If still missing, check Warren logs
sudo journalctl -u warren-manager | grep wireguard
```

---

### Issue: Containers Cannot Communicate

**Symptoms:**
```
# From container A
ping 10.0.1.5
100% packet loss
```

**Causes:**
- WireGuard tunnel down
- Routing issue
- Firewall blocking

**Solutions:**

```bash
# 1. Check WireGuard status
sudo wg show warren0

# 2. Verify peers are connected
sudo wg show warren0 latest-handshakes
# Handshakes should be < 2 minutes ago

# 3. Check routes
ip route show | grep warren0

# 4. Test WireGuard tunnel
ping <peer-wireguard-ip>

# 5. Check firewall
sudo iptables -L -n | grep warren0

# 6. Restart WireGuard (last resort)
sudo systemctl restart warren-manager
# Or warren-worker
```

---

### Issue: High Network Latency

**Symptoms:**
```
ping 10.0.1.5
100ms latency (expected < 10ms)
```

**Causes:**
- WireGuard encryption overhead
- Network congestion
- Bad routing

**Solutions:**

```bash
# 1. Check WireGuard is using kernel module (fast)
lsmod | grep wireguard
# If missing, using userspace (slower)

# 2. Check network bandwidth
iperf3 -c <peer-ip>

# 3. Check for packet loss
ping -c 100 <peer-ip>

# 4. Increase WireGuard MTU (if needed)
# Edit /etc/wireguard/warren0.conf
MTU = 1420  # Default
# Try: MTU = 1380

# 5. Check CPU usage (encryption load)
top
# Look for high CPU on wireguard process
```

---

## Storage Issues

### Issue: Volume Mount Fails

**Symptoms:**
```
Error: failed to mount volume: no such file or directory
```

**Causes:**
- Volume doesn't exist on node
- Wrong mount path
- Permissions issue

**Solutions:**

```bash
# 1. Check volume exists
warren volume inspect db-data --manager 127.0.0.1:8080

# 2. Check volume on worker node
sudo ls -la /var/lib/warren/volumes/db-data/

# 3. Create directory if missing
sudo mkdir -p /var/lib/warren/volumes/db-data/

# 4. Check permissions
sudo chown -R 999:999 /var/lib/warren/volumes/db-data/
# (999:999 = postgres UID/GID, adjust for your app)

# 5. Check containerd mounts
sudo ctr -n warren containers list
sudo ctr -n warren tasks list
```

---

### Issue: Secret Not Mounted

**Symptoms:**
```
# Inside container
cat /run/secrets/db-password
cat: can't open '/run/secrets/db-password': No such file or directory
```

**Causes:**
- Secret doesn't exist
- Service not referencing secret
- Worker failed to fetch secret

**Solutions:**

```bash
# 1. Check secret exists
warren secret list --manager 127.0.0.1:8080

# 2. Check service references secret
warren service inspect myapp --manager 127.0.0.1:8080
# Should show secret in "Secrets:" section

# 3. Check worker logs
sudo journalctl -u warren-worker | grep secret

# 4. Verify tmpfs mount
# Inside container:
mount | grep /run/secrets
# Should show tmpfs mounted

# 5. Recreate service
warren service delete myapp
warren service create myapp --secret db-password ...
```

---

### Issue: Volume Full

**Symptoms:**
```
ERROR: No space left on device
```

**Causes:**
- Disk full
- Volume quota reached (future)

**Solutions:**

```bash
# 1. Check disk usage
df -h /var/lib/warren/volumes/

# 2. Check volume size
du -sh /var/lib/warren/volumes/db-data/

# 3. Clean up old data
# (Application-specific)

# 4. Expand disk
# (Infrastructure-specific)

# 5. Move volume to larger disk
sudo systemctl stop warren-worker
sudo rsync -av /var/lib/warren/volumes/ /mnt/large-disk/volumes/
sudo ln -s /mnt/large-disk/volumes /var/lib/warren/volumes
sudo systemctl start warren-worker
```

---

## Performance Issues

### Issue: Slow Service Creation

**Symptoms:**
```
Service creation takes > 30 seconds
```

**Causes:**
- Image pull slow
- Scheduler bottleneck
- Raft commit slow

**Solutions:**

```bash
# 1. Check image size
docker images | grep myapp

# 2. Pre-pull images on workers
sudo ctr -n warren image pull docker.io/myapp:v1.0

# 3. Check manager CPU/memory
top
# Manager using > 80% CPU?

# 4. Enable profiling
sudo warren cluster init --enable-pprof

# 5. Profile scheduler
curl http://manager:9090/debug/pprof/profile?seconds=30 > profile.pb
go tool pprof -http=:8081 profile.pb
```

---

### Issue: High Manager Memory Usage

**Symptoms:**
```
Manager using > 512MB RAM
```

**Causes:**
- Large number of services/tasks
- Raft log growing
- Memory leak (report bug!)

**Solutions:**

```bash
# 1. Check Raft log size
sudo ls -lh /var/lib/warren/data/warren.db

# 2. Check number of services
warren service list --manager 127.0.0.1:8080 | wc -l

# 3. Enable profiling
curl http://manager:9090/debug/pprof/heap > heap.pb
go tool pprof -http=:8081 heap.pb

# 4. Compact Raft log (future feature)
# Currently: Restart manager
sudo systemctl restart warren-manager

# 5. Report memory leak
# If persistent, open GitHub issue with pprof profile
```

---

### Issue: Slow Failover

**Symptoms:**
```
Leader failover takes > 10 seconds
```

**Causes:**
- High network latency
- Raft timeouts too conservative

**Solutions:**

```bash
# 1. Check network latency between managers
ping <manager-ip>

# 2. Raft is tuned for < 50ms RTT
# If WAN deployment, increase timeouts (future config)

# 3. Check for network issues
sudo journalctl -u warren-manager | grep "raft"

# 4. Current timeouts (edge/LAN optimized):
# HeartbeatTimeout: 500ms
# ElectionTimeout: 500ms
# Expected failover: 2-3s

# 5. If > 10s consistently, report bug
```

---

## Debugging Tools

### View Logs

**Manager logs:**
```bash
# Follow logs
sudo journalctl -u warren-manager -f

# Last 100 lines
sudo journalctl -u warren-manager -n 100

# Search for errors
sudo journalctl -u warren-manager | grep ERROR

# Filter by time
sudo journalctl -u warren-manager --since "1 hour ago"
```

**Worker logs:**
```bash
sudo journalctl -u warren-worker -f
sudo journalctl -u warren-worker -n 100
sudo journalctl -u warren-worker | grep ERROR
```

### Check Metrics

```bash
# Manager metrics
curl http://manager:9090/metrics

# Filter for specific metrics
curl http://manager:9090/metrics | grep warren_

# Key metrics:
# - warren_services_total
# - warren_tasks_total
# - warren_nodes_total
# - raft_leader (1 = leader, 0 = follower)
# - raft_peers
```

### Profile Performance

```bash
# Enable profiling
sudo warren cluster init --enable-pprof

# CPU profile (30s)
curl http://manager:9090/debug/pprof/profile?seconds=30 > cpu.pb
go tool pprof -http=:8081 cpu.pb

# Memory profile
curl http://manager:9090/debug/pprof/heap > heap.pb
go tool pprof -http=:8081 heap.pb

# Goroutines
curl http://manager:9090/debug/pprof/goroutine > goroutine.pb
go tool pprof -http=:8081 goroutine.pb
```

### Inspect containerd

```bash
# List Warren containers
sudo ctr -n warren containers list

# List running tasks
sudo ctr -n warren tasks list

# Exec into container
sudo ctr -n warren tasks exec -t --exec-id debug <task-id> sh

# Container logs
sudo ctr -n warren tasks logs <task-id>

# Delete stuck task
sudo ctr -n warren tasks kill <task-id>
sudo ctr -n warren tasks delete <task-id>
sudo ctr -n warren containers delete <container-id>
```

### Inspect WireGuard

```bash
# Show interface
sudo wg show warren0

# Show peers
sudo wg show warren0 peers

# Show handshakes
sudo wg show warren0 latest-handshakes

# Show transfer stats
sudo wg show warren0 transfer

# Capture WireGuard traffic
sudo tcpdump -i warren0 -n
```

### Inspect iptables

```bash
# Show NAT rules (VIP load balancing)
sudo iptables -t nat -L -n -v

# Search for specific VIP
sudo iptables -t nat -L -n -v | grep 10.1.0.5

# Show packet counts
sudo iptables -t nat -L -n -v | grep DNAT
```

---

## Common Error Messages

### "connection refused"

**Meaning**: Cannot connect to manager API

**Fix**: Verify manager is running and address is correct

### "no quorum"

**Meaning**: Less than majority of managers online

**Fix**: Start additional managers to restore quorum

### "image not found"

**Meaning**: Container image doesn't exist or wrong name

**Fix**: Verify image name and pull manually to test

### "no such file or directory"

**Meaning**: Path doesn't exist (volume, secret, etc.)

**Fix**: Create missing resource or fix path

### "permission denied"

**Meaning**: Insufficient permissions

**Fix**: Run with sudo or fix file/directory permissions

### "deadline exceeded"

**Meaning**: Operation timed out

**Fix**: Check network connectivity and manager health

---

## Getting Help

### Before Asking for Help

Collect the following information:

```bash
# 1. Warren version
warren --version

# 2. System info
uname -a
cat /etc/os-release

# 3. Cluster state
warren cluster info --manager 127.0.0.1:8080
warren node list --manager 127.0.0.1:8080
warren service list --manager 127.0.0.1:8080

# 4. Recent logs
sudo journalctl -u warren-manager -n 200 > manager.log
sudo journalctl -u warren-worker -n 200 > worker.log

# 5. Network state
sudo wg show warren0 > wireguard.txt
sudo iptables-save > iptables.txt

# 6. Metrics
curl http://manager:9090/metrics > metrics.txt
```

### Community Support

- **GitHub Discussions**: [Community Q&A](https://github.com/cuemby/warren/discussions)
  - Search existing discussions first
  - Use "Q&A" category for questions
  - Provide system info and logs

- **GitHub Issues**: [Bug Reports](https://github.com/cuemby/warren/issues)
  - For confirmed bugs only
  - Use issue template
  - Include reproduction steps

- **Email**: opensource@cuemby.com
  - For security issues
  - For private support requests

### Commercial Support

Contact opensource@cuemby.com for:

- Priority support
- Custom feature development
- Consulting and training
- Production deployment assistance

---

## Frequently Asked Questions

**Q: Can I run Warren on a single node?**

A: Yes, for development. For production, use 3+ managers for HA.

**Q: How do I upgrade Warren?**

A: Download new binary, stop service, replace binary, start service. Rolling upgrades documented in M6.

**Q: Can I migrate from Docker Swarm?**

A: Yes! See [Migration from Docker Swarm](migration/from-docker-swarm.md).

**Q: Does Warren support Kubernetes manifests?**

A: No, Warren uses its own YAML format. See [CLI Reference](cli-reference.md#warren-apply).

**Q: How do I backup my cluster?**

A: Backup `/var/lib/warren/data/warren.db` from managers. See [High Availability](concepts/high-availability.md#backup-and-disaster-recovery).

**Q: Can I run Warren with Docker Swarm/Kubernetes?**

A: Yes, they use different namespaces in containerd. But don't mix workloads.

**Q: What's the maximum cluster size?**

A: Tested up to 100 nodes. Theoretical limit much higher.

**Q: Does Warren support Windows?**

A: Not yet. Linux and macOS only (containerd requirement).

---

## Still Having Issues?

If this guide didn't solve your problem:

1. Search [GitHub Discussions](https://github.com/cuemby/warren/discussions)
2. Ask in [Q&A Category](https://github.com/cuemby/warren/discussions/categories/q-a)
3. File a [Bug Report](https://github.com/cuemby/warren/issues/new?template=bug_report.md)

**Include:**
- Warren version
- Operating system
- Cluster configuration
- Steps to reproduce
- Logs and error messages

---

**Happy orchestrating!** 🚀
