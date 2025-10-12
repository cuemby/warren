package dns

import (
	"net"
	"testing"
	"time"

	"github.com/cuemby/warren/pkg/types"
)

// TestResolverStripDomain tests domain suffix removal
func TestResolverStripDomain(t *testing.T) {
	r := NewResolver(nil, "warren", nil)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "with domain suffix",
			input: "nginx.warren",
			want:  "nginx",
		},
		{
			name:  "without domain suffix",
			input: "nginx",
			want:  "nginx",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "multiple dots",
			input: "web.api.warren",
			want:  "web.api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.stripDomain(tt.input)
			if got != tt.want {
				t.Errorf("stripDomain(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestResolverMakeFQDN tests FQDN generation
func TestResolverMakeFQDN(t *testing.T) {
	r := NewResolver(nil, "warren", nil)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "without trailing dot",
			input: "nginx",
			want:  "nginx.",
		},
		{
			name:  "with trailing dot",
			input: "nginx.",
			want:  "nginx.",
		},
		{
			name:  "fqdn with domain",
			input: "nginx.warren",
			want:  "nginx.warren.",
		},
		{
			name:  "already fqdn",
			input: "nginx.warren.",
			want:  "nginx.warren.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := r.makeFQDN(tt.input)
			if got != tt.want {
				t.Errorf("makeFQDN(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestGetTaskIP tests task IP generation
func TestGetTaskIP(t *testing.T) {
	r := NewResolver(nil, "warren", nil)

	tests := []struct {
		name   string
		taskID string
	}{
		{
			name:   "task 1",
			taskID: "task-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		},
		{
			name:   "task 2",
			taskID: "task-bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		},
		{
			name:   "task 3",
			taskID: "task-cccccccc-cccc-cccc-cccc-cccccccccccc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task := &types.Task{ID: tt.taskID}
			ip := r.getTaskIP(task)

			if ip == nil {
				t.Fatal("getTaskIP() returned nil")
			}

			// Verify IP is in 10.0.0.0/8 range (private IP space)
			ipv4 := ip.To4()
			if ipv4 == nil || ipv4[0] != 10 {
				t.Errorf("getTaskIP() IP not in 10.0.0.0/8 range: %v", ip)
			}

			// Verify consistency - same task ID should give same IP
			ip2 := r.getTaskIP(task)
			if !ip.Equal(ip2) {
				t.Errorf("getTaskIP() not consistent: first=%v, second=%v", ip, ip2)
			}
		})
	}
}

// TestHashString tests string hashing
func TestHashString(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple string",
			input: "hello",
		},
		{
			name:  "task ID",
			input: "task-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		},
		{
			name:  "empty string",
			input: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash1 := hashString(tt.input)
			hash2 := hashString(tt.input)

			// Verify consistency
			if hash1 != hash2 {
				t.Errorf("hashString() not consistent: first=%v, second=%v", hash1, hash2)
			}

			// Verify non-zero for non-empty strings
			if tt.input != "" && hash1 == 0 {
				t.Errorf("hashString(%q) = 0, expected non-zero", tt.input)
			}

			// Verify zero for empty string
			if tt.input == "" && hash1 != 0 {
				t.Errorf("hashString(%q) = %v, expected 0", tt.input, hash1)
			}
		})
	}
}

// TestShuffleIPs tests IP shuffling
func TestShuffleIPs(t *testing.T) {
	r := NewResolver(nil, "warren", nil)

	ips := []net.IP{
		net.IPv4(10, 0, 0, 1),
		net.IPv4(10, 0, 0, 2),
		net.IPv4(10, 0, 0, 3),
		net.IPv4(10, 0, 0, 4),
		net.IPv4(10, 0, 0, 5),
	}

	// Make a copy for comparison
	original := make([]net.IP, len(ips))
	copy(original, ips)

	// Shuffle multiple times to verify it changes
	shuffled := false
	for i := 0; i < 10; i++ {
		r.shuffleIPs(ips)

		// Check if order changed
		orderChanged := false
		for j := range ips {
			if !ips[j].Equal(original[j]) {
				orderChanged = true
				break
			}
		}

		if orderChanged {
			shuffled = true
			break
		}
	}

	if !shuffled {
		t.Error("shuffleIPs() did not change order after 10 attempts")
	}

	// Verify all IPs still present (just reordered)
	for _, origIP := range original {
		found := false
		for _, ip := range ips {
			if ip.Equal(origIP) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("shuffleIPs() lost IP %v", origIP)
		}
	}
}

// TestSortTasksByCreationTime tests task sorting
func TestSortTasksByCreationTime(t *testing.T) {
	now := time.Now()

	tasks := []*types.Task{
		{ID: "task3", CreatedAt: now.Add(3 * time.Second)},
		{ID: "task1", CreatedAt: now.Add(1 * time.Second)},
		{ID: "task4", CreatedAt: now.Add(4 * time.Second)},
		{ID: "task2", CreatedAt: now.Add(2 * time.Second)},
	}

	sortTasksByCreationTime(tasks)

	// Verify sorted order (oldest first)
	expectedOrder := []string{"task1", "task2", "task3", "task4"}
	for i, task := range tasks {
		if task.ID != expectedOrder[i] {
			t.Errorf("sortTasksByCreationTime() position %d = %s, want %s", i, task.ID, expectedOrder[i])
		}
	}

	// Verify times are in ascending order
	for i := 0; i < len(tasks)-1; i++ {
		if tasks[i].CreatedAt.After(tasks[i+1].CreatedAt) {
			t.Errorf("sortTasksByCreationTime() not sorted: tasks[%d].CreatedAt > tasks[%d].CreatedAt", i, i+1)
		}
	}
}

// TestSortTasksByCreationTimeEdgeCases tests edge cases
func TestSortTasksByCreationTimeEdgeCases(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		tasks := []*types.Task{}
		sortTasksByCreationTime(tasks) // Should not panic
		if len(tasks) != 0 {
			t.Error("sortTasksByCreationTime() modified empty slice")
		}
	})

	t.Run("single task", func(t *testing.T) {
		now := time.Now()
		tasks := []*types.Task{
			{ID: "task1", CreatedAt: now},
		}
		sortTasksByCreationTime(tasks)
		if tasks[0].ID != "task1" {
			t.Error("sortTasksByCreationTime() modified single task")
		}
	})

	t.Run("already sorted", func(t *testing.T) {
		now := time.Now()
		tasks := []*types.Task{
			{ID: "task1", CreatedAt: now.Add(1 * time.Second)},
			{ID: "task2", CreatedAt: now.Add(2 * time.Second)},
			{ID: "task3", CreatedAt: now.Add(3 * time.Second)},
		}
		sortTasksByCreationTime(tasks)

		expectedOrder := []string{"task1", "task2", "task3"}
		for i, task := range tasks {
			if task.ID != expectedOrder[i] {
				t.Errorf("sortTasksByCreationTime() changed order: position %d = %s, want %s", i, task.ID, expectedOrder[i])
			}
		}
	})

	t.Run("reverse sorted", func(t *testing.T) {
		now := time.Now()
		tasks := []*types.Task{
			{ID: "task3", CreatedAt: now.Add(3 * time.Second)},
			{ID: "task2", CreatedAt: now.Add(2 * time.Second)},
			{ID: "task1", CreatedAt: now.Add(1 * time.Second)},
		}
		sortTasksByCreationTime(tasks)

		expectedOrder := []string{"task1", "task2", "task3"}
		for i, task := range tasks {
			if task.ID != expectedOrder[i] {
				t.Errorf("sortTasksByCreationTime() position %d = %s, want %s", i, task.ID, expectedOrder[i])
			}
		}
	})

	t.Run("duplicate timestamps", func(t *testing.T) {
		now := time.Now()
		tasks := []*types.Task{
			{ID: "task1", CreatedAt: now},
			{ID: "task2", CreatedAt: now},
			{ID: "task3", CreatedAt: now},
		}
		sortTasksByCreationTime(tasks)

		// Should not panic, order may vary but all tasks present
		if len(tasks) != 3 {
			t.Error("sortTasksByCreationTime() lost tasks with duplicate timestamps")
		}
	})
}

// TestNewResolver tests resolver creation
func TestNewResolver(t *testing.T) {
	domain := "warren"
	upstream := []string{"8.8.8.8:53", "1.1.1.1:53"}

	r := NewResolver(nil, domain, upstream)

	if r == nil {
		t.Fatal("NewResolver() returned nil")
	}

	if r.domain != domain {
		t.Errorf("NewResolver() domain = %q, want %q", r.domain, domain)
	}

	if len(r.upstream) != len(upstream) {
		t.Errorf("NewResolver() upstream count = %d, want %d", len(r.upstream), len(upstream))
	}

	if r.rnd == nil {
		t.Error("NewResolver() rnd is nil")
	}
}
