package task

import (
	"context"
	"sync"

	"github.com/peristera-io/ergonomos/server/internal/domain"
)

// MemoryStore is an in-memory Store for tests and early development. The
// Postgres adapter implementing the same port lands in a later slice.
type MemoryStore struct {
	mu   sync.RWMutex
	byID map[domain.ID]Task
}

// NewMemoryStore returns an empty in-memory Store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{byID: make(map[domain.ID]Task)}
}

func (m *MemoryStore) Create(_ context.Context, t Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byID[t.ID] = t
	return nil
}

func (m *MemoryStore) Get(_ context.Context, id domain.ID) (Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.byID[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	return t, nil
}

func (m *MemoryStore) Update(_ context.Context, t Task) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byID[t.ID]; !ok {
		return ErrNotFound
	}
	m.byID[t.ID] = t
	return nil
}

func (m *MemoryStore) Delete(_ context.Context, id domain.ID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.byID[id]; !ok {
		return ErrNotFound
	}
	delete(m.byID, id)
	return nil
}

func (m *MemoryStore) ListByOwner(_ context.Context, owner domain.ID) ([]Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Task
	for _, t := range m.byID {
		if t.Owner == owner {
			out = append(out, t)
		}
	}
	return out, nil
}
