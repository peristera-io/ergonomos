// Package authz defines the authorization boundary for ergonomos.
//
// Per ADR-0002 the v1 implementation embeds OpenFGA in-process, but every
// call site depends only on the Authorizer interface so the backend
// (in-process OpenFGA, remote OpenFGA, or SpiceDB) can be swapped without
// churn. Handlers must never call an authz backend directly.
package authz

import (
	"context"
	"sync"
)

// Relation is a named relationship in the authorization model, e.g. "viewer",
// "editor", "owner", "parent".
type Relation string

// Ref identifies an object or subject in ReBAC "type:id" form, e.g.
// "task:01J...", "user:ada@example.org".
type Ref string

// Tuple is a relationship assertion: Subject has Relation on Object.
type Tuple struct {
	Subject  Ref
	Relation Relation
	Object   Ref
}

// Authorizer is the authorization boundary. Implementations must be safe for
// concurrent use.
type Authorizer interface {
	// Check reports whether Subject has Relation on Object.
	Check(ctx context.Context, t Tuple) (bool, error)
	// Write atomically adds and removes relationship tuples.
	Write(ctx context.Context, add, remove []Tuple) error
}

// Memory is an in-memory Authorizer for tests and early development. It does a
// direct tuple lookup with no relation rewrites; the real model (with rewrites
// such as "editor implies viewer") lives in OpenFGA — see ADR-0002.
type Memory struct {
	mu  sync.RWMutex
	set map[Tuple]struct{}
}

// NewMemory returns an empty in-memory Authorizer.
func NewMemory() *Memory {
	return &Memory{set: make(map[Tuple]struct{})}
}

// Check reports whether the exact tuple has been written.
func (m *Memory) Check(_ context.Context, t Tuple) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.set[t]
	return ok, nil
}

// Write adds and removes tuples. Removes are applied after adds.
func (m *Memory) Write(_ context.Context, add, remove []Tuple) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range add {
		m.set[t] = struct{}{}
	}
	for _, t := range remove {
		delete(m.set, t)
	}
	return nil
}
