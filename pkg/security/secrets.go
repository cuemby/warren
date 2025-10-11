package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/cuemby/warren/pkg/types"
)

// SecretsManager handles encryption and decryption of secrets
type SecretsManager struct {
	encryptionKey []byte // 32 bytes for AES-256
}

// NewSecretsManager creates a new secrets manager with the given encryption key
// The key should be 32 bytes for AES-256-GCM
func NewSecretsManager(key []byte) (*SecretsManager, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes for AES-256, got %d", len(key))
	}

	return &SecretsManager{
		encryptionKey: key,
	}, nil
}

// NewSecretsManagerFromPassword creates a secrets manager using a password
// The password is hashed with SHA-256 to derive the encryption key
func NewSecretsManagerFromPassword(password string) (*SecretsManager, error) {
	if password == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}

	// Derive 32-byte key from password using SHA-256
	hash := sha256.Sum256([]byte(password))
	return NewSecretsManager(hash[:])
}

// EncryptSecret encrypts plaintext data using AES-256-GCM
// Returns encrypted data with nonce prepended
func (sm *SecretsManager) EncryptSecret(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, fmt.Errorf("cannot encrypt empty data")
	}

	// Create AES cipher
	block, err := aes.NewCipher(sm.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and prepend nonce
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptSecret decrypts data encrypted with EncryptSecret
// Expects nonce to be prepended to ciphertext
func (sm *SecretsManager) DecryptSecret(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, fmt.Errorf("cannot decrypt empty data")
	}

	// Create AES cipher
	block, err := aes.NewCipher(sm.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Check minimum length
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract nonce and ciphertext
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	// Decrypt
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// CreateSecret creates a new encrypted secret
func (sm *SecretsManager) CreateSecret(name string, plaintext []byte) (*types.Secret, error) {
	if name == "" {
		return nil, fmt.Errorf("secret name cannot be empty")
	}

	// Encrypt the data
	encrypted, err := sm.EncryptSecret(plaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Generate unique ID
	id := generateSecretID(name)

	return &types.Secret{
		ID:   id,
		Name: name,
		Data: encrypted,
	}, nil
}

// GetSecretData decrypts and returns the plaintext data from a secret
func (sm *SecretsManager) GetSecretData(secret *types.Secret) ([]byte, error) {
	if secret == nil {
		return nil, fmt.Errorf("secret cannot be nil")
	}

	return sm.DecryptSecret(secret.Data)
}

// generateSecretID generates a unique ID for a secret based on its name
func generateSecretID(name string) string {
	hash := sha256.Sum256([]byte(name))
	return base64.URLEncoding.EncodeToString(hash[:16])
}

// DeriveKeyFromClusterID derives an encryption key from the cluster ID
// This is used during cluster initialization to create a consistent key
func DeriveKeyFromClusterID(clusterID string) []byte {
	hash := sha256.Sum256([]byte(clusterID))
	return hash[:]
}

// clusterEncryptionKey is the global encryption key for the cluster
// This is derived from the cluster ID during initialization
var clusterEncryptionKey []byte

// SetClusterEncryptionKey sets the global cluster encryption key
// This should be called once during cluster initialization
func SetClusterEncryptionKey(key []byte) error {
	if len(key) != 32 {
		return fmt.Errorf("encryption key must be 32 bytes, got %d", len(key))
	}
	clusterEncryptionKey = key
	return nil
}

// Encrypt encrypts data using the cluster encryption key
// This is used for encrypting sensitive data like CA private keys
func Encrypt(plaintext []byte) ([]byte, error) {
	if len(clusterEncryptionKey) == 0 {
		return nil, fmt.Errorf("cluster encryption key not set")
	}

	block, err := aes.NewCipher(clusterEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using the cluster encryption key
// This is used for decrypting sensitive data like CA private keys
func Decrypt(ciphertext []byte) ([]byte, error) {
	if len(clusterEncryptionKey) == 0 {
		return nil, fmt.Errorf("cluster encryption key not set")
	}

	block, err := aes.NewCipher(clusterEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}
