---
name: Bug Report
about: Create a report to help us improve
title: '[BUG] '
labels: bug
assignees: ''
---

## Bug Description

<!-- A clear and concise description of what the bug is -->

## Steps to Reproduce

1. Go to '...'
2. Click on '....'
3. Scroll down to '....'
4. See error

## Expected Behavior

<!-- A clear and concise description of what you expected to happen -->

## Actual Behavior

<!-- A clear and concise description of what actually happened -->

## Environment

**Warren Version:**
```bash
warren --version
```

**Operating System:**
```bash
uname -a
```

**Go Version (if building from source):**
```bash
go version
```

**Cluster Configuration:**
- Number of managers:
- Number of workers:
- Deployment type: (single-node / multi-node)

## Logs

<!-- Provide relevant logs -->

**Manager logs:**
```
sudo journalctl -u warren-manager -n 100
```

**Worker logs:**
```
sudo journalctl -u warren-worker -n 100
```

## Additional Context

<!-- Add any other context about the problem here -->

**Cluster state:**
```bash
warren cluster info --manager <manager-ip>:8080
warren node list --manager <manager-ip>:8080
warren service list --manager <manager-ip>:8080
```

**Screenshots:**
<!-- If applicable, add screenshots to help explain your problem -->

## Possible Solution

<!-- Optional: Suggest a fix or reason for the bug -->
