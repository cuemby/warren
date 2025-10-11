package worker

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	// DefaultDNSDir is the directory where DNS config files are stored
	DefaultDNSDir = "/var/lib/warren/dns"

	// DefaultResolvConf is the default resolv.conf template filename
	DefaultResolvConf = "resolv.conf"
)

// DNSHandler manages DNS configuration for containers
type DNSHandler struct {
	worker      *Worker
	managerAddr string // Manager IP address for DNS queries
	dnsDir      string // Directory for DNS config files
}

// NewDNSHandler creates a new DNS handler
func NewDNSHandler(w *Worker, managerAddr string) (*DNSHandler, error) {
	dnsDir := DefaultDNSDir

	// Ensure DNS directory exists
	if err := os.MkdirAll(dnsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create DNS directory: %w", err)
	}

	return &DNSHandler{
		worker:      w,
		managerAddr: managerAddr,
		dnsDir:      dnsDir,
	}, nil
}

// GenerateResolvConf generates a resolv.conf file for containers
// This configures containers to use Warren DNS server on the manager
//
// Format:
//   nameserver <manager-ip>    # Warren DNS server
//   nameserver 8.8.8.8          # Google DNS fallback
//   nameserver 1.1.1.1          # Cloudflare DNS fallback
//   search warren               # Allow "nginx" instead of "nginx.warren"
//   options ndots:0             # Try search domains immediately
func (h *DNSHandler) GenerateResolvConf() (string, error) {
	resolvConfPath := filepath.Join(h.dnsDir, DefaultResolvConf)

	// Build resolv.conf content
	content := fmt.Sprintf(`# Warren DNS Configuration
# Generated automatically - do not edit manually

# Primary: Warren DNS server (service discovery)
nameserver %s

# Fallback: External DNS servers
nameserver 8.8.8.8
nameserver 1.1.1.1

# Search domain for Warren services
search warren

# Options
options ndots:0
`, h.managerAddr)

	// Write resolv.conf file
	if err := os.WriteFile(resolvConfPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write resolv.conf: %w", err)
	}

	return resolvConfPath, nil
}

// GetResolvConfPath returns the path to the generated resolv.conf file
// If the file doesn't exist, it generates it first
func (h *DNSHandler) GetResolvConfPath() (string, error) {
	resolvConfPath := filepath.Join(h.dnsDir, DefaultResolvConf)

	// Check if file exists
	if _, err := os.Stat(resolvConfPath); os.IsNotExist(err) {
		// Generate it
		return h.GenerateResolvConf()
	}

	return resolvConfPath, nil
}

// Cleanup removes the DNS configuration directory
func (h *DNSHandler) Cleanup() error {
	return os.RemoveAll(h.dnsDir)
}

// ExtractManagerIP extracts the IP address from manager address
// Examples:
//   "192.168.1.100:8080" -> "192.168.1.100"
//   "localhost:8080" -> "127.0.0.1"
//   "manager-1:8080" -> "manager-1" (hostname, DNS will resolve)
func ExtractManagerIP(managerAddr string) string {
	// Find the colon separating host from port
	for i := len(managerAddr) - 1; i >= 0; i-- {
		if managerAddr[i] == ':' {
			host := managerAddr[:i]
			// Handle localhost special case
			if host == "localhost" {
				return "127.0.0.1"
			}
			return host
		}
	}

	// No port found, return as-is
	if managerAddr == "localhost" {
		return "127.0.0.1"
	}
	return managerAddr
}
