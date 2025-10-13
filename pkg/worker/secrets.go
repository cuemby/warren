package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cuemby/warren/api/proto"
	"github.com/cuemby/warren/pkg/security"
	"github.com/cuemby/warren/pkg/types"
)

const (
	// SecretsBasePath is the base directory for secret tmpfs mounts
	SecretsBasePath = "/run/secrets"
)

// SecretsHandler manages secret mounting for tasks
type SecretsHandler struct {
	worker         *Worker
	secretsManager *security.SecretsManager
}

// NewSecretsHandler creates a new secrets handler
func NewSecretsHandler(worker *Worker, encryptionKey []byte) (*SecretsHandler, error) {
	sm, err := security.NewSecretsManager(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create secrets manager: %w", err)
	}

	return &SecretsHandler{
		worker:         worker,
		secretsManager: sm,
	}, nil
}

// MountSecretsForTask fetches secrets from manager and mounts them to tmpfs
// Returns the tmpfs mount path for the container
func (sh *SecretsHandler) MountSecretsForTask(task *types.Task) (string, error) {
	if len(task.Secrets) == 0 {
		return "", nil // No secrets to mount
	}

	// Create task-specific secrets directory in tmpfs
	taskSecretsPath := filepath.Join(SecretsBasePath, task.ID)
	if err := os.MkdirAll(taskSecretsPath, 0700); err != nil {
		return "", fmt.Errorf("failed to create secrets directory: %w", err)
	}

	// Fetch and mount each secret
	for _, secretName := range task.Secrets {
		if err := sh.mountSecret(task.ID, secretName, taskSecretsPath); err != nil {
			// Cleanup on error
			_ = sh.CleanupSecretsForTask(task.ID) // Ignore cleanup errors during rollback
			return "", fmt.Errorf("failed to mount secret %s: %w", secretName, err)
		}
	}

	return taskSecretsPath, nil
}

// mountSecret fetches a single secret from manager and writes it to tmpfs
func (sh *SecretsHandler) mountSecret(taskID, secretName, targetDir string) error {
	// Fetch secret from manager
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := sh.worker.client.GetSecretByName(ctx, &proto.GetSecretByNameRequest{
		Name: secretName,
	})
	if err != nil {
		return fmt.Errorf("failed to fetch secret from manager: %w", err)
	}

	// Convert proto secret to types.Secret
	secret := &types.Secret{
		ID:   resp.Secret.Id,
		Name: resp.Secret.Name,
		Data: resp.Secret.Data, // Encrypted data
	}

	// Decrypt the secret
	plaintext, err := sh.secretsManager.GetSecretData(secret)
	if err != nil {
		return fmt.Errorf("failed to decrypt secret: %w", err)
	}

	// Write to tmpfs (read-only for security)
	secretPath := filepath.Join(targetDir, secretName)
	if err := os.WriteFile(secretPath, plaintext, 0400); err != nil {
		return fmt.Errorf("failed to write secret file: %w", err)
	}

	return nil
}

// CleanupSecretsForTask removes all secrets for a task from tmpfs
func (sh *SecretsHandler) CleanupSecretsForTask(taskID string) error {
	taskSecretsPath := filepath.Join(SecretsBasePath, taskID)

	// Check if directory exists
	if _, err := os.Stat(taskSecretsPath); os.IsNotExist(err) {
		return nil // Nothing to clean up
	}

	// Remove the entire task secrets directory
	if err := os.RemoveAll(taskSecretsPath); err != nil {
		return fmt.Errorf("failed to cleanup secrets: %w", err)
	}

	return nil
}

// GetSecretPath returns the path to a specific secret for a task
func (sh *SecretsHandler) GetSecretPath(taskID, secretName string) string {
	return filepath.Join(SecretsBasePath, taskID, secretName)
}

// EnsureSecretsBaseDir ensures the base secrets directory exists
// This should be called during worker initialization
func EnsureSecretsBaseDir() error {
	// Create /run/secrets if it doesn't exist
	if err := os.MkdirAll(SecretsBasePath, 0700); err != nil {
		return fmt.Errorf("failed to create secrets base directory: %w", err)
	}

	// TODO: Mount as tmpfs for added security
	// This would typically be done via:
	// mount -t tmpfs -o size=10M,mode=0700 tmpfs /run/secrets
	// For now, we're using a regular directory which is sufficient for POC

	return nil
}
