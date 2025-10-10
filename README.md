# Warren - Simple Container Orchestrator for Edge

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Status](https://img.shields.io/badge/Status-Alpha-yellow)](https://github.com/cuemby/warren)

> **Warren**: Simple like Docker Swarm, feature-rich like Kubernetes, zero external dependencies.

Warren is a next-generation container orchestration platform built for edge computing with telco-grade reliability. Shipped as a single binary (< 100MB) with no external dependencies.

## ✨ Features

- 🚀 **Single Binary**: Zero external dependencies, < 100MB
- 🔒 **Secure by Default**: Built-in mTLS, encrypted overlay networking (WireGuard)
- 🌍 **Edge-First**: Partition-tolerant, autonomous operation during network failures
- 📦 **Feature-Rich**: Rolling/blue-green/canary deployments, secrets, volumes
- 🎯 **Simple**: Docker Swarm-like UX, minutes to production
- ⚡ **High Performance**: Near-native network speed, < 256MB memory footprint

## 🏗️ Architecture

```
┌─────────────────────────┐
│   Manager Nodes         │
│   (Raft Consensus)      │
│                         │
│  API │ Scheduler │ Sync │
└──────────┬──────────────┘
           │
    ┌──────┴──────┐
    │  WireGuard  │  Encrypted Overlay
    └──────┬──────┘
           │
    ┌──────┴──────────────┐
    ▼                     ▼
┌─────────┐         ┌─────────┐
│ Worker  │         │ Worker  │
│ Node    │         │ Node    │
│         │         │         │
│containerd│       │containerd│
└─────────┘         └─────────┘
```

**Tech Stack**:
- **Language**: Go 1.22+
- **Consensus**: Raft (hashicorp/raft)
- **Container Runtime**: containerd
- **Networking**: WireGuard
- **Storage**: BoltDB

## 🚀 Quick Start

### Installation

**From binary**:
```bash
# Download latest release
curl -L https://github.com/cuemby/warren/releases/download/v1.0.0/warren-linux-amd64 -o warren
chmod +x warren
sudo mv warren /usr/local/bin/

# Verify installation
warren version
```

**From source**:
```bash
git clone https://github.com/cuemby/warren.git
cd warren
make build
sudo make install
```

### Initialize Cluster

```bash
# Start first manager
warren cluster init

# On worker nodes, join the cluster
warren cluster join --token <token-from-manager>
```

### Deploy Your First Service

```bash
# Create a service
warren service create web \
  --image nginx:latest \
  --replicas 3 \
  --port 80:8080

# List services
warren service list

# Scale service
warren service update web --replicas 5
```

## 📚 Documentation

- [Product Requirements](specs/prd.md) - Product vision and features
- [Technical Specification](specs/tech.md) - Architecture deep-dive
- [Development Plan](tasks/todo.md) - Milestone roadmap
- [Architecture Decisions](docs/adr/) - ADRs for key technical choices

## 🛠️ Development

### Prerequisites

- Go 1.22+
- containerd (for container runtime)
- WireGuard (Linux 5.6+ or userspace)

### Building

```bash
# Development build
make build

# Run CLI
./bin/warren --help

# Run tests
make test

# Run linters
make lint
```

### Project Structure

```
warren/
├── cmd/warren/          # CLI entry point
├── pkg/
│   ├── types/           # Core data models
│   ├── manager/         # Manager components (Raft, scheduler, API)
│   ├── worker/          # Worker agent
│   ├── api/             # gRPC/REST API
│   ├── network/         # WireGuard networking
│   ├── security/        # mTLS, secrets encryption
│   ├── storage/         # BoltDB state storage
│   └── deploy/          # Deployment strategies
├── test/                # Integration tests
├── specs/               # PRD, tech spec
├── docs/                # Documentation, ADRs
└── poc/                 # Proof-of-concepts
```

## 🗺️ Roadmap

### Milestone 0: Foundation ✅
- [x] POCs (Raft, containerd, WireGuard)
- [x] Architecture Decision Records

### Milestone 1: Core Orchestration 🔄
- [ ] Single-manager cluster
- [ ] Basic scheduler
- [ ] Worker agent
- [ ] CLI (cluster, service, node commands)

### Milestone 2: High Availability
- [ ] Multi-manager Raft cluster
- [ ] Leader election & failover
- [ ] Edge partition tolerance
- [ ] Rolling updates

### Milestone 3: Advanced Deployment
- [ ] Blue/green deployment
- [ ] Canary deployment
- [ ] Secrets management
- [ ] Volume orchestration

### Milestone 4: Production Ready
- [ ] Prometheus metrics
- [ ] Structured logging
- [ ] Multi-platform builds
- [ ] Binary optimization

### Milestone 5: Open Source
- [ ] Public release
- [ ] Community building
- [ ] Package distribution

## 🤝 Contributing

Warren is currently in **alpha** development. Contributions welcome once we reach Milestone 5 (Open Source).

For now, follow our progress:
- Development happens in the open on GitHub
- See [tasks/todo.md](tasks/todo.md) for current status
- Architecture decisions documented in [docs/adr/](docs/adr/)

## 📝 License

Apache 2.0 (coming with public release)

## 🙏 Acknowledgments

Warren is inspired by:
- **Docker Swarm** - Simplicity of UX
- **Kubernetes** - Feature richness
- **Nomad** - Single binary distribution

Built with:
- [hashicorp/raft](https://github.com/hashicorp/raft) - Consensus
- [containerd](https://containerd.io/) - Container runtime
- [WireGuard](https://www.wireguard.com/) - VPN/networking

---

**Status**: Alpha - Milestone 1 in progress
**Maintained by**: [Cuemby](https://cuemby.com) 🐰
