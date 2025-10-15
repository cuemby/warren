package security

import (
	"bytes"
	"testing"
)

func TestNewSecretsManager(t *testing.T) {
	tests := []struct {
		name    string
		key     []byte
		wantErr bool
	}{
		{
			name:    "valid 32-byte key",
			key:     make([]byte, 32),
			wantErr: false,
		},
		{
			name:    "invalid short key",
			key:     make([]byte, 16),
			wantErr: true,
		},
		{
			name:    "invalid long key",
			key:     make([]byte, 64),
			wantErr: true,
		},
		{
			name:    "empty key",
			key:     []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm, err := NewSecretsManager(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSecretsManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && sm == nil {
				t.Error("NewSecretsManager() returned nil without error")
			}
		})
	}
}

func TestNewSecretsManagerFromPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "valid password",
			password: "my-secure-password",
			wantErr:  false,
		},
		{
			name:     "empty password",
			password: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm, err := NewSecretsManagerFromPassword(tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSecretsManagerFromPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && sm == nil {
				t.Error("NewSecretsManagerFromPassword() returned nil without error")
			}
		})
	}
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	copy(key, []byte("test-encryption-key-32-bytes-!!"))

	sm, err := NewSecretsManager(key)
	if err != nil {
		t.Fatalf("Failed to create SecretsManager: %v", err)
	}

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{
			name:      "simple string",
			plaintext: []byte("hello world"),
		},
		{
			name:      "json data",
			plaintext: []byte(`{"username":"admin","password":"secret123"}`),
		},
		{
			name:      "binary data",
			plaintext: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD},
		},
		{
			name:      "large data",
			plaintext: bytes.Repeat([]byte("test"), 1000),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := sm.EncryptSecret(tt.plaintext)
			if err != nil {
				t.Fatalf("EncryptSecret() error = %v", err)
			}

			// Verify ciphertext is different from plaintext
			if bytes.Equal(ciphertext, tt.plaintext) {
				t.Error("Ciphertext should not equal plaintext")
			}

			// Decrypt
			decrypted, err := sm.DecryptSecret(ciphertext)
			if err != nil {
				t.Fatalf("DecryptSecret() error = %v", err)
			}

			// Verify roundtrip
			if !bytes.Equal(decrypted, tt.plaintext) {
				t.Errorf("Decrypted data does not match original.\nGot:  %v\nWant: %v", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptSecret_Errors(t *testing.T) {
	key := make([]byte, 32)
	sm, _ := NewSecretsManager(key)

	tests := []struct {
		name      string
		plaintext []byte
		wantErr   bool
	}{
		{
			name:      "empty data",
			plaintext: []byte{},
			wantErr:   true,
		},
		{
			name:      "nil data",
			plaintext: nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sm.EncryptSecret(tt.plaintext)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncryptSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDecryptSecret_Errors(t *testing.T) {
	key := make([]byte, 32)
	sm, _ := NewSecretsManager(key)

	tests := []struct {
		name       string
		ciphertext []byte
		wantErr    bool
	}{
		{
			name:       "empty data",
			ciphertext: []byte{},
			wantErr:    true,
		},
		{
			name:       "nil data",
			ciphertext: nil,
			wantErr:    true,
		},
		{
			name:       "too short data",
			ciphertext: []byte{0x01, 0x02},
			wantErr:    true,
		},
		{
			name:       "corrupted data",
			ciphertext: bytes.Repeat([]byte("x"), 100),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sm.DecryptSecret(tt.ciphertext)
			if (err != nil) != tt.wantErr {
				t.Errorf("DecryptSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	copy(key1, []byte("key-one-32-bytes-long-!!!!!!!!!!"))

	key2 := make([]byte, 32)
	copy(key2, []byte("key-two-32-bytes-long-!!!!!!!!!!"))

	sm1, _ := NewSecretsManager(key1)
	sm2, _ := NewSecretsManager(key2)

	plaintext := []byte("secret data")

	// Encrypt with first key
	ciphertext, err := sm1.EncryptSecret(plaintext)
	if err != nil {
		t.Fatalf("EncryptSecret() error = %v", err)
	}

	// Try to decrypt with second key (should fail)
	_, err = sm2.DecryptSecret(ciphertext)
	if err == nil {
		t.Error("DecryptSecret() should fail with wrong key")
	}
}

func TestCreateSecret(t *testing.T) {
	key := make([]byte, 32)
	sm, _ := NewSecretsManager(key)

	tests := []struct {
		name       string
		secretName string
		plaintext  []byte
		wantErr    bool
	}{
		{
			name:       "valid secret",
			secretName: "db-password",
			plaintext:  []byte("supersecret123"),
			wantErr:    false,
		},
		{
			name:       "empty name",
			secretName: "",
			plaintext:  []byte("data"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret, err := sm.CreateSecret(tt.secretName, tt.plaintext)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if secret == nil {
					t.Fatal("CreateSecret() returned nil secret")
				}
				if secret.Name != tt.secretName {
					t.Errorf("Secret name = %v, want %v", secret.Name, tt.secretName)
				}
				if secret.ID == "" {
					t.Error("Secret ID should not be empty")
				}
				if len(secret.Data) == 0 {
					t.Error("Secret data should not be empty")
				}
			}
		})
	}
}

func TestGetSecretData(t *testing.T) {
	key := make([]byte, 32)
	sm, _ := NewSecretsManager(key)

	plaintext := []byte("my-secret-value")
	secret, err := sm.CreateSecret("test-secret", plaintext)
	if err != nil {
		t.Fatalf("CreateSecret() error = %v", err)
	}

	// Get secret data
	data, err := sm.GetSecretData(secret)
	if err != nil {
		t.Fatalf("GetSecretData() error = %v", err)
	}

	if !bytes.Equal(data, plaintext) {
		t.Errorf("GetSecretData() = %v, want %v", data, plaintext)
	}
}

func TestGetSecretData_NilSecret(t *testing.T) {
	key := make([]byte, 32)
	sm, _ := NewSecretsManager(key)

	_, err := sm.GetSecretData(nil)
	if err == nil {
		t.Error("GetSecretData() should fail with nil secret")
	}
}

func TestDeriveKeyFromClusterID(t *testing.T) {
	tests := []struct {
		name      string
		clusterID string
	}{
		{
			name:      "simple ID",
			clusterID: "cluster-123",
		},
		{
			name:      "UUID",
			clusterID: "550e8400-e29b-41d4-a716-446655440000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := DeriveKeyFromClusterID(tt.clusterID)

			if len(key) != 32 {
				t.Errorf("DeriveKeyFromClusterID() returned key of length %d, want 32", len(key))
			}

			// Verify key is deterministic
			key2 := DeriveKeyFromClusterID(tt.clusterID)
			if !bytes.Equal(key, key2) {
				t.Error("DeriveKeyFromClusterID() should be deterministic")
			}

			// Verify different IDs produce different keys
			differentKey := DeriveKeyFromClusterID(tt.clusterID + "-different")
			if bytes.Equal(key, differentKey) {
				t.Error("Different cluster IDs should produce different keys")
			}
		})
	}
}
