package events

import (
	"context"
	"sync"
)

// MemoryOutbox is an in-memory Outbox for tests and early development. The real
// transactional adapter (events written in the same Postgres transaction as the
// domain mutation, ADR-0003) lands in a later slice behind this same port.
type MemoryOutbox struct {
	mu     sync.Mutex
	events []Event
}

// NewMemoryOutbox returns an empty in-memory Outbox.
func NewMemoryOutbox() *MemoryOutbox {
	return &MemoryOutbox{}
}

// Append records events in order.
func (o *MemoryOutbox) Append(_ context.Context, events ...Event) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.events = append(o.events, events...)
	return nil
}

// Events returns a copy of the events appended so far, for inspection in tests.
func (o *MemoryOutbox) Events() []Event {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]Event(nil), o.events...)
}
