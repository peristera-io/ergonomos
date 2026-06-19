package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"sync"

	"github.com/peristera-io/ergonomos/server/internal/domain"
)

// MemoryStore is an in-memory Store for tests and early development. The
// Postgres adapter implementing the same port lands in a later slice.
type MemoryStore struct {
	mu      sync.RWMutex
	byEmail map[string]User
}

// NewMemoryStore returns an empty in-memory Store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{byEmail: make(map[string]User)}
}

func (m *MemoryStore) CreateUser(_ context.Context, u User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.byEmail[u.Email]; exists {
		return ErrEmailTaken
	}
	m.byEmail[u.Email] = u
	return nil
}

func (m *MemoryStore) FindByEmail(_ context.Context, email string) (User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.byEmail[email]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

// MemorySessions is an in-memory Sessions adapter for tests and early
// development. Tokens live only for the process lifetime; a persistent adapter
// (with expiry/revocation) implements the same port in a later slice.
type MemorySessions struct {
	mu      sync.RWMutex
	byToken map[string]domain.Actor
}

// NewMemorySessions returns an empty in-memory Sessions store.
func NewMemorySessions() *MemorySessions {
	return &MemorySessions{byToken: make(map[string]domain.Actor)}
}

func (m *MemorySessions) Issue(_ context.Context, actor domain.Actor) (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b[:])
	m.mu.Lock()
	m.byToken[token] = actor
	m.mu.Unlock()
	return token, nil
}

func (m *MemorySessions) Resolve(_ context.Context, token string) (domain.Actor, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	actor, ok := m.byToken[token]
	if !ok {
		return domain.Actor{}, ErrInvalidToken
	}
	return actor, nil
}
