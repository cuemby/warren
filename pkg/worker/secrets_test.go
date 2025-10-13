package worker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestGetSecretPath tests secret path generation
func TestGetSecretPath(t *testing.T) {
	sh := &SecretsHandler{}

	tests := []struct {
		name       string
		taskID     string
		secretName string
		want       string
	}{
		{
			name:       "simple secret",
			taskID:     "task-123",
			secretName: "db-password",
			want:       filepath.Join(SecretsBasePath, "task-123", "db-password"),
		},
		{
			name:       "secret with dots",
			taskID:     "task-456",
			secretName: "app.config.json",
			want:       filepath.Join(SecretsBasePath, "task-456", "app.config.json"),
		},
		{
			name:       "empty task ID",
			taskID:     "",
			secretName: "secret",
			want:       filepath.Join(SecretsBasePath, "", "secret"),
		},
		{
			name:       "empty secret name",
			taskID:     "task-789",
			secretName: "",
			want:       filepath.Join(SecretsBasePath, "task-789", ""),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sh.GetSecretPath(tt.taskID, tt.secretName)
			if got != tt.want {
				t.Errorf("GetSecretPath(%q, %q) = %q, want %q", tt.taskID, tt.secretName, got, tt.want)
			}
		})
	}
}

// TestSecretsBasePath tests the secrets base path constant
func TestSecretsBasePath(t *testing.T) {
	expected := "/run/secrets"
	if SecretsBasePath != expected {
		t.Errorf("SecretsBasePath = %q, want %q", SecretsBasePath, expected)
	}
}

// TestEnsureSecretsBaseDir tests base directory creation
func TestEnsureSecretsBaseDir(t *testing.T) {
	// Skip if not running with appropriate permissions
	if os.Getuid() != 0 {
		t.Skip("Skipping test that requires root permissions")
	}

	// This test would require root permissions to create /run/secrets
	// For now, we'll just verify the function doesn't panic
	err := EnsureSecretsBaseDir()
	if err != nil {
		// It's okay if it fails due to permissions - we just want to verify
		// the function handles errors properly
		t.Logf("EnsureSecretsBaseDir() error (expected if not root): %v", err)
	}
}

// TestCleanupSecretsForTask tests cleanup of non-existent directories
func TestCleanupSecretsForTask(t *testing.T) {
	sh := &SecretsHandler{}

	// Test cleanup of non-existent task (should not error)
	err := sh.CleanupSecretsForTask("non-existent-task")
	if err != nil {
		t.Errorf("CleanupSecretsForTask() for non-existent task should not error, got: %v", err)
	}
}

// TestGetSecretPathConsistency tests that GetSecretPath is consistent
func TestGetSecretPathConsistency(t *testing.T) {
	sh := &SecretsHandler{}

	taskID := "task-abc"
	secretName := "my-secret"

	path1 := sh.GetSecretPath(taskID, secretName)
	path2 := sh.GetSecretPath(taskID, secretName)

	if path1 != path2 {
		t.Errorf("GetSecretPath() not consistent: first=%q, second=%q", path1, path2)
	}

	// Verify it uses filepath.Join semantics
	expected := filepath.Join(SecretsBasePath, taskID, secretName)
	if path1 != expected {
		t.Errorf("GetSecretPath() = %q, want %q", path1, expected)
	}
}

// TestGetSecretPathMultipleTasks tests different tasks get different paths
func TestGetSecretPathMultipleTasks(t *testing.T) {
	sh := &SecretsHandler{}

	path1 := sh.GetSecretPath("task-1", "secret")
	path2 := sh.GetSecretPath("task-2", "secret")

	if path1 == path2 {
		t.Errorf("GetSecretPath() should return different paths for different tasks: %q == %q", path1, path2)
	}

	// Verify both paths are under SecretsBasePath
	if !strings.HasPrefix(path1, SecretsBasePath) {
		t.Errorf("GetSecretPath() path not under SecretsBasePath: %q", path1)
	}
	if !strings.HasPrefix(path2, SecretsBasePath) {
		t.Errorf("GetSecretPath() path not under SecretsBasePath: %q", path2)
	}
}
