package worker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExtractManagerIP tests IP extraction from manager addresses
func TestExtractManagerIP(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "IP with port",
			input: "192.168.1.100:8080",
			want:  "192.168.1.100",
		},
		{
			name:  "localhost with port",
			input: "localhost:8080",
			want:  "127.0.0.1",
		},
		{
			name:  "hostname with port",
			input: "manager-1:8080",
			want:  "manager-1",
		},
		{
			name:  "IP without port",
			input: "192.168.1.100",
			want:  "192.168.1.100",
		},
		{
			name:  "localhost without port",
			input: "localhost",
			want:  "127.0.0.1",
		},
		{
			name:  "hostname without port",
			input: "manager-1",
			want:  "manager-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractManagerIP(tt.input)
			if got != tt.want {
				t.Errorf("ExtractManagerIP(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestGenerateResolvConf tests resolv.conf generation
func TestGenerateResolvConf(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Create mock worker
	w := &Worker{
		nodeID:      "test-worker",
		managerAddr: "192.168.1.100:8080",
	}

	// Create DNS handler with custom directory
	handler := &DNSHandler{
		worker:      w,
		managerAddr: "192.168.1.100",
		dnsDir:      tmpDir,
	}

	// Generate resolv.conf
	path, err := handler.GenerateResolvConf()
	if err != nil {
		t.Fatalf("GenerateResolvConf() error = %v", err)
	}

	// Verify path is correct
	expectedPath := filepath.Join(tmpDir, DefaultResolvConf)
	if path != expectedPath {
		t.Errorf("GenerateResolvConf() path = %q, want %q", path, expectedPath)
	}

	// Read generated file
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read resolv.conf: %v", err)
	}

	contentStr := string(content)

	// Verify content contains required elements
	requiredElements := []string{
		"nameserver 192.168.1.100", // Manager DNS
		"nameserver 8.8.8.8",       // Google DNS
		"nameserver 1.1.1.1",       // Cloudflare DNS
		"search warren",            // Search domain
		"options ndots:0",          // Options
	}

	for _, element := range requiredElements {
		if !strings.Contains(contentStr, element) {
			t.Errorf("resolv.conf missing required element: %q\nContent:\n%s", element, contentStr)
		}
	}

	// Verify file permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Failed to stat resolv.conf: %v", err)
	}

	if info.Mode().Perm() != 0644 {
		t.Errorf("resolv.conf permissions = %o, want 0644", info.Mode().Perm())
	}
}

// TestGetResolvConfPath tests resolv.conf path retrieval
func TestGetResolvConfPath(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Create mock worker
	w := &Worker{
		nodeID:      "test-worker",
		managerAddr: "192.168.1.100:8080",
	}

	// Create DNS handler with custom directory
	handler := &DNSHandler{
		worker:      w,
		managerAddr: "192.168.1.100",
		dnsDir:      tmpDir,
	}

	// First call should generate the file
	path1, err := handler.GetResolvConfPath()
	if err != nil {
		t.Fatalf("GetResolvConfPath() error = %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path1); os.IsNotExist(err) {
		t.Errorf("resolv.conf was not created at %q", path1)
	}

	// Second call should return existing file
	path2, err := handler.GetResolvConfPath()
	if err != nil {
		t.Fatalf("GetResolvConfPath() error = %v", err)
	}

	if path1 != path2 {
		t.Errorf("GetResolvConfPath() returned different paths: %q vs %q", path1, path2)
	}
}

// TestDNSHandlerCleanup tests cleanup functionality
func TestDNSHandlerCleanup(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()

	// Create mock worker
	w := &Worker{
		nodeID:      "test-worker",
		managerAddr: "192.168.1.100:8080",
	}

	// Create DNS handler with custom directory
	handler := &DNSHandler{
		worker:      w,
		managerAddr: "192.168.1.100",
		dnsDir:      tmpDir,
	}

	// Generate resolv.conf
	_, err := handler.GenerateResolvConf()
	if err != nil {
		t.Fatalf("GenerateResolvConf() error = %v", err)
	}

	// Cleanup
	if err := handler.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	// Verify directory is removed
	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		t.Errorf("DNS directory still exists after cleanup")
	}
}
