package auth

import (
	"context"
	"sync"
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
