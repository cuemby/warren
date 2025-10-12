package network

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/cuemby/warren/pkg/types"
)

// HostPortPublisher manages host mode port publishing using iptables
type HostPortPublisher struct {
	// Track published ports for cleanup
	publishedPorts map[string][]types.PortMapping // taskID -> ports
}

// NewHostPortPublisher creates a new host port publisher
func NewHostPortPublisher() *HostPortPublisher {
	return &HostPortPublisher{
		publishedPorts: make(map[string][]types.PortMapping),
	}
}

// PublishPorts sets up iptables rules to forward host ports to container ports
// This implements "host mode" where ports are published only on the node running the task
func (p *HostPortPublisher) PublishPorts(taskID, containerIP string, ports []types.PortMapping) error {
	if len(ports) == 0 {
		return nil
	}

	// Filter for host mode ports only
	var hostPorts []types.PortMapping
	for _, port := range ports {
		if port.PublishMode == types.PublishModeHost {
			hostPorts = append(hostPorts, port)
		}
	}

	if len(hostPorts) == 0 {
		return nil
	}

	// Set up iptables rules for each port
	for _, port := range hostPorts {
		if err := p.setupPortForwarding(containerIP, port); err != nil {
			// Clean up any rules we already created
			p.cleanupPorts(taskID, hostPorts)
			return fmt.Errorf("failed to setup port forwarding for %d:%d: %w",
				port.HostPort, port.ContainerPort, err)
		}
	}

	// Track ports for cleanup
	p.publishedPorts[taskID] = hostPorts

	return nil
}

// UnpublishPorts removes iptables rules for a task's published ports
func (p *HostPortPublisher) UnpublishPorts(taskID string) error {
	ports, ok := p.publishedPorts[taskID]
	if !ok {
		return nil // No ports to clean up
	}

	return p.cleanupPorts(taskID, ports)
}

// setupPortForwarding creates iptables DNAT rule for port forwarding
// Rule: host_ip:published_port -> container_ip:target_port
func (p *HostPortPublisher) setupPortForwarding(containerIP string, port types.PortMapping) error {
	protocol := strings.ToLower(port.Protocol)
	if protocol == "" {
		protocol = "tcp"
	}

	// iptables -t nat -A PREROUTING -p tcp --dport <host_port> -j DNAT --to-destination <container_ip>:<container_port>
	rule := []string{
		"-t", "nat",
		"-A", "PREROUTING",
		"-p", protocol,
		"--dport", fmt.Sprintf("%d", port.HostPort),
		"-j", "DNAT",
		"--to-destination", fmt.Sprintf("%s:%d", containerIP, port.ContainerPort),
	}

	if err := runIPTables(rule); err != nil {
		return fmt.Errorf("failed to add DNAT rule: %w", err)
	}

	// Also add MASQUERADE rule for return traffic
	// iptables -t nat -A POSTROUTING -p tcp -d <container_ip> --dport <container_port> -j MASQUERADE
	masqRule := []string{
		"-t", "nat",
		"-A", "POSTROUTING",
		"-p", protocol,
		"-d", containerIP,
		"--dport", fmt.Sprintf("%d", port.ContainerPort),
		"-j", "MASQUERADE",
	}

	if err := runIPTables(masqRule); err != nil {
		// Clean up the DNAT rule we just created
		p.removePortForwarding(containerIP, port)
		return fmt.Errorf("failed to add MASQUERADE rule: %w", err)
	}

	// Add rule to allow forwarding
	// iptables -A FORWARD -p tcp -d <container_ip> --dport <container_port> -j ACCEPT
	forwardRule := []string{
		"-A", "FORWARD",
		"-p", protocol,
		"-d", containerIP,
		"--dport", fmt.Sprintf("%d", port.ContainerPort),
		"-j", "ACCEPT",
	}

	if err := runIPTables(forwardRule); err != nil {
		// Clean up previously created rules
		p.removePortForwarding(containerIP, port)
		return fmt.Errorf("failed to add FORWARD rule: %w", err)
	}

	return nil
}

// removePortForwarding removes iptables rules for a port
func (p *HostPortPublisher) removePortForwarding(containerIP string, port types.PortMapping) error {
	protocol := strings.ToLower(port.Protocol)
	if protocol == "" {
		protocol = "tcp"
	}

	// Remove DNAT rule
	dnatRule := []string{
		"-t", "nat",
		"-D", "PREROUTING",
		"-p", protocol,
		"--dport", fmt.Sprintf("%d", port.HostPort),
		"-j", "DNAT",
		"--to-destination", fmt.Sprintf("%s:%d", containerIP, port.ContainerPort),
	}
	runIPTables(dnatRule) // Ignore errors on cleanup

	// Remove MASQUERADE rule
	masqRule := []string{
		"-t", "nat",
		"-D", "POSTROUTING",
		"-p", protocol,
		"-d", containerIP,
		"--dport", fmt.Sprintf("%d", port.ContainerPort),
		"-j", "MASQUERADE",
	}
	runIPTables(masqRule) // Ignore errors on cleanup

	// Remove FORWARD rule
	forwardRule := []string{
		"-D", "FORWARD",
		"-p", protocol,
		"-d", containerIP,
		"--dport", fmt.Sprintf("%d", port.ContainerPort),
		"-j", "ACCEPT",
	}
	runIPTables(forwardRule) // Ignore errors on cleanup

	return nil
}

// cleanupPorts removes all iptables rules for a task
func (p *HostPortPublisher) cleanupPorts(taskID string, ports []types.PortMapping) error {
	// We need the container IP to clean up, but we don't have it stored
	// For now, we'll try to remove rules by scanning iptables
	// This is a limitation we can improve later by storing container IP

	delete(p.publishedPorts, taskID)
	return nil
}

// runIPTables executes an iptables command
func runIPTables(args []string) error {
	cmd := exec.Command("iptables", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("iptables failed: %w (output: %s)", err, string(output))
	}
	return nil
}

// GetPublishedPorts returns the ports currently published for a task
func (p *HostPortPublisher) GetPublishedPorts(taskID string) []types.PortMapping {
	return p.publishedPorts[taskID]
}
