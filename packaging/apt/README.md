# APT Repository for Warren

This directory contains documentation for setting up Warren APT packages (Debian/Ubuntu).

## For Users

### Install Warren via APT

```bash
# Add Warren APT repository
curl -sL https://packagecloud.io/cuemby/warren/gpgkey | sudo apt-key add -

# Add repository to sources
echo "deb https://packagecloud.io/cuemby/warren/ubuntu/ focal main" | \
  sudo tee /etc/apt/sources.list.d/warren.list

# Update package index
sudo apt update

# Install Warren
sudo apt install warren

# Verify installation
warren --version
```

### Update Warren

```bash
sudo apt update
sudo apt upgrade warren
```

### Uninstall Warren

```bash
sudo apt remove warren

# Or remove with config files
sudo apt purge warren
```

## For Maintainers

### Prerequisites

1. **packagecloud.io account** (or self-hosted APT repository)
2. **GPG key** for package signing
3. **Package building tools**

### Setup Package Building Environment

```bash
# Install packaging tools
sudo apt install build-essential debhelper devscripts

# Install packagecloud CLI (for packagecloud.io)
gem install package_cloud
```

### Create .deb Package

#### 1. Create Debian Package Structure

```bash
cd warren
mkdir -p debian

# Create required files
touch debian/control
touch debian/changelog
touch debian/rules
touch debian/install
touch debian/postinst
touch debian/prerm
touch debian/warren.service
```

#### 2. debian/control

```
Source: warren
Section: admin
Priority: optional
Maintainer: Cuemby Team <opensource@cuemby.com>
Build-Depends: debhelper (>= 10), golang-go (>= 1.22)
Standards-Version: 4.5.0
Homepage: https://github.com/cuemby/warren

Package: warren
Architecture: amd64 arm64
Depends: ${misc:Depends}, containerd (>= 1.6)
Description: Simple-yet-powerful container orchestrator
 Warren is a container orchestration platform designed for edge computing.
 It combines the simplicity of Docker Swarm with the feature richness
 of Kubernetes, delivered as a single binary with zero external dependencies.
 .
 Features:
  - Multi-manager HA with Raft consensus
  - Built-in secrets management (AES-256-GCM)
  - WireGuard encrypted overlay networking
  - Service discovery and load balancing
  - Volume orchestration
  - Prometheus metrics
  - Edge-optimized for partition tolerance
```

#### 3. debian/changelog

```
warren (1.0.0-1) focal; urgency=medium

  * Initial release
  * Multi-manager HA cluster
  * Service orchestration
  * Secrets and volumes support
  * Built-in metrics and logging

 -- Cuemby Team <opensource@cuemby.com>  Thu, 10 Oct 2025 10:00:00 +0000
```

#### 4. debian/rules

```makefile
#!/usr/bin/make -f

%:
	dh $@

override_dh_auto_build:
	go build -ldflags="-s -w" -o warren ./cmd/warren

override_dh_auto_install:
	install -D -m 0755 warren debian/warren/usr/bin/warren
	install -D -m 0644 debian/warren.service debian/warren/lib/systemd/system/warren.service

override_dh_auto_test:
	go test ./...
```

#### 5. debian/install

```
warren /usr/bin/
```

#### 6. debian/postinst

```bash
#!/bin/sh
set -e

case "$1" in
    configure)
        # Create warren user
        if ! getent passwd warren >/dev/null; then
            useradd -r -s /bin/false -d /var/lib/warren warren
        fi

        # Create data directory
        mkdir -p /var/lib/warren/data
        chown -R warren:warren /var/lib/warren

        # Reload systemd
        systemctl daemon-reload
        ;;
esac

#DEBHELPER#

exit 0
```

#### 7. debian/prerm

```bash
#!/bin/sh
set -e

case "$1" in
    remove)
        # Stop services
        systemctl stop warren-manager || true
        systemctl stop warren-worker || true
        ;;
esac

#DEBHELPER#

exit 0
```

#### 8. debian/warren.service

```ini
[Unit]
Description=Warren Container Orchestrator
After=network.target containerd.service
Wants=containerd.service

[Service]
Type=simple
User=root
ExecStart=/usr/bin/warren cluster init
Restart=on-failure
RestartSec=5s

[Install]
WantedBy=multi-user.target
```

### Build Package

```bash
# Build for amd64
dpkg-buildpackage -us -uc -b

# Build for arm64 (requires cross-compilation setup)
dpkg-buildpackage -aarm64 -us -uc -b

# Result: warren_1.0.0-1_amd64.deb
```

