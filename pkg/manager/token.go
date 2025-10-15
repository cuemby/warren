package manager

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// TokenManager manages join tokens for the cluster
type TokenManager struct {
	tokens map[string]*JoinToken
	mu     sync.RWMutex
}

// JoinToken represents a token for joining the cluster
type JoinToken struct {
	Token     string
	Role      string // "manager" or "worker"
	CreatedAt time.Time
	ExpiresAt time.Time
}

// NewTokenManager creates a new token manager
func NewTokenManager() *TokenManager {
	return &TokenManager{
		tokens: make(map[string]*JoinToken),
	}
}

// GenerateToken generates a new join token
func (tm *TokenManager) GenerateToken(role string, duration time.Duration) (*JoinToken, error) {
	// Generate a random token
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("failed to generate random token: %w", err)
	}

	token := hex.EncodeToString(bytes)

	jt := &JoinToken{
		Token:     token,
		Role:      role,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(duration),
	}

	tm.mu.Lock()
	tm.tokens[token] = jt
	tm.mu.Unlock()

	return jt, nil
}

// ValidateToken validates a join token and returns its role
func (tm *TokenManager) ValidateToken(token string) (string, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	jt, exists := tm.tokens[token]
	if !exists {
		return "", fmt.Errorf("invalid token")
	}

	if time.Now().After(jt.ExpiresAt) {
		return "", fmt.Errorf("token expired")
	}

	return jt.Role, nil
}

// RevokeToken revokes a join token
func (tm *TokenManager) RevokeToken(token string) {
	tm.mu.Lock()
	delete(tm.tokens, token)
	tm.mu.Unlock()
}

// CleanupExpiredTokens removes expired tokens
func (tm *TokenManager) CleanupExpiredTokens() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	now := time.Now()
	for token, jt := range tm.tokens {
		if now.After(jt.ExpiresAt) {
			delete(tm.tokens, token)
		}
	}
}

// ListTokens returns all active tokens
func (tm *TokenManager) ListTokens() []*JoinToken {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	tokens := make([]*JoinToken, 0, len(tm.tokens))
	for _, jt := range tm.tokens {
		tokens = append(tokens, jt)
	}

	return tokens
}
