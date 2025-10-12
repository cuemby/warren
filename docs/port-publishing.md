# Port Publishing

Warren supports publishing container ports to make services accessible from outside the cluster.

## Overview

Port publishing allows external traffic to reach containers running inside the Warren cluster. Warren currently supports **host mode** port publishing, where ports are published only on the node running the container.

## Host Mode Port Publishing

In host mode, ports are published on the worker node where the container is scheduled. Traffic to the published port on that node is forwarded to the container.

### How it Works

1. **Service Creation**: When you create a service with `--publish`, Warren stores the port mapping
2. **Task Scheduling**: The scheduler assigns the task to a worker node
3. **Container Start**: The worker starts the container and gets its IP address
4. **Port Forwarding**: The worker sets up iptables rules to forward traffic from the host port to the container
5. **Cleanup**: When the container stops, the iptables rules are automatically removed

### Implementation Details

Warren uses iptables DNAT (Destination Network Address Translation) to forward traffic:

```bash
# Forward host port to container IP
iptables -t nat -A PREROUTING -p tcp --dport 8080 -j DNAT --to-destination 10.88.0.2:80

# Allow forwarded traffic
iptables -A FORWARD -d 10.88.0.2 -p tcp --dport 80 -j ACCEPT

# Masquerade for return traffic
iptables -t nat -A POSTROUTING -s 10.88.0.2 -j MASQUERADE
```

### Components

- **pkg/network/hostports.go**: HostPortPublisher manages iptables rules
- **pkg/runtime/containerd.go**: GetContainerIP() retrieves container IP from network namespace
- **pkg/worker/worker.go**: Integration with task lifecycle (publish on start, unpublish on stop)

## Usage

### Create a Service with Published Ports

```bash
# Publish single port (TCP by default)
warren service create nginx --image nginx:alpine --publish 8080:80

# Publish multiple ports
warren service create web --image myapp:latest \
  --publish 8080:80 \
  --publish 8443:443

# Specify protocol explicitly
warren service create dns --image coredns:latest \
  --publish 53:53/udp

# Set publish mode (host is default)
warren service create api --image api:latest \
  --publish 3000:3000 \
  --publish-mode host
```

### Port Mapping Format

Port mappings use the format: `[host_port]:[container_port][/protocol]`

- **host_port**: Port on the host to listen on
- **container_port**: Port inside the container
- **protocol**: `tcp` (default) or `udp`

Examples:
- `8080:80` - Map host port 8080 to container port 80 (TCP)
- `53:53/udp` - Map UDP port 53
- `443:8443/tcp` - Map host port 443 to container port 8443 (TCP)

## Accessing Published Ports

With host mode publishing, you access the service using the worker node's IP address and the published port:

```bash
# If service is running on worker at 192.168.1.100
curl http://192.168.1.100:8080
```

To find which node is running the service:

```bash
# List services and their task assignments
warren service ps nginx
```

## Limitations

### Host Mode Limitations

1. **Node-Specific Access**: Traffic must be sent to the specific worker node running the container
2. **Port Conflicts**: Cannot run multiple replicas with the same published port on one node
3. **No Load Balancing**: No automatic distribution across replicas

### Current Limitations

1. **Ingress Mode Not Yet Implemented**: Routing mesh across all cluster nodes is planned for a future release
2. **IPv4 Only**: IPv6 support is not yet implemented
3. **Linux Only**: Port publishing requires Linux iptables

## Future Enhancements

### Ingress Mode (Planned)

In ingress mode, published ports will be available on **all** cluster nodes, with automatic routing to any replica:

```bash
# Future: ingress mode with routing mesh
warren service create api --image api:latest \
  --publish 8080:80 \
  --publish-mode ingress \
  --replicas 3
```

This will allow:
- Accessing the service from any node in the cluster
- Automatic load balancing across replicas
- No port conflicts even with multiple replicas

### Other Planned Features

- **Port Ranges**: Support publishing port ranges (e.g., `8000-8010:8000-8010`)
- **Dynamic Ports**: Automatically allocate host ports
- **IPv6 Support**: Publish ports on IPv6 addresses
- **Health Check Integration**: Only route to healthy containers

## Troubleshooting

### Check iptables Rules

On the worker node running the container:

```bash
# View NAT table PREROUTING rules
sudo iptables -t nat -L PREROUTING -n -v

# View FORWARD chain
sudo iptables -L FORWARD -n -v

# View NAT table POSTROUTING rules
sudo iptables -t nat -L POSTROUTING -n -v
```

### Check Container IP

```bash
# On the worker node
sudo ctr -n warren tasks ls
sudo nsenter -t <PID> -n ip addr show eth0
```

### Common Issues

**Port not accessible:**
1. Verify container is running: `warren service ps <name>`
2. Check iptables rules exist on the worker node
3. Verify no firewall blocking the port
4. Check container is listening on the port inside

**Port conflict:**
- Cannot bind same host port twice on one node
- Use different host ports or schedule on different nodes

## See Also

- [Networking Architecture](./networking.md)
- [Service Management](./services.md)
- [Health Checks](./health-checks.md)