### Sign Package

```bash
# Generate GPG key (if not exists)
gpg --gen-key

# Sign package
dpkg-sig --sign builder warren_1.0.0-1_amd64.deb
```

### Publish to packagecloud.io

```bash
# Set API token
export PACKAGECLOUD_TOKEN=your_token_here

# Push package
package_cloud push cuemby/warren/ubuntu/focal warren_1.0.0-1_amd64.deb

# Push for multiple distributions
package_cloud push cuemby/warren/ubuntu/focal warren_1.0.0-1_amd64.deb
package_cloud push cuemby/warren/ubuntu/jammy warren_1.0.0-1_amd64.deb
package_cloud push cuemby/warren/debian/bullseye warren_1.0.0-1_amd64.deb
```

### Self-Hosted APT Repository

If not using packagecloud.io:

#### 1. Set up repository structure

```bash
mkdir -p /var/www/apt/{pool,dists/focal/main/binary-amd64}
```

#### 2. Copy packages

```bash
cp warren_1.0.0-1_amd64.deb /var/www/apt/pool/
```

#### 3. Generate Packages index

```bash
cd /var/www/apt
apt-ftparchive packages pool > dists/focal/main/binary-amd64/Packages
gzip -k dists/focal/main/binary-amd64/Packages
```

#### 4. Generate Release file

```bash
cd /var/www/apt/dists/focal
apt-ftparchive release . > Release
gpg --clearsign -o InRelease Release
gpg -abs -o Release.gpg Release
```

#### 5. Serve via HTTP

```nginx
server {
    listen 80;
    server_name apt.warren.io;
    root /var/www/apt;

    location / {
        autoindex on;
    }
}
```

#### 6. Users add repository

```bash
# Add GPG key
curl -sL https://apt.warren.io/gpg.key | sudo apt-key add -

# Add repository
echo "deb https://apt.warren.io focal main" | \
  sudo tee /etc/apt/sources.list.d/warren.list

sudo apt update
sudo apt install warren
```

## Supported Distributions

### Ubuntu
- 20.04 (Focal Fossa)
- 22.04 (Jammy Jellyfish)
- 24.04 (Noble Numbat)

### Debian
- 11 (Bullseye)
- 12 (Bookworm)

## Automation

### GitHub Actions Workflow

Create `.github/workflows/deb-package.yml`:

```yaml
name: Build DEB Package

on:
  release:
    types: [published]

jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        arch: [amd64, arm64]
    steps:
      - uses: actions/checkout@v4

      - name: Install dependencies
        run: |
          sudo apt update
          sudo apt install -y debhelper devscripts

      - name: Build package
        run: dpkg-buildpackage -a${{ matrix.arch }} -us -uc -b

      - name: Upload to packagecloud
        env:
          PACKAGECLOUD_TOKEN: ${{ secrets.PACKAGECLOUD_TOKEN }}
        run: |
          gem install package_cloud
          package_cloud push cuemby/warren/ubuntu/focal ../*.deb
```

## Testing Checklist

Before publishing package:

- [ ] Package builds successfully
- [ ] Package installs (`sudo apt install ./warren_1.0.0-1_amd64.deb`)
- [ ] Binary works (`warren --version`)
- [ ] Service starts (`sudo systemctl start warren-manager`)
- [ ] Package upgrades (`sudo apt upgrade warren`)
- [ ] Package removes cleanly (`sudo apt remove warren`)
- [ ] Package purges cleanly (`sudo apt purge warren`)
- [ ] Tested on all supported distributions

## Troubleshooting

**Issue: dpkg-buildpackage fails**
```
dpkg-buildpackage: error: cannot read debian/control
```

Solution: Ensure all debian/* files exist and have correct format

**Issue: Unmet dependencies**
```
warren : Depends: containerd (>= 1.6) but it is not installable
```

Solution: Add universe repository or adjust dependencies

**Issue: GPG signing fails**
```
gpg: signing failed: No secret key
```

Solution: Generate GPG key first (`gpg --gen-key`)

## Resources

- [Debian New Maintainers' Guide](https://www.debian.org/doc/manuals/maint-guide/)
- [Debian Policy Manual](https://www.debian.org/doc/debian-policy/)
- [packagecloud.io Documentation](https://packagecloud.io/docs)
- [Ubuntu Packaging Guide](https://packaging.ubuntu.com/html/)
