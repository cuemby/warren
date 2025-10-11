package dns

import (
	"testing"
)

// TestParseInstanceName tests instance name parsing
func TestParseInstanceName(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantService   string
		wantInstance  int
		wantErr       bool
	}{
		{
			name:         "simple service instance 1",
			input:        "nginx-1",
			wantService:  "nginx",
			wantInstance: 1,
			wantErr:      false,
		},
		{
			name:         "simple service instance 2",
			input:        "nginx-2",
			wantService:  "nginx",
			wantInstance: 2,
			wantErr:      false,
		},
		{
			name:         "hyphenated service name",
			input:        "web-api-3",
			wantService:  "web-api",
			wantInstance: 3,
			wantErr:      false,
		},
		{
			name:         "service name only (no instance)",
			input:        "nginx",
			wantService:  "",
			wantInstance: 0,
			wantErr:      true,
		},
		{
			name:         "invalid instance number",
			input:        "nginx-abc",
			wantService:  "",
			wantInstance: 0,
			wantErr:      true,
		},
		{
			name:         "instance number zero",
			input:        "nginx-0",
			wantService:  "",
			wantInstance: 0,
			wantErr:      true,
		},
		{
			name:         "double hyphen before number",
			input:        "nginx--1",
			wantService:  "nginx-",
			wantInstance: 1,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotService, gotInstance, err := parseInstanceName(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("parseInstanceName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if gotService != tt.wantService {
					t.Errorf("parseInstanceName() service = %v, want %v", gotService, tt.wantService)
				}
				if gotInstance != tt.wantInstance {
					t.Errorf("parseInstanceName() instance = %v, want %v", gotInstance, tt.wantInstance)
				}
			}
		})
	}
}

// TestMakeInstanceName tests instance name generation
func TestMakeInstanceName(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		instanceNum int
		want        string
	}{
		{
			name:        "simple service",
			serviceName: "nginx",
			instanceNum: 1,
			want:        "nginx-1",
		},
		{
			name:        "hyphenated service",
			serviceName: "web-api",
			instanceNum: 3,
			want:        "web-api-3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := makeInstanceName(tt.serviceName, tt.instanceNum)
			if got != tt.want {
				t.Errorf("makeInstanceName() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestResolverStripDomain tests domain suffix removal
func TestResolverStripDomain(t *testing.T) {
	// TODO: Implement when memory store is available
	t.Skip("Skipping until memory store is implemented")
}

// TestResolverMakeFQDN tests FQDN generation
func TestResolverMakeFQDN(t *testing.T) {
	// TODO: Implement when memory store is available
	t.Skip("Skipping until memory store is implemented")
}

// TestResolverServiceResolution tests service name resolution
func TestResolverServiceResolution(t *testing.T) {
	// TODO: Implement when memory store is available
	t.Skip("Skipping until memory store is implemented")
}

// TestResolverInstanceResolution tests instance-specific resolution
func TestResolverInstanceResolution(t *testing.T) {
	// TODO: Implement when memory store is available
	t.Skip("Skipping until memory store is implemented")
}

// Helper function to generate consistent task IDs for tests
func generateTaskID(num int) string {
	switch num {
	case 1:
		return "task-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	case 2:
		return "task-bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	case 3:
		return "task-cccccccc-cccc-cccc-cccc-cccccccccccc"
	default:
		return "task-xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
	}
}
